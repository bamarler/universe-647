package api

import (
	"time"

	"github.com/bamarler/universe-647/sophon/internal/ent"
)

type TagRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TagDTO struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Kind        string     `json:"kind" enum:"project,context,tag"`
	Description string     `json:"description,omitempty"`
	ParentID    *int       `json:"parent_id,omitempty"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
}

type TaskDTO struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	BodyMD      string     `json:"body_md,omitempty"`
	Status      string     `json:"status" enum:"open,done"`
	Priority    int        `json:"priority"`
	DueAt       *time.Time `json:"due_at,omitempty"`
	DeferAt     *time.Time `json:"defer_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	ProjectID   *int       `json:"project_id,omitempty"`
	ProjectName string     `json:"project_name,omitempty"`
	Tags        []TagRef   `json:"tags"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type NoteDTO struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	BodyMD    string    `json:"body_md,omitempty"`
	Tags      []TagRef  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func tagDTO(t *ent.Tag) TagDTO {
	d := TagDTO{
		ID:          t.ID,
		Name:        t.Name,
		Kind:        string(t.Kind),
		Description: t.Description,
		ArchivedAt:  t.ArchivedAt,
	}
	if p := t.Edges.Parent; p != nil {
		d.ParentID = &p.ID
	}
	return d
}

func tagRefs(tags []*ent.Tag) []TagRef {
	out := make([]TagRef, len(tags))
	for i, t := range tags {
		out[i] = TagRef{ID: t.ID, Name: t.Name}
	}
	return out
}

// taskDTO expects the task loaded WithProject().WithTags().
func taskDTO(t *ent.Task) TaskDTO {
	d := TaskDTO{
		ID:          t.ID,
		Title:       t.Title,
		BodyMD:      t.BodyMd,
		Status:      string(t.Status),
		Priority:    t.Priority,
		DueAt:       t.DueAt,
		DeferAt:     t.DeferAt,
		CompletedAt: t.CompletedAt,
		Tags:        tagRefs(t.Edges.Tags),
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
	if p := t.Edges.Project; p != nil {
		d.ProjectID = &p.ID
		d.ProjectName = p.Name
	}
	return d
}

// noteDTO expects the note loaded WithTags().
func noteDTO(n *ent.Note) NoteDTO {
	return NoteDTO{
		ID:        n.ID,
		Title:     n.Title,
		BodyMD:    n.BodyMd,
		Tags:      tagRefs(n.Edges.Tags),
		CreatedAt: n.CreatedAt,
		UpdatedAt: n.UpdatedAt,
	}
}
