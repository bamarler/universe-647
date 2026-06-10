package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/bamarler/universe-647/sophon/internal/ent"
	"github.com/bamarler/universe-647/sophon/internal/ent/tag"
	"github.com/bamarler/universe-647/sophon/internal/errs"
)

// TreeNode is one folder in the browser: a project or context tag with its
// children and item counts. Plain tags are flat and excluded from the tree.
type TreeNode struct {
	ID        int         `json:"id"`
	Name      string      `json:"name"`
	Kind      string      `json:"kind" enum:"project,context"`
	TaskCount int         `json:"task_count"`
	NoteCount int         `json:"note_count"`
	Children  []*TreeNode `json:"children"`
}

type treeOutput struct {
	Body struct {
		Roots []*TreeNode `json:"roots"`
	}
}

func registerTree(api huma.API, deps Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "get-tree",
		Method:      http.MethodGet,
		Path:        "/api/tree",
		Summary:     "Project/context folder tree with item counts",
		Tags:        []string{"browse"},
	}, func(ctx context.Context, _ *struct{}) (*treeOutput, error) {
		tags, err := deps.Ent.Tag.Query().
			Where(
				tag.KindIn(tag.KindProject, tag.KindContext),
				tag.ArchivedAtIsNil(),
			).
			WithParent().
			Order(ent.Asc(tag.FieldName)).
			All(ctx)
		if err != nil {
			return nil, errs.HTTP(errs.FromDB(err))
		}

		nodes := make(map[int]*TreeNode, len(tags))
		for _, t := range tags {
			taskCount, err := t.QueryTasks().Count(ctx)
			if err != nil {
				return nil, errs.HTTP(errs.FromDB(err))
			}
			projCount, err := t.QueryProjectTasks().Count(ctx)
			if err != nil {
				return nil, errs.HTTP(errs.FromDB(err))
			}
			noteCount, err := t.QueryNotes().Count(ctx)
			if err != nil {
				return nil, errs.HTTP(errs.FromDB(err))
			}
			nodes[t.ID] = &TreeNode{
				ID:        t.ID,
				Name:      t.Name,
				Kind:      string(t.Kind),
				TaskCount: taskCount + projCount,
				NoteCount: noteCount,
				Children:  []*TreeNode{},
			}
		}

		out := &treeOutput{}
		out.Body.Roots = []*TreeNode{}
		for _, t := range tags {
			node := nodes[t.ID]
			if p := t.Edges.Parent; p != nil && nodes[p.ID] != nil {
				nodes[p.ID].Children = append(nodes[p.ID].Children, node)
			} else {
				out.Body.Roots = append(out.Body.Roots, node)
			}
		}
		return out, nil
	})
}
