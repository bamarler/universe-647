DROP INDEX IF EXISTS "chunk_tags";
DROP INDEX IF EXISTS "chunk_embedding_hnsw";
DROP INDEX IF EXISTS "chunk_fts";
ALTER TABLE "chunks" DROP COLUMN IF EXISTS "fts";
