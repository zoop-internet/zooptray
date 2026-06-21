package main

import (
	"encoding/json"
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
			Endpoint:                 "https://api.zoopnetwork.com/v1",
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
