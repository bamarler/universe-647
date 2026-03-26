#!/usr/bin/env python3
# Universe 647 — Uptime Kuma monitor seeding script
#
# Run via: make setup-monitors UPTIME_KUMA_USER=admin UPTIME_KUMA_PASS=yourpass
#
# Creates:
#   - Ntfy notification channel (topic: alerts)
#   - HTTP monitors for all web-facing services
#   - TCP monitor for NUT UPS
#   - DNS monitor for AdGuard resolver
#   - Docker Container monitors for all containers in both stacks
#
# Idempotent: skips monitors that already exist by name.

import os
import sys
import time

try:
    from uptime_kuma_api import UptimeKumaApi, MonitorType, NotificationType
except ImportError:
    print("ERROR: uptime-kuma-api not installed. Run: pip install uptime-kuma-api")
    sys.exit(1)

URL  = os.environ.get("UPTIME_KUMA_URL", "http://localhost:3001")
USER = os.environ.get("UPTIME_KUMA_USER")
PASS = os.environ.get("UPTIME_KUMA_PASS")

if not USER or not PASS:
    print("ERROR: UPTIME_KUMA_USER and UPTIME_KUMA_PASS must be set")
    sys.exit(1)

# ---------------------------------------------------------------------------
# Monitors to create
# ---------------------------------------------------------------------------

# HTTP monitors — internal container hostnames, no Caddy/Authelia in the path
HTTP_MONITORS = [
    # Core stack (app_net)
    {"name": "Homepage",     "url": "http://homepage:3000"},
    {"name": "Ntfy",         "url": "http://ntfy:80"},
    {"name": "Vikunja",      "url": "http://vikunja:3456"},
    # Monica intentionally excluded: redirects to APP_URL (external https://monica.DOMAIN),
    # not reachable from inside Docker. Docker Container monitor covers it.
    {"name": "Baikal",       "url": "http://baikal:80"},
    # n8n intentionally excluded: it's on automation_net only, not reachable
    # from uptime-kuma's networks. Docker Container monitor covers it instead.
    {"name": "Glances",      "url": "http://glances:61208"},
    # monitoring_net (directly reachable without app_net)
    {"name": "AdGuard UI",   "url": "http://adguard:80"},
    {"name": "Uptime Kuma",  "url": "http://localhost:3001"},
]

# TCP monitors
TCP_MONITORS = [
    {"name": "NUT UPS",  "hostname": "nut",  "port": 3493},
]

# DNS monitors — verify AdGuard is resolving correctly
DNS_MONITORS = [
    {
        "name":             "AdGuard DNS",
        "hostname":         "adguard",
        "port":             53,
        "dns_resolve_type": "A",
        "dns_resolve_server": "adguard",
        "dns_last_result":  "216.58.194.174",  # not checked — just verifies resolution works
    },
]

# Docker Container monitors — checks running state via socket-proxy
# socket-proxy must be reachable; it's on monitoring_net which uptime-kuma is on
DOCKER_MONITORS = [
    # Core stack
    "caddy",
    "tailscale",
    "postgres",
    "authelia",
    "homepage",
    "uptime-kuma",
    "adguard",
    "socket-proxy",
    "crowdsec",
    "nut",
    "glances",
    # Data stack
    "ntfy",
    "monica",
    "monica-db",
    "baikal",
    "n8n",
    "vikunja",
]

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def existing_monitor_names(api):
    monitors = api.get_monitors()
    return {m["name"] for m in monitors}

def existing_notification_names(api):
    notifications = api.get_notifications()
    return {n["name"] for n in notifications}

def add_if_new(api, name, existing, create_fn):
    if name in existing:
        print(f"  SKIP  {name} (already exists)")
        return None
    result = create_fn()
    print(f"  ADD   {name}")
    return result

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

print(f"Connecting to Uptime Kuma at {URL}...")

api = None
for attempt in range(1, 4):
    try:
        api = UptimeKumaApi(URL)
        api.login(USER, PASS)
        print("Logged in.")

        # Uptime Kuma v2 requires `conditions` (NOT NULL) but uptime-kuma-api
        # was written for v1 and omits it. Patch _call on this instance to
        # inject conditions=[] into every 'add' monitor event.
        _orig_call = api._call
        def _patched_call(event, data=None):
            if event == "add" and isinstance(data, dict) and data.get("conditions") is None:
                data["conditions"] = []
            return _orig_call(event, data)
        api._call = _patched_call
        print("(v2 compatibility patch applied)\n")
        break
    except Exception as e:
        print(f"  Attempt {attempt}/3 failed: [{type(e).__name__}] {e or '(no message)'}")
        if api:
            try:
                api.disconnect()
            except Exception:
                pass
            api = None
        if attempt < 3:
            print(f"  Retrying in 5s...")
            time.sleep(5)
else:
    print("ERROR: Login failed after 3 attempts.")
    print("Make sure Uptime Kuma is running and credentials are correct.")
    sys.exit(1)

# --- Docker Host (socket-proxy) ---
print("=== Docker Host ===")
docker_hosts = api.get_docker_hosts()
existing_docker_host_names = {h["name"] for h in docker_hosts}
docker_host_id = None

if "socket-proxy" in existing_docker_host_names:
    print("  SKIP  socket-proxy Docker host (already exists)")
    for h in docker_hosts:
        if h["name"] == "socket-proxy":
            docker_host_id = h["id"]
            break
else:
    result = api.add_docker_host(
        name="socket-proxy",
        dockerType="tcp",
        dockerDaemon="tcp://socket-proxy:2375",
    )
    docker_host_id = result.get("id")
    print(f"  ADD   socket-proxy Docker host  →  tcp://socket-proxy:2375 (id={docker_host_id})")

print()

# --- Ntfy notification channel ---
print("=== Notifications ===")
existing_notifs = existing_notification_names(api)
ntfy_id = None

if "Ntfy Alerts" in existing_notifs:
    print("  SKIP  Ntfy Alerts (already exists)")
    # Look up the ID so we can attach it to monitors
    for n in api.get_notifications():
        if n["name"] == "Ntfy Alerts":
            ntfy_id = n["id"]
            break
else:
    result = api.add_notification(
        type=NotificationType.NTFY,
        name="Ntfy Alerts",
        isDefault=True,
        applyExisting=True,
        ntfyserverurl="http://ntfy:80",
        ntfytopic="alerts",
        ntfyPriority=4,
    )
    ntfy_id = result.get("id")
    print(f"  ADD   Ntfy Alerts (id={ntfy_id})")

notification_ids = [ntfy_id] if ntfy_id else []

print()
print("=== HTTP Monitors ===")
existing = existing_monitor_names(api)

for m in HTTP_MONITORS:
    if m["name"] not in existing:
        api.add_monitor(
            type=MonitorType.HTTP,
            name=m["name"],
            url=m["url"],
            interval=60,
            retryInterval=30,
            maxretries=3,
            accepted_statuscodes=m.get("accepted_statuscodes", ["200-299"]),
            notificationIDList=notification_ids,
        )
        print(f"  ADD   {m['name']}  →  {m['url']}")
    else:
        print(f"  SKIP  {m['name']} (already exists)")

print()
print("=== TCP Monitors ===")
existing = existing_monitor_names(api)

for m in TCP_MONITORS:
    if m["name"] not in existing:
        api.add_monitor(
            type=MonitorType.PORT,
            name=m["name"],
            hostname=m["hostname"],
            port=m["port"],
            interval=60,
            retryInterval=30,
            maxretries=3,
            notificationIDList=notification_ids,
        )
        print(f"  ADD   {m['name']}  →  {m['hostname']}:{m['port']}")
    else:
        print(f"  SKIP  {m['name']} (already exists)")

print()
print("=== DNS Monitors ===")
existing = existing_monitor_names(api)

for m in DNS_MONITORS:
    if m["name"] not in existing:
        api.add_monitor(
            type=MonitorType.DNS,
            name=m["name"],
            hostname="google.com",
            port=m["port"],
            dns_resolve_server=m["dns_resolve_server"],
            dns_resolve_type=m["dns_resolve_type"],
            interval=120,
            retryInterval=60,
            maxretries=3,
            notificationIDList=notification_ids,
        )
        print(f"  ADD   {m['name']}")
    else:
        print(f"  SKIP  {m['name']} (already exists)")

print()
print("=== Docker Container Monitors ===")
existing = existing_monitor_names(api)

for container in DOCKER_MONITORS:
    monitor_name = f"Container: {container}"
    if monitor_name not in existing:
        api.add_monitor(
            type=MonitorType.DOCKER,
            name=monitor_name,
            docker_container=container,
            docker_host=docker_host_id,
            interval=60,
            retryInterval=30,
            maxretries=3,
            notificationIDList=notification_ids,
        )
        print(f"  ADD   {monitor_name}")
    else:
        print(f"  SKIP  {monitor_name} (already exists)")

print()
print("Done. Disconnecting...")
api.disconnect()
print("All monitors configured.")
