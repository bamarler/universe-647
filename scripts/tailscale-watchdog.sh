#!/usr/bin/env bash
# Deadman watchdog for host tailscaled (run from tailscale-watchdog.timer).
# Escalates on consecutive failures: restart tailscaled, then reboot.
# See docs/REMOTE-ACCESS.md for installation.
set -euo pipefail

RESTART_AFTER=2      # consecutive failures before restarting tailscaled
REBOOT_AFTER=5       # consecutive failures before rebooting the host
MIN_UPTIME_SECS=600  # never reboot within 10 min of boot (no boot loops)
STATE_DIR=/run/tailscale-watchdog
STATE_FILE="$STATE_DIR/failures"

mkdir -p "$STATE_DIR"
failures=$(cat "$STATE_FILE" 2>/dev/null || echo 0)

healthy() {
	# Backend running and logged in; --peers=false keeps it cheap.
	tailscale status --peers=false >/dev/null 2>&1
}

if healthy; then
	if [ "$failures" -gt 0 ]; then
		logger -t tailscale-watchdog "recovered after $failures failure(s)"
	fi
	echo 0 > "$STATE_FILE"
	exit 0
fi

failures=$((failures + 1))
echo "$failures" > "$STATE_FILE"
logger -t tailscale-watchdog "tailscale unhealthy (consecutive failures: $failures)"

if [ "$failures" -eq "$RESTART_AFTER" ]; then
	logger -t tailscale-watchdog "restarting tailscaled"
	systemctl restart tailscaled || true
fi

if [ "$failures" -ge "$REBOOT_AFTER" ]; then
	uptime_secs=$(cut -d. -f1 /proc/uptime)
	if [ "$uptime_secs" -lt "$MIN_UPTIME_SECS" ]; then
		logger -t tailscale-watchdog "would reboot but uptime < ${MIN_UPTIME_SECS}s; skipping"
		exit 0
	fi
	logger -t tailscale-watchdog "rebooting host (last resort after $failures failures)"
	echo 0 > "$STATE_FILE"
	systemctl reboot
fi
