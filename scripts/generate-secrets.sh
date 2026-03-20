#!/bin/bash
set -euo pipefail

# Generate .env files for all stacks with random secrets pre-filled.
# Dashboard tokens (Tailscale, Cloudflare, Gemini, Anthropic, etc.) are left
# as CHANGE_ME placeholders — fill those in manually before deploying.

REPO_ROOT="$(builtin cd "$(dirname "$0")/.." && pwd)"

gen32() { openssl rand -base64 32 | tr -d '\n'; }
gen64() { openssl rand -base64 64 | tr -d '\n'; }
gen16() { openssl rand -base64 16 | tr -d '\n'; }

# Shared values — POSTGRES_PASSWORD is generated once, used in all stacks
POSTGRES_PASSWORD="$(gen32)"

echo "Generating secrets for all stacks..."
echo ""

# ==============================================================================
# Core
# ==============================================================================
cat > "$REPO_ROOT/stacks/core/.env" <<EOF
# Universe 647 — Core Infrastructure
# Generated $(date +%Y-%m-%d) — dashboard tokens still need manual entry

DOMAIN=CHANGE_ME_YOUR_DOMAIN
TZ=America/New_York
POSTGRES_USER=postgres
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}

# Authelia SSO
AUTHELIA_JWT_SECRET=$(gen64)
AUTHELIA_SESSION_SECRET=$(gen64)
AUTHELIA_STORAGE_ENCRYPTION_KEY=$(gen64)

# Tailscale — https://login.tailscale.com/admin/settings/keys
TS_AUTHKEY=CHANGE_ME_TAILSCALE_AUTH_KEY
TS_HOSTNAME=universe-647

# Cloudflare — https://dash.cloudflare.com/profile/api-tokens
CLOUDFLARE_API_TOKEN=CHANGE_ME_CLOUDFLARE_API_TOKEN

# CrowdSec — generate AFTER first start:
#   docker exec crowdsec cscli bouncers add caddy-bouncer
CROWDSEC_API_KEY=CHANGE_ME_AFTER_FIRST_START

# NUT UPS
NUT_API_PASSWORD=$(gen16)
EOF
echo "  stacks/core/.env"

# ==============================================================================
# Data
# ==============================================================================
cat > "$REPO_ROOT/stacks/data/.env" <<EOF
# Universe 647 — Data Stack
# Generated $(date +%Y-%m-%d) — SMTP credentials need manual entry

DOMAIN=CHANGE_ME_YOUR_DOMAIN
TZ=America/New_York
POSTGRES_USER=postgres
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}

# Monica CRM
MONICA_APP_KEY=base64:$(gen32)
MONICA_DB_PASSWORD=$(gen32)
MONICA_DB_ROOT_PASSWORD=$(gen32)

# n8n
N8N_ENCRYPTION_KEY=$(gen32)
N8N_JWT_SECRET=$(gen64)

# SMTP (optional — for Monica email notifications)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=CHANGE_ME_EMAIL
SMTP_PASSWORD=CHANGE_ME_GMAIL_APP_PASSWORD
SMTP_FROM_ADDRESS=CHANGE_ME_EMAIL
EOF
echo "  stacks/data/.env"

# ==============================================================================
# Sophon
# ==============================================================================
cat > "$REPO_ROOT/stacks/sophon/.env" <<EOF
# Universe 647 — Sophon (AI Brain)
# Generated $(date +%Y-%m-%d) — API keys need manual entry

DOMAIN=CHANGE_ME_YOUR_DOMAIN
TZ=America/New_York
POSTGRES_USER=postgres
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}

# LiteLLM
LITELLM_MASTER_KEY=$(gen32)

# Gemini — https://aistudio.google.com/apikey
GEMINI_API_KEY=CHANGE_ME_GEMINI_API_KEY

# Anthropic — https://console.anthropic.com/settings/keys
ANTHROPIC_API_KEY=CHANGE_ME_ANTHROPIC_API_KEY

# Open WebUI
WEBUI_SECRET_KEY=$(gen32)

# mcpo proxy
MCPO_API_KEY=$(gen32)

# MCP server tokens — get from each app's UI after deploy
MONICA_API_TOKEN=CHANGE_ME_AFTER_DEPLOY
VIKUNJA_API_TOKEN=CHANGE_ME_AFTER_DEPLOY
EOF
echo "  stacks/sophon/.env"

# ==============================================================================
# Storage
# ==============================================================================
cat > "$REPO_ROOT/stacks/storage/.env" <<EOF
# Universe 647 — Storage (Nextcloud)
# Generated $(date +%Y-%m-%d)

DOMAIN=CHANGE_ME_YOUR_DOMAIN
POSTGRES_USER=postgres
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}

NEXTCLOUD_ADMIN_USER=admin
NEXTCLOUD_ADMIN_PASSWORD=$(gen32)
EOF
echo "  stacks/storage/.env"

echo ""
echo "Done. All openssl secrets generated."
echo ""
echo "REMAINING: search each .env for CHANGE_ME and fill in dashboard values."
echo "Then run: make encrypt"
