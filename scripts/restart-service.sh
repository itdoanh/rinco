#!/usr/bin/env bash
# ============================================================
# RINCO Restart Service
# ============================================================
# Restarts a specific service (infrastructure or microservice)
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

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# List of known services
INFRA_SERVICES=(
    "postgres"
    "pgbouncer"
    "scylla"
    "clickhouse"
    "mongodb"
    "valkey"
    "meilisearch"
    "qdrant"
    "minio"
    "nats"
    "prometheus"
    "grafana"
    "loki"
    "tempo"
    "jaeger"
    "alertmanager"
    "cadvisor"
    "node-exporter"
    "otel-collector"
    "traefik"
    "mailhog"
)

SERVICE_SERVICES=(
    "auth-service"
    "tenant-service"
    "crm-service"
    "dynamic-model-service"
    "lead-service"
    "landing-service"
    "email-service"
    "notification-service"
    "observability-service"
    "billing-service"
    "search-service"
    "analytics-service"
    "meta-capi-service"
    "chat-engine"
    "webrtc-sfu"
    "recording-service"
    "lead-scoring"
    "ai-sre"
    "rag-chatbot"
    "stt-service"
    "landing-frontend"
    "admin-portal-frontend"
    "tenant-site-frontend"
    "meeting-ui-frontend"
)

# Print usage
usage() {
    echo "Usage: $0 <service> [--infra|--services]"
    echo ""
    echo "Restart a specific service"
    echo ""
    echo "Arguments:"
    echo "  service           Service name to restart"
    echo ""
    echo "Options:"
    echo "  --infra           Search in infrastructure services"
    echo "  --services        Search in microservice services"
    echo "  --help, -h       Show this help message"
    echo ""
    echo "Available Infrastructure Services:"
    printf "    %s\n" "${INFRA_SERVICES[@]}"
    echo ""
    echo "Available Microservices:"
    printf "    %s\n" "${SERVICE_SERVICES[@]}"
}

# Check if service exists
is_infra_service() {
    local service="$1"
    for s in "${INFRA_SERVICES[@]}"; do
        if [[ "$s" == "$service" ]]; then
            return 0
        fi
    done
    return 1
}

is_service_service() {
    local service="$1"
    for s in "${SERVICE_SERVICES[@]}"; do
        if [[ "$s" == "$service" ]]; then
            return 0
        fi
    done
    return 1
}

# Find compose file for service
find_compose_file() {
    local service="$1"
    local compose_file=""

    if is_infra_service "$service"; then
        compose_file="${INFRA_DIR}/docker-compose.yml"
    elif is_service_service "$service"; then
        compose_file="${INFRA_DIR}/docker-compose.services.yml"
    fi

    echo "$compose_file"
}

# Restart a service
restart_service() {
    local service="$1"
    local compose_file
    compose_file=$(find_compose_file "$service")

    if [[ -z "$compose_file" ]]; then
        echo_error "Unknown service: $service"
        echo "Run '$0 --help' to see available services"
        exit 1
    fi

    local compose_file_name
    compose_file_name=$(basename "$compose_file")

    echo_step "Restarting service: $service"
    echo "  Using compose file: $compose_file_name"

    cd "$INFRA_DIR"
    
    # Check if service is running
    if ! $COMPOSE_CMD -f "$compose_file_name" ps "$service" | grep -q "Up"; then
        echo "Service $service is not running. Starting it..."
        $COMPOSE_CMD -f "$compose_file_name" up -d "$service"
    else
        echo "Restarting service..."
        $COMPOSE_CMD -f "$compose_file_name" restart "$service"
    fi

    # Wait for service to be healthy
    echo_step "Waiting for service to be ready..."
    sleep 5

    # Check status
    if $COMPOSE_CMD -f "$compose_file_name" ps "$service" | grep -q "Up"; then
        echo_success "Service $service restarted successfully"
        
        # Show logs
        echo ""
        echo "Last 10 log lines:"
        $COMPOSE_CMD -f "$compose_file_name" logs --tail=10 "$service"
    else
        echo_error "Service $service may not have started correctly"
        echo ""
        echo "Full logs:"
        $COMPOSE_CMD -f "$compose_file_name" logs "$service"
        exit 1
    fi
}

# Main
main() {
    local service=""
    local search_infra=false
    local search_services=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --infra)
                search_infra=true
                shift
                ;;
            --services)
                search_services=true
                shift
                ;;
            --help|-h)
                usage
                exit 0
                ;;
            -*)
                echo_error "Unknown option: $1"
                usage
                exit 1
                ;;
            *)
                if [[ -z "$service" ]]; then
                    service="$1"
                else
                    echo_error "Too many arguments"
                    usage
                    exit 1
                fi
                shift
                ;;
        esac
    done

    # Default: search both
    if ! $search_infra && ! $search_services; then
        search_infra=true
        search_services=true
    fi

    # Validate service name
    if [[ -z "$service" ]]; then
        echo_error "Service name is required"
        usage
        exit 1
    fi

    # Validate service exists
    local found=false
    if $search_infra && is_infra_service "$service"; then
        found=true
    fi
    if $search_services && is_service_service "$service"; then
        found=true
    fi

    if ! $found; then
        echo_error "Unknown service: $service"
        echo ""
        echo "Available services:"
        if $search_infra; then
            echo "Infrastructure:"
            printf "  %s\n" "${INFRA_SERVICES[@]}"
        fi
        if $search_services; then
            echo "Microservices:"
            printf "  %s\n" "${SERVICE_SERVICES[@]}"
        fi
        exit 1
    fi

    # Restart the service
    restart_service "$service"
}

main "$@"
