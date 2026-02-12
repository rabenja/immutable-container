// Copyright 2026 Benjamin Toso <benjamin.toso@gmail.com>
// Licensed under the Apache License, Version 2.0

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/immutable-container/imf/pkg/container"
)

// runExtract handles the "imf extract" command.
// Extracts files from a sealed container. If the container is encrypted,
// the correct passphrase must be provided (interactively or via -passphrase flag).
// Expired containers are blocked by default — use -ignore-expiry for forensic access.
func runExtract() {
	outputDir, passphrase, ignoreExpiry, containerPath := parseExtractArgs()

	if containerPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: imf extract <container.imf> [options]")
		fmt.Fprintln(os.Stderr, "\nOptions:")
		fmt.Fprintln(os.Stderr, "  -out string         Output directory (default \".\")")
		fmt.Fprintln(os.Stderr, "  -passphrase string  Decryption passphrase")
		fmt.Fprintln(os.Stderr, "  -ignore-expiry      Extract even if expired")
		os.Exit(1)
	}

	pp := passphrase
	if pp == "" {
		info, err := container.GetInfo(containerPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if info.Encrypted {
			pp = promptPassphrase("Decryption passphrase: ")
			if pp == "" {
				fmt.Fprintln(os.Stderr, "Error: container is encrypted, passphrase required")
				os.Exit(1)
			}
		}
	}

	err := container.Extract(containerPath, container.ExtractOptions{
		Passphrase:   pp,
		IgnoreExpiry: ignoreExpiry,
		OutputDir:    outputDir,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Extracted to %s\n", outputDir)
}

// parseExtractArgs manually parses extract command arguments.
// Uses manual parsing because the container path is positional.
func parseExtractArgs() (outputDir string, passphrase string, ignoreExpiry bool, containerPath string) {
	outputDir = "."
	args := os.Args[1:]
	i := 0
	for i < len(args) {
		switch args[i] {
		case "-out":
			if i+1 < len(args) {
				outputDir = args[i+1]
				i += 2
			} else {
				i++
			}
		case "-passphrase":
			if i+1 < len(args) {
				passphrase = args[i+1]
				i += 2
			} else {
				i++
			}
		case "-ignore-expiry":
			ignoreExpiry = true
			i++
		default:
			if containerPath == "" && !strings.HasPrefix(args[i], "-") {
				containerPath = args[i]
			}
			i++
		}
	}
	return
}
