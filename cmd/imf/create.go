// Copyright 2026 Benjamin Toso <benjamin.toso@gmail.com>
// Licensed under the Apache License, Version 2.0

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/immutable-container/imf/pkg/container"
)

func runCreate() {
	fs := flag.NewFlagSet("imf create", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: imf create <path.imf>")
		fmt.Fprintln(os.Stderr, "\nCreate a new empty .imf container.")
	}
	fs.Parse(os.Args[1:])

	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(1)
	}

	path := fs.Arg(0)
	if err := container.Create(path); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created %s\n", path)
}
