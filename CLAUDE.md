# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Self-hosted AI-powered personal OS on an HP EliteDesk 800 G6 Mini (i7-10700T, 16GB RAM, dual NVMe).
29 containers across 6 active stacks (core, data, sophon, tomoko, storage, red-coast).
The `voice` and `smarthome` stacks are scaffolded (`.gitkeep` only) but not yet deployed.
The second brain is **Sophon** (Go backend) + **Tomoko** (Svelte PWA frontend), with the
**Bifrost** gateway routing LLM calls. (The legacy Open WebUI + mcpo layer has been retired.)

## Shell Rules

- **NEVER use `cd` commands.** Zoxide aliases `cd` and breaks in Claude Code's shell.
  Use absolute paths or `builtin cd` if directory change is truly needed.
- Shell: bash (Ubuntu Server 24.04)
- Package manager: apt (system), uv (Python), bun (JavaScript/TypeScript)
- **NEVER use `npm` or `npx`.** Always use `bun` / `bunx` instead.
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
make db-migrate                 # Run pending database migrations
make db-migrate-create NAME=x   # Create a new migration pair

# Validation
docker compose -f stacks/<name>/compose.yaml config   # Validate compose syntax
yamllint stacks/<name>/compose.yaml                    # Lint YAML
shellcheck scripts/<name>.sh                           # Lint bash scripts

# Backup & Security
make backup                     # Full backup: local restic + Backblaze B2
make backup-verify              # Test restore to /tmp
make trivy-scan                 # Scan running images for vulnerabilities
make revoke-device DEVICE=nodekey:abc123  # Emergency device revocation
```

## Key Architecture Context

Read `agent-docs/` BEFORE starting any task. These are gitignored AI context docs:
- `agent-docs/architecture.md` — Full stack overview, security model, AI pipeline design
- `agent-docs/networking.md` — Container communication map (what CAN and CANNOT talk)
- `agent-docs/phases.md` — Deployment phases with per-phase container details and RAM budget
- `agent-docs/containers.md` — Quick reference for every container: RAM, NVMe, networks, notes
- `agent-docs/monorepo.md` — What's tracked vs gitignored, directory structure, data locations

## Architecture Overview

**Traffic flow**: iPhone/Laptop → Tailscale (WireGuard) → Caddy (reverse proxy) → Authelia (SSO + WebAuthn 2FA) → backend service. No ports are exposed to the public internet.

**AI pipeline**: Sophon (Go backend) → Bifrost gateway (aliases: "tool-caller" → Gemini 3.1 Flash-Lite, "smart" → Claude Haiku 4.5; Ollama dormant until 32GB upgrade; virtual-key inbound auth). Tomoko (Svelte PWA) is the user-facing frontend, same-origin with Sophon's API behind Caddy.

**Automation**: n8n reaches services through Caddy's authenticated API endpoints (never direct container access). Handles morning briefings, email monitoring, contact sync, task extraction.

**Database**: PostgreSQL is shared by Authelia, n8n, Nextcloud, and Sophon.

**Security**: Three independent layers — (1) Tailscale + Device Approval (Tailnet Lock OFF — see docs/REMOTE-ACCESS.md), (2) Authelia + mandatory WebAuthn FIDO2, (3) Application-level authorization. CrowdSec for intrusion detection. Docker Socket Proxy (Tecnativa) — no container ever mounts `/var/run/docker.sock` directly.

**Port binding rule**: Only Caddy (`0.0.0.0:443` + `:80`), AdGuard Home (`0.0.0.0:53`), and Tailscale (`network_mode: host`) bind to the host. All other containers have no `ports:` directive. ⚠️ **Known deviation**: the intended posture was Caddy bound to `127.0.0.1:443` (reachable only via Tailscale's host interface), but it currently binds `0.0.0.0` — to revisit (restrict to `127.0.0.1` or formally accept).

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
| app_net | yes | Caddy ↔ app backends (Baïkal, Ntfy, Homepage, Nextcloud, Sophon, Tomoko, red-coast UIs) |
| ai_net | no | LLM stack — Sophon, Bifrost, dormant Ollama (needs outbound for Gemini/Anthropic APIs) |
| automation_net | no | n8n (needs outbound for Gmail, Calendar, Canvas APIs) |
| media_net | no | red-coast outbound (torrent trackers + TMDb metadata) |
| authelia_redis_net | yes | Authelia ↔ Redis session store |
| monitoring_net | no | Uptime Kuma, Glances, AdGuard, CrowdSec, NUT, Docker Socket Proxy |
| iot_net | yes | MQTT + Zigbee — defined for the planned `smarthome` stack (not yet deployed) |

Networks with `internal: true` block all outbound internet. Containers on those networks cannot exfiltrate data.

The `storage` stack defines one more internal sidecar, `nextcloud_redis_net` (Nextcloud ↔ Redis) — **11 networks total**.

## Scripts & Secrets

- Scripts: bash with `set -euo pipefail`, colored output matching Makefile conventions
- Secrets: `.env` files per stack, encrypted with SOPS+age as `.env.enc`
- Age public key in `.sops.yaml`, private key in `~/.config/sops/age/keys.txt` (from 1Password)
- Always `make encrypt` before committing, `make decrypt` after cloning
- Config files tracked in git, runtime data directories (`/mnt/data/`, `/srv/u647/`) gitignored

## Security-Critical Version Pins

- **n8n**: pin to v1.121.0+ (CVE-2026-21858, CVSS 10.0 — unauthenticated RCE)
- **gluetun** (red-coast VPN): pin by image digest — it ships no semver releases, only `latest`/PR tags
- All other images: pin by exact version tag (never `:latest`)

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
