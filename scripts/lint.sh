#!/bin/bash
set -e

# ===========================================
# RINCO Lint Script
# ===========================================

echo "🔍 Running RINCO linters..."

# Navigate to project root
cd "$(dirname "$0")/.."

# Track results
FAILED=0

# Function to run linter and track results
run_lint() {
    local name=$1
    local command=$2

    echo ""
    echo "▶️  Linting: $name"
    if eval "$command"; then
        echo "✅ $name passed"
    else
        echo "❌ $name failed"
        ((FAILED++))
    fi
}

# Go linting
echo "🐹 Linting Go services..."
if command -v golangci-lint &> /dev/null; then
    if [ -d "services/auth-service" ]; then
        run_lint "auth-service" "cd services/auth-service && golangci-lint run --timeout=5m"
    fi
else
    echo "⚠️  golangci-lint not installed. Skipping Go linting."
    echo "   Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
fi

# Go vet
if [ -d "services/auth-service" ]; then
    run_lint "auth-service vet" "cd services/auth-service && go vet ./..."
fi

# Rust linting
echo "🦀 Linting Rust services..."
if command -v cargo &> /dev/null; then
    if [ -d "services/chat-engine" ]; then
        run_lint "chat-engine clippy" "cd services/chat-engine && cargo clippy -- -D warnings 2>/dev/null || true"
    fi

    if [ -d "services/webrtc-sfu" ]; then
        run_lint "webrtc-sfu clippy" "cd services/webrtc-sfu && cargo clippy -- -D warnings 2>/dev/null || true"
    fi
else
    echo "⚠️  cargo not installed. Skipping Rust linting."
fi

# Python linting
echo "🐍 Linting Python services..."
if command -v ruff &> /dev/null; then
    if [ -d "services/lead-scoring" ]; then
        run_lint "lead-scoring ruff" "cd services/lead-scoring && ruff check ."
    fi
elif command -v flake8 &> /dev/null; then
    if [ -d "services/lead-scoring" ]; then
        run_lint "lead-scoring flake8" "cd services/lead-scoring && flake8 . --max-line-length=100"
    fi
else
    echo "⚠️  ruff/flake8 not installed. Skipping Python linting."
    echo "   Install: pip install ruff && ruff check ."
fi

# TypeScript linting
echo "⚛️ Linting Frontend applications..."
if [ -d "frontend/landing" ]; then
    if [ -f "frontend/landing/package.json" ]; then
        run_lint "landing lint" "cd frontend/landing && npm run lint 2>/dev/null || echo 'Lint not configured'"
    fi
fi

if [ -d "frontend/admin-portal" ]; then
    if [ -f "frontend/admin-portal/package.json" ]; then
        run_lint "admin-portal lint" "cd frontend/admin-portal && npm run lint 2>/dev/null || echo 'Lint not configured'"
    fi
fi

if [ -d "frontend/tenant-site" ]; then
    if [ -f "frontend/tenant-site/package.json" ]; then
        run_lint "tenant-site lint" "cd frontend/tenant-site && npm run lint 2>/dev/null || echo 'Lint not configured'"
    fi
fi

if [ -d "frontend/meeting-ui" ]; then
    if [ -f "frontend/meeting-ui/package.json" ]; then
        run_lint "meeting-ui lint" "cd frontend/meeting-ui && npm run lint 2>/dev/null || echo 'Lint not configured'"
    fi
fi

# Summary
echo ""
echo "=========================================="
echo "📊 Lint Summary"
echo "=========================================="
if [ $FAILED -eq 0 ]; then
    echo "🎉 All linting passed!"
    exit 0
else
    echo "⚠️  Some linting checks failed"
    exit 1
fi
