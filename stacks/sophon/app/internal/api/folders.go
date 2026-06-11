package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/bamarler/universe-647/sophon/internal/ent"
	"github.com/bamarler/universe-647/sophon/internal/ent/note"
	"github.com/bamarler/universe-647/sophon/internal/ent/tag"
	"github.com/bamarler/universe-647/sophon/internal/ent/task"
	"github.com/bamarler/universe-647/sophon/internal/errs"
)

type folderInput struct {
	ID          int  `path:"id"`
	IncludeDone bool `query:"include_done"`
}

type folderOutput struct {
	Body struct {
		Tag      TagDTO    `json:"tag"`
		Children []TagDTO  `json:"children"`
		Tasks    []TaskDTO `json:"tasks"`
		Notes    []NoteDTO `json:"notes"`
	}
}

// registerFolders serves one drill-in view: a project/context tag with its
// child folders, tasks, and notes. Projects own tasks via the project edge;
// contexts (and plain tags) collect them via the m2m tag edge.
func registerFolders(api huma.API, deps Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "get-folder", Method: http.MethodGet, Path: "/api/folders/{id}",
		Summary: "Folder contents: child folders, tasks, notes", Tags: []string{"browse"},
	}, func(ctx context.Context, in *folderInput) (*folderOutput, error) {
		t, err := deps.Ent.Tag.Query().Where(tag.IDEQ(in.ID)).WithParent().Only(ctx)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}

		children, err := deps.Ent.Tag.Query().
			Where(tag.HasParentWith(tag.IDEQ(in.ID)), tag.ArchivedAtIsNil()).
			Order(ent.Asc(tag.FieldName)).
			All(ctx)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}

		membership := task.Or(
			task.HasProjectWith(tag.IDEQ(in.ID)),
			task.HasTagsWith(tag.IDEQ(in.ID)),
		)
		tq := deps.Ent.Task.Query().Where(membership).WithProject().WithTags()
		if !in.IncludeDone {
			tq = tq.Where(task.StatusEQ(task.StatusOpen))
		}
		tasks, err := tq.
			Order(ent.Asc(task.FieldDueAt), ent.Desc(task.FieldPriority), ent.Asc(task.FieldCreatedAt)).
			All(ctx)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}

		notes, err := deps.Ent.Note.Query().
			Where(note.HasTagsWith(tag.IDEQ(in.ID))).
			WithTags().
			Order(ent.Desc(note.FieldUpdatedAt)).
			All(ctx)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}

		out := &folderOutput{}
		out.Body.Tag = tagDTO(t)
		out.Body.Children = make([]TagDTO, len(children))
		for i, c := range children {
			out.Body.Children[i] = tagDTO(c)
		}
		out.Body.Tasks = make([]TaskDTO, len(tasks))
		for i, tk := range tasks {
			out.Body.Tasks[i] = taskDTO(tk)
		}
		out.Body.Notes = make([]NoteDTO, len(notes))
		for i, n := range notes {
			out.Body.Notes[i] = noteDTO(n)
		}
		return out, nil
	})
}
