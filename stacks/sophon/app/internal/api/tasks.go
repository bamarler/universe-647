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

type createTaskInput struct {
	Body struct {
		Title     string     `json:"title" minLength:"1" maxLength:"500"`
		BodyMD    string     `json:"body_md,omitempty"`
		ProjectID *int       `json:"project_id,omitempty"`
		TagIDs    []int      `json:"tag_ids,omitempty"`
		DueAt     *time.Time `json:"due_at,omitempty"`
		DeferAt   *time.Time `json:"defer_at,omitempty"`
		Priority  int        `json:"priority,omitempty" minimum:"0" maximum:"3"`
	}
}

type taskOutput struct {
	Body TaskDTO
}

type taskIDInput struct {
	ID int `path:"id"`
}

type updateTaskInput struct {
	ID   int `path:"id"`
	Body struct {
		Title        *string    `json:"title,omitempty" maxLength:"500"`
		BodyMD       *string    `json:"body_md,omitempty"`
		Status       *string    `json:"status,omitempty" enum:"open,done"`
		Priority     *int       `json:"priority,omitempty" minimum:"0" maximum:"3"`
		DueAt        *time.Time `json:"due_at,omitempty"`
		DeferAt      *time.Time `json:"defer_at,omitempty"`
		ProjectID    *int       `json:"project_id,omitempty"`
		TagIDs       []int      `json:"tag_ids,omitempty" doc:"full replacement set when present"`
		ClearDueAt   bool       `json:"clear_due_at,omitempty"`
		ClearDeferAt bool       `json:"clear_defer_at,omitempty"`
		ClearProject bool       `json:"clear_project,omitempty"`
	}
}

func loadTask(ctx context.Context, db *ent.Client, id int) (*ent.Task, error) {
	return db.Task.Query().Where(task.IDEQ(id)).WithProject().WithTags().Only(ctx)
}

func registerTasks(api huma.API, deps Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "create-task", Method: http.MethodPost, Path: "/api/tasks",
		Summary: "Create a task", Tags: []string{"tasks"},
	}, func(ctx context.Context, in *createTaskInput) (*taskOutput, error) {
		create := deps.Ent.Task.Create().
			SetTitle(in.Body.Title).
			SetBodyMd(in.Body.BodyMD).
			SetPriority(in.Body.Priority)
		if in.Body.ProjectID != nil {
			create.SetProjectID(*in.Body.ProjectID)
		}
		if len(in.Body.TagIDs) > 0 {
			create.AddTagIDs(in.Body.TagIDs...)
		}
		if in.Body.DueAt != nil {
			create.SetDueAt(*in.Body.DueAt)
		}
		if in.Body.DeferAt != nil {
			create.SetDeferAt(*in.Body.DeferAt)
		}
		t, err := create.Save(ctx)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		deps.Indexer.EnqueueTask(t.ID)
		full, err := loadTask(ctx, deps.Ent, t.ID)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		return &taskOutput{Body: taskDTO(full)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-task", Method: http.MethodGet, Path: "/api/tasks/{id}",
		Summary: "Get a task", Tags: []string{"tasks"},
	}, func(ctx context.Context, in *taskIDInput) (*taskOutput, error) {
		t, err := loadTask(ctx, deps.Ent, in.ID)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		return &taskOutput{Body: taskDTO(t)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-task", Method: http.MethodPatch, Path: "/api/tasks/{id}",
		Summary: "Update a task (partial)", Tags: []string{"tasks"},
	}, func(ctx context.Context, in *updateTaskInput) (*taskOutput, error) {
		upd := deps.Ent.Task.UpdateOneID(in.ID)
		if in.Body.Title != nil {
			upd.SetTitle(*in.Body.Title)
		}
		if in.Body.BodyMD != nil {
			upd.SetBodyMd(*in.Body.BodyMD)
		}
		if in.Body.Priority != nil {
			upd.SetPriority(*in.Body.Priority)
		}
		if in.Body.Status != nil {
			upd.SetStatus(task.Status(*in.Body.Status))
			if *in.Body.Status == "done" {
				upd.SetCompletedAt(time.Now())
			} else {
				upd.ClearCompletedAt()
			}
		}
		if in.Body.DueAt != nil {
			upd.SetDueAt(*in.Body.DueAt)
		} else if in.Body.ClearDueAt {
			upd.ClearDueAt()
		}
		if in.Body.DeferAt != nil {
			upd.SetDeferAt(*in.Body.DeferAt)
		} else if in.Body.ClearDeferAt {
			upd.ClearDeferAt()
		}
		if in.Body.ProjectID != nil {
			upd.SetProjectID(*in.Body.ProjectID)
		} else if in.Body.ClearProject {
			upd.ClearProject()
		}
		if in.Body.TagIDs != nil {
			upd.ClearTags().AddTagIDs(in.Body.TagIDs...)
		}
		if _, err := upd.Save(ctx); err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		deps.Indexer.EnqueueTask(in.ID)
		t, err := loadTask(ctx, deps.Ent, in.ID)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		return &taskOutput{Body: taskDTO(t)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-task", Method: http.MethodDelete, Path: "/api/tasks/{id}",
		Summary: "Delete a task", Tags: []string{"tasks"},
	}, func(ctx context.Context, in *taskIDInput) (*struct{}, error) {
		if err := deps.Ent.Task.DeleteOneID(in.ID).Exec(ctx); err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		_ = deps.Indexer.Remove(ctx, "task", in.ID)
		return &struct{}{}, nil
	})
}
