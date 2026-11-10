# Project Documentation

## README.md

The README is the project's front page — simple, clear, scannable. Use this order:

1. **Title** — the name as `# heading`
2. **Badges** — Go version, license, CI, coverage
3. **Summary** — 1–2 sentences on what it does
4. **Demo** — snippet (libraries), screenshot (web), GIF or video (CLIs)
5. **Getting Started** — install plus a minimal working example
6. **Features / specification** — the largest section, grouped by feature
7. **Contributing** — link to CONTRIBUTING.md
8. **License** — name and link

A LICENSE file must exist in every project. Uncomment the application-only sections (binary download table, Docker, Homebrew) only when they apply.

## CONTRIBUTING.md

The bar: a new contributor clones, makes a change, and runs the tests in under 10 minutes.

| Problem | Fix |
| --- | --- |
| Complex build steps | `Makefile` with `make build`, `make test`, `make lint` |
| External service dependencies | `docker-compose.yml` for local dev |
| Inconsistent environments | `.devcontainer/` for devcontainers |
| Slow test suite | Separate fast unit tests from integration (build tags) |
| Missing instructions | `make help` listing targets |

## Changelog

Keep a changelog for every release using the Keep a Changelog categories — `Added`, `Changed`, `Deprecated`, `Removed`, `Fixed`, `Security` — with a version link to the comparison diff:

```markdown
## [1.2.0] - 2026-03-08

### Added

- New `WithTimeout` option for client configuration

### Changed

- Improved retry logic to use exponential backoff

### Fixed

- Race condition in the connection pool under heavy load

[1.2.0]: https://github.com/{owner}/{repo}/compare/v1.1.0...v1.2.0
```

GitHub Releases can replace the file for simpler projects; GoReleaser drafts notes from commits. Automation lives in `golang-continuous-integration`.

## Distribution

Offer more than one installation path — binaries, containers, and package managers (APT, Homebrew) each serve a different user segment. A single path taxes adoption: workflow fit decides whether a tool gets tried.

### Dockerfile best practices

Use a multi-stage build to a minimal, non-root image:

```dockerfile
FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/binary ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /app/binary /binary
ENTRYPOINT ["/binary"]
```