// Copyright 2026 Benjamin Toso <benjamin.toso@gmail.com>
// Licensed under the Apache License, Version 2.0

// IMF (Immutable File) is a CLI tool for creating cryptographically sealed,
// tamper-proof file containers. It uses Ed25519 signatures for integrity,
// AES-256-GCM for optional encryption, and a ZIP-based format for portability.
//
// Typical workflow:
//   imf keygen                              # Generate signing keys
//   imf create archive.imf                  # Create empty container
//   imf add archive.imf file1.pdf file2.txt # Add files
//   imf seal archive.imf -key imf_private.pem -embed-pubkey  # Seal forever
//   imf verify archive.imf                  # Verify integrity
//   imf extract archive.imf -out ./output   # Extract files
//   imf gui                                 # Or use the web-based GUI
package main

import (
	"fmt"
	"os"
)

const usage = `imf — Immutable File Container

Usage:
  imf <command> [options]

Commands:
  create    Create a new empty .imf container
  add       Add files to an open container
  seal      Seal a container (sign, optionally encrypt)
  verify    Verify a sealed container's integrity
  extract   Extract files from a container
  list      List files in a container
  info      Show container metadata
  keygen    Generate an Ed25519 key pair
  anchor    Anchor container hash to Bitcoin via OpenTimestamps
  gui       Launch the web-based graphical interface

Run 'imf <command> -h' for command-specific help.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(1)
	}

	cmd := os.Args[1]
	os.Args = append([]string{os.Args[0] + " " + cmd}, os.Args[2:]...)

	switch cmd {
	case "create":
		runCreate()
	case "add":
		runAdd()
	case "seal":
		runSeal()
	case "verify":
		runVerify()
	case "extract":
		runExtract()
	case "list":
		runList()
	case "info":
		runInfo()
	case "keygen":
		runKeygen()
	case "anchor":
		runAnchor()
	case "gui":
		runGUI()
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		fmt.Print(usage)
		os.Exit(1)
	}
}
