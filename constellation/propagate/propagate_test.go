package propagate

import (
	"fmt"
	"math"
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

func TestAltitudesNearDesign(t *testing.T) {
	prop := New(cat.Satellites)
	tracks := prop.Ephemeris(cat.Epoch, cat.Epoch.Add(30*time.Minute), 60*time.Second)
	if len(tracks) != 12 {
		t.Fatalf("got %d satellites", len(tracks))
	}
	for _, track := range tracks {
		if len(track.Samples) < 2 {
			t.Fatalf("%s has no samples", track.Name)
		}
		for _, sample := range track.Samples {
			if sample.AltKm < 470 || sample.AltKm > 620 {
				t.Fatalf("%s altitude %.1f km at %s", track.Name, sample.AltKm, sample.Time)
			}
			if math.Abs(sample.Lat) > 83 {
				t.Fatalf("%s latitude %.2f", track.Name, sample.Lat)
			}
			if sample.Lon < -180 || sample.Lon > 180 {
				t.Fatalf("%s longitude %.2f", track.Name, sample.Lon)
			}
			radius := math.Hypot(sample.EcefX, math.Hypot(sample.EcefY, sample.EcefZ))
			if math.Abs(radius-(6378.137+sample.AltKm)) > 30 {
				t.Fatalf("%s ecef radius %.1f vs altitude %.1f", track.Name, radius, sample.AltKm)
			}
		}
	}
}
