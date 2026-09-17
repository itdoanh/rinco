# ============================================================
# WireGuard mesh — add peer to existing node
# ============================================================
# Adds a single peer to the wg0 interface without restarting.
# Useful for runtime onboarding of new nodes.
#
# Usage:
#   wg-add-peer.sh <peer-public-key> <peer-ip> [endpoint]
# ============================================================

set -euo pipefail

PEER_KEY="${1:?missing peer pubkey}"
PEER_IP="${2:?missing peer IP}"
PEER_ENDPOINT="${3:-}"

WG_IFACE="wg0"

if [ -n "${PEER_ENDPOINT}" ]; then
    wg set "${WG_IFACE}" peer "${PEER_KEY}" \
        allowed-ips "${PEER_IP}/32,10.200.0.0/24" \
        endpoint "${PEER_ENDPOINT}" \
        persistent-keepalive 25
else
    wg set "${WG_IFACE}" peer "${PEER_KEY}" \
        allowed-ips "${PEER_IP}/32,10.200.0.0/24" \
        persistent-keepalive 25
fi

# Persist by saving config
wg-quick save "${WG_IFACE}" 2>/dev/null || \
    wg setconf "${WG_IFACE}" <(wg-quick strip "${WG_IFACE}")

echo "✓ Peer ${PEER_IP} added to ${WG_IFACE}"
