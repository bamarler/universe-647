# Universe 647

An AI-powered personal operating system running on an HP EliteDesk 800 G6 Mini. Self-hosted CRM, task management, workflow automation, local LLM inference, and smart home control all unified through a single chat interface via MCP.

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
| Cloudflare account | cloudflare.com | DNS, HTTPS certificates, R2 backup storage |
| Tailscale account | tailscale.com | Mesh VPN, Device Approval, Tailnet Lock |
| 1Password (or equivalent) | — | Stores age private key, restic password, API keys |
| Google Cloud Podcast API | cloud.google.com | Optional: audio morning briefings |

## Quick Start

```bash
# 1. Hardware setup (see docs/SETUP.md)
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
restic -r s3:your-r2-endpoint/u647-backup init

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
| 2 | `core` | 10 | Reverse proxy, VPN, SSO + 2FA, monitoring, DNS blocking, intrusion detection, UPS shutdown, backups |
| 3 | `data` | 5 | CRM, contacts sync, task management, workflow automation, push notifications |
| 4 | `sophon` | 4 | Local LLM, cloud LLM routing, chat interface, MCP tool gateway, morning briefings |
| 5 | `storage` | 2 | Self-hosted files, semantic search, Obsidian sync |
| 6 | `voice` | 2 | Server-side speech-to-text and text-to-speech |
| 7 | `smarthome` | 3 | Zigbee device control, smart home automations via AI |

**19 containers for MVP (Phases 2–4). 25 total across all phases.**

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
| `make backup` | Full backup: local + Cloudflare R2 |
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
│   ├── data/                   # Phase 3: Monica, Baïkal, n8n, Vikunja, Ntfy
│   ├── sophon/                 # Phase 4: Ollama (dormant), LiteLLM, Open WebUI, mcpo
│   ├── storage/                # Phase 5: Nextcloud + Redis
│   ├── voice/                  # Phase 6: Wyoming Whisper + Piper
│   └── smarthome/              # Phase 7: Home Assistant, Mosquitto, Zigbee2MQTT
├── scripts/
│   ├── setup.sh                # First-time server setup
│   ├── backup.sh               # Nightly restic backup
│   ├── restore.sh              # Restore from backup
│   ├── revoke-device.sh        # Emergency device revocation
│   └── trivy-scan.sh           # Image vulnerability scanning
├── mcp-servers/                # Custom MCP wrappers (Monica, Vikunja)
├── mobile/                     # iOS Shortcuts exports
└── docs/
    ├── SETUP.md                # Hardware + OS installation
    ├── SECURITY.md             # Three-layer auth, Docker hardening
    ├── NETWORKING.md           # Tailscale, ACLs, Cloudflare
    ├── NETWORK-OVERVIEW.md     # Container communication diagram reference
    ├── BACKUP.md               # 3-2-1 strategy, restic usage
    ├── DISASTER-RECOVERY.md    # Full rebuild runbook
    ├── DEVICE-ONBOARDING.md    # Add/revoke devices
    ├── SECRETS.md              # SOPS + age usage
    ├── STORAGE.md              # NVMe layout, why not RAID
    └── MOBILE.md               # iOS Shortcuts + PWA setup
```

## Security

Three independent layers — compromising any single layer is insufficient for access:

1. **Tailscale + Tailnet Lock + Device Approval** — Services are invisible to the public internet. New devices require admin approval and cryptographic signing before any traffic flows.
2. **Authelia + Mandatory WebAuthn** — Every request requires password + physical FIDO2 security key. Phishing-resistant by design.
3. **Application-Level Authorization** — Per-model MCP tool assignment in Open WebUI controls which tools each model can use. Docker containers run with dropped capabilities, read-only filesystems, and resource limits.

CrowdSec provides intrusion detection at the reverse proxy layer. Network segmentation into 8 trust-tiered Docker networks prevents lateral movement between containers.

## Backup

Three copies, two media, one offsite:

1. **Live data** on NVMe #2
2. **Local restic repo** on NVMe #2 (encrypted, deduplicated, point-in-time snapshots)
3. **Cloudflare R2** offsite (encrypted, zero egress fees)

Nightly at 2 AM via cron. Healthchecks.io alerts on missed backups. Monthly restore test. Full disaster recovery in 2–4 hours on any replacement hardware — see [docs/DISASTER-RECOVERY.md](docs/DISASTER-RECOVERY.md).