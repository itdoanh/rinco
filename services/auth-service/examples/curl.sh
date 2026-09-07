#!/usr/bin/env bash
# auth-service — full curl smoke test
# Usage:
#   BASE=http://localhost:8081 EMAIL=alice@example.com PWD='correct horse battery staple' ./examples/curl.sh
set -euo pipefail

BASE="${BASE:-http://localhost:8081}"
EMAIL="${EMAIL:-alice@example.com}"
PWD="${PWD:-correct horse battery staple}"

say() { printf '\n\033[1;34m== %s ==\033[0m\n' "$*"; }

say "health"
curl -fsS "$BASE/healthz" | jq .

say "ready"
curl -fsS "$BASE/readyz" | jq .

say "metrics (first 8 lines)"
curl -fsS "$BASE/metrics" | head -8

say "register"
curl -fsS -X POST "$BASE/v1/auth/register" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PWD\",\"full_name\":\"Alice\"}" | jq .

say "login"
TOKENS=$(curl -fsS -X POST "$BASE/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PWD\"}")
ACCESS=$(echo "$TOKENS" | jq -r .access_token)
REFRESH=$(echo "$TOKENS" | jq -r .refresh_token)
echo "access: ${ACCESS:0:32}…"

say "me"
curl -fsS "$BASE/v1/auth/me" -H "Authorization: Bearer $ACCESS" | jq .

say "refresh"
curl -fsS -X POST "$BASE/v1/auth/refresh" \
  -H 'Content-Type: application/json' \
  -d "{\"refresh_token\":\"$REFRESH\"}" | jq .

say "api-keys create"
curl -fsS -X POST "$BASE/v1/auth/api-keys" \
  -H "Authorization: Bearer $ACCESS" \
  -H 'Content-Type: application/json' \
  -d '{"name":"smoke","scopes":["read"],"ttl_seconds":3600}' | jq .

say "rpc validate token"
curl -fsS -X POST "$BASE/internal/auth.v1.AuthService/ValidateToken" \
  -H 'Content-Type: application/json' \
  -d "{\"access_token\":\"$ACCESS\"}" | jq .

say "done"
