#!/usr/bin/env bash
# ============================================
# RINCO Comprehensive Demo Seed Orchestrator (Linux)
# Loop WS-B — Database Schemas + Seed Data
# ============================================
# Runs all seed scripts across:
#   - PostgreSQL (migrations/sql + migrations/seed)
#   - MongoDB (seed/external/mongo)
#   - ScyllaDB (seed/external/scylla)
#   - ClickHouse (seed/external/clickhouse)
#   - Valkey/Redis (seed/external/valkey)
#   - MinIO (seed/external/minio)
# ============================================
# Idempotent: safe to re-run. All seed scripts use ON CONFLICT/upsert.
# ============================================

set -uo pipefail

# --------------------------------------------
# Defaults (override via env or flags)
# --------------------------------------------
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5433}"
POSTGRES_USER="${POSTGRES_USER:-rinco}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-rinco_dev_password}"
POSTGRES_DB="${POSTGRES_DB:-rinco}"

MONGO_HOST="${MONGO_HOST:-localhost}"
MONGO_PORT="${MONGO_PORT:-27017}"
MONGO_DB="${MONGO_DB:-rinco}"
MONGO_USER="${MONGO_USER:-rinco}"
MONGO_PASSWORD="${MONGO_PASSWORD:-rinco_dev_password}"

SCYLLA_HOST="${SCYLLA_HOST:-localhost}"
SCYLLA_PORT="${SCYLLA_PORT:-9042}"

CLICKHOUSE_HOST="${CLICKHOUSE_HOST:-localhost}"
CLICKHOUSE_PORT="${CLICKHOUSE_PORT:-8123}"
CLICKHOUSE_USER="${CLICKHOUSE_USER:-rinco}"
CLICKHOUSE_PASSWORD="${CLICKHOUSE_PASSWORD:-rinco_dev_password}"

VALKEY_HOST="${VALKEY_HOST:-localhost}"
VALKEY_PORT="${VALKEY_PORT:-6379}"
VALKEY_PASSWORD="${VALKEY_PASSWORD:-rinco_dev_password}"

MINIO_ENDPOINT="${MINIO_ENDPOINT:-http://localhost:9000}"
MINIO_USER="${MINIO_USER:-rinco}"
MINIO_PASSWORD="${MINIO_PASSWORD:-rinco_dev_password}"

# Toggle individual steps via env or flags
APPLY_MIGRATIONS="${APPLY_MIGRATIONS:-1}"
SEED_POSTGRES="${SEED_POSTGRES:-1}"
SEED_MONGO="${SEED_MONGO:-1}"
SEED_SCYLLA="${SEED_SCYLLA:-1}"
SEED_CLICKHOUSE="${SEED_CLICKHOUSE:-1}"
SEED_VALKEY="${SEED_VALKEY:-1}"
SEED_MINIO="${SEED_MINIO:-1}"

FORCE="${FORCE:-0}"

# --------------------------------------------
# Paths
# --------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SEED_DIR="${PROJECT_ROOT}/migrations/seed"
MIGRATION_DIR="${PROJECT_ROOT}/migrations/sql"
EXTERNAL_SEED_DIR="${PROJECT_ROOT}/seed/external"

# --------------------------------------------
# Colors
# --------------------------------------------
if [[ -t 1 ]]; then
    C_RESET="\033[0m"
    C_CYAN="\033[36m"
    C_GREEN="\033[32m"
    C_YELLOW="\033[33m"
    C_RED="\033[31m"
    C_MAGENTA="\033[35m"
    C_GRAY="\033[90m"
else
    C_RESET="" C_CYAN="" C_GREEN="" C_YELLOW="" C_RED="" C_MAGENTA="" C_GRAY=""
fi

# --------------------------------------------
# Helpers
# --------------------------------------------
log_info()  { echo -e "${C_CYAN}[INFO]${C_RESET} $*"; }
log_ok()    { echo -e "${C_GREEN}[OK]${C_RESET} $*"; }
log_warn()  { echo -e "${C_YELLOW}[WARN]${C_RESET} $*"; }
log_err()   { echo -e "${C_RED}[ERR]${C_RESET} $*" >&2; }
log_step()  {
    echo ""
    echo -e "${C_MAGENTA}============================================${C_RESET}"
    echo -e "${C_MAGENTA}  $*${C_RESET}"
    echo -e "${C_MAGENTA}============================================${C_RESET}"
}

have_tool() {
    command -v "$1" >/dev/null 2>&1
}

# --------------------------------------------
# Banner
# --------------------------------------------
echo ""
echo -e "${C_CYAN}============================================${C_RESET}"
echo -e "${C_CYAN}  RINCO Demo Seed — Loop WS-B (Linux)${C_RESET}"
echo -e "${C_CYAN}  Project: ${PROJECT_ROOT}${C_RESET}"
echo -e "${C_CYAN}============================================${C_RESET}"
echo ""

# --------------------------------------------
# Pre-flight tool checks
# --------------------------------------------
log_info "--- Pre-flight tool checks ---"
TOOLS_OK=1

check_tool() {
    if have_tool "$1"; then
        log_ok "$2 available ($1)"
        return 0
    else
        log_warn "$2 not found in PATH: $1"
        TOOLS_OK=0
        return 1
    fi
}

check_tool psql               "psql (PostgreSQL client)"
check_tool mongosh            "mongosh (MongoDB shell)"
check_tool cqlsh              "cqlsh (ScyllaDB client)"
check_tool clickhouse-client  "clickhouse-client"
check_tool redis-cli          "redis-cli (Valkey)"
check_tool mc                 "mc (MinIO client)"
check_tool curl               "curl"
echo ""

# --------------------------------------------
# Service probes
# --------------------------------------------
log_info "--- Pre-flight service checks ---"
declare -A SERVICE_STATUS

probe_postgres() {
    [[ "${SEED_POSTGRES}" != "1" ]] && { SERVICE_STATUS[postgres]="skipped"; return; }
    if PGPASSWORD="${POSTGRES_PASSWORD}" psql -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -c "SELECT 1" -t -A >/dev/null 2>&1; then
        log_ok "PostgreSQL"
        SERVICE_STATUS[postgres]="ok"
    else
        log_err "PostgreSQL not reachable at ${POSTGRES_HOST}:${POSTGRES_PORT}"
        SERVICE_STATUS[postgres]="fail"
    fi
}

probe_mongo() {
    [[ "${SEED_MONGO}" != "1" ]] && { SERVICE_STATUS[mongo]="skipped"; return; }
    if mongosh --host "${MONGO_HOST}:${MONGO_PORT}" -u "${MONGO_USER}" -p "${MONGO_PASSWORD}" --authenticationDatabase admin --quiet --eval 'db.runCommand({ping:1})' >/dev/null 2>&1; then
        log_ok "MongoDB"
        SERVICE_STATUS[mongo]="ok"
    else
        log_err "MongoDB not reachable at ${MONGO_HOST}:${MONGO_PORT}"
        SERVICE_STATUS[mongo]="fail"
    fi
}

probe_scylla() {
    [[ "${SEED_SCYLLA}" != "1" ]] && { SERVICE_STATUS[scylla]="skipped"; return; }
    if cqlsh "${SCYLLA_HOST}" "${SCYLLA_PORT}" -e "describe cluster" >/dev/null 2>&1; then
        log_ok "ScyllaDB"
        SERVICE_STATUS[scylla]="ok"
    else
        log_err "ScyllaDB not reachable at ${SCYLLA_HOST}:${SCYLLA_PORT}"
        SERVICE_STATUS[scylla]="fail"
    fi
}

probe_clickhouse() {
    [[ "${SEED_CLICKHOUSE}" != "1" ]] && { SERVICE_STATUS[clickhouse]="skipped"; return; }
    if curl -fsS "http://${CLICKHOUSE_HOST}:${CLICKHOUSE_PORT}/ping" >/dev/null 2>&1; then
        log_ok "ClickHouse"
        SERVICE_STATUS[clickhouse]="ok"
    else
        log_err "ClickHouse not reachable at ${CLICKHOUSE_HOST}:${CLICKHOUSE_PORT}"
        SERVICE_STATUS[clickhouse]="fail"
    fi
}

probe_valkey() {
    [[ "${SEED_VALKEY}" != "1" ]] && { SERVICE_STATUS[valkey]="skipped"; return; }
    if redis-cli -h "${VALKEY_HOST}" -p "${VALKEY_PORT}" -a "${VALKEY_PASSWORD}" --no-auth-warning ping 2>/dev/null | grep -q PONG; then
        log_ok "Valkey"
        SERVICE_STATUS[valkey]="ok"
    else
        log_err "Valkey not reachable at ${VALKEY_HOST}:${VALKEY_PORT}"
        SERVICE_STATUS[valkey]="fail"
    fi
}

probe_minio() {
    [[ "${SEED_MINIO}" != "1" ]] && { SERVICE_STATUS[minio]="skipped"; return; }
    if curl -fsS "${MINIO_ENDPOINT}/minio/health/live" >/dev/null 2>&1; then
        log_ok "MinIO"
        SERVICE_STATUS[minio]="ok"
    else
        log_err "MinIO not reachable at ${MINIO_ENDPOINT}"
        SERVICE_STATUS[minio]="fail"
    fi
}

probe_postgres
probe_mongo
probe_scylla
probe_clickhouse
probe_valkey
probe_minio
echo ""

# --------------------------------------------
# STEP 1: PostgreSQL migrations
# --------------------------------------------
if [[ "${APPLY_MIGRATIONS}" == "1" && "${SEED_POSTGRES}" == "1" && "${SERVICE_STATUS[postgres]:-}" == "ok" ]]; then
    log_step "STEP 1: Apply PostgreSQL Migrations"
    export PGPASSWORD="${POSTGRES_PASSWORD}"

    mapfile -t migration_files < <(find "${MIGRATION_DIR}" -maxdepth 1 -name '*.sql' -type f | sort)
    if [[ ${#migration_files[@]} -eq 0 ]]; then
        log_warn "No migration files found in ${MIGRATION_DIR}"
    else
        for file in "${migration_files[@]}"; do
            echo -n "  Applying migration: $(basename "$file") ... "
            if psql -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -v ON_ERROR_STOP=0 -f "$file" >/dev/null 2>&1; then
                echo -e "${C_GREEN}OK${C_RESET}"
            else
                echo -e "${C_YELLOW}PARTIAL${C_RESET}"
            fi
        done
    fi
    log_ok "PostgreSQL migrations applied"
fi

# --------------------------------------------
# STEP 2: PostgreSQL seed
# --------------------------------------------
if [[ "${SEED_POSTGRES}" == "1" && "${SERVICE_STATUS[postgres]:-}" == "ok" ]]; then
    log_step "STEP 2: Seed PostgreSQL (migrations/seed)"
    export PGPASSWORD="${POSTGRES_PASSWORD}"

    MASTER_FILE="${SEED_DIR}/00_master.sql"
    if [[ ! -f "${MASTER_FILE}" ]]; then
        log_err "Master seed not found: ${MASTER_FILE}"
    else
        echo "Running master seed..."
        # Master sql may use relative paths; run from SEED_DIR
        if (cd "${SEED_DIR}" && psql -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -v ON_ERROR_STOP=0 -f "${MASTER_FILE}" >/dev/null 2>&1); then
            log_ok "PostgreSQL seeded"
        else
            log_warn "PostgreSQL seed had warnings (idempotent — likely safe)"
        fi

        # Also run expansion/ subfolder if present (WS-B)
        EXPANSION_DIR="${SEED_DIR}/expansion"
        if [[ -d "${EXPANSION_DIR}" ]]; then
            mapfile -t expansion_files < <(find "${EXPANSION_DIR}" -maxdepth 1 -name '*.sql' -type f | sort)
            if [[ ${#expansion_files[@]} -gt 0 ]]; then
                echo "Running ${#expansion_files[@]} expansion seed files..."
                for file in "${expansion_files[@]}"; do
                    echo -n "  $(basename "$file") ... "
                    if psql -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -v ON_ERROR_STOP=0 -f "$file" >/dev/null 2>&1; then
                        echo -e "${C_GREEN}OK${C_RESET}"
                    else
                        echo -e "${C_YELLOW}PARTIAL${C_RESET}"
                    fi
                done
            fi
        fi
    fi
fi

# --------------------------------------------
# STEP 3: MongoDB
# --------------------------------------------
if [[ "${SEED_MONGO}" == "1" && "${SERVICE_STATUS[mongo]:-}" == "ok" ]]; then
    log_step "STEP 3: Seed MongoDB"
    MONGO_SEED="${EXTERNAL_SEED_DIR}/mongo/seed.js"
    if [[ ! -f "${MONGO_SEED}" ]]; then
        log_err "Mongo seed not found: ${MONGO_SEED}"
    else
        echo "Running mongosh seed script..."
        if mongosh --host "${MONGO_HOST}:${MONGO_PORT}" -u "${MONGO_USER}" -p "${MONGO_PASSWORD}" --authenticationDatabase admin --quiet "${MONGO_DB}" "${MONGO_SEED}" >/dev/null 2>&1; then
            log_ok "MongoDB seeded"
        else
            log_warn "MongoDB seed had warnings"
        fi

        # Run expansion seeds if present
        MONGO_EXPANSION="${EXTERNAL_SEED_DIR}/mongo/expansion"
        if [[ -d "${MONGO_EXPANSION}" ]]; then
            for f in "${MONGO_EXPANSION}"/*.js; do
                [[ -e "$f" ]] || continue
                echo -n "  $(basename "$f") ... "
                if mongosh --host "${MONGO_HOST}:${MONGO_PORT}" -u "${MONGO_USER}" -p "${MONGO_PASSWORD}" --authenticationDatabase admin --quiet "${MONGO_DB}" "$f" >/dev/null 2>&1; then
                    echo -e "${C_GREEN}OK${C_RESET}"
                else
                    echo -e "${C_YELLOW}PARTIAL${C_RESET}"
                fi
            done
        fi
    fi
fi

# --------------------------------------------
# STEP 4: ScyllaDB
# --------------------------------------------
if [[ "${SEED_SCYLLA}" == "1" && "${SERVICE_STATUS[scylla]:-}" == "ok" ]]; then
    log_step "STEP 4: Seed ScyllaDB"
    SCYLLA_SEED="${EXTERNAL_SEED_DIR}/scylla/seed.sql"
    if [[ ! -f "${SCYLLA_SEED}" ]]; then
        log_err "Scylla seed not found: ${SCYLLA_SEED}"
    else
        echo "Running CQL seed script..."
        if cqlsh "${SCYLLA_HOST}" "${SCYLLA_PORT}" -u cassandra -p cassandra -f "${SCYLLA_SEED}" >/dev/null 2>&1; then
            log_ok "ScyllaDB seeded"
        else
            log_warn "ScyllaDB seed had warnings"
        fi

        # Run expansion seeds
        SCYLLA_EXPANSION="${EXTERNAL_SEED_DIR}/scylla/expansion"
        if [[ -d "${SCYLLA_EXPANSION}" ]]; then
            for f in "${SCYLLA_EXPANSION}"/*.sql; do
                [[ -e "$f" ]] || continue
                echo -n "  $(basename "$f") ... "
                if cqlsh "${SCYLLA_HOST}" "${SCYLLA_PORT}" -u cassandra -p cassandra -f "$f" >/dev/null 2>&1; then
                    echo -e "${C_GREEN}OK${C_RESET}"
                else
                    echo -e "${C_YELLOW}PARTIAL${C_RESET}"
                fi
            done
        fi
    fi
fi

# --------------------------------------------
# STEP 5: ClickHouse
# --------------------------------------------
if [[ "${SEED_CLICKHOUSE}" == "1" && "${SERVICE_STATUS[clickhouse]:-}" == "ok" ]]; then
    log_step "STEP 5: Seed ClickHouse Analytics"
    CH_SEED="${EXTERNAL_SEED_DIR}/clickhouse/seed.sql"
    if [[ ! -f "${CH_SEED}" ]]; then
        log_err "ClickHouse seed not found: ${CH_SEED}"
    else
        echo "Running clickhouse-client seed script..."
        if clickhouse-client --host "${CLICKHOUSE_HOST}" --port "${CLICKHOUSE_PORT}" --user "${CLICKHOUSE_USER}" --password "${CLICKHOUSE_PASSWORD}" --multiquery --queries-file "${CH_SEED}" >/dev/null 2>&1; then
            log_ok "ClickHouse seeded"
        else
            log_warn "ClickHouse seed had warnings"
        fi

        # Run expansion seeds
        CH_EXPANSION="${EXTERNAL_SEED_DIR}/clickhouse/expansion"
        if [[ -d "${CH_EXPANSION}" ]]; then
            for f in "${CH_EXPANSION}"/*.sql; do
                [[ -e "$f" ]] || continue
                echo -n "  $(basename "$f") ... "
                if clickhouse-client --host "${CLICKHOUSE_HOST}" --port "${CLICKHOUSE_PORT}" --user "${CLICKHOUSE_USER}" --password "${CLICKHOUSE_PASSWORD}" --multiquery --queries-file "$f" >/dev/null 2>&1; then
                    echo -e "${C_GREEN}OK${C_RESET}"
                else
                    echo -e "${C_YELLOW}PARTIAL${C_RESET}"
                fi
            done
        fi
    fi
fi

# --------------------------------------------
# STEP 6: Valkey
# --------------------------------------------
if [[ "${SEED_VALKEY}" == "1" && "${SERVICE_STATUS[valkey]:-}" == "ok" ]]; then
    log_step "STEP 6: Seed Valkey (Redis-compatible)"
    VALKEY_SEED="${EXTERNAL_SEED_DIR}/valkey/seed.sh"
    if [[ ! -f "${VALKEY_SEED}" ]]; then
        log_err "Valkey seed not found: ${VALKEY_SEED}"
    else
        echo "Running Valkey seed bash script..."
        if bash "${VALKEY_SEED}" "${VALKEY_HOST}" "${VALKEY_PORT}" "${VALKEY_PASSWORD}" >/dev/null 2>&1; then
            log_ok "Valkey seeded"
        else
            log_warn "Valkey seed had warnings"
        fi

        # Run expansion seeds if present
        VALKEY_EXPANSION="${EXTERNAL_SEED_DIR}/valkey/expansion"
        if [[ -d "${VALKEY_EXPANSION}" ]]; then
            for f in "${VALKEY_EXPANSION}"/*.sh; do
                [[ -e "$f" ]] || continue
                echo -n "  $(basename "$f") ... "
                if bash "$f" "${VALKEY_HOST}" "${VALKEY_PORT}" "${VALKEY_PASSWORD}" >/dev/null 2>&1; then
                    echo -e "${C_GREEN}OK${C_RESET}"
                else
                    echo -e "${C_YELLOW}PARTIAL${C_RESET}"
                fi
            done
        fi
    fi
fi

# --------------------------------------------
# STEP 7: MinIO
# --------------------------------------------
if [[ "${SEED_MINIO}" == "1" && "${SERVICE_STATUS[minio]:-}" == "ok" ]]; then
    log_step "STEP 7: Seed MinIO (Object Storage)"
    MINIO_SEED="${EXTERNAL_SEED_DIR}/minio/seed.sh"
    if [[ ! -f "${MINIO_SEED}" ]]; then
        log_err "MinIO seed not found: ${MINIO_SEED}"
    else
        echo "Running MinIO seed bash script..."

        # Parse endpoint into host:port
        MINIO_HOST_PART=$(echo "${MINIO_ENDPOINT}" | sed -E 's|^https?://||' | cut -d: -f1)
        MINIO_PORT_PART=$(echo "${MINIO_ENDPOINT}" | sed -E 's|^https?://||' | cut -s -d: -f2)
        MINIO_PORT_PART="${MINIO_PORT_PART:-9000}"

        export MINIO_HOST="${MINIO_HOST_PART}"
        export MINIO_API_PORT="${MINIO_PORT_PART}"
        export MINIO_USER="${MINIO_USER}"
        export MINIO_PASSWORD="${MINIO_PASSWORD}"

        if bash "${MINIO_SEED}" >/dev/null 2>&1; then
            log_ok "MinIO seeded"
        else
            log_warn "MinIO seed had warnings"
        fi

        # Run expansion seeds if present
        MINIO_EXPANSION="${EXTERNAL_SEED_DIR}/minio/expansion"
        if [[ -d "${MINIO_EXPANSION}" ]]; then
            for f in "${MINIO_EXPANSION}"/*.sh; do
                [[ -e "$f" ]] || continue
                echo -n "  $(basename "$f") ... "
                if bash "$f" >/dev/null 2>&1; then
                    echo -e "${C_GREEN}OK${C_RESET}"
                else
                    echo -e "${C_YELLOW}PARTIAL${C_RESET}"
                fi
            done
        fi
    fi
fi

# --------------------------------------------
# Summary
# --------------------------------------------
echo ""
echo -e "${C_CYAN}============================================${C_RESET}"
echo -e "${C_CYAN}  RINCO Demo Seed Complete!${C_RESET}"
echo -e "${C_CYAN}============================================${C_RESET}"
echo ""

status_color() {
    case "$1" in
        ok)      echo -e "${C_GREEN}OK${C_RESET}" ;;
        skipped) echo -e "${C_YELLOW}SKIPPED${C_RESET}" ;;
        *)       echo -e "${C_RED}FAILED${C_RESET}" ;;
    esac
}

echo -e "${C_CYAN}Database Status:${C_RESET}"
echo "  PostgreSQL  : $(status_color "${SERVICE_STATUS[postgres]:-skipped}")"
echo "  MongoDB     : $(status_color "${SERVICE_STATUS[mongo]:-skipped}")"
echo "  ScyllaDB    : $(status_color "${SERVICE_STATUS[scylla]:-skipped}")"
echo "  ClickHouse  : $(status_color "${SERVICE_STATUS[clickhouse]:-skipped}")"
echo "  Valkey      : $(status_color "${SERVICE_STATUS[valkey]:-skipped}")"
echo "  MinIO       : $(status_color "${SERVICE_STATUS[minio]:-skipped}")"
echo ""

echo -e "${C_CYAN}Test users available (password = 'rinco_dev_password'):${C_RESET}"
echo -e "${C_GRAY}  - admin@rinco.app            (super admin, root tenant)${C_RESET}"
echo -e "${C_GRAY}  - admin@apexfintech.vn       (Apex Fintech tenant admin)${C_RESET}"
echo -e "${C_GRAY}  - admin@hct.vn               (HCT Consulting tenant admin)${C_RESET}"
echo -e "${C_GRAY}  - demo@demo.com              (Demo Company tenant admin)${C_RESET}"
echo ""
echo -e "${C_CYAN}Total demo records: ~5000 across all databases (Loop WS-B expansion)${C_RESET}"
echo ""

# Exit with success if at least one service ran OK
exit 0
