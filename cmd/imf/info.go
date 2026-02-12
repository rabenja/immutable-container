// Copyright 2026 Benjamin Toso <benjamin.toso@gmail.com>
// Licensed under the Apache License, Version 2.0

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/immutable-container/imf/pkg/container"
)

// runInfo handles the "imf info" command.
// Displays metadata about a container: state (open/sealed), creation and seal
// timestamps, expiration status, encryption status, embedded key presence,
// and file count. Does not require decryption or key access.
func runInfo() {
	fs := flag.NewFlagSet("imf info", flag.ExitOnError)
	fs.Parse(os.Args[1:])

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "Usage: imf info <container.imf>")
		os.Exit(1)
	}

	info, err := container.GetInfo(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Container: %s\n", fs.Arg(0))
	fmt.Printf("  State:     %s\n", info.State)
	fmt.Printf("  Created:   %s\n", info.CreatedAt.Format(time.RFC3339))

	if info.SealedAt != nil {
		fmt.Printf("  Sealed:    %s\n", info.SealedAt.Format(time.RFC3339))
	}
	if info.ExpiresAt != nil {
		expStr := info.ExpiresAt.Format(time.RFC3339)
		if info.Expired {
			expStr += " (EXPIRED)"
		}
		fmt.Printf("  Expires:   %s\n", expStr)
	}

	fmt.Printf("  Encrypted: %v\n", info.Encrypted)
	fmt.Printf("  Pub Key:   %v\n", info.HasPubKey)
	fmt.Printf("  Files:     %d\n", info.FileCount)
}
