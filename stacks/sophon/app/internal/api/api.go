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
	"github.com/bamarler/universe-647/sophon/internal/index"
	"github.com/bamarler/universe-647/sophon/internal/llm"
	"github.com/bamarler/universe-647/sophon/internal/search"
)

type Deps struct {
	Ent      *ent.Client
	LLM      llm.Client
	Indexer  *index.Indexer
	Searcher *search.Searcher
	// Dev disables the Remote-User requirement for local runs.
	Dev bool
}

// New returns the chi router with the full API mounted under /api.
func New(deps Deps) (chi.Router, huma.API) {
	router := chi.NewRouter()

	cfg := huma.DefaultConfig("Sophon", "0.2.0")
	cfg.Info.Description = "Retrieval-only second brain: tasks, notes, supertags, hybrid search."
	cfg.Servers = []*huma.Server{{URL: "/api"}}
	cfg.OpenAPIPath = "/api/openapi"
	cfg.DocsPath = "/api/docs"

	humaAPI := humachi.New(router, cfg)

	// Public routes first.
	registerHealth(humaAPI)

	// Everything registered after this line requires an authenticated user.
	humaAPI.UseMiddleware(authMiddleware(humaAPI, deps.Dev))

	registerTree(humaAPI, deps)
	registerTags(humaAPI, deps)
	registerTasks(humaAPI, deps)
	registerNotes(humaAPI, deps)
	registerFolders(humaAPI, deps)
	registerViews(humaAPI, deps)
	registerSearch(humaAPI, deps)
	registerCommand(humaAPI, deps)
	registerReindex(humaAPI, deps)

	return router, humaAPI
}
