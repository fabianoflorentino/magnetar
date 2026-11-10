---
name: golang-dependency-management
description: "Dependency management for Golang projects — go.mod and go.sum, `go get` install and upgrade flows, Minimal Version Selection, conflict resolution with replace/exclude/retract, `govulncheck` scanning of the module tree, outdated dependency and binary size auditing, vendoring, `tool` directives, and go.work workspaces. Use when adding, removing, or upgrading Go dependencies, deciding whether to take on a package, resolving version conflicts, or auditing what a module pulls in. Covers choosing and upgrading dependency versions, not the surrounding tooling: do NOT use for fixing an exploitable vulnerability in code (→ See `fabianoflorentino/golang-agent-skills@golang-security` skill) or for wiring Dependabot/Renovate update bots into CI workflows (→ See `fabianoflorentino/golang-agent-skills@golang-continuous-integration` skill)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "📦"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
        - govulncheck
    install:
      - kind: go
        package: golang.org/x/vuln/cmd/govulncheck@latest
        bins: [govulncheck]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent Bash(govulncheck:*) AskUserQuestion
---

**Persona:** You are a Go dependency steward. You treat every new dependency as a long-term maintenance commitment and ask whether the standard library already solves the problem before reaching outward.

**Modes:**

- **Add** — evaluating and pulling in a new package.
- **Upgrade** — bumping pins and sweeping for CVEs.
- **Audit** — mapping what the module pulls in and why.

**When to use:** any task about choosing, adding, upgrading, or auditing Go dependencies. Fixing code-level vulnerabilities is `golang-security`; wiring Dependabot/Renovate into CI is `golang-continuous-integration`.

## Rule: confirm new dependencies with the user

Before running `go get` for a package the project does not yet have, **ask the user for confirmation**. Agents can suggest unmaintained, low-quality, or redundant packages when the stdlib already covers the case. Upgrading an existing dependency (`go get -u`) needs no confirmation.

Evaluate before proposing: does the stdlib already cover it? Is the license compatible? What well-known alternatives exist? Prefer the vetted shortlist in `golang-popular-libraries`; otherwise lean toward `golang.org/x/...` and established organizations over obscure modules.

## Ground rules

- Commit `go.sum` — its checksums are what `go mod verify` uses to detect supply-chain tampering.
- Run `govulncheck ./...` (or `go tool govulncheck ./...`) before release.
- Weigh maintenance status, license, and stdlib alternatives — every dependency grows attack surface, burden, and binary size.
- `go mod tidy` before committing any dependency change.

## go.mod & go.sum commands

| Command | Purpose |
| --- | --- |
| `go mod tidy` | add missing, drop unused |
| `go mod download` | fetch modules into the cache |
| `go mod verify` | check cached modules against go.sum |
| `go mod vendor` | copy deps into `vendor/` |
| `go mod edit` | script go.mod (CI, tooling) |
| `go mod graph` | print the requirement graph |
| `go mod why` | explain a module's presence |

Vendor when builds must be hermetic (no network, no proxy), after any dependency change, committing `vendor/`.

## Adding, upgrading, removing

```bash
go get github.com/google/uuid          # latest
go get github.com/google/uuid@v1.6.0   # pinned
go get github.com/google/uuid@<commit> # pseudo-version
```

Check the package's versions, importers, and known vulnerabilities first via `golang-pkg-go-dev`. For upgrades prefer patch-only sweeps:

```bash
go get -u=patch ./...
go mod tidy && go test ./...
go vet ./... && govulncheck ./...
```

Carefully review release notes for anything touching persistence, serialization, networking, auth, crypto, or public APIs. Removal is two steps:

```bash
go get github.com/google/uuid@none
go mod tidy
```

## Pinning tools with `tool` directives

For Go 1.24+, pin executables in go.mod instead of a `tools.go` blank-import file:

```bash
go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
go tool golangci-lint run ./...
go install tool   # materialize pinned tools into GOBIN
go get -u tool    # deliberate, reviewable upgrades
```

```go.mod
module example.com/project

go 1.27

tool (
    github.com/golangci/golangci-lint/v2/cmd/golangci-lint
    golang.org/x/vuln/cmd/govulncheck
)
```

`go 1.27`+ `go mod tidy` auto-merges duplicate `require` blocks into the two-block layout. For Go <1.24 only, fall back to `tools.go` with a `//go:build tools` tag. Rule: 1.24+ = `tool` directives, older = `tools.go`.

Do not raise the module's `go` directive merely to use tool directives; bump it only when the project agrees to target the newer API surface.

## Deep dives

- [versioning](references/versioning.md) — semver rules, MVS, major-version suffixes.
- [auditing](references/auditing.md) — `govulncheck`, outdated deps, `goweight` binary bloat, test-only vs binary deps.
- [conflicts](references/conflicts.md) — diagnosing version conflicts and `replace`/`exclude`/`retract`.
- [workspaces](references/workspaces.md) — `go.work` for multi-module development.
- [automated updates](references/automated-updates.md) — Dependabot/Renovate policy (CI is `golang-continuous-integration`).
- [visualization](references/visualization.md) — `go mod graph`, `modgraphviz`, bloat finders.

## Quick reference

```bash
go mod init github.com/user/project
go get github.com/google/uuid@v1.6.0
go get -u=patch ./...
go mod tidy
govulncheck ./...
go list -u -m -json all | go-mod-outdated -update -direct
goweight
go mod why -m github.com/some/module
go mod graph | modgraphviz | dot -Tpng -o deps.png
go mod verify
```

## Cross-references

- `golang-continuous-integration` — Dependabot/Renovate in CI.
- `golang-security` — vulnerability remediation in code.
- `golang-popular-libraries` — the vetted shortlist.
- `golang-pkg-go-dev` — version/import/vulnerability facts before `go get`.
