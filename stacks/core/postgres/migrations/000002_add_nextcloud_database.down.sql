-- Rollback: drop nextcloud database
-- Uses dblink_exec for the same reason as the up migration.
REVOKE ALL PRIVILEGES ON DATABASE nextcloud FROM postgres;

DO $$
BEGIN
  IF EXISTS (SELECT FROM pg_database WHERE datname = 'nextcloud') THEN
    PERFORM dblink_exec('dbname=postgres', 'DROP DATABASE nextcloud');
  END IF;
END $$;
