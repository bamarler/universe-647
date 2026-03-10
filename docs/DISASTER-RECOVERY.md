# Disaster Recovery

**Recovery Time Objective: 2–4 hours** on any replacement x86 hardware.

This is a step-by-step runbook. Follow it in order. Every command is copy-pasteable.

---

## What You Need Access To

Before anything breaks, confirm you can reach all of these from a device that is NOT the server:

| Item | Location | What It Contains |
|------|----------|-----------------|
| 1Password (or equivalent) | Phone / laptop | age private key, restic repo password, Cloudflare R2 credentials, Tailscale OAuth |
| Git repository | github.com/you/universe-647 | All compose files, configs, scripts, encrypted secrets |
| Cloudflare R2 | dash.cloudflare.com | Offsite restic backup (encrypted) |
| Tailscale admin console | login.tailscale.com | Device management, ACLs, Tailnet Lock |
| Backup FIDO2 security key | Physical safe | Authelia WebAuthn authentication |

If you cannot access 1Password, you cannot recover. Treat your 1Password master password as the single most important credential in this system.

---

## Scenario 1: Full Hardware Failure

The EliteDesk is dead. Motherboard, PSU, or both NVMe drives failed simultaneously. You're starting from scratch on new hardware.

**Time estimate: 2–4 hours** (mostly waiting on restic restore over the internet).

### Step 1: Get replacement hardware (30 min)

Any x86 machine with at least one NVMe slot and 16GB RAM works. The stack has no hardware-specific dependencies. If you have two NVMe slots, great — replicate the original dual-drive layout. If only one, put everything on it and adjust mount paths.

### Step 2: Install Ubuntu Server 24.04 (15–30 min)

Follow `docs/SETUP.md`. The condensed version:

```bash
# Flash Ubuntu Server 24.04 ISO to USB via Balena Etcher
# Boot from USB, install to the primary NVMe
# Choose: entire disk with LVM, enable OpenSSH, minimized install
# After first boot, SSH in and set a static IP via Netplan
```

If you have a second NVMe for data:
```bash
sudo parted /dev/nvme1n1 mklabel gpt
sudo parted /dev/nvme1n1 mkpart primary ext4 0% 100%
sudo mkfs.ext4 /dev/nvme1n1p1
sudo mkdir -p /mnt/data
echo "UUID=$(sudo blkid -s UUID -o value /dev/nvme1n1p1) /mnt/data ext4 defaults 0 2" | sudo tee -a /etc/fstab
sudo mount -a
```

### Step 3: Clone repo and run setup (15 min)

```bash
git clone git@github.com:you/universe-647.git
cd universe-647
./scripts/setup.sh
```

This installs Docker, SOPS, age, restic, and NUT client. It creates all required directories.

### Step 4: Import age key and decrypt secrets (5 min)

Copy your age private key from 1Password:

```bash
mkdir -p ~/.config/sops/age
# Paste your age key into this file:
nano ~/.config/sops/age/keys.txt

# Decrypt all secrets
make decrypt
```

### Step 5: Restore data from Cloudflare R2 (1–3 hours)

This is the longest step. Speed depends on your internet bandwidth and how much data you have.

```bash
# Set restic environment variables (from 1Password)
export AWS_ACCESS_KEY_ID="your-r2-access-key"
export AWS_SECRET_ACCESS_KEY="your-r2-secret-key"
export RESTIC_PASSWORD="your-restic-repo-password"
export RESTIC_REPOSITORY="s3:https://your-account-id.r2.cloudflarestorage.com/u647-backup"

# List available snapshots
restic snapshots

# Restore the latest snapshot
restic restore latest --target /

# If you only have one drive and need to adjust paths:
restic restore latest --target /tmp/restore
# Then manually move data to the right locations
```

### Step 6: Restore PostgreSQL (10 min)

```bash
# Start just PostgreSQL first
docker compose -f stacks/core/compose.yaml up -d postgres

# Wait for it to be ready
sleep 10

# Find the most recent dump in the restored data
DUMP=$(ls -t /mnt/data/restic-repo-staging/pg_all_*.sql.gz 2>/dev/null | head -1)
# Or from the restore target:
DUMP=$(ls -t /tmp/restore/tmp/backup-staging-*/pg_all_*.sql.gz 2>/dev/null | head -1)

# Restore
gunzip -c "$DUMP" | docker exec -i postgres psql -U postgres

# Stop PostgreSQL (it'll restart with the full stack)
docker compose -f stacks/core/compose.yaml down
```

### Step 7: Start all containers (5 min)

```bash
make up
```

This starts stacks in dependency order: core → data → sophon → storage → voice → smarthome.

### Step 8: Re-join Tailscale (10 min)

```bash
sudo tailscale up --advertise-tags=tag:server
```

Then from an already-approved device:
1. Open the Tailscale admin console
2. Approve the new server
3. Sign its node key with Tailnet Lock: `tailscale lock sign nodekey:<id> tlpub:<signing-key>`
4. Update ACLs if the Tailscale IP changed

### Step 9: Verify (15 min)

```bash
# Check all containers are running
make status

# Verify backup system works on the new hardware
make backup-verify

# Scan images for vulnerabilities
make trivy-scan
```

Test each service manually:
- [ ] Homepage dashboard loads
- [ ] Authelia login works with security key
- [ ] Open WebUI responds
- [ ] Monica CRM data is intact
- [ ] Vikunja tasks are present
- [ ] Ntfy push notifications deliver
- [ ] Uptime Kuma shows all services green
- [ ] AdGuard DNS resolves correctly

### Step 10: Re-enable automated backups (5 min)

```bash
# Initialize the local restic repo on the new drive
restic -r /mnt/data/restic-repo init

# Verify the cron job is in place
crontab -l | grep backup

# If missing, add it:
(crontab -l 2>/dev/null; echo "0 2 * * * cd /home/$(whoami)/universe-647 && ./scripts/backup.sh") | crontab -
```

---

## Scenario 2: Data Drive Failure

The 500GB NVMe (data drive) died. The OS drive and all containers still work, but PostgreSQL, Ollama models, and application data are gone.

**Time estimate: 1–2 hours.**

```bash
# 1. Stop all stacks
make down

# 2. Replace the dead drive and format it
sudo parted /dev/nvme1n1 mklabel gpt
sudo parted /dev/nvme1n1 mkpart primary ext4 0% 100%
sudo mkfs.ext4 /dev/nvme1n1p1
sudo mount -a    # fstab entry should still exist

# 3. Recreate directory structure
sudo mkdir -p /mnt/data/{postgres,ollama/models,nextcloud,monica,open-webui,vikunja,home-assistant,restic-repo}
sudo chown -R $USER:$USER /mnt/data

# 4. Restore from Cloudflare R2
export AWS_ACCESS_KEY_ID="your-r2-access-key"
export AWS_SECRET_ACCESS_KEY="your-r2-secret-key"
export RESTIC_PASSWORD="your-restic-repo-password"
export RESTIC_REPOSITORY="s3:https://your-account-id.r2.cloudflarestorage.com/u647-backup"

restic restore latest --target /

# 5. Restore PostgreSQL
docker compose -f stacks/core/compose.yaml up -d postgres
sleep 10
DUMP=$(ls -t /tmp/backup-staging-*/pg_all_*.sql.gz 2>/dev/null | head -1)
gunzip -c "$DUMP" | docker exec -i postgres psql -U postgres
docker compose -f stacks/core/compose.yaml down

# 6. Re-download Ollama models (not backed up)
make up STACK=sophon
docker exec ollama ollama pull qwen2.5:7b

# 7. Start everything
make up

# 8. Initialize local restic repo on the new drive
restic -r /mnt/data/restic-repo init

# 9. Verify
make status
make backup
```

---

## Scenario 3: OS Drive Failure

The 250GB NVMe (OS drive) died. Data drive is intact with all databases and application data.

**Time estimate: 45 min–1 hour.** Fastest recovery scenario.

```bash
# 1. Install Ubuntu Server 24.04 on a replacement drive
#    (Follow docs/SETUP.md)

# 2. Mount the surviving data drive
sudo mkdir -p /mnt/data
# Find the drive's UUID:
sudo blkid
echo "UUID=<the-uuid> /mnt/data ext4 defaults 0 2" | sudo tee -a /etc/fstab
sudo mount -a

# 3. Clone repo, run setup, import age key
git clone git@github.com:you/universe-647.git
cd universe-647
./scripts/setup.sh

mkdir -p ~/.config/sops/age
# Paste age key from 1Password into ~/.config/sops/age/keys.txt

make decrypt

# 4. Start everything — data is already on the surviving drive
make up

# 5. Re-join Tailscale
sudo tailscale up --advertise-tags=tag:server
# Approve + sign from an existing device

# 6. Verify
make status
```

No restic restore needed — the data drive has everything.

---

## Scenario 4: Stolen or Lost Device

Your phone, laptop, or a security key was lost or stolen.

**Time estimate: 10 minutes.**

### Lost iPhone

```bash
# From your laptop (must be on Tailscale already):
make revoke-device DEVICE=nodekey:<iphone-node-key>
```

Then manually:
1. Open Tailscale admin console → remove the iPhone
2. Apple Find My → Erase iPhone
3. Rotate your identity provider password
4. When you have a new phone: install Tailscale, approve it, sign it with Tailnet Lock, register a new passkey in Authelia

### Lost primary security key

1. Log in using your **backup security key** or **iPhone passkey** (Face ID)
2. In Authelia, deregister the lost key
3. Buy a replacement key, register it in Authelia
4. Store the backup key back in the safe

### Lost laptop

```bash
make revoke-device DEVICE=nodekey:<laptop-node-key>
```

Same flow as lost iPhone: remove from Tailscale, flush sessions, set up replacement.

---

## Scenario 5: Lost ALL Security Keys

Both FIDO2 keys are gone and your iPhone passkey is unavailable (phone also lost/broken). You still have SSH access to the server via Tailscale from another approved device.

**This is the nuclear option. It bypasses Authelia entirely.**

```bash
# 1. SSH into the server
ssh user@<server-tailscale-ip>

# 2. Temporarily disable Authelia (access services directly)
cd ~/universe-647
docker compose -f stacks/core/compose.yaml stop authelia

# 3. Access services directly through Caddy (no SSO)
#    Register new security keys in Authelia's config:
#    Edit stacks/core/authelia/users_database.yml
#    Reset your user's WebAuthn credentials

# 4. Restart Authelia
docker compose -f stacks/core/compose.yaml up -d authelia

# 5. Log in and register new FIDO2 keys
#    Navigate to https://auth.home.yourdomain.com
#    Register primary key + backup key + new phone passkey
```

If you also don't have Tailscale access (all devices are gone), you need physical access to the server — plug in a keyboard and monitor, log in locally, and follow the steps above.

---

## Post-Recovery Checklist

Run through this after ANY recovery scenario:

```
[ ] make status                    — all containers running
[ ] make backup                    — backup completes successfully
[ ] make backup-verify             — restore test passes
[ ] make trivy-scan                — no critical vulnerabilities
[ ] Authelia login works           — password + security key
[ ] Ntfy push notification works   — test from Uptime Kuma
[ ] CrowdSec is active             — docker exec crowdsec cscli metrics
[ ] Tailnet Lock signers correct   — tailscale lock status
[ ] Cron job exists                — crontab -l | grep backup
```

## Post-Recovery Hardening

After recovering, tighten things up:

1. **Rotate credentials** — Change your Authelia password, generate new API tokens for any services that use them
2. **Check Tailscale peers** — Verify only your known devices are listed: `tailscale status`
3. **Review Tailnet Lock signers** — Ensure only your iPhone + server are signing nodes: `tailscale lock status`
4. **Run a full vulnerability scan** — `make trivy-scan`
5. **Verify backup schedule** — `crontab -l` should show the 2 AM backup cron
6. **Test a full backup cycle** — `make backup && make backup-verify`