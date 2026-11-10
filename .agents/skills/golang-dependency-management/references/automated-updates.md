# Automated Dependency Updates

Automation keeps dependencies current with security fixes while suppressing the maintenance drag of hand-reviewed bumps. It only works behind a CI pipeline that gates merges: tests and lint must pass before anything auto-applies. Workflow configuration (dependabot.yml, renovate.json, automerge workflows) is `golang-continuous-integration`'s domain; this reference is the policy.

## Dependabot vs Renovate

| Capability | Dependabot | Renovate |
| --- | --- | --- |
| Platforms | GitHub only | GitHub, GitLab, Bitbucket, self-hosted |
| `go mod tidy` | automatic | opt-in (`gomodTidy`) |
| Automerge | separate workflow | native support |
| Version grouping | pattern-based | flexible rules |
| Monorepo / workspaces | basic | Go workspaces aware |
| Regex managers | no | yes (Dockerfiles, Makefiles, ...) |

Renovate is generally the more configurable, mature option; Dependabot wins on simplicity for GitHub-only repos.

## Auto-merge strategy

- **Minor and patch** - auto-merge once CI passes (tests + lint + `govulncheck`) and the package is low risk for the project.
- **Major** - open a PR for manual review; breaking changes need an eye.
- **Security advisories** - auto-merge regardless of version-bump type; the fix is the point.

## Verification before committing

1. `go test ./...` and `go build ./...` pass.
2. `govulncheck ./...` (or `go tool govulncheck ./...`) is clean.
3. Read the changelog for anything touching persistence, serialization, networking, authentication, authorization, cryptography, or public APIs - those areas break silently.
4. For major upgrades, confirm documented breaking changes against your usage.
5. Note new APIs or patterns the update introduces - they may be worth adopting later, but do not let them ride along in the same change as the version bump.
