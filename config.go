package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type NodeConfig struct {
	ID             string `json:"id"`
	DisplayName    string `json:"displayName"`
	PrivateKeyPath string `json:"privateKeyPath"`
	IsPublic       bool   `json:"isPublic"`
}

type SharingConfig struct {
	Enabled             bool   `json:"enabled"`
	MaxClients          int    `json:"maxClients"`
	BandwidthCapPercent int    `json:"bandwidthCapPercent"`
	RoutingPolicy       string `json:"routingPolicy"`
}

type SystemConfig struct {
	AutoStartOnBoot    bool   `json:"autoStartOnBoot"`
	PreferredInterface string `json:"preferredInterface"`
}

type ApiConfig struct {
	Endpoint                 string `json:"endpoint"`
	HeartbeatIntervalSeconds int    `json:"heartbeatIntervalSeconds"`
}

type Config struct {
	Node    NodeConfig    `json:"node"`
	Sharing SharingConfig `json:"sharing"`
	System  SystemConfig  `json:"system"`
	API     ApiConfig     `json:"api"`
}

// GetConfigPath returns the platform-independent path to config.json.
// It resolves to ~/.zoop/config.json.
func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// Use ~/.zoop/config.json as standard config location
	dir := filepath.Join(home, ".zoop")
	return filepath.Join(dir, "config.json"), nil
}

// GetDefaultConfig returns a Config struct initialized with default values.
func GetDefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	keyPath := filepath.Join(home, ".zoop", "identity.key")
	return &Config{
		Node: NodeConfig{
			ID:             "node-alpha-7",
			DisplayName:    "Node-Alpha-7",
			PrivateKeyPath: keyPath,
			IsPublic:       true,
		},
		Sharing: SharingConfig{
			Enabled:             false,
			MaxClients:          10,
			BandwidthCapPercent: 68,
			RoutingPolicy:       "balanced",
		},
		System: SystemConfig{
			AutoStartOnBoot:    false,
			PreferredInterface: "",
		},
		API: ApiConfig{
			Endpoint:                 "https://zoop-peer-discovery.allan-info-comp.workers.dev",
			HeartbeatIntervalSeconds: 15,
		},
	}
}

// LoadConfig reads the config file or creates it with defaults if not present.
func LoadConfig() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return GetDefaultConfig(), err
	}

	// Check if config directory exists, create if not
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return GetDefaultConfig(), err
		}
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := GetDefaultConfig()
		if err := SaveConfig(cfg); err != nil {
			return cfg, err
		}
		return cfg, nil
	}

	// Read and parse file
	data, err := os.ReadFile(path)
	if err != nil {
		return GetDefaultConfig(), err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		// Fallback to default config on parse error
		return GetDefaultConfig(), err
	}

	return &cfg, nil
}

// SaveConfig writes the given Config struct back to config.json.
func SaveConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (cfg *Config) Validate() error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}
	if cfg.Node.DisplayName == "" {
		return fmt.Errorf("display name must not be empty")
	}
	if cfg.Sharing.MaxClients < 1 || cfg.Sharing.MaxClients > 100 {
		return fmt.Errorf("max clients must be between 1 and 100")
	}
	if cfg.Sharing.BandwidthCapPercent < 10 || cfg.Sharing.BandwidthCapPercent > 100 {
		return fmt.Errorf("bandwidth cap percent must be between 10 and 100")
	}
	if cfg.Sharing.RoutingPolicy == "" {
		cfg.Sharing.RoutingPolicy = "balanced"
	}
	if cfg.API.Endpoint == "" {
		return fmt.Errorf("api endpoint must not be empty")
	}
	if cfg.API.HeartbeatIntervalSeconds < 5 {
		return fmt.Errorf("heartbeat interval must be at least 5 seconds")
	}
	return nil
}

func CopyConfig(cfg *Config) *Config {
	if cfg == nil {
		return nil
	}
	copy := *cfg
	copy.Node = cfg.Node
	copy.Sharing = cfg.Sharing
	copy.System = cfg.System
	copy.API = cfg.API
	return &copy
}
