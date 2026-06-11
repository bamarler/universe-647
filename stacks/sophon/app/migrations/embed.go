// Package migrations embeds the versioned SQL applied at startup.
// Files are Atlas-generated from the ent schema (the Alembic flow), plus
// manual ones for DDL ent can't express (extension, fts column, HNSW index).
package migrations

import (
	"database/sql"
	"embed"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
)

//go:embed *.sql
var FS embed.FS

// Apply runs all pending migrations. A fresh database converges to the full
// schema; an up-to-date one is a no-op.
//
// It opens its own private connection: golang-migrate's driver Close() closes
// the whole *sql.DB it is handed, so sharing the app's pool is unsafe.
func Apply(databaseURL string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	src, err := iofs.New(FS, ".")
	if err != nil {
		db.Close()
		return err
	}
	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{})
	if err != nil {
		db.Close()
		return err
	}
	m, err := migrate.NewWithInstance("iofs", src, "sophon", driver)
	if err != nil {
		db.Close()
		return err
	}
	defer m.Close() // closes driver + the private db
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
