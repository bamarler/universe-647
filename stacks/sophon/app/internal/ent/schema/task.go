package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Task follows the settled GTD model: a primary project (folder placement),
// any number of additional tags, and two distinct date semantics — due_at is
// the deadline, defer_at is "don't show me this before X" (the tickler).
type Task struct {
	ent.Schema
}

func (Task) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			NotEmpty().
			MaxLen(500),
		field.Text("body_md").
			Optional(),
		field.Enum("status").
			Values("open", "done").
			Default("open"),
		field.Int("priority").
			Default(0).
			Min(0).
			Max(3),
		field.Time("due_at").
			Optional().
			Nillable(),
		field.Time("defer_at").
			Optional().
			Nillable(),
		field.Time("completed_at").
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

func (Task) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("project", Tag.Type).
			Unique(),
		edge.To("tags", Tag.Type),
	}
}

func (Task) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("due_at"),
		index.Fields("defer_at"),
	}
}
