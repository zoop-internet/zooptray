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

// execWGSudo runs a wg command with sudo on Unix or directly on Windows.
func execWGSudo(args ...string) (string, error) {
	if _, err := exec.LookPath("wg"); err != nil {
		return "", fmt.Errorf("wg command not found. Please install wireguard-tools")
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("wg", args...)
	} else {
		cmd = exec.Command("sudo", append([]string{"wg"}, args...)...)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("wg command failed: %s (%w)", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

// execWGQuickSudo runs wg-quick with sudo on Unix or directly on Windows.
func execWGQuickSudo(action, configPath string) error {
	if _, err := exec.LookPath("wg-quick"); err != nil {
		return fmt.Errorf("wg-quick not found. Please install wireguard-tools")
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("wg-quick", action, configPath)
	} else {
		cmd = exec.Command("sudo", "wg-quick", action, configPath)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		errStr := strings.TrimSpace(string(output))
		if action == "down" && (strings.Contains(errStr, "is not a WireGuard interface") ||
			strings.Contains(errStr, "does not exist")) {
			return nil
		}
		return fmt.Errorf("wg-quick %s failed: %s (%w)", action, errStr, err)
	}
	return nil
}

// resolveInterfaceName resolves the interface name from the WireGuard config path.
func resolveInterfaceName(iface string) (string, error) {
	if iface == "" {
		var err error
		iface, err = GetWGInterfaceName()
		if err != nil {
			return "", err
		}
	}
	if iface == "" {
		return "", fmt.Errorf("wireguard interface name is required")
	}
	return iface, nil
}

// --- Unix (Linux + macOS) ---

// UnixWireGuardManager implements WireGuard operations for Linux and macOS.
type UnixWireGuardManager struct {
	osName string
}

func (m *UnixWireGuardManager) InstallDependencies() error {
	switch m.osName {
	case "linux":
		return InstallWGDependenciesIfNeeded()
	case "darwin":
		if _, err := exec.LookPath("wg"); err == nil {
			return nil
		}
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
	return fmt.Errorf("unsupported unix platform: %s", m.osName)
}

func (m *UnixWireGuardManager) StartTunnel(configPath string) error {
	return execWGQuickSudo("up", configPath)
}

func (m *UnixWireGuardManager) StopTunnel(configPath string) error {
	return execWGQuickSudo("down", configPath)
}

func (m *UnixWireGuardManager) AddPeer(iface, peerPubKey, allowedIPs string) error {
	iface, err := resolveInterfaceName(iface)
	if err != nil {
		return err
	}
	_, err = execWGSudo("set", iface, "peer", peerPubKey, "allowed-ips", allowedIPs)
	return err
}

func (m *UnixWireGuardManager) RemovePeer(iface, peerPubKey string) error {
	iface, err := resolveInterfaceName(iface)
	if err != nil {
		return err
	}
	_, err = execWGSudo("set", iface, "peer", peerPubKey, "remove")
	return err
}

func (m *UnixWireGuardManager) GetPeers(iface string) (string, error) {
	iface, err := resolveInterfaceName(iface)
	if err != nil {
		return "", err
	}
	return execWGSudo("show", iface, "dump")
}

func (m *UnixWireGuardManager) GetInterfaceName() (string, error) {
	return GetWGInterfaceName()
}

// --- Windows ---

type WindowsWireGuardManager struct{}

func (m *WindowsWireGuardManager) InstallDependencies() error {
	if _, err := exec.LookPath("wireguard"); err == nil {
		return nil
	}
	return fmt.Errorf("WireGuard not found. Please install WireGuard from https://www.wireguard.com/install/")
}

func (m *WindowsWireGuardManager) StartTunnel(configPath string) error {
	return execWGQuickSudo("up", configPath)
}

func (m *WindowsWireGuardManager) StopTunnel(configPath string) error {
	return execWGQuickSudo("down", configPath)
}

func (m *WindowsWireGuardManager) AddPeer(iface, peerPubKey, allowedIPs string) error {
	iface, err := resolveInterfaceName(iface)
	if err != nil {
		return err
	}
	_, err = execWGSudo("set", iface, "peer", peerPubKey, "allowed-ips", allowedIPs)
	return err
}

func (m *WindowsWireGuardManager) RemovePeer(iface, peerPubKey string) error {
	iface, err := resolveInterfaceName(iface)
	if err != nil {
		return err
	}
	_, err = execWGSudo("set", iface, "peer", peerPubKey, "remove")
	return err
}

func (m *WindowsWireGuardManager) GetPeers(iface string) (string, error) {
	iface, err := resolveInterfaceName(iface)
	if err != nil {
		return "", err
	}
	return execWGSudo("show", iface, "dump")
}

func (m *WindowsWireGuardManager) GetInterfaceName() (string, error) {
	return GetWGInterfaceName()
}

// --- Unsupported ---

type UnsupportedWireGuardManager struct {
	osName string
}

func (m *UnsupportedWireGuardManager) InstallDependencies() error {
	return fmt.Errorf("wireguard is not supported on %s", m.osName)
}
func (m *UnsupportedWireGuardManager) StartTunnel(string) error {
	return fmt.Errorf("wireguard tunnels are not supported on %s", m.osName)
}
func (m *UnsupportedWireGuardManager) StopTunnel(string) error {
	return fmt.Errorf("wireguard tunnels are not supported on %s", m.osName)
}
func (m *UnsupportedWireGuardManager) AddPeer(_, _, _ string) error {
	return fmt.Errorf("wireguard peer management is not supported on %s", m.osName)
}
func (m *UnsupportedWireGuardManager) RemovePeer(_, _ string) error {
	return fmt.Errorf("wireguard peer management is not supported on %s", m.osName)
}
func (m *UnsupportedWireGuardManager) GetPeers(string) (string, error) {
	return "", fmt.Errorf("wireguard peer listing is not supported on %s", m.osName)
}
func (m *UnsupportedWireGuardManager) GetInterfaceName() (string, error) {
	return "", fmt.Errorf("wireguard is not supported on %s", m.osName)
}

// GetWireGuardManager returns the appropriate WireGuard manager for the current platform
func GetWireGuardManager() WireGuardManager {
	switch runtime.GOOS {
	case "linux":
		return &UnixWireGuardManager{osName: "linux"}
	case "darwin":
		return &UnixWireGuardManager{osName: "darwin"}
	case "windows":
		return &WindowsWireGuardManager{}
	default:
		return &UnsupportedWireGuardManager{osName: runtime.GOOS}
	}
}
