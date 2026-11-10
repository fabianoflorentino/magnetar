# Competing clusters — deep disambiguation

Thirteen areas where two or more `fabianoflorentino/golang-agent-skills` skills appear to overlap. Each cluster gives a boundary table, concrete routing examples, and the gap cases that the skill descriptions themselves do not yet spell out. The parent SKILL.md carries the compressed "who owns what" lines; this file is the reasoning behind them.

## 1. Performance cluster

`golang-observability` runs always-on; the other three activate on demand.

| Skill | Owns | Does not own |
| --- | --- | --- |
| `golang-performance` | the optimization patterns — "allocation bottleneck → sync.Pool", "hot path → drop reflection" | measurement, profile capture, root-cause analysis |
| `golang-benchmark` | pprof/trace capture, flamegraph interpretation, benchstat comparison, CI regression checks | which optimization to apply |
| `golang-troubleshooting` | the debugging workflow — reproduce, bisect, Delve, race detector, GODEBUG, test-driven debugging | profile interpretation, optimization patterns |
| `golang-observability` | permanent production signals — structured logs, counters, traces, alerting | one-off investigation, benchmark runs |

**Routing examples:**

- "Handler is slow in production" → `golang-observability` (metrics/traces) → `golang-benchmark` (capture) → `golang-performance` (fix).
- "Benchmark regressed 20%" → `golang-benchmark` (benchstat + pprof root cause).
- "Cut allocations in the hot path" → `golang-performance`.
- "Process crashes after 10 minutes under load" → `golang-troubleshooting` (race detector, leaks, Delve attach).

## 2. Dependency injection cluster

Decide with `golang-dependency-injection`, then go to the library skill for the chosen stack.

| Skill | Owns |
| --- | --- |
| `golang-dependency-injection` | why DI, manual constructor wiring, the library comparison |
| `golang-google-wire` | compile-time codegen: `wire.Build`, `wire.NewSet`, `wire.Bind`, provider sets |
| `golang-uber-dig` | runtime reflection: `dig.Provide`, `dig.In`/`dig.Out`, value groups, `Decorate` |
| `golang-uber-fx` | the application framework over dig: `fx.New`, `fx.Module`, lifecycle, `fx.Annotate` |
| `golang-samber-do` | the type-safe container: `do.Provide`, scopes, health checks, shutdown |

**Routing examples:**

- "Should I use DI at all?" → `golang-dependency-injection`.
- "`wire.Build` is failing" → `golang-google-wire`.
- "Lifecycle hooks with my container" → `golang-uber-fx` — dig sits a level below.
- "Type-safe DI without codegen" → `golang-samber-do` or `golang-uber-dig`.

## 3. samber/* functional cluster

Three programming models that rarely compete in one task.

| Skill | Owns | Does not own |
| --- | --- | --- |
| `golang-samber-lo` | finite collections: `lo.Map`, `lo.Filter`, `lo.Reduce`, `lo.Uniq`, `lo.GroupBy` | infinite streams, event pipelines |
| `golang-samber-ro` | event-driven: observables, subjects, `Map`/`Filter`/`Throttle` operators, backpressure | finite slice transforms |
| `golang-samber-mo` | monads: `mo.Option[T]`, `mo.Result[T]`, `mo.Either[L,R]`, `mo.Future[T]` | slice helpers, reactive streams |

**Routing examples:**

- "Users slice → map by ID" → `golang-samber-lo` (`lo.KeyBy`).
- "Channel of events with backpressure and rate limiting" → `golang-samber-ro`.
- "Optional value without nil" → `golang-samber-mo` (`mo.Some`/`mo.None`).
- "Compose error-returning functions without if-err chains" → `golang-samber-mo` (`mo.Result[T]`).

> Note: the source `lo`/`ro` descriptions still lack a mutual cross-reference naming `mo`. The boundary above is the intended design.

## 4. Error handling cluster

| Skill | Owns |
| --- | --- |
| `golang-error-handling` | idiomatic flow: `%w`, `errors.Is`/`As`, sentinels, handle-once, panic/recover |
| `golang-samber-oops` | the builder: `oops.New().Code().User().Hint()`, stack traces, APM, `oops.Recover` |
| `golang-safety` | preventing errors: nil checks, zero-value design, overflow guards, append aliasing |

**Routing examples:**

- "Wrap errors so callers can match them" → `golang-error-handling`.
- "Errors with HTTP codes and stack traces" → `golang-samber-oops`.
- "Service panics on nil deref" → `golang-safety` (prevention) plus `golang-troubleshooting` (the live crash).

## 5. Style / naming / lint / docs cluster

Each owns one slice of "code quality"; their descriptions already carry mutual disclaimers.

| Skill | Owns |
| --- | --- |
| `golang-code-style` | line formatting, blank-line discipline, short names in small scopes, comment placement |
| `golang-naming` | identifier rules: MixedCaps, no `GetX`/`IFoo`, package names, error names |
| `golang-lint` | golangci-lint YAML, which linters run, `//nolint` policy, CI wiring |
| `golang-documentation` | exported-symbol comments, package docs, README sections, `example_test.go`, `llms.txt` |

**Routing examples:**

- "Name this constructor" → `golang-naming`.
- "`exhaustive` lint errors" → `golang-lint`.
- "Package-level godoc layout" → `golang-documentation`.
- "Blank line between functions?" → `golang-code-style`.

## 6. CLI cluster

| Skill | Owns |
| --- | --- |
| `golang-cli` | exit codes, signals, stdin/stdout/stderr patterns, progress bars, terminal detection |
| `golang-spf13-cobra` | `cobra.Command`, `PersistentPreRunE`, `Args` validators, completion, `SetArgs` in tests |
| `golang-spf13-viper` | `BindPFlag`, `AutomaticEnv`, config files, `OnConfigChange`, `viper.Reset()` isolation |

**Routing examples:**

- "Exit code 2 on bad args" → `golang-cli`.
- "Shell completion for a cobra command" → `golang-spf13-cobra`.
- "Env vars override config" → `golang-spf13-viper`.
- "New CLI from scratch" → all three: `golang-cli` (architecture) + `golang-spf13-cobra` (tree) + `golang-spf13-viper` (config).

## 7. Testing cluster

| Skill | Owns |
| --- | --- |
| `golang-testing` | strategy: table-driven patterns, `t.Parallel()`, testcontainers, goleak, coverage, fuzz |
| `golang-stretchr-testify` | the API: `assert.Equal`, `require.NoError`, `mock.On`, `AssertExpectations`, `suite` lifecycle |

**Routing examples:**

- "Table-driven test" → `golang-testing`.
- "Assert a mock was called with specific args" → `golang-stretchr-testify`.
- "Goroutine leak detection in tests" → `golang-testing` (goleak).

## 8. design-patterns vs structs-interfaces

Both answer "how to design Go types" — the split is type mechanics versus architecture.

| Skill | Owns | Does not own |
| --- | --- | --- |
| `golang-structs-interfaces` | composition, embedding, assertions, struct tags, receiver choice, interface segregation | how types assemble into patterns |
| `golang-design-patterns` | functional options, middleware chains, circuit breakers, graceful shutdown, retries | low-level type mechanics |

**Overlap zone:** "DI via interfaces" — defining small interfaces is `golang-structs-interfaces`; wiring many components through them is `golang-design-patterns`.

**Routing examples:**

- "Value or pointer receiver?" → `golang-structs-interfaces`.
- "HTTP middleware chain" → `golang-design-patterns`.
- "Embed without leaking methods" → `golang-structs-interfaces`.
- "Functional options pattern" → `golang-design-patterns`.

> Note: neither description-level disclaimer exists yet in the source skills; the boundary above is the intended design.

## 9. concurrency vs context

They overlap precisely at "cancelling goroutines via context" — load both there.

| Skill | Owns | Does not own |
| --- | --- | --- |
| `golang-concurrency` | goroutine lifecycle, channel patterns, `WaitGroup`, errgroup, worker pools, fan-out/fan-in, race detection | context propagation rules |
| `golang-context` | `WithCancel`/`WithTimeout`/`WithValue`, propagation, `WithoutCancel` | goroutine coordination patterns |

**Routing examples:**

- "Cancel a goroutine from outside" → both: `golang-context` for the cancel API, `golang-concurrency` for `select { case <-ctx.Done() }`.
- "Fan out N workers and collect results" → `golang-concurrency`.
- "Carry a deadline through several call layers" → `golang-context`.

> Note: no description-level disclaimer exists yet; load both whenever the task touches both concerns.

## 10. safety vs security

Both prevent bugs, under different threat models.

| Skill | Owns | Does not own |
| --- | --- | --- |
| `golang-safety` | nil panics, overflow, append aliasing, concurrent maps, float equality, zero-value design | attackers, crypto, secrets |
| `golang-security` | SQL/command/LDAP injection, weak RNG, hardcoded secrets, TLS misconfig, SSRF, path traversal | internal runtime correctness |

**Routing examples:**

- "Nil-pointer panic" → `golang-safety`.
- "Is this SQL injected?" → `golang-security`.
- "`math/rand` for a token" → `golang-security` — use `crypto/rand`.
- "Slice grows after append" → `golang-safety` (aliasing).

> Note: cross-references exist in the skill bodies but not in the YAML frontmatter.

## 11. modernize vs lint

Both suggest edits — one adopts language features, the other manages the linter.

| Skill | Owns | Does not own |
| --- | --- | --- |
| `golang-modernize` | feature adoption: range-over-int, `min`/`max`, `iter.Seq`, `slices.SortFunc`, `slog`, `t.Context` | static-analysis configuration |
| `golang-lint` | golangci-lint YAML, linter selection, output interpretation, `//nolint` policy | language-feature adoption |

**Overlap zone:** linters (`govet`, `deadcode`, `perfsprint`) occasionally warn in the same direction modernize rewrites (e.g. "use slog over log"). Lint owns the tool config and the suppress-vs-fix decision; modernize owns the rewrite pattern once the feature is adopted.

**Routing examples:**

- "Replace an index loop with range-over-int" → `golang-modernize`.
- "CI fails `perfsprint`" → `golang-lint`.
- "Migrate `log` → `slog`?" → `golang-modernize`.
- "Only security linters, please" → `golang-lint`.

## 12. Package lookup / discovery cluster

All four touch third-party packages at different *stages*: query, recommend, manage, scan.

| Skill | Owns | Does not own |
| --- | --- | --- |
| `golang-pkg-go-dev` | read-only pkg.go.dev facts via `godig`: versions, docs/symbols/examples, importers, licenses, CVEs | picking a library, editing go.mod, scanning your tree, navigating your resolved code (→ `golang-gopls`) |
| `golang-popular-libraries` | recommending a library; stdlib-vs-third-party judgment | facts about a specific published package |
| `golang-dependency-management` | go.mod edits: `go get`, upgrades, pinning, `replace`/`exclude`, workspaces | browsing a package's docs or history |
| `golang-security` | whole-tree reachable-CVE scanning with `govulncheck`, remediation | one module's CVEs without scanning the tree |

**Overlap zone:** "is dependency X safe/current?" — `golang-pkg-go-dev` states the facts, `golang-dependency-management` performs the upgrade/pin, `golang-security` decides whether your *code path* can reach a vulnerability.

**Routing examples:**

- "Versions of `github.com/samber/lo`?" → `golang-pkg-go-dev` (`versions`).
- "Does `golang.org/x/text v0.3.0` have CVEs?" → `golang-pkg-go-dev` (`vulns`).
- "Who imports my library?" → `golang-pkg-go-dev` (`imported-by`).
- "Which logging library?" → `golang-popular-libraries`.
- "Upgrade `github.com/foo/bar`" → `golang-dependency-management`.
- "Reachable CVEs across my whole module" → `golang-security` (`govulncheck`).

> Prefer `golang-pkg-go-dev` over Context7 for any Go module fact lookup.

**Sub-boundary — `godig` vs `gopls`:** `godig` queries the remote pkg.go.dev index (works for packages you have not added; no local build). `gopls` reasons about your *actual resolved build* in `go.sum`, including `replace`-ed forks and local paths.

- "Where is `Foo` defined in my repo" / "find the call sites of this dependency symbol in my code" → `golang-gopls` (`go_search`/`go_symbol_references`) — `godig` cannot see unpublished local code or intra-repo call sites.
- "Does the package I haven't added yet have CVEs?" → `golang-pkg-go-dev` (`vulns`).
- "Can my current build actually reach a vulnerability?" → `golang-gopls` (`go_vulncheck`, single shot) or `golang-security` (`govulncheck`, whole tree).

Full `godig` vs gopls vs Context7 vs govulncheck breakdown: the `golang-how-to` SKILL.md ("godig vs gopls vs Context7 vs govulncheck").

## 13. refactoring vs the target-state skills

`golang-refactoring` owns the *process*: safely restructure existing code at scale. The destination shape belongs to five other skills.

| Skill | Owns | Does not own |
| --- | --- | --- |
| `golang-refactoring` | the safe staged mechanics: blast-radius mapping, PR ordering, the refactor-branch git model, coverage-adaptive safety net, tool-driven transforms | the target name, layout, patterns, or idioms |
| `golang-naming` | what an identifier should be renamed *to* | applying the rename safely repo-wide |
| `golang-project-layout` | target package/directory layout, module splits | moving code there without breaking every caller at once |
| `golang-code-style` | target control-flow shape (guard clauses, function size) | the mechanical transform to that shape |
| `golang-design-patterns` | target patterns: options structs, consumer-side interfaces, DI | sequencing a multi-step migration as reviewable PRs |
| `golang-modernize` | version-driven idiom sweeps (`interface{}`→`any`, `slices`/`maps`) | multi-step structural refactors needing staged review |

**Overlap zone:** almost every real refactor needs the target skill *and* the process skill — "rename and move type X" needs `golang-naming` (the new name) and `golang-project-layout` (the new location) for the destination, plus `golang-refactoring` (type-alias gradual repair, PR staging) for the journey. Load `golang-refactoring` beside whichever skill defines the destination.

**Routing examples:**

- "Function is too long" → `golang-code-style` (target) + `golang-refactoring` (extract mechanics, behavior verification).
- "Rename `Client.Send` → `Client.Publish` repo-wide" → `golang-naming` + `golang-refactoring` (workspace rename, risk tier, PR staging).
- "Move this type to a new package without breaking callers" → `golang-project-layout` + `golang-refactoring`.
- "`interface{}` → `any` across the module" → `golang-modernize` alone usually suffices; escalate to `golang-refactoring`'s staged flow only when the sweep needs progressive human review.
- "Multi-week break-up of a god package" → `golang-refactoring` primary, pulling in `golang-project-layout` and `golang-design-patterns` for each step's target.