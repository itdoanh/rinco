#!/usr/bin/env bash
# ============================================================
# RINCO dev orchestration script (Loop WS-A)
# ============================================================
# Brings up the full local stack:
#   1. infra compose (postgres + scylla + clickhouse + mongo + valkey + ...)
#   2. waits for DBs to be healthy
#   3. runs SQL migrations + multi-DB seed
#   4. starts 17 backend services in background (logs/<svc>.log)
#   5. starts 4 frontend apps in background (logs/<fe>.log)
#   6. prints a status table
#
# Idempotent: re-running starts any container that was stopped.
# Stop with: scripts/stop-all.sh
# ============================================================
set -uo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
INFRA_DIR="${PROJECT_ROOT}/infra"
LOG_DIR="${PROJECT_ROOT}/logs"
PIDS_DIR="${PROJECT_ROOT}/.dev/pids"

mkdir -p "${LOG_DIR}" "${PIDS_DIR}"

# ---- Compose command ----------------------------------------------------
get_compose_cmd() {
    if docker compose version &>/dev/null; then
        echo "docker compose"
    elif command -v docker-compose &>/dev/null; then
        echo "docker-compose"
    else
        echo "docker compose"
    fi
}
COMPOSE_CMD=$(get_compose_cmd)

# ---- .env bootstrap -----------------------------------------------------
ENV_FILE="${PROJECT_ROOT}/.env"
if [ ! -f "$ENV_FILE" ]; then
    echo -e "${YELLOW}[env]${NC} creating default .env"
    cat > "$ENV_FILE" <<'EOF'
POSTGRES_PASSWORD=rinco_dev_password
CLICKHOUSE_PASSWORD=rinco_dev_password
VALKEY_PASSWORD=rinco_dev_password
MONGO_PASSWORD=rinco_dev_password
MINIO_ROOT_PASSWORD=rinco_dev_password
PASETO_KEY_CURRENT=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
PASETO_KEY_PREVIOUS=fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210
ADMIN_API_KEY=dev_admin_key_change_me
MEILI_MASTER_KEY=masterKey
EOF
fi

# ---- Service definitions (canonical port table) -------------------------
# Format: NAME|REPO_DIR|HEALTH_PATH|RESOURCE
SERVICES=(
    "auth-service|services/auth-service|http://localhost:8081/health|go"
    "tenant-service|services/tenant-service|http://localhost:8082/health|go"
    "crm-service|services/crm-service|http://localhost:8083/health|go"
    "dynamic-model-service|services/dynamic-model-service|http://localhost:8084/health|go"
    "lead-service|services/lead-service|http://localhost:8085/health|go"
    "landing-service|services/landing-service|http://localhost:8086/health|go"
    "email-service|services/email-service|http://localhost:8087/health|go"
    "notification-service|services/notification-service|http://localhost:8088/health|go"
    "billing-service|services/billing-service|http://localhost:8095/health|go"
    "observability-service|services/observability-service|http://localhost:8096/health|go"
    "search-service|services/search-service|http://localhost:8097/health|go"
    "meta-capi-service|services/meta-capi-service|http://localhost:8098/health|go"
    "analytics-service|services/analytics-service|http://localhost:8099/health|go"
    "ai-sre|services/ai-sre|http://localhost:8090/health|python"
    "lead-scoring|services/lead-scoring|http://localhost:8091/health|python"
    "rag-chatbot|services/rag-chatbot|http://localhost:8092/health|python"
    "stt-service|services/stt-service|http://localhost:8094/health|python"
    "recording-service|services/recording-service|http://localhost:8093/health|rust"
    "chat-engine|services/chat-engine|http://localhost:8101/health|rust"
    "webrtc-sfu|services/webrtc-sfu|http://localhost:8102/health|rust"
)

FRONTENDS=(
    "landing|frontend/landing|http://localhost:3000/health"
    "admin-portal|frontend/admin-portal|http://localhost:3001/health"
    "tenant-site|frontend/tenant-site|http://localhost:3002/health"
    "meeting-ui|frontend/meeting-ui|http://localhost:3003/health"
)

# ---- Helpers ------------------------------------------------------------
say()   { printf "${BLUE}[%s]${NC} %s\n" "$(date +%H:%M:%S)" "$1"; }
ok()    { printf "${GREEN}[OK]${NC} %s\n" "$1"; }
warn()  { printf "${YELLOW}[WARN]${NC} %s\n" "$1"; }
err()   { printf "${RED}[FAIL]${NC} %s\n" "$1"; }

wait_for() {
    # wait_for <url> <label> <max_seconds>
    local url="$1" label="$2" max="${3:-120}" waited=0
    while [ $waited -lt $max ]; do
        if curl -fsS -o /dev/null --max-time 2 "$url"; then
            return 0
        fi
        sleep 2
        waited=$((waited + 2))
    done
    return 1
}

# ---- 1. Docker network --------------------------------------------------
say "creating rinco-network if missing"
docker network inspect rinco-network &>/dev/null || \
    docker network create --driver bridge --subnet=172.25.0.0/16 rinco-network

# ---- 2. Infra compose ---------------------------------------------------
say "starting infrastructure stack (postgres + scylla + clickhouse + ...)"
cd "$INFRA_DIR"
$COMPOSE_CMD -f docker-compose.yml up -d

# ---- 3. Wait for DBs ----------------------------------------------------
say "waiting for Postgres + Valkey + NATS healthchecks..."
for i in {1..60}; do
    if docker exec rinco-postgres pg_isready -U rinco &>/dev/null && \
       docker exec rinco-valkey valkey-cli ping &>/dev/null && \
       docker exec rinco-nats nats-server --version &>/dev/null; then
        ok "infra healthy"
        break
    fi
    sleep 5
done

# ---- 4. Migrations + seed ----------------------------------------------
say "running migrations"
if [ -x "${PROJECT_ROOT}/scripts/migrate.sh" ]; then
    bash "${PROJECT_ROOT}/scripts/migrate.sh" || warn "migrate.sh failed (continuing)"
else
    warn "scripts/migrate.sh missing — skipping"
fi

say "seeding databases"
if [ -x "${PROJECT_ROOT}/scripts/seed-all.sh" ]; then
    bash "${PROJECT_ROOT}/scripts/seed-all.sh" || warn "seed-all.sh failed (continuing)"
elif [ -f "${PROJECT_ROOT}/scripts/seed-all.ps1" ]; then
    warn "found scripts/seed-all.ps1 — please run it once on Windows hosts"
else
    warn "no seed script found — skipping"
fi

# ---- 5. Services compose ------------------------------------------------
say "starting backend services via docker compose"
$COMPOSE_CMD -f docker-compose.services.yml up -d

# ---- 6. Wait for services ----------------------------------------------
say "waiting for backend /health endpoints..."
for entry in "${SERVICES[@]}"; do
    IFS='|' read -r name dir url lang <<< "$entry"
    if wait_for "$url" "$name" 60; then
        ok "$name ($lang) → $url"
    else
        warn "$name not healthy within 60s — see logs/$name.log if local"
    fi
done

# ---- 7. Frontends (local dev servers) ----------------------------------
say "starting 4 frontend dev servers (Bun) — each in background"
for entry in "${FRONTENDS[@]}"; do
    IFS='|' read -r name dir url <<< "$entry"
    log="${LOG_DIR}/${name}.log"
    pidf="${PIDS_DIR}/${name}.pid"
    cd "${PROJECT_ROOT}/${dir}"
    if [ ! -d node_modules ] && [ ! -d .bun ]; then
        warn "$name: node_modules missing — run 'bun install' in $dir first"
        continue
    fi
    if [ -f package.json ]; then
        # Use Bun when available, fall back to npm.
        if command -v bun &>/dev/null; then
            nohup bun run dev >>"$log" 2>&1 &
        else
            nohup npm run dev >>"$log" 2>&1 &
        fi
        echo $! >"$pidf"
        ok "$name started (pid $(cat "$pidf")) → $log"
    else
        warn "$name: no package.json"
    fi
done
cd "$PROJECT_ROOT"

# ---- 8. Status table ----------------------------------------------------
say "status summary"
printf "\n%-22s %-8s %-44s\n" "service" "lang" "url"
printf '%-22s %-8s %-44s\n' "----------------------" "------" "--------------------------------------------"
for entry in "${SERVICES[@]}"; do
    IFS='|' read -r name dir url lang <<< "$entry"
    if curl -fsS -o /dev/null --max-time 2 "$url"; then
        printf "${GREEN}✓${NC} %-20s %-8s %s\n" "$name" "$lang" "$url"
    else
        printf "${RED}✗${NC} %-20s %-8s %s\n" "$name" "$lang" "$url"
    fi
done
printf '\n%-22s %-44s\n' "frontend" "url"
printf '%-22s %-44s\n' "----------------------" "--------------------------------------------"
for entry in "${FRONTENDS[@]}"; do
    IFS='|' read -r name dir url <<< "$entry"
    pidf="${PIDS_DIR}/${name}.pid"
    if [ -f "$pidf" ] && kill -0 "$(cat "$pidf")" 2>/dev/null; then
        printf "${GREEN}✓${NC} %-20s %s\n" "$name" "$url"
    else
        printf "${RED}✗${NC} %-20s %s\n" "$name" "$url"
    fi
done

printf "\n"
ok "dev stack up. tail logs with: tail -f ${LOG_DIR}/<service>.log"
ok "stop everything with: scripts/stop-all.sh"
