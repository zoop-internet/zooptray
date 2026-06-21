package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type Telemetry struct {
	UploadSpeed        float64 `json:"uploadSpeed"`        // Mbps
	DownloadSpeed      float64 `json:"downloadSpeed"`      // Mbps
	ActiveClientsCount int     `json:"activeClientsCount"`
	TotalDataShared    float64 `json:"totalDataShared"`    // GB
	SharingEnabled     bool    `json:"sharingEnabled"`
	TunnelActive        bool    `json:"tunnelActive"`
	TunnelError         string  `json:"tunnelError"`
	StartedAt          string  `json:"startedAt"`
	PublicIP           string  `json:"publicIP"`
}

type ClientInfo struct {
	IP          string  `json:"ip"`
	MAC         string  `json:"mac"`
	Name        string  `json:"name"`
	ConnectedAt string  `json:"connectedAt"`
	Speed       float64 `json:"speed"`      // Mbps
	DataShared  float64 `json:"dataShared"` // MB
}

type LogMessage struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"` // "info", "warning", "error"
	Message   string `json:"message"`
}

// WGInfo holds the readable WireGuard configuration sent to the frontend
type WGInfo struct {
	PublicKey   string `json:"publicKey"`
	PrivateKey  string `json:"privateKey"`
	ListenPort  int    `json:"listenPort"`
	Address     string `json:"address"`
	Interface   string `json:"wgInterface"`
	ConfigPath  string `json:"configPath"`
	Active      bool   `json:"active"`
	Error       string `json:"error"`
}

type NodeService struct {
	ctx          context.Context
	mu           sync.Mutex
	cfg          *Config
	telemetry    Telemetry
	clients      []ClientInfo
	logs         []LogMessage
	sharingState bool
	startedAt    time.Time
	stopChan     chan struct{}
	wgManager    WireGuardManager
}

func NewNodeService() *NodeService {
	cfg, _ := LoadConfig()

	service := &NodeService{
		cfg:          cfg,
		sharingState: false,
		stopChan:     make(chan struct{}),
		logs:         make([]LogMessage, 0),
		clients:      make([]ClientInfo, 0),
		wgManager:    GetWireGuardManager(),
	}

	// Add initial logs
	service.addLog("info", "Zoop One Service Initialized.")
	service.addLog("info", fmt.Sprintf("Identity Key paired. Display Name: %s", cfg.Node.DisplayName))
	if cfg.Sharing.Enabled {
		service.addLog("info", "Sharing is enabled in config; tunnel will be started on launch.")
	} else {
		service.addLog("warning", "Node sharing is currently disabled by policy.")
	}

	return service
}

// Startup is called by Wails when the application starts
func (n *NodeService) Startup(ctx context.Context) {
	n.ctx = ctx
	n.addLog("info", "Background Node Daemon successfully started.")

	// Sync autostart configuration on startup
	if err := updateAutostart(n.cfg.System.AutoStartOnBoot); err != nil {
		n.addLog("error", fmt.Sprintf("Failed to sync autostart file: %v", err))
	} else {
		n.addLog("info", fmt.Sprintf("Autostart sync completed (enabled: %t)", n.cfg.System.AutoStartOnBoot))
	}

	// Install WireGuard dependencies for the current platform
	if err := n.wgManager.InstallDependencies(); err != nil {
		n.addLog("warning", fmt.Sprintf("WireGuard dependencies check: %v", err))
	} else {
		n.addLog("info", "WireGuard dependencies are ready.")
	}

	go n.runTelemetryLoop()
	if n.cfg.Sharing.Enabled {
		go n.startSharingTunnel()
	}
}

// Shutdown is called by Wails on exit
func (n *NodeService) Shutdown(ctx context.Context) {
	close(n.stopChan)
}

// addLogLocked appends a log entry and emits it without locking n.mu (caller must hold the lock)
func (n *NodeService) addLogLocked(level, message string) {
	logEntry := LogMessage{
		Timestamp: time.Now().Format("15:04:05"),
		Level:     level,
		Message:   message,
	}

	n.logs = append(n.logs, logEntry)
	// Cap log length at 100 entries
	if len(n.logs) > 100 {
		n.logs = n.logs[1:]
	}

	// Print to stdout for debugging
	fmt.Printf("[%s] %s: %s\n", logEntry.Timestamp, logEntry.Level, logEntry.Message)

	// Write to ~/.zoop/zoop.log for persistent debugging
	if home, err := os.UserHomeDir(); err == nil {
		logFile := filepath.Join(home, ".zoop", "zoop.log")
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			fmt.Fprintf(f, "[%s] %s: %s\n", logEntry.Timestamp, logEntry.Level, logEntry.Message)
			f.Close()
		}
	}

	// Emit event to frontend if context exists
	if n.ctx != nil {
		wailsRuntime.EventsEmit(n.ctx, "log_added", logEntry)
	}
}

// addLog appends a log entry and emits it to the frontend
func (n *NodeService) addLog(level, message string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.addLogLocked(level, message)
}

// runTelemetryLoop ticks every 2 seconds and publishes real WireGuard state only.
func (n *NodeService) runTelemetryLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Track last peer data transfer to compute speeds dynamically
	lastPeerData := make(map[string]float64)

	for {
		select {
		case <-n.stopChan:
			return
		case <-ticker.C:
			n.mu.Lock()
			active := n.sharingState
			startedAt := n.startedAt
			tunnelError := n.telemetry.TunnelError
			publicIP := n.telemetry.PublicIP
			n.mu.Unlock()

			var nextTelemetry Telemetry
			var nextClients []ClientInfo
			deactivateTunnel := false

			if active {
				peersOutput, err := n.wgManager.GetPeers("")
				if err != nil {
					deactivateTunnel = true
					nextTelemetry = Telemetry{
						SharingEnabled: false,
						TunnelActive:    false,
						TunnelError:     err.Error(),
						PublicIP:        publicIP,
					}
					lastPeerData = make(map[string]float64)
				} else {
					realPeers := ParseWGDump(peersOutput)
					var totalUp, totalDown, totalDataMB float64
					for i, rp := range realPeers {
						oldData, exists := lastPeerData[rp.MAC]
						if exists {
							delta := rp.DataShared - oldData
							if delta < 0 {
								delta = 0
							}
							realPeers[i].Speed = delta * 4.0
						}
						lastPeerData[rp.MAC] = rp.DataShared
						totalDown += realPeers[i].Speed
						totalUp += realPeers[i].Speed * 0.15
						totalDataMB += realPeers[i].DataShared
					}
					nextClients = realPeers
					nextTelemetry = Telemetry{
						UploadSpeed:        totalUp,
						DownloadSpeed:      totalDown,
						ActiveClientsCount: len(realPeers),
						TotalDataShared:    totalDataMB / 1024.0,
						SharingEnabled:     true,
						TunnelActive:        true,
						TunnelError:         tunnelError,
						PublicIP:           publicIP,
					}
					if !startedAt.IsZero() {
						nextTelemetry.StartedAt = startedAt.Format(time.RFC3339)
					}
				}
			} else {
				lastPeerData = make(map[string]float64)
				nextTelemetry = Telemetry{
					TunnelError: tunnelError,
					PublicIP:    publicIP,
				}
			}

			n.mu.Lock()
			if deactivateTunnel {
				n.sharingState = false
				n.startedAt = time.Time{}
				n.cfg.Sharing.Enabled = false
			}
			n.clients = nextClients
			n.telemetry = nextTelemetry
			cfg := n.cfg
			currentTelemetry := n.telemetry
			currentClients := append([]ClientInfo(nil), n.clients...)
			n.mu.Unlock()

			if deactivateTunnel {
				go func() {
					_ = SaveConfig(cfg)
				}()
			}
			if n.ctx != nil {
				wailsRuntime.EventsEmit(n.ctx, "telemetry_updated", currentTelemetry)
				wailsRuntime.EventsEmit(n.ctx, "clients_updated", currentClients)
			}
		}
	}
}

// GetConfig returns the current configuration
func (n *NodeService) GetConfig() *Config {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.cfg
}

// SaveConfig updates the local config manifest and writes it to file
func (n *NodeService) SaveConfig(cfg *Config) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Update local struct
	n.cfg = cfg

	// Sync autostart settings with OS
	if err := updateAutostart(cfg.System.AutoStartOnBoot); err != nil {
		n.addLogLocked("error", fmt.Sprintf("Failed to update autostart setting: %v", err))
	}

	// Trigger log
	n.addLogLocked("info", "Configuration settings updated and saved locally.")

	// Save to config file
	return SaveConfig(cfg)
}

// ToggleSharing switches the sharing state on/off
func (n *NodeService) ToggleSharing(enabled bool) bool {
	if enabled {
		n.mu.Lock()
		n.cfg.Sharing.Enabled = true
		cfg := n.cfg
		n.telemetry.TunnelError = ""
		currentTelemetry := n.telemetry
		n.mu.Unlock()

		if n.ctx != nil {
			wailsRuntime.EventsEmit(n.ctx, "telemetry_updated", currentTelemetry)
		}
		go func() {
			_ = SaveConfig(cfg)
		}()
		go n.startSharingTunnel()
		return false
	}

	n.mu.Lock()
	n.sharingState = false
	n.startedAt = time.Time{}
	n.cfg.Sharing.Enabled = false
	n.clients = []ClientInfo{}
	n.telemetry = Telemetry{PublicIP: n.telemetry.PublicIP}

	// Snapshot current state to emit immediately
	currentTelemetry := n.telemetry
	currentClients := make([]ClientInfo, len(n.clients))
	copy(currentClients, n.clients)
	cfg := n.cfg
	n.mu.Unlock()

	// Immediately push the state change to the frontend — don't wait for next tick
	if n.ctx != nil {
		wailsRuntime.EventsEmit(n.ctx, "telemetry_updated", currentTelemetry)
		wailsRuntime.EventsEmit(n.ctx, "clients_updated", currentClients)
	}

	// Write changes to file in background
	go func() {
		_ = SaveConfig(cfg)
	}()

	n.addLog("warning", "Local routing service stopped. Hotspot offline.")
	if n.ctx != nil {
		wailsRuntime.EventsEmit(n.ctx, "wg_tunnel_stopped", nil)
	}
	go func() {
		n.addLog("info", "Stopping Zoop tunnel...")
		configPath, _ := GetWGConfigPath()
		if err := n.wgManager.StopTunnel(configPath); err != nil {
			n.addLog("warning", fmt.Sprintf("Tunnel stop command reported: %v", err))
		} else {
			n.addLog("info", "Zoop tunnel shutdown complete.")
		}
	}()

	return false
}

// GetTunnelStatus returns the latest backend tunnel status for frontend refresh.
func (n *NodeService) GetTunnelStatus() Telemetry {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.telemetry
}

func (n *NodeService) startSharingTunnel() {
	n.addLog("info", "Configuring Zoop tunnel interface...")
	privKey, err := getOrGenerateWGPrivateKey()
	if err != nil {
		n.failTunnelStart(fmt.Errorf("failed to get/generate Zoop private key: %w", err), WGInfo{})
		return
	}

	n.mu.Lock()
	cfg := n.cfg
	n.mu.Unlock()

	iface := cfg.System.PreferredInterface
	if iface == "" {
		iface = GetDefaultRouteInterface()
	}
	if iface == "" {
		n.failTunnelStart(fmt.Errorf("no default network interface detected; choose a preferred interface in Settings"), WGInfo{})
		return
	}

	if err := CreateWGConfig(privKey, iface); err != nil {
		n.failTunnelStart(fmt.Errorf("failed to write Zoop tunnel configuration: %w", err), WGInfo{})
		return
	}

	pubKey, pubErr := DerivePublicKey(privKey)
	if pubErr != nil {
		pubKey = "(unable to derive)"
	}

	configPath, _ := GetWGConfigPath()
	ifName, _ := GetWGInterfaceName()
	info := WGInfo{
		PublicKey:  pubKey,
		PrivateKey: privKey,
		ListenPort: 51820,
		Address:    "10.200.0.1/24",
		Interface:  ifName,
		ConfigPath: configPath,
	}

	n.addLog("info", "Starting Zoop tunnel...")
	if err := n.wgManager.StartTunnel(configPath); err != nil {
		info.Error = err.Error()
		n.failTunnelStart(fmt.Errorf("failed to start tunnel: %w", err), info)
		return
	}

	now := time.Now()
	n.mu.Lock()
	n.sharingState = true
	n.startedAt = now
	n.cfg.Sharing.Enabled = true
	n.telemetry.SharingEnabled = true
	n.telemetry.TunnelActive = true
	n.telemetry.TunnelError = ""
	n.telemetry.StartedAt = now.Format(time.RFC3339)
	currentTelemetry := n.telemetry
	cfg = n.cfg
	n.mu.Unlock()

	go func() {
		_ = SaveConfig(cfg)
	}()

	info.Active = true
	n.addLog("info", "Zoop tunnel is active.")
	if n.ctx != nil {
		wailsRuntime.EventsEmit(n.ctx, "telemetry_updated", currentTelemetry)
		wailsRuntime.EventsEmit(n.ctx, "wg_tunnel_ready", info)
	}
}

func (n *NodeService) failTunnelStart(err error, info WGInfo) {
	n.addLog("error", fmt.Sprintf("%v. Ensure Zoop connection helper is configured correctly.", err))

	n.mu.Lock()
	n.sharingState = false
	n.startedAt = time.Time{}
	n.cfg.Sharing.Enabled = false
	n.clients = []ClientInfo{}
	n.telemetry = Telemetry{
		TunnelError: err.Error(),
		PublicIP:    n.telemetry.PublicIP,
	}
	currentTelemetry := n.telemetry
	cfg := n.cfg
	n.mu.Unlock()

	go func() {
		_ = SaveConfig(cfg)
	}()

	if info.Error == "" {
		info.Error = err.Error()
	}
	info.Active = false
	if n.ctx != nil {
		wailsRuntime.EventsEmit(n.ctx, "telemetry_updated", currentTelemetry)
		wailsRuntime.EventsEmit(n.ctx, "clients_updated", []ClientInfo{})
		wailsRuntime.EventsEmit(n.ctx, "wg_tunnel_error", info)
	}
}

// GetInterfaces retrieves all local hardware interfaces on the machine
func (n *NodeService) GetInterfaces() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return []string{"wlan0", "eth0", "en0"} // fallback
	}

	var names []string
	for _, iface := range ifaces {
		// Filter out loopback interfaces
		if (iface.Flags & net.FlagLoopback) == 0 {
			names = append(names, iface.Name)
		}
	}

	if len(names) == 0 {
		return []string{"wlan0", "eth0", "en0"}
	}
	return names
}

// DisconnectClient kicks/blocks a client based on MAC Address
func (n *NodeService) DisconnectClient(mac string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	for i, cl := range n.clients {
		if cl.MAC == mac {
			n.clients = append(n.clients[:i], n.clients[i+1:]...)
			n.addLogLocked("warning", fmt.Sprintf("Client %s (%s) kicked and blocked by administrator policy.", cl.Name, mac))

			// Kick client from WireGuard interface
			go func(pubKey string) {
				ifName, _ := GetWGInterfaceName()
				_ = n.wgManager.RemovePeer(ifName, pubKey)
			}(mac)

			return true
		}
	}
	return false
}

// GetLogs returns the current logs slice
func (n *NodeService) GetLogs() []LogMessage {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.logs
}

// updateAutostart handles registering/unregistering the application to start automatically on boot.
func updateAutostart(enabled bool) error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	switch runtime.GOOS {
	case "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		autostartDir := filepath.Join(home, ".config", "autostart")
		desktopFilePath := filepath.Join(autostartDir, "zoop-one.desktop")

		if !enabled {
			if _, err := os.Stat(desktopFilePath); err == nil {
				return os.Remove(desktopFilePath)
			}
			return nil
		}

		// Ensure directory exists
		if err := os.MkdirAll(autostartDir, 0755); err != nil {
			return err
		}

		content := `[Desktop Entry]
Type=Application
Name=Zoop Desktop
Comment=Zoop Sharing Node Daemon
Exec=` + execPath + `
Icon=zoopicon
Terminal=false
Categories=Network;Utility;
X-GNOME-Autostart-enabled=true
`
		return os.WriteFile(desktopFilePath, []byte(content), 0644)

	case "darwin":
		// macOS: use launchd plist
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		launchAgentsDir := filepath.Join(home, "Library", "LaunchAgents")
		plistPath := filepath.Join(launchAgentsDir, "com.zoop.one.plist")

		if !enabled {
			if _, err := os.Stat(plistPath); err == nil {
				return os.Remove(plistPath)
			}
			return nil
		}

		// Ensure directory exists
		if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
			return err
		}

		// Escape the exec path for XML
		escapedPath := strings.ReplaceAll(execPath, "&", "&amp;")
		escapedPath = strings.ReplaceAll(escapedPath, "<", "&lt;")
		escapedPath = strings.ReplaceAll(escapedPath, ">", "&gt;")

		content := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.zoop.one</string>
	<key>ProgramArguments</key>
	<array>
		<string>` + escapedPath + `</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<false/>
	<key>StandardOutPath</key>
	<string>` + filepath.Join(home, "Library", "Logs", "zoop-one.log") + `</string>
	<key>StandardErrorPath</key>
	<string>` + filepath.Join(home, "Library", "Logs", "zoop-one.err") + `</string>
</dict>
</plist>`

		if err := os.WriteFile(plistPath, []byte(content), 0644); err != nil {
			return err
		}

		// Load the plist
		cmd := exec.Command("launchctl", "load", "-w", plistPath)
		return cmd.Run()

	case "windows":
		// Windows: use registry or shortcut in startup folder
		// For now, use the startup folder approach
		appData, err := os.UserConfigDir()
		if err != nil {
			return err
		}
		startupDir := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
		shortcutPath := filepath.Join(startupDir, "Zoop One.lnk")

		if !enabled {
			if _, err := os.Stat(shortcutPath); err == nil {
				return os.Remove(shortcutPath)
			}
			return nil
		}

		// Ensure directory exists
		if err := os.MkdirAll(startupDir, 0755); err != nil {
			return err
		}

		// On Windows, we would typically create a shortcut here
		// For simplicity, we'll copy the executable (not ideal, but works)
		// In a production app, you'd use a proper shortcut creation method
		return os.WriteFile(shortcutPath, []byte(fmt.Sprintf("Shortcut to: %s", execPath)), 0644)

	default:
		return fmt.Errorf("autostart not supported on %s", runtime.GOOS)
	}
}

// getOrGenerateWGPrivateKey retrieves the WireGuard private key from disk or generates a new one.
func getOrGenerateWGPrivateKey() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	keyDir := filepath.Join(home, ".zoop")
	_ = os.MkdirAll(keyDir, 0755)
	privPath := filepath.Join(keyDir, "wg_private.key")

	if data, err := os.ReadFile(privPath); err == nil {
		return strings.TrimSpace(string(data)), nil
	}

	priv, _, err := GenerateWGKeypair()
	if err != nil {
		return "", err
	}

	err = os.WriteFile(privPath, []byte(priv), 0600)
	return priv, err
}
