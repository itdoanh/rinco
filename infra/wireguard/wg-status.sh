# ============================================================
# WireGuard mesh status
# ============================================================
# Print the current state of the WireGuard mesh.
# ============================================================

set -euo pipefail

WG_IFACE="${1:-wg0}"

echo "==> WireGuard mesh status (${WG_IFACE})"
echo ""
wg show "${WG_IFACE}" || echo "Interface ${WG_IFACE} not running"

echo ""
echo "==> Latest handshake (per peer):"
wg show "${WG_IFACE}" latest-handshakes 2>/dev/null || true

echo ""
echo "==> Transfer counters (per peer):"
wg show "${WG_IFACE}" transfer 2>/dev/null || true
