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
func InstallWGDependenciesIfNeeded() error {
	if runtime.GOOS != "linux" {
		return nil
	}

	_, err := exec.LookPath("wg-quick")
	if err == nil {
		// Already installed, ensure IP forwarding is active (non-blocking)
		_ = exec.Command("sudo", "-n", "sysctl", "-w", "net.ipv4.ip_forward=1").Run()
		return nil
	}

	// wg-quick is missing, attempt auto-install using pkexec with the detected package manager
	if _, err := os.Stat("/usr/bin/apt-get"); err == nil {
		// Debian / Ubuntu / Mint
		cmd := exec.Command("pkexec", "sh", "-c", "apt-get update && apt-get install -y wireguard wireguard-tools iptables && sysctl -w net.ipv4.ip_forward=1")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to auto-install dependencies: %s (%v)", string(output), err)
		}
	} else if _, err := os.Stat("/usr/bin/dnf"); err == nil {
		// Fedora / RHEL
		cmd := exec.Command("pkexec", "sh", "-c", "dnf install -y wireguard-tools iptables && sysctl -w net.ipv4.ip_forward=1")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to auto-install dependencies: %s (%v)", string(output), err)
		}
	} else if _, err := os.Stat("/usr/bin/pacman"); err == nil {
		// Arch Linux
		cmd := exec.Command("pkexec", "sh", "-c", "pacman -S --noconfirm wireguard-tools iptables && sysctl -w net.ipv4.ip_forward=1")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to auto-install dependencies: %s (%v)", string(output), err)
		}
	} else {
		return fmt.Errorf("wg-quick utility not found. Please install wireguard-tools manually for your Linux distribution")
	}

	return nil
}

// StartWGTunnel calls wg-quick to start the WireGuard interface
func StartWGTunnel() error {
	manager := GetWireGuardManager()
	configPath, err := GetWGConfigPath()
	if err != nil {
		return err
	}
	return manager.StartTunnel(configPath)
}

// StopWGTunnel calls wg-quick to stop the WireGuard interface
func StopWGTunnel() error {
	manager := GetWireGuardManager()
	configPath, err := GetWGConfigPath()
	if err != nil {
		return err
	}
	return manager.StopTunnel(configPath)
}

// AddWGPeer registers a client peer on the WireGuard interface
func AddWGPeer(peerPubKey, clientIP string) error {
	manager := GetWireGuardManager()
	ifName, err := GetWGInterfaceName()
	if err != nil {
		return err
	}
	return manager.AddPeer(ifName, peerPubKey, clientIP+"/32")
}

// RemoveWGPeer deletes a client peer from the WireGuard interface
func RemoveWGPeer(peerPubKey string) error {
	manager := GetWireGuardManager()
	ifName, err := GetWGInterfaceName()
	if err != nil {
		return err
	}
	return manager.RemovePeer(ifName, peerPubKey)
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
		// Usually contains 9 columns. The first line is the interface private-key details, which has 5 columns.
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
			// Peer has never completed a handshake; skip in active view
			continue
		}

		rxBytes, _ := strconv.ParseFloat(rxBytesStr, 64)
		txBytes, _ := strconv.ParseFloat(txBytesStr, 64)

		// Convert bytes to Megabytes
		dataSharedMB := (rxBytes + txBytes) / (1024.0 * 1024.0)

		// Create readable display name using beginning of public key
		shortKey := pubKey
		if len(shortKey) > 12 {
			shortKey = shortKey[:8] + "..." + shortKey[len(shortKey)-4:]
		}
		name := fmt.Sprintf("Client (%s)", shortKey)

		// Format connection time
		connectedAt := time.Unix(handshake, 0).Format("15:04:05")

		peers = append(peers, ClientInfo{
			IP:          allowedIPs,
			MAC:         pubKey, // Use public key as the unique MAC representation in UI
			Name:        name,
			ConnectedAt: connectedAt,
			Speed:       0.0, // Speed metric parsed/approximated or dynamic
			DataShared:  dataSharedMB,
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
		// Windows: use 'netsh interface ipv4 show interfaces' command
		cmd := exec.Command("netsh", "interface", "ipv4", "show", "interfaces")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "Connected") && strings.Contains(line, "Dedicated") {
					fields := strings.Fields(line)
					if len(fields) >= 1 {
						// The interface name is typically the first field
						return strings.TrimSpace(fields[0])
					}
				}
			}
		}
	}
	return ""
}
