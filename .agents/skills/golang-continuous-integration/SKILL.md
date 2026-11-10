---
name: golang-continuous-integration
description: "GitHub Actions CI/CD pipeline configuration for Golang projects — workflow files for test, lint, SAST, coverage and vulnerability-scan jobs, Dependabot and Renovate config files, GoReleaser release pipelines, Docker build/push, repository security settings, and AI-driven PR review. Use when setting up or improving Go project CI, writing or fixing `.github/workflows/*.yml`, adding a linter or security scanner as a pipeline job, wiring automated dependency-update bots, or adding quality gates. Covers wiring tools into a pipeline, not the analysis they perform: do NOT use for choosing or interpreting security findings (→ See `fabianoflorentino/golang-agent-skills@golang-security` skill) or for choosing, upgrading, or auditing dependency versions (→ See `fabianoflorentino/golang-agent-skills@golang-dependency-management` skill)."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🚀"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
        - goreleaser
        - gh
    install:
      - kind: brew
        formula: goreleaser
        bins: [goreleaser]
      - kind: brew
        formula: gh
        bins: [gh]
      - kind: npm
        package: skills
        bins: [skills]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch Bash(goreleaser:*) Bash(gh:*) AskUserQuestion
---

**Persona:** You are a Go DevOps engineer. CI is a quality gate: every decision is weighed against build speed, signal reliability, and security posture.

**Modes:**

- **Setup** — CI from scratch: start from the stage table, then generate workflows in order test → lint → security → release. Prefer the latest stable major version of each GitHub Action.
- **Improve** — audit/extend an existing pipeline: read current workflows first, find gaps against the stage table, add only what's missing.

**Dependencies:**

- goreleaser: `go install github.com/goreleaser/goreleaser/v2@latest`
- gh: `brew install gh`

**When to use:** any task writing or fixing `.github/workflows/*.yml`, or wiring linters, scanners, dependency bots, releases, and review agents into a Go project. Interpreting security findings is `golang-security`; choosing/upgrading dependency versions is `golang-dependency-management`.

## Stage table

| Stage | Tool | Purpose |
| --- | --- | --- |
| **Test** | `go test -race` | unit + race detection |
| **Coverage** | `codecov/codecov-action` | coverage reporting |
| **Lint** | `golangci-lint` | comprehensive linting |
| **Vet** | `go vet` | built-in static analysis |
| **SAST** | `gosec`, CodeQL, Bearer | security static analysis |
| **Vuln scan** | `govulncheck` | known-vulnerability detection |
| **Docker** | `docker/build-push-action` | multi-platform images |
| **Deps** | Dependabot / Renovate | automated dependency updates |
| **Release** | GoReleaser | automated binary releases |
| **AI Review** | Claude Code / Copilot | AI PR review |

Action versions in the ready-made assets are reference snapshots — GitHub Actions release frequently, so verify the current major of every action (`@vN`, never `@master`).

## Test workflow

`.github/workflows/test.yml` — see [test.yml](./assets/test.yml). Adapt the Go version matrix to `go.mod` (current: `go 1.27` → `["1.27","stable"]`; each `go N` row adds that version only). Matrix rules:

- `fail-fast: false` — one Go version failing must not cancel the rest.
- `-race` — CI MUST run tests with race detection.
- `-shuffle=on` — randomize order to catch inter-test dependencies.
- `-coverprofile` — feed coverage upload.
- `go mod tidy && git diff --exit-code` — fail on a dirty go.mod.
- Go 1.27 raises the Darwin floor to macOS 13 (Ventura); `macos-latest`/`macos-14`+ runners are unaffected.

Coverage threshold enforcement lives in `codecov.yml` — see [codecov.yml](./assets/codecov.yml).

Integration tests: [integration.yml](./assets/integration.yml) — use `-count=1` to disable caching, which hides flaky service interactions.

## Lint workflow

`.github/workflows/lint.yml` — see [lint.yml](./assets/lint.yml). `golangci-lint` MUST run on every PR. Create the root `.golangci.yml` from the recommended config in the `golang-lint` skill.

## Security & SAST

`.github/workflows/security.yml` — see [security.yml](./assets/security.yml). `govulncheck` MUST run — it only reports vulnerabilities in code paths the project actually calls, unlike generic CVE scanners. CodeQL results land in the repo's Security tab; Bearer is strong at sensitive-data-flow issues. Extended queries via `.github/codeql/codeql-config.yml` — see [codeql-config.yml](./assets/codeql-config.yml). Query suites: `default` / `security-extended` (more queries, lower precision) / `security-and-quality` (+ maintainability, reliability).

If the project builds images, the Docker workflow includes Trivy container scanning — see [docker.yml](./assets/docker.yml).

## Dependency-update bots

Dependabot: `.github/dependabot.yml` — see [dependabot.yml](./assets/dependabot.yml). Minor/patch updates group into a single PR; majors get individual PRs (breaking changes deserve review).

Auto-merge: [dependabot-auto-merge.yml](./assets/dependabot-auto-merge.yml). The guard `if: github.actor == 'dependabot[bot]'` restricts execution to Dependabot — do not remove it. But `github.actor` checks are not fully spoof-proof: **branch protection is the real safety net**. Configure required status checks and approvals so auto-merge only succeeds after everything passes, regardless of who triggered the run. The workflow needs `contents: write` and `pull-requests: write` — elevated on purpose, contained by the guard.

Renovate (alternative): install the [Renovate GitHub App](https://github.com/apps/renovate), then create `renovate.json` — see [renovate.json](./assets/renovate.json). Advantages over Dependabot: automatic `go mod tidy` after updates, native automerge without a companion workflow, more flexible grouping, regex managers (Dockerfiles, Makefiles), and monorepo/workspace awareness.

## Releases with GoReleaser

Release trigger: [release.yml](./assets/release.yml) on tag pushes `tags: ["v*"]` — with `contents: write` only for release creation, unreachable from PRs and branch pushes.

Config varies by project shape:

- **CLI / programs** — cross-compiled binaries, archives, optional images: [goreleaser-cli.yml](./assets/goreleaser-cli.yml).
- **Libraries** — no binaries; a minimal config that skips the build and just tags a changelog: [goreleaser-lib.yml](./assets/goreleaser-lib.yml). Often `gh release create` is enough.
- **Monorepo / multi-binary** (`cmd/api/`, `cmd/worker/`): [goreleaser-monorepo.yml](./assets/goreleaser-monorepo.yml).

### Docker build & push

[docker.yml](./assets/docker.yml) builds multi-platform images, generates SBOM + provenance, pushes to GHCR and Docker Hub, and runs Trivy. Permissions are scoped per job — the `container-scan` job gets only `contents: read` + `security-events: write`, so it cannot push even if compromised. `push: false` on PRs means untrusted code never publishes images. `DOCKERHUB_USERNAME`/`DOCKERHUB_TOKEN` secrets must be configured — never hardcode credentials. QEMU + Buildx enable `linux/amd64,linux/arm64`; drop platforms you don't need. Provenance/SBOM need `attestations: write` + `id-token: write`. For GHCR-only, drop the Docker Hub login and the `docker.io/` image line.

## Repository security settings

Branch protection, workflow permissions, secrets, and environments form the security foundation of the pipeline — see [repo-security.md](./references/repo-security.md).

## AI-driven PR review

Load this skill's plugin and the agent applies the relevant Go skills per review area — catching architectural drift, logic bugs, missing error context, and concurrency hazards linters cannot see. Cost note: review agents run per PR; remove jobs you don't need or filter triggers to specific branches.

The asset workflows run on a CI runner, not the local harness, so their tool names and permission flags are deliberately literal.

**Claude Code:** [claude-code-review.yml](./assets/claude-code-review.yml) runs parallel jobs:

| Job | Areas | Priority |
| --- | --- | --- |
| `quality` | style, naming, docs, design patterns | suggestion-first |
| `correctness` | error handling, code safety, concurrency | blocking-first |
| `security` | security, dependencies | blocking-first |
| `quality-depth` | tests, performance, observability, modernize | mixed |

Relevant skills vary per project: `golang-cli`, `golang-context`, `golang-data-structures`, `golang-database`, `golang-dependency-injection`, or any library-specific skill. App secrets are wired via the `/install-github-app` command.

**GitHub Copilot:** copy skills into the repo and append [copilot-review-instructions.md](./assets/copilot-review-instructions.md) to `.github/copilot-instructions.md`:

```bash
npx skills add https://github.com/fabianoflorentino/golang-agent-skills --agent github-copilot --skill '*' -y --copy
ln -s .agents .copilot
```

## Common mistakes

| Mistake | Fix |
| --- | --- |
| No `-race` in CI tests | always `go test -race` |
| No `-shuffle=on` | randomize to catch inter-test dependencies |
| Integration tests cached | `-count=1` |
| `go mod tidy` drift unchecked | `go mod tidy && git diff --exit-code` step |
| `fail-fast: true` default | coordinate matrix failures, don't cancel them |
| Unpinned actions | `@vN` major, never `@master` |
| Missing `permissions` | least-privilege per job |
| govulncheck findings ignored | fix or suppress with a written justification |
| No AI review | add Claude Code / Copilot — catches logic, security, architecture issues static analysis misses |

## Cross-references

- `golang-lint` — recommended `.golangci.yml` config.
- `golang-security` — interpreting scanner findings.
- `golang-testing` — test-writing practices the pipeline gates on.
- `golang-dependency-management` — choosing/upgrading the versions bots propose.
- `golang-modernize` — CI tooling refresh (golangci-lint v2, govulncheck, PGO).
