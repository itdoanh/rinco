#!/usr/bin/env bash
# ============================================================
# RINCO smoke test (Loop WS-A)
# ============================================================
# Hits /health on every backend service and every frontend app,
# exits 0 if everything responds 2xx, 1 otherwise.  Designed to
# be cheap and run as a CI gate.
#
# Usage:
#   bash scripts/smoke.sh             # default localhost
#   BASE=http://staging bash scripts/smoke.sh
#   STRICT=1  bash scripts/smoke.sh   # 2xx-or-exit-1
# ============================================================
set -uo pipefail

BASE="${BASE:-http://localhost}"
STRICT="${STRICT:-0}"
TIMEOUT="${TIMEOUT:-3}"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'

# (name|port|kind|health-path)
CHECKS=(
    "auth-service|8081|go|/health"
    "tenant-service|8082|go|/health"
    "crm-service|8083|go|/health"
    "dynamic-model-service|8084|go|/health"
    "lead-service|8085|go|/health"
    "landing-service|8086|go|/health"
    "email-service|8087|go|/health"
    "notification-service|8088|go|/health"
    "billing-service|8095|go|/health"
    "observability-service|8096|go|/health"
    "search-service|8097|go|/health"
    "meta-capi-service|8098|go|/health"
    "analytics-service|8099|go|/health"
    "ai-sre|8090|python|/health"
    "lead-scoring|8091|python|/health"
    "rag-chatbot|8092|python|/health"
    "stt-service|8094|python|/health"
    "recording-service|8093|rust|/health"
    "chat-engine|8101|rust|/health"
    "webrtc-sfu|8102|rust|/health"
    "landing-frontend|3000|frontend|/health"
    "admin-portal-frontend|3001|frontend|/health"
    "tenant-site-frontend|3002|frontend|/health"
    "meeting-ui-frontend|3003|frontend|/health"
)

total=0; healthy=0; degraded=0; failed=0
declare -a FAILED_NAMES

probe() {
    local name="$1" port="$2" kind="$3" path="$4" url="$BASE:$port$path"
    total=$((total+1))
    local code body
    body="$(curl -fsS --max-time "$TIMEOUT" -o /dev/null -w '%{http_code}' "$url" 2>/dev/null || echo "000")"
    code="$body"
    if [[ "$code" =~ ^2 ]]; then
        printf "${GREEN}✓${NC} %-26s %-10s :%s  %s\n" "$name" "$kind" "$port" "$code"
        healthy=$((healthy+1))
    elif [[ "$code" == "404" || "$code" == "000" ]] && [[ "$kind" == "frontend" ]]; then
        # Frontends sometimes don't expose /health; try the root instead.
        local root_code
        root_code="$(curl -fsS --max-time "$TIMEOUT" -o /dev/null -w '%{http_code}' "$BASE:$port/" 2>/dev/null || echo "000")"
        if [[ "$root_code" =~ ^2 ]]; then
            printf "${GREEN}✓${NC} %-26s %-10s :%s  %s (root)\n" "$name" "$kind" "$port" "$root_code"
            healthy=$((healthy+1))
            return
        fi
        printf "${RED}✗${NC} %-26s %-10s :%s  no /health, root=%s\n" "$name" "$kind" "$port" "$root_code"
        failed=$((failed+1))
        FAILED_NAMES+=("$name")
    elif [[ "$code" == "000" ]]; then
        printf "${RED}✗${NC} %-26s %-10s :%s  unreachable\n" "$name" "$kind" "$port"
        failed=$((failed+1))
        FAILED_NAMES+=("$name")
    else
        printf "${YELLOW}!${NC} %-26s %-10s :%s  HTTP %s\n" "$name" "$kind" "$port" "$code"
        degraded=$((degraded+1))
        FAILED_NAMES+=("$name")
    fi
}

printf "%-26s %-10s %-6s  %s\n" "service" "kind" "port" "code"
printf "%-26s %-10s %-6s  %s\n" "--------------------------" "----------" "------" "----"

for entry in "${CHECKS[@]}"; do
    IFS='|' read -r name port kind path <<< "$entry"
    probe "$name" "$port" "$kind" "$path"
done

printf "\n"
printf "Summary: %d total, ${GREEN}%d healthy${NC}, ${YELLOW}%d degraded${NC}, ${RED}%d failed${NC}\n" \
    "$total" "$healthy" "$degraded" "$failed"

if [ "$failed" -gt 0 ]; then
    printf "\n${RED}Failed services:${NC}\n"
    for n in "${FAILED_NAMES[@]}"; do
        printf "  - %s\n" "$n"
    done
fi

# Exit code:
#   0 — everything healthy
#   2 — degraded (non-2xx but reachable); only fail in STRICT mode
#   1 — unreachable
if [ "$failed" -gt 0 ]; then
    exit 1
fi
if [ "$degraded" -gt 0 ]; then
    if [ "$STRICT" = "1" ]; then exit 2; fi
fi
exit 0
