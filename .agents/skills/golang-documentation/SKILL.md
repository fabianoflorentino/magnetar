---
name: golang-documentation
description: "Golang documentation — doc comments, godoc/GoDoc rendering, package docs via doc.go, code review in documentation, examples as living documentation, license headers, Diátaxis guide structure (How-to / Reference / Explanation), and ASCII Doc conventions. Apply when writing or reviewing Go package documentation, creating package-level reads, or structuring skill and project docs. For lint enforcement of comment rules see `fabianoflorentino/golang-agent-skills@golang-lint`; for benchmark/doc templates see `fabianoflorentino/golang-agent-skills@golang-benchmark`."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "📝"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch
paths:
  - "**/*.go"
---

# Documentation in Go

**Persona:** You are a Go documentation engineer. Documentation is part of the product: `go doc` output, package headers, and examples are what users and review readers hit first, and they must be as disciplined as the code itself.

**Modes:**

- **Write** — produce or update doc comments, package READMEs, `doc.go` headers, and Examples. Sequential.
- **Review** — check a diff's documentation: doc-comment coverage, contract accuracy, `(deprecated)` markers, example drift. Sequential.
- **Audit** — sweep a package for missing or stale docs. Parallel sub-agents by package when the tree is large.

**When to use:** any task that adds, edits, or reviews comments, package registration, examples, or project guides. For enforcing comment rules in CI use `golang-lint`; for reference-generation structure use `golang-benchmark`'s reference conventions.

## Doc comments form `go doc` output

The first sentence of a doc comment is the summary shown by `go doc`, pkg.go.dev listings, and IDE hover. It must name the declared item and state the contract precisely. Comments are prose — never a rustic fragment.

```go
// Price takes a quantity and a unit price and returns the total
// before tax. Zero or negative quantities return zero.
func Price(qty, unit float64) float64
```

Structural conventions (a shorter form of standard rules — run the linter for the full set in `golang-lint`):

```
Package comment:    Package x implements Y. // ends with the name
Exported func:      Name starts the sentence in an imperative or declarative form
Unexported identifier: no doc comment required
Method on type:     Name — receiver verbs kept tight
Type:               Type name starts the sentence
(replaced):         // Deprecated: use X instead.
```

- Capitalize proper nouns (`HTTP`, `JSON`), keep messages lowercase and free of trailing punctuation.
- Deprecation uses the `// Deprecated:` marker, not freeform wording — tooling keys on it.
- Commentary must explain behavior and contracts, not restate mechanics. If the code reads itself, one sentence of "why" beats three of "what".

## Package documentation with doc.go

A `doc.go` in the package folder holds the package-level comment — the first screen of `godoc`. Put the big picture there: what the package is for, how pieces fit, who the audience is. Keep it tight and link further reading rather than embedding it.

```go
// Package price calculates prices and discounts for the storefront.
//
// It composes the following models:
//
//   - pricing rules: QuantityRules, CouponRules
//   - calculation: Price, Discount
//
// # Guides
//
// See README.md for the full pricing guide, and _examples for runnable
// recipes.
package price
```

`.go` files in `_test.go` are ignored by `go fmt`'s package docs; if your package is tiny (one file), the file's own doc comment is enough — do not invent a `doc.go` for ceremony.

## Guide structure: How-to / Reference / Explanation

Keep in-project guides aligned with the Diátaxis split so readers hop to the right section by intent:

| Kind | Answers | Example | Tone |
| --- | --- | --- | --- |
| How-to | "How do I do this?" | "Adding a discount tier" | Single goal, ordered steps, concrete |
| Reference | "What is this?" | API listing, options table | Exhaustive, terse, no narrative |
| Explanation | "Why is it like this?" | "Why percent discount is capped" | Context, trade-offs, reasoning |

Each how-to starts with the outcome and a one-line completion criterion, then steps. This mirrors the house template used by this catalog's skills (Persona / Modes / When to use + appendices).

## Examples are living documentation

`Example` functions in `_test.go` double as executable examples rendered by pkg.go.dev with their `// Output:` verified by `go test` (see `golang-testing`). Write one named example per non-trivial concept and keep them aggregate-worthy — a reader should be able to copy the block and run.

```go
func ExamplePrice_discount() {
    total := Price(3, Price(100, 0).WithTier(2))
    fmt.Println(total)
    // Output: 240
}
```

## License headers

Keep headers minimal — a single `SPDX-License-Identifier` line, or none when the repo license covers everything. Never paste long boilerplate per file; `reuse lint` expects the SPDX form.

## Code blocks in docs

Fenced blocks are literal; indented blocks are prose text by default in godoc. When documenting Go APIs in comments, fenced blocks keep formatting exact:

```go
//     qty := 3
//     total := Price(float64(qty), 10)
```

## Quick reference

```bash
go doc .            # this package
go doc ./...        # walk all packages
go doc price.Price  # the function
go test -run Example -v ./...   # verify examples
sed -i 's|// Deprecated: .*||' .  # always via the proper marker, never regex
```

## Audit checklist

- [ ] Every exported identifier has a doc comment; first line is the `go doc` summary
- [ ] Contracts stated, not mechanics restated
- [ ] `// Deprecated:` markers used for removals
- [ ] Package-level comment exists (`doc.go` or file header)
- [ ] Examples compile and match their `// Output:`
- [ ] No stale references to moved packages or renamed symbols

## Cross-references

- `golang-lint` — `revive`/`goimprover` rules enforcing comment coverage.
- `golang-testing` — Examples as executable specs.
- `golang-pkg-go-dev` — how pkg.go.dev renders and evaluates your docs.
- `golang-code-style` — comment style and naming adjacent to docs hygiene.
- `golang-benchmark` — reference documentation for measurement workflows.
