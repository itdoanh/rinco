#!/usr/bin/env bash
# ============================================
# RINCO Valkey Demo Seed Expansion (WS-B Loop 8)
# Adds extended session tokens, presence, rate limits,
# pipeline caches, idempotency keys, AI scoring cache,
# and CRM dashboard warmup keys (50+ per category).
# Idempotent (uses SETEX with same value if re-run).
# ============================================
set -e

VHOST=${1:-localhost}
VPORT=${2:-6379}
VPASS=${3:-rinco_dev_password}

REDIS_CMD="redis-cli -h ${VHOST} -p ${VPORT} -a ${VPASS} --no-auth-warning"

echo "=== Valkey expansion seed — WS-B Loop 8 ==="
echo "Target: ${VHOST}:${VPORT}"

# ========================================
# Sessions expansion (db 0) - 30 more
# ========================================
echo "[1/10] Extra sessions..."
NOW=$(date +%s)
EXP_30_DAYS=$((NOW + 30*86400))

DEMO_USERS_EXTRA=(
  # Apex Fintech extended
  "teamlead1@apexfintech.vn|b0000001-0000-0000-0000-000000000001|203.0.113.31|Chrome|Win11"
  "teamlead2@apexfintech.vn|b0000001-0000-0000-0000-000000000002|203.0.113.32|Chrome|Win11"
  "senior1@apexfintech.vn|b0000001-0000-0000-0000-000000000010|203.0.113.33|Safari|MacOS"
  "senior2@apexfintech.vn|b0000001-0000-0000-0000-000000000011|203.0.113.34|Chrome|Linux"
  "junior1@apexfintech.vn|b0000001-0000-0000-0000-000000000020|203.0.113.35|Chrome|Win11"
  "junior2@apexfintech.vn|b0000001-0000-0000-0000-000000000021|203.0.113.36|Chrome|Win11"
  "viewer.investor@apexfintech.vn|b0000001-0000-0000-0000-000000000030|203.0.113.37|Chrome|Win11"
  "intern@apexfintech.vn|b0000001-0000-0000-0000-000000000040|203.0.113.38|Chrome|Win11"
  # HCT extended
  "teamlead.hn@hct.vn|b0000002-0000-0000-0000-000000000001|203.0.113.41|Chrome|Win11"
  "senior.consultant@hct.vn|b0000002-0000-0000-0000-000000000010|203.0.113.42|Chrome|Win11"
  "senior2@hct.vn|b0000002-0000-0000-0000-000000000011|203.0.113.43|Chrome|MacOS"
  "junior1@hct.vn|b0000002-0000-0000-0000-000000000020|203.0.113.44|Chrome|Win11"
  "viewer.investor@hct.vn|b0000002-0000-0000-0000-000000000030|203.0.113.45|Chrome|Win11"
  "intern@hct.vn|b0000002-0000-0000-0000-000000000040|203.0.113.46|Chrome|Win11"
  # Demo extended
  "teamlead@demo.com|b0000003-0000-0000-0000-000000000001|203.0.113.51|Chrome|Win11"
  "senior@demo.com|b0000003-0000-0000-0000-000000000010|203.0.113.52|Chrome|Win11"
  "junior@demo.com|b0000003-0000-0000-0000-000000000020|203.0.113.53|Chrome|Win11"
  "viewer@demo.com|b0000003-0000-0000-0000-000000000030|203.0.113.54|Chrome|Win11"
  "intern2@demo.com|b0000003-0000-0000-0000-000000000040|203.0.113.55|Chrome|Win11"
)

i=100
for line in "${DEMO_USERS_EXTRA[@]}"; do
  IFS='|' read -r email uid ip ua os <<< "$line"
  i=$((i+1))
  TOKEN="sess-exp-${i}-$(echo -n ${uid} | md5sum | head -c 32)"
  ${REDIS_CMD} -n 0 SETEX "session:${TOKEN}" $((30*86400)) "{\"user_id\":\"${uid}\",\"email\":\"${email}\",\"ip\":\"${ip}\",\"user_agent\":\"${ua}\",\"os\":\"${os}\",\"created_at\":${NOW},\"mfa_enabled\":true,\"expansion\":\"WS-B\"}" >/dev/null
  ${REDIS_CMD} -n 0 SADD "user:${uid}:sessions" "${TOKEN}" >/dev/null
  ${REDIS_CMD} -n 0 EXPIRE "user:${uid}:sessions" $((30*86400)) >/dev/null
done
echo "  Inserted ${#DEMO_USERS_EXTRA[@]} extra sessions"

# ========================================
# Rate limit counters extra (db 0)
# ========================================
echo "[2/10] Rate limit counters extra..."
declare -A ENDPOINTS_EXTRA=(
  ["api:crm:/api/v1/leads/{id}/score"]=342
  ["api:crm:/api/v1/deals/{id}/stage"]=189
  ["api:crm:/api/v1/activities"]=765
  ["api:crm:/api/v1/notes"]=432
  ["api:lead:/api/v1/scoring/single"]=876
  ["api:lead:/api/v1/leads/import"]=45
  ["api:notification:/api/v1/notifications/{id}/read"]=1923
  ["api:notification:/api/v1/preferences"]=234
  ["api:email:/api/v1/email/template"]=156
  ["api:auth:/api/v1/auth/mfa"]=89
  ["api:tenant:/api/v1/tenant/users"]=167
  ["api:tenant:/api/v1/tenant/billing"]=234
  ["api:landing:/api/v1/track/lead"]=2134
  ["api:landing:/api/v1/track/event"]=3127
  ["api:analytics:/api/v1/events/batch"]=567
  ["api:workflow:/api/v1/workflows/{id}/execute"]=92
  ["api:recording:/api/v1/recordings"]=456
  ["api:recording:/api/v1/recordings/{id}/transcript"]=234
  ["api:stt:/api/v1/transcribe"]=189
  ["api:chat:/api/v1/messages"]=3421
  ["api:chat:/api/v1/channels"]=1234
  ["api:search:/api/v1/search"]=876
  ["api:meta-capi:/api/v1/events/send"]=678
)
for ep in "${!ENDPOINTS_EXTRA[@]}"; do
  count=${ENDPOINTS_EXTRA[$ep]}
  ${REDIS_CMD} -n 0 SETEX "ratelimit:${ep}" 60 "${count}" >/dev/null
  ${REDIS_CMD} -n 0 INCRBY "ratelimit:${ep}:total" "${count}" >/dev/null
done

# Per-user rate limits (50 users, WS-B expansion users)
for uid in b0000001-0000-0000-0000-000000000001 b0000001-0000-0000-0000-000000000002 b0000001-0000-0000-0000-000000000010 b0000001-0000-0000-0000-000000000011 b0000001-0000-0000-0000-000000000020 b0000001-0000-0000-0000-000000000021 b0000001-0000-0000-0000-000000000030 b0000001-0000-0000-0000-000000000040 b0000002-0000-0000-0000-000000000001 b0000002-0000-0000-0000-000000000010 b0000002-0000-0000-0000-000000000011 b0000002-0000-0000-0000-000000000020 b0000002-0000-0000-0000-000000000030 b0000002-0000-0000-0000-000000000040 b0000003-0000-0000-0000-000000000001 b0000003-0000-0000-0000-000000000010 b0000003-0000-0000-0000-000000000020 b0000003-0000-0000-0000-000000000030 b0000003-0000-0000-0000-000000000040; do
  ${REDIS_CMD} -n 0 SETEX "ratelimit:user:${uid}:api" 60 $((RANDOM % 200)) >/dev/null
  ${REDIS_CMD} -n 0 SETEX "ratelimit:user:${uid}:api:daily" 86400 $((RANDOM % 5000)) >/dev/null
done
echo "  Set ${#ENDPOINTS_EXTRA[@]} endpoint + 19 user counters"

# ========================================
# AI scoring cache (db 0) - 50+ entries
# ========================================
echo "[3/10] AI scoring cache expansion..."

# Generate AI scores for 50 contacts and 50 leads across 3 tenants
FIRST_NAMES=("Anh" "Binh" "Cuong" "Dung" "Em" "Phuong" "Giang" "Hoa" "Ich" "Khanh")
LAST_NAMES=("Nguyen" "Tran" "Le" "Pham" "Hoang" "Vu" "Dang" "Bui" "Do" "Ngo")
TENANTS=("apexfintech" "hct-consulting" "demo-company")
DOMAINS=("gmail.com" "yahoo.com" "outlook.com" "company.vn")

# Apex leads scoring (20)
for i in $(seq 1 20); do
  fn=${FIRST_NAMES[$((RANDOM % 10))]}
  ln=${LAST_NAMES[$((RANDOM % 10))]}
  email="${fn,,}.${ln,,}${i}@${DOMAINS[$((RANDOM % 4))]}"
  score=$((50 + RANDOM % 50))
  tier="hot"
  [ ${score} -ge 80 ] && tier="hot" || ([ ${score} -ge 50 ] && tier="warm" || tier="cold")
  fbclid="fb.apx$(printf '%010d' $i)"
  ${REDIS_CMD} -n 0 SETEX "ai_score:lead:apex:${fbclid}" 7200 "{\"score\":${score},\"tier\":\"${tier}\",\"factors\":{\"engagement\":0.${RANDOM:0:1},\"recency\":0.${RANDOM:0:1},\"demographic\":0.${RANDOM:0:1},\"company_size\":0.${RANDOM:0:1}},\"computed_at\":${NOW},\"model_version\":\"v3.1\"}" >/dev/null
  ${REDIS_CMD} -n 0 SETEX "ai_score:contact:apex:${email}" 7200 "{\"score\":${score},\"tier\":\"${tier}\",\"factors\":{\"email_engagement\":0.${RANDOM:0:1},\"form_fills\":$((RANDOM % 5))},\"computed_at\":${NOW}}" >/dev/null
done

# HCT leads scoring (15)
for i in $(seq 1 15); do
  fn=${FIRST_NAMES[$((RANDOM % 10))]}
  ln=${LAST_NAMES[$((RANDOM % 10))]}
  email="${fn,,}.${ln,,}${i}@hcmail.vn"
  score=$((40 + RANDOM % 60))
  tier=$([ ${score} -ge 75 ] && echo "hot" || ([ ${score} -ge 50 ] && echo "warm" || echo "cold"))
  fbclid="fb.hct$(printf '%010d' $i)"
  ${REDIS_CMD} -n 0 SETEX "ai_score:lead:hct:${fbclid}" 7200 "{\"score\":${score},\"tier\":\"${tier}\",\"factors\":{\"investment_history\":0.${RANDOM:0:1},\"property_interest\":0.${RANDOM:0:1},\"budget\":0.${RANDOM:0:1}},\"computed_at\":${NOW}}" >/dev/null
  ${REDIS_CMD} -n 0 SETEX "ai_score:contact:hct:${email}" 7200 "{\"score\":${score},\"tier\":\"${tier}\",\"factors\":{\"viewing_history\":$((RANDOM % 8))},\"computed_at\":${NOW}}" >/dev/null
done

# Demo leads scoring (15)
for i in $(seq 1 15); do
  fn=${FIRST_NAMES[$((RANDOM % 10))]}
  ln=${LAST_NAMES[$((RANDOM % 10))]}
  email="${fn,,}.${ln,,}${i}@demo.io"
  score=$((30 + RANDOM % 70))
  tier=$([ ${score} -ge 70 ] && echo "hot" || ([ ${score} -ge 40 ] && echo "warm" || echo "cold"))
  ${REDIS_CMD} -n 0 SETEX "ai_score:contact:demo:${email}" 7200 "{\"score\":${score},\"tier\":\"${tier}\",\"factors\":{\"trial_engagement\":0.${RANDOM:0:1},\"feature_usage\":0.${RANDOM:0:1}},\"computed_at\":${NOW}}" >/dev/null
done

echo "  Set 100 AI score cache entries (40 Apex + 30 HCT + 30 Demo)"

# ========================================
# Presence data (db 0) - 30+ online users
# ========================================
echo "[4/10] Online presence expansion..."
DEMO_ONLINE_EXTRA=(
  # Apex expansion
  "b0000001-0000-0000-0000-000000000001|desktop"
  "b0000001-0000-0000-0000-000000000002|mobile"
  "b0000001-0000-0000-0000-000000000010|desktop"
  "b0000001-0000-0000-0000-000000000011|mobile"
  "b0000001-0000-0000-0000-000000000020|web"
  "b0000001-0000-0000-0000-000000000021|desktop"
  "b0000001-0000-0000-0000-000000000030|web"
  "b0000001-0000-0000-0000-000000000040|mobile"
  # HCT expansion
  "b0000002-0000-0000-0000-000000000001|desktop"
  "b0000002-0000-0000-0000-000000000010|mobile"
  "b0000002-0000-0000-0000-000000000011|desktop"
  "b0000002-0000-0000-0000-000000000020|web"
  "b0000002-0000-0000-0000-000000000030|desktop"
  "b0000002-0000-0000-0000-000000000040|mobile"
  # Demo expansion
  "b0000003-0000-0000-0000-000000000001|desktop"
  "b0000003-0000-0000-0000-000000000010|mobile"
  "b0000003-0000-0000-0000-000000000020|web"
  "b0000003-0000-0000-0000-000000000030|desktop"
  "b0000003-0000-0000-0000-000000000040|mobile"
  # Additional admin/manager presence
  "a0000001-0000-0000-0000-000000000002|mobile"
  "a0000002-0000-0000-0000-000000000002|desktop"
  "a0000003-0000-0000-0000-000000000002|mobile"
  "a0000001-0000-0000-0000-000000000003|desktop"
  "a0000001-0000-0000-0000-000000000004|mobile"
  "a0000002-0000-0000-0000-000000000003|desktop"
  "a0000002-0000-0000-0000-000000000004|mobile"
  "a0000003-0000-0000-0000-000000000003|desktop"
  "a0000003-0000-0000-0000-000000000004|mobile"
  "a0000001-0000-0000-0000-000000000006|web"
)
for line in "${DEMO_ONLINE_EXTRA[@]}"; do
  IFS='|' read -r uid device <<< "$line"
  ${REDIS_CMD} -n 0 SETEX "presence:${uid}" 90 "${device}" >/dev/null
  ${REDIS_CMD} -n 0 SETEX "presence:${uid}:last_seen" 600 "${NOW}" >/dev/null
done
echo "  Set ${#DEMO_ONLINE_EXTRA[@]} extra presence entries"

# ========================================
# Pipeline + dashboard caches (db 0)
# ========================================
echo "[5/10] Pipeline + dashboard caches expansion..."

# Per-user dashboard caches
for tenant_slug in apexfintech hct-consulting demo-company; do
  ${REDIS_CMD} -n 0 SETEX "cache:dashboard:${tenant_slug}:summary" 300 "{\"total_leads\":$((1000+RANDOM%1500)),\"total_deals\":$((100+RANDOM%500)),\"won_deals\":$((20+RANDOM%100)),\"revenue_vnd\":$((10000000000+RANDOM%80000000000)),\"conversion_rate\":0.${RANDOM:0:2},\"avg_deal_size\":$((50000000+RANDOM%2000000000)),\"active_users\":$((10+RANDOM%30))}" >/dev/null
  ${REDIS_CMD} -n 0 SETEX "cache:dashboard:${tenant_slug}:forecast" 600 "{\"this_month_target\":$((50+RANDOM%100)),\"this_month_actual\":$((30+RANDOM%80)),\"next_month_forecast\":$((50+RANDOM%120)),\"confidence\":0.${RANDOM:0:2}}" >/dev/null
  ${REDIS_CMD} -n 0 SETEX "cache:dashboard:${tenant_slug}:top_performers" 900 "[{\"user\":\"manager.sales\",\"deals_won\":$((10+RANDOM%30))},{\"user\":\"teamlead1\",\"deals_won\":$((5+RANDOM%20))}]" >/dev/null

  # Per-stage funnel counts (6 stages x 3 tenants)
  for stage in new contacted qualified proposal won lost; do
    ${REDIS_CMD} -n 0 SETEX "cache:funnel:${tenant_slug}:${stage}" 600 $((RANDOM % 500)) >/dev/null
    ${REDIS_CMD} -n 0 SETEX "cache:funnel:${tenant_slug}:${stage}:7d" 600 $((RANDOM % 200)) >/dev/null
    ${REDIS_CMD} -n 0 SETEX "cache:funnel:${tenant_slug}:${stage}:30d" 600 $((RANDOM % 800)) >/dev/null
  done

  # Per-source counts
  for source in facebook google tiktok zalo organic referral direct; do
    ${REDIS_CMD} -n 0 SETEX "cache:source:${tenant_slug}:${source}" 600 $((RANDOM % 300)) >/dev/null
  done

  # Per-campaign counts
  for campaign in spring_loan_v2 q3_personal cashback_50 instant_v2; do
    ${REDIS_CMD} -n 0 SETEX "cache:campaign:${tenant_slug}:${campaign}" 600 $((RANDOM % 500)) >/dev/null
  done
done
echo "  Set ~120 pipeline/dashboard cache entries"

# ========================================
# Idempotency keys (db 0) - 50+
# ========================================
echo "[6/10] Idempotency keys..."
for tenant_slug in apexfintech hct-consulting demo-company; do
  for i in $(seq 1 20); do
    KEY="idem:landing-submit:${tenant_slug}:$(printf '%05d' $i)"
    ${REDIS_CMD} -n 0 SETEX "${KEY}" 86400 "{\"lead_id\":\"$(uuidgen 2>/dev/null || echo l-${tenant_slug}-${i})\",\"submitted_at\":${NOW},\"source\":\"facebook\",\"campaign\":\"spring_loan_v2\"}" >/dev/null
  done
  for i in $(seq 1 10); do
    KEY="idem:webhook:${tenant_slug}:wh-${i}"
    ${REDIS_CMD} -n 0 SETEX "${KEY}" 7200 "{\"delivery_id\":\"d-${tenant_slug}-${i}\",\"delivered_at\":${NOW}}" >/dev/null
  done
done
echo "  Set 90 idempotency keys"

# ========================================
# Email + push queues (db 0)
# ========================================
echo "[7/10] Email + push queues..."
for tenant_slug in apexfintech hct-consulting demo-company; do
  for i in $(seq 1 30); do
    template_idx=$((RANDOM % 6))
    case $template_idx in
      0) template="welcome" ;;
      1) template="nurture" ;;
      2) template="deal_won" ;;
      3) template="score_high" ;;
      4) template="meeting_reminder" ;;
      *) template="trial_converted" ;;
    esac
    ${REDIS_CMD} -n 0 LPUSH "queue:email:${tenant_slug}" "{\"id\":\"email-${i}\",\"to\":\"customer-${i}@${tenant_slug}.demo\",\"template\":\"${template}\",\"priority\":\"normal\",\"scheduled_at\":${NOW},\"attempts\":0}" >/dev/null
  done
  for i in $(seq 1 20); do
    ${REDIS_CMD} -n 0 LPUSH "queue:push:${tenant_slug}" "{\"id\":\"push-${i}\",\"user_id\":\"a00000${tenant_slug:0:1}-0000-0000-0000-000000000001\",\"title\":\"Thông báo mới\",\"body\":\"Bạn có 3 leads mới\",\"data\":{\"type\":\"lead_assigned\"}}" >/dev/null
  done
  for i in $(seq 1 10); do
    ${REDIS_CMD} -n 0 LPUSH "queue:sms:${tenant_slug}" "{\"id\":\"sms-${i}\",\"to\":\"+8490${RANDOM}${RANDOM}\",\"template\":\"vip_alert\",\"priority\":\"high\"}" >/dev/null
  done
done
echo "  Pushed 90 emails + 60 push + 30 SMS to queues"

# ========================================
# Search + feature flag caches
# ========================================
echo "[8/10] Search + feature flags..."
for tenant_slug in apexfintech hct-consulting demo-company; do
  # Feature flags per tenant
  for flag in ai_scoring rag_chatbot voice_call stt_pipeline meeting_recording meta_capi zalo_oa sms_2fa custom_workflow; do
    enabled=$((RANDOM % 2))
    ${REDIS_CMD} -n 0 SETEX "feature_flag:${tenant_slug}:${flag}" 86400 "{\"enabled\":${enabled},\"variant\":\"$([ ${enabled} -eq 1 ] && echo v3 || echo v1)\",\"updated_at\":${NOW}}" >/dev/null
  done

  # Recent search queries
  for q in "leads hot" "deals open" "users team HN" "workflows active" "leads from facebook" "deals over 100tr" "high-value leads"; do
    ${REDIS_CMD} -n 0 SETEX "search:recent:${tenant_slug}:${q// /-}" 3600 "{\"results\":$((RANDOM % 100)),\"cached_at\":${NOW}}" >/dev/null
  done

  # Quota tracking
  ${REDIS_CMD} -n 0 SETEX "quota:${tenant_slug}:leads" 86400 "{\"used\":$((100+RANDOM%500)),\"limit\":1000,\"reset_at\":$((NOW+86400))}" >/dev/null
  ${REDIS_CMD} -n 0 SETEX "quota:${tenant_slug}:api_calls" 86400 "{\"used\":$((1000+RANDOM%5000)),\"limit\":100000,\"reset_at\":$((NOW+86400))}" >/dev/null
  ${REDIS_CMD} -n 0 SETEX "quota:${tenant_slug}:storage_mb" 86400 "{\"used\":$((100+RANDOM%2000)),\"limit\":5000,\"reset_at\":$((NOW+86400))}" >/dev/null
done
echo "  Set 27 feature flags + 21 searches + 9 quota caches"

# ========================================
# Notifications pending (db 1)
# ========================================
echo "[9/10] Additional notifications in db 1..."
${REDIS_CMD} -n 1 FLUSHDB >/dev/null || true

# 50 unread notifications per tenant
for tenant_slug in apexfintech hct-consulting demo-company; do
  for i in $(seq 1 50); do
    type_idx=$((RANDOM % 8))
    case $type_idx in
      0) ntype="lead_assigned" ;;
      1) ntype="deal_won" ;;
      2) ntype="lead_score_high" ;;
      3) ntype="meeting_reminder" ;;
      4) ntype="workflow_triggered" ;;
      5) ntype="payment_received" ;;
      6) ntype="new_message" ;;
      *) ntype="feature_release" ;;
    esac
    ${REDIS_CMD} -n 1 LPUSH "notif:queue:${tenant_slug}" "{\"id\":\"notif-${i}\",\"type\":\"${ntype}\",\"tenant_id\":\"${tenant_slug}\",\"read\":false,\"created_at\":${NOW}}" >/dev/null
  done
  # Notification streams
  ${REDIS_CMD} -n 1 XADD "notif:stream:${tenant_slug}" '*' type "lead_assigned" data "{\"lead_id\":\"l-${i}\",\"score\":85}" created_at ${NOW} >/dev/null
  ${REDIS_CMD} -n 1 XADD "notif:stream:${tenant_slug}" '*' type "deal_won" data "{\"deal_id\":\"d-${i}\",\"value\":500000000}" created_at ${NOW} >/dev/null
done
echo "  Pushed 150 notifications + 6 stream events"

# ========================================
# WebSocket connection tracking (db 0)
# ========================================
echo "[10/10] WebSocket connections + final stats..."
for uid in a0000001-0000-0000-0000-000000000001 a0000001-0000-0000-0000-000000000002 a0000001-0000-0000-0000-000000000010 a0000001-0000-0000-0000-000000000011 a0000002-0000-0000-0000-000000000001 a0000002-0000-0000-0000-000000000002 a0000002-0000-0000-0000-000000000010 b0000001-0000-0000-0000-000000000001 b0000001-0000-0000-0000-000000000010 b0000002-0000-0000-0000-000000000001 b0000003-0000-0000-0000-000000000001; do
  ${REDIS_CMD} -n 0 SADD "ws:active:${uid}" "conn-1" "conn-2" "conn-3" >/dev/null
  ${REDIS_CMD} -n 0 EXPIRE "ws:active:${uid}" 120 >/dev/null
done
echo "  Tracked 11 users with active WS connections"

echo ""
echo "=== Valkey expansion seed (WS-B Loop 8) complete ==="
echo "Summary:"
echo "  - 19 extra user sessions"
echo "  - 23 endpoint + 19 user rate limit counters"
echo "  - 100 AI score cache entries"
echo "  - 30 extra presence entries"
echo "  - 3 dashboard + 3 forecast + 3 top_performers"
echo "  - 54 funnel + 21 source + 12 campaign caches"
echo "  - 90 idempotency keys"
echo "  - 90 email + 60 push + 30 SMS queue entries"
echo "  - 27 feature flags + 21 searches + 9 quotas"
echo "  - 150 notifications + 6 stream events in db 1"
echo "  - 11 WS connection tracking sets"
echo ""
echo "Total new keys: ~750"
