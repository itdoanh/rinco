#!/bin/bash
set -e

# ===========================================
# RINCO Database Migration Script
# ===========================================

echo "🔄 Running database migrations..."

# Navigate to project root
cd "$(dirname "$0")/.."

# Check if goose is installed
if ! command -v goose &> /dev/null; then
    echo "📦 Installing goose..."
    go install github.com/pressly/goose/v3@latest
    export PATH=$PATH:$(go env GOPATH)/bin
fi

# Migration function
run_migration() {
    local service=$1
    local db_url=$2
    local migrations_dir=$3

    if [ -d "$migrations_dir" ] && [ "$(ls -A $migrations_dir/*.sql 2>/dev/null)" ]; then
        echo "Migrating $service..."
        cd "$migrations_dir"
        goose postgres "$db_url" up
        cd ../..
    else
        echo "No migrations found for $service"
    fi
}

# Auth Service
if [ -d "services/auth-service" ]; then
    run_migration \
        "auth-service" \
        "${AUTH_DATABASE_URL:-postgres://postgres:password@localhost:5432/auth}" \
        "services/auth-service/migrations"
fi

# Tenant Service
if [ -d "services/tenant-service" ]; then
    run_migration \
        "tenant-service" \
        "${TENANT_DATABASE_URL:-postgres://postgres:password@localhost:5432/tenant}" \
        "services/tenant-service/migrations"
fi

# Lead Service
if [ -d "services/lead-service" ]; then
    run_migration \
        "lead-service" \
        "${LEAD_DATABASE_URL:-postgres://postgres:password@localhost:5432/lead}" \
        "services/lead-service/migrations"
fi

# CRM Service
if [ -d "services/crm-service" ]; then
    run_migration \
        "crm-service" \
        "${CRM_DATABASE_URL:-postgres://postgres:password@localhost:5432/crm}" \
        "services/crm-service/migrations"
fi

# Landing Service
if [ -d "services/landing-service" ]; then
    run_migration \
        "landing-service" \
        "${LANDING_DATABASE_URL:-postgres://postgres:password@localhost:5432/landing}" \
        "services/landing-service/migrations"
fi

echo "✅ Migrations completed!"
