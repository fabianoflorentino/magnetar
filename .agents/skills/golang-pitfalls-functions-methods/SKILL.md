---
name: golang-pitfalls-functions-methods
description: "Golang functions and methods — choosing value vs pointer receivers, named result parameters, unintended side effects of zero-value named results, returning a nil receiver inside an interface, accepting filenames instead of io.Reader, and how defer evaluates arguments and receivers. Distilled from mistakes #42-47 of 100 Go Mistakes and How to Avoid Them. Apply when designing Golang functions, methods, signatures, and deferred calls."
user-invocable: false
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🛠"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
paths:
  - "**/*.go"
---

# Golang Pitfalls: Functions & Methods

Source material: mistakes #42-47 from *100 Go Mistakes and How to Avoid Them* (teivah/100-go-mistakes).

Apply these rules when designing Go functions and methods.

## 42. Not knowing which receiver to use (#42)

Receiver **must** be a pointer:

- If the method mutates the receiver (including appending to a slice receiver: `func (s *slice) add(el int)`).
- If the receiver contains a field that cannot be copied (e.g., a `sync` type — see #74).

Receiver **should** be a pointer:

- If the receiver is a large object (avoids expensive copies; benchmark when unsure).

Receiver **must** be a value:

- If immutability must be enforced.
- If the receiver is a map, function, or channel (compilation error otherwise).

Receiver **should** be a value:

- Unmutated slices.
- Small arrays/structs that are naturally value types (`time.Time`).
- Basic types (`int`, `float64`, `string`).

Default: value receiver unless there's a good reason; when in doubt, use a pointer receiver.

## 43. Never using named result parameters (#43)

- Named results are initialized to their zero value and enable naked `return`s.
- Useful for readability, especially when multiple results share a type.
- Use sparingly, only when there's a clear benefit.

```go
func f(a int) (b int) {
    b = a
    return
}
```

## 44. Unintended side effects with named results (#44)

- Named results default to zero values; a premature `return err` may return `nil` if `err` was never assigned — silent bug.

```go
func (l loc) getCoordinates(ctx context.Context, address string) (lat, lng float32, err error) {
    if !l.validateAddress(address) {
        return 0, 0, errors.New("invalid address")
    }
    if ctx.Err() != nil {
        return 0, 0, err // BUG: err is still nil here
    }
    // ...
}
```

- Be cautious with named result parameters: verify every return path passes the intended value.

## 45. Returning a nil receiver (#45)

- When returning an interface, don't return a **nil pointer**; return an explicit `nil` value.
- A typed nil pointer inside an interface makes the interface non-nil, so callers checking `if err == nil` get a false "no error".

## 46. Using a filename as a function input (#46)

- Accepting a filename (string path) to read a file is a code smell (except in low-level funcs like `os.Open`).
- It complicates unit tests (must create files) and reduces reusability.
- Accept `io.Reader` instead — abstracts file, string, HTTP, or gRPC body; easy to test with `strings.NewReader`/`bytes.NewBuffer`.

## 47. Ignoring how `defer` arguments/receivers are evaluated (#47)

- `defer` evaluates its arguments (and method receiver) **immediately**, not when the surrounding function returns.
- Passing `status` to `defer notify(status)` captures the value at defer time, so later assignments are missed.
- Fixes that keep `defer`:
  - Pass a pointer: `defer notify(&status)`.
  - Wrap in a closure evaluated at return time:

```go
defer func() {
    notify(status)
    incrementCounter(status)
}()
```

- The closure reads variables when it runs, so it sees final values.

## Cross-References

- → See `fabianoflorentino/golang-agent-skills@golang-structs-interfaces` for pointer vs value receivers and interface design
- → See `fabianoflorentino/golang-agent-skills@golang-error-handling` for defer error handling
- → See `fabianoflorentino/golang-agent-skills@golang-naming` for method and parameter naming
- → See `fabianoflorentino/golang-agent-skills@golang-safety` for the nil interface trap (mistake #45 expands to a full section there)
