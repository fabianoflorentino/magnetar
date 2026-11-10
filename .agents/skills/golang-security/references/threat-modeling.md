# Threat Modeling Guide

Threat modeling surfaces the ways an attacker can break a system before the code gets exploited. Run it at design time, then refresh it every time the attack surface changes: new endpoints, new data flows, new integrations.

## STRIDE method

Walk each element of the data flow diagram and ask which threat categories apply:

| Category | Property under attack | Default defense |
| --- | --- | --- |
| Spoofing | identity | authentication |
| Tampering | integrity | signatures, validation |
| Repudiation | accountability | audit logging |
| Information Disclosure | confidentiality | encryption |
| Denial of Service | availability | timeouts, rate limits |
| Elevation of Privilege | authorization | server-side authz |

## STRIDE per data-flow element

Different diagram elements are exposed to different subsets:

| DFD element | S | T | R | I | D | E |
| --- | --- | --- | --- | --- | --- | --- |
| External entity (user, API client) | x | | x | | | |
| Process (HTTP handler, gRPC service) | x | x | x | x | x | x |
| Data store (database, cache, file) | | x | x | x | x | |
| Data flow (HTTP, gRPC, message queue) | | x | | x | x | |

**Spoofing** — Can a caller claim an identity it does not hold? Put authentication on every route group, validate tokens (algorithm, issuer, expiry), and use mTLS between services.

```go
r.Use(authMiddleware) // every protected group
```

**Tampering** — Can a payload be altered in transit or at rest? Validate all external input and verify webhook signatures with HMAC.

```go
mac := hmac.New(sha256.New, key)
mac.Write(payload)
if !hmac.Equal(signature, mac.Sum(nil)) {
    return errors.New("tampered payload")
}
```

**Repudiation** — Can an actor deny performing an action? Log every security-relevant action as structured data: actor, action, IP, timestamp.

```go
logger.Info("action_performed",
    "user_id", userID,
    "action", "delete_account",
    "ip", r.RemoteAddr,
    "timestamp", time.Now().UTC(),
)
```

**Information Disclosure** — Can sensitive data leak? Keep client-facing errors generic, keep PII out of logs, require TLS, and keep debug endpoints off public traffic.

**Denial of Service** — Can the service be exhausted? Set server timeouts, cap headers and bodies, and rate-limit.

```go
server := &http.Server{
    ReadTimeout:    5 * time.Second,
    WriteTimeout:   10 * time.Second,
    MaxHeaderBytes: 1 << 20,
}
```

**Elevation of Privilege** — Can a session gain power it lacks? Authorize server-side on every request and re-check object references for IDOR.

```go
if !user.HasPermission("admin:write") {
    http.Error(w, "Forbidden", http.StatusForbidden)
    return
}
```

## DREAD scoring

Score each threat on five factors from 1 (low) to 10 (high), then average:

| Factor | Low (1-3) | Medium (4-6) | High (7-10) |
| --- | --- | --- | --- |
| Damage | minor disclosure | partial data breach | full compromise |
| Reproducibility | timing-dependent | some effort | trivial, automated |
| Exploitability | custom exploit needed | basic tooling | public exploit exists |
| Affected Users | individual | subset of users | all users |
| Discoverability | insider knowledge | found via scanning | public knowledge |

Score = (D + R + E + A + D) / 5. Bands: 8-10 critical, 6-7.9 high, 4-5.9 medium, 1-3.9 low. These bands align with the severity table in the `golang-security` skill.

Example - SQL injection in a login handler:

| Factor | Score | Why |
| --- | --- | --- |
| Damage | 9 | full database access, credential theft |
| Reproducibility | 9 | deterministic; sqlmap automates it |
| Exploitability | 8 | well documented, easy tooling |
| Affected Users | 10 | every account holder |
| Discoverability | 7 | automated scanners catch it |

DREAD 8.6 - critical, remediate immediately.

## Trust boundaries

Draw the diagram, then for every arrow crossing a boundary answer three questions:

1. **Authentication** - who is calling?
2. **Validation** - is the data well-formed and within bounds?
3. **Authorization** - may this caller act on this resource?

```
Internet -> [LB/WAF] -> [HTTP server]
                          |
                     [middleware: authN, rate limit, validation, headers]
                          |
                     [service layer] -> [cache]
                          |
                     [database: parameterized queries]
                          |
                     [external APIs via mTLS]
```

Treat everything outside the boundary as untrusted - including env vars, uploads, queue messages, and rows written by other services.

## OWASP Top 10 mapping

| Rank | Issue | STRIDE | Go defense |
| --- | --- | --- | --- |
| A01 | Broken Access Control | E | server-side authz, RBAC, IDOR checks |
| A02 | Cryptographic Failures | I | AES-GCM, `crypto/rand`, TLS 1.2+ |
| A03 | Injection | T, E | placeholders, `exec.Command` args, `html/template` |
| A04 | Insecure Design | all | STRIDE at design time, defense in depth |
| A05 | Security Misconfiguration | I, E | timeouts, TLS config, no exposed pprof |
| A06 | Vulnerable Components | all | `govulncheck`, update cadence, `go.sum` |
| A07 | Authentication Failures | S, E | Argon2id/bcrypt, pinned-alg JWTs |
| A08 | Software/Data Integrity | T | module checksums, signed releases |
| A09 | Logging Failures | R | structured `log/slog`, PII-free audit trails |
| A10 | SSRF | I, T | URL allowlists, block internal IPs |

## Running a model

1. **Scope** - boundaries, assets to protect, threat actors.
2. **Diagram** - data flow diagram with trust boundaries.
3. **STRIDE** - apply the matrix to every element.
4. **Score** - DREAD every threat.
5. **Prioritize** - fix critical/high first; document accepted risks with a reason.
6. **Verify** - run `gosec ./...`, `govulncheck ./...`, and `go test -race ./...`.
7. **Iterate** - re-run when endpoints, data flows, or integrations change.

## Severity fallback

Without DREAD data, cross-reference impact with exploitability:

| Impact \ Exploitability | Easy | Moderate | Difficult |
| --- | --- | --- | --- |
| Critical | Critical | Critical | High |
| High | Critical | High | Medium |
| Medium | High | Medium | Low |
| Low | Medium | Low | Low |
