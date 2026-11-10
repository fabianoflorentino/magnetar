# Cookie Security Rules

Cookies are bearer credentials sitting in the client's browser and on the wire. Their flags decide how defensible the session is. Rules:

1. Session and authentication cookies must set `HttpOnly`.
2. `Secure` must be set in production so cookies never cross plain HTTP.
3. `SameSite` should be `Lax` or `Strict`; `None` only when cross-site delivery is genuinely required.

## Missing HttpOnly - Medium

Without `HttpOnly`, any XSS can read the session cookie straight out of `document.cookie`:

```go
// Bad - readable by JavaScript
cookie := &http.Cookie{Name: "session", Value: sessionID}

// Good
cookie := &http.Cookie{
    Name:     "session",
    Value:    sessionID,
    HttpOnly: true,
    Secure:   true,
    SameSite: http.SameSiteStrictMode,
    Path:     "/",
    MaxAge:   3600,
}
```

## Missing Secure - Medium

A cookie without `Secure` is sent in the clear whenever the user lands on an HTTP page, letting a network observer replay it:

```go
http.SetCookie(w, &http.Cookie{
    Name:     "auth_token",
    Value:    token,
    Secure:   true,   // HTTPS only
    HttpOnly: true,
    SameSite: http.SameSiteLaxMode,
    Path:     "/",
    MaxAge:   86400,
})
```

Leave `Domain` empty so the cookie binds to the exact host that issued it, rather than bleeding across subdomains.

## SameSite - Medium

`SameSite` blocks cross-site requests from carrying the cookie, which is most of CSRF prevention by itself:

```go
// Strict - high-security flows; the cookie is withheld even on same-site navigation
authCookie := &http.Cookie{
    Name: "auth", Value: token,
    Secure: true, HttpOnly: true,
    SameSite: http.SameSiteStrictMode,
}

// Lax - the usual default: sent on top-level navigations, not on cross-site requests
sessionCookie := &http.Cookie{
    Name: "session", Value: token,
    Secure: true, HttpOnly: true,
    SameSite: http.SameSiteLaxMode,
}

// None - cross-site delivery; requires Secure and a deliberate justification
crossSiteCookie := &http.Cookie{
    Name: "analytics", Value: trackingID,
    Secure: true, HttpOnly: true,
    SameSite: http.SameSiteNoneMode,
}
```

`SameSite` does not replace CSRF tokens, but it closes the browser side of the attack.

## Cookie name prefixes

Prefixes make behavior enforced by the browser instead of by your code:

- `__Secure-` requires the `Secure` flag or the browser refuses the cookie.
- `__Host-` requires `Secure`, forbids `Domain`, and pins `Path=/` - the strongest constraints.

```go
hostCookie := &http.Cookie{
    Name:     "__Host-CSRF",
    Value:    csrfToken,
    Secure:   true,
    HttpOnly: true,
    Domain:   "",
    Path:     "/",
}
```

## Gorilla sessions - High

The sessions package still needs the same discipline: no hardcoded keys, separate auth and encryption keys, and hardened options:

```go
store := sessions.NewCookieStore(
    []byte(os.Getenv("SESSION_AUTH_KEY")), // signing key
    []byte(os.Getenv("SESSION_ENC_KEY")),  // separate encryption key
)
store.Options = &sessions.Options{
    Path:     "/",
    MaxAge:   86400 * 30,
    HttpOnly: true,
    Secure:   true,
    SameSite: http.SameSiteStrictMode,
}
```

## Checklist

- `HttpOnly` on every authentication-related cookie.
- `Secure` on every cookie in a TLS deployment.
- `SameSite` set to `Lax` or `Strict` explicitly.
- `MaxAge` bounded; sessions do not live forever.
- No `Domain` unless subdomain sharing is required and understood.
- Cookie values validated and cryptographically signed server-side.
- Secrets rotated on a schedule; logout clears the cookie server-side.
- Double-submit csrf token or `SameSite` plus token for state-changing endpoints.

## CWE references

- CWE-1004 - sensitive cookie without `HttpOnly` flag
- CWE-614 - sensitive cookie without `Secure` attribute
- CWE-352 - cross-site request forgery
- CWE-285 - improper authorization
- CWE-565 - reliance on cookies without validation
