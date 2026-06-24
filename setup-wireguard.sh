#!/usr/bin/env bash
# Zoop One WireGuard setup helper.
# Run as your normal user; sudo is requested only for package/system changes.

set -euo pipefail

echo ""
echo "=================================================="
echo "  Zoop One - WireGuard & Network Tools Installer"
echo "=================================================="
echo ""

need_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "Missing required command: $1" >&2
        exit 1
    fi
}

install_packages() {
    if command -v apt-get >/dev/null 2>&1; then
        echo "[1/5] Updating package lists..."
        sudo apt-get update -qq
        echo "[2/5] Installing WireGuard, iptables, and sudo helpers..."
        sudo apt-get install -y wireguard wireguard-tools iptables sudo
    elif command -v dnf >/dev/null 2>&1; then
        echo "[1/5] Refreshing package metadata..."
        sudo dnf makecache -q
        echo "[2/5] Installing WireGuard, iptables, and sudo helpers..."
        sudo dnf install -y wireguard-tools iptables sudo
    elif command -v pacman >/dev/null 2>&1; then
        echo "[1/5] Syncing package metadata..."
        sudo pacman -Sy --noconfirm
        echo "[2/5] Installing WireGuard, iptables, and sudo helpers..."
        sudo pacman -S --noconfirm wireguard-tools iptables sudo
    else
        echo "Unsupported package manager. Install wireguard-tools, iptables, and sudo manually." >&2
        exit 1
    fi
}

echo "[0/5] Checking prerequisites..."
need_cmd sudo
install_packages

echo "[3/5] Enabling IPv4 forwarding..."
sudo sysctl -w net.ipv4.ip_forward=1 >/dev/null
SYSCTL_FILE="/etc/sysctl.d/99-zoop-wireguard.conf"
printf '%s\n' 'net.ipv4.ip_forward = 1' | sudo tee "$SYSCTL_FILE" >/dev/null
sudo sysctl --system >/dev/null

echo "[4/5] Locating WireGuard commands..."
# Use verified absolute paths to prevent PATH hijacking
for cmd_path in "/usr/bin/wg" "/usr/bin/wg-quick" "/usr/sbin/sysctl"; do
    if [ ! -x "$cmd_path" ]; then
        # Fallback: attempt PATH resolution if the standard path is missing
        resolved="$(command -v "${cmd_path##*/}" || true)"
        if [ -z "$resolved" ]; then
            echo "${cmd_path##*/} could not be found after installation." >&2
            exit 1
        fi
        eval "$(echo "${cmd_path##*/}" | tr 'a-z-' 'A-Z_')_PATH=$resolved"
    fi
done
WG_PATH="${WG_PATH:-/usr/bin/wg}"
WG_QUICK_PATH="${WG_QUICK_PATH:-/usr/bin/wg-quick}"
SYSCTL_PATH="${SYSCTL_PATH:-/usr/sbin/sysctl}"

echo "[5/5] Configuring passwordless sudo for Zoop WireGuard commands..."
SUDOERS_FILE="/etc/sudoers.d/zoop-wireguard"
SUDOERS_LINE="$USER ALL=(ALL) NOPASSWD: $WG_PATH, $WG_QUICK_PATH, $SYSCTL_PATH"
TMP_SUDOERS="$(mktemp)"
printf '%s\n' "$SUDOERS_LINE" > "$TMP_SUDOERS"

if sudo visudo -cf "$TMP_SUDOERS" >/dev/null; then
    sudo install -m 0440 "$TMP_SUDOERS" "$SUDOERS_FILE"
else
    rm -f "$TMP_SUDOERS"
    echo "Generated sudoers rule failed validation; no sudoers changes were applied." >&2
    exit 1
fi
rm -f "$TMP_SUDOERS"

echo ""
echo "=================================================="
echo "  WireGuard setup complete"
echo ""
echo "  Installed/configured:"
echo "    - wireguard-tools"
echo "    - iptables"
echo "    - persistent IPv4 forwarding at $SYSCTL_FILE"
echo "    - passwordless sudo for wg, wg-quick, and sysctl"
echo ""
echo "  The app uses those commands to create and manage"
echo "  the wg-zoop tunnel without prompting on each run."
echo "=================================================="
echo ""

echo "Verifying tools..."
"$WG_PATH" --version && echo "  wg: $WG_PATH"
"$WG_QUICK_PATH" --version 2>&1 | head -1 && echo "  wg-quick: $WG_QUICK_PATH"
echo "  sysctl: $SYSCTL_PATH"
echo ""
