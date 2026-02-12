// Copyright 2026 Benjamin Toso <benjamin.toso@gmail.com>
// Licensed under the Apache License, Version 2.0

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/immutable-container/imf/pkg/container"
)

func runList() {
	fs := flag.NewFlagSet("imf list", flag.ExitOnError)
	fs.Parse(os.Args[1:])

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "Usage: imf list <container.imf>")
		os.Exit(1)
	}

	files, err := container.ListFiles(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Println("(empty)")
		return
	}

	fmt.Printf("%-30s %10s  %s\n", "NAME", "SIZE", "SHA256")
	fmt.Printf("%-30s %10s  %s\n", "----", "----", "------")
	for _, f := range files {
		fmt.Printf("%-30s %10d  %s\n", f.OriginalName, f.OriginalSize, f.SHA256[:16]+"...")
	}
	fmt.Printf("\n%d file(s)\n", len(files))
}
