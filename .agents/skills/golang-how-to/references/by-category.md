# Golang skills — complete catalog

The routing index behind `golang-how-to`. Every skill below lives at `fabianoflorentino/golang-agent-skills@<name>`.

**Notation:** ⭐️ recommended for every Go project · ⚙️ overridable by a company-specific skill · 🧠 requires engineering judgment (profiling, test strategy, security review) rather than a copyable recipe.

Use the intent-based routing table in the parent SKILL.md to pick primary and secondary skills, then come here for each skill's full "when to use" hook.

## Code Quality

### `fabianoflorentino/golang-agent-skills@golang-code-style` ⭐️ ⚙️

Formatting and project-level conventions: gofmt/goimports, line length, variable declaration style, blank lines, comment placement.

Use when the question is about formatting rules, style review, or project coding standards. Not naming rules → `golang-naming`, linter configuration → `golang-lint`, doc comments → `golang-documentation`.

### `fabianoflorentino/golang-agent-skills@golang-documentation` ⭐️ ⚙️

Documentation standards: package docs, godoc conventions, example functions, README structure, CHANGELOG, `llms.txt`, API-reference generation.

Use when writing or reviewing Go doc comments, READMEs, or API reference material. Not for comments that explain logic inside a function → `golang-code-style`.

### `fabianoflorentino/golang-agent-skills@golang-error-handling` ⭐️ ⚙️

Idiomatic error flow: creation, wrapping with `fmt.Errorf`/`%w`, `errors.Is`/`errors.As`, sentinel errors, custom error types, panic recovery, the handle-once rule.

Use when writing or reviewing error propagation, wrapping, logging, or recovery. Structured stats → `golang-samber-oops`; preventing panics before they occur → `golang-safety`.

### `fabianoflorentino/golang-agent-skills@golang-lint`

golangci-lint: configuration, presets, custom rules, CI integration, `//nolint` suppression, which linters to enable and how to read their output.

Use when setting up or tuning golangci-lint, triaging lint failures, or deciding the linter set. Conventions that linters do not enforce → `golang-code-style`.

### `fabianoflorentino/golang-agent-skills@golang-naming` ⭐️ ⚙️

Naming for every identifier kind: packages, constructors, structs, interfaces, constants, errors, receivers, acronyms, tests. MixedCaps rules, the `Get`-prefix ban, and the `utils`/`helpers` anti-patterns.

Use when naming a new type, function, package, or constant. Broader formatting → `golang-code-style`.

### `fabianoflorentino/golang-agent-skills@golang-safety` ⭐️

Defensive coding against panics and silent corruption: nil safety, append aliasing, concurrent map access, float comparison, zero-value design, numeric overflow.

Use when writing or reviewing code that could silently produce wrong results or crash. External threats → `golang-security`; error-handling idioms → `golang-error-handling`.

### `fabianoflorentino/golang-agent-skills@golang-security` ⭐️ 🧠

Security best practices: injection (SQL, command, XSS), cryptography, filesystem/network safety, secrets management, cookie security, tool configuration. Dedicated audit and review modes.

Use when auditing a codebase, writing security-sensitive code, or reviewing auth/crypto/secrets. Runtime correctness bugs → `golang-safety`.

### `fabianoflorentino/golang-agent-skills@golang-structs-interfaces` ⚙️

Type design: composition, embedding, type assertions, interface segregation, struct tags (JSON/YAML/DB), value vs pointer receivers.

Use when designing types, choosing receivers, writing struct tags, or shaping interface hierarchies. Architectural patterns built on those types → `golang-design-patterns`.

## Architecture & Design

### `fabianoflorentino/golang-agent-skills@golang-concurrency` ⚙️

Concurrency patterns: goroutines, channels, sync primitives, context cancellation, worker pools, fan-out/fan-in, pipelines, errgroup.

Use when writing concurrent code, coordinating goroutines, or reviewing for races. When goroutines are cancelled via context, load `golang-context` alongside it.

### `fabianoflorentino/golang-agent-skills@golang-context` ⚙️

Context usage: creation, cancellation, timeouts, values, propagation through call chains, `WithoutCancel`, common anti-patterns.

Use when propagating deadlines or cancellation, or passing request-scoped values. Not for code that merely accepts `ctx` as its first parameter without using it.

### `fabianoflorentino/golang-agent-skills@golang-data-structures` ⭐️

Internals and trade-offs: slices (capacity growth, append aliasing), maps, channels, sync primitives, `container/*`, generic collections, and when each is the right choice.

Use when picking a data structure, reasoning about slice/map performance, or reaching for `container/list`, `container/heap`, or a custom generic type.

### `fabianoflorentino/golang-agent-skills@golang-database` ⭐️ ⚙️

Database access: parameter binding, connection pooling, transactions, migrations, sqlboiler/sqlc code generation, query builders.

Use when writing SQL, designing repository layers, or configuring connections. Injection safety on those queries → also `golang-security`.

### `fabianoflorentino/golang-agent-skills@golang-dependency-injection` ⚙️

DI decision and design: constructor injection, interface-based wiring, the wire/dig/fx/samber-do comparison, and when DI is worth the complexity.

Use when deciding whether to adopt DI or designing constructor signatures. Once a library is chosen → `golang-google-wire`, `golang-uber-dig`, `golang-uber-fx`, or `golang-samber-do`.

### `fabianoflorentino/golang-agent-skills@golang-design-patterns` ⭐️ ⚙️

Architectural patterns in idiomatic Go: functional options, constructors, builder, middleware chains, circuit breaker, and guidance on architecture.

Use when choosing patterns, designing APIs, or planning resilience. Type-level mechanics (embedding, receivers) → `golang-structs-interfaces`.

### `fabianoflorentino/golang-agent-skills@golang-modernize` ⭐️

Modern-idiom adoption: range-over-int, `min`/`max` builtins, iterators, the `slices`/`maps`/`cmp`/`slog` stdlib, testing patterns (`t.Context`, `b.Loop`, synctest), and tooling upgrades.

Use when upgrading a codebase to a newer Go, or replacing pre-generics patterns. Not lint-rule enforcement → `golang-lint`; not structural refactors → `golang-refactoring`.

### `fabianoflorentino/golang-agent-skills@golang-refactoring` 🧠

The process of changing existing code safely at scale: a coverage-adaptive safety net, behavior-preserving transforms (gopls Rename/Extract, `gofmt -r`, `gopatch`), the Fowler catalog mapped to Go, breaking import cycles, and small stacked PRs.

Use when a function or type has outgrown itself, a smell blocks a feature, or the task is renaming, extracting, moving code, or planning a multi-step refactor. The *target shape* of each refactor is owned elsewhere → `golang-naming`, `golang-project-layout`, `golang-modernize`, `golang-code-style`, `golang-design-patterns`.

## QA & Performance

### `fabianoflorentino/golang-agent-skills@golang-benchmark` 🧠

Measurement: pprof and execution-tracer capture, flame graphs, `benchstat` comparison, CI regression detection, continuous profiling.

Use when measuring performance, capturing profiles, comparing runs, or wiring regression gates. Applying an optimization pattern → `golang-performance`; debugging a crash → `golang-troubleshooting`.

### `fabianoflorentino/golang-agent-skills@golang-observability` ⚙️

Always-on production signals: structured logging (`slog`), Prometheus metrics, OpenTelemetry tracing, pprof endpoints, alerting, Grafana dashboards.

Use when instrumenting a service for production monitoring. Temporary deep-dive investigation → `golang-benchmark` / `golang-performance`.

### `fabianoflorentino/golang-agent-skills@golang-performance` 🧠

Optimization patterns once a bottleneck is proven: allocation reduction, CPU efficiency, memory layout, GC tuning, pooling, caching, hot-path work.

Use when applying the *right fix* after measurement. How to measure → `golang-benchmark`; the debugging workflow → `golang-troubleshooting`.

### `fabianoflorentino/golang-agent-skills@golang-testing` ⭐️ 🧠 ⚙️

Test strategy and craftsmanship: table-driven tests, fuzzing, fixtures, goroutine-leak detection (goleak), snapshot testing, coverage, integration tests, parallel tests.

Use when writing or reviewing tests. testify APIs → `golang-stretchr-testify`; performance measurement → `golang-benchmark`.

### `fabianoflorentino/golang-agent-skills@golang-troubleshooting` ⭐️ 🧠

Systematic debugging: the 10-step methodology, common pitfalls, test-driven debugging, pprof capture, Delve, the race detector, GODEBUG tracing, production debugging.

Use when reacting to a panic, wrong output, or a hard-to-reproduce bug. Interpreting profiles → `golang-benchmark`; optimization patterns → `golang-performance`.

## Project Setup

### `fabianoflorentino/golang-agent-skills@golang-cli`

CLI application development: project layout, exit codes, signal handling, stdin/stdout/stderr patterns, argument parsing, terminal UX.

Use when building a CLI from scratch. Cobra command trees → `golang-spf13-cobra`; viper configuration → `golang-spf13-viper`.

### `fabianoflorentino/golang-agent-skills@golang-continuous-integration`

CI/CD for Go projects on GitHub Actions: build, test, lint, and release workflows.

Use when setting up or improving a Go project's pipeline.

### `fabianoflorentino/golang-agent-skills@golang-dependency-management`

Module strategy: go.mod conventions, versioning, `replace` directives, tool dependencies, multi-module workspaces.

Use when editing go.mod, diagnosing `replace`/`exclude` setups, or structuring a multi-module repo.

### `fabianoflorentino/golang-agent-skills@golang-gopls`

Semantic Go intelligence via `gopls` (its MCP server, the native `LSP` tool, or its CLI): definitions, references, package API, diagnostics, safe rename, extract/inline refactors, formatting, generated tests.

Use when navigating or editing code in the local workspace — jump to a definition, find call sites before a rename, read package dependencies, check diagnostics after an edit. The published ecosystem (versions, licenses, importers) → `golang-pkg-go-dev`; a whole-tree CVE audit → `golang-security`.

### `fabianoflorentino/golang-agent-skills@golang-pkg-go-dev`

Fact lookup through `godig` (a pkg.go.dev API client, CLI + MCP): docs, symbols, versions, importers, licenses, known CVEs — read-only, no auth, works for packages not yet in `go.mod`. Prefer over Context7 for any Go module.

Use when answering "what versions exist", "does this version have CVEs", "who imports this", or "show me the API/docs". Upgrading deps → `golang-dependency-management`; choosing a library → `golang-popular-libraries`; local resolved source and call sites → `golang-gopls`.

### `fabianoflorentino/golang-agent-skills@golang-popular-libraries`

Curated production-ready recommendations: when the stdlib is enough, when to reach for a package, and which package to prefer.

Use when choosing a library for a new concern. Deep guidance on a chosen library → that library's own skill.

### `fabianoflorentino/golang-agent-skills@golang-project-layout`

Project structure: `cmd`/`internal`/`pkg` conventions, monorepo layout, CLI project anatomy, and when flat is the right call.

Use when starting a project or restructuring one. Patterns *inside* the layout → `golang-design-patterns`.

### `fabianoflorentino/golang-agent-skills@golang-stay-updated`

Where to keep current: official channels, newsletters, communities, key blogs and people, conferences, and learning resources.

Use when tracking Go releases, proposals, and ecosystem news. Querying a specific module's facts → `golang-pkg-go-dev`.

## APIs

### `fabianoflorentino/golang-agent-skills@golang-rest`

REST/JSON APIs on `net/http`: resource and method design, JSON contract and encoding, RFC 9457 `problem+json` errors, request validation, cursor vs offset pagination, idempotency keys, content negotiation, versioning with deprecation headers, `http.Server` timeouts, httptest-based testing.

Use when building, reviewing, or debugging a REST/JSON API in Go. gRPC → `golang-grpc`; GraphQL → `golang-graphql`; OpenAPI docs → `golang-swagger`.

### `fabianoflorentino/golang-agent-skills@golang-graphql`

GraphQL APIs with gqlgen/graphql-go: schema definition, resolvers, subscriptions, dataloader, federation.

Use when building a GraphQL API in Go.

### `fabianoflorentino/golang-agent-skills@golang-grpc`

gRPC services: protobuf organization, service definitions, streaming, interceptors, error codes, the code-generation workflow.

Use when building or consuming a gRPC service. REST/OpenAPI documentation → `golang-swagger`.

### `fabianoflorentino/golang-agent-skills@golang-swagger`

OpenAPI/Swagger with swaggo/swag: annotation comments, `swag init` generation, framework integrations (gin, echo, fiber, chi), security definitions, doc struct tags.

Use when generating and maintaining OpenAPI docs from Go annotations.

## Dependency Injection

`golang-dependency-injection` owns the decision — see Architecture & Design. The four library skills each cover the concrete API of one choice.

### `fabianoflorentino/golang-agent-skills@golang-google-wire`

Compile-time DI with google/wire: provider sets, generated injectors, `wire.Build`, and structured DI patterns.

Use when the codebase imports `github.com/google/wire` or compiled-in wiring is the chosen approach. Reflection-based containers → `golang-uber-dig`.

### `fabianoflorentino/golang-agent-skills@golang-uber-dig`

Reflection DI with uber-go/dig: `Provide`/`Invoke`, `dig.In`/`dig.Out`, named values, value groups, optional dependencies, `Decorate`.

Use when the codebase imports `go.uber.org/dig`. Lifecycle and modules on top of it → `golang-uber-fx`.

### `fabianoflorentino/golang-agent-skills@golang-uber-fx`

Application framework over dig: `fx.New`, `fx.Provide`/`Invoke`, `fx.Module`, lifecycle hooks, `fx.Annotate`, `fx.Decorate`, signal-aware `Run`.

Use when the codebase imports `go.uber.org/fx`. Bare DI without lifecycle → `golang-uber-dig`.

### `fabianoflorentino/golang-agent-skills@golang-samber-do`

Type-safe DI with samber/do: service containers, scopes, lifecycle management, health checks, graceful shutdown.

Use when the codebase imports `github.com/samber/do`.

## Frameworks

### `fabianoflorentino/golang-agent-skills@golang-spf13-cobra`

CLI command trees with spf13/cobra: command hierarchy, `RunE` hooks, flag management, shell completion, usage templates, testing with `SetArgs`.

Use when the codebase imports `github.com/spf13/cobra`. Config layering → `golang-spf13-viper`; general CLI architecture → `golang-cli`.

### `fabianoflorentino/golang-agent-skills@golang-spf13-viper`

Layered configuration with spf13/viper: flag > env > file > KV > default precedence, `BindPFlag`, automatic env binding, hot reload, test isolation via `viper.Reset()`, remote KV.

Use when the codebase imports `github.com/spf13/viper`. Command structure → `golang-spf13-cobra`; CLI architecture → `golang-cli`.

## samber/*

### `fabianoflorentino/golang-agent-skills@golang-samber-do`

See Dependency Injection above.

### `fabianoflorentino/golang-agent-skills@golang-samber-hot`

In-memory caching with samber/hot: eviction algorithms (LRU, LFU, TinyLFU, W-TinyLFU, S3FIFO, ARC, TwoQueue, SIEVE, FIFO), TTL, loaders, sharding, stale-while-revalidate, missing-key caching, Prometheus metrics.

Use when the codebase imports `github.com/samber/hot` or a hot low-to-mid-cardinality resource needs caching.

### `fabianoflorentino/golang-agent-skills@golang-samber-lo`

Functional helpers for finite collections: 500+ type-safe generic functions over slices, maps, channels, and strings — `Map`, `Filter`, `Reduce`, `GroupBy`, `Chunk`, `Flatten`, `Find`, `Uniq`. Immutable core (`lo`), parallel (`lop`), in-place (`lom`), lazy iterators (`loi`), experimental SIMD (`lo/exp/simd`).

Use when the codebase imports `github.com/samber/lo` or a collection transform is needed. Streaming pipelines → `golang-samber-ro`.

### `fabianoflorentino/golang-agent-skills@golang-samber-mo` 🧠

Monadic types with samber/mo: `Option`, `Result`, `Either`, `Future`, `IO`, `Task`, `State` for typed nullability, error composition, and functional pipelines.

Use when the codebase imports `github.com/samber/mo` or monadic structure beats nil/if-err chaining. Nil-safety without the library → `golang-safety`; native error wrapping → `golang-error-handling`.

### `fabianoflorentino/golang-agent-skills@golang-samber-oops`

Structured errors with samber/oops: error builders, stack traces, error codes, context attributes, public vs developer messages, panic recovery, APM integration.

Use when the codebase imports `github.com/samber/oops`.

### `fabianoflorentino/golang-agent-skills@golang-samber-ro` 🧠

Reactive streams with samber/ro (ReactiveX): 150+ operators, cold/hot observables, five subject types, 40+ plugins, automatic backpressure, Go context integration.

Use when the codebase imports `github.com/samber/ro` or an event-driven pipeline is being built. Finite slice transforms → `golang-samber-lo`.

### `fabianoflorentino/golang-agent-skills@golang-samber-slog`

Structured logging extensions for samber/slog-*: multi-handler routing (`slog-multi`), sampling, formatters, HTTP middleware, 20+ backend sinks (Datadog, Sentry, Loki, Syslog, ...).

Use when the codebase imports any `github.com/samber/slog-*` package.

## Testing

### `fabianoflorentino/golang-agent-skills@golang-stretchr-testify`

Testing with stretchr/testify: `assert`, `require`, `mock`, `suite` — matchers, mock expectations and argument matching, suite lifecycle, custom matchers.

Use when the codebase imports `github.com/stretchr/testify`. Test architecture and strategy → `golang-testing`.

### `fabianoflorentino/golang-agent-skills@golang-testing` ⭐️ 🧠 ⚙️

See QA & Performance above.

## Pitfalls (100 Go Mistakes)

Eleven mistake-ordered checklists distilled from *100 Go Mistakes and How to Avoid Them*, one per chapter cluster. They excel at "have we hit mistake X?" audits; load them beside the general skill that owns each domain. The general skills win for writing fresh code.

| Skill | Terrains (source mistakes) |
| --- | --- |
| `golang-pitfalls-code-organization` | shadowing, nested code, `init`, getters/setters, interface pollution, `any`/generics, embedding, options, package layout, docs (#1–16) |
| `golang-pitfalls-concurrency-foundations` | concurrency vs parallelism, channels vs mutexes, data race vs race condition, workload types, GOMAXPROCS, contexts (#55–60) |
| `golang-pitfalls-concurrency-practice` | inappropriate contexts, goroutine lifetime, loop variables, select determinism, `chan struct{}`, nil channels, buffer size, `sync.WaitGroup` misuse, `sync.Cond`, errgroup, copied sync types (#61–74) |
| `golang-pitfalls-control-structures` | range value copies, range argument evaluation, loop-variable capture, map iteration assumptions, label scoping, `defer` in loops, select draining (#30–35) |
| `golang-pitfalls-data-types` | octal, overflow, floats, length vs capacity, slice init, nil vs empty, copies, append side effects, slice/map leaks, comparisons (#17–29) |
| `golang-pitfalls-error-handling` | panic over error, `%w` wrapping, `errors.Is`/`As`, sentinels vs types, double handling, ignored errors, deferred errors (#48–54) |
| `golang-pitfalls-functions-methods` | receiver choice, named results, nil receivers in interfaces, filenames vs `io.Reader`, defer evaluation (#42–47) |
| `golang-pitfalls-optimizations` | CPU caches, false sharing, ILP, alignment, stack vs heap, allocation reduction, inlining, pprof/tracer, GC, containers (#91–100) |
| `golang-pitfalls-standard-library` | duration units, `time.After`, JSON traps, `database/sql`, resources, `http.Error` returns, default clients/servers (#75–81) |
| `golang-pitfalls-strings` | runes, byte-length, iteration, trim misuse, `strings.Builder`, conversions, substring leaks (#36–41) |
| `golang-pitfalls-testing` | test categorization, race flag, execution modes, table tests, sleeps, time API, httptest/iotest, benchmark accuracy, coverage, fuzzing (#82–90) |