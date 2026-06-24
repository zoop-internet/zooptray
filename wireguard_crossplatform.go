//go:build !windows
// +build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func init() {
	if runtime.GOOS == "windows" {
		wireGuardInstallDir = findWireGuardDir()
	}
}

// execCmd creates an exec.Cmd.
func execCmd(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

var wireGuardInstallDir string

func findWireGuardDir() string {
	if wireGuardInstallDir != "" {
		return wireGuardInstallDir
	}
	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "WireGuard"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "WireGuard"),
		`C:\Program Files\WireGuard`,
		`C:\Program Files (x86)\WireGuard`,
	}
	for _, d := range candidates {
		if d == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(d, "wg.exe")); err == nil {
			wireGuardInstallDir = d
			return d
		}
		if _, err := os.Stat(filepath.Join(d, "wireguard.exe")); err == nil {
			wireGuardInstallDir = d
			return d
		}
	}
	for _, name := range []string{"wg", "wireguard"} {
		if p, err := exec.LookPath(name); err == nil {
			wireGuardInstallDir = filepath.Dir(p)
			return wireGuardInstallDir
		}
	}
	return ""
}

func wgBin() string {
	if d := findWireGuardDir(); d != "" {
		return filepath.Join(d, "wg.exe")
	}
	return "wg"
}

func wireGuardBin() string {
	if d := findWireGuardDir(); d != "" {
		return filepath.Join(d, "wireguard.exe")
	}
	return "wireguard"
}

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
	wgPath := wgBin()
	if _, err := exec.LookPath(wgPath); err != nil {
		wgPath = "wg"
		if _, err := exec.LookPath(wgPath); err != nil {
			return "", fmt.Errorf("wg command not found. Please install wireguard-tools")
		}
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = execCmd(wgPath, args...)
	} else {
		cmd = execCmd("sudo", append([]string{wgPath}, args...)...)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("wg command failed: %s (%w)", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

// execWGQuickSudo runs wg-quick with sudo on Unix or directly on Windows.
// On Windows, use wireguard.exe /installtunnelservice instead.
func execWGQuickSudo(action, configPath string) error {
	if runtime.GOOS == "windows" {
		return execWindowsTunnel(action, configPath)
	}
	if _, err := exec.LookPath("wg-quick"); err != nil {
		return fmt.Errorf("wg-quick not found. Please install wireguard-tools")
	}
	cmd := execCmd("sudo", "wg-quick", action, configPath)
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

func resolveWireGuardExe() (string, error) {
	wgExe := wireGuardBin()
	if _, err := os.Stat(wgExe); err != nil {
		if _, err := exec.LookPath("wireguard"); err != nil {
			return "", fmt.Errorf("WireGuard not found. Please install from https://www.wireguard.com/install/")
		}
		return "wireguard", nil
	}
	return wgExe, nil
}

func execWindowsTunnel(action, configPath string) error {
	wgExe, err := resolveWireGuardExe()
	if err != nil {
		return err
	}
	if _, err := os.Stat(configPath); err != nil {
		return fmt.Errorf("config file not found: %s", configPath)
	}
	tunnelName := strings.TrimSuffix(filepath.Base(configPath), ".conf")
	if tunnelName == "" || tunnelName == filepath.Base(configPath) {
		return fmt.Errorf("invalid config filename (must end in .conf): %s", configPath)
	}

	switch action {
	case "up":
		return windowsTunnelUp(wgExe, configPath, tunnelName)
	case "down":
		return windowsTunnelDown(wgExe, tunnelName)
	default:
		return fmt.Errorf("unknown tunnel action: %s", action)
	}
}

func windowsTunnelUp(wgExe, configPath, tunnelName string) error {
	serviceName := "WireGuardTunnel$" + tunnelName

	if windowsServiceExists(serviceName) {
		windowsStopService(serviceName)
		execCmd(wgExe, "/uninstalltunnelservice", tunnelName).Run()
	}

	out, err := execCmd(wgExe, "/installtunnelservice", configPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install WireGuard tunnel service: %s (%w)",
			strings.TrimSpace(string(out)), err)
	}

	if err := windowsWaitForServiceRunning(serviceName, 15); err != nil {
		return err
	}

	return nil
}

func windowsTunnelDown(wgExe, tunnelName string) error {
	serviceName := "WireGuardTunnel$" + tunnelName

	if windowsServiceExists(serviceName) {
		windowsStopService(serviceName)
	}

	out, err := execCmd(wgExe, "/uninstalltunnelservice", tunnelName).CombinedOutput()
	if err != nil {
		errStr := strings.TrimSpace(string(out))
		if strings.Contains(errStr, "The specified service does not exist") ||
			strings.Contains(errStr, "does not exist") ||
			strings.Contains(errStr, "Could not find") {
			return nil
		}
		execCmd("sc", "delete", serviceName).Run()
		return fmt.Errorf("failed to uninstall WireGuard tunnel service: %s (%w)", errStr, err)
	}

	if err := windowsWaitForServiceGone(serviceName, 10); err != nil {
		execCmd("sc", "delete", serviceName).Run()
		return err
	}

	return nil
}

func windowsServiceExists(name string) bool {
	out, err := execCmd("sc", "query", name).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), name)
}

func windowsServiceState(name string) string {
	out, err := execCmd("sc", "query", name).CombinedOutput()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "STATE") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				return strings.TrimSpace(parts[3])
			}
		}
	}
	return ""
}

func windowsStopService(name string) {
	execCmd("net", "stop", name).Run()
	execCmd("sc", "stop", name).Run()
	windowsWaitForServiceState(name, "STOPPED", 10)
}

func windowsWaitForServiceRunning(name string, timeoutSec int) error {
	for i := 0; i < timeoutSec*2; i++ {
		state := windowsServiceState(name)
		switch state {
		case "RUNNING":
			return nil
		case "STOPPED":
			execCmd("net", "start", name).Run()
		case "":
			time.Sleep(500 * time.Millisecond)
			continue
		}
		time.Sleep(500 * time.Millisecond)
	}
	state := windowsServiceState(name)
	return fmt.Errorf("WireGuard tunnel service did not reach RUNNING state (current: %s). Check WireGuard driver installation.", state)
}

func windowsWaitForServiceGone(name string, timeoutSec int) error {
	for i := 0; i < timeoutSec*2; i++ {
		if !windowsServiceExists(name) {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("WireGuard tunnel service still exists after uninstall")
}

func windowsWaitForServiceState(name, targetState string, timeoutSec int) {
	for i := 0; i < timeoutSec*2; i++ {
		state := windowsServiceState(name)
		if state == targetState {
			return
		}
		if state == "" {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
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
			cmd := execCmd("brew", "install", "wireguard-tools")
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
	if _, err := exec.LookPath("wg"); err == nil {
		return nil
	}
	if d := findWireGuardDir(); d != "" {
		return nil
	}
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
