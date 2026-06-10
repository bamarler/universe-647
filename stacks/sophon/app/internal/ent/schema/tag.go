package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Tag is the supertag: a typed label that doubles as the organizational tree.
// kind=project and kind=context tags act as folders in the UI; plain tags are
// flat labels. A future CRM phase adds kind=person without schema surgery.
type Tag struct {
	ent.Schema
}

func (Tag) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			MaxLen(120),
		field.Enum("kind").
			Values("project", "context", "tag").
			Default("tag"),
		// Description is the only tag text that gets embedded for semantic
		// search; tag names themselves stay exact-match symbols.
		field.Text("description").
			Optional(),
		field.Time("archived_at").
			Optional().
			Nillable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Tag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("children", Tag.Type).
			From("parent").
			Unique(),
		edge.From("tasks", Task.Type).
			Ref("tags"),
		edge.From("notes", Note.Type).
			Ref("tags"),
		edge.From("project_tasks", Task.Type).
			Ref("project"),
	}
}

func (Tag) Indexes() []ent.Index {
	return []ent.Index{
		// One name per kind: #uni647 the project and #uni647 the plain tag can
		// coexist, but not two projects with the same name.
		index.Fields("name", "kind").Unique(),
		index.Fields("kind"),
	}
}
