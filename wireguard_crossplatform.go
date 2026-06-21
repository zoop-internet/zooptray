package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// WireGuardManager provides a cross-platform interface for WireGuard operations
type WireGuardManager interface {
	InstallDependencies() error
	StartTunnel(configPath string) error
	StopTunnel(configPath string) error
	AddPeer(interfaceName, peerPubKey, allowedIPs string) error
	RemovePeer(interfaceName, peerPubKey string) error
	GetPeers(interfaceName string) (string, error)
	GetInterfaceName() (string, error)
}

// LinuxWireGuardManager implements WireGuard operations for Linux
type LinuxWireGuardManager struct{}

func (m *LinuxWireGuardManager) InstallDependencies() error {
	return InstallWGDependenciesIfNeeded()
}

func (m *LinuxWireGuardManager) StartTunnel(configPath string) error {
	return StartWGTunnel()
}

func (m *LinuxWireGuardManager) StopTunnel(configPath string) error {
	return StopWGTunnel()
}

func (m *LinuxWireGuardManager) AddPeer(interfaceName, peerPubKey, allowedIPs string) error {
	return AddWGPeer(peerPubKey, allowedIPs)
}

func (m *LinuxWireGuardManager) RemovePeer(interfaceName, peerPubKey string) error {
	return RemoveWGPeer(peerPubKey)
}

func (m *LinuxWireGuardManager) GetPeers(interfaceName string) (string, error) {
	peers, err := GetRealWGPeers()
	if err != nil {
		return "", err
	}
	// Convert peers to string representation
	var result strings.Builder
	for _, peer := range peers {
		result.WriteString(fmt.Sprintf("%s\t%s\t%s\n", peer.Name, peer.MAC, peer.IP))
	}
	return result.String(), nil
}

func (m *LinuxWireGuardManager) GetInterfaceName() (string, error) {
	return GetWGInterfaceName()
}

// MacOSWireGuardManager implements WireGuard operations for macOS
type MacOSWireGuardManager struct{}

func (m *MacOSWireGuardManager) InstallDependencies() error {
	// Check if wireguard-tools is installed via Homebrew
	if _, err := exec.LookPath("wg"); err == nil {
		return nil // Already installed
	}

	// Try to install via Homebrew
	if _, err := exec.LookPath("brew"); err == nil {
		cmd := exec.Command("brew", "install", "wireguard-tools")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to install wireguard-tools via brew: %s (%v)", string(output), err)
		}
		return nil
	}

	return fmt.Errorf("Homebrew not found. Please install wireguard-tools manually via 'brew install wireguard-tools'")
}

func (m *MacOSWireGuardManager) StartTunnel(configPath string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("macOS WireGuard manager can only be used on macOS")
	}

	// Use wg-quick if available
	if _, err := exec.LookPath("wg-quick"); err == nil {
		cmd := exec.Command("sudo", "wg-quick", "up", configPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to start tunnel: %s (%v)", string(output), err)
		}
		return nil
	}

	return fmt.Errorf("wg-quick not found. Please install wireguard-tools")
}

func (m *MacOSWireGuardManager) StopTunnel(configPath string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("macOS WireGuard manager can only be used on macOS")
	}

	if _, err := exec.LookPath("wg-quick"); err == nil {
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

	return fmt.Errorf("wg-quick not found. Please install wireguard-tools")
}

func (m *MacOSWireGuardManager) AddPeer(interfaceName, peerPubKey, allowedIPs string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("macOS WireGuard manager can only be used on macOS")
	}

	if _, err := exec.LookPath("wg"); err != nil {
		return fmt.Errorf("wg command not found. Please install wireguard-tools")
	}

	cmd := exec.Command("sudo", "wg", "set", interfaceName, "peer", peerPubKey, "allowed-ips", allowedIPs)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add peer: %s (%v)", string(output), err)
	}
	return nil
}

func (m *MacOSWireGuardManager) RemovePeer(interfaceName, peerPubKey string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("macOS WireGuard manager can only be used on macOS")
	}

	if _, err := exec.LookPath("wg"); err == nil {
		cmd := exec.Command("sudo", "wg", "set", interfaceName, "peer", peerPubKey, "remove")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to remove peer: %s (%v)", string(output), err)
		}
		return nil
	}

	return fmt.Errorf("wg command not found. Please install wireguard-tools")
}

func (m *MacOSWireGuardManager) GetPeers(interfaceName string) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("macOS WireGuard manager can only be used on macOS")
	}

	if _, err := exec.LookPath("wg"); err == nil {
		cmd := exec.Command("sudo", "wg", "show", interfaceName, "dump")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to get peers: %s (%v)", string(output), err)
		}
		return string(output), nil
	}

	return "", fmt.Errorf("wg command not found. Please install wireguard-tools")
}

func (m *MacOSWireGuardManager) GetInterfaceName() (string, error) {
	return GetWGInterfaceName() // Reuse the existing function
}

// WindowsWireGuardManager implements WireGuard operations for Windows
type WindowsWireGuardManager struct{}

func (m *WindowsWireGuardManager) InstallDependencies() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("Windows WireGuard manager can only be used on Windows")
	}

	// Check if WireGuard is installed
	if _, err := exec.LookPath("wireguard"); err == nil {
		return nil // Already installed
	}

	return fmt.Errorf("WireGuard not found. Please install WireGuard from https://www.wireguard.com/install/")
}

func (m *WindowsWireGuardManager) StartTunnel(configPath string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("Windows WireGuard manager can only be used on Windows")
	}

	// On Windows, we can use the WireGuard CLI or integrate with the service
	// For now, we'll use wg-quick if available (via WireGuard installation)
	if _, err := exec.LookPath("wg-quick"); err == nil {
		cmd := exec.Command("wg-quick", "up", configPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to start tunnel: %s (%v)", string(output), err)
		}
		return nil
	}

	return fmt.Errorf("wg-quick not found. Please install WireGuard from https://www.wireguard.com/install/")
}

func (m *WindowsWireGuardManager) StopTunnel(configPath string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("Windows WireGuard manager can only be used on Windows")
	}

	if _, err := exec.LookPath("wg-quick"); err == nil {
		cmd := exec.Command("wg-quick", "down", configPath)
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

	return fmt.Errorf("wg-quick not found. Please install WireGuard from https://www.wireguard.com/install/")
}

func (m *WindowsWireGuardManager) AddPeer(interfaceName, peerPubKey, allowedIPs string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("Windows WireGuard manager can only be used on Windows")
	}

	if _, err := exec.LookPath("wg"); err == nil {
		cmd := exec.Command("wg", "set", interfaceName, "peer", peerPubKey, "allowed-ips", allowedIPs)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to add peer: %s (%v)", string(output), err)
		}
		return nil
	}

	return fmt.Errorf("wg command not found. Please install WireGuard from https://www.wireguard.com/install/")
}

func (m *WindowsWireGuardManager) RemovePeer(interfaceName, peerPubKey string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("Windows WireGuard manager can only be used on Windows")
	}

	if _, err := exec.LookPath("wg"); err == nil {
		cmd := exec.Command("wg", "set", interfaceName, "peer", peerPubKey, "remove")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to remove peer: %s (%v)", string(output), err)
		}
		return nil
	}

	return fmt.Errorf("wg command not found. Please install WireGuard from https://www.wireguard.com/install/")
}

func (m *WindowsWireGuardManager) GetPeers(interfaceName string) (string, error) {
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("Windows WireGuard manager can only be used on Windows")
	}

	if _, err := exec.LookPath("wg"); err == nil {
		cmd := exec.Command("wg", "show", interfaceName, "dump")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to get peers: %s (%v)", string(output), err)
		}
		return string(output), nil
	}

	return "", fmt.Errorf("wg command not found. Please install WireGuard from https://www.wireguard.com/install/")
}

func (m *WindowsWireGuardManager) GetInterfaceName() (string, error) {
	return GetWGInterfaceName() // Reuse the existing function
}

// GetWireGuardManager returns the appropriate WireGuard manager for the current platform
func GetWireGuardManager() WireGuardManager {
	switch runtime.GOOS {
	case "linux":
		return &LinuxWireGuardManager{}
	case "darwin":
		return &MacOSWireGuardManager{}
	case "windows":
		return &WindowsWireGuardManager{}
	default:
		// For unsupported platforms, return Linux manager as fallback
		// This will fail gracefully with appropriate error messages
		return &LinuxWireGuardManager{}
	}
}