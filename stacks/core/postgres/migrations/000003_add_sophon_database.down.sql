-- Rollback: drop sophon database
-- Uses dblink_exec for the same reason as the up migration.
REVOKE ALL PRIVILEGES ON DATABASE sophon FROM postgres;

DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_database WHERE datname = 'sophon') THEN
    PERFORM dblink_exec('dbname=postgres', 'DROP DATABASE sophon');
  END IF;
END $$;
