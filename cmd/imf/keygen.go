// Copyright 2026 Benjamin Toso <benjamin.toso@gmail.com>
// Licensed under the Apache License, Version 2.0

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	imfcrypto "github.com/immutable-container/imf/pkg/crypto"
)

// runKeygen handles the "imf keygen" command.
// Generates a new Ed25519 key pair and saves it as PEM files:
//   - imf_private.pem (mode 0600) — used for signing during seal
//   - imf_public.pem  (mode 0644) — used for verification
// The private key should be kept secret; the public key can be shared freely.
func runKeygen() {
	fs := flag.NewFlagSet("imf keygen", flag.ExitOnError)
	outDir := fs.String("out", ".", "Output directory for key files")
	fs.Parse(os.Args[1:])

	kp, err := imfcrypto.GenerateKeyPair()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	privPath := filepath.Join(*outDir, "imf_private.pem")
	pubPath := filepath.Join(*outDir, "imf_public.pem")

	if _, err := os.Stat(privPath); err == nil {
		fmt.Fprintf(os.Stderr, "Error: %s already exists\n", privPath)
		os.Exit(1)
	}

	if err := os.WriteFile(privPath, imfcrypto.MarshalPrivateKeyPEM(kp.PrivateKey), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing private key: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(pubPath, imfcrypto.MarshalPublicKeyPEM(kp.PublicKey), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing public key: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated key pair:\n  Private: %s (keep secret!)\n  Public:  %s\n", privPath, pubPath)
}
