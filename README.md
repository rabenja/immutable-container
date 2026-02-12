# IMF — Immutable File Container

A cryptographically sealed archive format for tamper-proof file storage and distribution.

IMF combines ZIP-based storage with Ed25519 digital signatures, AES-256-GCM authenticated encryption, and a manifest-driven integrity model. Once sealed, a container's contents cannot be modified — any tampering is cryptographically detectable.

## Features

- **Immutability**: Sealed containers reject all modifications
- **Integrity**: SHA-256 per-file hashing with Ed25519 signature over the manifest
- **Encryption**: Optional AES-256-GCM encryption with PBKDF2-derived keys
- **Self-verifying**: Optionally embed the public key so recipients can verify without key exchange
- **Expiration**: Optional time-based access control with override for forensic use
- **Portable**: Single binary, zero external dependencies, cross-platform
- **Auditable**: Pure Go stdlib implementation in under 1,000 lines

## Quick Start

```bash
# Build
go build -o imf ./cmd/imf/

# Generate a signing key pair
./imf keygen -out ./keys

# Create a container and add files
./imf create evidence.imf
./imf add evidence.imf document.pdf photo.jpg

# Seal it (sign + encrypt + set expiry)
./imf seal evidence.imf \
  -key ./keys/imf_private.pem \
  -embed-pubkey \
  -passphrase "my-secret" \
  -expires "2027-01-01T00:00:00Z"

# Verify integrity
./imf verify evidence.imf

# Extract
./imf extract evidence.imf -out ./extracted -passphrase "my-secret"
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `imf keygen` | Generate Ed25519 key pair |
| `imf create` | Create a new empty .imf container |
| `imf add` | Add files to an open container |
| `imf seal` | Seal (sign, optionally encrypt) |
| `imf verify` | Verify signature and integrity |
| `imf extract` | Extract files with verification |
| `imf list` | List files in a container |
| `imf info` | Show container metadata |

## Architecture

```
pkg/crypto/      Ed25519, AES-256-GCM, PBKDF2, PEM encoding
pkg/manifest/    Manifest schema, state machine, serialization
pkg/container/   Core API: Create, Add, Seal, Extract, Verify
cmd/imf/         CLI binary
```

## Container Format

An `.imf` file is a ZIP archive containing:

```
container.imf
├── manifest.json          # Metadata, file hashes, signature
├── files/                 # Stored files (.enc if encrypted)
│   ├── document.pdf.enc
│   └── photo.jpg.enc
├── keyring/
│   └── public.key         # Optional embedded public key
└── .sealed                # Seal marker
```

## Cryptographic Design

| Component | Algorithm | Purpose |
|-----------|-----------|---------|
| Signing | Ed25519 | Manifest authenticity |
| Hashing | SHA-256 | Per-file integrity |
| Encryption | AES-256-GCM | File confidentiality |
| KDF | PBKDF2-HMAC-SHA256 (600k iterations) | Passphrase → key |

## Use Cases

- Legal evidence packaging with chain-of-custody guarantees
- Regulatory compliance and audit-proof records
- Software supply chain integrity
- Secure document distribution with expiration
- Academic data preservation
- Whistleblower document authentication

## Testing

```bash
go test -v ./pkg/...
```

## License

Apache License 2.0 — see [LICENSE](LICENSE) for details.

This project includes a [technical whitepaper](docs/IMF_Whitepaper_v1.pdf) establishing prior art for the IMF container format.

## Author

Benjamin Toso — benjamin.toso@gmail.com
