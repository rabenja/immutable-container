package crypto_test

import (
	"bytes"
	"testing"

	imfcrypto "github.com/immutable-container/imf/pkg/crypto"
)

func TestKeyGenAndSigning(t *testing.T) {
	kp, err := imfcrypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	msg := []byte("immutable file container test message")
	sig := imfcrypto.Sign(kp.PrivateKey, msg)

	if !imfcrypto.Verify(kp.PublicKey, msg, sig) {
		t.Fatal("valid signature rejected")
	}

	// Tamper with message.
	msg[0] ^= 0xFF
	if imfcrypto.Verify(kp.PublicKey, msg, sig) {
		t.Fatal("tampered message accepted")
	}
	t.Log("✓ Signing and verification work correctly")
}

func TestPEMRoundTrip(t *testing.T) {
	kp, _ := imfcrypto.GenerateKeyPair()

	privPEM := imfcrypto.MarshalPrivateKeyPEM(kp.PrivateKey)
	pubPEM := imfcrypto.MarshalPublicKeyPEM(kp.PublicKey)

	privKey, err := imfcrypto.ParsePrivateKeyPEM(privPEM)
	if err != nil {
		t.Fatalf("ParsePrivateKeyPEM: %v", err)
	}
	pubKey, err := imfcrypto.ParsePublicKeyPEM(pubPEM)
	if err != nil {
		t.Fatalf("ParsePublicKeyPEM: %v", err)
	}

	if !bytes.Equal(privKey, kp.PrivateKey) {
		t.Fatal("private key roundtrip mismatch")
	}
	if !bytes.Equal(pubKey, kp.PublicKey) {
		t.Fatal("public key roundtrip mismatch")
	}
	t.Log("✓ PEM roundtrip works")
}

func TestEncryptDecrypt(t *testing.T) {
	salt, _ := imfcrypto.GenerateSalt()
	key, _ := imfcrypto.DeriveKey("test-passphrase", salt)

	plaintext := []byte("secret immutable data that must be protected")
	ciphertext, err := imfcrypto.Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if bytes.Equal(plaintext, ciphertext) {
		t.Fatal("ciphertext equals plaintext")
	}

	decrypted, err := imfcrypto.Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("decrypted doesn't match plaintext")
	}
	t.Log("✓ Encrypt/decrypt roundtrip works")

	// Wrong key should fail.
	wrongKey, _ := imfcrypto.DeriveKey("wrong-passphrase", salt)
	_, err = imfcrypto.Decrypt(wrongKey, ciphertext)
	if err == nil {
		t.Fatal("decryption with wrong key should fail")
	}
	t.Log("✓ Wrong key correctly rejected")
}

func TestDeterministicKDF(t *testing.T) {
	salt, _ := imfcrypto.GenerateSalt()
	k1, _ := imfcrypto.DeriveKey("same-passphrase", salt)
	k2, _ := imfcrypto.DeriveKey("same-passphrase", salt)

	if !bytes.Equal(k1, k2) {
		t.Fatal("same passphrase + salt should produce same key")
	}

	k3, _ := imfcrypto.DeriveKey("different-passphrase", salt)
	if bytes.Equal(k1, k3) {
		t.Fatal("different passphrase should produce different key")
	}
	t.Log("✓ KDF is deterministic and passphrase-sensitive")
}
