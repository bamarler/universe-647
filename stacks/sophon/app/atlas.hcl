# Atlas versioned-migration config (the Alembic flow):
# change internal/ent/schema → `atlas migrate diff <name> --env local`
# → reviewable .sql lands in migrations/ (golang-migrate format, applied at
# app startup). Dev database runs the same pgvector image as production so
# halfvec columns normalize correctly.
# The dev database must have the vector extension pre-installed (halfvec
# columns can't normalize without it). Start it with: ./dev/devdb.sh up
env "local" {
  src = "ent://internal/ent/schema"
  dev = "postgres://postgres:dev@localhost:5499/dev?sslmode=disable&search_path=public"
  migration {
    dir    = "file://migrations"
    format = golang-migrate
    # Manual search infrastructure (20260610200700) is invisible to the ent
    # schema; exclude it so diffs don't try to drop it.
    exclude = [
      "chunks.fts",
      "chunks.chunk_fts",
      "chunks.chunk_embedding_hnsw",
      "chunks.chunk_tags",
    ]
  }
}
