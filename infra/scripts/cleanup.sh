#!/usr/bin/env bash
# ============================================================
# Cleanup all RINCO local-dev containers/volumes
# ============================================================
# Use with care. Removes:
#   * docker compose stack
#   * dangling volumes
#   * network rinco-network
# ============================================================
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

echo "==> Stop & remove compose stack"
docker compose -f "${PROJECT_ROOT}/infra/docker-compose.yml" down -v --remove-orphans

echo "==> Prune system"
docker system prune -af

echo "==> Remove rinco-network"
docker network rm rinco-network || true

echo "==> Optional: remove data mount points"
if [ "${PURGE_DEEP:-false}" = "true" ]; then
  sudo rm -rf /var/lib/rinco/{postgres,clickhouse,valkey,scylla} 2>/dev/null || true
fi

echo "✓ Cleanup complete."
