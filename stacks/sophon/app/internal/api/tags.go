package api

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/bamarler/universe-647/sophon/internal/ent"
	"github.com/bamarler/universe-647/sophon/internal/ent/tag"
	"github.com/bamarler/universe-647/sophon/internal/errs"
)

type createTagInput struct {
	Body struct {
		Name        string `json:"name" minLength:"1" maxLength:"120"`
		Kind        string `json:"kind" enum:"project,context,tag" default:"tag"`
		Description string `json:"description,omitempty"`
		ParentID    *int   `json:"parent_id,omitempty"`
	}
}

type tagOutput struct {
	Body TagDTO
}

type listTagsInput struct {
	Kind            string `query:"kind" enum:"project,context,tag,," doc:"filter by kind"`
	IncludeArchived bool   `query:"include_archived"`
}

type listTagsOutput struct {
	Body struct {
		Tags []TagDTO `json:"tags"`
	}
}

type tagIDInput struct {
	ID int `path:"id"`
}

type updateTagInput struct {
	ID   int `path:"id"`
	Body struct {
		Name        *string `json:"name,omitempty" maxLength:"120"`
		Description *string `json:"description,omitempty"`
		ParentID    *int    `json:"parent_id,omitempty"`
		Archived    *bool   `json:"archived,omitempty"`
	}
}

func registerTags(api huma.API, deps Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "create-tag", Method: http.MethodPost, Path: "/api/tags",
		Summary: "Create a tag (project/context/plain)", Tags: []string{"tags"},
	}, func(ctx context.Context, in *createTagInput) (*tagOutput, error) {
		create := deps.Ent.Tag.Create().
			SetName(in.Body.Name).
			SetKind(tag.Kind(in.Body.Kind))
		if in.Body.Description != "" {
			create.SetDescription(in.Body.Description)
		}
		if in.Body.ParentID != nil {
			create.SetParentID(*in.Body.ParentID)
		}
		t, err := create.Save(ctx)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		deps.Indexer.EnqueueTag(t.ID)
		return &tagOutput{Body: tagDTO(t)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-tags", Method: http.MethodGet, Path: "/api/tags",
		Summary: "List tags", Tags: []string{"tags"},
	}, func(ctx context.Context, in *listTagsInput) (*listTagsOutput, error) {
		q := deps.Ent.Tag.Query().WithParent().Order(ent.Asc(tag.FieldName))
		if in.Kind != "" {
			q = q.Where(tag.KindEQ(tag.Kind(in.Kind)))
		}
		if !in.IncludeArchived {
			q = q.Where(tag.ArchivedAtIsNil())
		}
		tags, err := q.All(ctx)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		out := &listTagsOutput{}
		out.Body.Tags = make([]TagDTO, len(tags))
		for i, t := range tags {
			out.Body.Tags[i] = tagDTO(t)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-tag", Method: http.MethodPatch, Path: "/api/tags/{id}",
		Summary: "Update a tag", Tags: []string{"tags"},
	}, func(ctx context.Context, in *updateTagInput) (*tagOutput, error) {
		upd := deps.Ent.Tag.UpdateOneID(in.ID)
		if in.Body.Name != nil {
			upd.SetName(*in.Body.Name)
		}
		if in.Body.Description != nil {
			upd.SetDescription(*in.Body.Description)
		}
		if in.Body.ParentID != nil {
			upd.SetParentID(*in.Body.ParentID)
		}
		if in.Body.Archived != nil {
			if *in.Body.Archived {
				upd.SetArchivedAt(time.Now())
			} else {
				upd.ClearArchivedAt()
			}
		}
		t, err := upd.Save(ctx)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		deps.Indexer.EnqueueTag(t.ID)
		return &tagOutput{Body: tagDTO(t)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-tag", Method: http.MethodDelete, Path: "/api/tags/{id}",
		Summary: "Delete a tag", Tags: []string{"tags"},
	}, func(ctx context.Context, in *tagIDInput) (*struct{}, error) {
		if err := deps.Ent.Tag.DeleteOneID(in.ID).Exec(ctx); err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}
		_ = deps.Indexer.Remove(ctx, "tag", in.ID)
		return &struct{}{}, nil
	})
}
