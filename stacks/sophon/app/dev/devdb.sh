#!/usr/bin/env bash
# Throwaway Postgres for Atlas migration diffs and integration tests.
# The vector extension is installed into template1 so every database Atlas
# creates (and the dev db itself) inherits it.
set -euo pipefail

NAME=sophon-dev-pg
IMAGE=pgvector/pgvector:0.8.2-pg16
PORT=5499

case "${1:-}" in
  up)
    if docker ps --format '{{.Names}}' | grep -q "^${NAME}$"; then
      echo "${NAME} already running"
      exit 0
    fi
    docker run --rm -d --name "$NAME" \
      -e POSTGRES_PASSWORD=dev \
      -e POSTGRES_DB=dev \
      -p "127.0.0.1:${PORT}:5432" \
      "$IMAGE" >/dev/null
    until docker exec "$NAME" pg_isready -U postgres -q 2>/dev/null; do
      sleep 0.5
    done
    docker exec "$NAME" psql -U postgres -d template1 -q \
      -c "CREATE EXTENSION IF NOT EXISTS vector;"
    docker exec "$NAME" psql -U postgres -d dev -q \
      -c "CREATE EXTENSION IF NOT EXISTS vector;"
    echo "${NAME} ready on 127.0.0.1:${PORT}"
    ;;
  down)
    docker stop "$NAME" >/dev/null 2>&1 || true
    echo "${NAME} stopped"
    ;;
  *)
    echo "usage: $0 up|down" >&2
    exit 1
    ;;
esac
