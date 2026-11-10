# The Fowler Catalog, Mapped to Go

A refactoring is a named, behavior-preserving transform with a known smell that triggers it and a known mechanics for carrying it out. This file maps the entries of Fowler's catalog to Go: what shape of code signals each one, what the transform looks like in Go, which tool performs it, and how risky it is (matching the Risk Stratification table in [SKILL.md](../SKILL.md)).

## Quick reference

| Refactoring | Smell trigger | Primary tool | Risk |
| --- | --- | --- | --- |
| Extract Function/Method | Long Function | gopls `refactor.extract.function`/`method` | Medium |
| Inline Function/Call | Middle Man, one-line forwarder | gopls `refactor.inline.call` | Low |
| Extract Variable/Constant, Inline Variable | repeated or opaque expression | gopls `refactor.extract.variable`/`constant`, `refactor.inline.variable` | Low |
| Rename | misleading identifier | gopls `rename` | Low |
| Change Function Declaration | Long Parameter List, stale parameter | gopls `refactor.rewrite.*`, `eg` for adds | Medium/High |
| Move Function/Field/Type | Feature Envy, wrong package | manual alias/wrapper sequence | High |
| Split/Merge Package | God package, Divergent Change | gopls `source.splitPackage` (experimental) | High |
| Guard Clauses | nested preconditions | gopls `refactor.rewrite.invertIf` + hand pass | Low |
| Introduce Parameter Object | Data Clumps | manual (golang/go#65552 pending) | Medium |
| Replace Conditional with Polymorphism | Repeated Switches | manual | Medium |
| Hide Delegate / Remove Middle Man | Message Chains / pass-through method | struct embedding + `refactor.inline.call` | Low-Medium |
| Sprout/Wrap Method | new behavior for untested code | gopls `rename` + hand-written new code | Low |
| Replace Temp with Query | derived local hiding a computation | manual | Low |

## Extract Function / Extract Method

**Trigger:** Long Function — a body mixing several levels of abstraction, or a comment introducing a block that ought to be the block's name.

**Go mechanics:** gopls selects the statements, infers the parameter list from free variables and the return list from values used after the block, and infers a receiver for the method variant.

```bash
gopls codeaction -exec -kind=refactor.extract.function file.go:#start,#end
gopls codeaction -exec -kind=refactor.extract.method   file.go:#start,#end
```

**Risk:** Medium. gopls's own docs treat Extract as "considerably less rigorous" than Rename or Inline, and it is known to drop comments attached to the extracted statements (golang/go#20744). Diff the result and reread the extracted body — it is not behavior-preserving by construction.

## Inline Function / Inline Call

**Trigger:** a trivial forwarding function, or a Middle Man that has accreted no behavior of its own.

**Go mechanics:** the call site is replaced with the callee body using real substitution, not text splicing: side-effecting arguments are hoisted into `var` temporaries, implicit conversions at the boundary become explicit, and a `defer` in the callee is wrapped in an immediately-invoked function literal so it still fires at the right point. Inline cannot cross a dynamic dispatch (interface method, function value) or inline a generic function.

```bash
gopls codeaction -exec -kind=refactor.inline.call file.go:#offset
```

**Risk:** Low. Alongside Rename, this is provably behavior-preserving by construction — it refuses rather than produce a semantically wrong inline.

## Extract Variable / Extract Constant / Inline Variable

**Trigger:** the same nontrivial expression appears more than once, or a single occurrence is opaque enough to demand re-derivation on every read.

**Go mechanics:** Extract Variable introduces a `:=` immediately above the first use; the `-all` variant rewrites every syntactic occurrence in scope. Extract Constant names a literal that deserves compile-time immutability. Inline Variable is the reverse — substitute at the use sites and remove the binding.

```bash
gopls codeaction -exec -kind=refactor.extract.variable     file.go:#start,#end
gopls codeaction -exec -kind=refactor.extract.variable-all file.go:#start,#end
gopls codeaction -exec -kind=refactor.extract.constant     file.go:#start,#end
gopls codeaction -exec -kind=refactor.inline.variable      file.go:#offset
```

**Risk:** Low.

## Rename

**Trigger:** any identifier whose name no longer says what it holds or does. A wrong name cascades into every call site that reads it, so fixing it is often the first step of a larger refactor.

**Go mechanics:** gopls Rename is workspace-wide — it updates every reference across every importing package and type-checks the result. It refuses rather than proceed on a shadowing conflict, on a rename that breaks interface satisfaction elsewhere, or when the surrounding code does not type-check. Two cases trip people up:

- Renaming the `p` in `package p` moves the directory and rewrites every import path that referred to it.
- Renaming a method's *receiver declaration* (`s` in `func (s *Store) Get(...)`) propagates to every method of the type; renaming one *use* of the receiver inside a method body touches only that method.

```bash
gopls rename file.go:#offset newName
```

Editor-integrated as "Rename Symbol" (F2 in most gopls-backed editors).

**Risk:** Low — but treat a refusal as a real semantic hazard to investigate, not friction to hand-edit around.

→ See `fabianoflorentino/golang-agent-skills@golang-naming` skill for what to rename identifiers *to*; this entry covers only how to apply the rename safely at scale.

## Change Function Declaration

**Trigger:** a parameter list grown past what the function needs, or a parameter that no longer matches how the function is used.

**Go mechanics:** gopls offers single-purpose actions — remove a parameter nobody passes meaningfully, or swap two adjacent parameters:

```bash
gopls codeaction -exec -kind=refactor.rewrite.removeUnusedParam file.go:#offset
gopls codeaction -exec -kind=refactor.rewrite.moveParamLeft      file.go:#offset
gopls codeaction -exec -kind=refactor.rewrite.moveParamRight     file.go:#offset
```

Adding a parameter across every call site is not a single gopls action today. For that, stage it with `eg`: declare a new variant with the added or generalized parameter, migrate call sites via an `eg` template, verify, then Rename or delete the old function once unreferenced:

```bash
eg -t template.go -w ./...
```

**Risk:** Medium for a single-parameter add/remove over a handful of call sites; High when a signature changes across many callers with no one-shot mechanical action.

→ See `fabianoflorentino/golang-agent-skills@golang-design-patterns` skill for converting a long parameter list into an options struct instead of reordering it.

## Move Function / Move Field / Move Type

**Trigger:** Feature Envy — a function using another package's data more than its own — or a type whose responsibilities belong to a different package.

**Go mechanics:** no one-shot gopls action moves a symbol across package boundaries. The gradual-repair sequence: introduce the symbol in its new home; leave a **type alias** (`type Old = pkg.New`) for a type — or a thin wrapper/forwarding variable for a function — in the old location so both names keep working; migrate callers incrementally, one small commit at a time; delete the old name once nothing references it. Splitting a large file within the *same* package is a distinct one-shot action:

```bash
gopls codeaction -exec -kind=refactor.extract.toNewFile file.go:#start,#end
```

**Risk:** High. The full alias recipe lives in [structural.md](structural.md).

→ See `fabianoflorentino/golang-agent-skills@golang-project-layout` skill for the target package layout.

## Split Package / Merge Package

**Trigger:** a "god package" — the package-level analogue of a Large Class — or Divergent Change, where one package keeps changing for unrelated reasons.

**Go mechanics:** gopls's experimental `source.splitPackage` partitions top-level declarations into acyclic components. Treat its proposal as a draft to review, not a final answer — package boundaries encode API and ownership decisions the tool cannot see. Merging has no tool: move declarations into the target package and break whatever cycle results with a consumer-side interface first.

```bash
gopls codeaction -exec -kind=source.splitPackage file.go
```

**Risk:** High.

## Replace Nested Conditional with Guard Clauses

**Trigger:** an `if`/`else` chain or nested `if` blocks where most branches are preconditions and error cases rather than equal-weight alternatives.

**Go mechanics:** each non-primary branch becomes an early `return`/`continue`/`break`, leaving the main path at a single nesting level. gopls's invert-if action flips one condition in place and serves as the mechanical building block; flattening a whole chain is a structural pass assisted by that action, not a single invocation:

```bash
gopls codeaction -exec -kind=refactor.rewrite.invertIf file.go:#offset
```

**Risk:** Low.

→ See `fabianoflorentino/golang-agent-skills@golang-code-style` skill for the early-return style rule this works toward.

## Introduce Parameter Object

**Trigger:** Data Clumps — the same parameter cluster recurring across several functions — or a Long Parameter List where several parameters are conceptually one unit.

**Go mechanics:** define a struct holding the recurring group, then change each affected signature to take the struct. gopls's planned "Extract parameter struct" action (golang/go#65552) is not generally available as of this writing, so treat this as a manual Extract-Variable-style step (define the struct, construct it at each call site) followed by the signature-change tooling above.

**Risk:** Medium.

→ See `fabianoflorentino/golang-agent-skills@golang-design-patterns` skill for functional options as the alternative when the struct exists to configure construction rather than to group a data clump.

## Replace Conditional with Polymorphism

**Trigger:** Repeated Switches — the same `switch` on a type tag or state constant recurs at multiple call sites, so every new case must be added in lock-step at each one.

**Go mechanics:** define an interface with one method per varying behavior and give each case its own implementing type. Callers that used to switch on the tag call the interface method, and dispatch happens through Go's interface mechanism. Manual — carve out each case's behavior with Extract steps, then clean up seams with Rename/Inline.

**Risk:** Medium.

→ See `fabianoflorentino/golang-agent-skills@golang-design-patterns` skill for the target interface-dispatch pattern.

## Hide Delegate / Remove Middle Man

**Trigger:** Message Chains (`a.B().C().D()` reaching through several objects) call for Hide Delegate; a method whose body is nothing but `return x.SameMethod(...)` calls for Remove Middle Man.

**Go mechanics:** **struct embedding** is Go's usual Hide Delegate — embedding promotes the delegate's methods onto the containing type, so callers invoke them directly on the outer struct. Remove Middle Man is the inverse: delete the pass-through method after an Inline Call at each call site, or a Rename-and-redirect, and let callers reach the real implementation.

**Risk:** Low-Medium. The embedding decision is manual; `refactor.inline.call` mechanizes deleting the middle man once callers are ready to call through directly.

## Sprout Method / Wrap Method

**Trigger:** new behavior to add to a function or method with little or no test coverage.

**Go mechanics:** the Feathers move for untested code — add behavior without editing the untested code in place:

- **Sprout Method** writes the new behavior as a brand-new, fully tested function and calls it from the one site that needs it, leaving the original untouched.
- **Wrap Method** renames the original (`Save` → `saveInternal`), then adds a method with the old name that calls the renamed original and adds the new behavior around it — a manual decorator, since Go has no method-wrapping mechanism.

**Risk:** Low — by design it never touches the untested code directly. Use gopls `rename` for the rename step; the new code is hand-written and tested like any new function.

→ See [safety-net.md](safety-net.md) for when low/zero coverage should push you toward this pattern instead of editing in place.

## Replace Temp with Query

**Trigger:** a local assigned once from an expression and read several times later, whose name does not reveal it is derived rather than an input.

**Go mechanics:** replace the variable with a small unexported function recomputing the expression; each read becomes a call. Manual — Go's call cost makes this purely a readability call. If the function ends up single-use, gopls's inline action can fold it back later without losing the readability gain.

**Risk:** Low.

## Smell → refactoring quick reference

| Smell | Go-specific fix |
| --- | --- |
| Long Function | Extract Function/Method |
| God package | Split Package |
| Long Parameter List | Introduce Parameter Object, or an options struct |
| Data Clumps | Extract a struct for the recurring group |
| Primitive Obsession | Named type instead of a bare `string`/`int` |
| Divergent Change (one package, many reasons) | Split Package |
| Shotgun Surgery (one change ripples through many packages) | Move Function/Field to consolidate the concept into one package |
| Feature Envy | Move Function to the package whose data it actually uses |
| Repeated Switches | Replace Conditional with Polymorphism |
| Message Chains | Hide Delegate |
| Middle Man | Remove Middle Man |

Divergent Change and Shotgun Surgery are opposite fixes for the same symptom. Divergent Change is one package changing for many reasons — split it apart. Shotgun Surgery is one reason rippling across many packages — consolidate that concept into one package so a single change touches one place.

## Cross-references

- [structural.md](structural.md) — the type-alias gradual-repair recipe and cycle breaking that Move and Split/Merge build on.
- [safety-net.md](safety-net.md) — the coverage-adaptive strategy that decides when Sprout/Wrap replaces an in-place edit.
- [go-tooling.md](go-tooling.md) — full invocation details for every tool named above.
- `fabianoflorentino/golang-agent-skills@golang-naming` — what to rename identifiers to.
- `fabianoflorentino/golang-agent-skills@golang-project-layout` — where code belongs after a move.
- `fabianoflorentino/golang-agent-skills@golang-design-patterns` — options structs, consumer interfaces, interface-dispatch patterns.
- `fabianoflorentino/golang-agent-skills@golang-code-style` — the guard-clause/early-return style rule.
