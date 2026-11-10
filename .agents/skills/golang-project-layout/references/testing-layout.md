# Tests, Benchmarks, and Examples

Co-location is the rule: each `_test.go` file sits in the same directory as the code it exercises, and test data lives under `testdata/`.

## File naming

| Suffix | Purpose | Build behavior |
| --- | --- | --- |
| `_test.go` | Tests | Excluded from normal builds |
| `_bench_test.go` | Benchmarks | Excluded from normal builds |
| `_example_test.go` | Examples that verify output | Excluded from normal builds |
| No suffix | Regular code | Included in all builds |

## Where tests go

```
internal/
├── handler/
│   ├── handler.go              # production code
│   ├── handler_test.go         # tests
│   └── handler_bench_test.go   # benchmarks (optional)
├── service/
│   ├── service.go
│   └── service_test.go
└── model/
    ├── user.go
    └── user_test.go

pkg/
└── logger/
    ├── logger.go
    └── logger_test.go
```

## Test package choice

Two package declarations are available, and they buy different access:

**Same package — white-box:**

```go
package handler

import "testing"

func TestHandler(t *testing.T) {
    internalFunction() // can reach unexported functions and types
}
```

**`_test` suffix — black-box:**

```go
package handler_test

import "testing"

func TestHandler(t *testing.T) {
    handler.PublicMethod() // only the exported API
}
```

Use the same package for unit tests that need internals; use the `_test` suffix for integration and behavioral tests that exercise the public contract.

## Benchmarks

Benchmark functions live in `_bench_test.go` files and carry a `Benchmark` prefix.

## Examples

Examples do double duty as documentation and as runnable tests. In libraries, put them in `{package}_example_test.go`:

```go
package logger

import "fmt"

func ExampleLogger_Info() {
    log := New()
    log.Info("processing started")
    log.Info("processing complete")
    // Output:
    // INFO: processing started
    // INFO: processing complete
}
```

- Example functions start with `Example`.
- The `// Output:` comment states the expected output — `go test` fails if it does not match.
- `godoc` renders examples as documentation.
- For standalone demo programs, drop an executable under `examples/`:

```
examples/
└── basic-usage/
    └── main.go    # executable example
```

## Test utilities

Shared test helpers get their own package — either a top-level `test/testutils/` or the private `internal/testutil/` pattern:

```
internal/
└── testutil/
    ├── mock.go
    └── fixtures.go
```

## Test fixtures

Three patterns cover fixture files:

**Local `testdata/`** for package-specific data — Go ignores it when building regular packages:

```
internal/
└── handler/
    ├── handler.go
    ├── handler_test.go
    └── testdata/
        ├── users.json
        ├── request_valid.json
        └── request_invalid.json
```

**Global `test/fixtures/`** for data shared across packages:

```
test/
└── fixtures/
    ├── users.json
    ├── products.json
    └── responses/
        ├── success.json
        └── error.json
```

**Embedded fixtures** (Go 1.16+) via `//go:embed` when tests need the data at compile time — keep the files under `testdata/` and note that Go ignores any `.go` files placed there.

## Running tests

```bash
go test ./...                    # all tests
go test ./internal/handler       # one package
go test -v ./...                 # verbose
go test -race ./...              # race detection
go test -cover ./...             # coverage report
go test -short ./...             # skip long-running tests
```

## Summary

| File type | Suffix | Package | Purpose |
| --- | --- | --- | --- |
| Test | `*_test.go` | `package X` or `package X_test` | Unit/integration tests |
| Benchmark | `*_bench_test.go` | same as code | Performance tests |
| Example (godoc) | `*_example_test.go` | same as code | Documentation + verification |
| Executable example | none | `package main` | Standalone demo programs |
| Test utilities | `*_test.go` | `package testutil` | Shared test helpers |
