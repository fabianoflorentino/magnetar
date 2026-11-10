# Code Coverage

Coverage records which lines ran, not whether their behavior was asserted. Treat the number as a gap finder — read the uncovered paths and decide if each matters — never as a quality target to chase.

## Commands

```bash
# Generate the coverage file
go test -coverprofile=coverage.out ./...

# HTML report (uncovered lines in red)
go tool cover -html=coverage.out

# Coverage broken down by function
go tool cover -func=coverage.out

# Overall percentage
go tool cover -func=coverage.out | grep total

# Record execution counts, not just a boolean
go test -covermode=count -coverprofile=coverage.out ./...

# Race-safe counters (required under -race)
go test -race -covermode=atomic -coverprofile=coverage.out ./...

# Attribute package A's coverage to tests living in package B
go test -coverpkg=./... ./...

# One package, printed inline
go test -cover ./internal/store
```

## Modes

| Mode | Records | Use when |
| --- | --- | --- |
| `set` | whether a statement executed (default) | normal runs |
| `count` | executions per statement | finding never-taken branches in hot paths |
| `atomic` | count, race-safe | combined with `-race` or `t.Parallel()` |

## Pitfalls

- **Per-package by default.** Without `-coverpkg`, a test in `api` that exercises `store` reports nothing for `store`, making well-tested packages look untested.
- **Integration tests are invisible** unless you pass the build tag: `go test -tags=integration -coverprofile=...`.
- **Generated code inflates the number.** Exclude it before setting any threshold, or the metric measures the generator, not your code.
- **A covered line is not an asserted line.** A test that calls a function and discards its result reports 100% coverage and verifies nothing.

See `fabianoflorentino/golang-agent-skills@golang-continuous-integration` for wiring coverage reporting into CI.
