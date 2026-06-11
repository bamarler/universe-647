package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/bamarler/universe-647/sophon/internal/ent/note"
	"github.com/bamarler/universe-647/sophon/internal/ent/tag"
	"github.com/bamarler/universe-647/sophon/internal/ent/task"
	"github.com/bamarler/universe-647/sophon/internal/errs"
	"github.com/bamarler/universe-647/sophon/internal/search"
)

type searchInput struct {
	Body struct {
		Query         string   `json:"query,omitempty"`
		Project       string   `json:"project,omitempty" doc:"project tag name"`
		Tags          []string `json:"tags,omitempty"`
		Types         []string `json:"types,omitempty" doc:"task|note|tag"`
		DueWithinDays *int     `json:"due_within_days,omitempty"`
	}
}

// SearchHit is a navigable link: type+id to route to, title to show. Never
// generated text — the snippet is the item's own indexed content.
type SearchHit struct {
	SourceType string  `json:"source_type" enum:"task,note,tag"`
	SourceID   int     `json:"source_id"`
	Title      string  `json:"title"`
	Snippet    string  `json:"snippet,omitempty"`
	Project    string  `json:"project,omitempty"`
	Score      float64 `json:"score"`
}

type searchOutput struct {
	Body struct {
		Hits []SearchHit `json:"hits"`
	}
}

func registerSearch(api huma.API, deps Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "search", Method: http.MethodPost, Path: "/api/search",
		Summary: "Hybrid search (lexical + semantic, RRF-fused)", Tags: []string{"search"},
	}, func(ctx context.Context, in *searchInput) (*searchOutput, error) {
		hits, err := runSearch(ctx, deps, search.Params{
			Query:     in.Body.Query,
			Project:   in.Body.Project,
			Tags:      in.Body.Tags,
			Types:     in.Body.Types,
			DueBefore: dueBefore(in.Body.DueWithinDays),
		})
		if err != nil {
			return nil, err
		}
		out := &searchOutput{}
		out.Body.Hits = hits
		return out, nil
	})
}

func dueBefore(days *int) *time.Time {
	if days == nil {
		return nil
	}
	t := time.Now().AddDate(0, 0, *days)
	return &t
}

// runSearch executes hybrid search and resolves hits to display titles.
func runSearch(ctx context.Context, deps Deps, p search.Params) ([]SearchHit, error) {
	results, err := deps.Searcher.Search(ctx, p)
	if err != nil {
		return nil, errs.HTTP(errs.FromDB(err))
	}

	// Resolve titles per type in three cheap IN queries.
	var taskIDs, noteIDs, tagIDs []int
	for _, r := range results {
		switch r.SourceType {
		case "task":
			taskIDs = append(taskIDs, r.SourceID)
		case "note":
			noteIDs = append(noteIDs, r.SourceID)
		case "tag":
			tagIDs = append(tagIDs, r.SourceID)
		}
	}
	titles := map[string]string{}
	if len(taskIDs) > 0 {
		ts, err := deps.Ent.Task.Query().Where(task.IDIn(taskIDs...)).All(ctx)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		for _, t := range ts {
			titles["task:"+itoa(t.ID)] = t.Title
		}
	}
	if len(noteIDs) > 0 {
		ns, err := deps.Ent.Note.Query().Where(note.IDIn(noteIDs...)).All(ctx)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		for _, n := range ns {
			titles["note:"+itoa(n.ID)] = n.Title
		}
	}
	if len(tagIDs) > 0 {
		tgs, err := deps.Ent.Tag.Query().Where(tag.IDIn(tagIDs...)).All(ctx)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		for _, t := range tgs {
			titles["tag:"+itoa(t.ID)] = t.Name
		}
	}

	hits := make([]SearchHit, 0, len(results))
	for _, r := range results {
		title, ok := titles[r.SourceType+":"+itoa(r.SourceID)]
		if !ok {
			continue // chunk outlived its item; reindex will clean it
		}
		hits = append(hits, SearchHit{
			SourceType: r.SourceType,
			SourceID:   r.SourceID,
			Title:      title,
			Snippet:    r.Snippet,
			Project:    r.Project,
			Score:      r.Score,
		})
	}
	return hits, nil
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
