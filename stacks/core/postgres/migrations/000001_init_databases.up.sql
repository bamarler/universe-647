-- Initial migration: enable extensions on the default postgres database
-- NOTE: Database creation is handled by init-databases.sh (Docker entrypoint)
--       because CREATE DATABASE cannot run inside a transaction.
--       golang-migrate is used for schema changes within databases.

-- Enable dblink for cross-database queries if ever needed
CREATE EXTENSION IF NOT EXISTS dblink;

-- Enable pg_stat_statements for query performance monitoring
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
