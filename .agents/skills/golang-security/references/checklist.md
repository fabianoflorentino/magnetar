# Security Review Checklist

Domain-organized checklist for PR review and full-audit passes. Severity labels rank what to insist on first.

## Input handling

- [ ] **High** all user input validated at trust boundaries - code inside trusts the boundary
- [ ] **High** validation uses allowlists, never blocklists
- [ ] **High** output escaped per context (HTML, SQL, shell)
- [ ] **Medium** length limits enforced - prevents buffer abuse and DoS

## Database

- [ ] **Critical** SQL uses parameterized placeholders
- [ ] **Critical** no direct query construction from user input
- [ ] **Critical** ORM or driver path is itself injection-safe

## Code execution

- [ ] **Critical** no shell-interpolated `exec.Command` - arguments passed separately
- [ ] **Critical** no eval, reflection, or code generation on untrusted input
- [ ] **Critical** no deserialization of untrusted data with gob/binary formats

## Cryptography

- [ ] **Critical** no hardcoded secrets, keys, or credentials
- [ ] **High** `crypto/rand` for security-critical randomness
- [ ] **High** vetted algorithms only (AES-GCM, Argon2id, bcrypt)
- [ ] **Medium** HMAC or AEAD for integrity of sensitive payloads

## Web security

- [ ] **High** TLS 1.2+ configured, no `InsecureSkipVerify`
- [ ] **High** XSS prevented via `html/template` auto-escaping
- [ ] **Medium** security headers set (HSTS, CSP, X-Frame-Options)
- [ ] **Medium** CSRF protection on state-changing requests
- [ ] **Medium** open redirects validated
- [ ] **Medium** debug endpoints (pprof) unreachable from public traffic

## Authentication and authorization

- [ ] **High** passwords hashed with Argon2id or bcrypt
- [ ] **High** session tokens generated with `crypto/rand`
- [ ] **High** authorization checked on every privileged action, not just login
- [ ] **High** JWTs validated for algorithm, claims, and expiry
- [ ] **High** sessions invalidated server-side on expiry or logout

## Error handling

- [ ] **Medium** generic error messages to clients; details logged only
- [ ] **Medium** no stack traces or SQL text reach the client
- [ ] **Medium** failure paths fail closed

## Dependencies

- [ ] **High** `govulncheck ./...` passes
- [ ] **High** dependencies updated on a cadence, not ad hoc
- [ ] **Medium** new third-party libraries reviewed for posture and license

## Denial of service

- [ ] **Medium** server timeouts set (`ReadTimeout`, `WriteTimeout`, `IdleTimeout`)
- [ ] **Medium** request bodies bounded with `http.MaxBytesReader`
- [ ] **Medium** rate limiting on authentication and expensive endpoints

## Concurrency

- [ ] **High** `go test -race ./...` passes
- [ ] **High** shared state synchronized; no data races on globals
- [ ] **High** auth decisions cannot race into a bypass
