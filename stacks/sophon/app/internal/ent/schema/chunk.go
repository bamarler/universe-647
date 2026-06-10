package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	pgvector "github.com/pgvector/pgvector-go"
)

// Chunk is the search index: one row per embedded unit of content.
// Tasks embed as a single enriched chunk; notes split by heading.
// The fts tsvector GENERATED column and its GIN index, plus the HNSW index on
// embedding, live in a manual migration — Postgres DDL ent can't express.
// The index is derived data: fully rebuildable from tasks/notes/tags.
type Chunk struct {
	ent.Schema
}

func (Chunk) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("source_type").
			Values("task", "note", "tag"),
		field.Int("source_id"),
		field.Int("chunk_index").
			Default(0),
		// sha256 of the rendered content; unchanged hashes skip re-embedding
		// (the Cursor incremental-indexing pattern).
		field.String("chunk_hash").
			MaxLen(64),
		field.Text("content"),
		field.Other("embedding", pgvector.HalfVector{}).
			SchemaType(map[string]string{
				dialect.Postgres: "halfvec(768)",
			}).
			Optional(),
		// Denormalized filter metadata so hybrid search never joins.
		field.String("project").
			Optional().
			MaxLen(120),
		field.JSON("tags", []string{}).
			Optional(),
		field.Time("item_due_at").
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

func (Chunk) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("source_type", "source_id", "chunk_index").Unique(),
		index.Fields("source_type", "item_due_at"),
		index.Fields("chunk_hash"),
	}
}
