# Refactoring Workflow: Plan, Stage, Land

A refactor of any real size is an ordering problem before it is a coding problem. Getting the sequence wrong does not fail loudly at plan time — it surfaces later as broken builds and rebase conflicts once several PRs are already moving. This reference covers the planning gate, how to order the inventory, the git model that stages it, and how the plan survives context loss.

## 1. The planning gate

Map the blast radius with gopls before a single edit:

- List every reference to the symbols you will touch.
- Walk the call hierarchy in both directions.
- Audit the package's exported API for anything an external module could depend on.

Mechanical know-how (symbol search, references, call hierarchy, code actions) belongs to `fabianoflorentino/golang-agent-skills@golang-gopls`; this file assumes it is available.

Turn the blast radius into a **refactoring inventory** — one row per atomic change, visible as a single artifact before any PR exists:

| Transform | Files / callers | Risk | S/B |
| --- | --- | --- | --- |
| Extract `validateOrder` from `ProcessOrder` | `internal/orders/process.go`, no external callers | Low | S |
| Rename `Client.Send` → `Client.Publish` | `pkg/client/*.go`, 14 sites across 3 packages | Low | S |
| Break `billing` ↔ `orders` cycle with a consumer interface | `internal/billing/service.go`, `internal/orders/service.go` | High | S |
| Move `Invoice` to `pkg/billing`, alias the old location | `internal/orders/invoice.go`, ~9 call sites | High | S |
| Replace `Invoice.Total`'s O(n²) discount lookup with a map | `pkg/billing/invoice.go` | Medium | S |
| Switch `Invoice.Total` from float64 to Decimal | `pkg/billing/invoice.go` and tests | Medium | B |

Rules that make the inventory useful:

- **Risk tiers follow the Risk Stratification table in SKILL.md** (Low/Medium/High).
- **The S/B column is Kent Beck's split** — shape-only change versus behavior change. A PR must never carry both letters: a rename and a bug fix in the same function are two rows, two PRs, two review postures.
- **One row per concern even inside a letter.** The move and the loop-optimization rows above are both S, yet they stay separate: a move is verified by gopls plus a green build/test, while an optimization needs `benchstat` and a closer correctness read. Bundling them forces a reviewer to do both jobs at once, and they touch the same file, so ordering rule (b) below sequences them anyway.

**The gate ends with explicit user sign-off before code moves.** Present the inventory and the staged PR plan derived from it; wait for approval. A wrong assumption that surfaces after several PRs are stacked is expensive to unwind.

## 2. Ordering the inventory

Three independent ordering concerns combine into the final sequence. Ignoring any one of them sinks the plan.

| Ordering | Question it answers | Effect |
| --- | --- | --- |
| (a) Beck ordering | Within a dependency chain, does a row change structure or behavior? | Structural first, behavioral last. `git blame` stays meaningful, and reviewers wear one hat per PR. |
| (b) Conflict avoidance | Do two rows touch the same files or symbols/callers? | Sharing rows land sequentially on the branch; disjoint rows may run in parallel worktrees. |
| (c) Dependency ordering | Does a row need structural groundwork another row lands? | Cycle breaks, package extractions, and aliases are prerequisites, not peers — land them first. |

**A workspace-wide gopls rename is a barrier.** It rewrites every reference in the tree and can touch any file any other in-flight change touches. Schedule it alone: land everything else first, or hold everything else until it lands. Never run it concurrently with anything, no matter how unrelated the files look.

### Parallel-or-sequential checklist

Run this on every pair of rows you consider running at once:

| Question | If yes |
| --- | --- |
| Same file? | Sequential |
| Same symbol, or overlapping callers? | Sequential |
| One depends on groundwork the other lands? | Sequential — groundwork first |
| Either is a workspace-wide rename? | Sequential — it runs alone |
| None of the above | Parallel in separate worktrees |

## 3. The git model

This is a deliberate choice for *staged* refactors, not the only way to work: teams landing independent changes often merge straight to a fast-moving trunk. A staged refactor is different — it is one coherent transformation cut into reviewable slices, and it needs a place to accumulate before the whole is ready for `main`.

1. Cut a long-lived `refactor/<topic>` branch off `main` and seed it with `// REFACTOR(step N): ...` markers for the plan (see step 5).
2. For each inventory row, in the order from step 2, dispatch the change to a **sub-agent** rather than executing it in the orchestrating session. The sub-agent enters a fresh worktree, branches off the current tip of `refactor/<topic>`, and applies exactly one atomic change — if the diff threatens to exceed roughly 100-500 lines, split the row. It then verifies with `go build ./... && go vet ./... && go test ./...` (add `-race` or `-bench` per the risk tier), runs the same checks locally if CI is slow enough to stall the sequence, and opens a PR:

   ```bash
   gh pr create --base refactor/<topic> --title "..." --body "..."
   ```

   The PR targets the refactoring branch, not `main`, and is ready for review, not a draft — the sub-agent reports back only a short result (pass/fail, verification output, PR link), keeping the orchestrating session's context free for the rows still ahead.
3. A human reviews and merges each small PR at their own pace. Structural PRs move fast; behavioral PRs get full scrutiny. Any PR that changes logic rather than shape is also reviewed with `fabianoflorentino/golang-agent-skills@golang-security` (and `golang-safety`) loaded.
4. When every row has landed and the marker sweep in step 5 is clean, open the **final PR** merging `refactor/<topic>` into `main` — as a draft, because it packages the whole transformation and deserves a slow look.

**Never merge an intermediate PR to `main`.** While the refactor runs, `refactor/<topic>` is the only integration point; `main` sees the completed transformation once. An early PR landing on `main` exposes deliberately incomplete state (aliases still present, shims not yet removed) to every branch built on top of it.

## 4. Running the steps

- **Parallel when file-disjoint.** Launch ready rows concurrently, one sub-agent per row, each in its own worktree with its own branch and PR. Three disjoint structural changes reviewed at once cost the same calendar time as one.
- **Sequential when overlapping.** Land the first row on `refactor/<topic>` before branching the second off the new tip. Parallelizing overlapping rows only moves the conflict from merge time to rebase time, and a human reviewer ends up untangling a mixed diff.
- **The rename barrier holds in execution too:** never schedule a tree-wide rename beside any other in-flight worktree.

## 5. The marker convention

`// REFACTOR(step N): ...` comments do two jobs:

- **They survive context loss.** A staged refactor spans sessions, and each fresh session starts with no memory of the planning conversation. The codebase, committed to `refactor/<topic>`, does not forget. Seed markers liberally at every point the inventory identifies future work or a non-obvious decision — but skip the seeding for small refactors, where a single PR or one mechanical sweep has no plan large enough to lose.
- **They flag deliberate imperfection.** A staged refactor passes through intermediate states that are imperfect on purpose — a kept-alias, a shim, a still-reachable old path. The risk is not the imperfection but forgetting it exists after its introducing PR merged.

Every plan note or deliberate imperfection gets a comment naming the step and the reason:

```go
// REFACTOR(step 3): remove this alias once all callers in pkg/foo migrate to bar.New (see refactor/<topic>)
```

Run the **final sweep** just before opening the merge PR to `main` — it must return zero hits, because any hit means a planned step never landed:

```bash
grep -rn "REFACTOR(" .
```

Without the sweep, a "temporary" shim has a way of becoming permanent simply because nothing points back at it, and a planning-gate idea vanishes the moment the session holding it ends.

## 6. Workflows vs. human-in-the-loop

Claude Code's Workflows (`ultracode`) orchestrate many sub-agents with no human checkpoint between stages — exactly the wrong shape for a staged refactor, whose value is a human merging each small PR before the next step builds on it. Running a multi-step refactor through Workflows collapses the very checkpoints this file exists to preserve.

`ultracode` is appropriate only for a **single mechanical sweep in one pass** — one `gofmt -r` rule or one `eg` template applied tree-wide and verified green, with nothing else depending on it. There is no staging problem when there is only one step. Anything requiring progressive review across merges stays on the worktree + PR + review flow.

## 7. Human checkpoints

The sign-off triggers from SKILL.md — cross-package moves, exported-API changes, deletions, new major versions, untested code — are not one-time events. Re-obtain sign-off wherever one comes up mid-refactor, even after the planning gate cleared it once. For untested code that means approving the characterization-test baseline (see [safety-net.md](safety-net.md)) *before* refactoring it, not after.

Structural-only PRs are reversible and low-risk by construction and merit a fast review. Behavioral PRs get full scrutiny every time, no matter how small the diff.

## Cross-references

- [catalog.md](catalog.md) — the Fowler catalog mapped to Go: smell trigger, mechanics, tool, and risk per entry.
- [go-tooling.md](go-tooling.md) — the mechanical tools the inventory rows lean on, from gopls down to bespoke `go/analysis` fixers.
- [safety-net.md](safety-net.md) — the coverage-adaptive strategy that determines how much net a row needs before it runs.
- [structural.md](structural.md) — cycle breaking, package boundaries, and the type-alias gradual-repair mechanism behind the cross-package rows.
