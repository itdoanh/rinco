#!/usr/bin/env bash
# ============================================================
# Seed data loader (RINCO)
# ============================================================
# Loads SQL seed (migrations/sql/0011_seed.sql) + demo users
# into Mongo + ClickHouse so a freshly-clustered env immediately
# has demo content.
# ============================================================
set -euo pipefail
PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

: "${POSTGRES_HOST:=localhost}"
: "${POSTGRES_USER:=rinco}"
: "${POSTGRES_DB:=rinco}"
: "${POSTGRES_PASSWORD:=rinco_dev_password}"
export PGPASSWORD="${POSTGRES_PASSWORD}"

echo "==> Applying SQL seed"
psql -h "$POSTGRES_HOST" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -f \
  "${PROJECT_ROOT}/migrations/sql/0011_seed.sql" || true

echo "==> Loading Mongo demo data"
mongosh --quiet "mongodb://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:27017/rinco?authSource=admin" \
        --file "${PROJECT_ROOT}/infra/mongo/demo-data.js" || echo " (skipped Mongo seed)"

echo "==> Loading ClickHouse fixture data"
clickhouse-client --host localhost --queries-file \
  "${PROJECT_ROOT}/infra/clickhouse/seed-ch.sql" || true

echo "✓ Seed complete"
