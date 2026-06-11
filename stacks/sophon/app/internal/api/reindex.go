package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/bamarler/universe-647/sophon/internal/errs"
)

type reindexOutput struct {
	Body struct {
		Indexed int `json:"indexed"`
		Skipped int `json:"skipped" doc:"unchanged chunks (hash hit, no re-embed)"`
	}
}

func registerReindex(api huma.API, deps Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "reindex", Method: http.MethodPost, Path: "/api/admin/reindex",
		Summary: "Rebuild the search index (hash-cached: unchanged content is skipped)",
		Tags:    []string{"admin"},
	}, func(ctx context.Context, _ *struct{}) (*reindexOutput, error) {
		indexed, skipped, err := deps.Indexer.Reindex(ctx)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		out := &reindexOutput{}
		out.Body.Indexed = indexed
		out.Body.Skipped = skipped
		return out, nil
	})
}
