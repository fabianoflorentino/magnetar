---
name: golang-naming
description: "Go (Golang) naming conventions — covers packages, constructors, structs, interfaces, constants, enums, errors, booleans, receivers, getters/setters, functional options, acronyms, test functions, and subtest names. Use this skill when writing new Go code, reviewing or refactoring, choosing between naming alternatives (New vs NewTypeName, isConnected vs connected, ErrNotFound vs NotFoundError, StatusReady vs StatusUnknown at iota 0), debating Go package names (utils/helpers anti-patterns), or asking about Go naming best practices. Also trigger when the user mentions MixedCaps vs snake_case, ALL_CAPS constants, Get-prefix on getters, or error string casing. Do NOT use for general Go implementation questions that don't involve naming decisions."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🏷"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
paths:
  - "**/*.go"
---

# Naming in Go

**Persona:** You are a Go engineer for whom names are the cheapest documentation. Every identifier either reads as a clear contract at its call site or costs readers a lookup; the first job is making the reader never need the second.

**Modes:**

- **Write** — name new identifiers (choosing between the conventions below, or justifying a deviation with a comment). Sequential.
- **Review** — scan a diff for stuttering, snake_case, misnamed options, and naming debt. Sequential.
- **Refactor** — rename existing identifiers safely using `gopls` Rename, then apply via the staged workflow. Sequential.

**When to use:** any decision between naming alternatives, any review looking for readable names. Renaming at scale → `golang-gopls` + `golang-refactoring`; enforcement → `golang-lint`.

## The two load-bearing rules

**MixedCaps everywhere.** Go's export mechanism depends on capitalization; underscores in identifiers fight both the toolchain and every reader. Exceptions: subtest names like `TestFoo_InvalidInput`, generated code, cgo.

```go
MaxPacketSize // good
MAX_PACKET_SIZE   // C-style: no
max_packet_size   // snake_case: no
kMaxBufferSize    // Hungarian: no
```

**Never stutter.** Call sites always repeat the package name, so a name that re-states it forces double parsing — `http.HTTPClient` reads "HTTP" twice. The rule extends to every exported type in a package, not just the primary struct:

```go
http.Client     // not http.HTTPClient
json.Decoder    // not json.JSONDecoder
user.New()      // not user.NewUser()

// package dbpool
type Pool struct{}      // not DBPool
type Status struct{}    // not PoolStatus — callers write dbpool.Status
type Option func(*Pool) // not PoolOption
```

## Quick reference

| Element | Convention | Example |
| --- | --- | --- |
| Package | lowercase, single word, no plurals | `json`, `tabwriter`, `http_test` |
| File | lowercase, underscores OK | `user_handler.go` |
| Exported | UpperCamelCase | `ReadAll`, `HTTPClient` |
| Unexported | lowerCamelCase | `parseToken`, `userCount` |
| Interface | method name + `-er` | `Reader`, `Stringer` |
| Struct | MixedCaps noun | `Request`, `FileHeader` |
| Constant | MixedCaps (never ALL_CAPS) | `MaxRetries`, `defaultTimeout` |
| Receiver | 1–2 letter abbreviation | `func (s *Server)` |
| Sentinel error | `Err` prefix | `ErrNotFound` |
| Error type | `Error` suffix | `PathError` |
| Constructor | `New()` (single primary) / `NewType()` (multiple) | `apiclient.New()`, `http.NewRequest` |
| Boolean field/method | `is`/`has`/`can` prefix | `isConnected`, `IsConnected()` |
| Test function | `Test` + target name | `TestParseToken` |
| Acronym | all caps or all lower | `URL`, `HTTPServer`, `xmlParser` |
| Context variant | `WithContext` suffix | `FetchWithContext`, `QueryContext` |
| In-place variant | `In` suffix | `SortIn()`, `ReverseIn()` |
| Panic variant | `Must` prefix | `MustParse()`, `MustLoadConfig()` |
| Option funcs | `With` + field | `WithPort()`, `WithLogger()` |
| iota enum | type prefix; zero = unknown | `StatusUnknown` at 0 |
| Error string | lowercase (incl. acronyms), no punctuation | `"invalid message id"` |
| Import alias | only on collision | `mrand "math/rand"` |
| Format funcs | `f` suffix | `Errorf`, `Wrapf`, `Logf` |

## The conventions people miss

- **Constructors:** single primary type → `New()`, not `NewType()`. Only reach for `NewTypeName` when a package builds several constructible types.
- **Boolean fields:** the `is`/`has`/`can` prefix is mandatory on unexported fields (`isConnected`, not `connected`) and kept on the exported getter (`IsConnected() bool`). A bare adjective reads like it could be a slice or duration; the prefix reads like a question.
- **Getters:** Go omits `Get` — `user.Name()` over `user.GetName()`. Keep `Is`/`Has`/`Can` for predicates.
- **Error strings are fully lowercase, acronyms included** — `"invalid message id"`, so concatenated context (`fmt.Errorf("parsing token: %w", err)`) stays grammatical. Sentinels prefix the package: `errors.New("apiclient: not found")`.
- **Enum zero values:** put an explicit `Unknown`/`Invalid` sentinel at iota 0. `var s Status` silently becomes 0; if that maps to `StatusReady`, code behaves as if the caller chose a state it never did.
- **Subtest names:** lowercase descriptive phrases — `"valid id"`, `"empty input"`.
- **Scope length pairs:** `i` for a 3-line loop is correct; `userIndex` there is noise. Long names belong to long/package scope.

## Name length and stutter by category

The complete decision tables and rationale per category live in the references:

- **[Packages, Files & Imports](./references/packages-files.md)** — single-word lowercase packages, file naming, alias-only-on-collision.
- **[Identifiers](./references/identifiers.md)** — scope-based length, receivers, acronym casing, boolean naming.
- **[Functions, Methods & Options](./references/functions-methods.md)** — getters, constructors, named returns, `f` suffixes, functional options.
- **[Types, Constants & Errors](./references/types-errors.md)** — `-er` interfaces, enum sentinels, sentinel `Err`/type `Error`, error message casing.
- **[Test Naming](./references/testing.md)** — test function naming, table fields (`input`, `expected`), subcase names.

## Mistakes at a glance

| Mistake | Fix |
| --- | --- |
| `ALL_CAPS` constants | `MixedCaps` — casing means visibility, not emphasis |
| `GetName()` getter | `user.Name()` |
| `Url`, `Http`, `Json` | all caps or all lower — `HttpsUrl` is unreadable |
| `this`/`self` receiver | 1–2 letters: `s` for `Server` |
| `utils`/`helpers` packages | Name the abstraction, not the folder role |
| `http.HTTPClient` | `http.Client` |
| `user.NewUser()` | `user.New()` |
| `connected bool` | `isConnected` |
| `"invalid message ID"` | `"invalid message id"` |
| `StatusReady` at iota 0 | `StatusUnknown` at 0 |
| bare `"not found"` sentinel | `"pkg: not found"` to name the origin |
| `userSlice` type-in-name | `users` — encode what it holds, not how |
| switching receiver names | one consistent name per type |
| Naming constants by value | `DefaultPort` survives change; `Port8080` does not |
| `FetchCtx()` | `FetchWithContext()` |
| no `In` for in-place mutation | `SortIn()` signals mutation |
| `parse()` that panics | `MustParse()` puts the surprise in the name |
| mixing `With`/`Set`/`Use` | pick `With` for options, stay consistent |
| plural package names | singular — `net/url` |
| `Wrapf` without `f` | `f` suffix signals format strings |
| `user`/`account`/`person` mix | one concept, one name |

## Applying a rename safely

Renames reach every call site and can break interface satisfaction invisibly. Do them with `gopls` Rename (workspace-wide, interface-satisfaction aware) rather than grep/sed — see `golang-gopls` and `golang-refactoring` for the blast-radius mapping and staged PR workflow.

## Enforce with linters

`revive`, `predeclared`, `misspell`, `errname` catch the mechanical ones — wire them through `golang-lint`.

## Cross-references

- `golang-code-style` — broader formatting and style beyond identifiers.
- `golang-structs-interfaces` — interface design depth and receiver conventions.
- `golang-gopls` — safe workspace-wide rename.
- `golang-refactoring` — applying a rename at scale.
- `golang-lint` — automated enforcement.
