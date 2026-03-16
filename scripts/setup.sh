#!/bin/bash
set -euo pipefail

echo "=== Universe 647 Server Setup ==="

# 1. Update system
sudo apt update && sudo apt upgrade -y

# 2. Install Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# 3. Install SOPS
SOPS_VERSION="3.12.1"
curl -LO https://github.com/getsops/sops/releases/download/v${SOPS_VERSION}/sops-v${SOPS_VERSION}.linux.amd64
sudo mv sops-v${SOPS_VERSION}.linux.amd64 /usr/local/bin/sops
sudo chmod +x /usr/local/bin/sops

# 4. Install age
sudo apt install -y age

# 5. Install restic
sudo apt install -y restic

# 6. Install bun (JavaScript/TypeScript runtime)
curl -fsSL https://bun.sh/install | bash

# 7. Install NUT client (host-level upsmon for graceful shutdown)
sudo apt install -y nut-client

# 8. Mount data drive (update UUID for your 500GB NVMe)
echo "⚠️  If not already mounted, edit /etc/fstab:"
echo "   UUID=your-uuid /mnt/data ext4 defaults 0 2"
sudo mkdir -p /mnt/data

# 9. Create data directories
sudo mkdir -p /mnt/data/{postgres,ollama/models,nextcloud,monica,open-webui,vikunja,home-assistant,restic-repo}
sudo mkdir -p /srv/u647/{tailscale,baikal,uptime-kuma,adguard,ntfy}
sudo chown -R $USER:$USER /mnt/data /srv/u647

# 10. Create age key directory
mkdir -p ~/.config/sops/age
echo "⚠️  Import your age key from 1Password into ~/.config/sops/age/keys.txt"

# 11. Initialize local restic repository
echo "⚠️  Initialize restic repo (you'll need your repo password from 1Password):"
echo "   restic -r /mnt/data/restic-repo init"
echo "   restic -r s3:s3.us-west-000.backblazeb2.com/your-bucket init"

# 12. Configure Tailscale
echo "⚠️  Install and configure Tailscale:"
echo "   curl -fsSL https://tailscale.com/install.sh | sh"
echo "   sudo tailscale up --advertise-tags=tag:server"
echo "   Then initialize Tailnet Lock from an admin device"

# 13. Configure host-level UPS monitoring
echo "⚠️  Connect UPS via USB, then configure /etc/nut/upsmon.conf"
echo "   for graceful shutdown on low battery"

echo "✓ Server setup complete. Run 'make decrypt && make up-phase2' to start."