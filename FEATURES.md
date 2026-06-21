# Zoop Desktop features & Specifications

This document lists all features, assets, and styling components integrated into the Zoop Desktop tray application.

---

## 1. Application Layout & Window Frame Controls
- **Cross-Platform Frameless Actions**: Custom glassmorphic titlebar header featuring interactive control buttons that bind directly to Wails runtime APIs:
  - **Minimize**: Minimizes the window to the OS taskbar/dock via `runtime.WindowMinimise()`.
  - **Maximize/Restore**: Toggles window state between maximized and normal dimensions via `runtime.WindowToggleMaximise()`.
  - **Close / Termination**: Safely shuts down the background routing daemon and exits the application using `runtime.Quit()`.
- **Theme Switcher**: Header toggle button to transition between Dark and Light color palettes dynamically.

---

## 2. Sidebar Navigation Panel
- **Menu Options**: Restructured to match the reference design:
  - **Dashboard**: High-level real-time performance summary.
  - **Sharing**: Granular control of routing states, caps, and connected clients.
  - **Earnings**: Coming-soon details and reward structures.
  - **Settings**: System profiles and auto-start policies.
  - **About**: Credits, local directories, and live console logs.
- **Node Connection Card**:
  - Displays connection dot indicator: Green (Online/Active) or Red (Offline).
  - Version identifier badge (`v1.0.0`).
  - "Check for updates" checker.

---

## 3. Node Dashboard (`Dashboard.jsx`)
- **Interactive SVG Status Ring**:
  - Glowing circular ring indicating online state (green) or offline state (muted slate).
  - Toggles sharing on click.
- **Overview Metadata Grid**:
  - Displays Node Cryptographic ID.
  - Interactive ticking Uptime timer (`Xd Xh Xm`).
  - Active Peers Count (routing clients).
  - Host IP Address.
- **Live SVG Sparkline Cards**:
  - **Bandwidth Shared**: Rolling chart mapping combined speed metrics (Upload + Download).
  - **Data Shared Today**: Trend line mapping total gigabytes routed today.
  - **System Status Badge**: Glow indicator for node health (Healthy/Offline).

---

## 4. Sharing Control Console (`Sharing.jsx`)
- **Hotkey Switch**: Simple check switch to toggle sharing on the fly.
- **Configurable Speed Caps**:
  - Dropdown limiters for Max Upload Speed (`10 Mbps` to `Unlimited`).
  - Dropdown limiters for Max Download Speed (`50 Mbps` to `Unlimited`).
- **Identity Widgets**:
  - Host IP display card with instant "Copy to Clipboard" success checks.
  - Active Uptime ticking widget.
- **Client Management Table**:
  - List of active connected guest devices.
  - Shows device name, MAC address, IP address, connection time, speed, and data throughput.
  - **Kick Actions**: Professional "Kick" button to disconnect specific clients.

---

## 5. Earnings Panel (`Earnings.jsx`)
- **Reward Wallet Placeholder**:
  - Solid purple wallet icon container.
  - Description details regarding reward distribution.
  - "Learn more" action that launches `https://zoopnetwork.com` in the default OS web browser.

---

## 6. System settings Panel (`Settings.jsx`)
- **Node Identity**: Interactive Display Name editor with strict slug generation.
- **Visibility Options**: Select cards to toggle between Public (discoverable) and Private (cryptographically hidden) profiles.
- **Hardware Configurator**:
  - Interactive HTML5 slider capping allocated network speed percent.
  - Client count boundary editor.
  - Network Interface selector loading system interface configurations (e.g. `wlan0`, `eth0`).
- **Auto-Start Switch**: Toggle to register/deregister the daemon on system boot.

---

## 7. About & Diagnostics Panel (`About.jsx`)
- **Brand Info Card**: Displays application description, version index, and credits.
- **Local Directories**: Lists secure locations where identity keys and API endpoints are loaded.
- **Collapsible Diagnostic Console Log Stream**:
  - Toggles log pane visibility.
  - Live log streaming with category filters (ALL, INFO, WARNING, ERROR).
  - Severity-colored rows (green for info, yellow for warnings, red for errors) with timestamps.

---

## 8. Styling Engine (`style.css`)
- **Palette**: Dark obsidian styling with solid brand colors:
  - `--black`: `#000000`
  - `--surface`: `#070a0d`
  - `--surface-card`: `#0c1015`
  - `--blue`: `#126cff`
  - `--green`: `#14f06d`
  - `--cyan`: `#08f2ff`
  - `--lime`: `#b9ff00`
  - `--muted`: `#9aa8b8`
- **Typography**: Primary fallback stack prioritizing `"Google Sans"`, `"Inter"`, and system UI.
- **Interactive States**: Hover effects, responsive flexbox margins, glowing status badges, and rounded input borders.
