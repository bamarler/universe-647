-- Phase 5: Create nextcloud database
-- Uses dblink_exec because CREATE DATABASE cannot run inside a transaction.
-- dblink is enabled in migration 000001 for exactly this purpose.
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_database WHERE datname = 'nextcloud') THEN
    PERFORM dblink_exec('dbname=postgres', 'CREATE DATABASE nextcloud');
  END IF;
END $$;

GRANT ALL PRIVILEGES ON DATABASE nextcloud TO postgres;
