package tasks

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"aether/constellation/access"
)

// ErrNotFeasible means no free satellite can take the picture.
var ErrNotFeasible = errors.New("not feasible")

// ErrNotFound means the task id is unknown.
var ErrNotFound = errors.New("not found")

// ErrFinished means the satellite has already finished the task, so it cannot be cancelled.
var ErrFinished = errors.New("already finished")

// Task is one collection the fleet accepted. Status is the work the satellite is doing.
type Task struct {
	ID              string
	ClientRef       string
	Status          string
	CreatedAt       time.Time
	CancelledAt     *time.Time
	Lat             float64
	Lon             float64
	Mode            string
	RequestedLook   string
	RequestedPass   string
	IncidenceChoice string
	WindowStart     time.Time
	WindowEnd       time.Time
	Offer           access.Offer
}

// moving is every status the clock can still change. done, cancelled, and failed stay.
const moving = `'scheduled', 'imaging', 'downlink', 'processing'`

// Resolve is the constellation's status rule.
// failed and cancelled stay. Imaging occupies [start, end). Downlink and processing
// follow the imaging end and do not extend the busy interval. A zero delay skips that phase.
// A task already marked done stays done.
func Resolve(status string, start, end time.Time, cancelled *time.Time, now time.Time, delays Delays) string {
	if status == "failed" {
		return "failed"
	}
	if status == "cancelled" || cancelled != nil {
		return "cancelled"
	}
	if status == "done" {
		return "done"
	}
	delays = delays.clamp()
	if now.Before(start) {
		return "scheduled"
	}
	if now.Before(end) {
		return "imaging"
	}
	downlinkEnd := end.Add(delays.Downlink)
	if now.Before(downlinkEnd) {
		return "downlink"
	}
	if now.Before(downlinkEnd.Add(delays.Processing)) {
		return "processing"
	}
	return "done"
}

// Advance stores the status of every task the clock can still change.
func Advance(ctx context.Context, db *sql.DB, now time.Time, delays Delays) error {
	delays = delays.clamp()
	phase := `
		CASE
		  WHEN imaging_end + make_interval(secs => $2) + make_interval(secs => $3) <= $1 THEN 'done'
		  WHEN imaging_end + make_interval(secs => $2) <= $1 THEN 'processing'
		  WHEN imaging_end <= $1 THEN 'downlink'
		  WHEN imaging_start <= $1 THEN 'imaging'
		  ELSE 'scheduled'
		END`
	_, err := db.ExecContext(ctx, `
		UPDATE tasks
		SET status = `+phase+`
		WHERE status IN (`+moving+`)
		  AND status IS DISTINCT FROM `+phase, now.UTC(), delays.Downlink.Seconds(), delays.Processing.Seconds())
	return err
}

// Busy returns intervals the search must skip. Cancelled and failed tasks are free.
func Busy(ctx context.Context, db *sql.DB) ([]access.Span, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT satellite, imaging_start, imaging_end FROM tasks
		WHERE status NOT IN ('cancelled', 'failed')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSpans(rows)
}

// Create finds the earliest free collection and stores the task in one transaction.
// The same clientRef returns the task already stored for it.
func Create(ctx context.Context, db *sql.DB, search *access.Searcher, req access.Request, clientRef string, delays Delays) (Task, error) {
	if !search.Knows(req.Mode) {
		return Task{}, &access.Invalid{Msg: "unknown imaging mode"}
	}
	for attempt := 0; attempt < 5; attempt++ {
		created, err := insert(ctx, db, search, req, clientRef, delays)
		if errors.Is(err, errOverlap) {
			continue
		}
		if errors.Is(err, errSameClient) {
			return loadByClientRef(ctx, db, clientRef, delays)
		}
		return created, err
	}
	return Task{}, ErrNotFeasible
}

// ByIDs returns the tasks for those ids, with status resolved at now.
func ByIDs(ctx context.Context, db *sql.DB, ids []string, now time.Time, delays Delays) ([]Task, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	holders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		holders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	rows, err := db.QueryContext(ctx, `SELECT `+columns+` FROM tasks WHERE id IN (`+strings.Join(holders, ",")+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		if err := persistResolved(ctx, db, &task, now, delays); err != nil {
			return nil, err
		}
		out = append(out, task)
	}
	return out, rows.Err()
}

// Cancel marks the task cancelled when the satellite has not finished it.
func Cancel(ctx context.Context, db *sql.DB, id string, now time.Time, delays Delays) (Task, error) {
	task, err := loadByID(ctx, db, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	if err != nil {
		return Task{}, err
	}
	if err := persistResolved(ctx, db, &task, now, delays); err != nil {
		return Task{}, err
	}
	if task.Status == "cancelled" {
		return task, nil
	}
	if task.Status != "scheduled" && task.Status != "imaging" {
		return task, ErrFinished
	}
	stamp := now.UTC().Truncate(time.Second)
	res, err := db.ExecContext(ctx, `
		UPDATE tasks SET status = 'cancelled', cancelled_at = $1
		WHERE id = $2 AND status IN ('scheduled', 'imaging')`, stamp, id)
	if err != nil {
		return Task{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		task, err = loadByID(ctx, db, id)
		if err != nil {
			return Task{}, err
		}
		if task.Status == "cancelled" {
			return task, nil
		}
		return task, ErrFinished
	}
	return loadByID(ctx, db, id)
}

func insert(ctx context.Context, db *sql.DB, search *access.Searcher, req access.Request, clientRef string, delays Delays) (Task, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback()

	spans, err := busyTx(ctx, tx)
	if err != nil {
		return Task{}, err
	}
	offer, ok := search.Earliest(req, spans)
	if !ok {
		return Task{}, ErrNotFeasible
	}
	id, err := newID()
	if err != nil {
		return Task{}, err
	}
	created := time.Now().UTC().Truncate(time.Second)
	status := Resolve("scheduled", offer.Start, offer.End, nil, created, delays)
	var ref any
	if clientRef != "" {
		ref = clientRef
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO tasks (
		  id, satellite, plane, class, imaging_start, imaging_end,
		  duration_s, incidence, look_side, pass, squint, client_ref, created_at,
		  status, lat, lon, mode, requested_look, requested_pass, incidence_choice,
		  window_start, window_end
		) VALUES (
		  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
		  $14, $15, $16, $17, $18, $19, $20, $21, $22
		)`,
		id, offer.Satellite, offer.Plane, offer.Class,
		offer.Start.UTC(), offer.End.UTC(),
		offer.Duration, offer.Incidence, offer.LookSide, offer.Pass, offer.Squint,
		ref, created,
		status, req.Lat, req.Lon, req.Mode, blank(req.LookSide, "ANY"), blank(req.Pass, "ANY"), blank(req.Incidence, "performant"),
		req.WindowFrom.UTC(), req.WindowTo.UTC(),
	)
	if isExclusion(err) {
		return Task{}, errOverlap
	}
	if isUnique(err) {
		return Task{}, errSameClient
	}
	if err != nil {
		return Task{}, err
	}
	if err := tx.Commit(); err != nil {
		if isExclusion(err) {
			return Task{}, errOverlap
		}
		if isUnique(err) {
			return Task{}, errSameClient
		}
		return Task{}, err
	}
	return loadByID(ctx, db, id)
}

func persistResolved(ctx context.Context, db *sql.DB, task *Task, now time.Time, delays Delays) error {
	next := Resolve(task.Status, task.Offer.Start, task.Offer.End, task.CancelledAt, now, delays)
	if next == task.Status {
		return nil
	}
	res, err := db.ExecContext(ctx, `
		UPDATE tasks SET status = $1
		WHERE id = $2 AND status IN (`+moving+`)`, next, task.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 1 {
		task.Status = next
	}
	return nil
}

func loadByID(ctx context.Context, db *sql.DB, id string) (Task, error) {
	row := db.QueryRowContext(ctx, `SELECT `+columns+` FROM tasks WHERE id = $1`, id)
	task, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	return task, err
}

func loadByClientRef(ctx context.Context, db *sql.DB, clientRef string, delays Delays) (Task, error) {
	row := db.QueryRowContext(ctx, `SELECT `+columns+` FROM tasks WHERE client_ref = $1`, clientRef)
	task, err := scanTask(row)
	if err != nil {
		return Task{}, err
	}
	if err := persistResolved(ctx, db, &task, time.Now().UTC(), delays); err != nil {
		return Task{}, err
	}
	return task, nil
}

const columns = `id, client_ref, status, created_at, cancelled_at, lat, lon, mode,
	requested_look, requested_pass, incidence_choice, window_start, window_end,
	satellite, plane, class, imaging_start, imaging_end, duration_s, incidence, look_side, pass, squint`

func busyTx(ctx context.Context, tx *sql.Tx) ([]access.Span, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT satellite, imaging_start, imaging_end FROM tasks
		WHERE status NOT IN ('cancelled', 'failed')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSpans(rows)
}

func scanSpans(rows *sql.Rows) ([]access.Span, error) {
	var out []access.Span
	for rows.Next() {
		var span access.Span
		if err := rows.Scan(&span.Satellite, &span.Start, &span.End); err != nil {
			return nil, err
		}
		span.Start = span.Start.UTC()
		span.End = span.End.UTC()
		out = append(out, span)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanTask(row scanner) (Task, error) {
	var task Task
	var ref sql.NullString
	var cancelled sql.NullTime
	err := row.Scan(
		&task.ID, &ref, &task.Status, &task.CreatedAt, &cancelled, &task.Lat, &task.Lon, &task.Mode,
		&task.RequestedLook, &task.RequestedPass, &task.IncidenceChoice, &task.WindowStart, &task.WindowEnd,
		&task.Offer.Satellite, &task.Offer.Plane, &task.Offer.Class, &task.Offer.Start, &task.Offer.End,
		&task.Offer.Duration, &task.Offer.Incidence, &task.Offer.LookSide, &task.Offer.Pass, &task.Offer.Squint,
	)
	if err != nil {
		return Task{}, err
	}
	task.CreatedAt = task.CreatedAt.UTC()
	task.WindowStart = task.WindowStart.UTC()
	task.WindowEnd = task.WindowEnd.UTC()
	task.Offer.Start = task.Offer.Start.UTC()
	task.Offer.End = task.Offer.End.UTC()
	if ref.Valid {
		task.ClientRef = ref.String
	}
	if cancelled.Valid {
		t := cancelled.Time.UTC()
		task.CancelledAt = &t
	}
	return task, nil
}

// Marshal is the JSON for a task response.
func Marshal(task Task) ([]byte, error) {
	return json.Marshal(taskJSON(task))
}

// MarshalAll encodes a list of tasks.
func MarshalAll(list []Task) ([]byte, error) {
	body := make([]any, len(list))
	for i, task := range list {
		body[i] = taskJSON(task)
	}
	return json.Marshal(body)
}

func taskJSON(task Task) map[string]any {
	var cancelled any
	if task.CancelledAt != nil {
		cancelled = task.CancelledAt.UTC().Format(time.RFC3339)
	}
	return map[string]any{
		"id":              task.ID,
		"clientRef":       task.ClientRef,
		"status":          task.Status,
		"createdAt":       task.CreatedAt.UTC().Format(time.RFC3339),
		"cancelledAt":     cancelled,
		"satellite":       task.Offer.Satellite,
		"plane":           task.Offer.Plane,
		"class":           task.Offer.Class,
		"pointOfInterest": map[string]float64{"lat": task.Lat, "lon": task.Lon},
		"imagingMode":     task.Mode,
		"lookSide":        task.RequestedLook,
		"passDirection":   task.RequestedPass,
		"incidence":       task.IncidenceChoice,
		"acquisitionWindow": map[string]string{
			"start": task.WindowStart.UTC().Format(time.RFC3339),
			"end":   task.WindowEnd.UTC().Format(time.RFC3339),
		},
		"imagingTime": map[string]string{
			"start": task.Offer.Start.UTC().Format(time.RFC3339),
			"end":   task.Offer.End.UTC().Format(time.RFC3339),
		},
		"durationSeconds":  task.Offer.Duration,
		"incidenceAngle":   task.Offer.Incidence,
		"computedLookSide": task.Offer.LookSide,
		"computedPass":     task.Offer.Pass,
		"squintDeg":        task.Offer.Squint,
	}
}

// Unmarshal reads one task response.
func Unmarshal(raw []byte) (Task, error) {
	list, err := UnmarshalAll(raw)
	if err != nil {
		return Task{}, err
	}
	if len(list) != 1 {
		var task Task
		if err := decode(raw, &task); err != nil {
			return Task{}, err
		}
		return task, nil
	}
	return list[0], nil
}

// UnmarshalAll reads a list of tasks. A single task object is accepted too.
func UnmarshalAll(raw []byte) ([]Task, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	if raw[0] == '[' {
		var body []json.RawMessage
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, err
		}
		out := make([]Task, len(body))
		for i, item := range body {
			task, err := decodeTask(item)
			if err != nil {
				return nil, err
			}
			out[i] = task
		}
		return out, nil
	}
	task, err := decodeTask(raw)
	if err != nil {
		return nil, err
	}
	return []Task{task}, nil
}

func decode(raw []byte, task *Task) error {
	decoded, err := decodeTask(raw)
	if err != nil {
		return err
	}
	*task = decoded
	return nil
}

func decodeTask(raw []byte) (Task, error) {
	var body struct {
		ID              string  `json:"id"`
		ClientRef       string  `json:"clientRef"`
		Status          string  `json:"status"`
		CreatedAt       string  `json:"createdAt"`
		CancelledAt     *string `json:"cancelledAt"`
		Satellite       string  `json:"satellite"`
		Plane           string  `json:"plane"`
		Class           string  `json:"class"`
		PointOfInterest struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"pointOfInterest"`
		ImagingMode       string `json:"imagingMode"`
		LookSide          string `json:"lookSide"`
		PassDirection     string `json:"passDirection"`
		Incidence         string `json:"incidence"`
		AcquisitionWindow struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"acquisitionWindow"`
		ImagingTime struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"imagingTime"`
		DurationSeconds  int     `json:"durationSeconds"`
		IncidenceAngle   float64 `json:"incidenceAngle"`
		ComputedLookSide string  `json:"computedLookSide"`
		ComputedPass     string  `json:"computedPass"`
		SquintDeg        float64 `json:"squintDeg"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return Task{}, err
	}
	created, err := time.Parse(time.RFC3339Nano, body.CreatedAt)
	if err != nil {
		return Task{}, err
	}
	windowStart, err := time.Parse(time.RFC3339Nano, body.AcquisitionWindow.Start)
	if err != nil {
		return Task{}, err
	}
	windowEnd, err := time.Parse(time.RFC3339Nano, body.AcquisitionWindow.End)
	if err != nil {
		return Task{}, err
	}
	start, err := time.Parse(time.RFC3339Nano, body.ImagingTime.Start)
	if err != nil {
		return Task{}, err
	}
	end, err := time.Parse(time.RFC3339Nano, body.ImagingTime.End)
	if err != nil {
		return Task{}, err
	}
	task := Task{
		ID: body.ID, ClientRef: body.ClientRef, Status: body.Status, CreatedAt: created.UTC(),
		Lat: body.PointOfInterest.Lat, Lon: body.PointOfInterest.Lon, Mode: body.ImagingMode,
		RequestedLook: body.LookSide, RequestedPass: body.PassDirection, IncidenceChoice: body.Incidence,
		WindowStart: windowStart.UTC(), WindowEnd: windowEnd.UTC(),
		Offer: access.Offer{
			Satellite: body.Satellite, Plane: body.Plane, Class: body.Class,
			Start: start.UTC(), End: end.UTC(), Duration: body.DurationSeconds,
			Incidence: body.IncidenceAngle, LookSide: body.ComputedLookSide, Pass: body.ComputedPass,
			Squint: body.SquintDeg,
		},
	}
	if body.CancelledAt != nil && *body.CancelledAt != "" {
		stamp, err := time.Parse(time.RFC3339Nano, *body.CancelledAt)
		if err != nil {
			return Task{}, err
		}
		t := stamp.UTC()
		task.CancelledAt = &t
	}
	return task, nil
}

var (
	errOverlap    = errors.New("overlap")
	errSameClient = errors.New("same client")
)

func isExclusion(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23P01"
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
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
