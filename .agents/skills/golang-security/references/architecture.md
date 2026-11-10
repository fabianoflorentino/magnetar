# Security Architecture Patterns

Defense in depth: every layer defends on its own, so one bypass does not hand over the system. Zero Trust operationalizes that with explicit verification at every call.

## Defense-in-depth layers

```
PERIMETER   rate limiting, WAF, DDoS mitigation
NETWORK     TLS/mTLS, segmentation
APPLICATION input validation, authN, authz, secure coding
DATA        encryption at rest and in transit, access controls, backups
```

Each layer carries its own Go control:

**Layer 1 - rate limiting.** A global limiter protects the process; per-client limiters stop one abuser from consuming the whole budget:

```go
func RateLimitMiddleware(rps float64, burst int) func(http.Handler) http.Handler {
    limiter := rate.NewLimiter(rate.Limit(rps), burst)
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !limiter.Allow() {
                http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

**Layer 2 - mTLS for service-to-service.** Mutual certificates prove both ends and keep internal traffic off unauthenticated channels:

```go
func mTLSConfig(caFile, certFile, keyFile string) (*tls.Config, error) {
    pool := x509.NewCertPool()
    caPEM, err := os.ReadFile(caFile)
    if err != nil {
        return nil, err
    }
    pool.AppendCertsFromPEM(caPEM)

    cert, err := tls.LoadX509KeyPair(certFile, keyFile)
    if err != nil {
        return nil, err
    }
    return &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      pool,
        MinVersion:   tls.VersionTLS12,
    }, nil
}
```

**Layer 3 - request body bounds.** Cap bodies before decoding so a "friendly" client cannot exhaust memory:

```go
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
            next.ServeHTTP(w, r)
        })
    }
}
```

**Layer 4 - encryption at rest.** Use authenticated AES-GCM, not raw ECB/CBC (see the `cryptography` reference for the `EncryptAESGCM`/`DecryptAESGCM` pair and envelope encryption).

## Zero Trust

| Principle | In practice |
| --- | --- |
| Verify explicitly | authenticate and authorize every request; no trust from network location |
| Least privilege | minimum permissions, short-lived tokens (e.g. 15-minute access, 7-day refresh) |
| Assume breach | segment services, encrypt all traffic, log access for anomaly detection |

```go
func ZeroTrustMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims, err := validateJWT(r.Header.Get("Authorization"))
        if err != nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        if !hasPermission(claims.Subject, r.Method, r.URL.Path) {
            http.Error(w, "Forbidden", http.StatusForbidden)
            return
        }
        next.ServeHTTP(w, r.WithContext(withClaims(r.Context(), claims)))
    })
}
```

## Authentication pattern selection

| Use case | Recommended | Go implementation |
| --- | --- | --- |
| Web application | OAuth 2.0 + PKCE with OIDC | `golang.org/x/oauth2` |
| API authentication | JWT with short expiry + refresh | `github.com/golang-jwt/jwt/v5` |
| Service-to-service | mTLS with rotation | `crypto/tls`, `tls.LoadX509KeyPair` |
| CLI / automation | API keys + IP allowlisting | middleware with `net.ParseIP` |
| High security | FIDO2 / WebAuthn | `github.com/go-webauthn/webauthn` |

### JWT validation

Pin the signing algorithm. Without the `RS256 only` check, an attacker can downgrade to `HS256` and sign with the public key:

```go
func validateJWT(authHeader string) (*jwt.RegisteredClaims, error) {
    tokenString := strings.TrimPrefix(authHeader, "Bearer ")
    token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{},
        func(token *jwt.Token) (any, error) {
            if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
            }
            return publicKey, nil
        },
        jwt.WithIssuer("your-issuer"),
        jwt.WithAudience("your-audience"),
        jwt.WithExpirationRequired(),
    )
    if err != nil {
        return nil, err
    }
    claims, ok := token.Claims.(*jwt.RegisteredClaims)
    if !ok {
        return nil, errors.New("invalid claims")
    }
    return claims, nil
}
```

### Password hashing with Argon2id

Memory-hard and GPU-resistant; parameters belong in the stored hash so verification follows the record's own settings. See the `cryptography` reference for bcrypt/scrypt/PBKDF2 comparison.

```go
var defaultHashConfig = PasswordConfig{
    Time: 3, Memory: 64 * 1024, Threads: 4, KeyLen: 32, SaltLen: 16,
}

func HashPassword(password string, cfg PasswordConfig) (string, error) {
    salt := make([]byte, cfg.SaltLen)
    if _, err := rand.Read(salt); err != nil {
        return "", err
    }
    key := argon2.IDKey([]byte(password), salt, cfg.Time, cfg.Memory, cfg.Threads, cfg.KeyLen)
    return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
        argon2.Version, cfg.Memory, cfg.Time, cfg.Threads,
        base64.RawStdEncoding.EncodeToString(salt),
        base64.RawStdEncoding.EncodeToString(key),
    ), nil
}
```

## HTTP security headers

Set on every response through middleware:

| Header | Purpose | Recommended value |
| --- | --- | --- |
| Content-Security-Policy | restricts resource sources, dampens XSS | `default-src 'self'; script-src 'self'` |
| X-Frame-Options | blocks clickjacking framing | `DENY` |
| X-Content-Type-Options | stops MIME-type sniffing | `nosniff` |
| Strict-Transport-Security | forces HTTPS, blocks downgrade | `max-age=31536000; includeSubDomains` |
| Referrer-Policy | controls referrer leakage | `strict-origin-when-cross-origin` |
| Permissions-Policy | disables browser features | `geolocation=(), microphone=(), camera=()` |

```go
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
        next.ServeHTTP(w, r)
    })
}
```

## Security anti-patterns

| Anti-pattern | Why it fails | Fix |
| --- | --- | --- |
| Security through obscurity | hidden URLs surface via fuzzing, logs, or source | authN + authz on every endpoint |
| Trusting client headers | `X-Forwarded-For`, `X-Is-Admin` are forged by any client | server-side identity; trust proxy headers only from known LBs |
| Client-side authorization | JS checks are bypassed by any HTTP client | server-side role check per handler |
| Shared secrets across envs | a staging breach becomes a production breach | per-environment secrets |
| Ignoring crypto errors | `_, _ = encrypt(data)` proceeds unencrypted | check every error return; fail closed |
| Rolling your own crypto | unanalyzed designs are reliably broken | `crypto/aes` GCM, `golang.org/x/crypto/argon2` |
| Verbose error responses | stack traces and DB errors map your internals | generic client errors, detailed server logs |

The identity you act on must come from your own session, never from a header the client controls:

```go
func goodHandler(w http.ResponseWriter, r *http.Request) {
    claims := claimsFrom(r.Context())
    if !hasRole(claims.Subject, "admin") {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
    adminPanel(w, r)
}
```
