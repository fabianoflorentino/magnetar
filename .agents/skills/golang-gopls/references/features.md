# gopls feature catalog

Naming each LSP request or `CodeAction` kind by its exact upstream term so a specific behavior can be looked up in the official docs. Source: [tip.golang.org/gopls/features](https://tip.golang.org/gopls/features/).

## Navigation

- **Definition** (`textDocument/definition`, CLI `gopls definition`) — jumps to a symbol's declaration. Besides plain identifiers it understands more targets: an import path lists the imported package's declarations; a `go:linkname` directive resolves to the linked symbol; a `go:embed` pattern finds the embedded file; a doc-comment link follows the reference; a non-Go function can return the assembly implementation; `return` locates the named result variables; `goto`/`break`/`continue` find the target label or block. When you are already at the declaration, most clients quietly reinterpret the request as "find references".
- **Type Definition** (`textDocument/typeDefinition`, no CLI form) — unwraps pointer, array, slice, channel, and map constructions to the underlying named type (for `x chan []*T` it reports `T`'s definition). It resolves only symbols, not arbitrary expressions, and it is absent from the native `LSP` tool's fixed operation list, so only a full editor client can reach it — there is no agent-invocable path.
- **References** (`textDocument/references`, CLI `gopls references`) — every use of a symbol, with context sensitivity: interface method references include concrete implementations; a package declaration matches direct imports and other files' package clauses; an embedded field reports only field references (use Type Definition for the type itself). **Scoping gotcha:** results cover only the build configuration of the queried file — a query from `foo_windows.go` will not surface a match in `bar_linux.go`. Built-ins (`int`, `append`) are rejected as too numerous to be useful.
- **Implementation** (`textDocument/implementation`, CLI `gopls implementation`) — from an interface, concrete implementations and sub-interfaces; from a concrete type, satisfied interfaces; from an interface method, the concrete methods satisfying it, and vice versa. Type matching uses method sets with generic types as wildcards (a candidate counts when some instantiation would work), function matching uses signatures. LSP's subtype bias makes this query directionally asymmetric; Type Hierarchy gives the full bidirectional view.
- **Document Symbol** (`textDocument/documentSymbol`, CLI `gopls symbols`) — outline of one file's top-level declarations. File-scoped; cross-file lookup is Symbol/Workspace Symbol's job.
- **Symbol / Workspace Symbol** (`workspace/symbol`, CLI `gopls workspace_symbol`) — fuzzy search across the whole workspace. The default `fastFuzzy` matcher (FZF-inspired) tolerates abbreviations and typos — `DocSym` still finds `DocumentSymbol`. Behavior is shaped by `symbolMatcher`, `symbolStyle`, and `symbolScope` settings (see [settings.md](settings.md)); `directoryFilters` prunes directories from the search.
- **Selection Range** (`textDocument/selectionRange`, no CLI) — expands or contracts the current selection along syntactic boundaries (expression → statement → block → function). Useful for selecting exactly the region an Extract refactor needs.
- **Call Hierarchy** (`textDocument/prepareCallHierarchy` + incoming/outgoing calls, CLI `gopls call_hierarchy`) — callers and callees of a function as a static graph. **Static calls only:** dispatch through a function value or interface method is invisible because detecting it is not analytically tractable. Invoke on the function declaration's name, and corroborate with References when dynamic dispatch matters.
- **Type Hierarchy** (`textDocument/prepareTypeHierarchy` + subtypes/supertypes, no CLI yet) — the bidirectional subtyping relation: which types implement an interface and which interfaces a type satisfies. Limited to **named types** (Implementation also matches unnamed function types); type aliases are excluded; function-local types are visible only within the same package.

## Passive (always-on)

These run continuously across an editor session and need no invocation. Most degrade when the surrounding package has build errors, since they depend on successful type-checking.

- **Hover** (`textDocument/hover`) — symbol name/kind/type/value, doc comment (with clickable links like `[fmt.Printf]`), promoted methods from embedded fields, struct field size/offset with wasted-space percentage (flagged at ≥20% waste), expanded `//go:embed` patterns, `//go:linkname` targets, and the Go release that introduced a given stdlib symbol. `hoverKind` sets verbosity; `linkTarget` sets the doc-link base.
- **Signature Help** (`textDocument/signatureHelp`) — parameter names/types/docs for the call in progress with the active parameter highlighted; works even with the cursor inside the function name, not just the parens.
- **Document Highlight** (`textDocument/documentHighlight`) — highlights every identifier referencing the same symbol in view, plus related tokens: named results and their returns, loop control keywords (`for`/`break`/`continue`), switch tokens, and a function with its own return statements. Read vs. write references are usually color-coded differently by the client.
- **Inlay Hint** (`textDocument/inlayHint`) — inline annotations, off by default (visual clutter), toggled per kind via `hints`: `parameterNames` (argument labels at call sites), `assignVariableTypes`, `compositeLiteralFields`, `compositeLiteralTypes`, `constantValues` (computed `iota` values included), `functionTypeParameters` (generic instantiations), `rangeVariableTypes`.
- **Semantic Tokens** (`textDocument/semanticTokens`) — richer coloring than naive lexing: token types (`function`, `keyword`, `macro`, `method`, `namespace`, `number`, `operator`, `parameter`, `string`, `type`, `typeParameter`, `variable`, …) plus modifiers including a custom `shadowing` modifier that flags shadowed declarations. Off by default due to type-checking latency (`semanticTokens` setting); `noSemanticString`/`noSemanticNumber` let a client opt out of just those two kinds.
- **Folding Range** (`textDocument/foldingRange`) — collapsible regions for large comments, functions, and blocks.
- **Document Link** (`textDocument/documentLink`) — makes URLs in doc comments and import declarations clickable (imports link to their pkg.go.dev page). Controlled by `importShortcut` and `linkTarget`.

## Diagnostics

Three sources, distinguished by the LSP diagnostic's `source` field:

1. **Compilation errors** — `go list` supplies package metadata (`source: "go list"`); gopls then mimics the compiler front end itself: read, scan, parse, type-check (`source: "compiler"`).
2. **Analysis findings** — the `go vet` analysis framework plus gopls's own analyzers, each reporting under its own analyzer name; the `printf` analyzer (format string/argument mismatches) is a representative example.
3. **Compiler optimization details** — off by default; a per-package `source.toggleCompilerOptDetails` code action turns it on. Surfaces escape-analysis results, nil-check elimination, and inlining decisions, and only on packages that are otherwise error-free.

**Recomputation timing:** open-file compile errors update within tens of milliseconds of a keystroke. Workspace-wide analysis recomputes after roughly a second of idle, tunable via `diagnosticsDelay`; `diagnosticsTrigger` can shift this to save-triggered. Clients may also pull diagnostics explicitly (`textDocument/diagnostic`) when initialized with `pullDiagnostics: true` — off by default for performance.

**Notable quick fixes**, offered as code actions attached to a diagnostic:

- `fillreturns` — heuristically completes an incomplete `return`.
- `stubMissingInterfaceMethods` — generates stubs when a concrete type does not yet satisfy a required interface.
- `StubMissingCalledFunction` — stubs an undefined function/method, inferring its signature from the call site.
- `CreateUndeclared` — declares a missing variable or function based on how it is used.
- `source.fixAll`-marked fixes are considered unconditionally safe, and most editors offer a one-shot "apply all".

CLI: `gopls check <file>` (`-severity=hint|info|warning|error`, default `warning`).

## Transformation

Three mechanisms: **Formatting** and **Rename** are primary LSP requests; most of the rest are **CodeActions** (requested per range, returning a direct edit or a lazily-computed command); a few dependency-management actions are **CodeLenses**.

- **Formatting** (`textDocument/formatting`, CLI `gopls format`) — canonical Go formatting; client-supplied formatting options are ignored. `gofumpt: true` opts into `mvdan.cc/gofumpt`'s stricter rules.
- **Organize Imports** (`source.organizeImports`, CLI `gopls imports`) — removes unused/duplicate imports, adds missing ones (via workspace-wide heuristics that are occasionally surprising), sorts them. The `local` setting groups a path prefix as local, mirroring `goimports -local`. Most editors run this on save; disable per-language if that is unwanted.
- **Rename** (`textDocument/rename`, CLI `gopls rename`) — two-stage: `prepareRename` returns the current name, then `rename` rewrites every occurrence. Refuses renames that would introduce shadowing or break interface satisfaction. Special positions unlock extra behavior:
  - Rename a **method's receiver declaration** → renames the receiver across every method of that type; rename a receiver **use** → only that one variable.
  - Rename the **package name in the `package` clause** → moves every file in the package to a new directory (subpackages stay until `renameMovesSubpackages`); refused across module boundaries or into an existing package.
  - Rename the **`func` keyword** of a declaration → edit the whole signature, with parameter/result count and types unchanged (adding/removing parameters is the job of `refactor.rewrite.removeUnusedParam`/`moveParamLeft`/`moveParamRight`).
- **Extract** (`refactor.extract.*`) — replaces a selection with a reference to a new declaration:
  - `refactor.extract.variable` / `.constant` — one new local binding for the selected expression; the `-all` variants rewrite every occurrence within the enclosing function.
  - `refactor.extract.function` / `.method` — turns one or more complete statements into a call to a new function (or method on the same receiver when the extraction happens inside a method).
  - `refactor.extract.toNewFile` (gopls ≥ v0.17.0) — moves selected top-level declarations into a new file, adding imports as needed; the filename derives from the first declared symbol.
- **Inline** (`refactor.inline.*`):
  - `refactor.inline.call` — replaces a call with the function body, substituting parameters for arguments. Works only for static calls to accessible functions/methods (not through a function value or interface method, not to unexported names outside the package, not into `internal` packages, not generic). Preserves side-effect ordering (introducing `var`s when an argument must not be duplicated or reordered), keeps qualified references correct (`Printf` → `fmt.Printf` with the import added), keeps implicit conversions explicit, and never drops a variable's last use. `defer` bodies stay closure-wrapped since defer semantics bind to function boundaries.
  - `refactor.inline.variable` — replaces a local variable's use with its initializer; refuses when an identifier in that initializer has been shadowed since the declaration.
- **Miscellaneous rewrites** (`refactor.rewrite.*`):
  - `removeUnusedParam` — the `unusedparams` analyzer offers renaming to `_` or a full signature change that also updates every caller, preserving side-effecting arguments.
  - `moveParamLeft` / `moveParamRight` — reorders one parameter, updating every call site.
  - `changeQuote` — toggles a string literal between raw (`` `...` ``) and interpreted (`"..."`) form; idempotent, apply twice safely.
  - `invertIf` — negates a plain `if`/`else` condition (no `else if` chains) and swaps the two blocks.
  - `splitLines` / `joinLines` — expands or collapses a bracketed list (composite literal, call arguments, signature) to one item per line; skipped for lists that already contain `//` comments or have fewer than two items.
  - `fillStruct` — populates missing struct-literal fields, matching field names to in-scope variables/constants/functions when possible, zero value otherwise. Searches only the current file, above the cursor — run `source.organizeImports` first if the type was just introduced.
  - `fillSwitch` — adds missing cases for an enum-like set of named constants, or for a type switch (one case per concrete type implementing the interface, plus a default that panics on an unexpected type).
  - `eliminateDotImport` — removes a dot import and qualifies every reference, offered only when no name collision would result.
  - `addTags` / `removeTags` — adds or removes struct field tags (e.g. `json`); interactive clients can pick the naming transform (`camelCase`, `snake_case`, `lisp-case`, `PascalCase`, `Title Case`).
  - `implementInterface` — adds placeholder method declarations so a named type satisfies a chosen interface (defaults to `error`); interactive-dialog only, gopls-specific.
- **Add Test For Function** (`source.addTest`) — generates a table-driven test for the selected function/method, creating the `_test.go` file when needed (copying copyright/build-constraint comments), using an external `p_test` package to encourage testing exported API only, naming results `got`/`got2`/… against `want`/`want2`/…, and adding a `wantErr bool` field when the final result is `error`. For a method it searches the package for a constructor (preferring `NewT` for type `T`). A leading `context.Context` parameter gets `t.Context()` on Go 1.24+, `context.Background()` otherwise.

Gotcha shared by the Extract family: it is less rigorous than Rename/Inline — comments are sometimes dropped, and files carrying a `DO NOT EDIT` generated-code marker receive no code actions at all.

## Web-based features

For reports too rich for inline editor UI, gopls runs a small localhost web server (LSP `window/showDocument`). Every endpoint URL embeds a random auth token; restarting gopls invalidates open pages and shows a disconnected banner on them.

- **Package Documentation** (`source.doc`) — a pkgsite-style rendered view of a package's docs, including **internal, unpublished packages** pkg.go.dev never sees. Symbol links jump the editor to the source declaration; reload without saving to see current edits reflected.
- **Free Symbols** (`source.freesymbols`) — the symbols a selection references but does not define, grouped as imported (with doc links), local, or package-level — exactly the input list an Extract Function/Method refactor would need.
- **Assembly** (`source.assembly`) — the compiled assembly listing for a function, source-line-linked, recompiled on each reload; architecture follows the file's build tags (e.g. `foo_amd64.go`). Not yet supported for generic functions, `func init`, or functions in test packages.
- **Split Package** (`source.splitPackage`) — an interactive dependency-graph tool for planning a package split into smaller, acyclic components. It visualizes the split but does not yet perform the code movement itself.

All of these send edits/navigation back to the editor via `showDocument`, which works even against modified-but-unsaved source.

## Non-Go files

- **Templates** (`text/template`/`html/template`) — off until `templateExtensions` lists at least one extension (templates have no canonical extension); the editor must also associate that extension with the `tmpl`/`gotmpl` language ID (e.g. VS Code's `files.associations`). Inside `{{ }}`: diagnostics (parse errors; missing functions not flagged), full syntax highlighting, definitions and references (all templates share one global scope), and completions. Hover, semantic tokens, symbol search, and document highlight are not yet implemented, and custom delimiters other than `{{`/`}}` are not understood.
- **go.mod / go.work** — hover, hints, vulncheck-driven diagnostics, and code lenses (add dependency, upgrade dependency, tidy, run `govulncheck`) are supported; upstream still marks the per-item behavior as under documentation, so verify current behavior against a real `go.mod` in an editor rather than trusting an exhaustive list here.
- **Assembly (`.s`) files** — basic support exists; treat as best-effort.

## Completion

Upstream documentation for this feature is a stub as of this writing (tracked as [golang/go#62022](https://github.com/golang/go/issues/62022)) — rely on empirical behavior plus these known settings rather than a documented spec: `usePlaceholders` (fills placeholder parameter names on completion), `completeFunctionCalls` (adds trailing parentheses, on by default), `completeUnimported` and matcher/deep-completion-style settings shape whether not-yet-imported packages and nested field/method completions are offered. See [settings.md](settings.md) for the full settings surface.
