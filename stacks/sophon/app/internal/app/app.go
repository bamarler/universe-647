// Package app owns construction: one explicit InitApp builds every dependency
// (manual constructor DI — the GenerateNU house style). Optional infra
// degrades gracefully instead of failing boot (the selfserve tryInit pattern).
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	entsql "entgo.io/ent/dialect/sql"

	"entgo.io/ent/dialect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/bamarler/universe-647/sophon/internal/api"
	"github.com/bamarler/universe-647/sophon/internal/config"
	"github.com/bamarler/universe-647/sophon/internal/ent"
	"github.com/bamarler/universe-647/sophon/internal/index"
	"github.com/bamarler/universe-647/sophon/internal/llm"
	"github.com/bamarler/universe-647/sophon/internal/search"
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

	if err := migrations.Apply(cfg.DatabaseURL); err != nil {
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	db := stdlib.OpenDBFromPool(pool)

	entClient := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))

	llmClient := tryInitLLM(cfg, log)

	indexer := index.New(entClient, llmClient, log)
	indexer.Start(ctx)

	router, _ := api.New(api.Deps{
		Ent:      entClient,
		LLM:      llmClient,
		Indexer:  indexer,
		Searcher: search.New(db, llmClient),
		Dev:      cfg.Dev,
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

// tryInitLLM never fails the boot: with the gateway down, sophon still serves
// browse/CRUD and only semantic features report unavailable.
func tryInitLLM(cfg *config.Config, log *slog.Logger) llm.Client {
	client, err := llm.New(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.EmbedModel, cfg.IntentModel)
	if err != nil {
		log.Warn("llm gateway unavailable, semantic features disabled", "error", err)
		return llm.Disabled{}
	}
	return client
}
