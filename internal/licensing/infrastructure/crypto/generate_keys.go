//go:build ignore

// This file generates Ed25519 key pairs for license signing.
// Run with: go run generate_keys.go
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
)

func main() {
	// Generate key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate key pair: %v\n", err)
		os.Exit(1)
	}

	// Save public key (embedded in CLI binary)
	publicPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKey,
	})
	if err := os.WriteFile("license_public_key.pem", publicPEM, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write public key: %v\n", err)
		os.Exit(1)
	}

	// Save private key (KEEP SECRET - server-side only)
	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKey,
	})
	if err := os.WriteFile("license_private_key.pem", privatePEM, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write private key: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generated license_public_key.pem and license_private_key.pem")
	fmt.Println("IMPORTANT: Keep license_private_key.pem secret! It should only be on your server.")
}
