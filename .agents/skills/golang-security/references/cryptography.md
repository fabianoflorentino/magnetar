# Cryptography Security Rules

Broken or misused cryptography destroys confidentiality and integrity. These rules are non-negotiable:

1. TLS must be 1.2 or newer.
2. Never use DES, RC4, MD5, or SHA-1 for security purposes.
3. SSH host keys must be verified - never `InsecureIgnoreHostKey`.
4. Passwords must be hashed with Argon2id (preferred) or bcrypt.
5. Security-critical randomness must come from `crypto/rand`.

## Algorithm selection guide

Using the wrong primitive is as dangerous as using a broken one:

| Use case | Recommended | Avoid | Why |
| --- | --- | --- | --- |
| Symmetric encryption | AES-256-GCM, ChaCha20-Poly1305 | DES, 3DES, AES-ECB, RC4 | ECB leaks structure; DES/RC4 are broken |
| Password hashing | Argon2id, bcrypt, scrypt | MD5, SHA-1, plain SHA-256 | fast hashes enable brute force; memory-hard resists GPUs |
| Message authentication | HMAC-SHA256, Poly1305 | HMAC-MD5, HMAC-SHA1 | known collisions weaken MD5/SHA-1 |
| Digital signatures | Ed25519, ECDSA P-256 | RSA-PKCS1v1.5 | signature padding oracle issues |
| Key exchange | X25519, ECDH P-256 | static RSA key transport | forward secrecy needs ephemeral keys |
| Random generation | `crypto/rand` | `math/rand` | `math/rand` output is predictable |
| TLS | 1.2+ (prefer 1.3) | 1.0, 1.1, SSL | BEAST, POODLE and friends |

### Key sizes

| Algorithm | Minimum | Recommended |
| --- | --- | --- |
| RSA | 2048 bits | 4096 bits |
| AES | 128 bits | 256 bits |
| ECDSA | P-256 | P-256 or Ed25519 |

## Envelope encryption for key rotation

Rotating a master key should not mean re-encrypting every record. Encrypt data with a random data encryption key (DEK), encrypt that small DEK with the key encryption key (KEK), and store both alongside the ciphertext:

```go
func EnvelopeEncrypt(kek, plaintext []byte) (encryptedDEK, ciphertext []byte, err error) {
    dek := make([]byte, 32)
    if _, err := rand.Read(dek); err != nil {
        return nil, nil, err
    }
    ciphertext, err = EncryptAESGCM(dek, plaintext)
    if err != nil {
        return nil, nil, err
    }
    encryptedDEK, err := EncryptAESGCM(kek, dek)
    return encryptedDEK, ciphertext, err
}

func EnvelopeDecrypt(kek, encryptedDEK, ciphertext []byte) ([]byte, error) {
    dek, err := DecryptAESGCM(kek, encryptedDEK)
    if err != nil {
        return nil, err
    }
    return DecryptAESGCM(dek, ciphertext)
}

func EncryptAESGCM(key, plaintext []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    aead, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    nonce := make([]byte, aead.NonceSize())
    if _, err := rand.Read(nonce); err != nil {
        return nil, err
    }
    return aead.Seal(nonce, nonce, plaintext, nil), nil
}
```

Rotating the KEK re-encrypts only the small DEKs, never the data.

## Common cryptographic mistakes

**AES-ECB reveals patterns - High.** Identical plaintext blocks produce identical ciphertext blocks, exposing structure in things like images, credit cards, and configs. Calling `block.Encrypt` directly, or any ECB use, leaks the shape of the data. GCM mode is randomized and authenticates:

```go
// Bad - every identical block encrypts identically
block, _ := aes.NewCipher(key)
block.Encrypt(dst, src)

// Good - authenticated encryption with a random nonce
block, _ := aes.NewCipher(key)
aead, _ := cipher.NewGCM(block)
nonce := make([]byte, aead.NonceSize())
_, _ = rand.Read(nonce)
ciphertext := aead.Seal(nonce, nonce, plaintext, nil)
```

**Reusing nonces - Critical.** A single nonce reuse under AES-GCM breaks both confidentiality and authentication for every message sharing it. Nonces must be fresh random draws per encryption, never constants, counters you reset, or values reused across keys.

**Non-constant-time secret comparison - Medium.** `==` short-circuits on the first differing byte and leaks length and position timing. Compare tokens and values with `crypto/subtle` (see the `network.md` reference) and let password libraries handle their own comparison.

## TLS configuration - High

Never set `InsecureSkipVerify`; you remove exactly the protection TLS exists to provide:

```go
transport := &http.Transport{
    TLSClientConfig: &tls.Config{
        InsecureSkipVerify: true, // never for production
    },
}
```

Pin a minimum version and prefer the standard curves:

```go
func secureConfig() *tls.Config {
    return &tls.Config{
        MinVersion:       tls.VersionTLS12,
        CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
    }
}
```

## DES, RC4, and SHA-1 - High

DES is brute-forced, RC4 is biased, and SHA-1 collisions are practical. Do not reach for any of them; AES-GCM, ChaCha20-Poly1305, and SHA-256/SHA-512 stand in without difficulty. AES without a mode like GCM is not AEAD - prefer `cipher.NewGCM` or `chacha20poly1305.New` over raw block encryption.

## SSH host key verification - High

Connecting without verifying the server's host key exposes the channel to man-in-the-middle. Pin the expected key with `ssh.FixedHostKey` instead of ignoring verification:

```go
clientConfig := &ssh.ClientConfig{
    User:            "deploy",
    Auth:            []ssh.AuthMethod{ssh.Password(pw)},
    HostKeyCallback: ssh.FixedHostKey(knownHostKey),
    Timeout:         10 * time.Second,
}
```

## Password hashing - High

Single-iteration fast hashes (MD5, SHA-1, plain SHA-256) fall to GPU brute force. Standard-library SHA-2 is for general hash tables and checksums, never password storage:

```go
// Argon2id - memory-hard, resists GPU attacks
key := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)

// Or bcrypt - simpler API, no manual salt management
hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

// Go 1.24+ stdlib PBKDF2 with sufficient iterations
key, err := pbkdf2.Key(sha512.New, password, salt, 600_000, 32)
```

Store the parameters and salt alongside the hash so verification can depend on the original settings. On Go 1.24+ prefer stdlib `crypto/hkdf`, `crypto/pbkdf2`, and `crypto/sha3`; use `golang.org/x/crypto/...` only when targeting older Go versions.

## Weak random number generators - High

`math/rand` is a deterministic PRNG - tokens, nonces, and session IDs from it are guessable. `crypto/rand` reads the operating system's CSPRNG:

```go
// Bad - predictable
b := make([]byte, 16)
mathRand.Read(b)

// Good - cryptographically secure
b := make([]byte, 16)
if _, err := cryptorand.Read(b); err != nil { ... }
```

## Weak RSA keys - Medium

Keys under 2048 bits are breakable with modern hardware. Generate at 2048 minimum, 4096 for long-lived keys:

```go
key, err := rsa.GenerateKey(rand.Reader, 2048) // 2048 minimum
```

## CWE references

- CWE-327 - broken or risky cryptographic algorithm
- CWE-331 - insufficient entropy
- CWE-326 - inadequate encryption strength
- CWE-295 - improper certificate validation
- CWE-330 - insufficiently random values
- CWE-916 - password hash with insufficient computational effort
