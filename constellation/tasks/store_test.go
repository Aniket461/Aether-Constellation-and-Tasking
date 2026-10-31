package tasks

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"aether/constellation/access"
	"aether/constellation/catalog"
	"aether/constellation/testdb"
)

func TestMain(m *testing.M) {
	dsn, cleanup, err := testdb.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "postgres: %v\n", err)
		os.Exit(1)
	}
	target, err := testdb.WithDatabase(dsn, "constellation")
	if err != nil {
		cleanup()
		fmt.Fprintf(os.Stderr, "dsn: %v\n", err)
		os.Exit(1)
	}
	cat, err = testdb.Load(target)
	if err != nil {
		cleanup()
		fmt.Fprintf(os.Stderr, "constellation: %v\n", err)
		os.Exit(1)
	}
	db, err = sql.Open("pgx", target)
	if err != nil {
		cleanup()
		fmt.Fprintf(os.Stderr, "db: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	_ = db.Close()
	cleanup()
	os.Exit(code)
}

var (
	cat catalog.Catalog
	db  *sql.DB
)

func TestLisbonTaskIsDoneAndStillBusy(t *testing.T) {
	t.Cleanup(clearTasks)
	ctx := context.Background()
	search := access.New(cat)
	first, err := Create(ctx, db, search, lisbon(), "order-1", Delays{})
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 10, 29, 6, 6, 51, 0, time.UTC)
	if first.Offer.Satellite != "Aether-A1" || !first.Offer.Start.Equal(want) || first.Status != "done" {
		t.Fatalf("first %+v status %s", first.Offer, first.Status)
	}
	again, err := Create(ctx, db, search, lisbon(), "order-1", Delays{})
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != first.ID || again.Status != "done" {
		t.Fatalf("same client ref %+v", again)
	}
	second, err := Create(ctx, db, search, lisbon(), "order-2", Delays{})
	if err != nil {
		t.Fatal(err)
	}
	if second.Offer.Satellite == "Aether-A1" && second.Offer.Start.Equal(want) {
		t.Fatal("finished second was booked again")
	}
	if _, err := Cancel(ctx, db, first.ID, time.Now().UTC(), Delays{}); err != ErrFinished {
		t.Fatalf("cancel done: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE id = $1`, first.ID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("row count %d err %v", n, err)
	}
}

func TestCancelFreesAScheduledTask(t *testing.T) {
	t.Cleanup(clearTasks)
	ctx := context.Background()
	search := access.New(cat)
	start := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Hour)
	req := lisbon()
	req.WindowFrom = start
	req.WindowTo = start.Add(24 * time.Hour)
	first, err := Create(ctx, db, search, req, "future-1", Delays{})
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != "scheduled" {
		t.Fatalf("status %s", first.Status)
	}
	cancelled, err := Cancel(ctx, db, first.ID, time.Now().UTC(), Delays{})
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != "cancelled" || cancelled.CancelledAt == nil {
		t.Fatalf("cancelled %+v", cancelled)
	}
	third, err := Create(ctx, db, search, req, "future-2", Delays{})
	if err != nil {
		t.Fatal(err)
	}
	if third.Offer.Satellite != first.Offer.Satellite || !third.Offer.Start.Equal(first.Offer.Start) {
		t.Fatalf("slot stayed busy: first %s %s third %s %s", first.Offer.Satellite, first.Offer.Start, third.Offer.Satellite, third.Offer.Start)
	}
}

func TestAdvanceMovesImagingToDone(t *testing.T) {
	t.Cleanup(clearTasks)
	ctx := context.Background()
	search := access.New(cat)
	created, err := Create(ctx, db, search, lisbon(), "advance-1", Delays{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE tasks SET status = 'imaging' WHERE id = $1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if err := Advance(ctx, db, time.Now().UTC(), Delays{}); err != nil {
		t.Fatal(err)
	}
	got, err := ByIDs(ctx, db, []string{created.ID}, time.Now().UTC(), Delays{})
	if err != nil || len(got) != 1 || got[0].Status != "done" {
		t.Fatalf("got %+v err %v", got, err)
	}
}

func TestResolvePhases(t *testing.T) {
	start := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	end := start.Add(10 * time.Second)
	delays := Delays{Downlink: 15 * time.Second, Processing: 20 * time.Second}
	checks := []struct {
		at   time.Time
		want string
	}{
		{start.Add(-time.Second), "scheduled"},
		{start, "imaging"},
		{end.Add(-time.Second), "imaging"},
		{end, "downlink"},
		{end.Add(14 * time.Second), "downlink"},
		{end.Add(15 * time.Second), "processing"},
		{end.Add(34 * time.Second), "processing"},
		{end.Add(35 * time.Second), "done"},
	}
	for _, check := range checks {
		if got := Resolve("scheduled", start, end, nil, check.at, delays); got != check.want {
			t.Fatalf("at %s got %s want %s", check.at.Format(time.RFC3339), got, check.want)
		}
	}
	if got := Resolve("scheduled", start, end, nil, end, Delays{Processing: 20 * time.Second}); got != "processing" {
		t.Fatalf("zero downlink got %s", got)
	}
	if got := Resolve("scheduled", start, end, nil, end.Add(20*time.Second), Delays{Processing: 20 * time.Second}); got != "done" {
		t.Fatalf("end of processing got %s", got)
	}
	if got := Resolve("scheduled", start, end, nil, end, Delays{}); got != "done" {
		t.Fatalf("zero delays got %s", got)
	}
	if got := Resolve("done", start, end, nil, start, delays); got != "done" {
		t.Fatalf("done stayed %s", got)
	}
	if got := Resolve("failed", start, end, nil, end, delays); got != "failed" {
		t.Fatalf("failed stayed %s", got)
	}
	cancelled := end
	if got := Resolve("imaging", start, end, &cancelled, start, delays); got != "cancelled" {
		t.Fatalf("cancelled stayed %s", got)
	}
}

func TestAdvanceWalksDownlinkThenProcessing(t *testing.T) {
	t.Cleanup(clearTasks)
	ctx := context.Background()
	search := access.New(cat)
	start := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Hour)
	req := lisbon()
	req.WindowFrom = start
	req.WindowTo = start.Add(24 * time.Hour)
	created, err := Create(ctx, db, search, req, "phases-1", Delays{})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	imagingStart := now.Add(-20 * time.Second)
	imagingEnd := now.Add(-1 * time.Second)
	if _, err := db.Exec(`UPDATE tasks SET imaging_start = $2, imaging_end = $3, status = 'imaging' WHERE id = $1`,
		created.ID, imagingStart, imagingEnd); err != nil {
		t.Fatal(err)
	}
	delays := Delays{Downlink: 15 * time.Second, Processing: 20 * time.Second}
	if err := Advance(ctx, db, now, delays); err != nil {
		t.Fatal(err)
	}
	got, err := ByIDs(ctx, db, []string{created.ID}, now, delays)
	if err != nil || len(got) != 1 || got[0].Status != "downlink" {
		t.Fatalf("downlink %+v %v", got, err)
	}
	spans, err := Busy(ctx, db)
	if err != nil || len(spans) != 1 || !spans[0].Start.Equal(imagingStart) || !spans[0].End.Equal(imagingEnd) {
		t.Fatalf("busy interval changed %+v %v", spans, err)
	}
	if _, err := Cancel(ctx, db, created.ID, now, delays); err != ErrFinished {
		t.Fatalf("cancel downlink: %v", err)
	}
	if err := Advance(ctx, db, imagingEnd.Add(15*time.Second), delays); err != nil {
		t.Fatal(err)
	}
	got, err = ByIDs(ctx, db, []string{created.ID}, imagingEnd.Add(15*time.Second), delays)
	if err != nil || got[0].Status != "processing" {
		t.Fatalf("processing %+v %v", got, err)
	}
	if err := Advance(ctx, db, imagingEnd.Add(35*time.Second), delays); err != nil {
		t.Fatal(err)
	}
	got, err = ByIDs(ctx, db, []string{created.ID}, imagingEnd.Add(35*time.Second), delays)
	if err != nil || got[0].Status != "done" {
		t.Fatalf("done %+v %v", got, err)
	}
}

func TestParseDelays(t *testing.T) {
	env := map[string]string{}
	got, err := ParseDelays(func(key string) string { return env[key] })
	if err != nil || got.Downlink != 15*time.Second || got.Processing != 20*time.Second {
		t.Fatalf("defaults %+v %v", got, err)
	}
	env["TASK_DOWNLINK_DELAY"] = "0s"
	env["TASK_PROCESSING_DELAY"] = "2m"
	got, err = ParseDelays(func(key string) string { return env[key] })
	if err != nil || got.Downlink != 0 || got.Processing != 2*time.Minute {
		t.Fatalf("parsed %+v %v", got, err)
	}
	env["TASK_DOWNLINK_DELAY"] = "nope"
	if _, err := ParseDelays(func(key string) string { return env[key] }); err == nil {
		t.Fatal("expected a parse error")
	}
	env["TASK_DOWNLINK_DELAY"] = "-1s"
	if _, err := ParseDelays(func(key string) string { return env[key] }); err == nil {
		t.Fatal("expected a negative error")
	}
}

func clearTasks() {
	_, _ = db.Exec(`TRUNCATE tasks`)
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
