---
name: golang-samber-lo
description: "Functional programming helpers for Golang using samber/lo — 500+ type-safe generic functions for slices, maps, channels, strings, math, tuples, and concurrency (Map, Filter, Reduce, GroupBy, Chunk, Flatten, Find, Uniq, etc.). Core immutable package (lo), concurrent variants (lo/parallel aka lop), in-place mutations (lo/mutable aka lom), lazy iterators (lo/it aka loi for Go 1.23+), and experimental SIMD (lo/exp/simd). Apply when using or adopting samber/lo, when the codebase imports github.com/samber/lo, or when implementing functional-style data transformations in Go. Not for streaming pipelines (→ See `fabianoflorentino/golang-agent-skills@golang-samber-ro` skill)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🧰"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) mcp__context7__resolve-library-id mcp__context7__query-docs AskUserQuestion Bash(godig:*) Bash(gopls:*) LSP mcp__gopls__*
paths:
  - "**/*.go"
---

**Persona:** You are a Go engineer who prefers declarative collection transforms over hand-rolled loops. You reach for `lo` to cut boilerplate, and you know exactly when the stdlib already covers the operation.

**Modes:**

- **Build** — using `lo` in new or existing transforms.
- **Review** — spotting misuse: `lop`/`lom` where `lo` suffices, `Must` in request paths, and stdlib-redundant helpers.
- **Optimize** — confirming with a profiler before swapping eager code for `lom`/`lom`/`loi`.

**When to use:** any task transforming slices, maps, channels, or tuples in Go. For infinite/time-driven streams reach for `golang-samber-ro` instead; `golang-data-structures` covers picking the right collection type in the first place.

## What `lo` adds over the stdlib

The `slices` and `maps` packages cover the basics: sort, contains, keys, values, clone. Everything beyond that — `Map`, `Filter`, `Reduce`, `GroupBy`, `Chunk`, `Flatten`, `Zip` — used to mean a for-loop plus a scratch slice. `lo` provides those as generic, type-checked helpers with no reflection and no `interface{}` casts:

- returns new collections (immutable by default, safe to share across goroutines for reading);
- chains because each helper takes and yields plain `[]T`/`map[K]V`;
- drops to zero external dependencies.

## Packages and when to use them

| Import | Alias | Use when | Trade-off |
| --- | --- | --- | --- |
| `github.com/samber/lo` | `lo` | default for everything | allocates fresh collections |
| `github.com/samber/lo/parallel` | `lop` | CPU-bound transforms on 1000+ items | goroutine overhead; not for I/O or tiny slices |
| `github.com/samber/lo/mutable` | `lom` | allocation pressure proven by pprof | mutates the input; breaks read sharing |
| `github.com/samber/lo/it` | `loi` | Go 1.23+ iterator chains over large data | lazy evaluation, more moving parts |
| `github.com/samber/lo/exp/simd` | — | numeric bulk ops, only after benchmarking | experimental, amd64-only |

Rules of thumb: `lop` is for CPU parallelism, not I/O fan-out (use `errgroup` for that); `lom` sacrifices immutability, so only adopt it on measured alloc pressure; `loi` removes intermediate allocations in chains like `Map → Filter → Take`. See [package guide](references/package-guide.md) for the full comparison.

## Core transforms

```go
names := lo.Map(users, func(u User, _ int) string { return u.Name })

paid := lo.Filter(invoices, func(v Invoice, _ int) bool { return v.Status == "paid" })

byStatus := lo.GroupBy(tasks, func(t Task, _ int) string { return t.Status })

matches, found := lo.Find(users, func(u User, _ int) bool { return u.ID == id })
```

`Reduce` folds a collection into one value:

```go
total := lo.Reduce(orders, func(acc float64, o Order, _ int) float64 { return acc + o.Amount }, 0)
```

## Error-aware variants

Most helpers ship an `Err` form that stops at the first error and propagates it — `MapErr`, `FilterErr`, `ReduceErr`. Prefer these over collecting errors by hand:

```go
responses, err := lo.MapErr(urls, func(u string, _ int) (Response, error) {
    return client.Get(u)
})
```

## Common mistakes

| Mistake | Why it fails | Fix |
| --- | --- | --- |
| `lo.Contains`/`lo.Sort` when stdlib exists | dependency for a solved problem | `slices.Contains`, `slices.Sort` (1.21+) |
| `lop.Map` on 10 items | goroutine overhead beats the work | plain `lo.Map`; `lop` shines at ~1000+ |
| Assuming `lo.Filter` mutates | `lo` is immutable by default | `lom.Filter` only when in-place is intended |
| `lo.Must` on a request path | panics are a crash, not an error | non-Must variant + explicit handling |
| Chaining eager transforms on huge data | intermediate slice per step | `loi` lazy iterators |
| `lo.MustX` to "simplify" production code | hides failures and crashes the process | reserve `Must` for tests and init |

## Best practices

1. Prefer the stdlib when it covers the operation — `slices.Contains`, `slices.Sort`, `maps.Keys`/`maps.Values` (1.23+), `cmp.Compare`.
2. Compose helpers instead of writing nested loops; each one is a branded building block.
3. Only move to `lom`/`lop`/`loi` after `go tool pprof -alloc_objects` or a CPU profile points at the transform.
4. Use the `Err` variants in I/O-heavy transforms so one failure stops the batch early.
5. Verify magic: `Must` belongs in tests and initialization, not request handlers.
6. Keep `lo` out of hot inner loops when the data is small — a two-line loop is sometimes the honest answer.

## Full catalog

The complete function list (300+ entries) lives in [API reference](references/api-reference.md); composition, stdlib interop, and iterator pipelines are in [advanced patterns](references/advanced-patterns.md). Known bugs surface at [samber/lo issues](https://github.com/samber/lo/issues).

## Cross-references

- `golang-samber-ro` — reactive pipelines over infinite, time-driven streams.
- `golang-samber-mo` — Option/Result/Either values that compose with `lo` transforms.
- `golang-data-structures` — choosing the underlying collection before transforming it.
- `golang-performance` — profiling discipline before adopting `lop`/`lom`/`loi`.
- `golang-pkg-go-dev` / `golang-gopls` — package facts and call-site navigation for `lo` usage.
