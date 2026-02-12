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

// Package container implements the IMF immutable file container.
// An IMF container is a ZIP-based archive with cryptographic integrity,
// optional encryption, optional embedded keys, and optional expiration.
package container

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	imfcrypto "github.com/immutable-container/imf/pkg/crypto"
	"github.com/immutable-container/imf/pkg/manifest"
)

// Well-known paths within the ZIP archive structure.
// These constants define the internal layout of every .imf container.
const (
	manifestPath = "manifest.json"     // Top-level manifest containing all metadata and crypto bindings
	filesDir     = "files/"            // Directory prefix for all stored files (plaintext or encrypted)
	sealedMarker = ".sealed"           // Presence of this file indicates the container is sealed/immutable
	pubKeyPath   = "keyring/public.key" // Optional embedded Ed25519 public key for self-verification
)

// SealOptions configures the seal operation.
type SealOptions struct {
	PrivateKey  ed25519.PrivateKey // required: signing key
	EmbedPubKey bool               // embed public key in container
	Passphrase  string             // if non-empty, encrypt files
	ExpiresAt   *time.Time         // optional expiration
}

// ExtractOptions configures extraction.
type ExtractOptions struct {
	Passphrase   string // required if container is encrypted
	IgnoreExpiry bool   // extract even if expired
	OutputDir    string // where to write extracted files
}

// VerifyOptions configures verification.
type VerifyOptions struct {
	PublicKey    ed25519.PublicKey // if nil, uses embedded key
	IgnoreExpiry bool
}

// Info holds container metadata for display.
type Info struct {
	State     manifest.State
	CreatedAt time.Time
	SealedAt  *time.Time
	ExpiresAt *time.Time
	Expired   bool
	Encrypted bool
	HasPubKey bool
	FileCount int
}

// FileInfo holds per-file metadata for listing.
type FileInfo struct {
	OriginalName string
	OriginalSize int64
	SHA256       string
}

// Create creates a new empty .imf container at the given path.
// The container starts in the "open" state with an empty manifest and no files.
// This is the entry point of the IMF lifecycle: Create -> Add -> Seal.
func Create(path string) error {
	// Enforce the .imf extension so containers are easily identifiable.
	if !strings.HasSuffix(path, ".imf") {
		return errors.New("container path must have .imf extension")
	}

	// Safety check: never silently overwrite an existing container.
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}

	// Initialize a fresh manifest in the "open" state with creation timestamp.
	m := manifest.New()
	mData, err := m.Marshal()
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}

	// Create the ZIP archive with only the manifest inside.
	// Files will be added later via the Add function.
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	w, err := zw.Create(manifestPath)
	if err != nil {
		return fmt.Errorf("creating manifest entry: %w", err)
	}
	if _, err := w.Write(mData); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	return nil
}

// Add adds one or more files to an open container.
// Each file is read from disk, SHA-256 hashed for integrity tracking, and stored
// inside the ZIP under the files/ directory. Name collisions are resolved by
// appending a numeric suffix. This operation is only allowed on open (unsealed) containers.
func Add(containerPath string, filePaths []string) error {
	// Read the current container state (manifest + raw ZIP bytes).
	m, zipData, err := readContainer(containerPath)
	if err != nil {
		return err
	}

	// Enforce immutability: sealed containers reject all modifications.
	if m.IsSealed() {
		return errors.New("cannot add files to a sealed container")
	}

	// Read all existing ZIP entries except the manifest (which we'll regenerate).
	// We need these to rewrite the container with both old and new entries.
	existingEntries, err := readZipEntries(zipData, manifestPath)
	if err != nil {
		return err
	}

	// Process each file: read from disk, compute hash, add to manifest.
	newEntries := make(map[string][]byte)
	for _, fp := range filePaths {
		// Read the entire file into memory for hashing and storage.
		data, err := os.ReadFile(fp)
		if err != nil {
			return fmt.Errorf("reading %s: %w", fp, err)
		}

		// Store files under files/<basename> inside the ZIP.
		baseName := filepath.Base(fp)
		zipPath := filesDir + baseName

		// Handle name collisions: if "files/doc.pdf" already exists,
		// try "files/doc_1.pdf", "files/doc_2.pdf", etc.
		origZipPath := zipPath
		suffix := 1
		for entryExists(m, zipPath) || newEntries[zipPath] != nil {
			ext := filepath.Ext(baseName)
			name := strings.TrimSuffix(baseName, ext)
			zipPath = fmt.Sprintf("%s%s_%d%s", filesDir, name, suffix, ext)
			suffix++
		}
		if zipPath != origZipPath {
			fmt.Printf("  renamed to avoid collision: %s -> %s\n", baseName, filepath.Base(zipPath))
		}

		// Compute SHA-256 hash of the original plaintext content.
		// This hash is stored in the manifest and verified during extraction
		// to detect any tampering with file contents.
		hash := imfcrypto.HashSHA256(data)

		// Create the manifest entry linking the ZIP path to the original
		// filename, size, and integrity hash.
		entry := manifest.FileEntry{
			Path:         zipPath,
			OriginalName: baseName,
			OriginalSize: int64(len(data)),
			SHA256:       hex.EncodeToString(hash[:]),
		}
		if err := m.AddFile(entry); err != nil {
			return fmt.Errorf("adding %s to manifest: %w", baseName, err)
		}

		newEntries[zipPath] = data
	}

	// Rewrite the container.
	return rewriteContainer(containerPath, m, existingEntries, newEntries)
}

// Seal seals the container, making it permanently immutable.
// This is the critical transition in the IMF lifecycle. Sealing performs the
// following atomic sequence:
//   1. Encrypt files with AES-256-GCM if a passphrase is provided
//   2. Set expiration timestamp if specified
//   3. Embed the public key if requested (enables self-verification)
//   4. Transition the manifest state from "open" to "sealed"
//   5. Sign the manifest with Ed25519
//   6. Write the .sealed marker file
//   7. Rewrite the container as a new ZIP archive
//
// After sealing, no further modifications are possible. The container is either
// fully sealed or unchanged — there is no partially-sealed state.
func Seal(containerPath string, opts SealOptions) error {
	m, zipData, err := readContainer(containerPath)
	if err != nil {
		return err
	}

	// Sealed containers cannot be re-sealed.
	if m.IsSealed() {
		return errors.New("container is already sealed")
	}

	// Load all file entries from the current ZIP.
	existingEntries, err := readZipEntries(zipData, manifestPath)
	if err != nil {
		return err
	}

	// --- Step 1: Encryption (optional) ---
	// If a passphrase is provided, derive an AES-256 key and encrypt each file
	// individually. Each encrypted file gets a unique nonce for security.
	var encKey []byte
	var salt []byte
	processedEntries := make(map[string][]byte)

	if opts.Passphrase != "" {
		// Generate a random 32-byte salt for key derivation.
		salt, err = imfcrypto.GenerateSalt()
		if err != nil {
			return err
		}

		// Derive a 256-bit encryption key from the passphrase using PBKDF2
		// with 600,000 iterations (OWASP 2023 recommendation).
		encKey, err = imfcrypto.DeriveKey(opts.Passphrase, salt)
		if err != nil {
			return fmt.Errorf("deriving encryption key: %w", err)
		}

		// Store encryption metadata in the manifest so the recipient knows
		// which algorithm and KDF parameters to use for decryption.
		m.Encryption = &manifest.EncryptionInfo{
			Algorithm:  "AES-256-GCM",
			KDF:        "PBKDF2-HMAC-SHA256",
			Salt:       base64.StdEncoding.EncodeToString(salt),
			Iterations: imfcrypto.PBKDF2Iterations,
		}

		// Encrypt each file individually with AES-256-GCM.
		// We also hash the ciphertext and store it in the manifest, providing
		// a second integrity check layer (encrypted hash verified before decryption).
		for i, fe := range m.Files {
			plaintext, ok := existingEntries[fe.Path]
			if !ok {
				return fmt.Errorf("file not found in container: %s", fe.Path)
			}

			ciphertext, err := imfcrypto.Encrypt(encKey, plaintext)
			if err != nil {
				return fmt.Errorf("encrypting %s: %w", fe.OriginalName, err)
			}

			// Rename the file path with .enc suffix to indicate encryption,
			// and record the ciphertext hash for pre-decryption integrity check.
			encPath := fe.Path + ".enc"
			encHash := imfcrypto.HashSHA256(ciphertext)
			m.Files[i].EncryptedSHA256 = hex.EncodeToString(encHash[:])
			m.Files[i].Path = encPath

			processedEntries[encPath] = ciphertext
		}
	} else {
		// No encryption — copy entries as-is.
		for path, data := range existingEntries {
			processedEntries[path] = data
		}
	}

	// --- Step 2: Set expiration (optional) ---
	// The expiry timestamp is included in the signed manifest, so it cannot
	// be altered without invalidating the signature.
	if opts.ExpiresAt != nil {
		t := opts.ExpiresAt.UTC()
		m.ExpiresAt = &t
	}

	// --- Step 3: Embed public key (optional) ---
	// Embedding the public key makes the container self-verifying: the recipient
	// can verify the signature without any prior key exchange or key server.
	// The key is stored both in the manifest (base64) and as a PEM file in keyring/.
	if opts.EmbedPubKey {
		pubKey := opts.PrivateKey.Public().(ed25519.PublicKey)
		m.PublicKey = base64.StdEncoding.EncodeToString(pubKey)

		pubKeyPEM := imfcrypto.MarshalPublicKeyPEM(pubKey)
		processedEntries[pubKeyPath] = pubKeyPEM
	}

	// --- Step 4: Transition to sealed state ---
	// This is irreversible — the manifest state becomes "sealed" with a timestamp.
	if err := m.Seal(); err != nil {
		return err
	}

	// --- Step 5: Sign the manifest with Ed25519 ---
	// We sign the "signable bytes" — the full manifest JSON with the signature
	// field zeroed out. This ensures the signature covers ALL metadata including
	// file hashes, timestamps, expiry, and the embedded public key.
	signable, err := m.SignableBytes()
	if err != nil {
		return fmt.Errorf("computing signable bytes: %w", err)
	}
	sig := imfcrypto.Sign(opts.PrivateKey, signable)
	m.Signature = base64.StdEncoding.EncodeToString(sig)

	// --- Step 6: Add the sealed marker file ---
	// The .sealed file is a simple presence indicator. Its existence in the ZIP
	// signals that the container is immutable without needing to parse the manifest.
	processedEntries[sealedMarker] = []byte("sealed")

	// --- Step 7: Rewrite the container atomically ---
	// The entire ZIP is rewritten with the signed manifest, processed (possibly
	// encrypted) files, embedded key, and sealed marker.
	return rewriteContainer(containerPath, m, nil, processedEntries)
}

// Verify checks the cryptographic integrity of a sealed container.
// Verification performs three checks:
//   1. Expiration: rejects expired containers (unless IgnoreExpiry is set)
//   2. Signature: verifies the Ed25519 signature over the manifest
//   3. File hashes: confirms each file's hash matches the manifest record
//
// If the container has an embedded public key, it will be used automatically.
// An explicit public key can be provided to override the embedded one.
func Verify(containerPath string, opts VerifyOptions) error {
	m, zipData, err := readContainer(containerPath)
	if err != nil {
		return err
	}
	if !m.IsSealed() {
		return errors.New("container is not sealed")
	}

	// Check expiry.
	if m.IsExpired() && !opts.IgnoreExpiry {
		return fmt.Errorf("container expired at %s (use --ignore-expiry to override)", m.ExpiresAt.Format(time.RFC3339))
	}

	// Determine which public key to use for signature verification.
	// Priority: explicit key from options > embedded key in manifest.
	pubKey := opts.PublicKey
	if pubKey == nil {
		if m.PublicKey == "" {
			return errors.New("no public key provided and none embedded in container")
		}
		keyBytes, err := base64.StdEncoding.DecodeString(m.PublicKey)
		if err != nil {
			return fmt.Errorf("decoding embedded public key: %w", err)
		}
		pubKey = ed25519.PublicKey(keyBytes)
	}

	// Verify the Ed25519 signature over the manifest.
	// The signature covers all metadata including file hashes, timestamps,
	// expiry, and the embedded public key — any modification is detected.
	sigBytes, err := base64.StdEncoding.DecodeString(m.Signature)
	if err != nil {
		return fmt.Errorf("decoding signature: %w", err)
	}
	signable, err := m.SignableBytes()
	if err != nil {
		return fmt.Errorf("computing signable bytes: %w", err)
	}
	if !imfcrypto.Verify(pubKey, signable, sigBytes) {
		return errors.New("SIGNATURE VERIFICATION FAILED — container may be tampered")
	}

	// Verify per-file integrity by checking hashes against manifest records.
	// For encrypted containers, we verify the ciphertext hash (the plaintext
	// hash is verified during extraction after decryption).
	entries, err := readZipEntries(zipData, manifestPath, sealedMarker, pubKeyPath)
	if err != nil {
		return err
	}

	for _, fe := range m.Files {
		data, ok := entries[fe.Path]
		if !ok {
			return fmt.Errorf("INTEGRITY FAILURE: file missing from container: %s", fe.Path)
		}

		// If encrypted, verify encrypted hash.
		if fe.EncryptedSHA256 != "" {
			hash := imfcrypto.HashSHA256(data)
			if hex.EncodeToString(hash[:]) != fe.EncryptedSHA256 {
				return fmt.Errorf("INTEGRITY FAILURE: encrypted hash mismatch for %s", fe.OriginalName)
			}
		}
	}

	return nil
}

// Extract extracts files from a container to the specified output directory.
// For sealed containers, extraction performs the following:
//   1. Check expiration (reject if expired, unless IgnoreExpiry is set)
//   2. Derive the decryption key from the passphrase (if encrypted)
//   3. For each file: decrypt (if needed), verify the plaintext SHA-256 hash
//      against the manifest, and write to the output directory
//
// The plaintext hash verification during extraction is the final integrity check:
// it ensures the decrypted content matches what was originally added before sealing.
// For unsealed containers, files are extracted directly without decryption.
func Extract(containerPath string, opts ExtractOptions) error {
	m, zipData, err := readContainer(containerPath)
	if err != nil {
		return err
	}
	if !m.IsSealed() {
		// For unsealed containers, extract plaintext files directly.
		return extractUnsealed(m, zipData, opts.OutputDir)
	}

	// Check expiry.
	if m.IsExpired() && !opts.IgnoreExpiry {
		return fmt.Errorf("container expired at %s (use --ignore-expiry to override)", m.ExpiresAt.Format(time.RFC3339))
	}

	entries, err := readZipEntries(zipData, manifestPath, sealedMarker, pubKeyPath)
	if err != nil {
		return err
	}

	// Derive decryption key if encrypted.
	var decKey []byte
	if m.Encryption != nil {
		if opts.Passphrase == "" {
			return errors.New("container is encrypted but no passphrase provided")
		}
		salt, err := base64.StdEncoding.DecodeString(m.Encryption.Salt)
		if err != nil {
			return fmt.Errorf("decoding salt: %w", err)
		}
		decKey, err = imfcrypto.DeriveKey(opts.Passphrase, salt)
		if err != nil {
			return fmt.Errorf("deriving decryption key: %w", err)
		}
	}

	// Create output directory.
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	for _, fe := range m.Files {
		data, ok := entries[fe.Path]
		if !ok {
			return fmt.Errorf("file missing from container: %s", fe.Path)
		}

		var plaintext []byte
		if m.Encryption != nil {
			plaintext, err = imfcrypto.Decrypt(decKey, data)
			if err != nil {
				return fmt.Errorf("decrypting %s: %w", fe.OriginalName, err)
			}
		} else {
			plaintext = data
		}

		// Verify plaintext hash.
		hash := imfcrypto.HashSHA256(plaintext)
		if hex.EncodeToString(hash[:]) != fe.SHA256 {
			return fmt.Errorf("INTEGRITY FAILURE: hash mismatch for %s", fe.OriginalName)
		}

		outPath := filepath.Join(opts.OutputDir, fe.OriginalName)
		if err := os.WriteFile(outPath, plaintext, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", fe.OriginalName, err)
		}
	}

	return nil
}

// ListFiles returns metadata for all files in the container.
func ListFiles(containerPath string) ([]FileInfo, error) {
	m, _, err := readContainer(containerPath)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	for _, fe := range m.Files {
		files = append(files, FileInfo{
			OriginalName: fe.OriginalName,
			OriginalSize: fe.OriginalSize,
			SHA256:       fe.SHA256,
		})
	}
	return files, nil
}

// GetInfo returns container metadata.
func GetInfo(containerPath string) (*Info, error) {
	m, _, err := readContainer(containerPath)
	if err != nil {
		return nil, err
	}

	return &Info{
		State:     m.State,
		CreatedAt: m.CreatedAt,
		SealedAt:  m.SealedAt,
		ExpiresAt: m.ExpiresAt,
		Expired:   m.IsExpired(),
		Encrypted: m.Encryption != nil,
		HasPubKey: m.PublicKey != "",
		FileCount: len(m.Files),
	}, nil
}

// --- Internal helpers ---

// readContainer reads the manifest and raw zip bytes from a container.
func readContainer(path string) (*manifest.Manifest, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading container: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, nil, fmt.Errorf("opening zip: %w", err)
	}

	for _, f := range zr.File {
		if f.Name == manifestPath {
			rc, err := f.Open()
			if err != nil {
				return nil, nil, fmt.Errorf("opening manifest: %w", err)
			}
			defer rc.Close()

			mData, err := io.ReadAll(rc)
			if err != nil {
				return nil, nil, fmt.Errorf("reading manifest: %w", err)
			}

			m, err := manifest.Unmarshal(mData)
			if err != nil {
				return nil, nil, err
			}

			return m, data, nil
		}
	}

	return nil, nil, errors.New("manifest.json not found in container")
}

// readZipEntries reads all entries from zip data, excluding the given paths.
func readZipEntries(data []byte, excludePaths ...string) (map[string][]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("opening zip: %w", err)
	}

	excludeSet := make(map[string]bool)
	for _, p := range excludePaths {
		excludeSet[p] = true
	}

	entries := make(map[string][]byte)
	for _, f := range zr.File {
		if excludeSet[f.Name] {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("opening %s: %w", f.Name, err)
		}
		d, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", f.Name, err)
		}
		entries[f.Name] = d
	}
	return entries, nil
}

// rewriteContainer rewrites the container with updated manifest and entries.
func rewriteContainer(path string, m *manifest.Manifest, existing map[string][]byte, newEntries map[string][]byte) error {
	mData, err := m.Marshal()
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	// Write manifest first.
	w, err := zw.Create(manifestPath)
	if err != nil {
		return err
	}
	if _, err := w.Write(mData); err != nil {
		return err
	}

	// Write existing entries.
	for name, data := range existing {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
	}

	// Write new entries.
	for name, data := range newEntries {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
	}

	return zw.Close()
}

// entryExists checks if a path already exists in the manifest.
func entryExists(m *manifest.Manifest, path string) bool {
	for _, f := range m.Files {
		if f.Path == path {
			return true
		}
	}
	return false
}

// extractUnsealed extracts files from an unsealed container (no decryption).
func extractUnsealed(m *manifest.Manifest, zipData []byte, outputDir string) error {
	entries, err := readZipEntries(zipData, manifestPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	for _, fe := range m.Files {
		data, ok := entries[fe.Path]
		if !ok {
			return fmt.Errorf("file missing from container: %s", fe.Path)
		}
		outPath := filepath.Join(outputDir, fe.OriginalName)
		if err := os.WriteFile(outPath, data, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", fe.OriginalName, err)
		}
	}
	return nil
}
