---
name: golang-code-style
description: "Golang code style conventions — line length and breaking, variable declarations, control flow clarity, when comments help vs hurt. Use when writing or reviewing Go code, asking about style or clarity, or establishing project coding standards. Not for naming conventions (→ See `fabianoflorentino/golang-agent-skills@golang-naming` skill), linter configuration (→ See `fabianoflorentino/golang-agent-skills@golang-lint` skill), or doc comments (→ See `fabianoflorentino/golang-agent-skills@golang-documentation` skill)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🎨"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
paths:
  - "**/*.go"
---

# Code style in Go

**Persona:** You are a Go engineer for whom clarity is a deliverable. Linters handle formatting; this skill carries the judgment: line breaking at semantic boundaries, declarations that signal intent, and control flow that reads top-to-bottom.

**Modes:**

- **Write** — produce code following the style below. Sequential.
- **Review** — audit a diff for nesting, `else` chains, signal-sloppy declarations, needless comments. Sequential.

**When to use:** writing or reviewing Go for style/clarity, or setting project standards. Naming → `golang-naming`; lint config → `golang-lint`; doc comments → `golang-documentation`.

## Line breaking

No rigid column limit, but past ~120 characters a line MUST break — at a **semantic boundary**, not a character count. A call with 4+ arguments puts one argument per line:

```go
mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
    handleUsers(
        w,
        r,
        serviceName,
        cfg,
        logger,
        authMiddleware,
    )
})
```

A function signature that cannot fit on one line is usually a signature with too many parameters — the fix is an options struct, not folding.

## Declarations

`:=` for initialized values; `var` for "starts at zero" — the keyword is part of the meaning:

```go
var count int        // zero, set later
name := "default"    // non-zero
var buf bytes.Buffer // zero value ready to use
```

Slices and maps are initialized, never nil (nil map writes panic; nil slices marshal to `null`):

```go
users := []User{}                   // always initialized
m := map[string]int{}               // never nil
users := make([]User, 0, len(ids))  // preallocate when capacity is known
```

Don't preallocate speculatively. Composite literals use field names — positional fields break when the type grows.

## Control flow

- **Errors and edge cases first** (early return); the happy path runs at minimal indentation.
- **Drop `else` when the `if` body returns/breaks/continues.** For mutually exclusive overrides, default-then-override reads clearest:

  ```go
  level := slog.LevelInfo
  switch {
  case debug:
      level = slog.LevelDebug
  case verbose:
      level = slog.LevelWarn
  }
  ```

- **Repeated comparisons on one value → `switch`**, with a `default` that panics on the impossible.
- **3+ operand conditions become named booleans:**

  ```go
  isAdmin := user.Role == RoleAdmin
  isOwner := resource.OwnerID == user.ID
  if isAdmin || isOwner || permissions.Contains(PermOverride) {
      allow()
  }
  ```

- **Scope narrow check-only variables to the `if`:** `if err := validate(in); err != nil`.

## Functions

- Short, focused, one job.
- At most 4 parameters; beyond that, an options struct.
- Parameter order: `context.Context` first, then inputs, then output destinations.
- Explicit named returns in non-trivial functions; naked returns only where the values are obvious in 1–3 lines.
- Iterate with `range` (and `range n`, Go 1.22+), not index math.

## Value vs pointer parameters

Small types (`string`, `int`, `bool`, `time.Time`) by value. Pointers when mutating, for large structs (~128+ bytes), or when `nil` is meaningful.

## Organization within files

- Group by concern: package doc, imports, constants, types, constructors, methods, helpers.
- One primary type per file when it carries significant methods.
- Blank imports (`_ "pkg"`) are side-effect registration — restrict to `main` and test packages so the effect is visible, not hidden in library code.
- Dot imports pollute the namespace; never in library code.
- Unexport aggressively — exporting is easy, unexporting is a breaking change (`golang-gopls` renames safely across call sites).

## Strings and conversions

`strconv` for simple conversions (fast), `fmt.Sprintf` for structured output, `%q` in errors to show string boundaries, `strings.Builder` in loops, `+` for single concatenations. Prefer narrow explicit conversions; generics over `any`.

## Comments

Comments explain why and what the code promises — never restate what the code already shows. When you break a convention, comment that you are breaking it and why.

## Philosophy

- "A little copying is better than a little dependency."
- Reach for `slices`, `maps`, and `samber/lo` (filter/group/chunk) instead of hand-rolling.
- "Reflection is never clear" — avoid `reflect` unless it earns its opacity.
- Don't abstract prematurely; extract when the pattern stabilizes.
- Minimize the public surface: every exported name is a commitment.

## Parallel style reviews

For a large tree, fan out one sub-agent per independent concern — control flow, function design, declarations, strings, organization — and merge findings.

## Enforce with linters

`gofmt`, `gofumpt`, `goimports`, `gocritic`, `revive`, `wsl` automate most of this — wire them through `golang-lint`.

## Cross-references

- `golang-naming` — identifier conventions.
- `golang-structs-interfaces` — receivers and interface design.
- `golang-design-patterns` — options, builders, constructors.
- `golang-documentation` — doc comments.
- `golang-lint` — enforcement.
- `golang-refactoring` — mechanically applying guard-clause extraction and options-struct migration across many call sites.
