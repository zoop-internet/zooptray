package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
)

const (
	RendezvousNamespace = "zoop-share-v1"
	ConnectProtocolID   = protocol.ID("/zoop/connect/1.0.0")
	DisconnectProtocolID = protocol.ID("/zoop/disconnect/1.0.0")

	advertiseTTL     = 30 * time.Second
	discoveryTimeout = 10 * time.Second
	streamTimeout    = 30 * time.Second
)

type P2PConnectReq struct {
	NodeID       string `json:"nodeId"`
	ClientPubKey string `json:"clientPubKey"`
	ClientIP     string `json:"clientIP"`
	DisplayName  string `json:"displayName"`
}

type P2PConnectResp struct {
	Success      bool   `json:"success"`
	ServerIP     string `json:"serverIP"`
	ServerPubKey string `json:"serverPubKey"`
	Error        string `json:"error"`
}

type P2PDisconnectMsg struct {
	NodeID       string `json:"nodeId"`
	ClientPubKey string `json:"clientPubKey"`
}

type P2PPeerInfo struct {
	ID           peer.ID
	Addrs        []multiaddr.Multiaddr
	NodeID       string
	DisplayName  string
	WGPubKey     string
	WGEndpoint   string
}

type P2PNode struct {
	ctx         context.Context
	cancel      context.CancelFunc
	host        host.Host
	dht         *dht.IpfsDHT
	disc        *routing.RoutingDiscovery
	namespace   string

	mu            sync.RWMutex
	advertising   bool
	peers         map[peer.ID]*P2PPeerInfo
	connectedWG   map[string]bool

	onPeerFound     func(peer.ID, []multiaddr.Multiaddr)
	onConnectReq    func(peer.ID, P2PConnectReq)
	onDisconnectReq func(peer.ID, P2PDisconnectMsg)
	logFn           func(level, msg string)
}

func NewP2PNode(logFn func(level, msg string)) *P2PNode {
	ctx, cancel := context.WithCancel(context.Background())

	node := &P2PNode{
		ctx:           ctx,
		cancel:        cancel,
		namespace:     RendezvousNamespace,
		peers:         make(map[peer.ID]*P2PPeerInfo),
		connectedWG:   make(map[string]bool),
		logFn:         logFn,
	}

	return node
}

func (n *P2PNode) loadOrCreateKey() (crypto.PrivKey, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	keyDir := filepath.Join(home, ".zoop")
	os.MkdirAll(keyDir, 0755)
	keyPath := filepath.Join(keyDir, "p2p.key")

	if data, err := os.ReadFile(keyPath); err == nil {
		decoded, err := crypto.UnmarshalPrivateKey(data)
		if err == nil {
			return decoded, nil
		}
	}

	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate p2p key: %w", err)
	}

	data, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshal p2p key: %w", err)
	}
	if err := os.WriteFile(keyPath, data, 0600); err != nil {
		return nil, fmt.Errorf("write p2p key: %w", err)
	}

	return priv, nil
}

func (n *P2PNode) Start() error {
	priv, err := n.loadOrCreateKey()
	if err != nil {
		return fmt.Errorf("load p2p key: %w", err)
	}

	cm, err := connmgr.NewConnManager(200, 300, connmgr.WithGracePeriod(60*time.Second))
	if err != nil {
		return fmt.Errorf("connmgr: %w", err)
	}

	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/udp/0/quic-v1",
			"/ip6/::/tcp/0",
			"/ip6/::/udp/0/quic-v1",
		),
		libp2p.DefaultTransports,
		libp2p.EnableHolePunching(),
		libp2p.EnableAutoRelay(
			autorelay.WithPeerSource(n.findRelayPeers),
			autorelay.WithNumRelays(2),
		),
		libp2p.ConnectionManager(cm),
		libp2p.NATPortMap(),
	)
	if err != nil {
		return fmt.Errorf("create libp2p host: %w", err)
	}
	n.host = h

	kdht, err := dht.New(n.ctx, h, dht.Mode(dht.ModeAuto))
	if err != nil {
		h.Close()
		return fmt.Errorf("create dht: %w", err)
	}
	n.dht = kdht

	n.disc = routing.NewRoutingDiscovery(kdht)

	n.host.SetStreamHandler(ConnectProtocolID, n.handleConnectStream)
	n.host.SetStreamHandler(DisconnectProtocolID, n.handleDisconnectStream)

	if err := kdht.Bootstrap(n.ctx); err != nil {
		n.log("warning", fmt.Sprintf("DHT bootstrap error (non-fatal): %v", err))
	}

	go n.connectToBootstrapPeers()
	go n.runPeerDiscoveryLoop()

	n.log("info", fmt.Sprintf("P2P node started. Peer ID: %s", h.ID().String()))
	n.log("info", fmt.Sprintf("P2P addresses: %s", h.Addrs()))

	return nil
}

func (n *P2PNode) Stop() {
	n.cancel()
	if n.host != nil {
		n.host.Close()
	}
}

func (n *P2PNode) PeerID() peer.ID {
	if n.host == nil {
		return ""
	}
	return n.host.ID()
}

func (n *P2PNode) Addrs() []multiaddr.Multiaddr {
	if n.host == nil {
		return nil
	}
	return n.host.Addrs()
}

func (n *P2PNode) findRelayPeers(ctx context.Context, limit int) <-chan peer.AddrInfo {
	ch := make(chan peer.AddrInfo, limit)
	go func() {
		defer close(ch)
		for _, addr := range BootstrapAddresses() {
			select {
			case <-ctx.Done():
				return
			default:
			}
			maddr, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				continue
			}
			pi, err := peer.AddrInfoFromP2pAddr(maddr)
			if err != nil {
				continue
			}
			select {
			case ch <- *pi:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}

func (n *P2PNode) log(level, msg string) {
	if n.logFn != nil {
		n.logFn(level, msg)
	}
}

func (n *P2PNode) connectToBootstrapPeers() {
	peers := []string{
		"/dns4/bootstrap.libp2p.io/tcp/443/wss/p2p/QmNnooDu7bfnmQD2YgKUew9c9DCY9i5xJkh1oCFJ6Z35Qj",
		"/dns4/node0.preload.ipfs.io/tcp/443/wss/p2p/QmZMxNn25BsjznabQA2QCnncrzJnKCbcR56qsa5YgmgxV9",
		"/dns4/node1.preload.ipfs.io/tcp/443/wss/p2p/QmUjNmr8TgJCn1Ao7D17My4P1GkLmbBFhb3gFiFbWwJxqH",
		"/dns4/bootstrap-0.ingest.porsh.network/tcp/4001/p2p/12D3KooWCfGmzWQnngQVnc3gfKTcQMzWFmB9czWmdLfWAiQifGyZ",
		"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	}

	for _, addr := range peers {
		maddr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			n.log("warning", fmt.Sprintf("Invalid bootstrap addr %s: %v", addr, err))
			continue
		}
		pi, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			n.log("warning", fmt.Sprintf("Failed to parse bootstrap addr %s: %v", addr, err))
			continue
		}
		go func(info peer.AddrInfo) {
			ctx, cancel := context.WithTimeout(n.ctx, 10*time.Second)
			defer cancel()
			if err := n.host.Connect(ctx, info); err != nil {
				n.log("debug", fmt.Sprintf("Bootstrap connect to %s failed: %v", info.ID, err))
			} else {
				n.log("info", fmt.Sprintf("Connected to bootstrap peer: %s", info.ID))
			}
		}(*pi)
	}
}

func (n *P2PNode) StartAdvertising() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.advertising {
		return nil
	}
	if n.disc == nil {
		return fmt.Errorf("P2P discovery not initialized")
	}

	ttl, err := n.disc.Advertise(n.ctx, n.namespace)
	if err != nil {
		return fmt.Errorf("advertise: %w", err)
	}

	n.advertising = true
	n.log("info", fmt.Sprintf("Advertising on rendezvous namespace '%s' (TTL: %s)", n.namespace, ttl))

	go n.repeatAdvertise()

	return nil
}

func (n *P2PNode) StopAdvertising() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.advertising = false
	n.log("info", "Stopped advertising on P2P network")
}

func (n *P2PNode) repeatAdvertise() {
	ticker := time.NewTicker(advertiseTTL / 2)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.mu.RLock()
			if !n.advertising {
				n.mu.RUnlock()
				return
			}
			n.mu.RUnlock()

			if _, err := n.disc.Advertise(n.ctx, n.namespace); err != nil {
				n.log("warning", fmt.Sprintf("Re-advertise failed: %v", err))
			}
		}
	}
}

func (n *P2PNode) runPeerDiscoveryLoop() {
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-time.After(15 * time.Second):
			n.discoverPeers()
		}
	}
}

func (n *P2PNode) discoverPeers() {
	if n.disc == nil {
		return
	}

	peerCh, err := n.disc.FindPeers(n.ctx, n.namespace)
	if err != nil {
		n.log("debug", fmt.Sprintf("FindPeers error: %v", err))
		return
	}

	for pi := range peerCh {
		if pi.ID == n.host.ID() || pi.ID == "" {
			continue
		}

		n.mu.RLock()
		_, known := n.peers[pi.ID]
		n.mu.RUnlock()

		if !known {
			n.mu.Lock()
			n.peers[pi.ID] = &P2PPeerInfo{
				ID:    pi.ID,
				Addrs: pi.Addrs,
			}
			n.mu.Unlock()

			n.log("info", fmt.Sprintf("Discovered peer: %s via DHT", pi.ID))

			if n.onPeerFound != nil {
				n.onPeerFound(pi.ID, pi.Addrs)
			}
		}
	}
}

func (n *P2PNode) DiscoverPeersWithTimeout(timeout time.Duration) ([]P2PPeerInfo, error) {
	if n.disc == nil {
		return nil, fmt.Errorf("discovery not initialized")
	}

	ctx, cancel := context.WithTimeout(n.ctx, timeout)
	defer cancel()

	peerCh, err := n.disc.FindPeers(ctx, n.namespace)
	if err != nil {
		return nil, fmt.Errorf("find peers: %w", err)
	}

	var results []P2PPeerInfo
	seen := make(map[peer.ID]bool)

	for pi := range peerCh {
		if pi.ID == n.host.ID() || pi.ID == "" {
			continue
		}
		if seen[pi.ID] {
			continue
		}
		seen[pi.ID] = true

		n.mu.RLock()
		info := n.peers[pi.ID]
		n.mu.RUnlock()

		if info == nil {
			info = &P2PPeerInfo{ID: pi.ID, Addrs: pi.Addrs}
		}

		results = append(results, *info)
	}

	return results, nil
}

func (n *P2PNode) SendConnectRequest(targetPeerID peer.ID, req P2PConnectReq) (*P2PConnectResp, error) {
	ctx, cancel := context.WithTimeout(n.ctx, streamTimeout)
	defer cancel()

	stream, err := n.host.NewStream(ctx, targetPeerID, ConnectProtocolID)
	if err != nil {
		return nil, fmt.Errorf("open connect stream: %w", err)
	}
	defer stream.Close()

	stream.SetDeadline(time.Now().Add(streamTimeout))

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(req); err != nil {
		return nil, fmt.Errorf("send connect request: %w", err)
	}

	stream.CloseWrite()

	var resp P2PConnectResp
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&resp); err != nil && err != io.EOF {
		return nil, fmt.Errorf("read connect response: %w", err)
	}

	return &resp, nil
}

func (n *P2PNode) SendDisconnectMessage(targetPeerID peer.ID, msg P2PDisconnectMsg) error {
	ctx, cancel := context.WithTimeout(n.ctx, streamTimeout)
	defer cancel()

	stream, err := n.host.NewStream(ctx, targetPeerID, DisconnectProtocolID)
	if err != nil {
		return fmt.Errorf("open disconnect stream: %w", err)
	}
	defer stream.Close()

	stream.SetDeadline(time.Now().Add(streamTimeout))

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		return fmt.Errorf("send disconnect msg: %w", err)
	}

	return nil
}

func (n *P2PNode) GetConnectableAddrs(pid peer.ID) string {
	n.mu.RLock()
	info, ok := n.peers[pid]
	n.mu.RUnlock()

	if !ok {
		return ""
	}

	for _, addr := range info.Addrs {
		s := addr.String()
		if !strings.Contains(s, "/p2p-circuit") && !strings.Contains(s, "/webrtc") {
			relayAddr, _ := multiaddr.NewMultiaddr("/p2p-circuit")
			fullAddr := addr.Encapsulate(relayAddr)
			fullAddr = fullAddr.Encapsulate(
				multiaddr.StringCast("/p2p/" + pid.String()),
			)
			return fullAddr.String()
		}
	}

	return ""
}

func (n *P2PNode) handleConnectStream(stream network.Stream) {
	defer stream.Close()

	remotePeer := stream.Conn().RemotePeer()
	stream.SetDeadline(time.Now().Add(streamTimeout))

	var req P2PConnectReq
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&req); err != nil {
		n.log("warning", fmt.Sprintf("Invalid connect request from %s: %v", remotePeer, err))
		encoder := json.NewEncoder(stream)
		encoder.Encode(P2PConnectResp{Success: false, Error: fmt.Sprintf("parse error: %v", err)})
		return
	}

	n.log("info", fmt.Sprintf("Connect request from peer %s (node: %s)", remotePeer, req.NodeID))

	n.mu.Lock()
	n.connectedWG[req.ClientPubKey] = true
	n.mu.Unlock()

	if n.onConnectReq != nil {
		n.onConnectReq(remotePeer, req)
	}

	resp := P2PConnectResp{
		Success: true,
		ServerIP: "10.200.0.1/24",
	}
	if pk := getOrGenerateWGPublicKey(); pk != "" {
		resp.ServerPubKey = pk
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(resp); err != nil {
		n.log("warning", fmt.Sprintf("Failed to send connect response to %s: %v", remotePeer, err))
	}
}

func (n *P2PNode) handleDisconnectStream(stream network.Stream) {
	defer stream.Close()

	remotePeer := stream.Conn().RemotePeer()
	stream.SetDeadline(time.Now().Add(streamTimeout))

	var msg P2PDisconnectMsg
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&msg); err != nil {
		n.log("warning", fmt.Sprintf("Invalid disconnect msg from %s: %v", remotePeer, err))
		return
	}

	n.log("info", fmt.Sprintf("Disconnect from peer %s (node: %s, pubkey: %s)", remotePeer, msg.NodeID, msg.ClientPubKey))

	n.mu.Lock()
	delete(n.connectedWG, msg.ClientPubKey)
	n.mu.Unlock()

	if n.onDisconnectReq != nil {
		n.onDisconnectReq(remotePeer, msg)
	}
}

func (n *P2PNode) SetOnPeerFound(fn func(peer.ID, []multiaddr.Multiaddr)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.onPeerFound = fn
}

func (n *P2PNode) SetOnConnectReq(fn func(peer.ID, P2PConnectReq)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.onConnectReq = fn
}

func (n *P2PNode) SetOnDisconnectReq(fn func(peer.ID, P2PDisconnectMsg)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.onDisconnectReq = fn
}

func getOrGenerateWGPublicKey() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	keyPath := filepath.Join(home, ".zoop", "wg_private.key")
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return ""
	}
	privKey := strings.TrimSpace(string(data))
	pubKey, err := DerivePublicKey(privKey)
	if err != nil {
		return ""
	}
	return pubKey
}

// BootstrapAddresses returns the multiaddr strings for public bootstrap nodes
func BootstrapAddresses() []string {
	return []string{
		"/dns4/bootstrap.libp2p.io/tcp/443/wss/p2p/QmNnooDu7bfnmQD2YgKUew9c9DCY9i5xJkh1oCFJ6Z35Qj",
		"/dns4/node0.preload.ipfs.io/tcp/443/wss/p2p/QmZMxNn25BsjznabQA2QCnncrzJnKCbcR56qsa5YgmgxV9",
		"/dns4/node1.preload.ipfs.io/tcp/443/wss/p2p/QmUjNmr8TgJCn1Ao7D17My4P1GkLmbBFhb3gFiFbWwJxqH",
		"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	}
}


