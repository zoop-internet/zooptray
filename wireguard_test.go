package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
