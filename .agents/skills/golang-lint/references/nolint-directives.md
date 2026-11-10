# Nolint Directives

A suppression is an accepted debt entry. Fix the root cause first, then suppress only genuine false positives or intentional patterns — and make the suppression say so.

## Syntax and placement

```go
//nolint:lintername // justification explaining why this suppression is needed
```

Place the directive on the flagged line, or on the line immediately above it.

## Rules

1. **Name the linter.** A bare `//nolint` suppresses every linter on that line and makes it impossible to track what is being suppressed.
2. **Give a reason.** Future readers — and your future self — need to know why the hit is acceptable.
3. **`nolintlint` enforces both.** Configured with `require-specific` and `require-explanation`, it flags bare directives and missing reasons.
4. **Fix before suppressing.** Suppression is proportionate only after confirming the hit is a false positive or a deliberately chosen pattern.

## Examples

```go
// Fire-and-forget logging; the error is not actionable.
//nolint:errcheck
_ = logger.Sync()

// The type assertion is safe: the preceding type switch guarantees the type.
//nolint:forcetypeassert // guaranteed by the type switch on line 42
v := x.(MyType)

// Orchestration spans many subsystems; splitting it would obscure the flow.
//nolint:gocyclo // coordination boundary across 8 subsystems
func orchestrate() error {
}

// Length grows with the case count by design.
//nolint:funlen
func TestParser(t *testing.T) {
}

// Parallel structure beats abstracting the shared logic here.
//nolint:dupl // two intentionally parallel branches for readability
```

## Multiple linters on one line

Comma-separate the names:

```go
//nolint:errcheck,gosec // test helper; failure is ignored on purpose
```

## When to suppress vs. fix

**Fix, almost always:** `errcheck` (check the error, even if only logging it), `govet` (usually real bugs), `staticcheck` (deprecated APIs, logic errors), resource-leak linters like `bodyclose` and `sqlclosecheck`.

**Suppress only with justification:** `funlen` (table-driven tests), `gocyclo` (orchestration where splitting obscures the flow), `dupl` (intentional parallel structure), `exhaustive` (a `default` case intentionally absorbs the rest), `goconst` (extracting the literal would reduce clarity, e.g. in test assertions).

**Never suppress without a very strong argument:** security linters (`gosec`, `bodyclose`, `sqlclosecheck`, `rowserrcheck`) — they catch real resource leaks and exploits — and `errcheck` on production paths, where unchecked errors cause silent failures.
