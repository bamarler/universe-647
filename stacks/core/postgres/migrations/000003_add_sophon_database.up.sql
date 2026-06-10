-- Second brain: create sophon database (Go backend — tasks, notes, tags, chunks)
-- Uses dblink_exec because CREATE DATABASE cannot run inside a transaction.
-- dblink is enabled in migration 000001 for exactly this purpose.
-- The vector extension and schema DDL are owned by the sophon app's own
-- migrations (stacks/sophon/app), not by this bootstrap migration.
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_database WHERE datname = 'sophon') THEN
    PERFORM dblink_exec('dbname=postgres', 'CREATE DATABASE sophon');
  END IF;
END $$;

GRANT ALL PRIVILEGES ON DATABASE sophon TO postgres;
