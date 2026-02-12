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

// Package crypto provides cryptographic primitives for immutable containers.
// Uses Ed25519 for signing, AES-256-GCM for encryption, and scrypt for KDF.
// All implementations use Go stdlib only — no external dependencies.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/pem"
	"errors"
	"fmt"
	"io"

)

const (
	// SaltSize is the size of the salt used for key derivation.
	SaltSize = 32
	// NonceSize is the size of the AES-GCM nonce.
	NonceSize = 12
	// KeySize is the AES-256 key size.
	KeySize = 32

	// PBKDF2 iterations — high count for passphrase-based derivation.
	PBKDF2Iterations = 600000
)

// KeyPair holds an Ed25519 key pair.
type KeyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// GenerateKeyPair creates a new Ed25519 key pair.
func GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating keypair: %w", err)
	}
	return &KeyPair{PublicKey: pub, PrivateKey: priv}, nil
}

// MarshalPrivateKeyPEM encodes the private key as PEM.
func MarshalPrivateKeyPEM(key ed25519.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "IMF ED25519 PRIVATE KEY",
		Bytes: key,
	})
}

// MarshalPublicKeyPEM encodes the public key as PEM.
func MarshalPublicKeyPEM(key ed25519.PublicKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "IMF ED25519 PUBLIC KEY",
		Bytes: key,
	})
}

// ParsePrivateKeyPEM decodes a PEM-encoded private key.
func ParsePrivateKeyPEM(data []byte) (ed25519.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	if block.Type != "IMF ED25519 PRIVATE KEY" {
		return nil, fmt.Errorf("unexpected PEM type: %s", block.Type)
	}
	if len(block.Bytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: %d", len(block.Bytes))
	}
	return ed25519.PrivateKey(block.Bytes), nil
}

// ParsePublicKeyPEM decodes a PEM-encoded public key.
func ParsePublicKeyPEM(data []byte) (ed25519.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	if block.Type != "IMF ED25519 PUBLIC KEY" {
		return nil, fmt.Errorf("unexpected PEM type: %s", block.Type)
	}
	if len(block.Bytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: %d", len(block.Bytes))
	}
	return ed25519.PublicKey(block.Bytes), nil
}

// Sign signs data with the given private key.
func Sign(privateKey ed25519.PrivateKey, data []byte) []byte {
	return ed25519.Sign(privateKey, data)
}

// Verify checks the signature against data and public key.
func Verify(publicKey ed25519.PublicKey, data, signature []byte) bool {
	return ed25519.Verify(publicKey, data, signature)
}

// HashSHA256 returns the SHA-256 hash of data.
func HashSHA256(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// HashReaderSHA256 returns the SHA-256 hash of data read from a reader.
func HashReaderSHA256(r io.Reader) ([32]byte, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return [32]byte{}, err
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out, nil
}

// GenerateSalt creates a cryptographically random salt.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generating salt: %w", err)
	}
	return salt, nil
}

// DeriveKey derives an AES-256 key from a passphrase and salt using PBKDF2-HMAC-SHA256.
// Uses 600,000 iterations per OWASP 2023 recommendations.
func DeriveKey(passphrase string, salt []byte) ([]byte, error) {
	return pbkdf2([]byte(passphrase), salt, PBKDF2Iterations, KeySize), nil
}

// pbkdf2 implements PBKDF2-HMAC-SHA256 using only Go stdlib.
func pbkdf2(password, salt []byte, iterations, keyLen int) []byte {
	numBlocks := (keyLen + sha256.Size - 1) / sha256.Size
	dk := make([]byte, 0, numBlocks*sha256.Size)

	for block := 1; block <= numBlocks; block++ {
		dk = append(dk, pbkdf2Block(password, salt, iterations, block)...)
	}
	return dk[:keyLen]
}

func pbkdf2Block(password, salt []byte, iterations, blockNum int) []byte {
	mac := hmac.New(sha256.New, password)

	// U1 = PRF(password, salt || INT_32_BE(blockNum))
	mac.Write(salt)
	mac.Write([]byte{byte(blockNum >> 24), byte(blockNum >> 16), byte(blockNum >> 8), byte(blockNum)})
	u := mac.Sum(nil)

	result := make([]byte, len(u))
	copy(result, u)

	// U2..Uc
	for i := 1; i < iterations; i++ {
		mac.Reset()
		mac.Write(u)
		u = mac.Sum(u[:0])
		for j := range result {
			result[j] ^= u[j]
		}
	}
	return result
}

// Encrypt encrypts plaintext using AES-256-GCM with the given key.
// Returns nonce || ciphertext.
func Encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data encrypted by Encrypt (nonce || ciphertext).
func Decrypt(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}

	return plaintext, nil
}
