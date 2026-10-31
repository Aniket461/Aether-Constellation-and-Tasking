package schema

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"embed"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var files embed.FS

var safeName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// Ensure creates the database named in the connection string when it is missing.
func Ensure(dsn string) error {
	u, err := url.Parse(dsn)
	if err != nil {
		return err
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" || name == "postgres" {
		return nil
	}
	if !safeName.MatchString(name) {
		return fmt.Errorf("database name %q", name)
	}
	admin := *u
	admin.Path = "/postgres"
	db, err := sql.Open("pgx", admin.String())
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	var exists bool
	if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, name).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = db.Exec(`CREATE DATABASE ` + name)
	return err
}

// Up applies pending SQL migrations. A database that is already current is left as it is.
func Up(dsn string) error {
	source, err := iofs.New(files, "migrations")
	if err != nil {
		return err
	}
	migrator, err := migrate.NewWithSourceInstance("iofs", source, dsn)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = migrator.Close()
	}()

	err = migrator.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: %w", err)
	}
	version, dirty, err := migrator.Version()
	if err != nil {
		return fmt.Errorf("migrate version: %w", err)
	}
	if dirty {
		return fmt.Errorf("migrate: version %d is dirty", version)
	}
	log.Printf("constellation schema version %d", version)
	return nil
}
