package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

func TestNewP2PNode(t *testing.T) {
	node := NewP2PNode(func(level, msg string) {
		t.Logf("[%s] %s", level, msg)
	})
	if node == nil {
		t.Fatal("NewP2PNode() returned nil")
	}
	if node.namespace != RendezvousNamespace {
		t.Fatalf("namespace = %q, want %q", node.namespace, RendezvousNamespace)
	}
}

func TestP2PLoadOrCreateKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	node := NewP2PNode(nil)

	priv, err := node.loadOrCreateKey()
	if err != nil {
		t.Fatalf("loadOrCreateKey() error = %v", err)
	}
	if priv == nil {
		t.Fatal("loadOrCreateKey() returned nil key")
	}

	keyPath := filepath.Join(home, ".zoop", "p2p.key")
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("key file not created: %v", err)
	}

	priv2, err := node.loadOrCreateKey()
	if err != nil {
		t.Fatalf("loadOrCreateKey() second call error = %v", err)
	}
	if !priv.Equals(priv2) {
		t.Fatal("loaded key does not match saved key")
	}
}

func TestP2PConnectReqEncodeDecode(t *testing.T) {
	req := P2PConnectReq{
		NodeID:       "test-node-1",
		ClientPubKey: "dGVzdC1wdWIta2V5",
		ClientIP:     "10.200.0.2/32",
		DisplayName:  "Test Node",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal connect request: %v", err)
	}

	var decoded P2PConnectReq
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal connect request: %v", err)
	}

	if decoded.NodeID != req.NodeID {
		t.Fatalf("NodeID = %q, want %q", decoded.NodeID, req.NodeID)
	}
	if decoded.ClientPubKey != req.ClientPubKey {
		t.Fatalf("ClientPubKey = %q, want %q", decoded.ClientPubKey, req.ClientPubKey)
	}
	if decoded.ClientIP != req.ClientIP {
		t.Fatalf("ClientIP = %q, want %q", decoded.ClientIP, req.ClientIP)
	}
	if decoded.DisplayName != req.DisplayName {
		t.Fatalf("DisplayName = %q, want %q", decoded.DisplayName, req.DisplayName)
	}
}

func TestP2PConnectRespEncodeDecode(t *testing.T) {
	resp := P2PConnectResp{
		Success:      true,
		ServerIP:     "10.200.0.1/24",
		ServerPubKey: "dGVzdC1zZXJ2ZXIta2V5",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal connect response: %v", err)
	}

	var decoded P2PConnectResp
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal connect response: %v", err)
	}

	if decoded.Success != resp.Success {
		t.Fatalf("Success = %v, want %v", decoded.Success, resp.Success)
	}
	if decoded.ServerIP != resp.ServerIP {
		t.Fatalf("ServerIP = %q, want %q", decoded.ServerIP, resp.ServerIP)
	}
	if decoded.ServerPubKey != resp.ServerPubKey {
		t.Fatalf("ServerPubKey = %q, want %q", decoded.ServerPubKey, resp.ServerPubKey)
	}
}

func TestP2PDisconnectMsgEncodeDecode(t *testing.T) {
	msg := P2PDisconnectMsg{
		NodeID:       "test-node-1",
		ClientPubKey: "dGVzdC1wdWIta2V5",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal disconnect msg: %v", err)
	}

	var decoded P2PDisconnectMsg
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal disconnect msg: %v", err)
	}

	if decoded.NodeID != msg.NodeID {
		t.Fatalf("NodeID = %q, want %q", decoded.NodeID, msg.NodeID)
	}
	if decoded.ClientPubKey != msg.ClientPubKey {
		t.Fatalf("ClientPubKey = %q, want %q", decoded.ClientPubKey, msg.ClientPubKey)
	}
}

func TestP2PConfigDefaults(t *testing.T) {
	cfg := GetDefaultConfig()

	if !cfg.P2P.Enabled {
		t.Fatal("P2P.Enabled should be true by default")
	}
	if cfg.P2P.RendezvousNamespace != RendezvousNamespace {
		t.Fatalf("P2P.RendezvousNamespace = %q, want %q", cfg.P2P.RendezvousNamespace, RendezvousNamespace)
	}
	if len(cfg.P2P.BootstrapPeers) == 0 {
		t.Fatal("P2P.BootstrapPeers should not be empty")
	}
}

func TestP2PStartStop(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	node := NewP2PNode(func(level, msg string) {
		t.Logf("[%s] %s", level, msg)
	})

	if err := node.Start(); err != nil {
		t.Fatalf("P2P Start() error = %v", err)
	}

	pid := node.PeerID()
	if pid == "" {
		t.Fatal("PeerID() returned empty")
	}
	t.Logf("P2P PeerID: %s", pid)

	addrs := node.Addrs()
	if len(addrs) == 0 {
		t.Fatal("Addrs() returned empty")
	}
	t.Logf("P2P Addresses: %s", addrs)

	node.Stop()
}

func TestP2PStartStopWithAdvertising(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	node := NewP2PNode(func(level, msg string) {
		t.Logf("[%s] %s", level, msg)
	})

	if err := node.Start(); err != nil {
		t.Fatalf("P2P Start() error = %v", err)
	}

	// Advertising requires DHT peers; in offline test env this may fail gracefully
	err := node.StartAdvertising()
	if err != nil {
		t.Logf("StartAdvertising (expected offline): %v", err)
	} else {
		node.StopAdvertising()
	}

	node.Stop()
}

func TestBootstrapAddresses(t *testing.T) {
	addrs := BootstrapAddresses()
	if len(addrs) == 0 {
		t.Fatal("BootstrapAddresses() returned empty")
	}
	for _, addr := range addrs {
		if addr == "" {
			t.Fatal("Bootstrap address is empty")
		}
	}
}

func TestP2PNodeLogFn(t *testing.T) {
	var loggedMsg string
	node := NewP2PNode(func(level, msg string) {
		loggedMsg = msg
	})

	node.log("info", "test message")
	if loggedMsg != "test message" {
		t.Fatalf("log message = %q, want %q", loggedMsg, "test message")
	}
}

func TestP2PConfigInSaveLoad(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := GetDefaultConfig()
	cfg.P2P.Enabled = false
	cfg.P2P.RendezvousNamespace = "custom-ns"

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.P2P.Enabled != false {
		t.Fatalf("P2P.Enabled = %v, want false", loaded.P2P.Enabled)
	}
	if loaded.P2P.RendezvousNamespace != "custom-ns" {
		t.Fatalf("P2P.RendezvousNamespace = %q, want %q", loaded.P2P.RendezvousNamespace, "custom-ns")
	}
}

func TestP2PNodeSetCallbacks(t *testing.T) {
	node := NewP2PNode(nil)

	node.SetOnPeerFound(func(id peer.ID, addrs []multiaddr.Multiaddr) {})
	if node.onPeerFound == nil {
		t.Fatal("onPeerFound callback not set")
	}

	node.SetOnConnectReq(func(id peer.ID, req P2PConnectReq) {})
	if node.onConnectReq == nil {
		t.Fatal("onConnectReq callback not set")
	}

	node.SetOnDisconnectReq(func(id peer.ID, msg P2PDisconnectMsg) {})
	if node.onDisconnectReq == nil {
		t.Fatal("onDisconnectReq callback not set")
	}
}
