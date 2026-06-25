// +build test

package main

import (
	"fmt"
	"os"
)

func main() {
	// Test WireGuard config generation
	privKey := "testPrivateKey12345678901234567890123456="
	peerPubKey := "testPeerPublicKey123456789012345678901="
	peerEndpoint := "1.2.3.4:51820?node=test-node-1"
	clientIP := "10.200.0.2/32"
	
	err := CreateClientWGConfig(privKey, peerPubKey, peerEndpoint, clientIP)
	if err != nil {
		fmt.Printf("❌ WireGuard config generation failed: %v\n", err)
		os.Exit(1)
	}
	
	// Read and display the generated config
	configPath, _ := GetClientWGConfigPath()
	content, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("❌ Failed to read generated config: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("✅ WireGuard config generation: PASS")
	fmt.Printf("Generated config at: %s\n", configPath)
	fmt.Println("Config content:")
	fmt.Println(string(content))
}