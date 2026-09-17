#!/usr/bin/env bash
# run-integration-tests.sh — Convenience wrapper used by CI and by
# developers. Spins up the canonical local docker-compose stack, applies
# migrations + seed, then runs the integration test suite with coverage.
#
# Usage:
#   RINCO_NO_DOCKER=1 ./scripts/run-integration-tests.sh       # skip docker (use existing stack)
#   CHECK_COVERAGE_STRICT=1 ./scripts/run-integration-tests.sh # enforce coverage gate
#
# Requires: docker compose v2, go 1.23+.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="$ROOT_DIR/services/integration-tests"

if [ "${RINCO_NO_DOCKER:-}" != "1" ]; then
  echo "==> bringing up docker-compose stack (postgres + valkey + nats)"
  ( cd "$ROOT_DIR" && docker compose -f docker-compose.yml up -d )
  echo "==> waiting for postgres to accept connections"
  for i in $(seq 1 30); do
    if docker compose -f "$ROOT_DIR/docker-compose.yml" exec -T postgres \
       pg_isready -U rinco -d rinco_test 2>/dev/null; then
      break
    fi
    sleep 2
  done
fi

export RINCO_USE_EXTERNAL_STACK=true
export RINCO_TEST_POSTGRES_DSN="${RINCO_TEST_POSTGRES_DSN:-postgres://rinco:rinco@localhost:5432/rinco_test?sslmode=disable}"
export RINCO_TEST_VALKEY_ADDR="${RINCO_TEST_VALKEY_ADDR:-localhost:6379}"
export RINCO_TEST_NATS_URL="${RINCO_TEST_NATS_URL:-nats://localhost:4222}"

echo "==> running integration tests with coverage"
( cd "$TEST_DIR" && \
  go test -tags=integration -count=1 -race -timeout 10m \
           -coverprofile=coverage.out -covermode=atomic \
           -v ./... )

echo "==> generating HTML coverage report"
( cd "$TEST_DIR" && go tool cover -html=coverage.out -o coverage.html )

echo "==> coverage summary"
( cd "$TEST_DIR" && go tool cover -func=coverage.out | tail -1 )

bash "$ROOT_DIR/scripts/check-coverage.sh" "$TEST_DIR/coverage.out" "${COVERAGE_MIN:-80}"

echo "==> done. coverage.html is at $TEST_DIR/coverage.html"
