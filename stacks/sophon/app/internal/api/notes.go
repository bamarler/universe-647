package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/bamarler/universe-647/sophon/internal/ent"
	"github.com/bamarler/universe-647/sophon/internal/ent/note"
	"github.com/bamarler/universe-647/sophon/internal/errs"
)

type createNoteInput struct {
	Body struct {
		Title  string `json:"title" minLength:"1" maxLength:"500"`
		BodyMD string `json:"body_md,omitempty"`
		TagIDs []int  `json:"tag_ids,omitempty"`
	}
}

type noteOutput struct {
	Body NoteDTO
}

type noteIDInput struct {
	ID int `path:"id"`
}

type updateNoteInput struct {
	ID   int `path:"id"`
	Body struct {
		Title  *string `json:"title,omitempty" maxLength:"500"`
		BodyMD *string `json:"body_md,omitempty"`
		TagIDs []int   `json:"tag_ids,omitempty" doc:"full replacement set when present"`
	}
}

func loadNote(ctx context.Context, db *ent.Client, id int) (*ent.Note, error) {
	return db.Note.Query().Where(note.IDEQ(id)).WithTags().Only(ctx)
}

func registerNotes(api huma.API, deps Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "create-note", Method: http.MethodPost, Path: "/api/notes",
		Summary: "Create a note", Tags: []string{"notes"},
	}, func(ctx context.Context, in *createNoteInput) (*noteOutput, error) {
		create := deps.Ent.Note.Create().
			SetTitle(in.Body.Title).
			SetBodyMd(in.Body.BodyMD)
		if len(in.Body.TagIDs) > 0 {
			create.AddTagIDs(in.Body.TagIDs...)
		}
		n, err := create.Save(ctx)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		deps.Indexer.EnqueueNote(n.ID)
		full, err := loadNote(ctx, deps.Ent, n.ID)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		return &noteOutput{Body: noteDTO(full)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-note", Method: http.MethodGet, Path: "/api/notes/{id}",
		Summary: "Get a note", Tags: []string{"notes"},
	}, func(ctx context.Context, in *noteIDInput) (*noteOutput, error) {
		n, err := loadNote(ctx, deps.Ent, in.ID)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		return &noteOutput{Body: noteDTO(n)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-note", Method: http.MethodPatch, Path: "/api/notes/{id}",
		Summary: "Update a note (partial)", Tags: []string{"notes"},
	}, func(ctx context.Context, in *updateNoteInput) (*noteOutput, error) {
		upd := deps.Ent.Note.UpdateOneID(in.ID)
		if in.Body.Title != nil {
			upd.SetTitle(*in.Body.Title)
		}
		if in.Body.BodyMD != nil {
			upd.SetBodyMd(*in.Body.BodyMD)
		}
		if in.Body.TagIDs != nil {
			upd.ClearTags().AddTagIDs(in.Body.TagIDs...)
		}
		if _, err := upd.Save(ctx); err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		deps.Indexer.EnqueueNote(in.ID)
		n, err := loadNote(ctx, deps.Ent, in.ID)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		return &noteOutput{Body: noteDTO(n)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-note", Method: http.MethodDelete, Path: "/api/notes/{id}",
		Summary: "Delete a note", Tags: []string{"notes"},
	}, func(ctx context.Context, in *noteIDInput) (*struct{}, error) {
		if err := deps.Ent.Note.DeleteOneID(in.ID).Exec(ctx); err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		_ = deps.Indexer.Remove(ctx, "note", in.ID)
		return &struct{}{}, nil
	})
}
