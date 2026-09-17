#!/usr/bin/env bash
# ============================================================
# RINCO Status
# ============================================================
# Quick overview of all running containers
# ============================================================
set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

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

# URLs for quick access
echo ""
echo "============================================"
echo -e "${BLUE}  RINCO Status${NC}"
echo "============================================"
echo ""

# Infrastructure URLs
echo -e "${CYAN}Infrastructure:${NC}"
echo "  PostgreSQL     : localhost:5432"
echo "  PgBouncer      : localhost:6432"
echo "  ScyllaDB       : localhost:9042"
echo "  ClickHouse     : localhost:8123"
echo "  MongoDB        : localhost:27017"
echo "  Valkey/Redis   : localhost:6379"
echo "  Meilisearch    : localhost:7700"
echo "  Qdrant         : localhost:6333"
echo "  MinIO          : localhost:9000 (console: 9001)"
echo "  NATS           : localhost:4222"
echo ""

# Observability URLs
echo -e "${CYAN}Observability:${NC}"
echo "  Prometheus     : localhost:9090"
echo "  Grafana        : localhost:3000 (admin/admin)"
echo "  Loki           : localhost:3100"
echo "  Tempo          : localhost:3200"
echo "  Jaeger         : localhost:16686"
echo "  cAdvisor       : localhost:8080"
echo "  Node Exporter  : localhost:9100"
echo "  OTEL Collector : localhost:4317"
echo ""

# Service URLs
echo -e "${CYAN}Services:${NC}"
echo "  auth-service         : localhost:8081"
echo "  tenant-service       : localhost:8082"
echo "  crm-service          : localhost:8083"
echo "  dynamic-model-service: localhost:8084"
echo "  lead-service         : localhost:8085"
echo "  landing-service      : localhost:8086"
echo "  email-service        : localhost:8087"
echo "  notification-service : localhost:8088"
echo "  observability-service: localhost:8089"
echo "  billing-service      : localhost:8097"
echo "  search-service       : localhost:8094"
echo "  analytics-service   : localhost:8099"
echo "  meta-capi-service   : localhost:8100"
echo "  chat-engine         : localhost:8101"
echo "  webrtc-sfu          : localhost:8095"
echo "  recording-service   : localhost:8096"
echo "  lead-scoring        : localhost:8092"
echo "  ai-sre              : localhost:8090"
echo "  rag-chatbot         : localhost:8091"
echo "  stt-service         : localhost:8093"
echo ""

# Frontend URLs
echo -e "${CYAN}Frontends:${NC}"
echo "  Landing Page   : localhost:3000"
echo "  Admin Portal   : localhost:3001"
echo "  Tenant Site    : localhost:3002"
echo "  Meeting UI     : localhost:3003"
echo ""

# Docker status
echo "============================================"
echo -e "${BLUE}  Container Status${NC}"
echo "============================================"
echo ""

echo -e "${CYAN}Infrastructure Containers:${NC}"
$COMPOSE_CMD -f "${INFRA_DIR}/docker-compose.yml" ps 2>/dev/null || echo "  No containers running"
echo ""

echo -e "${CYAN}Microservice Containers:${NC}"
$COMPOSE_CMD -f "${INFRA_DIR}/docker-compose.services.yml" ps 2>/dev/null || echo "  No containers running"
echo ""

# Resource usage
echo "============================================"
echo -e "${BLUE}  Quick Commands${NC}"
echo "============================================"
echo ""
echo "  ./scripts/start-all.sh      - Start all services"
echo "  ./scripts/stop-all.sh       - Stop all services"
echo "  ./scripts/restart-service.sh [name] - Restart specific service"
echo "  ./scripts/logs.sh [name]    - View logs"
echo "  ./scripts/health-check.sh   - Check health status"
echo ""
