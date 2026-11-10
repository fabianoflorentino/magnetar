---
name: golang-pitfalls-testing
description: "Golang testing and benchmarks — not categorizing tests, not enabling the race flag, not using execution modes (parallel/shuffle), not using table-driven tests, sleeping in unit tests, not dealing with the time API efficiently, not using httptest and iotest utilities, writing inaccurate benchmarks, and not exploring coverage, external test packages, and fuzzing. Distilled from mistakes #82-90 plus the community fuzzing mistake of 100 Go Mistakes and How to Avoid Them. Apply when writing or reviewing Golang tests, benchmarks, or fuzz targets."
user-invocable: false
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
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
paths:
  - "**/*.go"
---

# Golang Pitfalls: Testing & Benchmarks

Source material: mistakes #82-90 (plus community fuzzing) from *100 Go Mistakes and How to Avoid Them* (teivah/100-go-mistakes).

Apply these rules when writing Go tests and benchmarks.

## 82. Not categorizing tests (#82)

- Categorize with **build tags**, **environment variables**, or **short mode** to separate unit vs. integration vs. long-running tests.
- Example: gate slow tests behind `-short` (`if testing.Short() { t.Skip(...) }`) or separate files with `//go:build integration`.

## 83. Not enabling the race flag (#83)

- Run tests with the race detector: `go test -race ./...`.
- It instruments memory accesses at runtime (not static analysis) and detects data races in concurrent tests, with added memory/time overhead — enable it locally and in CI, not production.
- Reading a `WARNING: DATA RACE` report: it lists the concurrent goroutines, the offending read/write locations in code, and where the goroutine was created.
- You can exclude a specific test file from race detection with `//go:build !race` (temporarily).

## 84. Not using test execution modes (#84)

- `-parallel <n>` speeds up test suites, especially long-running ones.
- `-shuffle <seed>` randomizes test order to reveal order-dependent bugs (wrong assumptions about global state, etc.).

## 85. Not using table-driven tests (#85)

- Group similar test cases into a table to avoid duplication and ease future updates.

```go
tests := []struct{ name string; in string; want int }{
    {name: "empty", in: "", want: 0},
    {name: "one", in: "a", want: 1},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) { ... })
}
```

## 86. Sleeping in unit tests (#86)

- Avoid `time.Sleep` as a synchronization mechanism — it makes tests slow and flaky.
- Synchronize explicitly: use channels, `sync.WaitGroup`, `errgroup`, or polling with timeouts (retry) when synchronization isn't possible.

## 87. Not dealing with the time API efficiently (#87)

- Make time-controlling functions testable: inject the current time as a hidden dependency (struct field / parameter) or ask clients to provide a `time.Time`.
- This avoids flaky tests from wall-clock timing assumptions.

## 88. Not using testing utility packages (#88)

- **`net/http/httptest`**: utilities for HTTP tests — spin up a test server (`httptest.NewServer`) and/or test handlers/servers without real network.
- **`testing/iotest`**: wrap `io.Reader`/`io.Writer` to simulate errors/edge cases and verify your code tolerates them (e.g., `iotest.ErrReader`, `iotest.HalfReader`, `iotest.OneByteReader`).

## 89. Writing inaccurate benchmarks (#89)

- Reset or pause the timer around setup:
  - One-time expensive setup: `b.ResetTimer()` before the loop.
  - Per-iteration setup: `b.StopTimer()` / `b.StartTimer()` around it.
- Micro-benchmarks are noisy (machine activity, power/thermal scaling, cache alignment). Increase `-benchtime` or repeat with `-count=N` and analyze with **`benchstat`**. Don't trust single-run nanosecond differences.
- Results are machine-specific: a micro-benchmark on your machine may not reflect the production system.
- Guard against compiler optimizations — a pure function can be inlined/eliminated. Assign results to a local var, then a global at the end:

```go
var global uint64
func BenchmarkPopcnt(b *testing.B) {
    var v uint64
    for i := 0; i < b.N; i++ {
        v = popcnt(uint64(i))
    }
    global = v
}
```

- Guard against the observer effect: reusing the same data across iterations lets CPU caches skew results. Re-create the input inside the loop for CPU-bound functions.

## 90. Not exploring all the Go testing features (#90)

- **Coverage**: use `-coverprofile` to see which code needs more tests.
- **External test package**: put tests in `<pkg>_test` to enforce testing exposed behavior, not internals.
- **Utility functions**: fail through `*testing.T` helpers (`t.Fatalf`, custom helpers calling `t.Helper()`) to keep tests short instead of `if err != nil` everywhere.
- **Setup/teardown**: use setup/teardown functions to configure complex environments (e.g., integration tests).

## Community: Not using fuzzing

- Fuzzing automatically feeds random, unexpected, or malformed inputs to a function to uncover vulnerabilities, bugs, and crashes.
- Write `FuzzXxx` functions (`f.Add(seed)`, `f.Fuzz(func(t *testing.T, ...))`) and run with `go test -fuzz=<FuzzName>`.
- Ideal for parsers, format decoders, and functions that take complex inputs.

## Cross-References

- → See `fabianoflorentino/golang-agent-skills@golang-testing` for the full testing best practices (fakes, fixtures, goleak, snapshot testing, integration tests)
- → See `fabianoflorentino/golang-agent-skills@golang-benchmark` for benchmark measurement methodology, benchstat, and profiling
- → See `fabianoflorentino/golang-agent-skills@golang-continuous-integration` for running tests and coverage gates in CI
- → See `fabianoflorentino/golang-agent-skills@golang-concurrency` for race-free concurrent test design
