package container_test

import (
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
