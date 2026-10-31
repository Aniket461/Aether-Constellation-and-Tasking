package testdb

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"aether/constellation/catalog"
	"aether/constellation/schema"
)

// Start runs a throwaway Postgres and returns a connection string for its default database.
func Start() (string, func(), error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return "", nil, err
	}
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", nil, err
	}
	name := "aether-pg-" + hex.EncodeToString(buf[:])
	out, err := exec.Command("docker", "run", "-d", "--rm", "--name", name,
		"-e", "POSTGRES_USER=aether",
		"-e", "POSTGRES_PASSWORD=aether",
		"-e", "POSTGRES_DB=aether",
		"-p", "127.0.0.1::5432",
		"postgres:17-alpine",
	).CombinedOutput()
	if err != nil {
		return "", nil, fmt.Errorf("%w: %s", err, out)
	}
	cleanup := func() {
		_ = exec.Command("docker", "rm", "-f", name).Run()
	}
	portOut, err := exec.Command("docker", "port", name, "5432/tcp").Output()
	if err != nil {
		cleanup()
		return "", nil, err
	}
	line := strings.TrimSpace(strings.Split(string(portOut), "\n")[0])
	colon := strings.LastIndex(line, ":")
	if colon < 0 {
		cleanup()
		return "", nil, fmt.Errorf("docker port: %q", line)
	}
	dsn := "postgres://aether:aether@127.0.0.1:" + line[colon+1:] + "/aether?sslmode=disable"
	if err := wait(dsn); err != nil {
		cleanup()
		return "", nil, err
	}
	return dsn, cleanup, nil
}

// WithDatabase points a connection string at another database on the same server.
func WithDatabase(dsn, name string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	u.Path = "/" + name
	return u.String(), nil
}

// Load creates the database when needed, migrates it, and reads the fleet.
func Load(dsn string) (catalog.Catalog, error) {
	if err := schema.Ensure(dsn); err != nil {
		return catalog.Catalog{}, err
	}
	if err := schema.Up(dsn); err != nil {
		return catalog.Catalog{}, err
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return catalog.Catalog{}, err
	}
	defer db.Close()
	return catalog.Load(db)
}

func wait(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	deadline := time.Now().Add(30 * time.Second)
	var pingErr error
	for time.Now().Before(deadline) {
		pingErr = db.Ping()
		if pingErr == nil {
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("postgres did not start: %w", pingErr)
}
