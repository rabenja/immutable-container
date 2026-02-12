// Copyright 2026 Benjamin Toso <benjamin.toso@gmail.com>
// Licensed under the Apache License, Version 2.0

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/immutable-container/imf/pkg/anchor"
	"github.com/immutable-container/imf/pkg/container"
)

// runAnchor handles the "imf anchor" command.
// Submits the SHA-256 hash of a sealed .imf container to OpenTimestamps,
// which anchors it to the Bitcoin blockchain. The proof receipt (.ots file)
// is saved alongside the container. This provides a third-party, immutable
// timestamp proving the container existed at a specific point in time.
//
// Usage:
//   imf anchor archive.imf          # Submit hash and save proof
//   imf anchor archive.imf -verify  # Verify existing proof matches container
func runAnchor() {
	fs := flag.NewFlagSet("imf anchor", flag.ExitOnError)
	verify := fs.Bool("verify", false, "Verify existing .ots proof instead of creating one")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: imf anchor <container.imf> [options]")
		fmt.Fprintln(os.Stderr, "\nAnchor a sealed container's hash to the Bitcoin blockchain")
		fmt.Fprintln(os.Stderr, "via OpenTimestamps. No accounts or fees required.")
		fmt.Fprintln(os.Stderr, "\nOptions:")
		fmt.Fprintln(os.Stderr, "  -verify  Verify existing .ots proof matches the container")
	}
	fs.Parse(os.Args[1:])

	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(1)
	}

	containerPath := fs.Arg(0)

	// Verify the container is sealed before anchoring — anchoring an open
	// container would be pointless since its contents can still change.
	info, err := container.GetInfo(containerPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if info.State != "sealed" {
		fmt.Fprintln(os.Stderr, "Error: container must be sealed before anchoring")
		fmt.Fprintln(os.Stderr, "  Run: imf seal <container.imf> -key <private.pem>")
		os.Exit(1)
	}

	if *verify {
		// Verify mode: check that existing .ots proof matches the container.
		result, err := anchor.VerifyAnchor(containerPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAILED: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("OK — proof matches container")
		fmt.Printf("  Container hash: %s\n", result.ContainerHash)
		fmt.Printf("  Proof file:     %s\n", result.ProofPath)
		fmt.Printf("  Proof size:     %d bytes\n", result.ProofSize)
		fmt.Println("\n  Note: For full Bitcoin verification, use the OpenTimestamps")
		fmt.Println("  verifier at https://opentimestamps.org or the ots CLI tool.")
	} else {
		// Anchor mode: submit hash to OpenTimestamps.
		fmt.Printf("Anchoring %s to Bitcoin via OpenTimestamps...\n", containerPath)

		result, err := anchor.AnchorContainer(containerPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Anchored successfully!")
		fmt.Printf("  Container hash: %s\n", result.ContainerHash)
		fmt.Printf("  Proof saved:    %s\n", result.ProofPath)
		fmt.Printf("  Server:         %s\n", result.Server)
		fmt.Printf("  Submitted:      %s\n", result.Timestamp.Format("2006-01-02 15:04:05 MST"))
		fmt.Println("\n  The proof will be confirmed on the Bitcoin blockchain within")
		fmt.Println("  a few hours. Keep the .ots file alongside your .imf container.")
		fmt.Println("  Verify anytime: imf anchor <container.imf> -verify")
		fmt.Println("  Full verification: https://opentimestamps.org")
	}
}
