# ============================================================
# WireGuard mesh — remove peer
# ============================================================
# Usage:
#   wg-remove-peer.sh <peer-public-key>
# ============================================================

set -euo pipefail

PEER_KEY="${1:?missing peer pubkey}"
WG_IFACE="wg0"

wg set "${WG_IFACE}" peer "${PEER_KEY}" remove

wg-quick save "${WG_IFACE}" 2>/dev/null || \
    wg setconf "${WG_IFACE}" <(wg-quick strip "${WG_IFACE}")

echo "✓ Peer removed from ${WG_IFACE}"
