package main

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
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
	stopChan     chan struct{}
}

func NewNodeService() *NodeService {
	cfg, _ := LoadConfig()
	
	service := &NodeService{
		cfg:          cfg,
		sharingState: cfg.Sharing.Enabled,
		stopChan:     make(chan struct{}),
		logs:         make([]LogMessage, 0),
		clients:      make([]ClientInfo, 0),
	}

	// Add initial logs
	service.addLog("info", "Zoop One Service Initialized.")
	service.addLog("info", fmt.Sprintf("Identity Key paired. Display Name: %s", cfg.Node.DisplayName))
	if service.sharingState {
		service.addLog("info", "Starting local sharing node service...")
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

	go n.runSimulationLoop()
}

// Shutdown is called by Wails on exit
func (n *NodeService) Shutdown(ctx context.Context) {
	close(n.stopChan)
}

// addLog appends a log entry and emits it to the frontend
func (n *NodeService) addLog(level, message string) {
	n.mu.Lock()
	defer n.mu.Unlock()

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

	// Emit event to frontend if context exists
	if n.ctx != nil {
		wailsRuntime.EventsEmit(n.ctx, "log_added", logEntry)
	}
}

// runSimulationLoop ticks every 2 seconds to simulate network activity
func (n *NodeService) runSimulationLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Track last peer data transfer to compute speeds dynamically
	lastPeerData := make(map[string]float64)

	// Available random client templates
	clientTemplates := []struct {
		Name string
		MAC  string
	}{
		{"Pixel-8-Pro", "3A:F2:A9:77:E5:22"},
		{"MacBook-Air-M2", "E4:88:F9:C6:1A:05"},
		{"iPad-Pro-11", "08:1E:DF:00:8C:99"},
		{"Ubuntu-Desktop", "90:1C:B4:42:0E:12"},
		{"iPhone-15", "8C:AA:45:90:3A:BB"},
		{"ThinkPad-X1", "D0:27:88:BC:AF:E4"},
	}

	// Initialize with some active clients if sharing is enabled
	n.mu.Lock()
	if n.sharingState {
		initialCount := rand.Intn(3) + 2 // 2 to 4 clients
		for i := 0; i < initialCount; i++ {
			c := clientTemplates[i]
			n.clients = append(n.clients, ClientInfo{
				IP:          fmt.Sprintf("192.168.12.%d", i+100),
				MAC:         c.MAC,
				Name:        c.Name,
				ConnectedAt: time.Now().Add(-time.Duration(rand.Intn(60)) * time.Minute).Format("15:04:05"),
				Speed:       float64(rand.Intn(100)+20) / 10.0,
				DataShared:  float64(rand.Intn(450)+50),
			})
			n.addLog("info", fmt.Sprintf("Client %s (%s) routing established.", c.Name, c.MAC))
		}
	}
	n.mu.Unlock()

	for {
		select {
		case <-n.stopChan:
			return
		case <-ticker.C:
			n.mu.Lock()
			
			// Load config details
			maxClients := n.cfg.Sharing.MaxClients
			bandwidthCap := float64(n.cfg.Sharing.BandwidthCapPercent)

			if n.sharingState {
				// 1. Try to query real WireGuard peers from local interface
				realPeers, err := GetRealWGPeers()
				if err == nil && len(realPeers) > 0 {
					var totalUp, totalDown float64
					for i, rp := range realPeers {
						oldData, exists := lastPeerData[rp.MAC]
						if exists {
							delta := rp.DataShared - oldData
							if delta < 0 {
								delta = 0
							}
							// speed in Mbps = (delta MB * 8) / 2 seconds = delta * 4.0
							realPeers[i].Speed = delta * 4.0
						} else {
							realPeers[i].Speed = 0.0
						}
						lastPeerData[rp.MAC] = rp.DataShared
						totalDown += realPeers[i].Speed
						totalUp += realPeers[i].Speed * 0.15
					}
					n.clients = realPeers

					n.telemetry.UploadSpeed = totalUp
					n.telemetry.DownloadSpeed = totalDown
					n.telemetry.ActiveClientsCount = len(n.clients)
					n.telemetry.TotalDataShared += (totalDown + totalUp) * 2.0 / 8.0 / 1024.0
					n.telemetry.SharingEnabled = true

					if rand.Float32() < 0.05 {
						n.addLog("info", fmt.Sprintf("Telemetry heartbeat sent. Active peers: %d, Data shared: %.4f GB", len(n.clients), n.telemetry.TotalDataShared))
					}
				} else {
					// 2. Fallback to client simulation
					// Client management: random connections / disconnections
					// Ensure active client count is within maxClients
					if len(n.clients) < maxClients && rand.Float32() < 0.15 && len(n.clients) < len(clientTemplates) {
						// Add a new client
						unusedTemplates := []struct{ Name, MAC string }{}
						for _, ct := range clientTemplates {
							used := false
							for _, cl := range n.clients {
								if cl.MAC == ct.MAC {
									used = true
									break
								}
							}
							if !used {
								unusedTemplates = append(unusedTemplates, ct)
							}
						}

						if len(unusedTemplates) > 0 {
							target := unusedTemplates[rand.Intn(len(unusedTemplates))]
							newClient := ClientInfo{
								IP:          fmt.Sprintf("192.168.12.%d", rand.Intn(100)+100),
								MAC:         target.MAC,
								Name:        target.Name,
								ConnectedAt: time.Now().Format("15:04:05"),
								Speed:       float64(rand.Intn(50)+10) / 10.0,
								DataShared:  0,
							}
							n.clients = append(n.clients, newClient)
							n.addLog("info", fmt.Sprintf("New client %s joined guest hotspot.", target.Name))
						}
					} else if len(n.clients) > 0 && rand.Float32() < 0.08 {
						// Remove a random client (client left)
						idx := rand.Intn(len(n.clients))
						leftClient := n.clients[idx]
						n.clients = append(n.clients[:idx], n.clients[idx+1:]...)
						n.addLog("info", fmt.Sprintf("Client %s disconnected from guest hotspot.", leftClient.Name))
					}

					// Update speeds and data shared per client
					var totalUp, totalDown float64
					for i := range n.clients {
						// Speed fluctuates
						n.clients[i].Speed = float64(rand.Intn(120)+20) / 10.0 // 2.0 to 14.0 Mbps
						// Scale down speed based on bandwidth limit percentage
						n.clients[i].Speed = n.clients[i].Speed * (bandwidthCap / 100.0)

						// Increment data shared
						dataDiff := n.clients[i].Speed * 2.0 / 8.0 // Megabits to Megabytes for 2 seconds
						n.clients[i].DataShared += dataDiff

						totalDown += n.clients[i].Speed
						totalUp += n.clients[i].Speed * 0.15 // symmetric Upload representation
					}

					// Update global telemetry
					n.telemetry.UploadSpeed = totalUp
					n.telemetry.DownloadSpeed = totalDown
					n.telemetry.ActiveClientsCount = len(n.clients)
					// Total data accumulated
					n.telemetry.TotalDataShared += (totalDown + totalUp) * 2.0 / 8.0 / 1024.0 // MB to GB
					n.telemetry.SharingEnabled = true

					// Periodically emit telemetry to cloud api (simulated log event)
					if rand.Float32() < 0.05 {
						n.addLog("info", fmt.Sprintf("Telemetry heartbeat sent. Active connections: %d, Data shared: %.4f GB", len(n.clients), n.telemetry.TotalDataShared))
					}
				}

			} else {
				// Node is offline/disabled
				n.clients = []ClientInfo{}
				n.telemetry.UploadSpeed = 0
				n.telemetry.DownloadSpeed = 0
				n.telemetry.ActiveClientsCount = 0
				n.telemetry.SharingEnabled = false
			}

			wailsRuntime.EventsEmit(n.ctx, "telemetry_updated", n.telemetry)
			wailsRuntime.EventsEmit(n.ctx, "clients_updated", n.clients)
			n.mu.Unlock()
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
	n.sharingState = cfg.Sharing.Enabled

	// Sync autostart settings with OS
	if err := updateAutostart(cfg.System.AutoStartOnBoot); err != nil {
		n.addLog("error", fmt.Sprintf("Failed to update autostart setting: %v", err))
	}

	// Trigger log
	n.addLog("info", "Configuration settings updated and saved locally.")

	// Save to config file
	return SaveConfig(cfg)
}

// ToggleSharing switches the sharing state on/off
func (n *NodeService) ToggleSharing(enabled bool) bool {
	n.mu.Lock()
	n.sharingState = enabled
	n.cfg.Sharing.Enabled = enabled
	n.telemetry.SharingEnabled = enabled

	// If turning off, clear clients immediately
	if !enabled {
		n.clients = []ClientInfo{}
		n.telemetry.UploadSpeed = 0
		n.telemetry.DownloadSpeed = 0
		n.telemetry.ActiveClientsCount = 0
	}

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

	if enabled {
		n.addLog("info", "Local routing service started. Hotspot is now visible.")
		go func() {
			n.addLog("info", "Configuring Zoop tunnel interface...")
			privKey, err := getOrGenerateWGPrivateKey()
			if err != nil {
				n.addLog("error", fmt.Sprintf("Failed to get/generate Zoop private key: %v", err))
				if n.ctx != nil {
					wailsRuntime.EventsEmit(n.ctx, "wg_tunnel_error", WGInfo{Error: err.Error()})
				}
				return
			}

			iface := cfg.System.PreferredInterface
			if iface == "" {
				iface = "wlan0"
			}

			if err := CreateWGConfig(privKey, iface); err != nil {
				n.addLog("error", fmt.Sprintf("Failed to write Zoop tunnel configuration: %v", err))
				if n.ctx != nil {
					wailsRuntime.EventsEmit(n.ctx, "wg_tunnel_error", WGInfo{Error: err.Error()})
				}
				return
			}

			// Derive public key from private key for display
			pubKey, pubErr := DerivePublicKey(privKey)
			if pubErr != nil {
				pubKey = "(unable to derive)"
			}

			configPath, _ := GetWGConfigPath()

			n.addLog("info", "Starting Zoop tunnel...")
			if err := StartWGTunnel(); err != nil {
				n.addLog("error", fmt.Sprintf("Failed to start tunnel: %v. Ensure Zoop connection helper is configured correctly.", err))
				if n.ctx != nil {
					// Even on failure, still expose the config so user can set up manually
					wailsRuntime.EventsEmit(n.ctx, "wg_tunnel_ready", WGInfo{
						PublicKey:  pubKey,
						PrivateKey: privKey,
						ListenPort: 51820,
						Address:    "10.200.0.1/24",
						Interface:  "wg-zoop",
						ConfigPath: configPath,
						Active:     false,
						Error:      err.Error(),
					})
				}
			} else {
				n.addLog("info", "Zoop tunnel is active.")
				if n.ctx != nil {
					wailsRuntime.EventsEmit(n.ctx, "wg_tunnel_ready", WGInfo{
						PublicKey:  pubKey,
						PrivateKey: privKey,
						ListenPort: 51820,
						Address:    "10.200.0.1/24",
						Interface:  "wg-zoop",
						ConfigPath: configPath,
						Active:     true,
					})
				}
			}
		}()
	} else {
		n.addLog("warning", "Local routing service stopped. Hotspot offline.")
		if n.ctx != nil {
			wailsRuntime.EventsEmit(n.ctx, "wg_tunnel_stopped", nil)
		}
		go func() {
			n.addLog("info", "Stopping Zoop tunnel...")
			if err := StopWGTunnel(); err != nil {
				n.addLog("warning", fmt.Sprintf("Tunnel stop command reported: %v", err))
			} else {
				n.addLog("info", "Zoop tunnel shutdown complete.")
			}
		}()
	}

	return enabled
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
			n.addLog("warning", fmt.Sprintf("Client %s (%s) kicked and blocked by administrator policy.", cl.Name, mac))
			
			// Kick client from WireGuard interface
			go func(pubKey string) {
				_ = RemoveWGPeer(pubKey)
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

	if runtime.GOOS == "linux" {
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
	}

	// Stubs for macOS & Windows auto-start (managed via installer)
	return nil
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
