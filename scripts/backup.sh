#!/usr/bin/env bash
# ============================================================
# RINCO — Backup all stateful data
# ============================================================
# Backs up:
#   * PostgreSQL   via pg_dump
#   * MongoDB      via mongodump
#   * MinIO        via mc mirror
#   * ScyllaDB     via nodetool snapshot
#   * ClickHouse   via FREEZE + cp
#   * Valkey       via SAVE
#
# Usage:
#   backup.sh [output-dir]
# ============================================================

set -euo pipefail

OUTPUT_DIR="${1:-./backups/$(date -u +%Y%m%d_%H%M%S)}"
COMPOSE_FILE="${COMPOSE_FILE:-infra/docker-compose.yml}"
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

mkdir -p "${OUTPUT_DIR}"

echo "==> Backup output directory: ${OUTPUT_DIR}"

# Helper to run a command inside a compose service container
compose_exec() {
    local svc="$1"; shift
    docker compose -f "${PROJECT_ROOT}/${COMPOSE_FILE}" exec -T "$svc" "$@"
}

# ----- Postgres -----
echo "==> Postgres: pg_dump"
if docker ps --format '{{.Names}}' | grep -q '^rinco-postgres$'; then
    docker exec rinco-postgres pg_dump -U rinco -d rinco > "${OUTPUT_DIR}/postgres.sql"
    echo "    -> ${OUTPUT_DIR}/postgres.sql"
else
    echo "    [skip] rinco-postgres container not running"
fi

# ----- MongoDB -----
echo "==> MongoDB: mongodump"
if docker ps --format '{{.Names}}' | grep -q '^rinco-mongodb$'; then
    docker exec rinco-mongodb mongodump --archive="${OUTPUT_DIR}/mongodb.archive"
    echo "    -> ${OUTPUT_DIR}/mongodb.archive"
else
    echo "    [skip] rinco-mongodb container not running"
fi

# ----- MinIO -----
echo "==> MinIO: mc mirror"
if docker ps --format '{{.Names}}' | grep -q '^rinco-minio$'; then
    if docker ps --format '{{.Names}}' | grep -q '^rinco-minio-init$'; then
        docker exec rinco-minio mc mirror --remove --overwrite /data "${OUTPUT_DIR}/minio" || true
        echo "    -> ${OUTPUT_DIR}/minio/"
    else
        echo "    [skip] mc client not initialized"
    fi
else
    echo "    [skip] rinco-minio container not running"
fi

# ----- ScyllaDB -----
echo "==> ScyllaDB: nodetool snapshot"
if docker ps --format '{{.Names}}' | grep -q '^rinco-scylla$'; then
    docker exec rinco-scylla nodetool snapshot -t "rinco-$(date +%s)" || true
    docker cp rinco-scylla:/var/lib/scylla/data/snapshots "${OUTPUT_DIR}/scylla-snapshot" 2>/dev/null || true
    echo "    -> ${OUTPUT_DIR}/scylla-snapshot/"
else
    echo "    [skip] rinco-scylla container not running"
fi

# ----- ClickHouse -----
echo "==> ClickHouse: freeze + copy"
if docker ps --format '{{.Names}}' | grep -q '^rinco-clickhouse$'; then
    docker exec rinco-clickhouse clickhouse-client --query "SYSTEM FREEZE" || true
    docker cp rinco-clickhouse:/var/lib/clickhouse "${OUTPUT_DIR}/clickhouse" 2>/dev/null || true
    echo "    -> ${OUTPUT_DIR}/clickhouse/"
else
    echo "    [skip] rinco-clickhouse container not running"
fi

# ----- Valkey -----
echo "==> Valkey: SAVE"
if docker ps --format '{{.Names}}' | grep -q '^rinco-valkey$'; then
    docker exec rinco-valkey valkey-cli SAVE || true
    docker cp rinco-valkey:/data/dump.rdb "${OUTPUT_DIR}/valkey-dump.rdb" 2>/dev/null || true
    echo "    -> ${OUTPUT_DIR}/valkey-dump.rdb"
else
    echo "    [skip] rinco-valkey container not running"
fi

# ----- Compress -----
echo "==> Compressing"
cd "$(dirname "${OUTPUT_DIR}")"
tar czf "${OUTPUT_DIR}.tar.gz" "$(basename "${OUTPUT_DIR}")"
echo "    -> ${OUTPUT_DIR}.tar.gz"

# Optionally remove the uncompressed copy
rm -rf "${OUTPUT_DIR}"

echo ""
echo "✓ Backup completed: ${OUTPUT_DIR}.tar.gz"
