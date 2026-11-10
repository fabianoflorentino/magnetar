# Tooling for Refactoring

The tool reference for `fabianoflorentino/golang-agent-skills@golang-refactoring`: every mechanical-rewrite tool worth reaching for, from the primary actuator (`gopls`) down to hand-rolled `go/analysis` fixers. Arrange them as an escalation ladder — pick the least powerful tool that solves the problem, and climb only when its limits block you. [catalog.md](catalog.md) maps tools to Fowler refactorings; [workflow.md](workflow.md) places a tool-driven step inside the staged-PR flow.

## gopls — the primary actuator

gopls performs most Low- and Medium-risk transforms — Rename, Inline, Extract, and the `refactor.rewrite.*` family — and its actions are listed on each entry in [catalog.md](catalog.md) and tiered in the Risk Stratification table in [SKILL.md](../SKILL.md). The full code-action reference, CLI invocation, safety behavior, and MCP server setup belong to `fabianoflorentino/golang-agent-skills@golang-gopls`; this file covers the tools that skill does not.

## The bulk rewrite ladder

When a change recurs across many call sites, generate a rewrite rather than hand-editing each one. Start at the top; move down only when the current tool blocks the rewrite you need.

### `gofmt -r` — syntactic, single-expression

Purely syntactic and type-unaware: it matches expression shape, not types, and one call rewrites a single expression. Wildcards are single lowercase identifiers that match any sub-expression.

```bash
gofmt -r 'bytes.Compare(a, b) == 0 -> bytes.Equal(a, b)' -w file.go
gofmt -r 'bytes.Compare(a, b) != 0 -> !bytes.Equal(a, b)' -w file.go
gofmt -s -w file.go    # -s simplifies (s[a:len(s)] -> s[a:])
gofmt -l .             # list non-conforming files — a one-line CI gate
gofmt -d file.go       # diff without writing
```

Because it cannot see types, it cannot distinguish `bytes.Compare` from a same-named function in another package — that distinction needs `eg`.

### `eg` — type-aware, example-based

`golang.org/x/tools/cmd/eg` rewrites by example: a template declares `before`/`after` functions of identical type, each with a single-expression body.

```go
// template.go
package template

import (
	"errors"
	"fmt"
)

func before(s string) error { return fmt.Errorf("%s", s) }
func after(s string) error  { return errors.New(s) }
```

```bash
eg -t template.go -w ./...
```

Matching is semantic, not textual — `func(x int)` in the template matches `func(y int)` at the call site. Limits: expressions only (no statements or function literals); the rewrite cannot change the expression's type; imports are added but never removed (run `goimports` afterward); duplicating a wildcard in `after` duplicates the side effects of the matched expression.

### `gopatch` — statement-level, import-aware

`github.com/uber-go/gopatch` operates at the statement level and tracks imports as part of the patch, so it can work on code that does not fully compile mid-refactor — useful for the messy in-between states of a large migration. A patch declares metavariables between `@@` markers, then a diff-like body:

```
@@
var x expression
@@
-errors.New(fmt.Sprintf(x))
+fmt.Errorf(x)
```

```bash
gopatch -p rewrite.patch ./...
gopatch -d -p rewrite.patch ./...    # dry-run — diff only
```

Still beta — the project frames it as covering roughly 80% of a migration. A pattern cannot match an import statement in isolation; something must follow it in the pattern.

### `go/analysis` + SuggestedFixes — bespoke, testable

For a rewrite too specific for the above three, write an `analysis.Analyzer`. Its `Run(pass)` walks the type-checked AST and reports `analysis.Diagnostic{SuggestedFixes: [...]}` at each match. Test against `.golden` files with `analysistest.RunWithSuggestedFixes`, then ship it as a `singlechecker`/`multichecker -fix` binary, or run it through `go vet -vettool=<path>`.

### `go fix` — the `go/analysis`-based fixer suite

As of Go 1.26, `go fix` is rewritten onto the `go/analysis` framework and has converged with `go vet`; this is where the `modernize` fixer suite lives (`rangeint`, `mapsloop`, `minmax`, `any`, `stringscut`, `omitzero`, and more). Coverage shifts release to release — Go 1.27 added `atomictypes`, `embedlit`, `slicesbackward`, `unsafefuncs`, removed `fmtappendf`, and renamed `waitgroup` to `waitgroupgo`; → See `fabianoflorentino/golang-agent-skills@golang-modernize` skill for the idiom-by-idiom breakdown. On an older toolchain, run the equivalent analyzers through `singlechecker`/`multichecker -fix` instead.

The `//go:fix inline` directive marks a function or constant so that `go fix` inlines every call site into its replacement — a machine-executable way to finish a deprecation migration once the replacement exists:

```bash
go fix ./...
go run golang.org/x/tools/go/analysis/passes/inline/cmd/inline@latest -fix ./...
```

### `dave/dst` — comment- and formatting-preserving AST edits

`go/ast` stores comments in a side table keyed by byte offset, so reordering or deleting nodes desyncs comments from the code they were attached to — the root cause of the Extract/Inline comment-loss caveat in [catalog.md](catalog.md). `github.com/dave/dst` (Decorated Syntax Tree) attaches comments and blank-line spacing as node-local decorations, so hand-rolled AST rewrites round-trip them via `decorator.Parse` / `decorator.Print`. `dstutil.Apply` mirrors `astutil.Apply`'s visitor API, so existing `go/ast` logic ports over directly. Reach for it only when a bespoke fixer must preserve comments that `go/ast` rewriting would scatter.

### Always run after a bulk rewrite

```bash
goimports -w .    # order imports left dangling by gofmt -r, eg, or a hand-rolled fixer
deadcode ./...                     # code orphaned by a removal-heavy refactor
deadcode -test ./...               # include test binaries — unreached public API here signals coverage, not dead code
deadcode -whylive=funcName ./...   # shortest reachability path proving a function is live
```

`deadcode` builds its reachability graph with Rapid Type Analysis from `main`/`init`, so it is unsound with respect to assembly, `go:linkname`, and reflection-driven dispatch — treat a "dead" verdict as a strong hint, not a proof, on code using any of those.

**Never `sed`/`perl` a structural Go change.** None of these text tools have grammar awareness — a pattern that matches inside a string literal or comment gets rewritten alongside real code. Always finish a bulk rewrite with `goimports`, even after a tool that claims to manage imports itself.

## Structure-discovery tools

These feed the planning gate in [workflow.md](workflow.md) — map the blast radius before choosing a tool from the ladder above.

| Tool | What it answers |
| --- | --- |
| `golang.org/x/tools/go/callgraph` (`rta`/`cha`/`static`/`vta` algorithms) | Who can reach this function statically across the whole build — the algorithms trade precision for speed differently |
| gopls call hierarchy (`textDocument/prepareCallHierarchy`) | Incoming/outgoing calls for one symbol, interactively |
| `go_references` / `go_symbol_references` (gopls MCP) | Every reference to a symbol from an agent context, without a full callgraph build |
| `go mod graph` | Module-level dependency edges, for blast radius crossing module boundaries |

## Cross-references

- [catalog.md](catalog.md) — which tool mechanizes each Fowler refactoring.
- [workflow.md](workflow.md) — the planning gate these discovery tools feed, and where a tool-driven step fits into the staged-PR process.
