package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestGenerateWGKeypairAndDerivePublicKey(t *testing.T) {
	privateKey, publicKey, err := GenerateWGKeypair()
	if err != nil {
		t.Fatalf("GenerateWGKeypair() error = %v", err)
	}

	privateBytes, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		t.Fatalf("private key is not base64: %v", err)
	}
	if len(privateBytes) != 32 {
		t.Fatalf("private key length = %d, want 32", len(privateBytes))
	}

	derived, err := DerivePublicKey(privateKey)
	if err != nil {
		t.Fatalf("DerivePublicKey() error = %v", err)
	}
	if derived != publicKey {
		t.Fatalf("derived public key mismatch")
	}
}

func TestWGConfigPathAndInterfaceName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath, err := GetWGConfigPath()
	if err != nil {
		t.Fatalf("GetWGConfigPath() error = %v", err)
	}
	wantPath := filepath.Join(home, ".zoop", "wg-zoop.conf")
	if configPath != wantPath {
		t.Fatalf("config path = %q, want %q", configPath, wantPath)
	}

	ifName, err := GetWGInterfaceName()
	if err != nil {
		t.Fatalf("GetWGInterfaceName() error = %v", err)
	}
	if ifName != "wg-zoop" {
		t.Fatalf("interface name = %q, want wg-zoop", ifName)
	}
}

func TestCreateWGConfigCreatesDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ZOOP_SKIP_WG_SUBNET_VALIDATION", "1")

	privateKey, _, err := GenerateWGKeypair()
	if err != nil {
		t.Fatalf("GenerateWGKeypair() error = %v", err)
	}

	if err := CreateWGConfig(privateKey, "lo"); err != nil {
		t.Fatalf("CreateWGConfig() error = %v", err)
	}

	configPath, err := GetWGConfigPath()
	if err != nil {
		t.Fatalf("GetWGConfigPath() error = %v", err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file was not written: %v", err)
	}

	content := string(data)
	for _, want := range []string{
		"PrivateKey = " + privateKey,
		"Address = 10.200.0.1/24",
		"PostUp = iptables -A FORWARD",
		"PostDown = iptables -D FORWARD",
		"|| true",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("config does not contain %q", want)
		}
	}
}

type fakeWireGuardManager struct{}

func (f *fakeWireGuardManager) InstallDependencies() error          { return nil }
func (f *fakeWireGuardManager) StartTunnel(configPath string) error { return nil }
func (f *fakeWireGuardManager) StopTunnel(configPath string) error  { return nil }
func (f *fakeWireGuardManager) AddPeer(interfaceName, peerPubKey, allowedIPs string) error {
	return nil
}
func (f *fakeWireGuardManager) RemovePeer(interfaceName, peerPubKey string) error { return nil }
func (f *fakeWireGuardManager) GetPeers(interfaceName string) (string, error)     { return "", nil }
func (f *fakeWireGuardManager) GetInterfaceName() (string, error)                 { return "wg-zoop", nil }

func TestParseWGDump(t *testing.T) {
	output := strings.Join([]string{
		"wg-zoop\tserver-private\tserver-public\t51820\toff",
		"wg-zoop\tpeer-one\t(none)\t198.51.100.4:12345\t10.200.0.2/32\t1710000000\t1048576\t2097152\toff",
		"wg-zoop\tpeer-never\t(none)\t(none)\t10.200.0.3/32\t0\t0\t0\toff",
		"malformed",
		"",
	}, "\n")

	peers := ParseWGDump(output)
	if len(peers) != 1 {
		t.Fatalf("len(peers) = %d, want 1", len(peers))
	}
	peer := peers[0]
	if peer.MAC != "peer-one" {
		t.Fatalf("peer MAC = %q, want peer-one", peer.MAC)
	}
	if peer.IP != "10.200.0.2/32" {
		t.Fatalf("peer IP = %q, want 10.200.0.2/32", peer.IP)
	}
	if peer.DataShared != 3 {
		t.Fatalf("peer DataShared = %f, want 3 MB", peer.DataShared)
	}
	if peer.Speed != 0 {
		t.Fatalf("initial parsed speed = %f, want 0", peer.Speed)
	}
}

func TestToggleSharingDisablesState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	service := NewNodeServiceWithManager(&fakeWireGuardManager{})
	service.sharingState = true
	service.telemetry = Telemetry{SharingEnabled: true, TunnelActive: true}
	service.cfg.Sharing.Enabled = true

	result := service.ToggleSharing(false)
	if result != false {
		t.Fatalf("ToggleSharing(false) = %v, want false", result)
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.sharingState {
		t.Fatalf("sharingState = %v, want false", service.sharingState)
	}
	if service.cfg.Sharing.Enabled {
		t.Fatalf("cfg.Sharing.Enabled = %v, want false", service.cfg.Sharing.Enabled)
	}
	if service.telemetry.TunnelActive {
		t.Fatalf("telemetry.TunnelActive = %v, want false", service.telemetry.TunnelActive)
	}
}

func TestConfigValidation(t *testing.T) {
	cfg := &Config{
		Node: NodeConfig{
			ID:             "node-1",
			DisplayName:    "Test Node",
			PrivateKeyPath: "/tmp/test.key",
			IsPublic:       true,
		},
		Sharing: SharingConfig{
			Enabled:             true,
			MaxClients:          0,
			BandwidthCapPercent: 105,
			RoutingPolicy:       "",
		},
		System: SystemConfig{
			AutoStartOnBoot:    false,
			PreferredInterface: "eth0",
		},
		API: ApiConfig{
			Endpoint:                 "https://api.zoopnetwork.test/v1",
			HeartbeatIntervalSeconds: 2,
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected validation error for invalid config, got nil")
	}
}

func TestSaveConfigAndLoadConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := GetDefaultConfig()
	cfg.Node.DisplayName = "Saved Node"
	cfg.Sharing.MaxClients = 25
	cfg.API.HeartbeatIntervalSeconds = 30

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.Node.DisplayName != cfg.Node.DisplayName {
		t.Fatalf("loaded display name = %q, want %q", loaded.Node.DisplayName, cfg.Node.DisplayName)
	}
	if loaded.Sharing.MaxClients != cfg.Sharing.MaxClients {
		t.Fatalf("loaded max clients = %d, want %d", loaded.Sharing.MaxClients, cfg.Sharing.MaxClients)
	}
	if loaded.API.HeartbeatIntervalSeconds != cfg.API.HeartbeatIntervalSeconds {
		t.Fatalf("loaded heartbeat interval = %d, want %d", loaded.API.HeartbeatIntervalSeconds, cfg.API.HeartbeatIntervalSeconds)
	}
}

func TestGetPlatform(t *testing.T) {
	service := NewNodeServiceWithManager(&fakeWireGuardManager{})
	platform := service.GetPlatform()
	if platform != runtime.GOOS {
		t.Fatalf("GetPlatform() = %q, want %q", platform, runtime.GOOS)
	}
}

func TestGetInterfacesNoLoopback(t *testing.T) {
	service := NewNodeServiceWithManager(&fakeWireGuardManager{})
	ifaces := service.GetInterfaces()
	for _, name := range ifaces {
		if name == "lo" {
			t.Fatalf("GetInterfaces() includes loopback interface 'lo'")
		}
	}
}

func TestParseWGDumpWithRealPeers(t *testing.T) {
	output := strings.Join([]string{
		"wg-zoop\tserver-priv\tserver-pub\t51820\toff",
		"wg-zoop\tpeer-one\t(none)\t10.0.0.1:12345\t10.200.0.2/32\t1710000000\t1048576\t2097152\toff",
		"wg-zoop\tpeer-two\t(none)\t10.0.0.2:12345\t10.200.0.3/32\t1710000001\t512\t1024\toff",
	}, "\n")

	peers := ParseWGDump(output)
	if len(peers) != 2 {
		t.Fatalf("len(peers) = %d, want 2", len(peers))
	}
	if peers[0].IP != "10.200.0.2/32" {
		t.Fatalf("peer[0].IP = %q, want 10.200.0.2/32", peers[0].IP)
	}
	if peers[1].IP != "10.200.0.3/32" {
		t.Fatalf("peer[1].IP = %q, want 10.200.0.3/32", peers[1].IP)
	}
}

func TestRotateLogFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	// Write a small file — should not rotate
	if err := os.WriteFile(logPath, []byte("small"), 0644); err != nil {
		t.Fatal(err)
	}
	rotateLogFile(logPath)
	if _, err := os.Stat(logPath + ".1"); err == nil {
		t.Fatal("rotateLogFile created a backup for a small file")
	}

	// Write a 2MB file — should rotate
	data := make([]byte, 2<<20)
	if err := os.WriteFile(logPath, data, 0644); err != nil {
		t.Fatal(err)
	}
	rotateLogFile(logPath)
	if _, err := os.Stat(logPath + ".1"); err != nil {
		t.Fatal("rotateLogFile did not create a backup for a 2MB file")
	}
}

func TestGetWireGuardManagerDispatch(t *testing.T) {
	mgr := GetWireGuardManager()
	if mgr == nil {
		t.Fatal("GetWireGuardManager() returned nil")
	}

	switch runtime.GOOS {
	case "linux", "darwin":
		if _, ok := mgr.(*UnixWireGuardManager); !ok {
			t.Fatalf("expected *UnixWireGuardManager for %s, got %T", runtime.GOOS, mgr)
		}
	case "windows":
		if _, ok := mgr.(*WindowsWireGuardManager); !ok {
			t.Fatalf("expected *WindowsWireGuardManager for windows, got %T", mgr)
		}
	default:
		if _, ok := mgr.(*UnsupportedWireGuardManager); !ok {
			t.Fatalf("expected *UnsupportedWireGuardManager for %s, got %T", runtime.GOOS, mgr)
		}
	}
}

func TestDiscoveryHeartbeatContextCancellation(t *testing.T) {
	service := NewNodeServiceWithManager(&fakeWireGuardManager{})
	service.cfg.Node.IsPublic = true
	service.cfg.API.HeartbeatIntervalSeconds = 1

	// Start the heartbeat in a goroutine
	go service.startDiscoveryHeartbeat()

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	// Stop it — should cancel the context and return
	stopped := make(chan struct{})
	go func() {
		service.stopDiscoveryHeartbeat()
		close(stopped)
	}()

	select {
	case <-stopped:
		// Success
	case <-time.After(3 * time.Second):
		t.Fatal("stopDiscoveryHeartbeat did not return within 3s (context cancellation failed)")
	}
}
