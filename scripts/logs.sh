#!/usr/bin/env bash
# ============================================================
# RINCO View Logs
# ============================================================
# Tails logs from infrastructure or microservice containers
# ============================================================
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
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

# All services list
ALL_SERVICES=(
    # Infrastructure
    "postgres" "pgbouncer" "scylla" "scylla-init" "clickhouse"
    "mongodb" "valkey" "meilisearch" "qdrant" "minio" "minio-init"
    "nats" "nats-init" "prometheus" "alertmanager" "grafana"
    "loki" "promtail" "tempo" "jaeger" "cadvisor" "node-exporter"
    "otel-collector" "traefik" "mailhog"
    # Microservices
    "auth-service" "tenant-service" "crm-service" "dynamic-model-service"
    "lead-service" "landing-service" "email-service" "notification-service"
    "observability-service" "billing-service" "search-service" "analytics-service"
    "meta-capi-service" "chat-engine" "webrtc-sfu" "recording-service"
    "lead-scoring" "ai-sre" "rag-chatbot" "stt-service"
    "landing-frontend" "admin-portal-frontend" "tenant-site-frontend" "meeting-ui-frontend"
)

# Print usage
usage() {
    echo "Usage: $0 [service] [options]"
    echo ""
    echo "View logs from RINCO containers"
    echo ""
    echo "Arguments:"
    echo "  service    Service name (optional, defaults to all)"
    echo ""
    echo "Options:"
    echo "  -f, --follow    Follow log output (tail -f)"
    echo "  -n, --lines     Number of lines to show (default: 100)"
    echo "  --since         Show logs since timestamp (e.g., '1h', '30m')"
    echo "  --timestamps    Show timestamps"
    echo "  --color         Force color output"
    echo "  --no-color      Disable color output"
    echo "  --infra         Filter infrastructure services only"
    echo "  --services      Filter microservice only"
    echo "  --list          List available services"
    echo "  --help, -h      Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                           # View all logs"
    echo "  $0 auth-service              # View auth-service logs"
    echo "  $0 -f auth-service          # Follow auth-service logs"
    echo "  $0 --infra                   # View infrastructure logs"
    echo "  $0 --services                # View microservice logs"
    echo "  $0 postgres -n 500           # Last 500 lines of postgres"
}

# List services
list_services() {
    echo "Available services:"
    echo ""
    echo -e "${BLUE}Infrastructure:${NC}"
    echo "  postgres, pgbouncer, scylla, clickhouse, mongodb, valkey"
    echo "  meilisearch, qdrant, minio, nats"
    echo "  prometheus, grafana, loki, tempo, jaeger"
    echo "  alertmanager, cadvisor, node-exporter, otel-collector"
    echo "  traefik, mailhog"
    echo ""
    echo -e "${BLUE}Microservices:${NC}"
    echo "  auth-service, tenant-service, crm-service"
    echo "  dynamic-model-service, lead-service, landing-service"
    echo "  email-service, notification-service"
    echo "  observability-service, billing-service"
    echo "  search-service, analytics-service, meta-capi-service"
    echo "  chat-engine, webrtc-sfu, recording-service"
    echo "  lead-scoring, ai-sre, rag-chatbot, stt-service"
    echo "  landing-frontend, admin-portal-frontend"
    echo "  tenant-site-frontend, meeting-ui-frontend"
}

# Check if service is infra
is_infra_service() {
    local service="$1"
    local infra_services=("postgres" "pgbouncer" "scylla" "scylla-init" "clickhouse"
        "mongodb" "valkey" "meilisearch" "qdrant" "minio" "minio-init"
        "nats" "nats-init" "prometheus" "alertmanager" "grafana"
        "loki" "promtail" "tempo" "jaeger" "cadvisor" "node-exporter"
        "otel-collector" "traefik" "mailhog")
    
    for s in "${infra_services[@]}"; do
        if [[ "$s" == "$service" ]]; then
            return 0
        fi
    done
    return 1
}

# Get compose file for service
get_compose_file() {
    local service="$1"
    if is_infra_service "$service"; then
        echo "docker-compose.yml"
    else
        echo "docker-compose.services.yml"
    fi
}

# Main
main() {
    local service=""
    local follow=false
    local lines=100
    local since=""
    local timestamps=false
    local color_mode="auto"
    local filter="all"

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -f|--follow)
                follow=true
                shift
                ;;
            -n|--lines)
                lines="$2"
                shift 2
                ;;
            --since)
                since="$2"
                shift 2
                ;;
            --timestamps)
                timestamps=true
                shift
                ;;
            --color)
                color_mode="always"
                shift
                ;;
            --no-color)
                color_mode="never"
                shift
                ;;
            --infra)
                filter="infra"
                shift
                ;;
            --services)
                filter="services"
                shift
                ;;
            --list)
                list_services
                exit 0
                ;;
            --help|-h)
                usage
                exit 0
                ;;
            -*)
                echo "Unknown option: $1"
                usage
                exit 1
                ;;
            *)
                if [[ -z "$service" ]]; then
                    service="$1"
                else
                    echo "Too many arguments"
                    usage
                    exit 1
                fi
                shift
                ;;
        esac
    done

    cd "$INFRA_DIR"

    # Build docker logs command
    local logs_cmd="docker logs"
    
    if $follow; then
        logs_cmd="$logs_cmd -f"
    fi
    
    if [[ -n "$since" ]]; then
        logs_cmd="$logs_cmd --since $since"
    else
        logs_cmd="$logs_cmd --tail $lines"
    fi
    
    if $timestamps; then
        logs_cmd="$logs_cmd -t"
    fi

    # Color output based on container name
    local color_args=""
    if [[ "$color_mode" == "always" ]] || [[ "$color_mode" == "auto" ]]; then
        if [[ -t 1 ]]; then
            color_args="--tail $lines"
        fi
    fi

    # If specific service requested
    if [[ -n "$service" ]]; then
        # Validate service exists
        local found=false
        for s in "${ALL_SERVICES[@]}"; do
            if [[ "$s" == "$service" ]]; then
                found=true
                break
            fi
        done

        if ! $found; then
            echo -e "${RED}Unknown service: $service${NC}"
            echo ""
            list_services
            exit 1
        fi

        # Try infra compose first, then services compose
        local container_name="rinco-$service"
        
        # Build the command
        local cmd="docker logs"
        if $follow; then
            cmd="$cmd -f"
        fi
        if [[ -n "$since" ]]; then
            cmd="$cmd --since $since"
        else
            cmd="$cmd --tail $lines"
        fi
        if $timestamps; then
            cmd="$cmd -t"
        fi
        
        # For docker compose logs
        local compose_file
        compose_file=$(get_compose_file "$service")
        local compose_file_name
        compose_file_name=$(basename "$compose_file")

        if $follow; then
            echo -e "${BLUE}Following logs: ${CYAN}$service${NC}"
            echo "Container: $container_name"
            echo ""
            $COMPOSE_CMD -f "$compose_file_name" logs -f "$service"
        else
            echo -e "${BLUE}Last ${lines} lines from: ${CYAN}$service${NC}"
            echo ""
            $COMPOSE_CMD -f "$compose_file_name" logs --tail="$lines" "$service"
        fi
    else
        # Show all logs (filtered by category)
        echo -e "${BLUE}RINCO Logs${NC}"
        echo ""
        
        if [[ "$filter" == "infra" ]]; then
            echo -e "${YELLOW}Showing infrastructure logs...${NC}"
            $COMPOSE_CMD -f docker-compose.yml logs --tail="$lines" --timestamps
        elif [[ "$filter" == "services" ]]; then
            echo -e "${YELLOW}Showing microservice logs...${NC}"
            $COMPOSE_CMD -f docker-compose.services.yml logs --tail="$lines" --timestamps
        else
            echo -e "${YELLOW}Showing all logs...${NC}"
            echo ""
            echo -e "${BLUE}=== Infrastructure ===${NC}"
            $COMPOSE_CMD -f docker-compose.yml logs --tail=50 --timestamps 2>&1 | head -100
            echo ""
            echo -e "${BLUE}=== Microservices ===${NC}"
            $COMPOSE_CMD -f docker-compose.services.yml logs --tail=50 --timestamps 2>&1 | head -100
        fi
    fi
}

main "$@"
