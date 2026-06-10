package config

import (
	"context"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	// Addr is the listen address for the HTTP server.
	Addr string `env:"SOPHON_ADDR, default=:8080"`
	// DatabaseURL points at the sophon database in the shared Postgres.
	DatabaseURL string `env:"DATABASE_URL, required"`
	// LiteLLM is the OpenAI-compatible gateway for embeddings + intent parsing.
	LiteLLMBaseURL string `env:"LITELLM_BASE_URL, default=http://litellm:4000/v1"`
	LiteLLMAPIKey  string `env:"LITELLM_API_KEY"`
	EmbedModel     string `env:"SOPHON_EMBED_MODEL, default=gemini-embedding-001"`
	IntentModel    string `env:"SOPHON_INTENT_MODEL, default=tool-caller"`
	// Dev disables the Remote-User auth requirement for local runs.
	Dev bool `env:"SOPHON_DEV, default=false"`
}

func Load(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
