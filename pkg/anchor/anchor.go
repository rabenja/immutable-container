// Copyright 2026 Benjamin Toso <benjamin.toso@gmail.com>
// Licensed under the Apache License, Version 2.0

// Package anchor provides blockchain timestamping for IMF containers via OpenTimestamps.
//
// OpenTimestamps (https://opentimestamps.org) is a free, open-source protocol that
// anchors SHA-256 digests to the Bitcoin blockchain. The process:
//
//  1. Submit: POST the container's SHA-256 hash to an OTS calendar server
//  2. Receive: Get back a compact proof (.ots file) proving the timestamp
//  3. Verify: Anyone can independently verify the proof against Bitcoin
//
// The proof is initially "pending" — it takes a few hours for the calendar server
// to batch multiple timestamps into a single Bitcoin transaction. After confirmation,
// the proof upgrades to a full Bitcoin attestation.
//
// No accounts, API keys, wallets, or tokens are required.
package anchor

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Default OpenTimestamps calendar servers.
// Multiple servers are tried in order for redundancy.
var calendarServers = []string{
	"https://a.pool.opentimestamps.org",
	"https://b.pool.opentimestamps.org",
	"https://a.pool.eternitywall.com",
}

// AnchorResult contains the result of a timestamping operation.
type AnchorResult struct {
	ContainerHash string    // SHA-256 hex digest of the .imf file
	ProofPath     string    // Path where the .ots proof file was saved
	Server        string    // Calendar server that accepted the submission
	Timestamp     time.Time // When the submission was made
}

// AnchorContainer computes the SHA-256 hash of a sealed .imf container and
// submits it to OpenTimestamps for blockchain anchoring. The proof receipt
// is saved as <containerPath>.ots alongside the container.
//
// Returns an AnchorResult with the hash, proof path, and server used.
func AnchorContainer(containerPath string) (*AnchorResult, error) {
	// Read the entire container and compute its SHA-256 hash.
	data, err := os.ReadFile(containerPath)
	if err != nil {
		return nil, fmt.Errorf("reading container: %w", err)
	}

	hash := sha256.Sum256(data)
	hashHex := hex.EncodeToString(hash[:])

	// Submit the raw 32-byte digest to an OpenTimestamps calendar server.
	// The server returns an OTS proof file (binary format).
	var proof []byte
	var usedServer string

	for _, server := range calendarServers {
		url := server + "/digest"
		proof, err = submitDigest(url, hash[:])
		if err == nil {
			usedServer = server
			break
		}
	}

	if proof == nil {
		return nil, errors.New("all OpenTimestamps servers failed — check your internet connection")
	}

	// Save the proof receipt alongside the container.
	// e.g., "archive.imf" → "archive.imf.ots"
	proofPath := containerPath + ".ots"
	if err := os.WriteFile(proofPath, proof, 0644); err != nil {
		return nil, fmt.Errorf("saving proof: %w", err)
	}

	return &AnchorResult{
		ContainerHash: hashHex,
		ProofPath:     proofPath,
		Server:        usedServer,
		Timestamp:     time.Now(),
	}, nil
}

// VerifyAnchor checks that a .ots proof file matches the container's hash.
// This is a local check only — it confirms the proof was generated for this
// specific container. Full Bitcoin verification requires an OTS verifier.
func VerifyAnchor(containerPath string) (*VerifyResult, error) {
	// Read container and compute hash.
	data, err := os.ReadFile(containerPath)
	if err != nil {
		return nil, fmt.Errorf("reading container: %w", err)
	}
	hash := sha256.Sum256(data)
	hashHex := hex.EncodeToString(hash[:])

	// Read the proof file.
	proofPath := containerPath + ".ots"
	proof, err := os.ReadFile(proofPath)
	if err != nil {
		return nil, fmt.Errorf("reading proof file: %w", err)
	}

	// Check that the proof contains the expected hash.
	// OTS proof files embed the original digest — verify it matches.
	if !bytes.Contains(proof, hash[:]) {
		return nil, errors.New("proof does not match container — container may have been modified after anchoring")
	}

	return &VerifyResult{
		ContainerHash: hashHex,
		ProofPath:     proofPath,
		ProofSize:     len(proof),
		HashMatches:   true,
	}, nil
}

// VerifyResult contains the result of a local anchor verification.
type VerifyResult struct {
	ContainerHash string // SHA-256 hex digest of the .imf file
	ProofPath     string // Path to the .ots proof file
	ProofSize     int    // Size of the proof in bytes
	HashMatches   bool   // Whether the proof matches the container hash
}

// submitDigest POSTs a raw 32-byte SHA-256 digest to an OTS calendar server.
// Returns the binary OTS proof on success.
func submitDigest(url string, digest []byte) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequest("POST", url, bytes.NewReader(digest))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/vnd.opentimestamps.v1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server %s returned status %d", url, resp.StatusCode)
	}

	proof, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if len(proof) == 0 {
		return nil, errors.New("empty proof received")
	}

	return proof, nil
}
