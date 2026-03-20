-- Initial migration: enable extensions on the default postgres database
--
-- NOTE: Bootstrap databases (authelia, n8n, vikunja, open_webui) are created
--       by init-databases.sh (Docker entrypoint, runs once on first container start).
--       Databases added in later phases use migrations with dblink_exec, which
--       executes CREATE DATABASE in a separate connection outside the current
--       transaction — this is why dblink is enabled here.

-- Enable dblink for cross-database queries if ever needed
CREATE EXTENSION IF NOT EXISTS dblink;

-- Enable pg_stat_statements for query performance monitoring
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
