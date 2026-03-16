#!/bin/bash
set -euo pipefail

# Create databases for multi-service PostgreSQL
# Only runs on first startup when data volume is empty

create_database() {
	local db="$1"
	echo "Creating database: $db"
	psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<-EOSQL
		CREATE DATABASE "$db";
		GRANT ALL PRIVILEGES ON DATABASE "$db" TO "$POSTGRES_USER";
	EOSQL
}

create_database authelia
create_database n8n
create_database vikunja
create_database open_webui

echo "All databases created successfully"
