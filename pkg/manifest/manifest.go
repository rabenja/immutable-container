// Copyright 2026 Benjamin Toso <benjamin.toso@gmail.com>
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package manifest defines the IMF container manifest structure.
// The manifest tracks all files, cryptographic metadata, and container lifecycle state.
package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Version is the current manifest schema version.
const Version = 1

// State represents the container lifecycle state.
type State string

const (
	StateOpen   State = "open"
	StateSealed State = "sealed"
)

// EncryptionInfo holds encryption-related metadata.
type EncryptionInfo struct {
	Algorithm  string `json:"algorithm"`            // e.g., "AES-256-GCM"
	KDF        string `json:"kdf"`                  // e.g., "PBKDF2-HMAC-SHA256"
	Salt       string `json:"salt"`                 // base64-encoded salt
	Iterations int    `json:"iterations,omitempty"` // KDF iterations
}

// FileEntry describes a single file stored in the container.
type FileEntry struct {
	Path            string `json:"path"`                       // path inside zip (e.g., "files/doc.pdf.enc")
	OriginalName    string `json:"original_name"`              // original filename
	OriginalSize    int64  `json:"original_size"`              // size before encryption
	SHA256          string `json:"sha256"`                     // hash of original plaintext content
	EncryptedSHA256 string `json:"encrypted_sha256,omitempty"` // hash of encrypted content
}

// Manifest is the top-level container metadata.
type Manifest struct {
	Version    int            `json:"version"`
	State      State          `json:"state"`
	CreatedAt  time.Time      `json:"created_at"`
	SealedAt   *time.Time     `json:"sealed_at,omitempty"`
	ExpiresAt  *time.Time     `json:"expires_at,omitempty"`
	PublicKey  string         `json:"public_key,omitempty"`   // base64-encoded Ed25519 public key
	Encryption *EncryptionInfo `json:"encryption,omitempty"`
	Files      []FileEntry    `json:"files"`
	Signature  string         `json:"signature,omitempty"` // base64-encoded Ed25519 signature
}

// New creates a new open manifest.
func New() *Manifest {
	return &Manifest{
		Version:   Version,
		State:     StateOpen,
		CreatedAt: time.Now().UTC(),
		Files:     []FileEntry{},
	}
}

// AddFile adds a file entry to the manifest. Fails if sealed.
func (m *Manifest) AddFile(entry FileEntry) error {
	if m.State == StateSealed {
		return errors.New("cannot add files to a sealed container")
	}

	// Check for duplicate paths.
	for _, f := range m.Files {
		if f.Path == entry.Path {
			return fmt.Errorf("duplicate file path: %s", entry.Path)
		}
	}

	m.Files = append(m.Files, entry)
	return nil
}

// IsSealed returns true if the container is sealed.
func (m *Manifest) IsSealed() bool {
	return m.State == StateSealed
}

// IsExpired returns true if the container has an expiration date that has passed.
func (m *Manifest) IsExpired() bool {
	if m.ExpiresAt == nil {
		return false
	}
	return time.Now().UTC().After(*m.ExpiresAt)
}

// Seal transitions the manifest to sealed state.
func (m *Manifest) Seal() error {
	if m.State == StateSealed {
		return errors.New("container is already sealed")
	}
	if len(m.Files) == 0 {
		return errors.New("cannot seal an empty container")
	}
	now := time.Now().UTC()
	m.SealedAt = &now
	m.State = StateSealed
	return nil
}

// SignableBytes returns the manifest bytes used for signing.
// This is the JSON representation with the signature field zeroed out.
func (m *Manifest) SignableBytes() ([]byte, error) {
	// Create a copy with no signature for signing.
	cp := *m
	cp.Signature = ""
	return json.Marshal(cp)
}

// Marshal serializes the manifest to JSON.
func (m *Manifest) Marshal() ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// Unmarshal deserializes JSON into a manifest.
func Unmarshal(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}
	if m.Version == 0 {
		return nil, errors.New("invalid manifest: missing version")
	}
	if m.Version > Version {
		return nil, fmt.Errorf("unsupported manifest version: %d (max supported: %d)", m.Version, Version)
	}
	return &m, nil
}
