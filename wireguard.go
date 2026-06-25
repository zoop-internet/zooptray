package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/curve25519"
)

// GenerateWGKeypair generates a WireGuard compatible Curve25519 keypair in Base64
func GenerateWGKeypair() (string, string, error) {
	var privateKey [32]byte
	_, err := rand.Read(privateKey[:])
	if err != nil {
		return "", "", err
	}
	// Clamp key bits
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	privBase64 := base64.StdEncoding.EncodeToString(privateKey[:])
	pubBase64 := base64.StdEncoding.EncodeToString(publicKey[:])

	return privBase64, pubBase64, nil
}

// DerivePublicKey computes the WireGuard public key from a base64-encoded private key
func DerivePublicKey(privBase64 string) (string, error) {
	privBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(privBase64))
	if err != nil {
		return "", err
	}
	if len(privBytes) != 32 {
		return "", fmt.Errorf("invalid private key length: %d", len(privBytes))
	}
	var privArr [32]byte
	copy(privArr[:], privBytes)

	var pubArr [32]byte
	curve25519.ScalarBaseMult(&pubArr, &privArr)
	return base64.StdEncoding.EncodeToString(pubArr[:]), nil
}

const (
	defaultWGAddress = "10.200.0.1/24"
	defaultWGNetwork = "10.200.0.0/24"
)

// GetWGConfigPath returns the path to the temporary wireguard configuration file
func GetWGConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".zoop", "wg-zoop.conf"), nil
}

// GetWGInterfaceName derives the WireGuard interface name from its config file.
func GetWGInterfaceName() (string, error) {
	configPath, err := GetWGConfigPath()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(filepath.Base(configPath), ".conf"), nil
}

// GetClientWGConfigPath returns the path to the client WireGuard config
func GetClientWGConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".zoop", "wg-zoop-client.conf"), nil
}

// GetClientWGInterfaceName derives the client WireGuard interface name
func GetClientWGInterfaceName() (string, error) {
	configPath, err := GetClientWGConfigPath()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(filepath.Base(configPath), ".conf"), nil
}

// CreateClientWGConfig generates a WireGuard client config to connect to a peer
func CreateClientWGConfig(privateKey, peerPubKey, peerEndpoint, clientIP string) error {
	configPath, err := GetClientWGConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}
	// Use provided client IP or default to 10.200.0.2/32
	if clientIP == "" {
		clientIP = "10.200.0.2/32"
	}
	
	// Parse endpoint to separate host:port from query parameters
	var cleanEndpoint, relayServer string
	if strings.Contains(peerEndpoint, "?") {
		parts := strings.Split(peerEndpoint, "?")
		cleanEndpoint = parts[0]
		// Check for relay parameter
		if strings.Contains(peerEndpoint, "relay=") {
			for _, param := range strings.Split(parts[1], "&") {
				if strings.HasPrefix(param, "relay=") {
					relayServer = strings.TrimPrefix(param, "relay=")
					break
				}
			}
		}
	} else {
		cleanEndpoint = peerEndpoint
	}
	
	// Build config with primary endpoint and optional relay fallback
	content := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s
DNS = 1.1.1.1

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25
`, privateKey, clientIP, peerPubKey, cleanEndpoint)
	
	// Add relay server as secondary peer if available
	if relayServer != "" {
		content += fmt.Sprintf(`

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25
`, peerPubKey, relayServer)
	}

	return os.WriteFile(configPath, []byte(content), 0600)
}

// CreateWGConfig generates a standard WireGuard config file for the sharing hotspot
func CreateWGConfig(privateKey, preferredInterface string) error {
	configPath, err := GetWGConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}
	if preferredInterface == "" {
		return fmt.Errorf("preferred network interface is required")
	}
	if _, err := net.InterfaceByName(preferredInterface); err != nil {
		return fmt.Errorf("preferred network interface %q is not available: %w", preferredInterface, err)
	}
	if !skipWireGuardSubnetValidation() {
		if err := ensureSubnetAvailable(defaultWGNetwork); err != nil {
			return err
		}
	}

	// Generate platform-specific WireGuard configuration
	var content string
	switch runtime.GOOS {
	case "linux":
		// Linux uses iptables for NAT
		content = fmt.Sprintf(`[Interface]
PrivateKey = %s
ListenPort = 51820
Address = 10.200.0.1/24

# Packet forwarding & NAT configuration
PostUp = iptables -A FORWARD -i %%i -j ACCEPT; iptables -A FORWARD -o %%i -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT; iptables -t nat -A POSTROUTING -o %s -j MASQUERADE
PostDown = iptables -D FORWARD -i %%i -j ACCEPT || true; iptables -D FORWARD -o %%i -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT || true; iptables -t nat -D POSTROUTING -o %s -j MASQUERADE || true
`, privateKey, preferredInterface, preferredInterface)
	case "darwin":
		// macOS uses pf for NAT
		content = fmt.Sprintf(`[Interface]
PrivateKey = %s
ListenPort = 51820
Address = 10.200.0.1/24

# macOS NAT configuration using pf
PostUp = echo "nat on %s from 10.200.0.0/24 to any -> (%s)" | pfctl -ef -
PostDown = pfctl -F all -f /etc/pf.conf
`, privateKey, preferredInterface, preferredInterface)
	case "windows":
		// Windows uses netsh for NAT (simplified - actual implementation may need admin setup)
		content = fmt.Sprintf(`[Interface]
PrivateKey = %s
ListenPort = 51820
Address = 10.200.0.1/24

# Windows NAT note: Manual setup required via netsh or PowerShell
# PostUp = netsh interface portproxy add v4tov4 listenport=51820 listenaddress=0.0.0.0 connectport=51820 connectaddress=10.200.0.1
# PostDown = netsh interface portproxy delete v4tov4 listenport=51820 listenaddress=0.0.0.0
`, privateKey)
	default:
		// Basic configuration without NAT for unsupported platforms
		content = fmt.Sprintf(`[Interface]
PrivateKey = %s
ListenPort = 51820
Address = 10.200.0.1/24
`, privateKey)
	}

	return os.WriteFile(configPath, []byte(content), 0600)
}

// InstallWGDependenciesIfNeeded checks for wireguard-tools and installs them using PolicyKit (pkexec) if missing
func ensureSubnetAvailable(subnet string) error {
	_, desiredNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return fmt.Errorf("invalid WireGuard subnet %q: %w", subnet, err)
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("failed to enumerate interfaces for subnet validation: %w", err)
	}

	for _, iface := range ifaces {
		// Skip validation for the wg-zoop interface itself (we're about to recreate it)
		if iface.Name == "wg-zoop" {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ipNet *net.IPNet
			switch v := addr.(type) {
			case *net.IPNet:
				ipNet = v
			case *net.IPAddr:
				ipNet = &net.IPNet{IP: v.IP, Mask: v.IP.DefaultMask()}
			}
			if ipNet == nil {
				continue
			}
			if desiredNet.Contains(ipNet.IP) {
				return fmt.Errorf("wireguard subnet %s conflicts with existing network address %s on interface %s", subnet, ipNet.String(), iface.Name)
			}
		}
	}
	return nil
}

func skipWireGuardSubnetValidation() bool {
	return os.Getenv("ZOOP_SKIP_WG_SUBNET_VALIDATION") == "1"
}

func InstallWGDependenciesIfNeeded() error {
	if runtime.GOOS != "linux" {
		return nil
	}

	_, err := exec.LookPath("wg-quick")
	if err == nil {
		_ = exec.Command("sudo", "-n", "sysctl", "-w", "net.ipv4.ip_forward=1").Run()
		return nil
	}

	if _, err := os.Stat("/usr/sbin/sysctl"); err == nil {
		_ = exec.Command("sudo", "-n", "/usr/sbin/sysctl", "-w", "net.ipv4.ip_forward=1").Run()
	}

	if stat, err := os.Stat("/usr/bin/apt-get"); err == nil && stat.Mode().IsRegular() {
		_ = exec.Command("pkexec", "/usr/bin/apt-get", "update").Run()
		cmd := exec.Command("pkexec", "/usr/bin/apt-get", "install", "-y", "wireguard", "wireguard-tools", "iptables")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to auto-install dependencies: %s (%v)", string(output), err)
		}
	} else if stat, err := os.Stat("/usr/bin/dnf"); err == nil && stat.Mode().IsRegular() {
		cmd := exec.Command("pkexec", "/usr/bin/dnf", "install", "-y", "wireguard-tools", "iptables")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to auto-install dependencies: %s (%v)", string(output), err)
		}
	} else if stat, err := os.Stat("/usr/bin/pacman"); err == nil && stat.Mode().IsRegular() {
		cmd := exec.Command("pkexec", "/usr/bin/pacman", "-S", "--noconfirm", "wireguard-tools", "iptables")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to auto-install dependencies: %s (%v)", string(output), err)
		}
	} else {
		return fmt.Errorf("wg-quick utility not found. Please install wireguard-tools manually for your Linux distribution")
	}

	if _, err := os.Stat("/usr/sbin/sysctl"); err == nil {
		_ = exec.Command("pkexec", "/usr/sbin/sysctl", "-w", "net.ipv4.ip_forward=1").Run()
	}
	return nil
}

// GetRealWGPeers runs wg show wg-zoop dump to return a slice of ClientInfo representing real WireGuard peers
func GetRealWGPeers() ([]ClientInfo, error) {
	manager := GetWireGuardManager()
	ifName, err := GetWGInterfaceName()
	if err != nil {
		return nil, err
	}

	peersOutput, err := manager.GetPeers(ifName)
	if err != nil {
		return nil, err
	}

	return ParseWGDump(peersOutput), nil
}

// ParseWGDump parses "wg show <iface> dump" output into display-ready clients.
func ParseWGDump(output string) []ClientInfo {
	lines := strings.Split(output, "\n")
	var peers []ClientInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		// wg show dump format:
		// interface public-key preshared-key endpoint allowed-ips latest-handshake rx-bytes tx-bytes persistent-keepalive
		if len(parts) < 9 {
			continue // skip the interface info line
		}

		pubKey := parts[1]
		allowedIPs := parts[4]
		handshakeStr := parts[5]
		rxBytesStr := parts[6]
		txBytesStr := parts[7]

		handshake, _ := strconv.ParseInt(handshakeStr, 10, 64)
		if handshake == 0 {
			continue
		}

		rxBytes, _ := strconv.ParseFloat(rxBytesStr, 64)
		txBytes, _ := strconv.ParseFloat(txBytesStr, 64)

		dataSharedMB := (rxBytes + txBytes) / (1024.0 * 1024.0)

		shortKey := pubKey
		if len(shortKey) > 12 {
			shortKey = shortKey[:8] + "..." + shortKey[len(shortKey)-4:]
		}
		name := fmt.Sprintf("Client (%s)", shortKey)

		connectedAt := time.Unix(handshake, 0).Format("15:04:05")

		peers = append(peers, ClientInfo{
			IP:          allowedIPs,
			MAC:         pubKey,
			Name:        name,
			ConnectedAt: connectedAt,
			Speed:       0.0,
			DataShared:  dataSharedMB,
			RxBytes:     rxBytes,
			TxBytes:     txBytes,
		})
	}

	return peers
}

// GetDefaultRouteInterface finds the active network interface using platform-specific methods
func GetDefaultRouteInterface() string {
	switch runtime.GOOS {
	case "linux":
		// Linux: parse /proc/net/route
		file, err := os.Open("/proc/net/route")
		if err != nil {
			return ""
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				// The second field is the destination IP in hex.
				// Default route has destination "00000000".
				if fields[1] == "00000000" {
					return fields[0] // Interface name
				}
			}
		}
	case "darwin":
		// macOS: use 'route get default' command
		cmd := exec.Command("route", "get", "default")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "   interface: ") {
					parts := strings.Split(line, ": ")
					if len(parts) == 2 {
						return strings.TrimSpace(parts[1])
					}
				}
			}
		}
	case "windows":
		// Windows: use PowerShell to get the default route interface name
		cmd := execCmd("powershell", "-c", "Get-NetRoute -DestinationPrefix '0.0.0.0/0' | Select-Object -ExpandProperty InterfaceAlias")
		output, err := cmd.Output()
		if err == nil {
			if name := strings.TrimSpace(string(output)); name != "" && !strings.Contains(name, "Get-NetRoute") {
				return name
			}
		}
		// Fallback: parse 'netsh interface ipv4 show interfaces' for any connected interface
		cmd = execCmd("netsh", "interface", "ipv4", "show", "interfaces")
		output, err = cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "connected") || strings.Contains(line, "Connected") {
					fields := strings.Fields(line)
					if len(fields) >= 4 {
						// Name is everything after the state column
						stateIdx := -1
						for i, f := range fields {
							if f == "connected" || f == "Connected" {
								stateIdx = i
								break
							}
						}
						if stateIdx > 0 && stateIdx < len(fields)-1 {
							name := strings.Join(fields[stateIdx+1:], " ")
							if name != "" {
								return name
							}
						}
					}
				}
			}
		}
	}
	return ""
}
