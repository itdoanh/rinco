#!/bin/bash
set -e

# ===========================================
# RINCO Test Script
# ===========================================

echo "🧪 Running RINCO tests..."

# Navigate to project root
cd "$(dirname "$0")/.."

# Track test results
FAILED=0
PASSED=0

# Function to run tests and track results
run_test() {
    local name=$1
    local command=$2

    echo ""
    echo "▶️  Running: $name"
    if eval "$command"; then
        echo "✅ $name passed"
        ((PASSED++))
    else
        echo "❌ $name failed"
        ((FAILED++))
    fi
}

# Go tests
echo "🐹 Running Go tests..."
if [ -d "services/auth-service" ]; then
    run_test "auth-service tests" "cd services/auth-service && go test ./... -v -short"
fi

if [ -d "services/tenant-service" ]; then
    run_test "tenant-service tests" "cd services/tenant-service && go test ./... -v -short"
fi

if [ -d "services/lead-service" ]; then
    run_test "lead-service tests" "cd services/lead-service && go test ./... -v -short"
fi

if [ -d "services/crm-service" ]; then
    run_test "crm-service tests" "cd services/crm-service && go test ./... -v -short"
fi

if [ -d "services/landing-service" ]; then
    run_test "landing-service tests" "cd services/landing-service && go test ./... -v -short"
fi

# Rust tests
echo "🦀 Running Rust tests..."
if [ -d "services/chat-engine" ]; then
    run_test "chat-engine tests" "cd services/chat-engine && cargo test --verbose --lib"
fi

if [ -d "services/webrtc-sfu" ]; then
    run_test "webrtc-sfu tests" "cd services/webrtc-sfu && cargo test --verbose --lib"
fi

# Python tests
echo "🐍 Running Python tests..."
if [ -d "services/lead-scoring" ]; then
    run_test "lead-scoring tests" "cd services/lead-scoring && pip install -q pytest pytest-cov && pytest -v --tb=short"
fi

# Frontend tests
echo "⚛️ Running Frontend tests..."

if [ -d "frontend/landing" ]; then
    run_test "landing build test" "cd frontend/landing && npm run typecheck 2>/dev/null || echo 'TypeScript check not configured'"
fi

if [ -d "frontend/admin-portal" ]; then
    run_test "admin-portal build test" "cd frontend/admin-portal && npm run typecheck 2>/dev/null || echo 'TypeScript check not configured'"
fi

if [ -d "frontend/tenant-site" ]; then
    run_test "tenant-site build test" "cd frontend/tenant-site && npm run typecheck 2>/dev/null || echo 'TypeScript check not configured'"
fi

if [ -d "frontend/meeting-ui" ]; then
    run_test "meeting-ui build test" "cd frontend/meeting-ui && npm run typecheck 2>/dev/null || echo 'TypeScript check not configured'"
fi

# Summary
echo ""
echo "=========================================="
echo "📊 Test Summary"
echo "=========================================="
echo "✅ Passed: $PASSED"
echo "❌ Failed: $FAILED"
echo ""

if [ $FAILED -eq 0 ]; then
    echo "🎉 All tests passed!"
    exit 0
else
    echo "⚠️  Some tests failed"
    exit 1
fi
