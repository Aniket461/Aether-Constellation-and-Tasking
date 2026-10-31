package catalog_test

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

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
	testDSN = target
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

var (
	testDSN string
	cat     catalog.Catalog
)

func TestModesFollowTheClass(t *testing.T) {
	if cat.Version != 1 {
		t.Fatalf("version %d", cat.Version)
	}
	if len(cat.Satellites) != 12 || len(cat.Modes) != 9 {
		t.Fatalf("satellites %d modes %d", len(cat.Satellites), len(cat.Modes))
	}
	spot, ok := mode(cat, "SPOTLIGHT")
	if !ok || spot.Seconds != 10 || spot.Label != "Spot" {
		t.Fatalf("spotlight %+v", spot)
	}
	if !hasClass(spot, "Fine") || hasClass(spot, "Wide") {
		t.Fatalf("spotlight classes %v", spot.Classes)
	}
	wide, ok := mode(cat, "SCAN_WIDE")
	if !ok || wide.Seconds != 42 || !hasClass(wide, "Wide") || len(wide.Classes) != 1 {
		t.Fatalf("scan wide %+v", wide)
	}
}

func TestModeChangeBumpsTheVersion(t *testing.T) {
	db, err := sql.Open("pgx", testDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`UPDATE imaging_modes SET seconds = 12 WHERE id = 'SPOTLIGHT'`); err != nil {
		t.Fatal(err)
	}
	updated, err := catalog.Load(db)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 2 {
		t.Fatalf("version %d", updated.Version)
	}
	spot, ok := mode(updated, "SPOTLIGHT")
	if !ok || spot.Seconds != 12 {
		t.Fatalf("spotlight %+v", spot)
	}
	if len(updated.Satellites) != 12 {
		t.Fatalf("satellites %d", len(updated.Satellites))
	}
}

func mode(cat catalog.Catalog, id string) (catalog.Mode, bool) {
	for _, item := range cat.Modes {
		if item.ID == id {
			return item, true
		}
	}
	return catalog.Mode{}, false
}

func hasClass(mode catalog.Mode, name string) bool {
	for _, class := range mode.Classes {
		if class == name {
			return true
		}
	}
	return false
}
