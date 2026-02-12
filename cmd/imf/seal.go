// Copyright 2026 Benjamin Toso <benjamin.toso@gmail.com>
// Licensed under the Apache License, Version 2.0

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/immutable-container/imf/pkg/container"
	imfcrypto "github.com/immutable-container/imf/pkg/crypto"
)

// runSeal handles the "imf seal" command.
// Sealing is the core operation that makes a container immutable:
//   1. Reads the Ed25519 private key from a PEM file
//   2. Optionally encrypts all files with AES-256-GCM (if passphrase provided)
//   3. Computes SHA-256 hashes for every file and records them in the manifest
//   4. Signs the manifest with the private key (Ed25519)
//   5. Optionally embeds the public key for self-verification
//   6. Writes a .sealed marker — after this, no modifications are possible
func runSeal() {
	// Parse command-line flags for key path, encryption, expiry, etc.
	keyPath, embedPub, passphrase, expiresStr, containerPath := parseSealArgs()

	if containerPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: imf seal <container.imf> [options]")
		fmt.Fprintln(os.Stderr, "\nOptions:")
		fmt.Fprintln(os.Stderr, "  -key string         Path to Ed25519 private key (PEM)")
		fmt.Fprintln(os.Stderr, "  -embed-pubkey       Embed public key in container")
		fmt.Fprintln(os.Stderr, "  -passphrase string  Encryption passphrase ('none' to skip)")
		fmt.Fprintln(os.Stderr, "  -expires string     Expiration time (RFC3339)")
		os.Exit(1)
	}

	// A signing key is always required — it proves authorship and enables
	// tamper detection via the Ed25519 signature on the manifest.
	if keyPath == "" {
		fmt.Fprintln(os.Stderr, "Error: -key is required")
		os.Exit(1)
	}
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading key: %v\n", err)
		os.Exit(1)
	}
	privKey, err := imfcrypto.ParsePrivateKeyPEM(keyData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing key: %v\n", err)
		os.Exit(1)
	}

	// Prompt for passphrase interactively if not provided via flag.
	// Use "none" to explicitly skip encryption.
	pp := passphrase
	if pp == "" {
		pp = promptPassphrase("Encryption passphrase (enter to skip): ")
	}
	if pp == "none" {
		pp = ""
	}

	// Build seal options and execute the seal operation.
	opts := container.SealOptions{
		PrivateKey:  privKey,
		EmbedPubKey: embedPub,
		Passphrase:  pp,
	}

	// Parse optional expiration date (RFC3339 format, e.g. "2026-12-31T23:59:59Z").
	// After expiry, extraction is blocked unless -ignore-expiry is used.
	if expiresStr != "" {
		t, err := time.Parse(time.RFC3339, expiresStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing expiry: %v\n", err)
			os.Exit(1)
		}
		opts.ExpiresAt = &t
	}

	if err := container.Seal(containerPath, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print summary of what was sealed and how.
	fmt.Printf("Sealed %s\n", containerPath)
	if pp != "" {
		fmt.Println("  Encrypted: yes")
	}
	if embedPub {
		fmt.Println("  Public key: embedded")
	}
	if opts.ExpiresAt != nil {
		fmt.Printf("  Expires: %s\n", opts.ExpiresAt.Format(time.RFC3339))
	}
}

// promptPassphrase reads a passphrase from stdin with a visible prompt.
func promptPassphrase(prompt string) string {
	fmt.Fprint(os.Stderr, prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// parseSealArgs manually parses seal command arguments.
// We use manual parsing instead of flag.FlagSet because the container path
// is a positional argument mixed with flags.
func parseSealArgs() (keyPath string, embedPub bool, passphrase string, expiresStr string, containerPath string) {
	args := os.Args[1:]
	i := 0
	for i < len(args) {
		switch args[i] {
		case "-key":
			if i+1 < len(args) {
				keyPath = args[i+1]
				i += 2
			} else {
				i++
			}
		case "-embed-pubkey":
			embedPub = true
			i++
		case "-passphrase":
			if i+1 < len(args) {
				passphrase = args[i+1]
				i += 2
			} else {
				i++
			}
		case "-expires":
			if i+1 < len(args) {
				expiresStr = args[i+1]
				i += 2
			} else {
				i++
			}
		case "-h", "-help":
			return
		default:
			if containerPath == "" && !strings.HasPrefix(args[i], "-") {
				containerPath = args[i]
			}
			i++
		}
	}
	return
}
