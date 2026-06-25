package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type Telemetry struct {
	UploadSpeed        float64 `json:"uploadSpeed"`   // Mbps
	DownloadSpeed      float64 `json:"downloadSpeed"` // Mbps
	ActiveClientsCount int     `json:"activeClientsCount"`
	TotalDataShared    float64 `json:"totalDataShared"` // GB
	SharingEnabled     bool    `json:"sharingEnabled"`
	TunnelActive       bool    `json:"tunnelActive"`
	TunnelError        string  `json:"tunnelError"`
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
	RxBytes     float64 `json:"-"`
	TxBytes     float64 `json:"-"`
}

type DiscoveryPeer struct {
	NodeID      string `json:"nodeId"`
	PublicKey   string `json:"publicKey"`
	Endpoint    string `json:"endpoint"`
	DisplayName string `json:"displayName"`
	Region      string `json:"region"`
	IsPublic    bool   `json:"isPublic"`
	UpdatedAt   int64  `json:"updatedAt"`
}

type LogMessage struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"` // "info", "warning", "error"
	Message   string `json:"message"`
}

// WGInfo holds the readable WireGuard configuration sent to the frontend
type WGInfo struct {
	PublicKey  string `json:"publicKey"`
	ListenPort int    `json:"listenPort"`
	Address    string `json:"address"`
	Interface  string `json:"wgInterface"`
	ConfigPath string `json:"configPath"`
	Active     bool   `json:"active"`
	Error      string `json:"error"`
}

type NodeService struct {
	ctx             context.Context
	mu              sync.Mutex
	cfg             *Config
	telemetry       Telemetry
	clients         []ClientInfo
	logs            []LogMessage
	sharingState    bool
	startedAt       time.Time
	discoveryCtx    context.Context
	discoveryCancel context.CancelFunc
	stopChan        chan struct{}
	wgManager       WireGuardManager
	connectedPeerID string // nodeId of the peer we are currently connected to
}

func NewNodeService() *NodeService {
	return NewNodeServiceWithManager(nil)
}

func NewNodeServiceWithManager(manager WireGuardManager) *NodeService {
	if manager == nil {
		manager = GetWireGuardManager()
	}
	cfg, _ := LoadConfig()

	service := &NodeService{
		cfg:          cfg,
		sharingState: false,
		stopChan:     make(chan struct{}),
		logs:         make([]LogMessage, 0),
		clients:      make([]ClientInfo, 0),
		wgManager:    manager,
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

	// Stop any active discovery heartbeat and unregister from public discovery.
	n.stopDiscoveryHeartbeat()
	if err := n.unregisterDiscovery(); err != nil {
		n.addLog("warning", fmt.Sprintf("Failed to unregister from discovery during shutdown: %v", err))
	} else {
		n.addLog("info", "Unregistered from Cloudflare discovery during shutdown.")
	}

	configPath, err := GetWGConfigPath()
	if err == nil {
		if err := n.wgManager.StopTunnel(configPath); err != nil {
			n.addLog("warning", fmt.Sprintf("Failed to stop Zoop tunnel during shutdown: %v", err))
		} else {
			n.addLog("info", "Zoop tunnel stopped during app shutdown.")
		}
	} else {
		n.addLog("warning", fmt.Sprintf("Unable to resolve WireGuard config path during shutdown: %v", err))
	}
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

	// Write to ~/.zoop/zoop.log for persistent debugging with rotation
	if home, err := os.UserHomeDir(); err == nil {
		logFile := filepath.Join(home, ".zoop", "zoop.log")
		rotateLogFile(logFile)
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

// rotateLogFile rotates the log file if it exceeds 1MB, keeping up to 3 backups.
func rotateLogFile(logPath string) {
	info, err := os.Stat(logPath)
	if err != nil || info.Size() < 1<<20 {
		return
	}
	for i := 2; i >= 0; i-- {
		old := fmt.Sprintf("%s.%d", logPath, i)
		new := fmt.Sprintf("%s.%d", logPath, i+1)
		os.Rename(old, new)
	}
	os.Rename(logPath, logPath+".1")
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
	lastPeerData := make(map[string][2]float64)

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

			if active {
				n.checkPendingConnections()
				n.checkDisconnectNotifications()
				n.checkActiveClients()
			}

			var nextTelemetry Telemetry
			var nextClients []ClientInfo
			deactivateTunnel := false

			if active {
				peersOutput, err := n.wgManager.GetPeers("")
				if err != nil {
					deactivateTunnel = true
					nextTelemetry = Telemetry{
						SharingEnabled: false,
						TunnelActive:   false,
						TunnelError:    err.Error(),
						PublicIP:       publicIP,
					}
					lastPeerData = make(map[string][2]float64)
				} else {
					realPeers := ParseWGDump(peersOutput)
					var totalUp, totalDown, totalDataMB float64
					for i, rp := range realPeers {
						oldData := lastPeerData[rp.MAC]
						deltaRx := rp.RxBytes - oldData[0]
						deltaTx := rp.TxBytes - oldData[1]
						if deltaRx < 0 {
							deltaRx = 0
						}
						if deltaTx < 0 {
							deltaTx = 0
						}
						lastPeerData[rp.MAC] = [2]float64{rp.RxBytes, rp.TxBytes}
						speedMbps := (deltaRx + deltaTx) * 8.0 / (1024.0 * 1024.0) / 2.0
						realPeers[i].Speed = speedMbps
						totalDown += deltaRx * 8.0 / (1024.0 * 1024.0) / 2.0
						totalUp += deltaTx * 8.0 / (1024.0 * 1024.0) / 2.0
						totalDataMB += rp.DataShared
					}
					nextClients = realPeers
					nextTelemetry = Telemetry{
						UploadSpeed:        totalUp,
						DownloadSpeed:      totalDown,
						ActiveClientsCount: len(realPeers),
						TotalDataShared:    totalDataMB / 1024.0,
						SharingEnabled:     true,
						TunnelActive:       true,
						TunnelError:        tunnelError,
						PublicIP:           publicIP,
					}
					if !startedAt.IsZero() {
						nextTelemetry.StartedAt = startedAt.Format(time.RFC3339)
					}
				}
			} else {
				lastPeerData = make(map[string][2]float64)
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
				go n.saveConfigBackground(cfg)
				// Stop discovery heartbeat and unregister so this node
				// doesn't appear as available when the tunnel is broken.
				go func() {
					n.stopDiscoveryHeartbeat()
					if err := n.unregisterDiscovery(); err != nil {
						n.addLog("warning", fmt.Sprintf("Failed to unregister after tunnel failure: %v", err))
					} else {
						n.addLog("info", "Unregistered from discovery after tunnel failure.")
					}
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
	return CopyConfig(n.cfg)
}

// SaveConfig updates the local config manifest and writes it to file
func (n *NodeService) SaveConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if err := cfg.Validate(); err != nil {
		n.addLog("error", fmt.Sprintf("Configuration validation failed: %v", err))
		return err
	}

	savedCfg := CopyConfig(cfg)

	// Sync autostart settings with OS
	if err := updateAutostart(savedCfg.System.AutoStartOnBoot); err != nil {
		n.addLog("error", fmt.Sprintf("Failed to update autostart setting: %v", err))
	}

	n.mu.Lock()
	n.cfg = savedCfg
	n.addLogLocked("info", "Configuration settings updated and saved locally.")
	n.mu.Unlock()

	err := SaveConfig(savedCfg)
	if err != nil {
		n.addLog("warning", fmt.Sprintf("Failed to persist configuration: %v", err))
	}
	return err
}

func (n *NodeService) saveConfigBackground(cfg *Config) {
	if cfg == nil {
		return
	}
	if err := SaveConfig(cfg); err != nil {
		n.addLog("warning", fmt.Sprintf("Failed to persist configuration: %v", err))
	}
}

func (n *NodeService) discoveryBaseURL() string {
	url := strings.TrimRight(n.cfg.API.Endpoint, "/")
	// Ensure the URL starts with https://
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		url = "https://" + url
	}
	return url
}

func (n *NodeService) discoveryRequest(method, url string, payload interface{}) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		// Check if this is a DNS resolution error
		if strings.Contains(err.Error(), "no such host") || strings.Contains(err.Error(), "lookup") {
			return nil, fmt.Errorf("discovery service DNS resolution failed: %v. Please check your internet connection and that the discovery URL %s is correct", err, n.discoveryBaseURL())
		}
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Check for DNS errors in the response
		if strings.Contains(err.Error(), "no such host") || strings.Contains(err.Error(), "lookup") {
			return nil, fmt.Errorf("discovery service DNS resolution failed: %v. Please check your internet connection and that the discovery URL %s is correct", err, n.discoveryBaseURL())
		}
		return nil, fmt.Errorf("discovery request failed: %w", err)
	}
	return resp, nil
}

func (n *NodeService) refreshPublicIP() {
	ip, err := fetchPublicIP()
	if err != nil {
		n.addLog("warning", fmt.Sprintf("Unable to fetch public IP: %v", err))
		return
	}

	n.mu.Lock()
	n.telemetry.PublicIP = ip
	n.mu.Unlock()
}

func fetchPublicIP() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.ipify.org?format=text")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("public IP lookup returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func (n *NodeService) registerDiscovery(pubKey, endpoint string) error {
	if !n.cfg.Node.IsPublic {
		return nil
	}
	if n.cfg.API.Endpoint == "" {
		return fmt.Errorf("discovery endpoint is not configured. Please set api.endpoint in your config")
	}
	// Validate the endpoint URL
	if !strings.Contains(n.cfg.API.Endpoint, "zoop") && !strings.Contains(n.cfg.API.Endpoint, "localhost") && !strings.HasPrefix(n.cfg.API.Endpoint, "http") {
		return fmt.Errorf("discovery endpoint %s appears to be misconfigured. Expected format: https://your-discovery-worker.workers.dev", n.cfg.API.Endpoint)
	}

	payload := map[string]interface{}{
		"nodeId":      n.cfg.Node.ID,
		"publicKey":   pubKey,
		"endpoint":    endpoint,
		"displayName": n.cfg.Node.DisplayName,
		"region":      "global",
		"isPublic":    true,
	}

	resp, err := n.discoveryRequest("POST", n.discoveryBaseURL()+"/register", payload)
	if err != nil {
		return fmt.Errorf("discovery registration request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response body for better error diagnostics
	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusOK {
		errorMsg := fmt.Sprintf("discovery registration failed: status %d", resp.StatusCode)
		if len(body) > 0 {
			errorMsg += fmt.Sprintf(", response: %s", string(body))
		}
		return fmt.Errorf(errorMsg)
	}
	
	// Log successful registration
	n.addLog("info", fmt.Sprintf("Successfully registered with discovery service at %s", n.discoveryBaseURL()))
	return nil
}

func (n *NodeService) heartbeatDiscovery() error {
	if !n.cfg.Node.IsPublic {
		return nil
	}
	if n.cfg.API.Endpoint == "" {
		return fmt.Errorf("discovery endpoint is not configured")
	}

	n.mu.Lock()
	displayName := n.cfg.Node.DisplayName
	isPublic := n.cfg.Node.IsPublic
	isActive := n.telemetry.TunnelActive
	publicIP := n.telemetry.PublicIP
	n.mu.Unlock()

	payload := map[string]interface{}{
		"nodeId":      n.cfg.Node.ID,
		"displayName": displayName,
		"isPublic":    isPublic,
	}
	// Only send key/endpoint when tunnel is active to avoid overwriting
	// registered values with empty strings.
	if isActive {
		if pk := n.getWGPubKey(); pk != "" {
			payload["publicKey"] = pk
		}
		if publicIP != "" {
			payload["endpoint"] = fmt.Sprintf("%s:51820", publicIP)
		}
	}
	resp, err := n.discoveryRequest("POST", n.discoveryBaseURL()+"/heartbeat", payload)
	if err != nil {
		return fmt.Errorf("discovery heartbeat request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response body for better error diagnostics
	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusOK {
		errorMsg := fmt.Sprintf("discovery heartbeat failed: status %d", resp.StatusCode)
		if len(body) > 0 {
			errorMsg += fmt.Sprintf(", response: %s", string(body))
		}
		return fmt.Errorf(errorMsg)
	}
	
	// Log successful heartbeat
	n.addLog("debug", "Discovery heartbeat successful")
	return nil
}

func (n *NodeService) getWGPubKey() string {
	privKey, err := getOrGenerateWGPrivateKey()
	if err != nil {
		return ""
	}
	pubKey, err := DerivePublicKey(privKey)
	if err != nil {
		return ""
	}
	return pubKey
}

func (n *NodeService) isSelfPeer(peer *DiscoveryPeer) bool {
	if peer == nil {
		return false
	}
	if peer.NodeID == n.cfg.Node.ID {
		return true
	}
	ourPubKey := n.getWGPubKey()
	if ourPubKey != "" && peer.PublicKey == ourPubKey {
		return true
	}
	return false
}

func (n *NodeService) unregisterDiscovery() error {
	if !n.cfg.Node.IsPublic {
		return nil
	}
	if n.cfg.API.Endpoint == "" {
		return fmt.Errorf("discovery endpoint is not configured")
	}

	payload := map[string]interface{}{
		"nodeId": n.cfg.Node.ID,
	}
	resp, err := n.discoveryRequest("DELETE", n.discoveryBaseURL()+"/register", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("discovery unregister failed: status %d", resp.StatusCode)
	}
	return nil
}

func (n *NodeService) DiscoverPeers() ([]DiscoveryPeer, error) {
	if n.cfg.API.Endpoint == "" {
		return nil, fmt.Errorf("discovery endpoint is not configured")
	}

	url := fmt.Sprintf("%s/discover?region=global", n.discoveryBaseURL())
	resp, err := n.discoveryRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery failed: status %d", resp.StatusCode)
	}

	var result struct {
		Success bool            `json:"success"`
		Peers   []DiscoveryPeer `json:"peers"`
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, fmt.Errorf("discovery service returned failure")
	}

	return result.Peers, nil
}

func (n *NodeService) CreateConnectionProfile(nodeID string) (string, error) {
	peers, err := n.DiscoverPeers()
	if err != nil {
		return "", err
	}

	var peer *DiscoveryPeer
	for _, entry := range peers {
		if entry.NodeID == nodeID {
			peer = &entry
			break
		}
	}
	if peer == nil {
		return "", fmt.Errorf("peer %q not found", nodeID)
	}

	profile := fmt.Sprintf(`[Interface]
PrivateKey = <CLIENT_PRIVATE_KEY>
Address = 10.200.0.2/32
DNS = 1.1.1.1

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25
`, peer.PublicKey, peer.Endpoint)

	return profile, nil
}

// ConnectToPeer generates a client WireGuard config and connects to the given peer
func (n *NodeService) ConnectToPeer(nodeID string) error {
	n.addLog("info", fmt.Sprintf("Connecting to peer %s...", nodeID))

	peers, err := n.DiscoverPeers()
	if err != nil {
		err = fmt.Errorf("failed to discover peers: %w", err)
		n.addLog("error", err.Error())
		return err
	}

	var peer *DiscoveryPeer
	for _, entry := range peers {
		if entry.NodeID == nodeID {
			peer = &entry
			break
		}
	}
	if peer == nil {
		err := fmt.Errorf("peer %q not found in discovery", nodeID)
		n.addLog("error", err.Error())
		return err
	}

	if n.isSelfPeer(peer) {
		err := fmt.Errorf("cannot connect to your own node (%s)", peer.Endpoint)
		n.addLog("error", err.Error())
		return err
	}

	// Test connectivity and get best endpoint (direct or relay)
	effectiveEndpoint, err := n.testPeerConnectivity(peer)
	if err != nil {
		n.addLog("error", fmt.Sprintf("Cannot establish connectivity to peer %s: %v", peer.NodeID, err))
		return err
	}
	
	// Update peer endpoint with the working one
	peer.Endpoint = effectiveEndpoint

	privKey, pubKey, err := GenerateWGKeypair()
	if err != nil {
		err = fmt.Errorf("failed to generate client keypair: %w", err)
		n.addLog("error", err.Error())
		return err
	}
	n.addLog("info", fmt.Sprintf("Generated client keypair (public: %s)", pubKey))

	// Generate a unique client IP for this connection
	clientIP, err := n.generateUniqueClientIP()
	if err != nil {
		err = fmt.Errorf("failed to generate unique client IP: %w", err)
		n.addLog("error", err.Error())
		return err
	}
	
	if err := CreateClientWGConfig(privKey, peer.PublicKey, peer.Endpoint, clientIP); err != nil {
		err = fmt.Errorf("failed to create client WireGuard config: %w", err)
		n.addLog("error", err.Error())
		return err
	}

	configPath, _ := GetClientWGConfigPath()
	n.addLog("info", fmt.Sprintf("Stopping any existing tunnel before connecting to peer..."))
	if stopErr := n.wgManager.StopTunnel(configPath); stopErr != nil {
		n.addLog("info", fmt.Sprintf("Tunnel stop before connect: %v", stopErr))
	}

	if err := n.wgManager.StartTunnel(configPath); err != nil {
		err = fmt.Errorf("failed to start client tunnel: %w", err)
		n.addLog("error", err.Error())
		return err
	}

	// Register with the peer via the discovery worker so the peer adds us to its WG config
	if err := n.sendConnectRequest(peer.NodeID, pubKey, clientIP); err != nil {
		n.addLog("warning", fmt.Sprintf("Failed to send connect request to discovery: %v", err))
		// Clean up the tunnel we just started
		_ = n.wgManager.StopTunnel(configPath)
		return fmt.Errorf("failed to send connect request: %w", err)
	}
	n.addLog("info", "Sent connect request to peer via discovery service")

	// Wait for connection confirmation by checking if handshake completed
	if err := n.waitForHandshakeCompletion(configPath, peer.PublicKey); err != nil {
		n.addLog("warning", fmt.Sprintf("Connection handshake failed: %v", err))
		// Clean up
		_ = n.cancelConnectRequest(peer.NodeID)
		_ = n.wgManager.StopTunnel(configPath)
		return fmt.Errorf("connection handshake failed: %w", err)
	}

	n.mu.Lock()
	n.connectedPeerID = peer.NodeID
	n.telemetry.TunnelActive = true
	n.telemetry.SharingEnabled = false
	n.telemetry.TunnelError = ""
	n.mu.Unlock()

	n.addLog("info", fmt.Sprintf("Successfully connected to peer %s (%s)", peer.DisplayName, peer.Endpoint))
	return nil
}

// testPeerConnectivity tests if we can reach a peer directly and falls back to relay if needed
func (n *NodeService) testPeerConnectivity(peer *DiscoveryPeer) (string, error) {
	// First try direct connection
	timeout := 5 * time.Second
	conn, err := net.DialTimeout("udp", peer.Endpoint, timeout)
	if err == nil {
		conn.Close()
		n.addLog("info", fmt.Sprintf("Direct connection to %s successful", peer.Endpoint))
		return peer.Endpoint, nil
	}
	
	n.addLog("warning", fmt.Sprintf("Direct connection to %s failed: %v", peer.Endpoint, err))
	
	// Try UDP hole punching by sending a discovery packet
	if n.tryUDPHolePunching(peer) {
		// Wait a moment for NAT tables to update
		time.Sleep(1 * time.Second)
		
		// Try direct connection again
		conn, err := net.DialTimeout("udp", peer.Endpoint, timeout)
		if err == nil {
			conn.Close()
			n.addLog("info", fmt.Sprintf("UDP hole punching successful for %s", peer.Endpoint))
			return peer.Endpoint, nil
		}
	}
	
	// If direct connection still fails, try to get a relay server from discovery service
	relayEndpoint, err := n.requestRelayServer(peer.NodeID)
	if err != nil {
		return "", fmt.Errorf("direct connection and hole punching failed, no relay available: %w", err)
	}
	
	if relayEndpoint != "" {
		n.addLog("info", fmt.Sprintf("Using relay server %s for peer %s", relayEndpoint, peer.NodeID))
		// Append relay parameter to original endpoint
		return fmt.Sprintf("%s?relay=%s", peer.Endpoint, relayEndpoint), nil
	}
	
	return "", fmt.Errorf("cannot connect to peer %s directly, via hole punching, or via relay", peer.NodeID)
}

// tryUDPHolePunching attempts to establish UDP connectivity through NAT
func (n *NodeService) tryUDPHolePunching(peer *DiscoveryPeer) bool {
	// Parse the peer endpoint
	hostPort := strings.Split(peer.Endpoint, "?")[0] // Remove query params
	host, portStr, err := net.SplitHostPort(hostPort)
	if err != nil {
		n.addLog("warning", fmt.Sprintf("Invalid peer endpoint format: %v", err))
		return false
	}
	
	port, err := strconv.Atoi(portStr)
	if err != nil {
		n.addLog("warning", fmt.Sprintf("Invalid peer port: %v", err))
		return false
	}
	
	// Create UDP connection
	conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		n.addLog("debug", fmt.Sprintf("UDP hole punching dial failed: %v", err))
		return false
	}
	defer conn.Close()
	
	// Set timeout
	conn.SetDeadline(time.Now().Add(2 * time.Second))
	
	// Send a small discovery packet
	discoveryPacket := []byte("ZOOP_DISCOVER")
	_, err = conn.Write(discoveryPacket)
	if err != nil {
		n.addLog("debug", fmt.Sprintf("UDP hole punching send failed: %v", err))
		return false
	}
	
	n.addLog("debug", fmt.Sprintf("Sent UDP hole punching packet to %s", peer.Endpoint))
	return true
}

// requestRelayServer asks the discovery service for a relay server endpoint
func (n *NodeService) requestRelayServer(targetNodeID string) (string, error) {
	if n.cfg.API.Endpoint == "" {
		return "", fmt.Errorf("discovery endpoint is not configured")
	}
	
	url := fmt.Sprintf("%s/relay-request?target=%s", n.discoveryBaseURL(), targetNodeID)
	resp, err := n.discoveryRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("relay request failed: status %d", resp.StatusCode)
	}
	
	var result struct {
		Success bool   `json:"success"`
		Relay   string `json:"relay"`
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	if !result.Success || result.Relay == "" {
		return "", fmt.Errorf("no relay server available")
	}
	
	return result.Relay, nil
}

// generateUniqueClientIP generates a unique IP address for a connecting client
func (n *NodeService) generateUniqueClientIP() (string, error) {
	// Try to find an available IP in the 10.200.0.0/24 range
	// Start from .3 and go up to .254 (avoid .1 for server, .2 for first client)
	
	baseIP := "10.200.0."
	
	// Check existing peers to avoid conflicts
	ifName, err := GetWGInterfaceName()
	if err != nil {
		// If we can't get the interface, just use a default
		return "10.200.0.2/32", nil
	}
	
	peersOutput, err := n.wgManager.GetPeers(ifName)
	if err != nil {
		// If we can't get peers, just use a default
		return "10.200.0.2/32", nil
	}
	
	// Parse existing peer IPs
	existingIPs := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(peersOutput))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")
		if len(parts) >= 4 {
			// allowedIPs field contains the client IP
			allowedIPs := parts[3]
			if strings.Contains(allowedIPs, "10.200.0.") {
				// Extract the IP (format: 10.200.0.x/32)
				ipPart := strings.Split(allowedIPs, "/")[0]
				existingIPs[ipPart] = true
			}
		}
	}
	
	// Find first available IP starting from .3
	for i := 3; i <= 254; i++ {
		testIP := fmt.Sprintf("%s%d/32", baseIP, i)
		ipWithoutMask := fmt.Sprintf("%s%d", baseIP, i)
		if !existingIPs[ipWithoutMask] {
			return testIP, nil
		}
	}
	
	return "", fmt.Errorf("no available client IPs in the 10.200.0.0/24 range")
}

// waitForHandshakeCompletion polls the WireGuard interface to check if handshake with peer completed
func (n *NodeService) waitForHandshakeCompletion(configPath, peerPubKey string) error {
	// Extract interface name from config path
	ifName := strings.TrimSuffix(filepath.Base(configPath), ".conf")
	
	// Try for up to 10 seconds (5 attempts, 2 seconds apart)
	for i := 0; i < 5; i++ {
		if i > 0 {
			// Wait before retrying
			time.Sleep(2 * time.Second)
		}
		
		// Check if handshake completed
		peersOutput, err := n.wgManager.GetPeers(ifName)
		if err != nil {
			continue // Try again
		}
		
		// Parse the output to check for handshake
		hasHandshake := false
		scanner := bufio.NewScanner(strings.NewReader(peersOutput))
		for scanner.Scan() {
			line := scanner.Text()
			// WireGuard dump format: <pubkey>\t<presharedkey>\t<endpoint>\t<allowedips>\t<latesthandshaketime>\t<rx>\t<tx>\t<persistentkeepalive>
			parts := strings.Split(line, "\t")
			if len(parts) >= 5 && parts[0] == peerPubKey {
				// Check if latest handshake time is non-zero
				if parts[4] != "0" {
					hasHandshake = true
					break
				}
			}
		}
		
		if hasHandshake {
			return nil // Success!
		}
	}
	
	return fmt.Errorf("handshake with peer %s did not complete within timeout", peerPubKey)
}

// DisconnectFromPeer stops the client WireGuard tunnel
func (n *NodeService) DisconnectFromPeer() error {
	n.addLog("info", "Disconnecting from peer...")

	n.mu.Lock()
	targetID := n.connectedPeerID
	n.connectedPeerID = ""
	n.mu.Unlock()

	// Get our client public key to notify the target node
	clientPubKey := ""
	if targetID != "" {
		configPath, _ := GetClientWGConfigPath()
		if configPath != "" {
			// Read the client config to get our public key
			if content, err := os.ReadFile(configPath); err == nil {
				scanner := bufio.NewScanner(strings.NewReader(string(content)))
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					if strings.HasPrefix(line, "PrivateKey =") {
						privKey := strings.TrimPrefix(line, "PrivateKey =")
						privKey = strings.TrimSpace(privKey)
						if pubKey, err := DerivePublicKey(privKey); err == nil {
							clientPubKey = pubKey
							break
						}
					}
				}
			}
		}
		
		// Notify target node to remove us from their config
		if clientPubKey != "" {
			if err := n.sendDisconnectNotification(targetID, clientPubKey); err != nil {
				n.addLog("warning", fmt.Sprintf("Failed to send disconnect notification to peer: %v", err))
			} else {
				n.addLog("info", "Sent disconnect notification to peer.")
			}
		}
		
		if err := n.cancelConnectRequest(targetID); err != nil {
			n.addLog("warning", fmt.Sprintf("Failed to cancel connect request: %v", err))
		} else {
			n.addLog("info", "Cancelled connect request with discovery service.")
		}
	}

	configPath, err := GetClientWGConfigPath()
	if err != nil {
		return err
	}
	if err := n.wgManager.StopTunnel(configPath); err != nil {
		n.addLog("warning", fmt.Sprintf("Failed to stop client tunnel: %v", err))
	}

	n.mu.Lock()
	n.telemetry.TunnelActive = false
	n.mu.Unlock()

	n.addLog("info", "Disconnected from peer.")
	return nil
}

// checkActiveClients monitors connected clients and removes stale ones
func (n *NodeService) checkActiveClients() {
	ifName, err := GetWGInterfaceName()
	if err != nil {
		n.addLog("debug", fmt.Sprintf("Cannot get WG interface for client monitoring: %v", err))
		return
	}
	
	peersOutput, err := n.wgManager.GetPeers(ifName)
	if err != nil {
		n.addLog("debug", fmt.Sprintf("Cannot get peers for monitoring: %v", err))
		return
	}
	
	// Parse active peers and their last handshake times
	activePeers := make(map[string]bool) // pubkey -> active
	scanner := bufio.NewScanner(strings.NewReader(peersOutput))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")
		if len(parts) >= 5 {
			pubKey := parts[0]
			handshakeTime := parts[4]
			
			// Consider active if handshake was within last 30 seconds
			if handshakeTime != "0" {
				if handshakeSec, err := strconv.ParseInt(handshakeTime, 10, 64); err == nil {
					currentTime := time.Now().Unix()
					if currentTime-handshakeSec < 30 {
						activePeers[pubKey] = true
					}
				}
			}
		}
	}
	
	// Remove inactive peers
	n.mu.Lock()
	var activeClients []ClientInfo
	for _, client := range n.clients {
		if activePeers[client.IP] { // Using IP as proxy for pubkey in this context
			activeClients = append(activeClients, client)
		} else {
			n.addLog("info", fmt.Sprintf("Removed stale client: %s (%s)", client.Name, client.IP))
		}
	}
	n.clients = activeClients
	n.mu.Unlock()
	
	if n.ctx != nil {
		wailsRuntime.EventsEmit(n.ctx, "clients_updated", n.clients)
	}
}

// checkDisconnectNotifications polls for clients that have disconnected from us
func (n *NodeService) checkDisconnectNotifications() {
	if n.cfg.API.Endpoint == "" {
		return
	}
	url := fmt.Sprintf("%s/disconnect-notifications?nodeId=%s", n.discoveryBaseURL(), n.cfg.Node.ID)
	resp, err := n.discoveryRequest("GET", url, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}

	var result struct {
		Success     bool          `json:"success"`
		Disconnects []struct {
			NodeID       string `json:"nodeId"`
			ClientPubKey string `json:"clientPubKey"`
		} `json:"disconnects"`
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return
	}
	if !result.Success {
		return
	}

	ifName, err := GetWGInterfaceName()
	if err != nil {
		n.addLog("warning", fmt.Sprintf("Cannot get WG interface for disconnect processing: %v", err))
		return
	}

	for _, d := range result.Disconnects {
		if err := n.wgManager.RemovePeer(ifName, d.ClientPubKey); err != nil {
			n.addLog("warning", fmt.Sprintf("Failed to remove disconnected peer %s: %v", d.NodeID, err))
		} else {
			n.addLog("info", fmt.Sprintf("Removed peer %s who disconnected from us", d.NodeID))
		}
	}
}

// sendDisconnectNotification tells the target peer to remove us from their WireGuard config
func (n *NodeService) sendDisconnectNotification(targetNodeId, clientPubKey string) error {
	if n.cfg.API.Endpoint == "" {
		return fmt.Errorf("discovery endpoint is not configured")
	}
	payload := map[string]interface{}{
		"nodeId":       n.cfg.Node.ID,
		"targetNodeId": targetNodeId,
		"clientPubKey": clientPubKey,
	}
	resp, err := n.discoveryRequest("POST", n.discoveryBaseURL()+"/disconnect-notification", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("disconnect notification failed: status %d", resp.StatusCode)
	}
	return nil
}

// sendConnectRequest tells the discovery worker that we (nodeId) want to connect to targetNodeId.
// The target node will pick this up on its next poll and add our public key to its WireGuard config.
func (n *NodeService) sendConnectRequest(targetNodeId, clientPubKey, clientIP string) error {
	if n.cfg.API.Endpoint == "" {
		return fmt.Errorf("discovery endpoint is not configured")
	}
	payload := map[string]interface{}{
		"nodeId":       n.cfg.Node.ID,
		"targetNodeId": targetNodeId,
		"clientPubKey": clientPubKey,
		"clientIP":     clientIP,
	}
	resp, err := n.discoveryRequest("POST", n.discoveryBaseURL()+"/connect-request", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("connect request failed: status %d", resp.StatusCode)
	}
	return nil
}

// cancelConnectRequest removes our pending connect request from the discovery worker.
func (n *NodeService) cancelConnectRequest(targetNodeId string) error {
	if n.cfg.API.Endpoint == "" {
		return nil
	}
	payload := map[string]interface{}{
		"nodeId":       n.cfg.Node.ID,
		"targetNodeId": targetNodeId,
	}
	resp, err := n.discoveryRequest("DELETE", n.discoveryBaseURL()+"/connect-request", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cancel connect request failed: status %d", resp.StatusCode)
	}
	return nil
}

type pendingPeer struct {
	NodeID       string `json:"nodeId"`
	TargetNodeID string `json:"targetNodeId"`
	ClientPubKey string `json:"clientPubKey"`
	ClientIP     string `json:"clientIP"`
}

// checkPendingConnections polls the discovery worker for pending peers wanting to connect to us,
// and adds their public keys to our WireGuard interface.
func (n *NodeService) checkPendingConnections() {
	if n.cfg.API.Endpoint == "" {
		return
	}
	url := fmt.Sprintf("%s/pending-peers?nodeId=%s", n.discoveryBaseURL(), n.cfg.Node.ID)
	resp, err := n.discoveryRequest("GET", url, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}

	var result struct {
		Success bool          `json:"success"`
		Peers   []pendingPeer `json:"peers"`
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return
	}
	if !result.Success {
		return
	}

	for _, p := range result.Peers {
		ifName, err := GetWGInterfaceName()
		if err != nil {
			n.addLog("warning", fmt.Sprintf("Cannot get WG interface for pending peer: %v", err))
			continue
		}
		if err := n.wgManager.AddPeer(ifName, p.ClientPubKey, p.ClientIP); err != nil {
			n.addLog("warning", fmt.Sprintf("Failed to add peer %s: %v", p.NodeID, err))
			continue
		}
		n.addLog("info", fmt.Sprintf("Accepted connection from peer %s", p.NodeID))
	}
}

func (n *NodeService) stopDiscoveryHeartbeat() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.discoveryCancel != nil {
		n.discoveryCancel()
		n.discoveryCancel = nil
		n.discoveryCtx = nil
	}
}

func (n *NodeService) startDiscoveryHeartbeat() {
	n.mu.Lock()
	// Cancel any existing heartbeat
	if n.discoveryCancel != nil {
		n.discoveryCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	n.discoveryCtx = ctx
	n.discoveryCancel = cancel
	interval := time.Duration(n.cfg.API.HeartbeatIntervalSeconds) * time.Second
	n.mu.Unlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := n.heartbeatDiscovery(); err != nil {
				n.addLog("warning", fmt.Sprintf("Discovery heartbeat failed: %v", err))
			}
		}
	}
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
	go n.saveConfigBackground(cfg)
	go func() {
		n.stopDiscoveryHeartbeat()
		if err := n.unregisterDiscovery(); err != nil {
			n.addLog("warning", fmt.Sprintf("Failed to unregister from discovery: %v", err))
		} else {
			n.addLog("info", "Unregistered from Cloudflare discovery.")
		}
	}()

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
		ListenPort: 51820,
		Address:    "10.200.0.1/24",
		Interface:  ifName,
		ConfigPath: configPath,
	}

	// Attempt to clean up any stale interface before starting.
	// This avoids `wg-quick: `wg-zoop' already exists` errors when a previous run
	// left the interface behind.
	if stopErr := n.wgManager.StopTunnel(configPath); stopErr != nil {
		n.addLog("info", fmt.Sprintf("Pre-start: attempt to stop existing wg interface reported: %v", stopErr))
	} else {
		n.addLog("info", "Pre-start: removed stale wg interface (if present)")
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

	if n.cfg.Node.IsPublic {
		n.refreshPublicIP()
		endpoint := n.telemetry.PublicIP
		if endpoint != "" {
			// Add node ID as a query parameter to make endpoint unique per device
			// This helps with NAT traversal and device identification
			endpoint = fmt.Sprintf("%s:51820?node=%s", endpoint, n.cfg.Node.ID)
		}
		if err := n.registerDiscovery(pubKey, endpoint); err != nil {
			n.addLog("warning", fmt.Sprintf("Discovery registration failed: %v", err))
		} else {
			n.addLog("info", "Registered node with Cloudflare discovery service.")
			go n.startDiscoveryHeartbeat()
		}
	}

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

// GetPlatform returns the current operating system (for frontend platform detection)
func (n *NodeService) GetPlatform() string {
	return runtime.GOOS
}

// GetInterfaces retrieves all local non-loopback network interfaces on the machine
func (n *NodeService) GetInterfaces() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return []string{}
	}

	var names []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback == 0 {
			names = append(names, iface.Name)
		}
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
	logsCopy := append([]LogMessage(nil), n.logs...)
	return logsCopy
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
		if !enabled {
			// Check if the value exists before deleting
			check := execCmd("reg", "query", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run", "/v", "Zoop One")
			if err := check.Run(); err != nil {
				return nil
			}
			output, err := execCmd("reg", "delete", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run", "/v", "Zoop One", "/f").CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to update Windows registry autostart: %s (%w)", strings.TrimSpace(string(output)), err)
			}
			return nil
		}
		output, err := execCmd("reg", "add", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run", "/v", "Zoop One", "/t", "REG_SZ", "/d", execPath, "/f").CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to update Windows registry autostart: %s (%w)", strings.TrimSpace(string(output)), err)
		}
		return nil

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
