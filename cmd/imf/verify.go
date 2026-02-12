// Copyright 2026 Benjamin Toso <benjamin.toso@gmail.com>
// Licensed under the Apache License, Version 2.0

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/immutable-container/imf/pkg/container"
	imfcrypto "github.com/immutable-container/imf/pkg/crypto"
)

// runVerify handles the "imf verify" command.
// Verifies a sealed container's cryptographic integrity by:
//   1. Checking the Ed25519 signature on the manifest
//   2. Recomputing SHA-256 hashes for every file and comparing to manifest
//   3. Checking expiration date (unless -ignore-expiry is set)
// If -key is omitted and the container has an embedded public key, that key is used.
func runVerify() {
	fs := flag.NewFlagSet("imf verify", flag.ExitOnError)
	keyPath := fs.String("key", "", "Path to Ed25519 public key (PEM). Uses embedded key if omitted.")
	ignoreExpiry := fs.Bool("ignore-expiry", false, "Verify even if container is expired")
	fs.Parse(os.Args[1:])

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "Usage: imf verify <container.imf> [options]")
		os.Exit(1)
	}

	opts := container.VerifyOptions{
		IgnoreExpiry: *ignoreExpiry,
	}

	if *keyPath != "" {
		keyData, err := os.ReadFile(*keyPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading key: %v\n", err)
			os.Exit(1)
		}
		pubKey, err := imfcrypto.ParsePublicKeyPEM(keyData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing key: %v\n", err)
			os.Exit(1)
		}
		opts.PublicKey = pubKey
	}

	if err := container.Verify(fs.Arg(0), opts); err != nil {
		fmt.Fprintf(os.Stderr, "FAILED: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("OK — signature and integrity verified")
}
