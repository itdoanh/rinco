#!/usr/bin/env bash
# ============================================================
# RINCO Health Check
# ============================================================
# Checks health status of all infrastructure and services
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

# Service definitions with health check commands
declare -A INFRA_SERVICES=(
    ["postgres"]="pg_isready -U rinco -d rinco"
    ["pgbouncer"]="echo 'SELECT 1;' | pgbouncer -U rinco postgres -d pgbouncer"
    ["scylla"]="cqlsh -e 'describe cluster'"
    ["clickhouse"]="wget --no-verbose --tries=1 --spider http://localhost:8123/ping"
    ["mongodb"]="mongosh --quiet --eval 'db.adminCommand({ping:1})'"
    ["valkey"]="valkey-cli ping"
    ["meilisearch"]="wget --no-verbose --tries=1 --spider http://localhost:7700/health"
    ["qdrant"]="curl -f http://localhost:6333/healthz"
    ["minio"]="curl -f http://localhost:9000/minio/health/live"
    ["nats"]="wget --no-verbose --tries=1 --spider http://localhost:8222/healthz"
    ["prometheus"]="wget --no-verbose --tries=1 --spider http://localhost:9090/-/healthy"
    ["grafana"]="wget --no-verbose --tries=1 --spider http://localhost:3000/api/health"
    ["loki"]="wget --no-verbose --tries=1 --spider http://localhost:3100/ready"
    ["tempo"]="wget --no-verbose --tries=1 --spider http://localhost:3200/ready"
    ["jaeger"]="wget --no-verbose --tries=1 --spider http://localhost:16686/"
    ["alertmanager"]="wget --no-verbose --tries=1 --spider http://localhost:9093/-/healthy"
    ["cadvisor"]="wget --no-verbose --tries=1 --spider http://localhost:8080/healthz"
    ["node-exporter"]="wget --no-verbose --tries=1 --spider http://localhost:9100/-/healthy"
    ["otel-collector"]="wget --no-verbose --tries=1 --spider http://localhost:13133/"
    ["traefik"]="wget --no-verbose --tries=1 --spider http://localhost:8080/api/overview"
    ["mailhog"]="wget --no-verbose --tries=1 --spider http://localhost:8025/"
)

declare -A SERVICE_SERVICES=(
    ["auth-service"]="http://localhost:8081/health"
    ["tenant-service"]="http://localhost:8082/health"
    ["crm-service"]="http://localhost:8083/health"
    ["dynamic-model-service"]="http://localhost:8084/health"
    ["lead-service"]="http://localhost:8085/health"
    ["landing-service"]="http://localhost:8086/health"
    ["email-service"]="http://localhost:8087/health"
    ["notification-service"]="http://localhost:8088/health"
    ["observability-service"]="http://localhost:8089/health"
    ["billing-service"]="http://localhost:8097/health"
    ["search-service"]="http://localhost:8094/health"
    ["analytics-service"]="http://localhost:8099/health"
    ["meta-capi-service"]="http://localhost:8100/health"
    ["chat-engine"]="http://localhost:8101/health"
    ["webrtc-sfu"]="http://localhost:8095/health"
    ["recording-service"]="http://localhost:8096/health"
    ["lead-scoring"]="http://localhost:8092/health"
    ["ai-sre"]="http://localhost:8090/health"
    ["rag-chatbot"]="http://localhost:8091/health"
    ["stt-service"]="http://localhost:8093/health"
    ["landing-frontend"]="http://localhost:3000"
    ["admin-portal-frontend"]="http://localhost:3001"
    ["tenant-site-frontend"]="http://localhost:3002"
    ["meeting-ui-frontend"]="http://localhost:3003"
)

# Port mappings
declare -A PORTS=(
    ["postgres"]="5432"
    ["pgbouncer"]="6432"
    ["scylla"]="9042"
    ["clickhouse"]="8123"
    ["mongodb"]="27017"
    ["valkey"]="6379"
    ["meilisearch"]="7700"
    ["qdrant"]="6333"
    ["minio"]="9000"
    ["nats"]="4222"
    ["prometheus"]="9090"
    ["grafana"]="3000"
    ["loki"]="3100"
    ["tempo"]="3200"
    ["jaeger"]="16686"
    ["alertmanager"]="9093"
    ["cadvisor"]="8080"
    ["node-exporter"]="9100"
    ["otel-collector"]="13133"
    ["traefik"]="8080"
    ["mailhog"]="8025"
    ["auth-service"]="8081"
    ["tenant-service"]="8082"
    ["crm-service"]="8083"
    ["dynamic-model-service"]="8084"
    ["lead-service"]="8085"
    ["landing-service"]="8086"
    ["email-service"]="8087"
    ["notification-service"]="8088"
    ["observability-service"]="8089"
    ["billing-service"]="8097"
    ["search-service"]="8094"
    ["analytics-service"]="8099"
    ["meta-capi-service"]="8100"
    ["chat-engine"]="8101"
    ["webrtc-sfu"]="8095"
    ["recording-service"]="8096"
    ["lead-scoring"]="8092"
    ["ai-sre"]="8090"
    ["rag-chatbot"]="8091"
    ["stt-service"]="8093"
    ["landing-frontend"]="3000"
    ["admin-portal-frontend"]="3001"
    ["tenant-site-frontend"]="3002"
    ["meeting-ui-frontend"]="3003"
)

# Status counters
total=0
running=0
stopped=0
unhealthy=0

# Check if container is running
is_container_running() {
    local container_name="rinco-$1"
    docker inspect -f '{{.State.Running}}' "$container_name" 2>/dev/null | grep -q "true"
}

# Get container status
get_container_status() {
    local container_name="rinco-$1"
    docker inspect -f '{{.State.Status}}' "$container_name" 2>/dev/null || echo "not_found"
}

# Check infra service health
check_infra_health() {
    local service="$1"
    local container_name="rinco-$service"
    
    if ! is_container_running "$service"; then
        echo -e "${RED}✗${NC}"
        return 1
    fi
    
    local health_check="${INFRA_SERVICES[$service]}"
    
    if docker exec "$container_name" sh -c "$health_check" &>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${YELLOW}⚠${NC}"
        return 2
    fi
}

# Check service health (HTTP)
check_service_health() {
    local service="$1"
    local port="${PORTS[$service]}"
    
    if curl -sf "http://localhost:$port/health" &>/dev/null || \
       curl -sf "http://localhost:$port" &>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        # Check if container is running
        if is_container_running "$service"; then
            echo -e "${YELLOW}⚠${NC}"
            return 2
        else
            echo -e "${RED}✗${NC}"
            return 1
        fi
    fi
}

# Print status line
print_status() {
    local service="$1"
    local status="$2"
    local port="${PORTS[$service]:-N/A}"
    
    printf "  %-30s %s  :%-5s\n" "$service" "$status" "$port"
}

# Main
main() {
    local filter="${1:-all}"
    
    echo ""
    echo "============================================"
    echo -e "${BLUE}  RINCO Health Check${NC}"
    echo "============================================"
    echo ""
    
    cd "$INFRA_DIR"
    
    # Summary header
    echo -e "${CYAN}Infrastructure Services:${NC}"
    echo "----------------------------------------"
    
    for service in "${!INFRA_SERVICES[@]}"; do
        local status
        if check_infra_health "$service"; then
            status="${GREEN}✓ Running${NC}"
            ((running++))
        else
            status="${RED}✗ Stopped${NC}"
            ((stopped++))
        fi
        print_status "$service" "$status"
        ((total++))
    done
    
    echo ""
    echo -e "${CYAN}Microservices:${NC}"
    echo "----------------------------------------"
    
    for service in "${!SERVICE_SERVICES[@]}"; do
        local status
        if check_service_health "$service"; then
            status="${GREEN}✓ Healthy${NC}"
            ((running++))
        else
            if is_container_running "$service"; then
                status="${YELLOW}⚠ Unhealthy${NC}"
                ((unhealthy++))
            else
                status="${RED}✗ Stopped${NC}"
                ((stopped++))
            fi
        fi
        print_status "$service" "$status"
        ((total++))
    done
    
    echo ""
    echo "============================================"
    echo -e "${BLUE}  Summary${NC}"
    echo "============================================"
    echo ""
    echo -e "  Total Services:    $total"
    echo -e "  ${GREEN}Healthy/Running: $running${NC}"
    echo -e "  ${YELLOW}Unhealthy:        $unhealthy${NC}"
    echo -e "  ${RED}Stopped:          $stopped${NC}"
    echo ""
    
    # Show docker compose status
    echo ""
    echo -e "${CYAN}Docker Compose Status:${NC}"
    echo "----------------------------------------"
    echo -e "${BLUE}Infrastructure:${NC}"
    $COMPOSE_CMD -f docker-compose.yml ps 2>/dev/null | head -30 || echo "  No infrastructure containers running"
    echo ""
    echo -e "${BLUE}Microservices:${NC}"
    $COMPOSE_CMD -f docker-compose.services.yml ps 2>/dev/null | head -30 || echo "  No service containers running"
    echo ""
    
    # Exit code based on status
    if [[ $stopped -gt 0 ]]; then
        exit 1
    elif [[ $unhealthy -gt 0 ]]; then
        exit 2
    else
        exit 0
    fi
}

main "$@"
