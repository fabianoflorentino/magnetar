# Network and Web Security Rules

Transport and server-configuration bugs leak data, enable phishing, and let attackers strong-arm the process. Rules:

1. Redirects must resolve only to allowlisted domains.
2. HTTP servers must set `ReadTimeout`, `WriteTimeout`, and `IdleTimeout`.
3. pprof and other debug endpoints must never be publicly reachable.
4. XML parsing must reject DTDs and external entities (XXE).

## Open redirect - Medium

An unvalidated `?next=` style parameter turns your trusted domain into a phishing launcher. Validate scheme, host, and path before redirecting:

```go
target := r.URL.Query().Get("url")
u, err := url.Parse(target)
if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
    return errors.New("invalid redirect target")
}
if !allowedHosts[u.Host] {
    return errors.New("domain not allowed")
}
http.Redirect(w, r, u.String(), http.StatusFound)
```

Prefer relative paths (`/login`, `/next-step`) where possible - they cannot leave your origin.

## Binding to every interface - Medium

Addressing `0.0.0.0:8080` on a shared host exposes the service to every reachable interface, including the internal network. Bind the specific address the service is meant to serve:

```go
net.Listen("tcp", "127.0.0.1:8080") // loopback only
net.Listen("tcp", "10.0.1.5:8080")   // one explicit internal IP
```

## Slowloris and resource exhaustion - Medium

Sockets held open with trickled bytes and oversized headers exhaust the connection pool. An explicit `http.Server` with timeouts and a header cap is the baseline defense:

```go
server := &http.Server{
    Addr:           ":8080",
    ReadTimeout:    5 * time.Second,
    WriteTimeout:   10 * time.Second,
    IdleTimeout:    120 * time.Second,
    MaxHeaderBytes: 1 << 20, // 1 MB
}
```

Use `http.ListenAndServe` only where you accept the defaults - deliberately. `MaxHeaderBytes` and the read/idle timeouts are what stop header-flood and slow-loris stalls.

## Observable timing - Medium

`==` over secrets short-circuits on the first difference, leaking length and position through timing. Compare fixed-length values constant-time, and let password libraries handle their own verification:

```go
tokenOK := subtle.ConstantTimeCompare([]byte(input), []byte(expected)) == 1

macOK := hmac.Equal(receivedMAC, expectedMAC) // constant-time by design
```

For passwords stay with Argon2id or bcrypt (`bcrypt.CompareHashAndPassword`), which hash and compare in one constant-time operation.

## Exposed pprof - High

The `net/http/pprof` import registers `/debug/pprof/*` on any handler it rides into - heap dumps, goroutine stacks, and CPU profiles that map your entire service. Gate the import behind a build tag and serve it only on a loopback listener:

```go
// debug_pprof.go
//go:build debug

package main

import _ "net/http/pprof"

func startDebugServer() {
    debugMux := http.NewServeMux()
    debugMux.HandleFunc("/debug/pprof/", pprof.Index)
    debugMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
    debugMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
    debugMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
    debugMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
    go http.ListenAndServe("127.0.0.1:6060", debugMux)
}
```

Require authentication even for internal-only debug listeners.

## XXE - High

Go's `encoding/xml` does not fetch external entities by default, but a document may still declare internal entities or DTDs that expand into parser resources. Reject documents that reference a DTD before decoding:

```go
if strings.Contains(string(raw), "<!DOCTYPE") || strings.Contains(string(raw), "<!ENTITY") {
    return errors.New("DTD declarations not allowed")
}
decoder := xml.NewDecoder(bytes.NewReader(raw))
decoder.Strict = true

var person Person
if err := decoder.Decode(&person); err != nil { ... }
```

Prefer typed decode into a struct over `interface{}`; combine with `MaxBytesReader` so entity expansion cannot exhaust memory.

## Permissive regex validation - Low

A regex that accepts nearly anything is worse than no validation because it feels like a check. Match real, bounded shapes - a `.+@.+` "email" check passes `x@@..` and absurd injections:

```go
var emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
if !emailRe.MatchString(email) {
    return errors.New("invalid email")
}
```

Also reject control characters and injection metacharacters where the value lands in logs, HTML, or queries.

## CWE references

- CWE-601 - open redirect
- CWE-208 - observable timing discrepancy
- CWE-611 - improper restriction of XML external entity references
- CWE-770 - allocation of resources without limits
- CWE-20 - improper input validation
- CWE-200 - exposure of sensitive information
