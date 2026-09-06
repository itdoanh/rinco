.PHONY: help setup infra services migrate build test clean stop logs health

# Default target
help:
	@echo "==================================="
	@echo " RINCO Development Makefile"
	@echo "==================================="
	@echo ""
	@echo "Setup:"
	@echo "  make setup       - First time setup (checks tools, creates .env)"
	@echo "  make infra       - Start infrastructure (databases, observability)"
	@echo "  make services    - Start all microservices"
	@echo "  make stop        - Stop all services"
	@echo ""
	@echo "Development:"
	@echo "  make build       - Build Go services"
	@echo "  make test        - Run tests"
	@echo "  make lint        - Lint code"
	@echo "  make health      - Health check all services"
	@echo "  make logs        - Tail logs from all services"
	@echo ""
	@echo "Database:"
	@echo "  make migrate     - Run database migrations"
	@echo "  make seed        - Seed demo data"
	@echo ""
	@echo "Maintenance:"
	@echo "  make clean       - Clean up (stop + remove volumes)"
	@echo "  make reset       - Reset everything (clean + setup)"
	@echo ""

# Detect OS
ifeq ($(OS),Windows_NT)
	SETUP_SCRIPT := scripts\setup.ps1
	HEALTH_SCRIPT := scripts\health-check.ps1
	TEST_SCRIPT := scripts\test-lead-flow.ps1
	SHELL_CMD := powershell -ExecutionPolicy Bypass
else
	SETUP_SCRIPT := scripts/setup.sh
	HEALTH_SCRIPT := scripts/health-check.sh
	TEST_SCRIPT := scripts/test-lead-flow.sh
	SHELL_CMD := bash
endif

setup:
	$(SHELL_CMD) -File $(SETUP_SCRIPT)

infra:
	docker compose -f infra/docker-compose.yml up -d
	@echo "Waiting for databases..."
	@sleep 10

services: infra
	docker compose -f infra/docker-compose.services.yml up -d
	@echo "All services started"
	@$(SHELL_CMD) -File $(HEALTH_SCRIPT)

stop:
	docker compose -f infra/docker-compose.services.yml down
	docker compose -f infra/docker-compose.yml down

logs:
	docker compose -f infra/docker-compose.services.yml logs -f --tail=100

build:
	cd services/auth-service && go build -o bin/auth-service ./cmd
	cd services/tenant-service && go build -o bin/tenant-service ./cmd
	cd services/crm-service && go build -o bin/crm-service ./cmd
	cd services/dynamic-model-service && go build -o bin/dynamic-model-service ./cmd
	cd services/lead-service && go build -o bin/lead-service ./cmd
	cd services/landing-service && go build -o bin/landing-service ./cmd
	cd services/email-service && go build -o bin/email-service ./cmd
	cd services/notification-service && go build -o bin/notification-service ./cmd
	cd services/observability-service && go build -o bin/observability-service ./cmd
	@echo "All Go services built"

test:
	$(SHELL_CMD) -File $(TEST_SCRIPT)

health:
	$(SHELL_CMD) -File $(HEALTH_SCRIPT)

lint:
	cd services && for d in */cmd; do golangci-lint run $$d/.. 2>&1 || true; done

migrate:
	@echo "Running PostgreSQL migrations..."
	docker exec -i rinco-postgres psql -U rinco -d rinco < infra/postgres/init/01-init.sql
	@echo "Running ScyllaDB migrations..."
	docker exec -i rinco-scylla cqlsh -f infra/scylla/init.cql
	@echo "Running ClickHouse migrations..."
	docker exec -i rinco-clickhouse clickhouse-client --multiquery < infra/clickhouse/init.sql
	@echo "✓ All migrations applied"

seed:
	docker exec -i rinco-postgres psql -U rinco -d rinco < infra/postgres/init/01-init.sql
	@echo "✓ Demo data seeded"

clean:
	docker compose -f infra/docker-compose.services.yml down -v
	docker compose -f infra/docker-compose.yml down -v
	@echo "✓ Cleaned"

reset: clean setup
