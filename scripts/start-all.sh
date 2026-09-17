#!/usr/bin/env bash
# ============================================================
# RINCO Start All Services
# ============================================================
# Starts:
#   1. Infrastructure (databases, cache, message broker, storage)
#   2. Microservices (all 21 services)
# ============================================================
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
INFRA_DIR="${PROJECT_ROOT}/infra"

# Compose files
INFRA_COMPOSE="${INFRA_DIR}/docker-compose.yml"
SERVICES_COMPOSE="${INFRA_DIR}/docker-compose.services.yml"

# Determine docker compose command
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

echo_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

echo_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

echo_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    echo_step "Checking prerequisites..."

    if ! command -v docker &>/dev/null; then
        echo_error "Docker is not installed. Please install Docker first."
        exit 1
    fi

    if ! docker info &>/dev/null; then
        echo_error "Docker is not running. Please start Docker first."
        exit 1
    fi

    echo_success "Docker is running"
}

# Create network if not exists
setup_network() {
    echo_step "Setting up Docker network..."
    if ! docker network inspect rinco-network &>/dev/null; then
        docker network create --driver bridge --subnet=172.25.0.0/16 rinco-network
        echo_success "Network rinco-network created"
    else
        echo_success "Network rinco-network already exists"
    fi
}

# Create .env file if not exists
setup_env() {
    echo_step "Setting up environment variables..."

    ENV_FILE="${PROJECT_ROOT}/.env"
    if [ ! -f "$ENV_FILE" ]; then
        echo_warn "No .env file found. Creating default .env..."
        cat > "$ENV_FILE" << 'EOF'
# RINCO Development Environment Variables
# =========================================

# Database Passwords
POSTGRES_PASSWORD=rinco_dev_password
CLICKHOUSE_PASSWORD=rinco_dev_password
VALKEY_PASSWORD=rinco_dev_password
MONGO_PASSWORD=rinco_dev_password
MINIO_ROOT_PASSWORD=rinco_dev_password

# Auth Service
PASETO_KEY_CURRENT=rinco_dev_paseto_key_change_in_prod_32bytes
PASETO_KEY_PREVIOUS=rinco_dev_paseto_key_change_in_prod_32bytes

# Admin
ADMIN_API_KEY=dev_admin_key_change_me

# Meilisearch
MEILI_MASTER_KEY=masterKey

# External Services (optional)
STRIPE_SECRET_KEY=
STRIPE_WEBHOOK_SECRET=
TELEGRAM_BOT_TOKEN=
SLACK_WEBHOOK_URL=
GITHUB_TOKEN=
EOF
        echo_success "Created .env file. Please update with production values."
    else
        echo_success "Using existing .env file"
    fi
}

# Start infrastructure services
start_infrastructure() {
    echo_step "Starting infrastructure services..."

    # Load .env file
    set -a
    source "${ENV_FILE}" 2>/dev/null || true
    set +a

    cd "$INFRA_DIR"
    $COMPOSE_CMD -f docker-compose.yml up -d

    echo_success "Infrastructure services started"
    echo ""
    echo "  Infrastructure services:"
    echo "    - PostgreSQL     : localhost:5432"
    echo "    - PgBouncer      : localhost:6432"
    echo "    - ScyllaDB       : localhost:9042"
    echo "    - ClickHouse     : localhost:8123, 9000"
    echo "    - MongoDB        : localhost:27017"
    echo "    - Valkey/Redis   : localhost:6379"
    echo "    - Meilisearch    : localhost:7700"
    echo "    - Qdrant         : localhost:6333, 6334"
    echo "    - MinIO          : localhost:9000, 9001"
    echo "    - NATS           : localhost:4222, 8222, 6222"
    echo ""
}

# Wait for infrastructure to be healthy
wait_for_infrastructure() {
    echo_step "Waiting for infrastructure to be healthy..."

    local max_wait=300
    local waited=0
    local interval=10

    while [ $waited -lt $max_wait ]; do
        # Check key services
        if docker exec rinco-postgres pg_isready -U rinco &>/dev/null && \
           docker exec rinco-valkey valkey-cli ping &>/dev/null && \
           docker exec rinco-nats nats-server --version &>/dev/null; then
            echo_success "Infrastructure is healthy"
            return 0
        fi

        echo "  Waiting for services... ($waited/$max_wait seconds)"
        sleep $interval
        waited=$((waited + interval))
    done

    echo_warn "Some infrastructure services may not be fully healthy yet"
    return 1
}

# Start microservices
start_services() {
    echo_step "Starting microservices..."

    cd "$INFRA_DIR"
    $COMPOSE_CMD -f docker-compose.services.yml up -d

    echo_success "Microservices started"
    echo ""
    echo "  Go Services:"
    echo "    - auth-service          : localhost:8081"
    echo "    - tenant-service        : localhost:8082"
    echo "    - crm-service           : localhost:8083"
    echo "    - dynamic-model-service : localhost:8084"
    echo "    - lead-service          : localhost:8085"
    echo "    - landing-service       : localhost:8086"
    echo "    - email-service         : localhost:8087"
    echo "    - notification-service  : localhost:8088"
    echo "    - observability-service : localhost:8089"
    echo "    - billing-service       : localhost:8097"
    echo "    - search-service        : localhost:8094"
    echo "    - analytics-service     : localhost:8099"
    echo "    - meta-capi-service     : localhost:8100"
    echo ""
    echo "  Rust Services:"
    echo "    - chat-engine    : localhost:8101"
    echo "    - webrtc-sfu     : localhost:8095"
    echo "    - recording-service : localhost:8096"
    echo ""
    echo "  Python Services:"
    echo "    - ai-sre        : localhost:8090"
    echo "    - rag-chatbot   : localhost:8091"
    echo "    - lead-scoring  : localhost:8092"
    echo "    - stt-service   : localhost:8093"
    echo ""
    echo "  Frontends:"
    echo "    - landing-frontend    : localhost:3000"
    echo "    - admin-portal       : localhost:3001"
    echo "    - tenant-site        : localhost:3002"
    echo "    - meeting-ui         : localhost:3003"
    echo ""
}

# Main
main() {
    echo ""
    echo "============================================"
    echo -e "${BLUE}  RINCO Start All Services${NC}"
    echo "============================================"
    echo ""

    check_prerequisites
    setup_network
    setup_env
    start_infrastructure
    wait_for_infrastructure
    start_services

    echo ""
    echo "============================================"
    echo -e "${GREEN}  All services started successfully!${NC}"
    echo "============================================"
    echo ""
    echo "Next steps:"
    echo "  - Check status: ./scripts/status.sh"
    echo "  - View logs:   ./scripts/logs.sh [service]"
    echo "  - Health check: ./scripts/health-check.sh"
    echo ""
}

main "$@"
