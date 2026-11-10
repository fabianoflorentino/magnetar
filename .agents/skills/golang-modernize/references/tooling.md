# Tooling and CI modernization

Beyond language features, surface the free, low-risk tooling upgrades that raise code quality and security: CI actions, linters, SAST and vulnerability scanners, Docker base images, coverage reporters, and dependency-update bots. Each is a small, high-value change with a clear rollback path.

## Raising the Go version

Compare the `go` directive in `go.mod` (and any `toolchain` directive) against the latest stable release and propose an upgrade when it lags — every release bundles performance work, security fixes, and new stdlib features.

```bash
go version                # current toolchain
go mod edit -go=1.27      # retarget the module
go get toolchain@latest   # refresh the toolchain requirement
```

## golangci-lint v2

Move linting to golangci-lint v2 (v2.0.0+, March 2025). `golangci-lint migrate` converts a v1 config to v2; the recommended configuration lives in the `golang-lint` skill.

## govulncheck

`govulncheck` consults the Go vulnerability database at `vuln.go.dev` and analyzes call graphs so it reports only **reachable** vulnerabilities. Database coverage of the standard library starts at Go 1.18; third-party module vulnerabilities are tracked regardless of Go version. For better call-graph analysis, run it with Go 1.22+.

```bash
# Pin as a module tool (Go 1.24+)
go get -tool golang.org/x/vuln/cmd/govulncheck@latest

# Scan source
go tool govulncheck ./...

# Scan a compiled binary
go tool govulncheck -mode=binary ./myapp
```

## Profile-guided optimization

PGO has been generally available since Go 1.21, typically delivering 2-14% speedups:

```bash
go test -cpuprofile=default.pgo -bench=. ./...
# place default.pgo in the main package directory, then rebuild
go build ./...
```

PGO applies automatically once `default.pgo` is present. Go 1.22 extended it to devirtualize more interface calls; Go 1.23 cut the PGO build-time overhead to single digits.

## `go fix`, `go doc`, `go mod tidy` (Go 1.27+)

`go fix` now applies a share of the `modernize` fixer suite automatically. Coverage shifted between Go 1.26 and 1.27:

- **Added:** `atomictypes`, `embedlit`, `slicesbackward`, `unsafefuncs`
- **Removed:** `fmtappendf` (superseded by other fixers)
- **Renamed:** `waitgroup` → `waitgroupgo`

```bash
go fix ./...             # apply the enabled safe transformations
go tool fix help         # exact coverage for the installed toolchain
```

`go doc` gained `package@version` (document a specific module version without touching `go.mod`) and `-ex` (list executable examples):

```bash
go doc golang.org/x/tools/cmd/stringer@v0.30.0
go doc -ex net/http.Client
```

For modules declaring `go 1.27` or newer, `go mod tidy` auto-merges duplicate `require` blocks and enforces the two-block layout (direct, then indirect), preserving existing comments.

`go test -json` output gained an optional `OutputType` field (`"error"`, `"error-continue"`, `"frame"`) for consumers that need to tell test-output kinds apart programmatically.

`go tool trace -http` binds to localhost by default; pass `-http=0.0.0.0:6060` explicitly to expose the trace viewer beyond the local machine (relevant when tracing in a container or remote dev environment).

## AI-driven code review in CI

Add an AI agent as a PR reviewer alongside static analysis. Configured with this skill's plugin, the agent loads the relevant Go skills — `golang-security` for security review, `golang-concurrency` for concurrency, `golang-error-handling` for error handling, and so on — giving it senior-Go-reviewer expertise. It catches architectural drift, logic bugs, missing error context, and subtle concurrency hazards that linters cannot see. Ready-to-use GitHub Actions assets for both Claude Code and GitHub Copilot live in the `golang-continuous-integration` skill.
