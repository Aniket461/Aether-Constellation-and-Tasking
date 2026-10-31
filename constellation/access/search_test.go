package access

import (
	"fmt"
	"os"
	"testing"
	"time"

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
	code := m.Run()
	cleanup()
	os.Exit(code)
}

var cat catalog.Catalog

func TestLisbonSpotEarliest(t *testing.T) {
	s := New(cat)
	offer, ok := s.Earliest(lisbon(), nil)
	if !ok {
		t.Fatal("expected a Spot pass over Lisbon")
	}
	want := time.Date(2026, 10, 29, 6, 6, 51, 0, time.UTC)
	if !offer.Start.Equal(want) || offer.Satellite != "Aether-A1" {
		t.Fatalf("got %s at %s, want Aether-A1 at %s", offer.Satellite, offer.Start, want)
	}
	if offer.LookSide != "RIGHT" || offer.Pass != "ASCENDING" {
		t.Fatalf("geometry %s %s", offer.LookSide, offer.Pass)
	}
	if offer.Incidence < 26 || offer.Incidence > 26.2 {
		t.Fatalf("incidence %f", offer.Incidence)
	}
	if abs(offer.Squint) > 1 {
		t.Fatalf("squint %f", offer.Squint)
	}
	if offer.End.Sub(offer.Start) != 10*time.Second || offer.Duration != 10 {
		t.Fatalf("duration %+v", offer)
	}
}

func TestLisbonLeftSpotNotInWindow(t *testing.T) {
	s := New(cat)
	req := lisbon()
	req.LookSide = "LEFT"
	_, ok := s.Earliest(req, nil)
	if ok {
		t.Fatal("left-looking Spot of Lisbon is outside this window")
	}
}

func TestBusySatelliteIsSkipped(t *testing.T) {
	s := New(cat)
	busy := []Span{{
		Satellite: "Aether-A1",
		Start:     time.Date(2026, 10, 29, 6, 6, 51, 0, time.UTC),
		End:       time.Date(2026, 10, 29, 6, 7, 1, 0, time.UTC),
	}}
	offer, ok := s.Earliest(lisbon(), busy)
	if !ok {
		t.Fatal("another satellite should still be able to image Lisbon")
	}
	if offer.Satellite == "Aether-A1" && offer.Start.Equal(busy[0].Start) {
		t.Fatal("reserved Aether-A1 second was offered again")
	}
}

func lisbon() Request {
	return Request{
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
