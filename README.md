# Universe 647

An AI-powered personal operating system running on an HP EliteDesk 800 G6 Mini. A self-hosted second brain (notes, tasks, semantic search), workflow automation, file sync, and movie/TV streaming — fronted by a custom Svelte PWA and backed by cloud-LLM routing. Local LLM inference and smart-home control are planned.

*Named after the pocket universe in Liu Cixin's Remembrance of Earth's Past trilogy; a self-contained world preserving the memory of civilization, designed to outlast everything.*

## Architecture

<!-- TODO: Replace with Miro diagram -->
![Architecture Diagram](docs/architecture-diagram.png)

See [docs/NETWORK-OVERVIEW.md](docs/NETWORK-OVERVIEW.md) for a detailed breakdown of container communication paths and network zones.

## Prerequisites

Before cloning this repo, you need:

| Item | Where | Purpose |
|------|-------|---------|
| HP EliteDesk 800 G6 Mini | — | i7-10700T, 16GB+ RAM, 250GB + 500GB NVMe |
| UPS | Connected via USB | Graceful shutdown on power loss |
| 2x FIDO2 security keys (NFC) | Keychain + safe | WebAuthn 2FA for all services |
| Cloudflare account | cloudflare.com | DNS, HTTPS certificates |
| Backblaze B2 account | backblaze.com | Offsite encrypted restic backups (S3-compatible) |
| Tailscale account | tailscale.com | Mesh VPN, Device Approval (Tailnet Lock currently off) |
| 1Password (or equivalent) | — | Stores age private key, restic password, API keys |
| Google Cloud Podcast API | cloud.google.com | Optional: audio morning briefings |

## Quick Start

```bash
# 1. Hardware setup
#    BIOS → USB boot, disable Secure Boot, enable VT-x/VT-d
#    Install Ubuntu Server 24.04 on 250GB NVMe
#    Format 500GB NVMe as ext4, mount at /mnt/data

# 2. Clone and setup
git clone git@github.com:you/universe-647.git
cd universe-647
./scripts/setup.sh

# 3. Import age key from 1Password
#    Paste into ~/.config/sops/age/keys.txt

# 4. Initialize restic backup repos
restic -r /mnt/data/restic-repo init
restic -r s3:s3.us-west-000.backblazeb2.com/your-bucket init

# 5. Decrypt secrets and start core infrastructure
make decrypt
make up STACK=core

# 6. Register security keys in Authelia
#    Navigate to https://auth.home.yourdomain.com
#    Register primary FIDO2 key + iPhone passkey + backup key

# 7. Verify and run first backup
make status
make backup
```

## Deployment Phases

| Phase | Stack | Containers | What It Unlocks |
|:-----:|-------|:----------:|-----------------|
| 2 | `core` | 13 | Reverse proxy, VPN, SSO + 2FA, monitoring, DNS blocking, intrusion detection, UPS shutdown, backups |
| 3 | `data` | 3 | CalDAV/CardDAV (Baïkal), workflow automation (n8n), push notifications (Ntfy) |
| 4 | `sophon` | 3 | Second-brain backend (Sophon), cloud-LLM routing (Bifrost), dormant local LLM (Ollama) |
| 4 | `tomoko` | 1 | Svelte PWA frontend for the second brain |
| 5 | `storage` | 2 | Self-hosted files (Nextcloud + Redis), semantic search |
| 6 | `voice` | *(planned)* | Server-side speech-to-text and text-to-speech — not yet deployed |
| 7 | `smarthome` | *(planned)* | Zigbee device control, smart-home automations — not yet deployed |
| 8 | `red-coast` | 7 | Movie/TV streaming: Jellyfin + Radarr/Sonarr/Prowlarr + qBittorrent behind a VPN |

**29 containers across 6 active stacks (core, data, sophon, tomoko, storage, red-coast). `voice` + `smarthome` are scaffolded for later.**

Start each phase with `make up STACK=<name>` once the previous phase is stable.

## Makefile Commands

Run `make help` for the full list. Most-used commands:

| Command | Description |
|---------|-------------|
| `make up` | Start all stacks in dependency order |
| `make up STACK=core` | Start a single stack |
| `make down` | Stop everything (reverse order) |
| `make status` | Show all container states |
| `make logs STACK=sophon` | Tail logs for a stack |
| `make decrypt` | Decrypt secrets (run before `make up`) |
| `make encrypt` | Encrypt secrets (run before `git commit`) |
| `make backup` | Full backup: local + Backblaze B2 |
| `make backup-verify` | Test restore to /tmp |
| `make db-dump` | Manual PostgreSQL dump |
| `make trivy-scan` | Scan images for vulnerabilities |
| `make revoke-device DEVICE=...` | Emergency: revoke a stolen device |

> **Restarting core over SSH:** The core stack includes Tailscale, so `make restart STACK=core` will kill your SSH connection mid-restart, leaving containers in a broken state. Use `nohup` to detach the process from your terminal:
> ```bash
> nohup make restart STACK=core &
> ```
> Reconnect after ~30 seconds. Output is saved to `~/nohup.out`.

## Repository Structure

```
universe-647/
├── Makefile                    # All management commands
├── .sops.yaml                  # SOPS encryption rules
├── stacks/
│   ├── core/                   # Phase 2: Caddy, Tailscale, PostgreSQL, Authelia, etc.
│   ├── data/                   # Phase 3: Baïkal, n8n, Ntfy
│   ├── sophon/                 # Phase 4: Sophon (Go backend), Bifrost gateway, Ollama (dormant)
│   ├── tomoko/                 # Phase 4: Svelte PWA frontend for the second brain
│   ├── storage/                # Phase 5: Nextcloud + Redis
│   ├── red-coast/              # Phase 8: Jellyfin + Radarr/Sonarr/Prowlarr + qBittorrent/VPN
│   ├── voice/                  # Phase 6 (planned, scaffold only): Wyoming Whisper + Piper
│   └── smarthome/              # Phase 7 (planned, scaffold only): Home Assistant, Mosquitto, Zigbee2MQTT
├── scripts/
│   ├── setup.sh                # First-time server setup
│   ├── generate-secrets.sh     # Generate per-stack .env secrets
│   ├── backup.sh               # Nightly restic backup
│   ├── restore.sh              # Restore from backup
│   ├── revoke-device.sh        # Emergency device revocation
│   └── trivy-scan.sh           # Image vulnerability scanning
├── mobile/                     # iOS Shortcuts exports
├── agent-docs/                 # gitignored AI context (architecture, networking, phases, containers, monorepo)
└── docs/
    ├── NETWORK-OVERVIEW.md     # Container communication diagram reference
    ├── REMOTE-ACCESS.md        # Tailscale remote-access runbook
    ├── DISASTER-RECOVERY.md    # Full rebuild runbook
    └── RESEARCH-SECOND-BRAIN.md # sophon/tomoko design research
```

## Security

Three independent layers — compromising any single layer is insufficient for access:

1. **Tailscale + Device Approval** — Services are invisible to the public internet; new devices require admin approval. (Tailnet Lock is currently *off* — see [docs/REMOTE-ACCESS.md](docs/REMOTE-ACCESS.md) for the rationale.)
2. **Authelia + Mandatory WebAuthn** — Every request requires password + physical FIDO2 security key. Phishing-resistant by design.
3. **Application-Level Authorization** — Sophon's API trusts Caddy's `Remote-User` header; every container runs with dropped capabilities, read-only filesystems where possible, and CPU/memory/PID limits.

CrowdSec provides intrusion detection at the reverse proxy layer. Network segmentation into 11 trust-tiered Docker networks (10 in `core` + a `nextcloud_redis_net` sidecar) prevents lateral movement between containers.

## Backup

Three copies, two media, one offsite:

1. **Live data** on NVMe #2
2. **Local restic repo** on NVMe #2 (encrypted, deduplicated, point-in-time snapshots)
3. **Backblaze B2** offsite (encrypted, restic S3-compatible; the red-coast media library is excluded — it's re-acquirable)

Nightly at 2 AM via cron. Healthchecks.io alerts on missed backups. Monthly restore test. Full disaster recovery in 2–4 hours on any replacement hardware — see [docs/DISASTER-RECOVERY.md](docs/DISASTER-RECOVERY.md).