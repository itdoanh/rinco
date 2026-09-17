#!/usr/bin/env bash
# ============================================================
# RINCO Stop All Services
# ============================================================
# Stops all microservices and infrastructure gracefully
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

echo_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Parse arguments
STOP_INFRA=false
STOP_SERVICES=false
REMOVE_VOLUMES=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --infra)
            STOP_INFRA=true
            STOP_SERVICES=false
            shift
            ;;
        --services)
            STOP_SERVICES=true
            STOP_INFRA=false
            shift
            ;;
        --all)
            STOP_INFRA=true
            STOP_SERVICES=true
            shift
            ;;
        --volumes|-v)
            REMOVE_VOLUMES=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --infra      Stop only infrastructure services"
            echo "  --services    Stop only microservices"
            echo "  --all         Stop both infrastructure and services (default)"
            echo "  --volumes, -v Remove data volumes (DESTRUCTIVE)"
            echo "  --help, -h    Show this help message"
            exit 0
            ;;
        *)
            echo_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Default: stop everything
if ! $STOP_INFRA && ! $STOP_SERVICES; then
    STOP_INFRA=true
    STOP_SERVICES=true
fi

# Stop microservices
stop_services() {
    echo_step "Stopping microservices..."

    cd "$INFRA_DIR"
    
    if $REMOVE_VOLUMES; then
        $COMPOSE_CMD -f docker-compose.services.yml down -v --remove-orphans 2>/dev/null || true
    else
        $COMPOSE_CMD -f docker-compose.services.yml down --remove-orphans 2>/dev/null || true
    fi
    
    echo_success "Microservices stopped"
}

# Stop infrastructure
stop_infrastructure() {
    echo_step "Stopping infrastructure services..."

    cd "$INFRA_DIR"
    
    if $REMOVE_VOLUMES; then
        echo_warn "Removing data volumes (DESTRUCTIVE)..."
        $COMPOSE_CMD -f docker-compose.yml down -v --remove-orphans 2>/dev/null || true
    else
        $COMPOSE_CMD -f docker-compose.yml down --remove-orphans 2>/dev/null || true
    fi
    
    echo_success "Infrastructure stopped"
}

# Cleanup network
cleanup_network() {
    if $REMOVE_VOLUMES; then
        echo_step "Cleaning up network..."
        docker network rm rinco-network 2>/dev/null || true
        echo_success "Network cleaned up"
    fi
}

# Main
main() {
    echo ""
    echo "============================================"
    echo -e "${BLUE}  RINCO Stop All Services${NC}"
    echo "============================================"
    echo ""

    if $REMOVE_VOLUMES; then
        echo_warn "WARNING: This will remove all data volumes!"
        echo ""
    fi

    if $STOP_SERVICES; then
        stop_services
    fi

    if $STOP_INFRA; then
        stop_infrastructure
        cleanup_network
    fi

    echo ""
    echo "============================================"
    echo -e "${GREEN}  All services stopped${NC}"
    echo "============================================"
    echo ""
}

main "$@"
