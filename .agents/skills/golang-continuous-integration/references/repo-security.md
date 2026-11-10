# Repository security settings

Workflows are only as trustworthy as the repo they run in. These settings are the security foundation the pipeline stands on — configure them after the workflow files exist.

Find the project's GitHub URL from its git remote (`git remote -v`). For a project at `https://github.com/{owner}/{repo}`, the relevant settings pages are:

- Branch protection: `https://github.com/{owner}/{repo}/settings/branches`
- Actions permissions: `https://github.com/{owner}/{repo}/settings/actions`
- Actions secrets: `https://github.com/{owner}/{repo}/settings/secrets/actions`
- Environments: `https://github.com/{owner}/{repo}/settings/environments`

## Branch protection rules

Protect `main` (or the default branch) with a rule that covers:

- **Require a pull request before merging** — no direct pushes to the protected branch.
- **Require approvals (at least 1)** — a change cannot self-merge without review.
- **Dismiss stale approvals on new commits** — prevents "approve, then sneak in changes".
- **Require status checks to pass** — add every CI job name as a required check (e.g. `Test (Go 1.27)`, `Test (Go stable)`, `Lint`), keep them updated when jobs are renamed.
- **Require branches up to date** — blocks merging a PR never tested against latest `main`.
- **Do not allow bypassing the settings** — apply the rules to admins as well.

## Workflow permissions

Default the repository-level `GITHUB_TOKEN` to read-only:

1. Open **Actions → Settings** (link above).
2. Under **Workflow permissions**, choose **"Read repository contents and packages permissions"** — least privilege is the baseline.
3. Uncheck **"Allow GitHub Actions to create and approve pull requests"** unless auto-merge needs it — then check it only for that purpose.

Workflows therefore start with no write access; any job that needs more must declare it in its own `permissions:` block. That is defense-in-depth: a compromised workflow cannot write to the repo unless it was explicitly granted the token scope to.

## Fork pull request restrictions

For public or open-source repositories:

- In **Actions permissions**, set **"Fork pull request workflows from outside collaborators"** to **"Require approval for all outside collaborators"**.
- This stops untrusted forks from running workflows that burn your Actions minutes or touch your secrets.
- Never use `pull_request_target` against untrusted code — that event runs with write access to the base repository.

## Secrets and environments

- Keep secrets out of workflow files entirely — define them under **Settings → Secrets → Actions**.
- Add a manual approval gate to publishing by creating a **"release" environment** (with required reviewers) for the release workflow.
- Rotate `CODECOV_TOKEN` and other third-party tokens on a schedule.

## Permission risk reference

The security implications of each permission a workflow can request:

| Permission | Workflows that need it | Risk |
| --- | --- | --- |
| `contents: read` | All workflows | **Low** — read-only, the safe default |
| `contents: write` | Release, auto-merge | **High** — can modify repo contents and create releases |
| `packages: write` | Docker | **High** — can push container images to GHCR |
| `pull-requests: write` | Auto-merge | **High** — can approve and merge PRs |
| `attestations: write` | Docker | **Medium** — creates provenance/SBOM attestations |
| `id-token: write` | Docker | **Medium** — OIDC token for signing attestations |
| `security-events: write` | Security/SAST, Docker | **Medium** — uploads SARIF results to the Security tab |

Grant the narrowest scope that works. If a job only needs `contents: read`, it should not receive anything more.
