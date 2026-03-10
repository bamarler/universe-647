# Container Communication Map

Reference for the architecture diagram. Every line in the diagram represents a connection listed below. If a path isn't listed here, those containers **cannot** talk to each other.

---

## How Traffic Flows In and Out

**Inbound (internet → server): BLOCKED.** The server has no open ports on the public internet. No port forwarding, no public IP binding. The only way in is through Tailscale's encrypted WireGuard tunnel, which requires Device Approval + Tailnet Lock. DNS records point to Tailscale IPs (100.x.x.x), which are unroutable from the public internet.

**Outbound (server → internet): ALLOWED** on non-`internal` Docker networks. Containers make direct HTTPS requests through the host's normal internet connection (home router NAT). This is how LiteLLM calls Claude/OpenAI, n8n polls Gmail, CrowdSec syncs IP reputation, and restic uploads to Cloudflare R2. These requests do NOT route through Caddy or Tailscale — they go straight out via the host network.

**LAN (WiFi devices → server): MOSTLY BLOCKED.** Only three containers bind ports to the host: Caddy (`127.0.0.1:443` — localhost only, reachable via Tailscale but not LAN), AdGuard Home (`0.0.0.0:53` — must be LAN-reachable for DNS), and Tailscale (`network_mode: host`). All other containers have no published ports and are unreachable from the LAN even by port scanning.

---

## Inbound Path (Tailscale → Caddy → Services)

| From | To | Protocol | Data Flow |
|------|----|----------|-----------|
| iPhone / Laptop | **Tailscale** | WireGuard (UDP) | Encrypted tunnel into the private network |
| **Tailscale** | **Caddy** | HTTPS (443) | Decrypted request forwarded to reverse proxy |

---

## Authentication Layer (`auth_net`)

Every request to any service passes through Authelia first. Caddy asks Authelia "is this user authenticated?" before forwarding to the backend.

| From | To | Protocol | Data Flow |
|------|----|----------|-----------|
| **Caddy** | **Authelia** | HTTP (9091) | `forward_auth` check — validates session cookie or triggers login |
| **Authelia** | **PostgreSQL** | PostgreSQL (5432) | Session storage, user database queries |
| **Authelia** → | iPhone / Laptop | HTTPS | WebAuthn challenge (security key tap / Face ID) |

---

## Application Backends (`app_net`)

Caddy routes authenticated requests to the right backend based on subdomain.

| From | To | Protocol | Data Flow |
|------|----|----------|-----------|
| **Caddy** | **Monica CRM** | HTTP (8080) | `monica.home.domain.com` → contact management UI |
| **Caddy** | **Baïkal** | HTTP (80) | `dav.home.domain.com` → CardDAV/CalDAV sync |
| **Caddy** | **Vikunja** | HTTP (3456) | `tasks.home.domain.com` → task management UI + API |
| **Caddy** | **Ntfy** | HTTP (80) | `ntfy.home.domain.com` → push notification API |
| **Caddy** | **Homepage** | HTTP (3000) | `home.domain.com` → dashboard |
| **Caddy** | **Nextcloud** | HTTP (80) | `files.home.domain.com` → file storage UI + WebDAV |
| **Caddy** | **Home Assistant** | HTTP (8123) | `ha.home.domain.com` → smart home UI |

---

## AI / LLM Stack (`ai_net`)

The AI pipeline: user talks to Open WebUI, which calls LiteLLM for model routing and MCP Gateway for tools.

| From | To | Protocol | Data Flow |
|------|----|----------|-----------|
| **Caddy** | **Open WebUI** | HTTP (8080) | `chat.home.domain.com` → chat interface |
| **Open WebUI** | **LiteLLM** | HTTP (4000) | LLM API requests (prompt → completion) |
| **LiteLLM** | **Ollama** | HTTP (11434) | "local" model requests → 7B inference on CPU |
| **LiteLLM** | Claude / OpenAI API | HTTPS (443) | "smart" model requests → cloud (direct outbound via host network) |
| **Open WebUI** | **MCP Gateway** | HTTP/stdio | Tool call requests (e.g., "add a task", "who is John?") |
| **MCP Gateway** | **Monica CRM** | HTTP (8080) | CRM queries: contacts, notes, relationships |
| **MCP Gateway** | **Vikunja** | HTTP (3456) | Task CRUD: create, complete, query tasks |
| **MCP Gateway** | **Ntfy** | HTTP (80) | Human-in-the-loop confirmations for destructive actions |
| **MCP Gateway** | Google Calendar API | HTTPS (443) | Read-only schedule queries (direct outbound via host network) |
| **MCP Gateway** | **PostgreSQL** | PostgreSQL (5432) | Direct database queries |
| **MCP Gateway** | **Home Assistant** | HTTP (8123) | Smart home control: lights, sensors, automations |
| **MCP Gateway** | **Nextcloud** (Astrolabe) | HTTP (80) | Semantic file search |
| **Open WebUI** | **Wyoming Whisper** | TCP (10300) | Audio → text (speech-to-text) |
| **Open WebUI** | **Wyoming Piper** | TCP (10200) | Text → audio (text-to-speech) |

---

## Automation (`automation_net`)

n8n reaches services through Caddy (their authenticated API endpoints), not directly. This is why n8n sits on its own isolated network.

| From | To | Protocol | Data Flow |
|------|----|----------|-----------|
| **n8n** | **Caddy** → Monica | HTTPS | Contact sync: export vCards, push to Baïkal |
| **n8n** | **Caddy** → Baïkal | HTTPS | CardDAV push: synced contacts for iOS |
| **n8n** | **Caddy** → Vikunja | HTTPS | Task creation: Canvas assignments, email action items |
| **n8n** | **Caddy** → Open WebUI | HTTPS | Post morning briefing to new chat |
| **n8n** | **Caddy** → Ntfy | HTTPS | Push notifications: alerts, reminders, briefing links |
| **n8n** | **Caddy** → LiteLLM | HTTPS | LLM calls: email classification, briefing generation |
| **n8n** | **PostgreSQL** | PostgreSQL (5432) | n8n's own workflow/execution database |
| **n8n** | Gmail API | HTTPS (443) | Email polling + send (via Google OAuth) |
| **n8n** | Outlook API | HTTPS (443) | Email polling (via Microsoft OAuth) |
| **n8n** | Google Calendar API | HTTPS (443) | Read events (source of truth for schedule) |
| **n8n** | Canvas LMS API | HTTPS (443) | Assignment due dates → Vikunja tasks |
| **n8n** | Weather API | HTTPS (443) | Morning briefing weather data |

---

## Database Layer (`db_net`)

PostgreSQL is the shared database. Only containers that need database access are on this network.

| From | To | Protocol | Data Flow |
|------|----|----------|-----------|
| **Authelia** | **PostgreSQL** | PostgreSQL (5432) | User sessions, 2FA registration data |
| **Open WebUI** | **PostgreSQL** | PostgreSQL (5432) | Chat history, user preferences, RAG metadata |
| **n8n** | **PostgreSQL** | PostgreSQL (5432) | Workflow definitions, execution logs, credentials |
| **Vikunja** | **PostgreSQL** | PostgreSQL (5432) | Tasks, projects, labels, team data |
| **Nextcloud** | **PostgreSQL** | PostgreSQL (5432) | File metadata, shares, app data |

Monica uses its own **MariaDB** sidecar (bundled in the Monica compose), not the shared PostgreSQL.

---

## IoT / Smart Home (`iot_net`)

Completely isolated. No other containers can reach Mosquitto.

| From | To | Protocol | Data Flow |
|------|----|----------|-----------|
| **Zigbee2MQTT** | **Mosquitto** | MQTT (1883/TLS) | Zigbee device state → MQTT topics (e.g., `zigbee2mqtt/light/state`) |
| **Mosquitto** | **Home Assistant** | MQTT (1883/TLS) | HA subscribes to device topics, publishes commands |
| **Home Assistant** | **Mosquitto** | MQTT (1883/TLS) | Control commands → Zigbee2MQTT → physical devices |
| **Zigbee2MQTT** | SONOFF Dongle (USB) | Serial | Raw Zigbee radio communication with physical devices |

Home Assistant is multi-homed: it's on `iot_net` for MQTT and on `app_net` so Caddy can serve its web UI.

---

## Monitoring (`monitoring_net`)

These containers observe the system. They don't serve user-facing traffic.

| From | To | Protocol | Data Flow |
|------|----|----------|-----------|
| **CrowdSec** | Caddy access logs | File read | Parses `/var/log/caddy/access.log` for attack patterns |
| **CrowdSec** | CrowdSec Central API | HTTPS (443) | Shares + receives crowd-sourced IP reputation data |
| **Caddy** | **CrowdSec** (bouncer) | HTTP (8080) | Bouncer checks: "is this IP blocked?" before routing |
| **Uptime Kuma** | **Docker Socket Proxy** | TCP (2375) | Container status: running, stopped, health |
| **Homepage** | **Docker Socket Proxy** | TCP (2375) | Container metadata for dashboard widgets |
| **Uptime Kuma** | All containers | HTTP (various) | Health checks: ping each service endpoint |
| **Uptime Kuma** | **Ntfy** | HTTP (80) | Outage alerts → push notification |
| **NUT** | UPS (USB) | USB/Serial | Battery level, load, runtime remaining |
| **NUT** | Host `upsmon` | TCP (3493) | Shutdown signal when battery critical |

---

## iOS Device Connections (External)

These aren't Docker-to-Docker flows, but they're important for the diagram since the phone is a primary interface.

| From | To | Via | Data Flow |
|------|----|-----|-----------|
| iPhone Contacts | **Baïkal** | CardDAV over Tailscale | Native contact sync |
| iPhone Reminders | **Vikunja** | CalDAV over Tailscale | Native task sync |
| iPhone Ntfy app | **Ntfy** | HTTPS over Tailscale | Push notifications |
| iPhone PWA / Conduit | **Open WebUI** | HTTPS over Tailscale | Chat interface |
| iOS Vocal Shortcut | **n8n** webhook | HTTPS over Tailscale | Voice → LLM → spoken response |
| iPhone Tailscale app | Tailscale coordination | HTTPS | Network management, device approval |

---

## Outbound Internet (Server → External APIs)

These connections go directly from the container through the host's internet connection (NAT via home router). They do NOT route through Caddy or Tailscale.

| Container | Destination | Protocol | Data Flow |
|-----------|------------|----------|-----------|
| **LiteLLM** | Claude / OpenAI API | HTTPS (443) | "smart" model completions (sends prompts off-device) |
| **n8n** | Gmail API | HTTPS (443) | Email polling + send |
| **n8n** | Outlook API | HTTPS (443) | Email polling |
| **n8n** | Google Calendar API | HTTPS (443) | Read events (source of truth) |
| **n8n** | Canvas LMS API | HTTPS (443) | Assignment due dates |
| **n8n** | Weather API | HTTPS (443) | Morning briefing data |
| **MCP Gateway** | Google Calendar API | HTTPS (443) | Schedule queries |
| **CrowdSec** | CrowdSec Central API | HTTPS (443) | IP reputation sync (send + receive) |
| **Caddy** | Let's Encrypt / Cloudflare | HTTPS (443) | SSL certificate issuance via DNS challenge |
| **Ollama** | ollama.com / HuggingFace | HTTPS (443) | Model weight downloads (first pull only) |
| **restic** (host cron) | Cloudflare R2 | HTTPS (443) | Encrypted backup upload (S3-compatible) |

Containers on `internal: true` networks (`auth_net`, `db_net`, `app_net`, `iot_net`) **cannot** make outbound requests. A compromised container on those networks cannot exfiltrate data or contact a command-and-control server.

---

## Connections That Do NOT Exist

For clarity in the diagram, these are intentionally blocked by network segmentation:

- Ollama ✗ PostgreSQL — AI models cannot access the database directly
- Mosquitto ✗ anything outside `iot_net` — MQTT broker is fully isolated
- n8n ✗ direct container access — n8n reaches services through Caddy's authenticated API, never direct
- Any container ✗ Docker socket — only the Socket Proxy mounts the socket; all others use TCP proxy
- CrowdSec ✗ application data — CrowdSec reads Caddy logs only, never application databases
- LAN devices ✗ any container except AdGuard (port 53) — no other ports published to host