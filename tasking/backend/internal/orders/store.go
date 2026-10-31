package orders

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"aether/constellation/access"
	"aether/constellation/tasks"
	"aether/tasking/internal/schema"
)

// ErrNotFeasible means no free satellite can take the picture.
var ErrNotFeasible = errors.New("not feasible")

// ErrNotFound means the order id is unknown.
var ErrNotFound = errors.New("not found")

// ErrFinished means the collection is already over, so it cannot be cancelled.
var ErrFinished = errors.New("already finished")

// Order is one booked collection. Satellite, plane, class, and mode are copies from the moment it was booked.
type Order struct {
	ID              string
	TaskID          string
	Status          string
	CreatedAt       time.Time
	CancelledAt     *time.Time
	Satellite       string
	Plane           string
	Class           string
	Lat             float64
	Lon             float64
	Mode            string
	RequestedLook   string
	RequestedPass   string
	IncidenceChoice string
	WindowStart     time.Time
	WindowEnd       time.Time
	ImagingStart    time.Time
	ImagingEnd      time.Time
	Duration        int
	Incidence       float64
	LookSide        string
	Pass            string
	Squint          float64
}

// Finder asks the constellation for a collection and for the task that records it.
type Finder interface {
	Earliest(ctx context.Context, req access.Request) (access.Offer, bool, error)
	Create(ctx context.Context, req access.Request, clientRef string) (tasks.Task, error)
	Cancel(ctx context.Context, id string) (tasks.Task, error)
	Query(ctx context.Context, ids []string) ([]tasks.Task, error)
}

// Store keeps orders in Postgres. The constellation decides whether a collection is valid.
type Store struct {
	db     *sql.DB
	finder Finder
}

// Open connects to Postgres and applies pending order migrations.
func Open(dsn string, finder Finder) (*Store, error) {
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
	return &Store{db: db, finder: finder}, nil
}

// Close releases the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// Earliest asks the constellation for the soonest collection. The constellation skips its own holds.
func (s *Store) Earliest(ctx context.Context, req access.Request) (access.Offer, bool, error) {
	return s.finder.Earliest(ctx, req)
}

// List returns orders newest first.
func (s *Store) List() ([]Order, error) {
	rows, err := s.db.Query(`SELECT ` + columns + ` FROM orders ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Order
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, order)
	}
	return out, rows.Err()
}

// Get loads one order.
func (s *Store) Get(id string) (Order, error) {
	row := s.db.QueryRow(`SELECT `+columns+` FROM orders WHERE id = $1`, id)
	order, err := scanOrder(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	return order, err
}

// Place asks the constellation to hold the earliest free collection, then stores the order.
func (s *Store) Place(ctx context.Context, req access.Request) (Order, error) {
	id, err := newID()
	if err != nil {
		return Order{}, err
	}
	task, err := s.finder.Create(ctx, req, id)
	if errors.Is(err, tasks.ErrNotFeasible) {
		return Order{}, ErrNotFeasible
	}
	if err != nil {
		return Order{}, err
	}
	order, err := s.insert(id, req, task)
	if err != nil {
		_, _ = s.finder.Cancel(ctx, task.ID)
		return Order{}, err
	}
	return order, nil
}

// Cancel frees a collection that has not finished. The satellite is released first.
func (s *Store) Cancel(ctx context.Context, id string) (Order, error) {
	order, err := s.Get(id)
	if err != nil {
		return Order{}, err
	}
	if order.Status == "cancelled" || order.CancelledAt != nil {
		return order, nil
	}
	if order.TaskID == "" {
		return order, ErrNotFound
	}
	task, err := s.finder.Cancel(ctx, order.TaskID)
	if errors.Is(err, tasks.ErrFinished) {
		return order, ErrFinished
	}
	if errors.Is(err, tasks.ErrNotFound) {
		return order, ErrNotFound
	}
	if err != nil {
		return Order{}, err
	}
	var cancelled any
	if task.CancelledAt != nil {
		cancelled = task.CancelledAt.UTC()
	}
	res, err := s.db.Exec(`UPDATE orders SET status = $1, cancelled_at = $2 WHERE id = $3`, task.Status, cancelled, id)
	if err != nil {
		return Order{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return s.Get(id)
	}
	return s.Get(id)
}

// Sync copies constellation status onto orders that have not finished.
func (s *Store) Sync(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT task_id FROM orders WHERE status IN ('scheduled', 'imaging', 'downlink', 'processing')`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		if id != "" {
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil || len(ids) == 0 {
		return err
	}
	list, err := s.finder.Query(ctx, ids)
	if err != nil {
		return err
	}
	for _, task := range list {
		var cancelled any
		if task.CancelledAt != nil {
			cancelled = task.CancelledAt.UTC()
		}
		if _, err := s.db.ExecContext(ctx, `
			UPDATE orders SET status = $1, cancelled_at = COALESCE($2, cancelled_at)
			WHERE task_id = $3 AND status IN ('scheduled', 'imaging', 'downlink', 'processing')`,
			task.Status, cancelled, task.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) insert(id string, req access.Request, task tasks.Task) (Order, error) {
	offer := task.Offer
	created := time.Now().UTC().Truncate(time.Second)
	_, err := s.db.Exec(`
		INSERT INTO orders (
		  id, task_id, status, created_at, satellite, plane, class, lat, lon, mode,
		  requested_look, requested_pass, incidence_choice,
		  window_start, window_end, imaging_start, imaging_end,
		  duration_s, incidence, look_side, pass, squint
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)`,
		id, task.ID, task.Status, created,
		offer.Satellite, offer.Plane, offer.Class, req.Lat, req.Lon, req.Mode,
		blank(req.LookSide, "ANY"), blank(req.Pass, "ANY"), blank(req.Incidence, "performant"),
		req.WindowFrom.UTC(), req.WindowTo.UTC(),
		offer.Start.UTC(), offer.End.UTC(),
		offer.Duration, offer.Incidence, offer.LookSide, offer.Pass, offer.Squint,
	)
	if err != nil {
		return Order{}, err
	}
	return s.Get(id)
}

const columns = `id, task_id, status, created_at, cancelled_at, satellite, plane, class, lat, lon, mode,
	requested_look, requested_pass, incidence_choice, window_start, window_end,
	imaging_start, imaging_end, duration_s, incidence, look_side, pass, squint`

type scanner interface {
	Scan(dest ...any) error
}

func scanOrder(row scanner) (Order, error) {
	var o Order
	var cancelled sql.NullTime
	err := row.Scan(
		&o.ID, &o.TaskID, &o.Status, &o.CreatedAt, &cancelled, &o.Satellite, &o.Plane, &o.Class, &o.Lat, &o.Lon, &o.Mode,
		&o.RequestedLook, &o.RequestedPass, &o.IncidenceChoice, &o.WindowStart, &o.WindowEnd,
		&o.ImagingStart, &o.ImagingEnd, &o.Duration, &o.Incidence, &o.LookSide, &o.Pass, &o.Squint,
	)
	if err != nil {
		return Order{}, err
	}
	o.CreatedAt = o.CreatedAt.UTC()
	o.WindowStart = o.WindowStart.UTC()
	o.WindowEnd = o.WindowEnd.UTC()
	o.ImagingStart = o.ImagingStart.UTC()
	o.ImagingEnd = o.ImagingEnd.UTC()
	if cancelled.Valid {
		t := cancelled.Time.UTC()
		o.CancelledAt = &t
	}
	return o, nil
}

func (s *Store) empty() error {
	_, err := s.db.Exec(`TRUNCATE orders`)
	return err
}

func newID() (string, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func blank(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
