// Package testdb provisions an isolated database per test against the local
// dev Postgres (./dev/devdb.sh up). Tests skip when it isn't running, so
// `go test ./...` stays green without Docker.
package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"testing"

	entsql "entgo.io/ent/dialect/sql"

	"entgo.io/ent/dialect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/bamarler/universe-647/sophon/internal/ent"
	"github.com/bamarler/universe-647/sophon/migrations"
)

const adminURL = "postgres://postgres:dev@localhost:5499/dev?sslmode=disable"

// Open creates a fresh database, applies migrations, and returns an ent
// client plus the raw DB handle. Everything is cleaned up with the test.
func Open(t *testing.T) (*ent.Client, *sql.DB) {
	t.Helper()
	ctx := context.Background()

	admin, err := sql.Open("pgx", adminURL)
	if err != nil {
		t.Skipf("dev database unavailable: %v (run ./dev/devdb.sh up)", err)
	}
	if err := admin.PingContext(ctx); err != nil {
		t.Skipf("dev database unavailable: %v (run ./dev/devdb.sh up)", err)
	}

	name := fmt.Sprintf("t_%d", rand.Int63())
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)")
		_ = admin.Close()
	})

	url := fmt.Sprintf("postgres://postgres:dev@localhost:5499/%s?sslmode=disable", name)
	if err := migrations.Apply(url); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	db := stdlib.OpenDBFromPool(pool)
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() {
		_ = client.Close()
		pool.Close()
	})
	return client, db
}
