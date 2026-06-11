package api

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/bamarler/universe-647/sophon/internal/ent"
	"github.com/bamarler/universe-647/sophon/internal/ent/task"
	"github.com/bamarler/universe-647/sophon/internal/errs"
)

type viewInput struct {
	View string `path:"view" enum:"today,upcoming,inbox"`
}

type taskListOutput struct {
	Body struct {
		Tasks []TaskDTO `json:"tasks"`
	}
}

// registerViews serves the smart views — saved-filter semantics over the GTD
// date model. Deferred tasks (defer_at in the future) are hidden everywhere:
// that's the tickler working.
func registerViews(api huma.API, deps Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "get-view", Method: http.MethodGet, Path: "/api/views/{view}",
		Summary: "Smart views: today / upcoming / inbox", Tags: []string{"browse"},
	}, func(ctx context.Context, in *viewInput) (*taskListOutput, error) {
		now := time.Now()
		notDeferred := task.Or(task.DeferAtIsNil(), task.DeferAtLTE(now))
		q := deps.Ent.Task.Query().
			Where(task.StatusEQ(task.StatusOpen), notDeferred).
			WithProject().WithTags()

		switch in.View {
		case "today":
			endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
			q = q.Where(task.DueAtNotNil(), task.DueAtLTE(endOfDay))
		case "upcoming":
			endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
			q = q.Where(task.DueAtGT(endOfDay), task.DueAtLTE(now.AddDate(0, 0, 14)))
		case "inbox":
			q = q.Where(task.Not(task.HasProject()))
		}

		tasks, err := q.
			Order(ent.Asc(task.FieldDueAt), ent.Desc(task.FieldPriority), ent.Asc(task.FieldCreatedAt)).
			All(ctx)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		out := &taskListOutput{}
		out.Body.Tasks = make([]TaskDTO, len(tasks))
		for i, t := range tasks {
			out.Body.Tasks[i] = taskDTO(t)
		}
		return out, nil
	})
}
