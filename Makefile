# ===========================================
# RINCO Development Makefile
# ===========================================

.PHONY: help dev build test lint migrate deploy clean install check诗人

# Default target
help:
	@echo "RINCO - Development Commands"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  install    Install all dependencies"
	@echo "  dev        Start development environment"
	@echo "  build      Build all Docker images"
	@echo "  test       Run all tests"
	@echo "  lint       Run all linters"
	@echo "  migrate    Run database migrations"
	@echo "  deploy     Deploy to Kubernetes"
	@echo "  clean      Clean up containers and volumes"
	@echo "  check      Run all checks (lint + test)"

# Install dependencies
install:
	@echo "📦 Installing dependencies..."
	cd services/auth-service && go mod download
	cd services/tenant-service && go mod download
	cd services/lead-service && go mod download
	cd services/crm-service && go mod download
	cd services/landing-service && go mod download
	cd services/chat-engine && cargo fetch
	cd services/lead-scoring && pip install -r requirements.txt
	cd frontend/landing && npm install
	cd frontend/admin-portal && npm install
	cd frontend/tenant-site && npm install
	cd frontend/meeting-ui && npm install

# Development environment
dev:
	@echo "🚀 Starting development environment..."
	@chmod +x scripts/*.sh
	./scripts/dev.sh

# Build Docker images
build:
	@echo "🔨 Building Docker images..."
	@chmod +x scripts/*.sh
	./scripts/build.sh

# Run tests
test:
	@echo "🧪 Running tests..."
	@chmod +x scripts/*.sh
	./scripts/test.sh

# Run linters
lint:
	@echo "🔍 Running linters..."
	@chmod +x scripts/*.sh
	./scripts/lint.sh

# Run migrations
migrate:
	@echo "🔄 Running migrations..."
	@chmod +x scripts/*.sh
	./scripts/migrate.sh

# Deploy to Kubernetes
deploy:
	@echo "🚀 Deploying to Kubernetes..."
	@chmod +x scripts/*.sh
	./scripts/deploy.sh

# Clean up
clean:
	@echo "🧹 Cleaning up..."
	docker-compose -f infra/docker-compose.services.yml down -v 2>/dev/null || true
	docker-compose -f infra/docker-compose.yml down -v 2>/dev/null || true
	find . -type d -name node_modules -exec rm -rf {} + 2>/dev/null || true
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	find . -type d -name target -exec rm -rf {} + 2>/dev/null || true

# Run all checks
check: lint test

# Docker compose services
up:
	docker-compose -f infra/docker-compose.services.yml up -d

down:
	docker-compose -f infra/docker-compose.services.yml down

restart:
	docker-compose -f infra/docker-compose.services.yml restart

# Service-specific targets
go-build:
	cd services/auth-service && go build -o bin/auth-service ./cmd/server
	cd services/tenant-service && go build -o bin/tenant-service ./cmd/server
	cd services/lead-service && go build -o bin/lead-service ./cmd/server
	cd services/crm-service && go build -o bin/crm-service ./cmd/server
	cd services/landing-service && go build -o bin/landing-service ./cmd/server

rust-build:
	cd services/chat-engine && cargo build --release
	cd services/webrtc-sfu && cargo build --release

python-build:
	cd services/lead-scoring && pip install -r requirements.txt

frontend-build:
	cd frontend/landing && npm run build
	cd frontend/admin-portal && npm run build
	cd frontend/tenant-site && npm run build
	cd frontend/meeting-ui && npm run build

# Database operations
db-reset:
	@echo "⚠️  Resetting database..."
	docker-compose -f infra/docker-compose.services.yml exec -T postgres dropdb --username=postgres auth 2>/dev/null || true
	docker-compose -f infra/docker-compose.services.yml exec -T postgres createdb --username=postgres auth 2>/dev/null || true
	./scripts/migrate.sh

# Logs
logs:
	docker-compose -f infra/docker-compose.services.yml logs -f

logs-auth:
	docker-compose -f infra/docker-compose.services.yml logs -f auth-service

# Shell access
shell-postgres:
	docker-compose -f infra/docker-compose.services.yml exec postgres psql -U postgres

shell-redis:
	docker-compose -f infra/docker-compose.services.yml exec redis redis-cli

# Health check
health:
	@echo "🏥 Checking service health..."
	@curl -s http://localhost:8081/health || echo "auth-service: DOWN"
	@curl -s http://localhost:8082/health || echo "tenant-service: DOWN"
	@curl -s http://localhost:8083/health || echo "lead-service: DOWN"
	@curl -s http://localhost:8084/health || echo "crm-service: DOWN"
	@curl -s http://localhost:8085/health || echo "landing-service: DOWN"
