// Package app owns construction: one explicit InitApp builds every dependency
// (manual constructor DI — the GenerateNU house style). Optional infra
// degrades gracefully instead of failing boot (the selfserve tryInit pattern).
package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	entsql "entgo.io/ent/dialect/sql"

	"entgo.io/ent/dialect"
	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/bamarler/universe-647/sophon/internal/api"
	"github.com/bamarler/universe-647/sophon/internal/config"
	"github.com/bamarler/universe-647/sophon/internal/ent"
	"github.com/bamarler/universe-647/sophon/internal/llm"
	"github.com/bamarler/universe-647/sophon/migrations"
)

type App struct {
	Config  *config.Config
	Ent     *ent.Client
	LLM     llm.Client
	Handler http.Handler
}

func Init(ctx context.Context, cfg *config.Config, log *slog.Logger) (*App, error) {
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	db := stdlib.OpenDBFromPool(pool)

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	entClient := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))

	llmClient := tryInitLLM(cfg, log)

	router, _ := api.New(api.Deps{
		Ent: entClient,
		LLM: llmClient,
		Dev: cfg.Dev,
	})

	return &App{
		Config:  cfg,
		Ent:     entClient,
		LLM:     llmClient,
		Handler: router,
	}, nil
}

func (a *App) Close() error {
	return a.Ent.Close()
}

// runMigrations applies the embedded versioned migrations. A fresh database
// converges to the full schema; an up-to-date one is a no-op.
func runMigrations(db *sql.DB) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}
	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", src, "sophon", driver)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// tryInitLLM never fails the boot: with LiteLLM down, sophon still serves
// browse/CRUD and only semantic features report unavailable.
func tryInitLLM(cfg *config.Config, log *slog.Logger) llm.Client {
	client, err := llm.New(cfg.LiteLLMBaseURL, cfg.LiteLLMAPIKey, cfg.EmbedModel)
	if err != nil {
		log.Warn("llm gateway unavailable, semantic features disabled", "error", err)
		return llm.Disabled{}
	}
	return client
}
