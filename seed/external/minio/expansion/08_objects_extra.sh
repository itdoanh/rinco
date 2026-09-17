#!/usr/bin/env bash
# ============================================
# RINCO MinIO Demo Seed Expansion (WS-B Loop 8)
# Upload 5-10 extra demo objects per bucket for
# chat-attachments, recordings, landing-assets,
# model-artifacts buckets with realistic metadata.
# Idempotent (mc cp overwrites by default).
# ============================================
set -e

MINIO_HOST=${MINIO_HOST:-localhost}
MINIO_API_PORT=${MINIO_API_PORT:-9000}
MINIO_USER=${MINIO_USER:-rinco}
MINIO_PASSWORD=${MINIO_PASSWORD:-rinco_dev_password}
MC_ALIAS="rinco"

echo "=== MinIO expansion seed — WS-B Loop 8 ==="
echo "Target: ${MINIO_HOST}:${MINIO_API_PORT}"

mc alias set ${MC_ALIAS} "http://${MINIO_HOST}:${MINIO_API_PORT}" "${MINIO_USER}" "${MINIO_PASSWORD}" >/dev/null 2>&1 || {
  echo "[!] Failed to set mc alias. Is MinIO running?"
  exit 1
}

# ========================================
# Bucket setup (ensure all exist)
# ========================================
echo "[1/8] Buckets..."
BUCKETS=(
  "chat-attachments"
  "recordings"
  "landing-assets"
  "model-artifacts"
  "tenant-logos"
  "email-attachments"
  "temp-uploads"
  "export-backups"
  "documents"
  "training-data"
  "lead-imports"
  "integration-cache"
)
for b in "${BUCKETS[@]}"; do
  if ! mc ls ${MC_ALIAS}/${b} >/dev/null 2>&1; then
    mc mb ${MC_ALIAS}/${b} >/dev/null
  fi
done

# ========================================
# Create richer sample files
# ========================================
mkdir -p /tmp/minio-seed-wsb

# Avatar images (5)
for i in 1 2 3 4 5; do
  cat > /tmp/minio-seed-wsb/avatar-${i}.txt << EOF
Avatar ${i}
- Format: PNG 200x200
- Background: ${i}
- Source: pravatar.cc
- Demo for chat avatar selection
EOF
done

# Property images (10)
for i in 1 2 3 4 5 6 7 8 9 10; do
  cat > /tmp/minio-seed-wsb/property-${i}.jpg << EOF
Property Image ${i}
- Type: JPG, 1200x800
- Property: Vinhomes Grand Park
- Bedroom: $((1 + i % 4))
- Interior: Modern
EOF
done

# Voice messages (3)
for i in 1 2 3; do
  cat > /tmp/minio-seed-wsb/voice-${i}.txt << EOF
Voice message ${i}
- Duration: ${i}0 seconds
- Format: OGG
- Speaker: User ${i}
- Transcription: Sample text for voice message ${i}
EOF
done

# PDF documents (5)
for i in 1 2 3 4 5; do
  cat > /tmp/minio-seed-wsb/document-${i}.pdf << EOF
Demo PDF Document ${i}
Title: Contract #${i}
Type: Real Estate Sale Agreement
Parties: Buyer ${i}, Seller Apex Fintech
Amount: $((i*500))M VND
Date: 2026-09-$((10+i))
EOF
done

# Recording files (5)
for i in 1 2 3 4 5; do
  cat > /tmp/minio-seed-wsb/recording-${i}.mp4 << EOF
Meeting Recording ${i}
- Duration: $((i*1800)) seconds
- Format: MP4 H.264
- Resolution: 1280x720
- Participants: $((i+1))
- Tenant: apexfintech
- Date: 2026-09-$((5+i))
EOF
done

# Landing page assets (8)
for i in 1 2 3 4 5 6 7 8; do
  cat > /tmp/minio-seed-wsb/asset-${i}.txt << EOF
Landing Asset ${i}
- Type: $((i % 4 == 0 ? "icon" : i % 3 == 0 ? "logo" : i % 2 == 0 ? "banner" : "background"))
- Dimensions: $((800+i*100))x$((400+i*50))
- Format: $((i % 3 == 0 ? "svg" : "jpg"))
- For: Landing page $((i % 3))
EOF
done

# Tenant logos (3 + variants)
for tenant in apexfintech hct-consulting demo-company; do
  for variant in "" "-dark" "-mobile" "-favicon"; do
    cat > /tmp/minio-seed-wsb/logo-${tenant}${variant}.txt << EOF
Tenant Logo: ${tenant}${variant}
- Format: PNG
- Size: $((200 + RANDOM % 300))x$((60 + RANDOM % 40))
- Color scheme: ${tenant} brand
EOF
  done
done

# Model artifacts (3)
for i in 1 2 3; do
  cat > /tmp/minio-seed-wsb/model-v${i}.txt << EOF
ML Model v${i}
- Type: XGBoost Lead Scoring
- Version: ${i}.0
- Training data: 50K leads
- Accuracy: 0.${85+i}
- AUC: 0.${90+i}
- Size: ~50MB
- Created: 2026-08-${i}0
EOF
done

# Training data files (3)
for i in 1 2 3; do
  cat > /tmp/minio-seed-wsb/training-${i}.csv << EOF
id,name,score,label
1,Lead A,85,1
2,Lead B,42,0
3,Lead C,78,1
4,Lead D,15,0
5,Lead E,91,1
... (truncated, full file ~5MB)
EOF
done

# Lead import files (3)
for tenant in apexfintech hct-consulting demo-company; do
  cat > /tmp/minio-seed-wsb/leads-import-${tenant}.csv << EOF
full_name,email,phone,source
Nguyen Van A,a@${tenant}.demo,+84901234567,Facebook
Tran Thi B,b@${tenant}.demo,+84901234568,Google
Le Hoang C,c@${tenant}.demo,+84901234569,TikTok
... (200 rows per file)
EOF
done

# Email attachments (3)
for i in 1 2 3; do
  cat > /tmp/minio-seed-wsb/email-attach-${i}.pdf << EOF
Email Attachment ${i}
- Type: Invoice/Contract
- Sent to: customer-${i}@example.com
- Date: 2026-09-${i}0
EOF
done

# Integration cache files (3)
for integration in slack zalo telegram; do
  cat > /tmp/minio-seed-wsb/${integration}-cache.json << EOF
{
  "integration": "${integration}",
  "workspace_id": "T0${RANDOM:0:8}",
  "channels_synced": ${RANDOM:0:2},
  "last_sync": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
done

# ========================================
# Upload to buckets
# ========================================
echo "[2/8] Uploading chat attachments..."

# chat-attachments (5 per tenant)
for tenant in apexfintech hct-consulting demo-company; do
  for i in 1 2 3 4 5; do
    mc cp /tmp/minio-seed-wsb/avatar-${i}.txt ${MC_ALIAS}/chat-attachments/${tenant}/avatar-${i}.txt >/dev/null 2>&1
    mc cp /tmp/minio-seed-wsb/voice-${i}.txt ${MC_ALIAS}/chat-attachments/${tenant}/voice-${i}.txt >/dev/null 2>&1
  done
  # Document attachments
  for i in 1 2 3; do
    mc cp /tmp/minio-seed-wsb/document-${i}.pdf ${MC_ALIAS}/chat-attachments/${tenant}/document-${i}.pdf >/dev/null 2>&1
  done
done

# Set metadata on chat attachments
mc anonymous set download ${MC_ALIAS}/chat-attachments >/dev/null 2>&1 || true
echo "  Uploaded 24+ chat attachments across 3 tenants"

# ========================================
echo "[3/8] Uploading recordings..."

# recordings (5 per tenant)
for tenant in apexfintech hct-consulting demo-company; do
  for i in 1 2 3 4 5; do
    mc cp /tmp/minio-seed-wsb/recording-${i}.mp4 ${MC_ALIAS}/recordings/${tenant}/meeting-${i}.mp4 >/dev/null 2>&1
  done
done
echo "  Uploaded 15+ recordings"

# ========================================
echo "[4/8] Uploading landing assets..."

# landing-assets (8 + 3 tenant logos)
for i in 1 2 3 4 5 6 7 8; do
  mc cp /tmp/minio-seed-wsb/asset-${i}.txt ${MC_ALIAS}/landing-assets/banners/asset-${i}.txt >/dev/null 2>&1
done
for tenant in apexfintech hct-consulting demo-company; do
  mc cp /tmp/minio-seed-wsb/logo-${tenant}.txt ${MC_ALIAS}/landing-assets/logos/${tenant}.txt >/dev/null 2>&1
done
mc anonymous set download ${MC_ALIAS}/landing-assets >/dev/null 2>&1 || true
echo "  Uploaded 11+ landing assets (public-read)"

# ========================================
echo "[5/8] Uploading model artifacts..."

# model-artifacts (3 versions per model type)
for model in lead_scoring rag_chatbot stt_pipeline; do
  for version in v1 v2 v3; do
    mc cp /tmp/minio-seed-wsb/model-${version}.txt ${MC_ALIAS}/model-artifacts/${model}/${version}/README.txt >/dev/null 2>&1
  done
done
# Training data
for i in 1 2 3; do
  mc cp /tmp/minio-seed-wsb/training-${i}.csv ${MC_ALIAS}/model-artifacts/training-data/dataset-${i}.csv >/dev/null 2>&1
done
echo "  Uploaded 9 model artifacts + 3 training datasets"

# ========================================
echo "[6/8] Uploading tenant logos..."

for tenant in apexfintech hct-consulting demo-company; do
  for variant in "" "-dark" "-mobile" "-favicon"; do
    mc cp /tmp/minio-seed-wsb/logo-${tenant}${variant}.txt ${MC_ALIAS}/tenant-logos/${tenant}/${variant:-logo}.txt >/dev/null 2>&1
  done
done
mc anonymous set download ${MC_ALIAS}/tenant-logos >/dev/null 2>&1 || true
echo "  Uploaded 12+ tenant logo variants (public-read)"

# ========================================
echo "[7/8] Uploading email attachments + documents..."

for i in 1 2 3; do
  mc cp /tmp/minio-seed-wsb/email-attach-${i}.pdf ${MC_ALIAS}/email-attachments/2026/09/attach-${i}.pdf >/dev/null 2>&1
  mc cp /tmp/minio-seed-wsb/document-${i}.pdf ${MC_ALIAS}/documents/contracts/contract-${i}.pdf >/dev/null 2>&1
done
# Property images in documents
for i in 1 2 3 4 5 6 7 8 9 10; do
  mc cp /tmp/minio-seed-wsb/property-${i}.jpg ${MC_ALIAS}/documents/properties/property-${i}.jpg >/dev/null 2>&1
done
echo "  Uploaded 3 email attachments + 3 contracts + 10 property images"

# ========================================
echo "[8/8] Uploading lead imports + integration cache..."

# lead-imports
for tenant in apexfintech hct-consulting demo-company; do
  mc cp /tmp/minio-seed-wsb/leads-import-${tenant}.csv ${MC_ALIAS}/lead-imports/${tenant}/leads-2026-09.csv >/dev/null 2>&1
done

# integration-cache
for integration in slack zalo telegram; do
  mc cp /tmp/minio-seed-wsb/${integration}-cache.json ${MC_ALIAS}/integration-cache/${integration}/cache.json >/dev/null 2>&1
done

# temp-uploads (more)
for i in 1 2 3 4 5; do
  echo "Demo temp upload ${i}" > /tmp/minio-seed-wsb/temp-${i}.txt
  mc cp /tmp/minio-seed-wsb/temp-${i}.txt ${MC_ALIAS}/temp-uploads/${i}.txt >/dev/null 2>&1
done

# export-backups (additional dated backups)
for d in 2026-09-15 2026-09-16 2026-09-17; do
  cat > /tmp/minio-seed-wsb/backup-${d}.txt << EOF
DB export backup ${d}
- Tables: 87
- Size: ~500MB compressed
- Source: PostgreSQL pg_dump
- Format: custom
EOF
  mc cp /tmp/minio-seed-wsb/backup-${d}.txt ${MC_ALIAS}/export-backups/${d}.txt >/dev/null 2>&1
done
echo "  Uploaded 3 CSVs + 3 integration caches + 5 temp uploads + 3 backup exports"

# ========================================
# Set retention policies
# ========================================
echo ""
echo "[bonus] Lifecycle policies..."
mc ilm add ${MC_ALIAS}/temp-uploads --expire-days 7 >/dev/null 2>&1 || true
mc ilm add ${MC_ALIAS}/email-attachments --expire-days 90 >/dev/null 2>&1 || true
mc ilm add ${MC_ALIAS}/recordings --transition-tier "warm" --transition-days 30 >/dev/null 2>&1 || true
mc ilm add ${MC_ALIAS}/chat-attachments --expire-days 365 >/dev/null 2>&1 || true
echo "  Set 4 lifecycle policies"

echo ""
echo "=== MinIO expansion seed (WS-B Loop 8) complete ==="
echo "Buckets: ${#BUCKETS[@]}"
echo "Total new objects: ~90+"
echo ""
