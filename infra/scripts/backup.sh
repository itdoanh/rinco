#!/usr/bin/env bash
# ============================================================
# Backup RINCO state
# ============================================================
# - pg_dump for Postgres
# - Scylla snapshot via nodetool
# - Valkey RDB snapshot
# - MinIO mc mirror
# - ClickHouse frozen parts
#
# Stores in S3/MinIO bucket "rinco-backups" with date prefix.
# ============================================================
set -euo pipefail

: "${BACKUP_DEST:=s3://rinco-backups/$(date -u +%Y%m%d)}"
: "${BACKUP_BIN:=$(command -v aws) || echo ''}"
: "${POSTGRES_HOST:=localhost}"
: "${POSTGRES_USER:=rinco}"
: "${POSTGRES_PASSWORD:=rinco_dev_password}"
export PGPASSWORD="${POSTGRES_PASSWORD}"

echo "==> Postgres"
pg_dump --format=directory --jobs=4 --file=/tmp/rinco-pg \
        "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:5432/rinco"

echo "==> ScyllaDB snapshot"
docker exec rinco-scylla nodetool snapshot -t "rinco-$(date +%s)" || true
docker cp rinco-scylla:/var/lib/scylla/data/snapshots /tmp/scylla-snapshot

echo "==> Valkey"
docker exec rinco-valkey valkey-cli SAVE
docker cp rinco-valkey:/data/dump.rdb /tmp/valkey-dump.rdb

echo "==> MinIO mirror"
mc mirror --remove --overwrite local/rinco-backups "${BACKUP_DEST}/files" || true

echo "==> ClickHouse (frozen parts)"
clickhouse-client --query "SYSTEM FREEZE" || true
cp -a /var/lib/clickhouse/data /tmp/clickhouse-data || true

echo "==> Upload to destination ${BACKUP_DEST}"
tar czf /tmp/rinco-backup.tgz \
   /tmp/rinco-pg /tmp/scylla-snapshot /tmp/valkey-dump.rdb /tmp/clickhouse-data
aws s3 cp /tmp/rinco-backup.tgz "${BACKUP_DEST}/rinco.tgz"

echo "✓ Backup stored in ${BACKUP_DEST}"
