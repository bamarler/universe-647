# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Self-hosted AI-powered personal OS on an HP EliteDesk 800 G6 Mini (i7-10700T, 16GB RAM, dual NVMe).
25 containers across 6 stacks (core, data, sophon, storage, voice, smarthome) deployed in 7 phases.
The unified AI agent is **Sophon** — Open WebUI connected to all services via Docker MCP Gateway.

## Shell Rules

- **NEVER use `cd` commands.** Zoxide aliases `cd` and breaks in Claude Code's shell.
  Use absolute paths or `builtin cd` if directory change is truly needed.
- Shell: bash (Ubuntu Server 24.04)
- Package manager: apt (system), uv (Python)
- Container runtime: Docker with Compose v2 (`docker compose`, not `docker-compose`)

## Common Commands

All management goes through the Makefile. Run `make help` for the full list.

```bash
# Secrets (SOPS + age)
make decrypt                    # Decrypt .env.enc → .env (required before starting stacks)
make encrypt                    # Encrypt .env → .env.enc (required before git commit)

# Stack lifecycle (omit STACK= to target all in dependency order)
make up STACK=core              # Start a single stack
make down STACK=core            # Stop a single stack
make restart STACK=data         # Restart a stack
make status                     # Show all container states
make logs STACK=sophon          # Tail logs for a stack
make pull                       # Pull latest images for all stacks

# Database
make db-dump                    # Dump PostgreSQL to /tmp/pg_dump_<date>.sql.gz
make db-shell                   # Open psql interactive shell

# Validation
docker compose -f stacks/<name>/compose.yaml config   # Validate compose syntax
yamllint stacks/<name>/compose.yaml                    # Lint YAML
shellcheck scripts/<name>.sh                           # Lint bash scripts

# Backup & Security
make backup                     # Full backup: local restic + Cloudflare R2
make backup-verify              # Test restore to /tmp
make trivy-scan                 # Scan running images for vulnerabilities
make revoke-device DEVICE=nodekey:abc123  # Emergency device revocation
```

## Key Architecture Context

Read `agent-docs/` BEFORE starting any task. These are gitignored AI context docs:
- `agent-docs/architecture.md` — Full stack overview, security model, MCP Gateway design
- `agent-docs/networking.md` — Container communication map (what CAN and CANNOT talk)
- `agent-docs/phases.md` — Deployment phases with per-phase container details and RAM budget
- `agent-docs/containers.md` — Quick reference for every container: RAM, NVMe, networks, notes
- `agent-docs/monorepo.md` — What's tracked vs gitignored, directory structure, data locations

## Architecture Overview

**Traffic flow**: iPhone/Laptop → Tailscale (WireGuard) → Caddy (reverse proxy) → Authelia (SSO + WebAuthn 2FA) → backend service. No ports are exposed to the public internet.

**AI pipeline**: Open WebUI → LiteLLM (routes "local" to Ollama 7B, "smart" to Claude API) + MCP Gateway (routes tool calls to Monica, Vikunja, Calendar, PostgreSQL, Home Assistant, Nextcloud).

**Automation**: n8n reaches services through Caddy's authenticated API endpoints (never direct container access). Handles morning briefings, email monitoring, contact sync, task extraction.

**Database**: PostgreSQL is shared by Authelia, Open WebUI, n8n, Vikunja, Nextcloud. Monica uses its own MariaDB sidecar.

**Security**: Three independent layers — (1) Tailscale + Tailnet Lock + Device Approval, (2) Authelia + mandatory WebAuthn FIDO2, (3) Application-level authorization + MCP tool allowlists. CrowdSec for intrusion detection. Docker Socket Proxy (Tecnativa) — no container ever mounts `/var/run/docker.sock` directly.

**Port binding rule**: Only Caddy (`127.0.0.1:443`), AdGuard Home (`0.0.0.0:53`), and Tailscale (`network_mode: host`) bind to the host. All other containers have no `ports:` directive.

## Docker Compose Conventions

- Files named `compose.yaml` (not docker-compose.yml)
- All services use explicit `container_name:`
- Pin ALL image versions by tag (never `:latest`)
- Every container gets: `security_opt: [no-new-privileges:true]`, `restart: unless-stopped`
- Every service gets `mem_limit` and `cpus` resource constraints
- Use `read_only: true` where possible, with `tmpfs` mounts for `/tmp` and `/run`
- Networks: defined in `stacks/core/compose.yaml`, referenced as `external: true` in other stacks
- Each container connects ONLY to the networks it needs (see network table below)

## Docker Networks (defined in stacks/core/compose.yaml)

| Network | Internal | Purpose |
|---------|----------|---------|
| proxy_net | no | Caddy + Tailscale ingress |
| auth_net | yes | Caddy ↔ Authelia |
| db_net | yes | PostgreSQL connections |
| app_net | yes | Caddy ↔ app backends (Monica, Baïkal, Vikunja, Ntfy, Homepage, Nextcloud, HA) |
| ai_net | no | LLM + MCP stack (needs outbound for cloud APIs and model downloads) |
| automation_net | no | n8n (needs outbound for Gmail, Calendar, Canvas APIs) |
| iot_net | yes | MQTT + Zigbee (fully isolated — no other container can reach Mosquitto) |
| monitoring_net | no | Uptime Kuma, AdGuard, CrowdSec, NUT, Docker Socket Proxy |

Networks with `internal: true` block all outbound internet. Containers on those networks cannot exfiltrate data.

## Scripts & Secrets

- Scripts: bash with `set -euo pipefail`, colored output matching Makefile conventions
- Secrets: `.env` files per stack, encrypted with SOPS+age as `.env.enc`
- Age public key in `.sops.yaml`, private key in `~/.config/sops/age/keys.txt` (from 1Password)
- Always `make encrypt` before committing, `make decrypt` after cloning
- Config files tracked in git, runtime data directories (`/mnt/data/`, `/srv/u647/`) gitignored

## Security-Critical Version Pins

- **n8n**: pin to v1.121.0+ (CVE-2026-21858, CVSS 10.0 — unauthenticated RCE)
- **Open WebUI**: pin to v0.6.35+ (CVE-2025-64496 — code injection via malicious model servers)
- MCP servers: pin by image digest to prevent supply chain attacks

## Workflow

1. Read relevant `agent-docs/` files first
2. Make changes in one stack at a time
3. Validate compose syntax: `docker compose -f stacks/<name>/compose.yaml config`
4. Keep commits atomic — one logical change per commit
5. Run `make encrypt` before committing if any `.env` files changed

## Codebase Navigation

Use MCP codebase tools FIRST when exploring the repo or understanding how files relate.
Fall back to reading files directly only when MCP tools don't have what you need.
For config files (YAML, .env, Caddyfile), read directly — MCP tools are best for code.
