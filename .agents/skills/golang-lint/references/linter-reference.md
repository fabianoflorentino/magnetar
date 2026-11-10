# Linter Reference

`golangci-lint` v2 reads a `.golangci.yml` declaring `version: "2"` at the project root. Five sections carry the config:

- **`run`** — concurrency, timeout, directory exclusions.
- **`linters.enable` / `linters.disable`** — which linters are active.
- **`linters.settings`** — per-linter thresholds and options.
- **`formatters`** — code formatters (`gofmt`, `gofumpt`, `goimports`).
- **`issues`** — output limits and exclusion rules.

To adopt a linter, add it to `linters.enable` and, if it needs tuning, configure it under `linters.settings`. To drop one, move it to `linters.disable` with a comment explaining why. The baseline shipped with this skill ([`../assets/.golangci.yml`](../assets/.golangci.yml)) is a production starting point; the catalog below shows what each linter adds so you can prove every choice by running it.

## The catalog, by concern

### Correctness & safety

| Linter | Catches |
| --- | --- |
| `govet` | Go's built-in checker: `copylocks`, printf format mismatches, struct tag validation, context stored in structs, unreachable code, nil dereferences |
| `staticcheck` | Deprecated APIs, common mistakes, unnecessary code, simplifications, stdlib misuse |
| `unused` | Unused variables, functions, types, and struct fields |
| `errcheck` | Unchecked error returns — enable `check-type-assertions: true` to cover type assertions too |
| `nilerr` | Returning nil while `err` is non-nil — a common silent-failure source |
| `forcetypeassert` | Type assertions without the comma-ok form |
| `copyloopvar` | Loop variable copy issues (Go 1.22+) |
| `errorlint` | Correct `errors.Is`/`errors.As` and `%w` wrapping (Go 1.13+) |
| `durationcheck` | `time.Duration * time.Duration` multiplication, which yields nanoseconds squared |
| `reassign` | Reassignment of package-level variables outside `init()` |

### Style & readability

| Linter | Catches |
| --- | --- |
| `gocritic` | Opinionated style: unnecessary conversions, range copies, append-assign patterns, redundant code |
| `revive` | Naming conventions: exported-type comments, unexported returns, receiver and error naming, stuttered package names |
| `wsl_v5` | Whitespace rules for visual grouping |
| `whitespace` | Trailing whitespace and needless blank lines in function bodies |
| `godot` | Exported-symbol comments ending with a period |
| `misspell` | Common English misspellings in identifiers and comments |
| `dupword` | Duplicate words in comments and strings ("the the") — usually copy-paste artifacts |
| `predeclared` | Shadowing of built-in identifiers such as `len`, `cap`, `error` |
| `errname` | Error naming: types suffixed `...Error`, variables prefixed `Err` |
| `asciicheck` | Non-ASCII identifiers — the homoglyph/trojan-source attack surface |

### Complexity

| Linter | Typical threshold |
| --- | --- |
| `gocyclo` | Cyclomatic complexity — commonly capped around 13; split functions that exceed it |
| `nestif` | Deeply nested `if`/`else` chains |
| `funlen` | Function length — commonly 120 lines or 80 statements |
| `dupl` | Code duplication — commonly a 100-token threshold |

### Performance

| Linter | Catches |
| --- | --- |
| `perfsprint` | Faster alternatives to `fmt.Sprintf` (e.g. `strconv.Itoa`) |
| `unconvert` | Unnecessary type conversions |
| `ineffassign` | Assignments never subsequently read |
| `goconst` | Repeated string/number literals worth extracting — commonly min 3 chars, min 4 occurrences |

### Security & resources

| Linter | Catches |
| --- | --- |
| `gosec` | SQL injection, hardcoded credentials, weak crypto, path traversal, unsafe usage — the primary SAST tool; never suppress without strong justification |
| `bidichk` | Bidirectional Unicode sequences (the CVE-2021-42574 trojan-source attack) |
| `noctx` | HTTP requests sent without a `context.Context` |
| `containedctx` | `context.Context` stored in struct fields instead of passed as a parameter |
| `fatcontext` | `context.WithValue`/`WithCancel` in loops, building unbounded context chains |
| `bodyclose` | HTTP response bodies never closed — leaked connections |
| `sqlclosecheck` | `sql.Rows` and `sql.Stmt` never closed |
| `rowserrcheck` | `sql.Rows.Err()` not checked after iteration |

### Logging

| Linter | Catches |
| --- | --- |
| `sloglint` | Consistent `log/slog` style: key-value pairing, message formatting, level usage |
| `loggercheck` | Key-value pair formatting for zap, slog, logr — odd argument counts, missing keys |

### Testing

| Linter | Catches |
| --- | --- |
| `thelper` | Test helpers that omit `t.Helper()`, misattributing failures |
| `paralleltest` | Tests and subtests missing `t.Parallel()` |
| `testifylint` | testify best practices (e.g. `assert.Equal` over `assert.True(a == b)`) |
| `usetesting` | `t.Setenv`/`t.TempDir` instead of `os.Setenv`/`os.MkdirTemp` — automatic cleanup and isolation |

### Modernization & meta

| Linter | Catches |
| --- | --- |
| `modernize` | Code rewriteable with newer Go features (requires golangci-lint v2.6.0+) |
| `exptostd` | `golang.org/x/exp/` functions with stdlib equivalents (slices, maps, cmp as of Go 1.21) |
| `intrange` | `range N` over C-style `for i := 0; i < N; i++` loops (Go 1.22+) |
| `usestdlibvars` | Hardcoded strings replaced by stdlib constants (e.g. `http.MethodGet` instead of `"GET"`) |
| `exhaustive` | `switch` statements over enum types missing possible values |
| `nolintlint` | Improper `//nolint` usage: names the linter and demands a reason comment |

### Formatting

Formatters run via `golangci-lint fmt ./...`:

| Formatter | Purpose |
| --- | --- |
| `gofmt` | Canonical Go formatting |
| `gofumpt` | Stricter formatting: consistent empty lines, grouped imports, simplified patterns (with `extra-rules: true`) |

The shipped baseline enables the `correctness`, core `style`, `gocritic`, `testing`, `security`, and `nolintlint` sets plus `forbidigo` as a workhorse start; everything else in the tables above is a deliberate opt-in. Disabled-by-default linters must be enabled in `.golangci.yml` with the reason documented in a comment beside each.
