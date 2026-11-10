---
name: golang-testing
description: "Production-ready Golang tests — table-driven tests, testify suites and mocks, parallel tests, fuzzing, fixtures, goroutine leak detection with goleak, snapshot testing, code coverage, integration tests, idiomatic test naming. Use when writing or reviewing Go tests, choosing a testing approach, setting up Go test CI, or debugging flaky/slow tests. For testify-specific APIs see `fabianoflorentino/golang-agent-skills@golang-stretchr-testify`; for measurement methodology see `fabianoflorentino/golang-agent-skills@golang-benchmark`."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🧪"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
        - gotests
    install:
      - kind: go
        package: github.com/cweill/gotests/gotests@latest
        bins: [gotests]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent Bash(gotests:*) AskUserQuestion
paths:
  - "**/*.go"
---

# Testing in Go

**Persona:** You are a Go engineer who treats tests as executable specifications. A test is a statement about behavior that must stay true forever; if it can be satisfied by an implementation detail, it is not doing its job.

**Modes:**

- **Write** — create tests for new or existing code. Scan the target source file, scaffold table-driven cases with `gotests`, then enrich with edge cases, error paths, and the failure modes you can predict from reading the production code. Sequential.
- **Review** — review a PR's test changes. Read the diff for coverage of new behavior, assertion quality, drift from the source file order, and flakiness patterns. Sequential.
- **Audit** — survey a whole suite. Split the work by concern: unit test quality and coverage gaps, integration test isolation, goroutine leaks and race conditions. Parallel sub-agents allowed.
- **Debug** — a test fails or flakes. Reproduce reliably first, then isolate the failing assertion, then trace the root cause in product code or setup. Sequential.

**When to use:** any task touching `_test.go` files, test strategy, or test CI. For `assert`/`require`/suite/mock APIs reach for `golang-stretchr-testify`; for benchmark methodology use `golang-benchmark`; for CI wiring see `golang-continuous-integration`; for lint rules on tests see `golang-lint`.

## Principles

1. **Tests constrain behavior, not implementation.** Assert on public contracts and observable outcomes; a test coupled to internals becomes a rewrite on every refactor and proves nothing about the contract.
2. **Coverage locates gaps; it is not a target.** `go test -cover` tells you where behavior is unexercised — read the uncovered paths, decide if each matters, and treat the percentage as a clue.
3. **Every test runs alone.** If a test only passes because another ran first, the suite is order-dependent and will flake in CI. No shared mutation of globals, caches, or ports without explicit setup.
4. **Speed is a property, not a preference.** Unit tests should stay in the low milliseconds; anything that needs a network, database, or filesystem moves behind build tags and runs as integration.

## Files and naming

Mirror the source tree: `foo.go` gets `foo_test.go` (white-box, same package) or a `foo_test` black-box package suffix files. Name the test file after the source file, never after a single function — `go test`, coverage, IDE navigation, and `gotests` all resolve by file.

```text
helloworld.go   -> helloworld_test.go   // TestHelloWorld, TestServe, FuzzParse...
hello/
├── hello.go
└── hello_test.go   // black-box: package hello_test
```

Order test functions to match the order of the functions they target in the source file — a reader comparing the two files side by side finds the match by position.

Identifier prefixes: `TestXxx`, `TestType_Method`, `BenchmarkXxx`, `FuzzXxx`, `ExampleXxx`.

## Table-driven tests

One `[]struct{...}` of inputs and expectations, one named `t.Run` per case. Always give cases a `name` — it is the only thing that survives in failure output and in `-run 'Name/sub'` filtering.

```go
func TestPrice(t *testing.T) {
    cases := []struct{ name string; qty int; price, want float64 }{
        {"single", 1, 10, 10},
        {"bulk 100", 100, 10, 900},
        {"zero qty", 0, 10, 0},
    }
    for _, tt := range cases {
        t.Run(tt.name, func(t *testing.T) {
            if got := Price(tt.qty, tt.price); got != tt.want {
                t.Errorf("Price(%d, %.2f)=%.2f, want %.2f", tt.qty, tt.price, got, tt.want)
            }
        })
    }
}
```

**Scope leak to avoid:** do not build a testify `assert.New(t)` in the parent and reuse it inside `t.Run`. The parent `t` is baked into the assertor, so failures inside the subtest get attributed to the parent test — the subtest still reports `--- PASS`, silently hiding the broken case. Build the assertor per subtest with its own `t`.

```go
t.Run(tt.name, func(t *testing.T) {
    is := assert.New(t) // bind to the subtest's t
    is.Equal(tt.want, Price(tt.qty, tt.price))
})
```

Verify with a deliberately broken case: if `go test -v` shows `--- FAIL: TestPrice` but every `--- PASS: TestPrice/<case>` line stays PASS, the scope is leaking.

## Parallelism and races

- Call `t.Parallel()` on independent subtests — it amortizes CI wall time without losing determinism, as long as each test owns its data.
- Every CI invocation runs `go test -race ./...`. The race detector is the cheapest concurrency bug finder you will ever get.
- Never parallelize a test that sneaks into the same shared resource as another parallel test without serializing it (`sync.Mutex` around setup or a `t.Parallel()` sibling after fix).

## Determinism for concurrent code

- **goroutine leaks:** add `goleak.VerifyTestMain(m)` in `TestMain` for packages that spawn goroutines; scope a leak check to a test with `defer goleak.VerifyNone(t)`. See `golang-concurrency`.
- **fake time:** Go 1.25+ `testing/synctest` runs goroutines, timers and context deadlines under a synthetic clock that only advances when every goroutine blocks — reproducibility without real sleeps. Use it for time-sensitive concurrency instead of `time.Sleep` tuning.

## HTTP handlers

Exercise handlers with `httptest.NewRequest` + `httptest.NewRecorder` in table form: happy path, validation rejections, 403/404/422 statuses, authorization, idempotent replays. Assert status, `Content-Type`, and the response contract — not the handler's internals. For full end-to-end server behavior (timeouts, keep-alive), spin `httptest.NewServer`.

## Fuzzing

Seed a `FuzzXxx(f *testing.F)` with representative inputs and assert the invariant the function must always hold:

```go
func FuzzParse(f *testing.F) {
    f.Add("a=1&b=2")
    f.Fuzz(func(t *testing.T, in string) {
        got := Parse(in)
        if got.Keys() != len(got) { t.Fatalf("duplicate key survived: %q", in) }
    })
}
```

## Examples are tests

`ExampleXxx` in a `_test.go` file is run by `go test`: its stdout is compared to the `// Output:` comment, so a drifting example fails the build instead of misleading readers. Names with a `_suffix` match the last word of a value (`ExampleSession_ID`).

## Integration tests

Separate anything touching the real world with a build tag so `go test ./...` stays hermetic:

```go
//go:build integration

package store

func TestPGRoundTrip(t *testing.T) { ... }
```

Run with `go test -tags=integration ./...`. Keep integration fixtures (schemas, compose files) next to the tests in `testdata/`.

## Mocking

Define small interfaces at the consumer side and mock those — never mock concrete types or library packages you don't own. If a mock becomes heavy, the interface is probably wrong (see `golang-structs-interfaces`).

## Enforcing with linters

Have the linter hold the line: `thelper` (use `t.Helper()`), `paralleltest` (undraggers `t.Parallel()`), `testifylint` (assert-scope and comparison hygiene). See `golang-lint` for the configuration.

## Quick reference

```bash
go test ./...                              # everything
go test -race ./...                        # race detector
go test -run 'TestPrice/single' ./...      # one subtest
go test -tags=integration ./...            # integration suite
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
go test -bench=. -benchmem ./...           # see golang-benchmark
go test -fuzz=FuzzParse ./...              # fuzzing (see docs for -fuzztime)
```

## Cross-references

- `golang-stretchr-testify` — assert/require/mock/suite API.
- `golang-benchmark` — bench methodology, benchstat, profiling.
- `golang-concurrency` — race patterns, goleak, goroutine discipline.
- `golang-continuous-integration` — test CI, caching, `-race` gates.
- `golang-lint` — thelper/paralleltest/testifylint config.
- `golang-error-handling` — testing error paths with `errors.Is`.
- `golang-database` — database test fixtures behind build tags.
