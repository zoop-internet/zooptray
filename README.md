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
*   **Local Node Provisioning**: Turns any desktop or laptop network interface into a secure, access-controlled gateway hotspot for Zoop clients.
*   **Hardware Policy Enforcement**: Applies sharing limits, traffic rules, and bandwidth caps directly at the local OS firewall level to guarantee performance for the host computer.
*   **Automatic Node Discovery**: Continuously broadcasts local telemetry (signal strength, latency, device load) to the Zoop platform, making the node discoverable to nearby consumers looking for high-quality connections.
*   **Offline Fallbacks**: Automatically suspends sharing or shuts down connections if the host computer falls below configured battery thresholds or if the host's primary internet link disconnects.
*   **Zero-Config Handshake**: Generates a secure cryptographic identity key during installation to pair with the Zoop Console instantly.

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

Zoop One is designed with a strict security-first architecture to protect node hosts:
1.  **Isolated Guest Network**: Routed guest traffic is sandboxed and completely segregated from the host's local private intranet (LAN). Guest devices cannot communicate with the host's personal devices (e.g., home printers, local file shares, private smart TVs).
2.  **Encrypted Tunneling**: All data shared from consumer to provider node is wrapped inside a secure cryptographic tunnel, protecting guest privacy and preventing the host from seeing user-specific payload details.
3.  **Cryptographic Identity Verification**: Connections are authorized using a pre-authenticated handshake protocol verified through the Zoop Core backend. Unapproved devices are instantly blocked at the firewall.

---

## 🤝 Interaction with Zoop Console

*   **Monitoring**: While Zoop One does the heavy lifting of routing packets, the **Zoop Console** provides the real-time visual interface.
*   **Analytics**: Historical bandwidth allocation charts, revenue statistics, and device logs are logged securely by Zoop One and forwarded to the cloud, so you can explore them at your leisure on the web console.
