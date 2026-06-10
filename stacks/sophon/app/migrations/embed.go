// Package migrations embeds the versioned SQL applied at startup.
// Files are Atlas-generated from the ent schema (the Alembic flow), plus
// manual ones for DDL ent can't express (extension, fts column, HNSW index).
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
