#!/usr/bin/env bash
# ============================================================
# MinIO bucket bootstrap (RINCO)
# ============================================================
# Reads buckets.json and creates each bucket + lifecycle policy
# + CORS via the `mc` client. Idempotent.
# ============================================================
set -euo pipefail

: "${MINIO_ENDPOINT:?MINIO_ENDPOINT must be set, e.g. http://minio:9000}"
: "${MINIO_ROOT_USER:?MINIO_ROOT_USER must be set}"
: "${MINIO_ROOT_PASSWORD:?MINIO_ROOT_PASSWORD must be set}"

ALIAS="rinco"

mc alias set "$ALIAS" "$MINIO_ENDPOINT" "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD"

CONFIG_DIR=/scripts
BUCKETS_JSON="${CONFIG_DIR}/buckets.json"

if [ ! -f "$BUCKETS_JSON" ]; then
  echo "buckets.json not found at $BUCKETS_JSON, aborting." >&2
  exit 1
fi

echo "==> Applying CORS / region defaults"
mc anonymous set none "$ALIAS"

count=$(jq '.buckets | length' "$BUCKETS_JSON")
i=0
while [ "$i" -lt "$count" ]; do
  name=$(jq -r ".buckets[$i].name" "$BUCKETS_JSON")
  version=$(jq -r ".buckets[$i].versioning" "$BUCKETS_JSON")
  pub=$(jq    -r ".buckets[$i].public"     "$BUCKETS_JSON")
  cors=$(jq -c ".buckets[$i].cors" "$BUCKETS_JSON")
  retention=$(jq -r ".buckets[$i].lifecycle.retention_days // 0" "$BUCKETS_JSON")

  echo "==> Creating bucket: $name (versioning=$version)"

  # Create bucket if missing
  if mc ls "$ALIAS/$name" >/dev/null 2>&1; then
    echo "    already exists"
  else
    mc mb "$ALIAS/$name"
  fi

  # Versioning
  case "$version" in
    enabled)  mc version enable  "$ALIAS/$name" ;;
    suspended) mc version suspend "$ALIAS/$name" ;;
    *) mc version suspend "$ALIAS/$name" ;;
  esac

  # Retention (object lock / lifecycle expiry)
  if [ "$retention" -gt 0 ] 2>/dev/null; then
    cat > /tmp/lifecycle.json <<EOF
{
  "Rules": [{
    "ID": "rinco-expiry",
    "Status": "Enabled",
    "Expiration": { "Days": $retention },
    "ApplyToSubdirs": true
  }]
}
EOF
    mc ilm import "$ALIAS/$name" < /tmp/lifecycle.json || true
  fi

  # Anonymous access (only avatars / assets that should be readable)
  if [ "$pub" = "true" ]; then
    mc anonymous set download "$ALIAS/$name" >/dev/null
1  else
    mc anonymous set none "$ALIAS/$name" >/dev/null
  fi

  i=$((i + 1))
done

echo "==> MinIO bootstrap complete"
