package httpapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"aether/constellation/access"
	"aether/constellation/catalog"
	"aether/constellation/propagate"
	"aether/constellation/schema"
	"aether/constellation/tasks"
)

const maxWindow = 6 * time.Hour

// Server serves the fleet catalog and propagated positions.
type Server struct {
	db     *sql.DB
	mu     sync.Mutex
	cat    catalog.Catalog
	search *access.Searcher
	built  int64
	delays tasks.Delays
}

// New creates the database when needed, migrates it, and loads the fleet.
func New(dsn string, delays tasks.Delays) (*Server, error) {
	if err := schema.Ensure(dsn); err != nil {
		return nil, err
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	var pingErr error
	for attempt := 0; attempt < 30; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if pingErr != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres: %w", pingErr)
	}
	if err := schema.Up(dsn); err != nil {
		_ = db.Close()
		return nil, err
	}
	cat, err := catalog.Load(db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	log.Printf("constellation catalog version %d", cat.Version)
	return &Server{db: db, cat: cat, delays: delays}, nil
}

// Handler is the HTTP API.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/catalog/version", s.version)
	mux.HandleFunc("GET /api/catalog", s.catalog)
	mux.HandleFunc("POST /api/acquisitions", s.earliest)
	mux.HandleFunc("POST /api/tasks", s.createTask)
	mux.HandleFunc("POST /api/tasks/query", s.queryTasks)
	mux.HandleFunc("POST /api/tasks/{id}/cancel", s.cancelTask)
	mux.HandleFunc("GET /api/modes", s.modes)
	mux.HandleFunc("GET /api/satellites", s.satellites)
	mux.HandleFunc("GET /api/ephemeris", s.ephemeris)
	return mux
}

func (s *Server) current() (catalog.Catalog, error) {
	var version int64
	if err := s.db.QueryRow(`SELECT version FROM constellation`).Scan(&version); err != nil {
		return catalog.Catalog{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cat.Version == version {
		return s.cat, nil
	}
	cat, err := catalog.Load(s.db)
	if err != nil {
		return catalog.Catalog{}, err
	}
	s.cat = cat
	log.Printf("constellation catalog version %d", cat.Version)
	return s.cat, nil
}

func (s *Server) acquire() (*access.Searcher, error) {
	cat, err := s.current()
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.search == nil || s.built != cat.Version {
		s.search = access.New(cat)
		s.built = cat.Version
	}
	return s.search, nil
}

func (s *Server) earliest(w http.ResponseWriter, r *http.Request) {
	req, err := access.Parse(r.Body)
	if err != nil {
		writeInvalid(w, err)
		return
	}
	search, spans, ok := s.ready(w, r, req.Mode)
	if !ok {
		return
	}
	offer, found := search.Earliest(req, spans)
	raw, err := access.MarshalResult(offer, found)
	if err != nil {
		writeStatus(w, http.StatusInternalServerError, map[string]string{"error": "could not encode the acquisition"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(raw)
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeStatus(w, http.StatusBadRequest, map[string]string{"error": "request body must be JSON"})
		return
	}
	req, err := access.Parse(bytes.NewReader(raw))
	if err != nil {
		writeInvalid(w, err)
		return
	}
	var extra struct {
		ClientRef string `json:"clientRef"`
	}
	if err := json.Unmarshal(raw, &extra); err != nil {
		writeStatus(w, http.StatusBadRequest, map[string]string{"error": "request body must be JSON"})
		return
	}
	search, ok := s.searchOnly(w, req.Mode)
	if !ok {
		return
	}
	created, err := tasks.Create(r.Context(), s.db, search, req, extra.ClientRef, s.delays)
	if errors.Is(err, tasks.ErrNotFeasible) {
		writeStatus(w, http.StatusConflict, map[string]string{"error": "not feasible"})
		return
	}
	var inv *access.Invalid
	if errors.As(err, &inv) {
		writeStatus(w, http.StatusBadRequest, map[string]string{"error": inv.Msg})
		return
	}
	if err != nil {
		writeStatus(w, http.StatusInternalServerError, map[string]string{"error": "could not store the task"})
		return
	}
	body, err := tasks.Marshal(created)
	if err != nil {
		writeStatus(w, http.StatusInternalServerError, map[string]string{"error": "could not encode the task"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (s *Server) queryTasks(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeStatus(w, http.StatusBadRequest, map[string]string{"error": "request body must be JSON"})
		return
	}
	list, err := tasks.ByIDs(r.Context(), s.db, body.IDs, time.Now().UTC(), s.delays)
	if err != nil {
		writeStatus(w, http.StatusInternalServerError, map[string]string{"error": "could not read tasks"})
		return
	}
	raw, err := tasks.MarshalAll(list)
	if err != nil {
		writeStatus(w, http.StatusInternalServerError, map[string]string{"error": "could not encode the tasks"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(raw)
}

func (s *Server) cancelTask(w http.ResponseWriter, r *http.Request) {
	task, err := tasks.Cancel(r.Context(), s.db, r.PathValue("id"), time.Now().UTC(), s.delays)
	if errors.Is(err, tasks.ErrNotFound) {
		writeStatus(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if errors.Is(err, tasks.ErrFinished) {
		writeStatus(w, http.StatusConflict, map[string]string{"error": "already finished"})
		return
	}
	if err != nil {
		writeStatus(w, http.StatusInternalServerError, map[string]string{"error": "could not cancel the task"})
		return
	}
	body, err := tasks.Marshal(task)
	if err != nil {
		writeStatus(w, http.StatusInternalServerError, map[string]string{"error": "could not encode the task"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

// Advance stores status changes for tasks that have not finished.
func (s *Server) Advance(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			_ = tasks.Advance(context.Background(), s.db, now.UTC(), s.delays)
		}
	}
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request, mode string) (*access.Searcher, []access.Span, bool) {
	search, ok := s.searchOnly(w, mode)
	if !ok {
		return nil, nil, false
	}
	spans, err := tasks.Busy(r.Context(), s.db)
	if err != nil {
		writeStatus(w, http.StatusInternalServerError, map[string]string{"error": "could not read tasks"})
		return nil, nil, false
	}
	return search, spans, true
}

func (s *Server) searchOnly(w http.ResponseWriter, mode string) (*access.Searcher, bool) {
	search, err := s.acquire()
	if err != nil {
		writeStatus(w, http.StatusInternalServerError, map[string]string{"error": "catalog unavailable"})
		return nil, false
	}
	if !search.Knows(mode) {
		writeStatus(w, http.StatusBadRequest, map[string]string{"error": "unknown imaging mode"})
		return nil, false
	}
	return search, true
}

func writeInvalid(w http.ResponseWriter, err error) {
	msg := "request body must be JSON"
	var inv *access.Invalid
	if errors.As(err, &inv) {
		msg = inv.Msg
	}
	writeStatus(w, http.StatusBadRequest, map[string]string{"error": msg})
}

func (s *Server) version(w http.ResponseWriter, _ *http.Request) {
	cat, err := s.current()
	if err != nil {
		http.Error(w, "catalog unavailable", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]int64{"version": cat.Version})
}

func (s *Server) catalog(w http.ResponseWriter, _ *http.Request) {
	cat, err := s.current()
	if err != nil {
		http.Error(w, "catalog unavailable", http.StatusInternalServerError)
		return
	}
	writeJSON(w, cat)
}

func (s *Server) modes(w http.ResponseWriter, _ *http.Request) {
	cat, err := s.current()
	if err != nil {
		http.Error(w, "catalog unavailable", http.StatusInternalServerError)
		return
	}
	writeJSON(w, cat.Modes)
}

func (s *Server) satellites(w http.ResponseWriter, _ *http.Request) {
	cat, err := s.current()
	if err != nil {
		http.Error(w, "catalog unavailable", http.StatusInternalServerError)
		return
	}
	type plane struct {
		Name      string  `json:"name"`
		Color     string  `json:"color"`
		PeriodMin float64 `json:"periodMin"`
	}
	type item struct {
		Name    string `json:"name"`
		Catalog int    `json:"catalog"`
		Plane   string `json:"plane"`
		Class   string `json:"class"`
		Look    string `json:"look"`
	}
	body := struct {
		Epoch      string  `json:"epoch"`
		Planes     []plane `json:"planes"`
		Satellites []item  `json:"satellites"`
	}{
		Epoch:      cat.Epoch.Format(time.RFC3339),
		Planes:     make([]plane, len(cat.Planes)),
		Satellites: make([]item, len(cat.Satellites)),
	}
	for i, p := range cat.Planes {
		body.Planes[i] = plane{Name: p.Name, Color: p.Color, PeriodMin: p.PeriodMin}
	}
	for i, sat := range cat.Satellites {
		body.Satellites[i] = item{Name: sat.Name, Catalog: sat.Catalog, Plane: sat.Plane, Class: sat.Class, Look: sat.Look}
	}
	writeJSON(w, body)
}

func (s *Server) ephemeris(w http.ResponseWriter, r *http.Request) {
	from, to, step, ok := queryWindow(w, r)
	if !ok {
		return
	}
	cat, err := s.current()
	if err != nil {
		http.Error(w, "catalog unavailable", http.StatusInternalServerError)
		return
	}
	tracks := propagate.New(cat.Satellites).Ephemeris(from, to, step)
	type sample struct {
		T     string  `json:"t"`
		Lat   float64 `json:"latDeg"`
		Lon   float64 `json:"lonDeg"`
		AltKm float64 `json:"altKm"`
		EcefX float64 `json:"ecefX"`
		EcefY float64 `json:"ecefY"`
		EcefZ float64 `json:"ecefZ"`
	}
	type track struct {
		Name    string   `json:"name"`
		Samples []sample `json:"samples"`
	}
	body := struct {
		From        string  `json:"from"`
		To          string  `json:"to"`
		StepSeconds int     `json:"stepSeconds"`
		Satellites  []track `json:"satellites"`
	}{
		From:        from.Format(time.RFC3339),
		To:          to.Format(time.RFC3339),
		StepSeconds: int(step / time.Second),
		Satellites:  make([]track, len(tracks)),
	}
	for i, tr := range tracks {
		samples := make([]sample, len(tr.Samples))
		for j, sampleAt := range tr.Samples {
			samples[j] = sample{
				T: sampleAt.Time.Format(time.RFC3339), Lat: sampleAt.Lat, Lon: sampleAt.Lon, AltKm: sampleAt.AltKm,
				EcefX: sampleAt.EcefX, EcefY: sampleAt.EcefY, EcefZ: sampleAt.EcefZ,
			}
		}
		body.Satellites[i] = track{Name: tr.Name, Samples: samples}
	}
	writeJSON(w, body)
}

func queryWindow(w http.ResponseWriter, r *http.Request) (from, to time.Time, step time.Duration, ok bool) {
	q := r.URL.Query()
	var err error
	from, err = time.Parse(time.RFC3339Nano, q.Get("from"))
	if err != nil {
		http.Error(w, "from must be RFC3339", http.StatusBadRequest)
		return
	}
	to, err = time.Parse(time.RFC3339Nano, q.Get("to"))
	if err != nil {
		http.Error(w, "to must be RFC3339", http.StatusBadRequest)
		return
	}
	if !to.After(from) {
		http.Error(w, "to must be after from", http.StatusBadRequest)
		return
	}
	if to.Sub(from) > maxWindow {
		http.Error(w, "window must be 6 hours or less", http.StatusBadRequest)
		return
	}
	seconds, err := strconv.Atoi(q.Get("stepSeconds"))
	if err != nil || seconds < 5 || seconds > 60 {
		http.Error(w, "stepSeconds must be from 5 to 60", http.StatusBadRequest)
		return
	}
	return from, to, time.Duration(seconds) * time.Second, true
}

func writeStatus(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(body)
}
