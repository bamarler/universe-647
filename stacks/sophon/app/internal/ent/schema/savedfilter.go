package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// SavedFilter is a persisted query — the only "views" feature
// (the OmniFocus-perspective / Todoist-filter pattern).
type SavedFilter struct {
	ent.Schema
}

func (SavedFilter) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			MaxLen(120).
			Unique(),
		field.JSON("query", map[string]any{}),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}
