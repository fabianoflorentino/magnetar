---
name: golang-pitfalls-error-handling
description: "Golang error handling — panicking instead of returning errors, ignoring when to wrap with %w, comparing error types with errors.As, comparing error values with errors.Is, sentinel errors vs custom types, handling an error twice (log-and-return), not handling errors, and not handling defer errors. Distilled from mistakes #48-54 of 100 Go Mistakes and How to Avoid Them. Apply when writing or reviewing Golang error creation, wrapping, comparison, or handling."
user-invocable: false
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🚨"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
paths:
  - "**/*.go"
---

# Golang Pitfalls: Error Handling

Source material: mistakes #48-54 from *100 Go Mistakes and How to Avoid Them* (teivah/100-go-mistakes).

Apply these rules when handling errors in Go.

## 48. Panicking (#48)

- `panic` stops the normal flow. Use it sparingly, only for unrecoverable conditions:
  - Signaling a programmer error (e.g., `sql.Register` with a nil/already-registered driver).
  - Failing to create a mandatory dependency.
- In almost all other cases, return a proper error as the last return value instead of panicking.

## 49. Ignoring when to wrap an error (#49)

- Since Go 1.13, `fmt.Errorf("...: %w", err)` wraps an error, making the source error available to the caller.
- Wrap to **add context** (use `%w`) or to **mark an error as a specific type** (create a custom error type).
- Wrapping creates coupling — callers can unwrap/compare the source error. If that's unwanted, transform the error instead (`fmt.Errorf("...: %v", err)`).

## 50. Comparing an error type inaccurately (#50)

- With Go 1.13 wrapping (`%w` + `fmt.Errorf`), checking the error **type** must use **`errors.As`** — a direct type assertion fails on wrapped errors.

```go
var target *MyError
if errors.As(err, &target) { ... }
```

## 51. Comparing an error value inaccurately (#51)

- **Expected** errors → sentinel error values: `var ErrFoo = errors.New("foo")`.
- **Unexpected** errors → error types implementing the `error` interface.
- With wrapping, compare against sentinel values using **`errors.Is`**, never `==`, so wrapped errors still match.

```go
if errors.Is(err, ErrFoo) { ... }
```

## 52. Handling an error twice (#52)

- Handle an error **once**. Logging an error counts as handling it.
- Choose between logging or returning the error — don't do both for the same error.
- Error wrapping is the convenient way to propagate the source error while adding context for the caller.

## 53. Not handling an error (#53)

- Ignoring an error is ambiguous to readers — intentional or a miss? Make it explicit with the blank identifier.

```go
_ = notify()
```

## 54. Not handling `defer` errors (#54)

- Errors returned by functions called in `defer` are usually not ignorable — handle or propagate them.
- If you deliberately ignore a `defer` error, use `_ =` and optionally comment why:

```go
// At-most-once delivery; missing some is accepted.
_ = notify()
```

## Cross-References

- → See `fabianoflorentino/golang-agent-skills@golang-error-handling` for the full idiomatic error-handling best practices (creation, wrapping, logging, the single handling rule)
- → See `fabianoflorentino/golang-agent-skills@golang-safety` for panic/recover and nil error comparison traps
- → See `fabianoflorentino/golang-agent-skills@golang-samber-oops` for structured errors, stack traces, and APM integration
- → See `fabianoflorentino/golang-agent-skills@golang-observability` for slog structured logging at error sites
