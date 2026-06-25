# Zoop One

Desktop node daemon for the Zoop Network — share your internet connection securely via WireGuard. Runs on Linux, macOS, and Windows.

## Architecture

```
  ┌─────────────────────────────────────────┐
  │              ZOOP ONE APP               │
  │  (Wails v2 + Go backend + React front)  │
  │                                         │
  │  ┌──────────┐  ┌─────────────────────┐  │
  │  │  Node     │  │  WireGuard Manager  │  │
  │  │  Service  │  │  (cross-platform)   │  │
  │  │           │  │                     │  │
  │  │ • Tunnel  │  │ • wg-quick (Linux)  │  │
  │  │ • Peers   │  │ • wireguard.exe     │  │
  │  │ • Config  │  │   (Windows)         │  │
  │  │ • Discov. │  │ • sudo/doas/pkexec  │  │
  │  └────┬──────┘  │   /systemd-run      │  │
  │       │         └─────────┬───────────┘  │
  └───────┼───────────────────┼──────────────┘
          │                   │
          ▼                   ▼
  ┌───────────────┐  ┌──────────────────┐
  │  Discovery    │  │  WireGuard TUN   │
  │  Worker       │  │  Interface       │
  │  (Cloudflare) │  │  (wg-zoop)       │
  │               │  │                  │
  │ • Register    │  │ • NAT / Routing  │
  │ • Heartbeat   │  │ • Peer mgmt      │
  │ • Discover    │  │ • Encrypted TUN  │
  │ • Connect     │  │                  │
  │   Relay       │  │                  │
  └───────────────┘  └──────────────────┘
```

## Features

### Core
- **WireGuard tunnel management** — Creates and manages `wg-zoop` interface with platform-specific NAT (iptables on Linux, pf on macOS, manual netsh on Windows)
- **Peer-to-peer internet sharing** — Share your connection so other Zoop nodes can route traffic through you
- **One-click Connect** — Connect to any discovered peer and route all traffic through their node
- **Client management** — View connected clients with real-time bandwidth, data usage, and kick/disconnect

### Peer Discovery
- **Cloudflare Worker backend** — Nodes register with a global discovery service at `zoop-peer-discovery.allan-info-comp.workers.dev`
- **Heartbeat keepalive** — Every 15s (configurable) to maintain registration
- **Display name propagation** — Name changes in Settings sync to discovery on next heartbeat
- **Automatic cleanup** — Nodes auto-unregister on graceful shutdown; KV TTL (60s) cleans up crashed nodes
- **Self-connection protection** — Can't connect to your own node (detected via public IP comparison)

### Privilege Escalation (Linux)
- **Automatic fallback chain**: `sudo` → `doas` → `pkexec` → `systemd-run --uid=0`
- Properly tries each method before giving up (fixed in v1.2.1)
- Clear error messages suggesting `sudo setcap cap_net_admin+ep $(which zoop-one)`

### Windows Support
- **`requireAdministrator` manifest** — App requests admin rights for WireGuard/service operations
- **Hidden console windows** — Subprocesses use `HideWindow` flag to avoid flashing cmd.exe windows
- **WireGuard detection** — Checks both `wg.exe` and `wireguard.exe` in Program Files and PATH
- **Default route detection** — Uses PowerShell `Get-NetRoute` with `netsh` fallback
- **Service state parsing** — Correctly reads `sc query` output (tab-delimited `STATE : 4 RUNNING`)
- **Autostart** — Windows registry via `HKCU\...\Run`, with existence check before deletion

### Configuration
- **Settings UI** — Display name, public/private visibility, bandwidth cap, max clients, routing policy, preferred network interface, heartbeat interval, auto-start on boot
- **Persistent config** — `~/.zoop/config.json` with JSON schema

### UI / Frontend
- **React + Wails** — Desktop-native UI with system tray integration
- **Dark/Light theme** — Persisted in localStorage
- **Real-time telemetry** — 2-second refresh of tunnel status, client list, bandwidth charts
- **Discovered Peers panel** — Always visible (even when local tunnel is offline)
- **EULA acceptance flow** — Windows-only first-run dialog

## Version History

| Tag | Date | Changes |
|-----|------|---------|
| v1.2.1 | — | Linux privilege fallback fix (real chain, systemd-run); stale node cleanup on tunnel failure; display name sync via heartbeat; connect relay for peer-to-peer routing; worker deployed with connect-request endpoints |
| v1.2.0 | — | Fixed `build/windows/wails.exe.manifest` (added `requireAdministrator`); added `systemd-run` to Linux fallback |
| v1.1.9 | — | `pkexec` and `doas` fallback for Linux when sudo has NoNewPrivileges |
| v1.1.8 | — | Removed `ListenPort` from client config; self-connection detection |
| v1.1.7 | — | Fixed build tag bug that broke Windows compilation |
| v1.1.6 | — | `ListenPort` added to client config; console windows hidden via `exec_windows.go` |
| v1.1.5 | — | `ConnectToPeer`/`DisconnectFromPeer` backend + "Connect" button UI |
| v1.1.4 | — | Discovered peers always visible; `GetPlatform` binding fix |
| v1.1.3 | — | WireGuard detection (check both `wg.exe` and `wireguard.exe`); default route detection via PowerShell |
| v1.1.2 | — | Autostart registry fix; `sc query` state parsing fix |
| v1.1.1 | — | Initial release |

## Platform Support

| Feature | Linux | macOS | Windows |
|---------|-------|-------|---------|
| WireGuard tunnel | `wg-quick` | `wg-quick` | `wireguard.exe /installtunnelservice` |
| NAT | iptables | pf (Packet Filter) | Manual netsh |
| Auto-start | `~/.config/autostart/*.desktop` | launchd plist | Registry `HKCU\...\Run` |
| Privilege escalation | sudo/doas/pkexec/systemd-run | sudo | Admin manifest |
| Dependency install | apt/dnf/pacman (auto) | Homebrew (auto) | Manual MSI |

## Build & Development

### Prerequisites
- Go 1.23+
- Node.js 22+
- Wails v2 CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

### Quick start
```bash
git clone https://github.com/Zoop-Network/zooptray
cd zooptray
go mod download
cd frontend && npm ci && cd ..
wails dev
```

### Build release binary
```bash
wails build -platform linux/amd64 -clean
```

### Deploy peer discovery worker
```bash
cd cloudflare-worker
npx wrangler deploy
```

### GitHub CI
Three workflows (linux-release.yml, macos-release.yml, windows-release.yml) build on `v*` tag pushes and publish GitHub Releases with binaries and platform-specific packages (.deb, .dmg, .exe installer).

## Configuration

Located at `~/.zoop/config.json`:

```json
{
  "node": {
    "id": "node-alpha-7",
    "displayName": "Node-Alpha-7",
    "privateKeyPath": "~/.zoop/identity.key",
    "isPublic": true
  },
  "sharing": {
    "enabled": false,
    "maxClients": 10,
    "bandwidthCapPercent": 68,
    "routingPolicy": "balanced"
  },
  "system": {
    "autoStartOnBoot": false,
    "preferredInterface": ""
  },
  "api": {
    "endpoint": "https://zoop-peer-discovery.allan-info-comp.workers.dev",
    "heartbeatIntervalSeconds": 15
  }
}
```

## How Connect Relay Works

1. Machine A enables Sharing → registers with discovery worker, starts heartbeat
2. Machine B clicks "Refresh Network" → sees Machine A in discovered peers
3. Machine B clicks "Connect" → generates fresh WireGuard keypair, creates client config, starts `wg-zoop-client` tunnel
4. Machine B sends `POST /connect-request` to discovery worker with its public key
5. Machine A (in its 2-second telemetry loop) calls `GET /pending-peers`, finds Machine B's request
6. Machine A runs `wg set wg-zoop peer <B's pubkey> allowed-ips 10.200.0.2/32`
7. WireGuard handshake completes → B's traffic routes through A with NAT
8. Machine B clicks "Disconnect" → tunnel stops, `DELETE /connect-request` removes pending entry

## Security

- **WireGuard encryption** — All peer-to-peer traffic is encrypted with Curve25519 key exchange and ChaCha20Poly1305
- **No plaintext credentials stored** — Keys in `~/.zoop/` with `0600` permissions
- **Admin rights required** — Windows manifest enforces `requireAdministrator`; Linux uses privilege escalation for `CAP_NET_ADMIN` operations
- **Self-connection prevented** — Can't route through your own node
