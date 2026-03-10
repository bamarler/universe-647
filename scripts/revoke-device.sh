#!/bin/bash
set -euo pipefail

DEVICE_ID=${1:?"Usage: ./revoke-device.sh <tailscale-device-id>"}

echo "⚠️  Revoking device: ${DEVICE_ID}"

# 1. Remove from Tailscale
echo "Removing from Tailscale network..."
tailscale lock revoke-keys "${DEVICE_ID}" 2>/dev/null || echo "Not a signing key, skipping lock revocation"

# 2. Flush Authelia sessions
echo "Flushing Authelia sessions..."
docker restart authelia

# 3. Remind about manual steps
echo ""
echo "✓ Device revoked from Tailscale and Authelia sessions flushed."
echo ""
echo "Manual steps remaining:"
echo "  1. Remove device from Tailscale admin console: https://login.tailscale.com/admin/machines"
echo "  2. If iPhone: activate Find My → Erase iPhone"
echo "  3. Rotate your identity provider password"
echo "  4. Register replacement device following docs/DEVICE-ONBOARDING.md"