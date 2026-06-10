// Package api wires the huma v2 API over chi. Route registration order
// matters: public routes (healthz) are registered before the auth middleware
// is installed, so only later registrations require identity (the skillspark
// ordering pattern).
package api

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/bamarler/universe-647/sophon/internal/ent"
	"github.com/bamarler/universe-647/sophon/internal/llm"
)

type Deps struct {
	Ent *ent.Client
	LLM llm.Client
	// Dev disables the Remote-User requirement for local runs.
	Dev bool
}

// New returns the chi router with the full API mounted under /api.
func New(deps Deps) (chi.Router, huma.API) {
	router := chi.NewRouter()

	cfg := huma.DefaultConfig("Sophon", "0.1.0")
	cfg.Info.Description = "Retrieval-only second brain: tasks, notes, tags, hybrid search."
	cfg.Servers = []*huma.Server{{URL: "/api"}}
	cfg.OpenAPIPath = "/api/openapi"
	cfg.DocsPath = "/api/docs"

	humaAPI := humachi.New(router, cfg)

	// Public routes first.
	registerHealth(humaAPI)

	// Everything registered after this line requires an authenticated user.
	humaAPI.UseMiddleware(authMiddleware(humaAPI, deps.Dev))

	registerTree(humaAPI, deps)

	return router, humaAPI
}
