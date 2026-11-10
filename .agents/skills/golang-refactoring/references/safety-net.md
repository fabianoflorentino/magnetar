# The Coverage-Adaptive Safety Net

How much caution a refactor needs is not a fixed policy — it is gated on how well tested the *blast radius* is, not the project's global coverage number. A 90%-covered codebase can have the one function you are about to touch at 0%, and a 30%-covered one can have your function fully pinned by table-driven tests. Measure the code you will actually change, then pick the tier below.

## The three tiers

| Tier | Blast-radius coverage | Strategy | Allowed transforms |
| --- | --- | --- | --- |
| **High** | ≥80% function coverage | Refactor first, verify after each step | gopls Rename/Inline/Extract, `eg`/`gofmt -r`, generated fixers |
| **Medium** | ~40-80% | Harden the touched paths first, confirm green, then refactor | Same as High, once targeted tests exist and `-coverprofile` proves they hit the touched lines |
| **Low/Zero** | <40%, or the touched lines specifically | Characterization first (Feathers mode), introduce a seam, refactor last | gopls Rename/Inline only; Sprout/Wrap for new behavior; no Extract or cross-package move until a net exists |

**High — refactor with tools and trust the green bar.** Prefer gopls Rename/Inline/Extract, `eg`/`gofmt -r` for bulk work, and generated fixers over hand-edits. Run the fast net (build/vet/test) after every step and escalate to `-race`/`-bench` only for concurrency or hot paths. Larger steps are fine here: a well-covered blast radius catches a regression within one test run, so being wrong is cheap and immediate.

**Medium — harden before touching.** Measure exactly what the change touches with gopls references and call hierarchy — the real call graph, not the one you remember. Add targeted table-driven or golden tests covering those paths, confirm they pass against current code, then refactor. The step that is easy to skip and should not be: re-run with `-coverprofile` and check the new tests actually reach the lines you will edit. A test that imports the right package but never reaches the branch you are changing is a safety net that looks solid from the outside and catches nothing.

**Low/Zero — Feathers mode.** Write characterization (golden/pinning) tests *first*, capturing what the code does today, warts and all, before any edit. This is deliberately not a correctness test — you are recording current behavior so a refactor can be checked against it. Introduce the minimum seam needed to make the code testable (below), restrict yourself to behavior-preserving-by-construction transforms (gopls Rename and Inline), and prefer Sprout/Wrap for new behavior. Running `deadcode -test` first is worth two minutes: some "untested code" is exported API nothing calls, where the honest fix is deletion, not testing.

## Measuring the blast radius

```bash
go test -covermode=atomic -coverpkg=./... -coverprofile=cover.out ./...
go tool cover -func=cover.out     # every function ranked by coverage — read the touched functions, not the package average
go tool cover -html=cover.out     # visual red/green view of the exact touched lines
```

Two caveats before trusting the numbers:

- **Go's coverage is statement coverage, not branch coverage.** A line inside an `if` that ran once counts as fully covered even if the `else` never ran or a `switch` hit one `case`. A function reporting 100% can still have an untested branch — read the actual branches in the code you are touching rather than trusting the summary.
- **`go test ./...` silently drops any package with no `_test.go` file.** It is not counted as 0%; it is absent from the report entirely, making an untested package invisible instead of visibly red. `-coverpkg=./...` forces every package in the module into the profile so a silently untested dependency cannot slip past the tier decision.

## Seams for code with no net yet

A seam — in Michael Feathers's sense — is a place where you can alter behavior without editing that exact spot. Two types matter in Go:

- **Object seam:** an interface, or a function-typed field/parameter, injected at the point of construction. A test substitutes a fake through that injection point instead of exercising the real dependency. This is the seam type that matters most in Go: interfaces are satisfied implicitly, so introducing one at the point of use requires touching only the consumer, never the producer.
- **Link/build-tag seam:** swaps an entire implementation at build time via `//go:build` constraints. Used rarely, mostly for platform- or environment-specific substitutions where an interface would be overkill.

The enabling move for untested code with no seam: extract the smallest possible interface — often one method — exactly where the untested code depends on something external (a database client, the filesystem, a clock), and inject the concrete implementation through a constructor parameter instead of constructing it inline:

```go
// Before — NewReport builds its own client, so a test of Generate
// cannot substitute a fake and is stuck hitting a real database.
func NewReport(dsn string) *Report {
    db, _ := sql.Open("postgres", dsn)
    return &Report{db: db}
}

// After — a one-method interface extracted at the point of use.
// The concrete *sql.DB satisfies it implicitly, so the producer
// package needs no change at all.
type rowQuerier interface {
    QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func NewReport(db rowQuerier) *Report {
    return &Report{db: db}
}
```

This single move does two things at once: it breaks a potential import cycle between the consumer and the concrete type it depended on, and it opens the door for a fake in a characterization test. → See `fabianoflorentino/golang-agent-skills@golang-design-patterns` skill for the constructor/DI patterns this builds on, and [catalog.md](catalog.md) for the Sprout/Wrap mechanics that typically pair with a freshly introduced seam.

## Verification command reference

The fast net from the Core Loop in [SKILL.md](../SKILL.md), escalated only as far as the change requires:

```bash
go build ./...                              # fastest gate — compile errors
go vet ./...                                # correctness checks the compiler does not do
go test ./...                               # full test suite
go test -run TestName ./pkg/...             # one test while iterating
go test -race ./...                        # concurrency changes — see golang-testing for race-detector mechanics
go test -covermode=atomic -coverpkg=./... -coverprofile=cover.out ./...
go tool cover -func=cover.out               # per-function and total coverage
go tool cover -html=cover.out               # red/green source view
go test -bench=. -benchmem -count=10 > new.txt
benchstat old.txt new.txt                   # `~` = no significant difference — the desired result for a behavior-preserving refactor
```

A `benchstat` result other than `~` on a hot path is a behavior change to investigate, not noise to shrug off. → See `fabianoflorentino/golang-agent-skills@golang-testing` for test-writing craft and race-detector mechanics, and `fabianoflorentino/golang-agent-skills@golang-benchmark` for interpreting the delta.

## Cross-references

- [catalog.md](catalog.md) — the Sprout/Wrap entries referenced in Feathers mode.
- [workflow.md](workflow.md) — the planning gate that measures the blast radius these tiers depend on, and the human-checkpoint rule for untested code.
