package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
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

// CreateWGConfig generates a standard WireGuard config file for the sharing hotspot
func CreateWGConfig(privateKey, preferredInterface string) error {
	configPath, err := GetWGConfigPath()
	if err != nil {
		return err
	}

	// Default template for Zoop sharing server interface
	// Runs NAT forwarding so clients can access the internet
	content := fmt.Sprintf(`[Interface]
PrivateKey = %s
ListenPort = 51820
Address = 10.200.0.1/24

# Packet forwarding & NAT configuration
PostUp = iptables -A FORWARD -i %%i -j ACCEPT; iptables -t nat -A POSTROUTING -o %s -j MASQUERADE
PostDown = iptables -D FORWARD -i %%i -j ACCEPT; iptables -t nat -D POSTROUTING -o %s -j MASQUERADE
`, privateKey, preferredInterface, preferredInterface)

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
		_ = exec.Command("sudo", "sysctl", "-w", "net.ipv4.ip_forward=1").Run()
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
	if runtime.GOOS != "linux" {
		return fmt.Errorf("wireguard tunnel is only supported on Linux in this release")
	}

	// Automatically check and install dependencies if missing
	if err := InstallWGDependenciesIfNeeded(); err != nil {
		return err
	}

	configPath, err := GetWGConfigPath()
	if err != nil {
		return err
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("wireguard configuration file not found: %s", configPath)
	}

	// Execute wg-quick up
	cmd := exec.Command("sudo", "wg-quick", "up", configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start tunnel: %s (%v)", string(output), err)
	}

	return nil
}

// StopWGTunnel calls wg-quick to stop the WireGuard interface
func StopWGTunnel() error {
	if runtime.GOOS != "linux" {
		return nil // NOP on other platforms
	}

	configPath, err := GetWGConfigPath()
	if err != nil {
		return err
	}

	// Check whether the interface is actually up before trying to bring it down
	checkCmd := exec.Command("sudo", "wg", "show", "wg-zoop")
	if checkErr := checkCmd.Run(); checkErr != nil {
		// Interface not found / not running — nothing to stop
		return nil
	}

	cmd := exec.Command("sudo", "wg-quick", "down", configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Suppress "not running" errors gracefully
		errStr := string(output)
		if strings.Contains(errStr, "is not a WireGuard interface") ||
			strings.Contains(errStr, "does not exist") {
			return nil
		}
		return fmt.Errorf("failed to stop tunnel: %s (%v)", errStr, err)
	}

	return nil
}

// AddWGPeer registers a client peer on the WireGuard interface
func AddWGPeer(peerPubKey, clientIP string) error {
	if runtime.GOOS != "linux" {
		return nil
	}

	cmd := exec.Command("sudo", "wg", "set", "wg-zoop", "peer", peerPubKey, "allowed-ips", clientIP+"/32")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add peer: %s (%v)", string(output), err)
	}

	return nil
}

// RemoveWGPeer deletes a client peer from the WireGuard interface
func RemoveWGPeer(peerPubKey string) error {
	if runtime.GOOS != "linux" {
		return nil
	}

	cmd := exec.Command("sudo", "wg", "set", "wg-zoop", "peer", peerPubKey, "remove")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove peer: %s (%v)", string(output), err)
	}

	return nil
}

// GetRealWGPeers runs wg show wg-zoop dump to return a slice of ClientInfo representing real WireGuard peers
func GetRealWGPeers() ([]ClientInfo, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("supported only on Linux")
	}

	// Resolve config path to know interface name (it names the interface after the config filename)
	configPath, err := GetWGConfigPath()
	if err != nil {
		return nil, err
	}
	ifName := strings.TrimSuffix(filepath.Base(configPath), ".conf")

	cmd := exec.Command("sudo", "wg", "show", ifName, "dump")
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	output := string(outputBytes)
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

	return peers, nil
}
