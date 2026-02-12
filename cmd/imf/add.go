// Copyright 2026 Benjamin Toso <benjamin.toso@gmail.com>
// Licensed under the Apache License, Version 2.0

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/immutable-container/imf/pkg/container"
)

func runAdd() {
	fs := flag.NewFlagSet("imf add", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: imf add <container.imf> <file1> [file2 ...]")
		fmt.Fprintln(os.Stderr, "\nAdd files to an open container.")
	}
	fs.Parse(os.Args[1:])

	if fs.NArg() < 2 {
		fs.Usage()
		os.Exit(1)
	}

	containerPath := fs.Arg(0)
	filePaths := fs.Args()[1:]

	if err := container.Add(containerPath, filePaths); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Added %d file(s) to %s\n", len(filePaths), containerPath)
}
