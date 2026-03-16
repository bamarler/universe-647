#!/bin/bash
set -euo pipefail

# Universe 647 — mcpo entrypoint wrapper
# Substitutes environment variables in config template, then starts mcpo

CONFIG_TEMPLATE="/app/config.template.json"
CONFIG_OUTPUT="/tmp/config.json"

: "${POSTGRES_USER:?POSTGRES_USER is required}"
: "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}"
: "${DOMAIN:?DOMAIN is required}"
: "${MONICA_API_TOKEN:?MONICA_API_TOKEN is required}"
: "${VIKUNJA_API_TOKEN:?VIKUNJA_API_TOKEN is required}"

envsubst < "${CONFIG_TEMPLATE}" > "${CONFIG_OUTPUT}"

exec mcpo --config "${CONFIG_OUTPUT}" "$@"
