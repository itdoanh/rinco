#!/usr/bin/env bash
# ============================================================
# WireGuard mesh onboarding (RINCO)
# ============================================================
# Auto-onboards a tenant VPS into the secure admin / data mesh.
#
# Usage:
#   setup-mesh.sh <tenant_slug> <vps_public_ip> [ssh_user]
# ============================================================

set -euo pipefail

SLUG="${1:?Usage: $0 <tenant_slug> <vps_public_ip> [ssh_user]}"
VPS_IP="${2:?missing VPS public IP}"
SSH_USER="${3:-root}"
WG_DIR="$(cd "$(dirname "$0")" && pwd)"
TENANT_INDEX=$(($(echo "${SLUG}" | md5sum | cut -c1-2 | sed 's/^0*//') + 2))  # deterministic /24 hostid
PRIVATE_KEY=$(wg genkey)
PUBLIC_KEY=$(echo "${PRIVATE_KEY}" | wg pubkey)

SERVER_PUB=$(cat "${WG_DIR}/server.pub" 2>/dev/null || true)
if [ -z "${SERVER_PUB}" ]; then
  echo "Missing ${WG_DIR}/server.pub — export the control-plane public key first." >&2
  exit 1
fi

PEER_CONFIG="${WG_DIR}/peer.conf.template"
RENDER=$(sed -e "s|\${TENANT_PRIVATE_KEY}|${PRIVATE_KEY}|" \
             -e "s|\${SERVER_PUBLIC_KEY}|${SERVER_PUB}|" \
             -e "s|\${SERVER_ENDPOINT}|${VPS_IP}|" \
             -e "s|\${INDEX}|${TENANT_INDEX}|" \
             "${PEER_CONFIG}")

echo "==> Installing WireGuard on ${VPS_IP}"
ssh -o StrictHostKeyChecking=no "${SSH_USER}@${VPS_IP}" <<EOF
set -euo pipefail
apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y wireguard
install -d -m 0700 /etc/wireguard
cat > /etc/wireguard/wg0.conf <<'WG0'
${RENDER}
WG0
chmod 600 /etc/wireguard/wg0.conf
systemctl enable --now wg-quick@wg0
EOF

echo "==> Adding peer to server config"
cat >> "${WG_DIR}/wg0.conf" <<EOF

# peer ${SLUG}
[Peer]
PublicKey = ${PUBLIC_KEY}
AllowedIPs = 10.13.13.${TENANT_INDEX}/32,172.25.0.0/16
PersistentKeepalive = 25
EOF

wg set wg0 peer "${PUBLIC_KEY}" allowed-ips "10.13.13.${TENANT_INDEX}/32,172.25.0.0/16" || \
  wg-quick strip wg0 > /etc/wireguard/wg0.conf.new
wg syncconf wg0 <(wg-quick strip wg0) || true

echo "✓ Tenant ${SLUG} joined the mesh at 10.13.13.${TENANT_INDEX}"
