-- Manual migration: hybrid-search infrastructure ent's DSL can't express.
-- These objects are excluded from Atlas diffs in atlas.hcl.

-- Lexical half: generated tsvector + GIN. RRF consumes ranks, so native
-- ts_rank is sufficient at this scale (upgrade path: pg_search/pg_textsearch).
ALTER TABLE "chunks" ADD COLUMN "fts" tsvector
  GENERATED ALWAYS AS (to_tsvector('english', "content")) STORED;
CREATE INDEX "chunk_fts" ON "chunks" USING GIN ("fts");

-- Semantic half: HNSW cosine ANN over the halfvec embeddings.
CREATE INDEX "chunk_embedding_hnsw" ON "chunks"
  USING hnsw ("embedding" halfvec_cosine_ops);

-- Tag containment filters (tags is jsonb: ["health", "errand"]).
CREATE INDEX "chunk_tags" ON "chunks" USING GIN ("tags" jsonb_path_ops);
