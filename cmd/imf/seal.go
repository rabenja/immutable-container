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

func runSeal() {
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

	pp := passphrase
	if pp == "" {
		pp = promptPassphrase("Encryption passphrase (enter to skip): ")
	}
	if pp == "none" {
		pp = ""
	}

	opts := container.SealOptions{
		PrivateKey:  privKey,
		EmbedPubKey: embedPub,
		Passphrase:  pp,
	}
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

func promptPassphrase(prompt string) string {
	fmt.Fprint(os.Stderr, prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

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
