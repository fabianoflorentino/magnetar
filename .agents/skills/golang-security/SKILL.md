---
name: golang-security
description: "Security best practices and vulnerability prevention for Golang — injection (SQL, command, XSS), cryptography, path traversal, SSRF and HTTP security headers, cookies, secrets management, memory safety, PII in logs, STRIDE/DREAD threat modeling, plus `gosec` SAST, race detection, and fuzz testing. Apply when writing, reviewing, or auditing Go code for security, or when touching crypto, file or network I/O, secrets, user input, or authentication. Not for non-exploitable defensive bugs such as nil panics or slice aliasing (→ See `fabianoflorentino/golang-agent-skills@golang-safety` skill), dependency vulnerability scanning with govulncheck (→ See `fabianoflorentino/golang-agent-skills@golang-dependency-management` skill), or wiring security scanners into CI pipelines (→ See `fabianoflorentino/golang-agent-skills@golang-continuous-integration` skill)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🔒"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
        - govulncheck
    install:
      - kind: go
        package: golang.org/x/vuln/cmd/govulncheck@latest
        bins: [govulncheck]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch Bash(govulncheck:*) WebSearch AskUserQuestion EnterWorktree ExitWorktree
paths:
  - "**/*.go"
---

**Persona:** You are a senior Go security engineer. You apply security thinking when auditing existing code and when writing new code — threats are cheaper to prevent than to fix.

**Thinking mode:** Reason as thoroughly as possible for audits and vulnerability analysis — security bugs hide in subtle interactions, and surface-level review misses them. On Claude Code, use `ultrathink` for extended reasoning.

**Orchestration mode:** Fan out the five vulnerability-domain sub-agents from Audit mode as a fan-out-then-synthesize workflow for a whole-codebase audit. Parallelism widens attack-surface coverage per pass; the synthesis step dedupes findings and ranks by severity. On Claude Code, use `ultracode` to opt in.

**Modes:**

- **Review** — PR security review. Start from the changed files, then trace call sites and data flows into adjacent code: a vulnerability can live outside the diff but be triggered by it. Sequential.
- **Audit** — full-codebase scan. Launch up to 5 parallel sub-agents, each owning one independent domain: (1) injection patterns, (2) cryptography and secrets, (3) web security and headers, (4) authentication and authorization, (5) concurrency safety and dependency vulnerabilities. Aggregate, score with DREAD, report by severity. Each fix lands in its own isolated worktree — one fix = one worktree = one focused, reviewable, independently revertible PR.
- **Coding** — writing new code or fixing a reported vulnerability. Follow the sequential guidance; optionally a background agent greps the freshly written code for common vulnerability patterns while the main agent keeps implementing.

**When to use:** writing, reviewing, or auditing Go code for security; touching crypto, file/network I/O, secrets, user input, or authentication. Internal-correctness bugs (`golang-safety`), CVE scanning (`golang-dependency-management`), and CI wiring (`golang-continuous-integration`) are separate owners.

## Threat thinking

Security in Go is defense in depth: protect at multiple layers, validate all inputs, use secure defaults, and lean on the stdlib's security-aware design. Before writing or reviewing, ask three questions:

1. **Where are the trust boundaries?** Where does untrusted data enter? (HTTP requests, uploads, env vars, DB rows written by other services)
2. **What does the attacker control?** Which inputs flow into sensitive operations? (SQL, shell commands, HTML output, file paths, crypto)
3. **What is the blast radius?** If this defense fails, what's the worst outcome? (data leak, RCE, privilege escalation, DoS)

## Severity via DREAD

| Level | DREAD | Meaning |
| --- | --- | --- |
| Critical | 8–10 | RCE, full data breach, credential theft — fix immediately |
| High | 6–7.9 | auth bypass, significant data exposure, broken crypto — current sprint |
| Medium | 4–5.9 | limited exposure, session issues, weakened defense — next sprint |
| Low | 1–3.9 | minor disclosure, best-practice deviation — opportunistically |

Alignment with [DREAD scoring](./references/threat-modeling.md).

## Research before reporting

Trace the full data flow before flagging anything — never assess a snippet in isolation:

1. **Data origin** — user input, hardcoded constant, or internal-only value?
2. **Upstream validation** — is there sanitization, type parsing, or allow-listing earlier in the chain?
3. **Trust boundary** — data that never crosses a boundary (e.g. mTLS service-to-service) has a different risk profile.
4. **Surrounding code, not just the diff** — middleware, interceptors, or wrappers may already add a layer.

**Severity adjustment, not dismissal.** Upstream protection doesn't eliminate a finding — each layer must defend itself — but it changes severity: a SQL concatenation only reachable through a strict input parser is medium, not critical. Always adjust severity and note which upstream defenses exist and what happens if they're removed or bypassed. When downgrading or skipping, add a short inline comment (`// security: SQL concat safe here — parseUserID() returns int`), so the call is documented and won't be re-flagged.

## STRIDE threat modeling

Apply STRIDE at every trust-boundary crossing and data flow: **S**poofing (authentication), **T**ampering (integrity), **R**epudiation (audit logs), **I**nformation Disclosure (encryption), **D**enial of Service (rate limiting), **E**levation of Privilege (authorization). Prioritize with DREAD — Critical (8+) demands immediate action. Full methodology, DFD trust boundaries, DREAD scoring, OWASP mapping: [Threat Modeling Guide](./references/threat-modeling.md).

## Vulnerability → defense quick table

| Severity | Vulnerability | Defense | Stdlib solution |
| --- | --- | --- | --- |
| Critical | SQL injection | parameterized queries | `database/sql` `?` placeholders |
| Critical | command injection | args separate, never shell concat | `exec.Command` with separate args |
| High | XSS | auto-escaping renders data as text | `html/template` |
| High | path traversal | scope file access to a root | `os.Root` (Go 1.24+); pre-1.24 `filepath.IsLocal`+`filepath.Rel`, never `Clean`+`HasPrefix` |
| High | crypto misuse | vetted algorithms, no custom crypto | `crypto/aes`, `crypto/rand` |
| High | broken equality on secrets | constant-time compare | `crypto/subtle.ConstantTimeCompare` |
| Medium | timing attacks | constant-time operations | `crypto/subtle` |
| Medium | HTTP downgrade | TLS + security headers | `net/http` + `TLSConfig` |
| Low | missing headers | HSTS, CSP, X-Frame-Options | headers middleware |
| Medium | brute force / exhaustion | rate limits | `golang.org/x/time/rate`, timeouts |
| High | races | protect shared state | `sync.Mutex`, channels, avoid sharing |

## Detailed categories

Full examples, code snippets, CWE mappings:

- [Cryptography](./references/cryptography.md) — algorithms, KDF, TLS config.
- [Injection](./references/injection.md) — SQL, command, template, XSS, SSRF.
- [Filesystem](./references/filesystem.md) — traversal, zip bombs, permissions, symlinks.
- [Network/Web](./references/network.md) — SSRF, open redirects, headers, timing, session fixation.
- [Cookies](./references/cookies.md) — Secure, HttpOnly, SameSite.
- [Third-party leaks](./references/third-party.md) — analytics, GDPR/CCPA.
- [Memory safety](./references/memory-safety.md) — overflow, aliasing, `unsafe`.
- [Secrets](./references/secrets.md) — hardcoded creds, env vars, secret managers.
- [Logging](./references/logging.md) — PII, log injection, sanitization.
- [Architecture](./references/architecture.md) — defense in depth, Zero Trust, auth patterns, rate limiting.
- [Review checklist](./references/checklist.md) — domain-organized review checklist.

## Tooling & verification

Security-relevant linters (`bodyclose`, `sqlclosecheck`, `nilerr`, `errcheck`, `govet`, `staticcheck`) are configured in the `golang-lint` skill. Beyond them:

```bash
# SAST
go get -tool github.com/securego/gosec/v2/cmd/gosec@latest
go tool gosec ./...

# Reachable-CVE scan — full usage in golang-dependency-management
go get -tool golang.org/x/vuln/cmd/govulncheck@latest
go tool govulncheck ./...

# Race detector
go test -race ./...

# Fuzz testing
go test -fuzz=Fuzz
```

For the known CVEs of a specific module without a tree-wide scan (vetting a dependency on pkg.go.dev), → See `golang-pkg-go-dev`.

## Common mistakes

| Severity | Mistake | Fix |
| --- | --- | --- |
| Critical | SQL string concatenation | parameterized queries |
| Critical | `exec.Command("bash", "-c", ...)` | pass args separately; shell parses metacharacters |
| Critical | hardcoded secrets | env vars / secret managers (else history, CI logs, backups keep them) |
| Critical | ignoring crypto errors | fail closed — `_, _ = encrypt(data)` proceeds unencrypted |
| High | `math/rand` for tokens | sequence is predictable — `crypto/rand` |
| High | trusting unsanitized input | validate at trust boundaries |
| High | secrets compared with `==` | `ConstantTimeCompare` — `==` leaks timing |
| High | detailed errors to clients | generic messages; log details server-side |
| High | ignoring `-race` | races corrupt data and can bypass authz checks |
| High | MD5/SHA-1 passwords | Argon2id or bcrypt — memory-hard, intentionally slow |
| High | AES without GCM | ECB/CBC unauthenticated → GCM |
| High | client-side authorization | JS checks bypassed by any HTTP client — enforce server-side |
| Medium | binding `0.0.0.0` | bind the specific interface |
| Medium | rolling your own crypto | `crypto/aes` GCM, `golang.org/x/crypto/argon2` |

## Anti-patterns

| Anti-pattern | Why it fails | Fix |
| --- | --- | --- |
| Security through obscurity | hidden URLs surface via fuzzing/logs/source | authn + authz on every endpoint |
| Trusting client headers | `X-Forwarded-For`, `X-Is-Admin` are forged | server-side identity |
| Shared secrets across envs | staging breach → production | per-env secrets |
| Returning stack traces | helps attackers map the system | generic messages, server-side logs |

Anti-patterns with Go examples: [Architecture](./references/architecture.md).

## Cross-references

- `golang-database` — SQL layer security alongside this skill.
- `golang-safety` — internal correctness bugs, not exploits.
- `golang-observability` — audit logging and PII-safe logs.
- `golang-continuous-integration` — wiring scanners + AI review into CI.
- `golang-lint` / `golang-pkg-go-dev` / `golang-dependency-management` / `golang-testing` — linting, per-package CVEs, tree-wide scanning, security tests.

## Additional resources

- [Go Security Best Practices](https://go.dev/doc/security/best-practices)
- [gosec](https://github.com/securego/gosec)
- [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)
- [OWASP Go Secure Coding Practices](https://owasp.org/www-project-go-secure-coding-practices-guide/)
