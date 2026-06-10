-- Manual migration: pgvector extension (ent/Atlas can't own extensions).
-- Image pgvector/pgvector:0.8.2-pg16 ships the extension binaries.
CREATE EXTENSION IF NOT EXISTS vector;
