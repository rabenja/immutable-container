// Copyright 2026 Benjamin Toso <benjamin.toso@gmail.com>
// Licensed under the Apache License, Version 2.0

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/immutable-container/imf/pkg/container"
	imfcrypto "github.com/immutable-container/imf/pkg/crypto"
)

func main() {
	os.Remove("demo.imf")
	os.RemoveAll("extracted")

	testContent := "This is my important document that must remain immutable."
	if err := os.WriteFile("testfile.txt", []byte(testContent), 0644); err != nil {
		fatal("creating test file", err)
	}
	fmt.Println("Created testfile.txt")

	if err := container.Create("demo.imf"); err != nil {
		fatal("creating container", err)
	}
	fmt.Println("Created demo.imf")

	if err := container.Add("demo.imf", []string{"testfile.txt"}); err != nil {
		fatal("adding file", err)
	}
	fmt.Println("Added testfile.txt to container")

	kp, err := imfcrypto.GenerateKeyPair()
	if err != nil {
		fatal("generating keys", err)
	}
	fmt.Println("Generated Ed25519 key pair")

	exp := time.Now().Add(24 * time.Hour)
	err = container.Seal("demo.imf", container.SealOptions{
		PrivateKey:  kp.PrivateKey,
		EmbedPubKey: true,
		Passphrase:  "my-secret-passphrase",
		ExpiresAt:   &exp,
	})
	if err != nil {
		fatal("sealing", err)
	}
	fmt.Println("Sealed container")

	if err := container.Verify("demo.imf", container.VerifyOptions{}); err != nil {
		fatal("verifying", err)
	}
	fmt.Println("Verified")

	err = container.Extract("demo.imf", container.ExtractOptions{
		Passphrase: "my-secret-passphrase",
		OutputDir:  "extracted",
	})
	if err != nil {
		fatal("extracting", err)
	}
	fmt.Println("Extracted to ./extracted/")

	extracted, err := os.ReadFile("extracted/testfile.txt")
	if err != nil {
		fatal("reading extracted file", err)
	}

	if string(extracted) == testContent {
		fmt.Println("PASS — content matches")
	} else {
		fmt.Println("FAIL — content mismatch")
		os.Exit(1)
	}
}

func fatal(action string, err error) {
	fmt.Fprintf(os.Stderr, "Error %s: %v\n", action, err)
	os.Exit(1)
}
