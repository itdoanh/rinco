#!/usr/bin/env bash
# ============================================================
# RINCO WS-E — Secret rotation script
# ============================================================
#
# What it does (best-effort, idempotent, no destructive default):
#
#   1. PASETO signing key rotation with a configurable overlap
#      period so tokens issued by the previous key remain valid
#      until they expire.
#   2. SMTP password rotation: generates a new password, updates
#      the K8s secret (or local .env stub), and prints the
#      diff for review.
#   3. K8s secrets stub update: bumps the secret version label
#      and triggers a rolling restart annotation so pods pick
#      up the new env values.
#
# Usage:
#
#   scripts/rotate-secrets.sh paseto [--overlap 24h] [--env production]
#   scripts/rotate-secrets.sh smtp   [--overlap 1h]
#   scripts/rotate-secrets.sh all    [--overlap 24h] [--env production] [--dry-run]
#
# The script does NOT call any provider APIs by default.  When
# run with --apply it would dispatch to your secret manager
# (Vault / AWS SM / K8s).  By default it prints the commands
# that would be run, plus a JSON manifest, for review.

set -euo pipefail

# ----- Defaults ----------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
ENVIRONMENT="${RINCO_ENV:-staging}"
NAMESPACE="${RINCO_NAMESPACE:-rinco}"
OVERLAP="24h"
DRY_RUN=1
APPLY=0
SECRET_DIR="${RINCO_SECRET_DIR:-${REPO_ROOT}/infra/k8s/base/secrets}"
KUBECTL="${KUBECTL:-kubectl}"
LOG_PREFIX="[rotate-secrets]"

# ----- Args --------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
  case "$1" in
    paseto|smtp|all) TARGET="$1"; shift ;;
    --overlap) OVERLAP="$2"; shift 2 ;;
    --env) ENVIRONMENT="$2"; shift 2 ;;
    --namespace) NAMESPACE="$2"; shift 2 ;;
    --apply) APPLY=1; DRY_RUN=0; shift ;;
    --dry-run) DRY_RUN=1; APPLY=0; shift ;;
    -h|--help)
      grep -E '^#( |$)' "$0" | sed -e 's/^# \{0,1\}//' | head -40
      exit 0
      ;;
    *) echo "${LOG_PREFIX} unknown arg: $1" >&2; exit 64 ;;
  esac
done
TARGET="${TARGET:-all}"

log()  { echo "${LOG_PREFIX} $*"; }
fail() { echo "${LOG_PREFIX} ERROR: $*" >&2; exit 1; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

require_cmd openssl
require_cmd awk
require_cmd date

# ----- Helpers -----------------------------------------------------------------
timestamp() { date -u +"%Y%m%dT%H%M%SZ"; }

# generate_key_hex <bytes> -> prints hex string
generate_key_hex() {
  local bytes="${1:-32}"
  openssl rand -hex "${bytes}"
}

# generate_password <length>
generate_password() {
  local len="${1:-32}"
  openssl rand -base64 "$((len * 3 / 4))" | tr -dc 'A-Za-z0-9' | head -c "$len"
}

# apply_or_echo <command...>
apply_or_echo() {
  if [[ "${APPLY}" -eq 1 ]]; then
    log "APPLY: $*"
    "$@"
  else
    log "DRY-RUN: $*"
  fi
}

# ----- 1. PASETO rotation ------------------------------------------------------
rotate_paseto() {
  log "rotating PASETO signing key (env=${ENVIRONMENT}, overlap=${OVERLAP})"

  local new_id new_hex previous_id previous_hex new_at overlap_at
  new_id="paseto-$(timestamp)"
  new_hex="$(generate_key_hex 32)"  # 32 bytes = 256-bit symmetric PASETO key
  new_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

  # Read the previous key (if any) from the K8s secret or local stub.
  previous_id=""
  previous_hex=""
  if [[ -f "${SECRET_DIR}/paseto-current.json" ]]; then
    previous_id="$(awk -F'"' '/"id"/ {print $4; exit}' "${SECRET_DIR}/paseto-current.json" || true)"
    previous_hex="$(awk -F'"' '/"key"/ {print $4; exit}' "${SECRET_DIR}/paseto-current.json" || true)"
  fi

  # Compute overlap expiry (overlap-period after rotation).
  overlap_at="$(date -u -d "+${OVERLAP}" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null \
    || date -u +"%Y-%m-%dT%H:%M:%SZ")"

  # Write new manifest (would be a Vault write in production).
  cat > "${SECRET_DIR}/paseto-current.json.new" <<EOF
{
  "id":        "${new_id}",
  "key":       "${new_hex}",
  "created_at": "${new_at}",
  "overlap_until": "${overlap_at}",
  "previous": {
    "id":         "${previous_id}",
    "key":        "${previous_hex}",
    "rotated_out": "${overlap_at}"
  },
  "env": "${ENVIRONMENT}"
}
EOF

  if [[ "${APPLY}" -eq 1 ]]; then
    mv "${SECRET_DIR}/paseto-current.json.new" \
       "${SECRET_DIR}/paseto-current.json"
    log "wrote ${SECRET_DIR}/paseto-current.json"
  else
    log "would have written ${SECRET_DIR}/paseto-current.json.new"
  fi

  # Patch the K8s secret (would be `kubectl create secret generic -n
  # $NAMESPACE rinco-auth-paseto ... --from-literal=current=$new_hex ...`).
  apply_or_echo "${KUBECTL}" -n "${NAMESPACE}" \
    create secret generic "rinco-auth-paseto-${new_id}" \
    --from-literal="current=${new_hex}" \
    --from-literal="previous=${previous_hex}" \
    --from-literal="current_id=${new_id}" \
    --from-literal="previous_id=${previous_id}" \
    --from-literal="overlap_until=${overlap_at}" \
    --dry-run=client -o yaml \
    | ${KUBECTL} -n "${NAMESPACE}" apply -f - || true

  # Bump the version label so deployment pods re-read the secret.
  apply_or_echo "${KUBECTL}" -n "${NAMESPACE}" \
    annotate secret "rinco-auth-paseto-${new_id}" \
    "rinco.app/secret-version=${new_id}" \
    --overwrite

  log "PASETO rotation complete (new=${new_id}, overlap until ${overlap_at})"
}

# ----- 2. SMTP password rotation -----------------------------------------------
rotate_smtp() {
  log "rotating SMTP password (env=${ENVIRONMENT})"

  local new_password
  new_password="$(generate_password 32)"

  cat > "${SECRET_DIR}/smtp-current.json.new" <<EOF
{
  "host":      "smtp.${ENVIRONMENT}.rinco.app",
  "port":      587,
  "username":  "noreply@rinco.app",
  "password":  "${new_password}",
  "rotated_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF

  if [[ "${APPLY}" -eq 1 ]]; then
    mv "${SECRET_DIR}/smtp-current.json.new" \
       "${SECRET_DIR}/smtp-current.json"
  else
    log "would have written ${SECRET_DIR}/smtp-current.json.new"
  fi

  apply_or_echo "${KUBECTL}" -n "${NAMESPACE}" \
    create secret generic "rinco-email-smtp" \
    --from-literal="password=${new_password}" \
    --dry-run=client -o yaml \
    | ${KUBECTL} -n "${NAMESPACE}" apply -f - || true

  apply_or_echo "${KUBECTL}" -n "${NAMESPACE}" \
    annotate secret "rinco-email-smtp" \
    "rinco.app/secret-version=$(timestamp)" \
    --overwrite

  log "SMTP rotation complete (password length=${#new_password})"
}

# ----- 3. K8s secrets stub ------------------------------------------------------
# Bumps the version label on every rinco-managed secret so a fresh
# rolling restart picks up the new values.

bump_k8s_secrets() {
  log "bumping K8s secret versions (namespace=${NAMESPACE})"
  local ts version
  ts="$(timestamp)"
  version="rinco.app/secret-version=${ts}"

  local secrets
  secrets="$( ${KUBECTL} -n "${NAMESPACE}" get secret -l 'rinco.app/managed-by=rinco' \
    -o jsonpath='{range .items[*]}{.metadata.name}{" "}{end}' 2>/dev/null || true)"

  if [[ -z "${secrets}" ]]; then
    log "no rinco-managed secrets found in namespace ${NAMESPACE}"
    return 0
  fi

  for s in ${secrets}; do
    apply_or_echo "${KUBECTL}" -n "${NAMESPACE}" annotate secret "${s}" \
      "${version}" --overwrite
  done

  log "annotated $(echo ${secrets} | wc -w) secrets with version ${ts}"
}

# ----- Dispatch ----------------------------------------------------------------
case "${TARGET}" in
  paseto)
    rotate_paseto
    ;;
  smtp)
    rotate_smtp
    ;;
  all)
    rotate_paseto
    rotate_smtp
    bump_k8s_secrets
    ;;
  *)
    fail "unknown target: ${TARGET}"
    ;;
esac

log "done (apply=${APPLY}, dry-run=${DRY_RUN}, env=${ENVIRONMENT})"