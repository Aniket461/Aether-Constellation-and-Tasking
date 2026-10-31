package orders

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"aether/constellation/access"
	"aether/constellation/catalog"
	"aether/constellation/tasks"
	"aether/constellation/testdb"
)

func TestMain(m *testing.M) {
	dsn, cleanup, err := testdb.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "postgres: %v\n", err)
		os.Exit(1)
	}
	constellationDSN, err = testdb.WithDatabase(dsn, "constellation")
	if err != nil {
		cleanup()
		fmt.Fprintf(os.Stderr, "dsn: %v\n", err)
		os.Exit(1)
	}
	cat, err = testdb.Load(constellationDSN)
	if err != nil {
		cleanup()
		fmt.Fprintf(os.Stderr, "constellation: %v\n", err)
		os.Exit(1)
	}
	taskDSN, err = testdb.WithDatabase(dsn, "tasking")
	if err != nil {
		cleanup()
		fmt.Fprintf(os.Stderr, "dsn: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	cleanup()
	os.Exit(code)
}

var (
	taskDSN          string
	constellationDSN string
	cat              catalog.Catalog
)

func TestPlaceReservesTheSlot(t *testing.T) {
	store := openTest(t)
	req := lisbon()
	first, err := store.Place(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if first.Satellite != "Aether-A1" || !first.ImagingStart.Equal(time.Date(2026, 10, 29, 6, 6, 51, 0, time.UTC)) {
		t.Fatalf("first order %+v", first)
	}
	if first.Plane != "Dawn" || first.Class != "Standard" || first.Mode != "SPOTLIGHT" || first.Duration != 10 || first.TaskID == "" {
		t.Fatalf("snapshot %+v", first)
	}
	if first.Status != "done" {
		t.Fatalf("status %s", first.Status)
	}

	second, err := store.Place(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if second.Satellite == first.Satellite && second.ImagingStart.Equal(first.ImagingStart) {
		t.Fatal("reserved second was offered again")
	}

	if _, err := store.Cancel(context.Background(), first.ID); !errors.Is(err, ErrFinished) {
		t.Fatalf("done task should stay done: %v", err)
	}
}

func TestCancelFinished(t *testing.T) {
	store := openTest(t)
	order, err := store.Place(context.Background(), lisbon())
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Cancel(context.Background(), order.ID)
	if !errors.Is(err, ErrFinished) {
		t.Fatalf("got %v", err)
	}
}

func openTest(t *testing.T) *Store {
	t.Helper()
	fleetDB, err := sql.Open("pgx", constellationDSN)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fleetDB.Exec(`TRUNCATE tasks`); err != nil {
		t.Fatal(err)
	}
	store, err := Open(taskDSN, fleet{db: fleetDB, search: access.New(cat)})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.empty(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = store.Close()
		_ = fleetDB.Close()
	})
	return store
}

type fleet struct {
	db     *sql.DB
	search *access.Searcher
}

func (f fleet) Earliest(ctx context.Context, req access.Request) (access.Offer, bool, error) {
	spans, err := tasks.Busy(ctx, f.db)
	if err != nil {
		return access.Offer{}, false, err
	}
	offer, ok := f.search.Earliest(req, spans)
	return offer, ok, nil
}

func (f fleet) Create(ctx context.Context, req access.Request, clientRef string) (tasks.Task, error) {
	return tasks.Create(ctx, f.db, f.search, req, clientRef, tasks.Delays{})
}

func (f fleet) Cancel(ctx context.Context, id string) (tasks.Task, error) {
	return tasks.Cancel(ctx, f.db, id, time.Now().UTC(), tasks.Delays{})
}

func (f fleet) Query(ctx context.Context, ids []string) ([]tasks.Task, error) {
	return tasks.ByIDs(ctx, f.db, ids, time.Now().UTC(), tasks.Delays{})
}

func lisbon() access.Request {
	return access.Request{
		Lat:        38.7223,
		Lon:        -9.1393,
		WindowFrom: time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC),
		WindowTo:   time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC),
		Mode:       "SPOTLIGHT",
		LookSide:   "ANY",
		Pass:       "ANY",
		Incidence:  "performant",
	}
}
