#!/bin/bash
set -euo pipefail

DATE=$(date +%Y%m%d_%H%M)
BACKUP_STAGING=/tmp/backup-staging-${DATE}
LOCAL_REPO=/mnt/data/restic-repo
CLOUD_REPO=s3:s3.us-west-000.backblazeb2.com/your-bucket  # Update with your bucket
HEALTHCHECK_URL=https://hc-ping.com/YOUR-UUID-HERE         # Update with your ping URL

mkdir -p ${BACKUP_STAGING}

# 1. Dump PostgreSQL
echo "Dumping PostgreSQL..."
docker exec postgres pg_dumpall -U postgres | gzip -9 > ${BACKUP_STAGING}/pg_all_${DATE}.sql.gz

# 2. Put Nextcloud in maintenance mode (if running)
if docker ps --format '{{.Names}}' | grep -q nextcloud; then
    docker exec nextcloud php occ maintenance:mode --on 2>/dev/null || true
fi

# 3. Back up to local restic repo
echo "Backing up to local repo..."
restic -r ${LOCAL_REPO} backup \
    ${BACKUP_STAGING} \
    /mnt/data/monica \
    /mnt/data/vikunja \
    /mnt/data/open-webui \
    /mnt/data/home-assistant \
    /mnt/data/nextcloud \
    --exclude='*/cache/*' \
    --exclude='*/logs/*'

# 4. Back up to cloud
echo "Backing up to cloud..."
restic -r ${CLOUD_REPO} backup \
    ${BACKUP_STAGING} \
    /mnt/data/monica \
    /mnt/data/vikunja \
    /mnt/data/open-webui \
    /mnt/data/home-assistant \
    /mnt/data/nextcloud \
    --exclude='*/cache/*' \
    --exclude='*/logs/*'

# 5. Restore Nextcloud
if docker ps --format '{{.Names}}' | grep -q nextcloud; then
    docker exec nextcloud php occ maintenance:mode --off 2>/dev/null || true
fi

# 6. Prune old snapshots (keep 7 daily, 4 weekly, 6 monthly)
echo "Pruning old snapshots..."
restic -r ${LOCAL_REPO} forget --keep-daily 7 --keep-weekly 4 --keep-monthly 6 --prune
restic -r ${CLOUD_REPO} forget --keep-daily 7 --keep-weekly 4 --keep-monthly 6 --prune

# 7. Weekly integrity check (Sundays)
if [ "$(date +%u)" -eq 7 ]; then
    echo "Running weekly integrity check..."
    restic -r ${LOCAL_REPO} check
fi

# 8. Clean up staging
rm -rf ${BACKUP_STAGING}

# 9. Signal success to healthchecks.io
curl -fsS -m 10 --retry 5 ${HEALTHCHECK_URL} > /dev/null

echo "✓ Backup complete"