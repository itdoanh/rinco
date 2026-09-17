#!/usr/bin/env bash
# ============================================
# RINCO Valkey Demo Seed (Loop 202)
# Usage:   bash valkey/seed.sh [host] [port] [password]
# Inserts sessions, rate-limit counters, presence data, AI cache.
# ============================================
set -e

VHOST=${1:-localhost}
VPORT=${2:-6379}
VPASS=${3:-rinco_dev_password}

REDIS_CMD="redis-cli -h ${VHOST} -p ${VPORT} -a ${VPASS} --no-auth-warning"

echo "=== Valkey demo seed — Loop 202 ==="
echo "Target: ${VHOST}:${VPORT}"

# ========================================
# Flush relevant db (be careful in prod!)
# ========================================
echo "[1/8] FLUSHDB (clear demo keyspace)..."
${REDIS_CMD} -n 0 FLUSHDB || true
${REDIS_CMD} -n 1 FLUSHDB || true

# ========================================
# Session tokens (db 0)
# ========================================
echo "[2/8] Sessions..."
NOW=$(date +%s)
EXPIRE_14_DAYS=$((NOW + 14*86400))
DEMO_USERS=(
  "admin@rinco.app|00000000-0000-0000-0000-000000000001|0:0:0:0:0:0:0:1|Chrome|Win11"
  "admin@apexfintech.vn|a0000001-0000-0000-0000-000000000001|203.0.113.10|Chrome|Win11"
  "admin@hct.vn|a0000002-0000-0000-0000-000000000001|203.0.113.20|Chrome|Win11"
  "demo@demo.com|a0000003-0000-0000-0000-000000000001|203.0.113.30|Chrome|Win11"
  "agent1@apexfintech.vn|a0000001-0000-0000-0000-000000000010|203.0.113.11|Safari|iPhone"
  "agent2@apexfintech.vn|a0000001-0000-0000-0000-000000000011|203.0.113.12|Firefox|Linux"
  "consultant1@hct.vn|a0000002-0000-0000-0000-000000000010|203.0.113.21|Chrome|Win11"
  "cs1@hct.vn|a0000002-0000-0000-0000-000000000041|203.0.113.22|Chrome|Android"
  "agent1@demo.com|a0000003-0000-0000-0000-000000000010|203.0.113.31|Chrome|Win11"
  "mkt1@demo.com|a0000003-0000-0000-0000-000000000030|203.0.113.32|Chrome|Win11"
)

i=0
for line in "${DEMO_USERS[@]}"; do
  IFS='|' read -r email uid ip ua os <<< "$line"
  i=$((i+1))
  TOKEN="sess-demo-${i}-$(echo -n ${uid} | md5sum | head -c 32)"
  ${REDIS_CMD} -n 0 SETEX "session:${TOKEN}" $((14*86400)) "{\"user_id\":\"${uid}\",\"email\":\"${email}\",\"ip\":\"${ip}\",\"user_agent\":\"${ua}\",\"os\":\"${os}\",\"created_at\":${NOW}}" >/dev/null
  ${REDIS_CMD} -n 0 SADD "user:${uid}:sessions" "${TOKEN}" >/dev/null
  ${REDIS_CMD} -n 0 EXPIRE "user:${uid}:sessions" $((30*86400)) >/dev/null
done
echo "  Inserted ${#DEMO_USERS[@]} sessions"

# ========================================
# Rate limit counters (db 0)
# ========================================
echo "[3/8] Rate limit counters..."
declare -A ENDPOINTS=(
  ["api:crm:/api/v1/leads"]=950
  ["api:crm:/api/v1/deals"]=421
  ["api:auth:/api/v1/auth/login"]=88
  ["api:lead:/api/v1/scoring"]=215
  ["api:notification:/api/v1/notifications"]=1200
  ["api:email:/api/v1/email/send"]=618
)
for ep in "${!ENDPOINTS[@]}"; do
  count=${ENDPOINTS[$ep]}
  ${REDIS_CMD} -n 0 SETEX "ratelimit:${ep}" 60 "${count}" >/dev/null
  ${REDIS_CMD} -n 0 INCRBY "ratelimit:${ep}:total" "${count}" >/dev/null
done

# Per-user rate limits
for i in 1 2 3 4 5 6 7 8 9 10; do
  ${REDIS_CMD} -n 0 SETEX "ratelimit:user:user-${i}:api" 60 $((RANDOM % 100)) >/dev/null
done
echo "  Set ${#ENDPOINTS[@]} endpoint + 10 user counters"

# ========================================
# AI scoring cache (db 0)
# ========================================
echo "[4/8] AI scoring cache..."
SCORE_KEYS=(
  "ai_score:lead:fb.abc123|85"
  "ai_score:lead:g.xyz789|92"
  "ai_score:lead:tt.qwe456|78"
  "ai_score:contact:anh.nguyen@example.com|88"
  "ai_score:contact:binh.tran@example.com|65"
  "ai_score:contact:cuong.le@example.com|95"
)
for line in "${SCORE_KEYS[@]}"; do
  IFS='|' read -r k score <<< "$line"
  ${REDIS_CMD} -n 0 SETEX "${k}" 3600 "{\"score\":${score},\"tier\":\"$([ ${score} -ge 80 ] && echo hot || ([ ${score} -ge 50 ] && echo warm || echo cold)),\"factors\":{\"engagement\":0.8,\"recency\":0.7,\"demographic\":0.9},\"computed_at\":${NOW}}" >/dev/null
done
echo "  Set 6 AI score cache entries"

# ========================================
# Presence data (db 0)
# ========================================
echo "[5/8] Online presence..."
DEMO_ONLINE=(
  "a0000001-0000-0000-0000-000000000001|desktop"
  "a0000002-0000-0000-0000-000000000001|mobile"
  "a0000003-0000-0000-0000-000000000001|web"
  "a0000001-0000-0000-0000-000000000010|desktop"
  "a0000001-0000-0000-0000-000000000011|mobile"
  "a0000002-0000-0000-0000-000000000010|desktop"
  "a0000002-0000-0000-0000-0000aaaaaaaa|web"
  "a0000003-0000-0000-0000-000000000010|desktop"
)
for line in "${DEMO_ONLINE[@]}"; do
  IFS='|' read -r uid device <<< "$line"
  ${REDIS_CMD} -n 0 SETEX "presence:${uid}" 60 "${device}" >/dev/null
done
echo "  Set ${#DEMO_ONLINE[@]} presence entries"

# ========================================
# Lead pipeline caches (db 0)
# ========================================
echo "[6/8] Pipeline + dashboard caches..."
${REDIS_CMD} -n 0 SETEX "cache:dashboard:apexfintech:metrics" 300 '{"total_leads":1287,"won_deals":42,"revenue_vnd":18750000000,"conversion_rate":0.034,"avg_deal_size":446428571}' >/dev/null
${REDIS_CMD} -n 0 SETEX "cache:dashboard:hct-consulting:metrics" 300 '{"total_leads":482,"won_deals":18,"revenue_vnd":42000000000,"conversion_rate":0.058,"avg_deal_size":2333333333}' >/dev/null
${REDIS_CMD} -n 0 SETEX "cache:dashboard:demo-company:metrics" 300 '{"total_leads":124,"won_deals":5,"revenue_vnd":250000000,"conversion_rate":0.04,"avg_deal_size":50000000}' >/dev/null

# Lead funnel counts
for stage in new contacted qualified proposal won lost; do
  ${REDIS_CMD} -n 0 SETEX "cache:funnel:apexfintech:${stage}" 600 $((RANDOM % 500)) >/dev/null
  ${REDIS_CMD} -n 0 SETEX "cache:funnel:hct-consulting:${stage}" 600 $((RANDOM % 200)) >/dev/null
  ${REDIS_CMD} -n 0 SETEX "cache:funnel:demo-company:${stage}" 600 $((RANDOM % 50)) >/dev/null
done
echo "  Set 3 dashboards + 18 funnel counts"

# ========================================
# Idempotency keys (db 0)
# ========================================
echo "[7/8] Idempotency keys..."
for i in 1 2 3 4 5; do
  KEY="idem:landing-submit:apexfintech:${i}"
  ${REDIS_CMD} -n 0 SETEX "${KEY}" 86400 "{\"lead_id\":\"aaaaaaaa-0000-0000-0000-${i}\",\"submitted_at\":${NOW}}" >/dev/null
done

# ========================================
# Email send queue (db 0 list)
# ========================================
echo "[8/8] Email queue (lists)..."
for tenant in apexfintech hct-consulting demo-company; do
  for i in 1 2 3 4 5; do
    ${REDIS_CMD} -n 0 LPUSH "queue:email:${tenant}" "{\"id\":\"email-${i}\",\"to\":\"customer-${i}@example.com\",\"template\":\"welcome\",\"scheduled_at\":${NOW}}" >/dev/null
  done
done
echo "  Pushed 15 emails to queue"

echo ""
echo "=== Valkey seed complete ==="
echo "Total keys inserted: ~80"
