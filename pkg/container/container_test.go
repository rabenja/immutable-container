package container_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/immutable-container/imf/pkg/container"
	imfcrypto "github.com/immutable-container/imf/pkg/crypto"
)

func TestFullLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	imfPath := filepath.Join(tmpDir, "test.imf")

	// 1. Create container.
	if err := container.Create(imfPath); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Log("✓ Created container")

	// 2. Create test files.
	testFiles := map[string]string{
		"hello.txt":   "Hello, immutable world!",
		"data.csv":    "name,value\nalpha,1\nbeta,2\n",
		"readme.md":   "# IMF Test\nThis is a test file.\n",
	}
	var filePaths []string
	for name, content := range testFiles {
		p := filepath.Join(tmpDir, name)
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
		filePaths = append(filePaths, p)
	}

	// 3. Add files.
	if err := container.Add(imfPath, filePaths); err != nil {
		t.Fatalf("Add: %v", err)
	}
	t.Log("✓ Added files")

	// 4. List files.
	files, err := container.ListFiles(imfPath)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}
	t.Logf("✓ Listed %d files", len(files))

	// 5. Info before seal.
	info, err := container.GetInfo(imfPath)
	if err != nil {
		t.Fatalf("GetInfo: %v", err)
	}
	if info.State != "open" {
		t.Fatalf("expected open state, got %s", info.State)
	}
	t.Log("✓ Info shows open state")

	// 6. Generate keys.
	kp, err := imfcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	t.Log("✓ Generated key pair")

	// 7. Seal with encryption + embedded key + expiry.
	expires := time.Now().Add(24 * time.Hour)
	err = container.Seal(imfPath, container.SealOptions{
		PrivateKey:  kp.PrivateKey,
		EmbedPubKey: true,
		Passphrase:  "test-passphrase-123",
		ExpiresAt:   &expires,
	})
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	t.Log("✓ Sealed container")

	// 8. Info after seal.
	info, err = container.GetInfo(imfPath)
	if err != nil {
		t.Fatalf("GetInfo after seal: %v", err)
	}
	if info.State != "sealed" {
		t.Fatalf("expected sealed state, got %s", info.State)
	}
	if !info.Encrypted {
		t.Fatal("expected encrypted")
	}
	if !info.HasPubKey {
		t.Fatal("expected embedded public key")
	}
	if info.ExpiresAt == nil {
		t.Fatal("expected expiry")
	}
	t.Log("✓ Info shows sealed, encrypted, with pubkey and expiry")

	// 9. Verify (using embedded key).
	err = container.Verify(imfPath, container.VerifyOptions{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	t.Log("✓ Verified signature and integrity")

	// 10. Verify with explicit key.
	err = container.Verify(imfPath, container.VerifyOptions{PublicKey: kp.PublicKey})
	if err != nil {
		t.Fatalf("Verify with explicit key: %v", err)
	}
	t.Log("✓ Verified with explicit public key")

	// 11. Cannot add to sealed container.
	err = container.Add(imfPath, filePaths[:1])
	if err == nil {
		t.Fatal("expected error adding to sealed container")
	}
	t.Log("✓ Add to sealed container correctly rejected")

	// 12. Cannot re-seal.
	err = container.Seal(imfPath, container.SealOptions{PrivateKey: kp.PrivateKey})
	if err == nil {
		t.Fatal("expected error re-sealing")
	}
	t.Log("✓ Re-seal correctly rejected")

	// 13. Extract with correct passphrase.
	extractDir := filepath.Join(tmpDir, "extracted")
	err = container.Extract(imfPath, container.ExtractOptions{
		Passphrase: "test-passphrase-123",
		OutputDir:  extractDir,
	})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	// Verify extracted contents.
	for name, expectedContent := range testFiles {
		data, err := os.ReadFile(filepath.Join(extractDir, name))
		if err != nil {
			t.Fatalf("reading extracted %s: %v", name, err)
		}
		if string(data) != expectedContent {
			t.Fatalf("content mismatch for %s: got %q, want %q", name, string(data), expectedContent)
		}
	}
	t.Log("✓ Extracted and verified all file contents match originals")

	// 14. Extract with wrong passphrase.
	badDir := filepath.Join(tmpDir, "bad-extract")
	err = container.Extract(imfPath, container.ExtractOptions{
		Passphrase: "wrong-passphrase",
		OutputDir:  badDir,
	})
	if err == nil {
		t.Fatal("expected error with wrong passphrase")
	}
	t.Log("✓ Wrong passphrase correctly rejected")
}

func TestExpiredContainer(t *testing.T) {
	tmpDir := t.TempDir()
	imfPath := filepath.Join(tmpDir, "expired.imf")

	// Create and add a file.
	container.Create(imfPath)
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)
	container.Add(imfPath, []string{testFile})

	// Seal with already-expired time.
	kp, _ := imfcrypto.GenerateKeyPair()
	pastTime := time.Now().Add(-1 * time.Hour)
	container.Seal(imfPath, container.SealOptions{
		PrivateKey:  kp.PrivateKey,
		EmbedPubKey: true,
		ExpiresAt:   &pastTime,
	})

	// Verify should fail.
	err := container.Verify(imfPath, container.VerifyOptions{})
	if err == nil {
		t.Fatal("expected expiry error on verify")
	}
	t.Logf("✓ Expired verify rejected: %v", err)

	// Verify with ignore-expiry should pass.
	err = container.Verify(imfPath, container.VerifyOptions{IgnoreExpiry: true})
	if err != nil {
		t.Fatalf("Verify with ignore-expiry: %v", err)
	}
	t.Log("✓ Verify with ignore-expiry passed")

	// Extract should fail.
	err = container.Extract(imfPath, container.ExtractOptions{OutputDir: filepath.Join(tmpDir, "out")})
	if err == nil {
		t.Fatal("expected expiry error on extract")
	}
	t.Logf("✓ Expired extract rejected: %v", err)

	// Extract with ignore-expiry should pass.
	err = container.Extract(imfPath, container.ExtractOptions{
		OutputDir:    filepath.Join(tmpDir, "out2"),
		IgnoreExpiry: true,
	})
	if err != nil {
		t.Fatalf("Extract with ignore-expiry: %v", err)
	}
	t.Log("✓ Extract with ignore-expiry passed")
}

func TestNoEncryption(t *testing.T) {
	tmpDir := t.TempDir()
	imfPath := filepath.Join(tmpDir, "noenc.imf")

	container.Create(imfPath)
	testFile := filepath.Join(tmpDir, "plain.txt")
	os.WriteFile(testFile, []byte("no encryption here"), 0644)
	container.Add(imfPath, []string{testFile})

	kp, _ := imfcrypto.GenerateKeyPair()
	err := container.Seal(imfPath, container.SealOptions{
		PrivateKey:  kp.PrivateKey,
		EmbedPubKey: true,
		// No passphrase = no encryption.
	})
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	// Verify.
	if err := container.Verify(imfPath, container.VerifyOptions{}); err != nil {
		t.Fatalf("Verify: %v", err)
	}

	// Extract without passphrase.
	extractDir := filepath.Join(tmpDir, "out")
	err = container.Extract(imfPath, container.ExtractOptions{OutputDir: extractDir})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(extractDir, "plain.txt"))
	if string(data) != "no encryption here" {
		t.Fatalf("content mismatch: %q", string(data))
	}
	t.Log("✓ No-encryption lifecycle passed")
}

func TestCreateDuplicateRejected(t *testing.T) {
	tmpDir := t.TempDir()
	imfPath := filepath.Join(tmpDir, "dup.imf")

	container.Create(imfPath)
	err := container.Create(imfPath)
	if err == nil {
		t.Fatal("expected error creating duplicate container")
	}
	t.Log("✓ Duplicate creation rejected")
}

func TestEmptySealRejected(t *testing.T) {
	tmpDir := t.TempDir()
	imfPath := filepath.Join(tmpDir, "empty.imf")

	container.Create(imfPath)
	kp, _ := imfcrypto.GenerateKeyPair()
	err := container.Seal(imfPath, container.SealOptions{PrivateKey: kp.PrivateKey})
	if err == nil {
		t.Fatal("expected error sealing empty container")
	}
	t.Log("✓ Empty seal rejected")
}

// TestTamperDetectionSingleBitFlip verifies that flipping a single bit anywhere
// in the sealed container causes verification to fail. This is the core guarantee
// of the IMF format: even 1 bit of tampering is detectable.
func TestTamperDetectionSingleBitFlip(t *testing.T) {
	tmpDir := t.TempDir()
	imfPath := filepath.Join(tmpDir, "tamper-test.imf")

	// Create and seal a container with a known file.
	container.Create(imfPath)
	testFile := filepath.Join(tmpDir, "secret.txt")
	os.WriteFile(testFile, []byte("This content must remain untouched."), 0644)
	container.Add(imfPath, []string{testFile})

	kp, _ := imfcrypto.GenerateKeyPair()
	container.Seal(imfPath, container.SealOptions{
		PrivateKey:  kp.PrivateKey,
		EmbedPubKey: true,
		Passphrase:  "tamper-test",
	})

	// Verify it passes before tampering.
	err := container.Verify(imfPath, container.VerifyOptions{})
	if err != nil {
		t.Fatalf("Pre-tamper verify failed: %v", err)
	}
	t.Log("✓ Container verifies before tampering")

	// Read the sealed container bytes.
	original, err := os.ReadFile(imfPath)
	if err != nil {
		t.Fatalf("Reading container: %v", err)
	}

	// Flip a single bit at multiple positions throughout the file.
	// Test near the beginning, middle, and end to cover different sections
	// (ZIP headers, file data, manifest, signature).
	positions := []int{
		50,                   // Near start (ZIP local file header area)
		len(original) / 4,   // Quarter way through
		len(original) / 2,   // Middle (likely in file data)
		len(original) * 3/4, // Three quarters (likely in manifest/signature area)
		len(original) - 50,  // Near end (ZIP central directory)
	}

	for _, pos := range positions {
		if pos < 0 || pos >= len(original) {
			continue
		}

		// Make a copy and flip one bit.
		tampered := make([]byte, len(original))
		copy(tampered, original)
		tampered[pos] ^= 0x01 // Flip the lowest bit

		// Write the tampered version.
		tamperedPath := filepath.Join(tmpDir, fmt.Sprintf("tampered-bit-%d.imf", pos))
		os.WriteFile(tamperedPath, tampered, 0644)

		// Verification must fail.
		err := container.Verify(tamperedPath, container.VerifyOptions{})
		if err == nil {
			t.Fatalf("SECURITY FAILURE: Verification passed after flipping bit at byte %d (file size: %d)", pos, len(original))
		}
		t.Logf("✓ Bit flip at byte %d/%d detected: %v", pos, len(original), err)
	}
}

// TestTamperDetectionTruncation verifies that truncating the container is detected.
func TestTamperDetectionTruncation(t *testing.T) {
	tmpDir := t.TempDir()
	imfPath := filepath.Join(tmpDir, "truncate-test.imf")

	// Create and seal a container.
	container.Create(imfPath)
	testFile := filepath.Join(tmpDir, "data.txt")
	os.WriteFile(testFile, []byte("Important data that must not be lost."), 0644)
	container.Add(imfPath, []string{testFile})

	kp, _ := imfcrypto.GenerateKeyPair()
	container.Seal(imfPath, container.SealOptions{
		PrivateKey:  kp.PrivateKey,
		EmbedPubKey: true,
	})

	// Read original and truncate by removing the last 100 bytes.
	original, _ := os.ReadFile(imfPath)
	truncated := original[:len(original)-100]

	truncPath := filepath.Join(tmpDir, "truncated.imf")
	os.WriteFile(truncPath, truncated, 0644)

	err := container.Verify(truncPath, container.VerifyOptions{})
	if err == nil {
		t.Fatal("SECURITY FAILURE: Verification passed on truncated container")
	}
	t.Logf("✓ Truncation detected: %v", err)
}

// TestTamperDetectionByteOverwrite verifies that overwriting a chunk of bytes is detected.
func TestTamperDetectionByteOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	imfPath := filepath.Join(tmpDir, "overwrite-test.imf")

	// Create and seal.
	container.Create(imfPath)
	testFile := filepath.Join(tmpDir, "report.txt")
	os.WriteFile(testFile, []byte("Quarterly results: revenue up 15%."), 0644)
	container.Add(imfPath, []string{testFile})

	kp, _ := imfcrypto.GenerateKeyPair()
	container.Seal(imfPath, container.SealOptions{
		PrivateKey:  kp.PrivateKey,
		EmbedPubKey: true,
		Passphrase:  "overwrite-test",
	})

	// Read and overwrite 16 bytes in the middle with zeros.
	original, _ := os.ReadFile(imfPath)
	tampered := make([]byte, len(original))
	copy(tampered, original)
	mid := len(tampered) / 2
	for i := mid; i < mid+16 && i < len(tampered); i++ {
		tampered[i] = 0x00
	}

	overwritePath := filepath.Join(tmpDir, "overwritten.imf")
	os.WriteFile(overwritePath, tampered, 0644)

	err := container.Verify(overwritePath, container.VerifyOptions{})
	if err == nil {
		t.Fatal("SECURITY FAILURE: Verification passed after overwriting 16 bytes")
	}
	t.Logf("✓ 16-byte overwrite detected: %v", err)
}
