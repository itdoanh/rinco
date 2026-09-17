#!/usr/bin/env bash
# ============================================
# RINCO MinIO Demo Seed (Loop 202)
# Usage:   bash minio/seed.sh
# - Creates buckets: chat-attachments, recordings, landing-assets,
#   model-artifacts, tenant-logos
# - Uploads 1-2 sample files per bucket
# - Sets anonymous read policy on landing-assets
# ============================================
set -e

MINIO_HOST=${MINIO_HOST:-localhost}
MINIO_API_PORT=${MINIO_API_PORT:-9000}
MINIO_CONSOLE_PORT=${MINIO_CONSOLE_PORT:-9001}
MINIO_USER=${MINIO_USER:-rinco}
MINIO_PASSWORD=${MINIO_PASSWORD:-rinco_dev_password}
MC_ALIAS="rinco"

echo "=== MinIO demo seed — Loop 202 ==="
echo "Target: ${MINIO_HOST}:${MINIO_API_PORT}"

# Configure mc client
mc alias set ${MC_ALIAS} "http://${MINIO_HOST}:${MINIO_API_PORT}" "${MINIO_USER}" "${MINIO_PASSWORD}" >/dev/null 2>&1 || {
  echo "[!] Failed to set mc alias. Is MinIO running?"
  exit 1
}

# Helper
ensure_bucket() {
  local bucket=$1
  if ! mc ls ${MC_ALIAS}/${bucket} >/dev/null 2>&1; then
    echo "  Creating bucket: ${bucket}"
    mc mb ${MC_ALIAS}/${bucket}
  else
    echo "  Bucket exists: ${bucket}"
  fi
}

# ============================================
# Create buckets
# ============================================
echo "[1/4] Buckets..."
BUCKETS=(
  "chat-attachments"
  "recordings"
  "landing-assets"
  "model-artifacts"
  "tenant-logos"
  "email-attachments"
  "temp-uploads"
  "export-backups"
)
for b in "${BUCKETS[@]}"; do
  ensure_bucket "$b"
done

# ============================================
# Create dummy sample files in /tmp
# ============================================
echo "[2/4] Sample files..."
mkdir -p /tmp/minio-seed

cat > /tmp/minio-seed/sample-chat-image.txt << 'EOF'
This is a placeholder for a chat attachment image.
Real content: avatar.jpg, photo.png, document.pdf, etc.
Created: Loop 202
EOF

cat > /tmp/minio-seed/sample-recording.txt << 'EOF'
Meeting Recording Placeholder
- Duration: 60 seconds (demo)
- Format: mp4
- Tenant: apexfintech
- Recording ID: demo-rec-001
EOF

cat > /tmp/minio-seed/sample-landing-asset.txt << 'EOF'
Landing page asset (demo).
- hero-banner.jpg
- logo.svg
- icon-pack.zip
EOF

cat > /tmp/minio-seed/sample-model.txt << 'EOF'
ML model artifact placeholder.
- Model: lead_scoring_v1
- Size: ~50MB (mocked)
- TensorFlow SavedModel format
EOF

cat > /tmp/minio-seed/sample-tenant-logo.txt << 'EOF'
Tenant branding logos.
- apexfintech.png (200x60 PNG)
- hct-consulting.png (200x60 PNG)
- demo-company.png (200x60 PNG)
EOF

cat > /tmp/minio-seed/sample-email.txt << 'EOF'
Sample email attachment.
- contract.pdf
- invoice.pdf
EOF

# ============================================
# Upload files
# ============================================
echo "[3/4] Uploads..."

# chat-attachments: per-tenant
for tenant in apexfintech hct-consulting demo-company; do
  mc cp /tmp/minio-seed/sample-chat-image.txt ${MC_ALIAS}/chat-attachments/${tenant}/README.txt >/dev/null
done

# recordings: per-tenant meeting recordings
for i in 1 2 3; do
  mc cp /tmp/minio-seed/sample-recording.txt ${MC_ALIAS}/recordings/apexfintech/meeting-${i}.mp4.txt >/dev/null
  mc cp /tmp/minio-seed/sample-recording.txt ${MC_ALIAS}/recordings/hct-consulting/meeting-${i}.mp4.txt >/dev/null
  mc cp /tmp/minio-seed/sample-recording.txt ${MC_ALIAS}/recordings/demo-company/meeting-${i}.mp4.txt >/dev/null
done

# landing-assets (public-read)
mc cp /tmp/minio-seed/sample-landing-asset.txt ${MC_ALIAS}/landing-assets/README.txt >/dev/null
mc anonymous set download ${MC_ALIAS}/landing-assets >/dev/null 2>&1 || true

# model-artifacts
mc cp /tmp/minio-seed/sample-model.txt ${MC_ALIAS}/model-artifacts/lead_scoring/v1/README.txt >/dev/null

# tenant-logos (public-read)
for tenant in apexfintech hct-consulting demo-company; do
  mc cp /tmp/minio-seed/sample-tenant-logo.txt ${MC_ALIAS}/tenant-logos/${tenant}/README.txt >/dev/null
done
mc anonymous set download ${MC_ALIAS}/tenant-logos >/dev/null 2>&1 || true

# email-attachments
mc cp /tmp/minio-seed/sample-email.txt ${MC_ALIAS}/email-attachments/README.txt >/dev/null

# temp-uploads (one demo entry)
mc cp /tmp/minio-seed/sample-chat-image.txt ${MC_ALIAS}/temp-uploads/demo-upload.txt >/dev/null

# export-backups
cat > /tmp/minio-seed/sample-export.txt << 'EOF'
DB export placeholder (Loop 202 seed).
EOF
mc cp /tmp/minio-seed/sample-export.txt ${MC_ALIAS}/export-backups/demo-backup.txt >/dev/null

echo "  Uploaded ~25 sample objects"

# ============================================
# Set retention policies for some buckets
# ============================================
echo "[4/4] Lifecycle policies..."
# temp-uploads: 7-day expiration
mc ilm add ${MC_ALIAS}/temp-uploads --expire-days 7 >/dev/null 2>&1 || true
mc ilm add ${MC_ALIAS}/email-attachments --expire-days 90 >/dev/null 2>&1 || true
mc ilm add ${MC_ALIAS}/recordings --transition-tier "warm" --transition-days 30 >/dev/null 2>&1 || true

echo ""
echo "=== MinIO seed complete ==="
echo "Buckets created: ${#BUCKETS[@]}"
echo "Sample objects uploaded: ~25"
echo "Lifecycle rules applied: 3"
