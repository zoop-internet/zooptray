# 🖥️ Zoop One

**Zoop One** is the official desktop and laptop node daemon for the **Zoop Network**. It is a lightweight, background-running system tray application that empowers everyday computer owners to participate as active network providers by sharing their local internet connection securely.

---

## ⚡ Core Concept

While the **Zoop Console** serves as the administrative dashboard for managing accounts, viewing global analytics, and configuring remote policies, **Zoop One** is the operational engine that runs directly on the provider's hardware. It acts as the local system service that handles the raw connection routing, visibility broadcasting, and local policy enforcement.

```text
  ┌─────────────────────────────────────────────────────────────┐
  │                         ZOOP ONE                            │
  │                     (Local Desktop App)                     │
  │                                                             │
  │   - Runs in System Tray         - Creates Shared WiFi Node  │
  │   - Enforces Sharing Limits     - Broadcasts State to Cloud │
  └──────────────────────────────┬──────────────────────────────┘
                                 │
                                 ▼ (Telemetry & Auth API)
  ┌─────────────────────────────────────────────────────────────┐
  │                       ZOOP NETWORK                          │
  │                       (Cloud Core)                          │
  └──────────────────────────────┬──────────────────────────────┘
                                 │
                                 ▼ (Web Management Interface)
  ┌─────────────────────────────────────────────────────────────┐
  │                       ZOOP CONSOLE                          │
  │                    (Admin Dashboard)                        │
  └─────────────────────────────────────────────────────────────┘
```

---

## 🚀 Key Features

*   **Zero-Disruption Background Operation**: Operates silently inside the system tray (macOS Menu Bar, Windows Taskbar, Linux System Tray) without cluttering the screen or impacting daily productivity.
*   **Cross-Platform Support**: Now supports Linux, macOS, and Windows with platform-specific WireGuard integration and auto-start functionality.
*   **Local Node Provisioning**: Creates and manages a local WireGuard interface for Zoop client routing on all supported platforms.
*   **Automatic Dependency Management**: Installs required WireGuard tools automatically on Linux (via apt/dnf/pacman) and provides clear installation instructions for macOS (Homebrew) and Windows.
*   **Platform-Specific NAT Configuration**: Uses iptables on Linux, pf on macOS, and provides guidance for Windows netsh configuration.
*   **Hardware Policy Enforcement (planned)**: Sharing limits, firewall isolation policy, and bandwidth caps are UI/config concepts until OS-level enforcement is wired in.
*   **Automatic Node Discovery (planned)**: Cloud discovery and telemetry broadcasting are not active in the current desktop build.
*   **Offline Fallbacks (planned)**: Battery and upstream-link shutdown policies are documented goals, not active runtime behavior yet.
*   **Zero-Config Handshake**: Generates a secure cryptographic identity key during installation to pair with the Zoop Console instantly.

---

## 🌍 Cross-Platform Support

Zoop One now supports Linux, macOS, and Windows with platform-specific optimizations:

### Platform Requirements

| Platform | WireGuard Installation | Auto-Start Method | NAT Configuration |
|----------|----------------------|------------------|-------------------|
| **Linux** | `wireguard-tools` (auto-installed) | `.desktop` file | iptables |
| **macOS** | `wireguard-tools` via Homebrew | launchd plist | pf (Packet Filter) |
| **Windows** | WireGuard MSI installer | Startup folder | netsh (manual setup) |

### Platform-Specific Setup

#### Linux (Debian/Ubuntu/Fedora/Arch)
```bash
# Automatic setup (handled by the application)
sudo apt install wireguard wireguard-tools  # Debian/Ubuntu
sudo dnf install wireguard-tools           # Fedora/RHEL
sudo pacman -S wireguard-tools              # Arch Linux
```

#### macOS
```bash
# Install WireGuard tools via Homebrew
brew install wireguard-tools

# The application will automatically create a launchd plist for auto-start
```

#### Windows
1. Download and install WireGuard from https://www.wireguard.com/install/
2. The application will create a startup shortcut for auto-launch
3. NAT configuration may require manual setup using `netsh` or PowerShell

---

## ⚙️ Configuration & Customization

All local parameters are configured via a clean settings popup in the system tray or stored locally in a config manifest. These settings determine how the node presents itself to the Zoop ecosystem:

### Settings & Controls
*   **Node Identity**: Customize the display name of your node (e.g., `Node-Alpha-7`).
*   **Visibility**: Toggle the node visibility status between `Public` (discoverable by any Zoop user) and `Private` (restricted to specific trusted identities).
*   **Bandwidth Control**: Enforce hard limits on total shared network bandwidth (e.g., limit sharing to `68%` of your max speeds).
*   **Client Management**: Define the maximum number of concurrent active users allowed to route through your node.
*   **System Auto-Start**: Configure the application to launch automatically on system boot.

---

## 🛠️ Typical Configuration Schema (`config.json`)

The background daemon reads its operational rules from a local JSON settings file. Below is an example schema:

```json
{
  "node": {
    "id": "node-alpha-7",
    "displayName": "Node-Alpha-7",
    "privateKeyPath": "~/.zoop/identity.key",
    "isPublic": true
  },
  "sharing": {
    "enabled": true,
    "maxClients": 10,
    "bandwidthCapPercent": 68,
    "routingPolicy": "balanced"
  },
  "system": {
    "autoStartOnBoot": true,
    "batteryThresholdShutdownPercent": 20,
    "preferredInterface": "wlan0"
  },
  "api": {
    "endpoint": "https://api.zoopnetwork.com/v1",
    "heartbeatIntervalSeconds": 15
  }
}
```

---

## 🔒 Security & Sandboxing

Zoop One is designed with a strict security-first architecture to protect node hosts across all platforms:
1.  **WireGuard Tunnel**: Guest traffic uses WireGuard encryption while the local tunnel is active on all supported platforms.
2.  **Platform-Specific Firewall Integration**:
    - **Linux**: iptables for NAT and basic isolation
    - **macOS**: pf (Packet Filter) for NAT configuration
    - **Windows**: netsh integration with manual setup guidance
3.  **Cryptographic Identity Verification (planned)**: Backend-verified client authorization is not active in this local desktop build yet.
4.  **Cross-Platform Key Management**: Secure storage of WireGuard keys in platform-appropriate locations (`~/.zoop/` on Unix, AppData on Windows).

---

## 🤝 Interaction with Zoop Console

*   **Monitoring (planned)**: The desktop app currently shows local WireGuard status; console integration is pending.
*   **Analytics (planned)**: Historical bandwidth allocation charts, revenue statistics, and cloud-forwarded device logs are not active in this release.
