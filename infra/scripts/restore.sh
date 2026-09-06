#!/usr/bin/env bash
# ============================================================
# Restore RINCO state
# ============================================================
# Usage: restore.sh <archive_url>
#   archive_url  s3://rinco-backups/<date>/rinco.tgz
#                or  /local/path/rinco.tgz
# ============================================================
set -euo pipefail

ARCH="${1:?usage: $0 <archive_url|s3 path>}"
WORK=/tmp/rinco-restore
rm -rf "$WORK" && mkdir -p "$WORK"

echo "==> Downloading archive"
if [[ "$ARCH" =~ ^s3:// ]]; then
  aws s3 cp "$ARCH" "$WORK/rinco.tgz"
else
  cp "$ARCH" "$WORK/rinco.tgz"
fi
tar xzf "$WORK/rinco.tgz" -C "$WORK"

echo "==> Stopping services"
docker compose -f "$WORK/../../infra/docker-compose.yml" stop \
  postgres scylla clickhouse valkey mongo || true

echo "==> Restoring Postgres"
pg_restore --clean --jobs=4 --dbname=rinco "$WORK/rinco-pg" || true

echo "==> Restoring Valkey"
docker cp "$WORK/valkey-dump.rdb" rinco-valkey:/data/dump.rdb
docker exec rinco-valkey valkey-cli DEBUG RELOAD

echo "==> Restoring ScyllaDB"
for cf in $WORK/scylla-snapshot/*; do
  docker cp "$cf" rinco-scylla:/var/lib/scylla/data/$(basename "$cf")
done
docker exec rinco-scylla nodetool refresh || true

echo "==> Restarting services"
docker compose -f "$WORK/../../infra/docker-compose.yml" start || true

echo "✓ Restore complete."
