#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Zoop One — WireGuard Setup Script
# Run this once as your user with sudo privileges to install all required tools
# ─────────────────────────────────────────────────────────────────────────────

set -e

echo ""
echo "══════════════════════════════════════════════════"
echo "  Zoop One — WireGuard & Network Tools Installer"
echo "══════════════════════════════════════════════════"
echo ""

# 1. Update package list
echo "[1/5] Updating package lists..."
sudo apt-get update -qq

# 2. Install WireGuard kernel module + userspace tools
echo "[2/5] Installing wireguard and wireguard-tools..."
sudo apt-get install -y wireguard wireguard-tools

# 3. Install iptables (needed for NAT masquerade in wg config PostUp/PostDown)
echo "[3/5] Installing iptables..."
sudo apt-get install -y iptables

# 4. Enable IP forwarding (required for traffic to route through the WG node)
echo "[4/5] Enabling IP forwarding..."
sudo sysctl -w net.ipv4.ip_forward=1
# Make it persistent across reboots
if ! grep -q "net.ipv4.ip_forward = 1" /etc/sysctl.conf; then
    echo "net.ipv4.ip_forward = 1" | sudo tee -a /etc/sysctl.conf > /dev/null
fi

# 5. Allow app to run wg-quick and wg without a password prompt each time
# This adds a sudoers rule for wireguard commands specifically
echo "[5/5] Configuring passwordless sudo for wg and wg-quick..."
SUDOERS_LINE="$USER ALL=(ALL) NOPASSWD: /usr/bin/wg, /usr/bin/wg-quick"
SUDOERS_FILE="/etc/sudoers.d/zoop-wireguard"
if [ ! -f "$SUDOERS_FILE" ]; then
    echo "$SUDOERS_LINE" | sudo tee "$SUDOERS_FILE" > /dev/null
    sudo chmod 440 "$SUDOERS_FILE"
    echo "  → Sudoers rule written to $SUDOERS_FILE"
else
    echo "  → Sudoers rule already exists at $SUDOERS_FILE"
fi

echo ""
echo "══════════════════════════════════════════════════"
echo "  ✓ WireGuard setup complete!"
echo ""
echo "  Installed:"
echo "    - wireguard (kernel module)"
echo "    - wireguard-tools (wg, wg-quick)"
echo "    - iptables (for NAT routing)"
echo ""
echo "  Configuration:"
echo "    - IP forwarding: ENABLED (persistent)"
echo "    - Passwordless sudo for wg/wg-quick: CONFIGURED"
echo ""
echo "  You can now start the Zoop One app and press"
echo "  'Start Sharing' to activate the WireGuard tunnel."
echo "══════════════════════════════════════════════════"
echo ""

# Verify installation
echo "Verifying tools..."
wg --version && echo "  ✓ wg found: $(which wg)"
wg-quick --version 2>&1 | head -1 && echo "  ✓ wg-quick found: $(which wg-quick)"
echo ""
