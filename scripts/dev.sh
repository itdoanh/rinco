#!/bin/bash
set -e

# ===========================================
# RINCO Development Environment Setup Script
# ===========================================

echo "🚀 Starting RINCO Development Environment..."

# Check if Docker is running
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

if ! docker info &> /dev/null; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

# Use docker compose if available, otherwise docker-compose
if docker compose version &> /dev/null; then
    COMPOSE_CMD="docker compose"
else
    COMPOSE_CMD="docker-compose"
fi

# Navigate to project root
cd "$(dirname "$0")/.."

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "📝 Creating .env file from .env.example..."
    if [ -f .env.example ]; then
        cp .env.example .env
        echo "⚠️  Please update .env file with your configuration."
    else
        echo "⚠️  No .env.example found. Creating default .env..."
        cat > .env << 'EOF'
# Development Environment Variables
NODE_ENV=development

# Auth Service
AUTH_HTTP_ADDR=:8081
AUTH_DATABASE_URL=postgres://postgres:password@localhost:5432/auth
AUTH_VALKEY_URL=redis://localhost:6379

# Tenant Service
TENANT_HTTP_ADDR=:8082
TENANT_DATABASE_URL=postgres://postgres:password@localhost:5432/tenant

# Lead Service
LEAD_HTTP_ADDR=:8083
LEAD_DATABASE_URL=postgres://postgres:password@localhost:5432/lead

# CRM Service
CRM_HTTP_ADDR=:8084
CRM_DATABASE_URL=postgres://postgres:password@localhost:5432/crm

# Landing Service
LANDING_HTTP_ADDR=:8085
LANDING_DATABASE_URL=postgres://postgres:password@localhost:5432/landing

# Frontend
NEXT_PUBLIC_API_URL=http://localhost:8080
EOF
    fi
fi

# Start infrastructure services
echo "🔧 Starting infrastructure services (Postgres, Redis, NATS)..."
$COMPOSE_CMD -f infra/docker-compose.services.yml up -d

# Wait for services to be ready
echo "⏳ Waiting for services to be ready..."
sleep 10

# Check if services are healthy
for service in postgres redis nats; do
    echo "Checking $service..."
    until $COMPOSE_CMD -f infra/docker-compose.services.yml exec -T $service ping -c 1 &> /dev/null 2>&1 || nc -z localhost $(docker inspect --format='{{range $p, $c := .NetworkSettings.Ports}}{{range $k, $v := $c}}{{index $v "HostPort"}}{{end}}{{end}}' $($COMPOSE_CMD -f infra/docker-compose.services.yml ps -q $service) 2>/dev/null); do
        sleep 2
    done
    echo "✓ $service is ready"
done

# Run migrations
echo "🔄 Running database migrations..."
./scripts/migrate.sh

# Seed data if needed
echo "📊 Seeding database..."
$COMPOSE_CMD -f infra/docker-compose.services.yml exec -T postgres psql -U postgres -d auth -c "SELECT 1" &> /dev/null && echo "✓ Database seeded"

# Start all services
echo "🚀 Starting all services..."
$COMPOSE_CMD -f infra/docker-compose.services.yml up -d

# Start frontend dev servers
echo "🎨 Starting frontend development servers..."

# Start landing
cd frontend/landing
npm run dev &
LANDING_PID=$!
cd ../..

# Start admin portal
cd frontend/admin-portal
npm run dev &
ADMIN_PID=$!
cd ../..

echo ""
echo "✅ Development environment is ready!"
echo ""
echo "Services:"
echo "  - Landing Page:  http://localhost:3000"
echo "  - Admin Portal:  http://localhost:3001"
echo "  - Auth Service:  http://localhost:8081"
echo "  - Postgres:      localhost:5432"
echo "  - Redis:         localhost:6379"
echo ""
echo "Press Ctrl+C to stop all services."

# Wait for all background processes
wait
