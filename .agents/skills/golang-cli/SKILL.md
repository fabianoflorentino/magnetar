---
name: golang-cli
description: "Golang CLI application development. Use when building, modifying, or reviewing a Go CLI tool — especially for command structure, flag handling, configuration layering, version embedding, exit codes, I/O patterns, signal handling, shell completion, argument validation, and CLI unit testing. Also triggers when code uses cobra, viper, or urfave/cli. For cobra-specific APIs → See `fabianoflorentino/golang-agent-skills@golang-spf13-cobra` skill; for viper configuration layering → See `fabianoflorentino/golang-agent-skills@golang-spf13-viper` skill."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "💻"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent AskUserQuestion
paths:
  - "**/*.go"
---

# CLI applications in Go

**Persona:** You are a Go CLI engineer. You build tools that feel native to the Unix shell — composable pipes, scriptable output, predictable exit codes, and zero surprises under automation.

**Modes:**

- **Build** — a new CLI from scratch: project structure, root setup, flags, version embedding, in order. Sequential.
- **Extend** — add subcommands, flags, completions to an existing tree: read the command structure first, then change consistently. Sequential.
- **Review** — audit correctness: `SilenceUsage`/`SilenceErrors`, flag-to-Viper binding, exit codes, stdout/stderr discipline. Sequential.

**When to use:** any Go CLI work. Cobra specifics → `golang-spf13-cobra`; viper layering → `golang-spf13-viper`. For trivial single-purpose tools with a few flags, stdlib `flag` suffices.

## Stack

Cobra + Viper is the default: it powers kubectl, docker, gh, hugo. Related accessories: `spf13/pflag` (flag parsing via Cobra), `fatih/color` (auto-disabling color), `olekukonko/tablewriter`, `charmbracelet/bubbletea` (interactive), `ldflags` (version injection), `goreleaser` (distribution).

## Project structure

`cmd/myapp/` with one file per command; `main.go` only calls `Execute()`:

```text
myapp/
├── cmd/myapp/
│   ├── main.go          # package main; only Execute()
│   ├── root.go          # root command + Viper init
│   ├── serve.go         # "serve" subcommand
│   ├── migrate.go       # "migrate" subcommand
│   └── version.go       # "version" subcommand
├── go.mod
└── go.sum
```

Runnable examples of every piece: [`assets/examples/`](./assets/examples/) (main, root, serve, flags, args, config, version, exit_codes, output, signal, completion, and `cli_test.go`).

## Root command

- `SilenceUsage: true` — full usage text is noise on every error; reserve it for `--help`.
- `SilenceErrors: true` — you format errors yourself.
- `PersistentPreRunE` — config is always initialized before any subcommand runs.
- Logs to stderr; program output to stdout.

## Flags

- **Persistent** flags are inherited by all subcommands (`--config`); **local** flags apply to one command (`--port`).
- Enforce with `MarkFlagRequired`, `MarkFlagsMutuallyExclusive`, `MarkFlagsOneRequired`.
- Suggest values with `RegisterFlagCompletionFunc`.
- **Always `viper.BindPFlag`** configurable flags so `viper.GetInt("port")` honors flag > env > config > default.
- Set `viper.SetEnvPrefix("MYAPP")` — `PORT` collides with other tools; `MYAPP_PORT` does not.

## Argument validation

| Validator | Requires |
| --- | --- |
| `cobra.NoArgs` | No args |
| `cobra.ExactArgs(n)` | Exactly n |
| `cobra.MinimumNArgs(n)` | At least n |
| `cobra.MaximumNArgs(n)` | At most n |
| `cobra.RangeArgs(min, max)` | Between min and max |
| `cobra.ExactValidArgs(n)` | Exactly n, drawn from `ValidArgs` |

## Configuration layering

Viper precedence, highest first: **flags** → **environment variables** → **config file** → **code defaults**. An optional config file is a convenience, not a requirement — ignore `ConfigFileNotFoundError` so users without a file still boot.

## Version embedding

Never hardcode a version string that drifts from the tag — inject at build:

```bash
go build -ldflags "-X main.version=$(git describe --tags --always)"
```

## Exit codes

Unix conventions:

| Code | Meaning |
| --- | --- |
| 0 | Success |
| 1 | General error |
| 2 | Usage error |
| 64–78 | BSD sysexits |
| 126 | Not executable |
| 127 | Command not found |
| 128+N | Killed by signal N (130 = SIGINT) |

`main()` maps errors to codes once — never `os.Exit` inside `RunE`, or Cobra's error handling, deferred cleanup, and logging never run.

## I/O discipline

- stdout is pipeable program output — never logs/errors (corrupts the next program in the pipe); stderr carries diagnostics.
- Detect pipe vs terminal with `os.ModeCharDevice`.
- Offer `--output table|json|plain` for machine consumption.
- In commands use `cmd.OutOrStdout()`/`cmd.ErrOrStderr()`, not `os.Stdout`/`os.Stderr`, so tests can redirect.

## Signals

`signal.NotifyContext` propagates SIGINT/SIGTERM as context cancellation; wire it into HTTP servers and workers for graceful shutdown (see the signal example).

## Completions

Cobra generates bash/zsh/fish/PowerShell completions for free — ship a completion command and custom flag/value completions where they pay (see the completion example).

## Testing

Execute commands programmatically, capture `OutOrStdout`, assert. The `cli_test.go` example shows the shape; obey the I/O discipline above or tests cannot observe your output.

## Common mistakes

| Mistake | Fix |
| --- | --- |
| Writing to `os.Stdout` in commands | `cmd.OutOrStdout()` — redirectable in tests |
| `os.Exit` inside `RunE` | Return error; let `main()` decide |
| Flags not bound to Viper | `viper.BindPFlag` for every configurable flag |
| No `SetEnvPrefix` | Namespace env vars (`MYAPP_PORT`) |
| Logging to stdout | Logs to stderr |
| Usage on every error | `SilenceUsage: true` |
| Required config file | Ignore `ConfigFileNotFoundError` — optional |
| No `PersistentPreRunE` | Config init must precede every subcommand |
| Hardcoded version | `ldflags` from git tags |
| No machine-readable output | `--output` flag |

## Cross-references

- `golang-spf13-cobra` / `golang-spf13-viper` — library-specific APIs.
- `golang-project-layout` — where the CLI fits in the repo.
- `golang-testing` — programmatic command tests.
- `golang-design-patterns` — constructor/options style for CLI config.
- `golang-dependency-injection` — wiring components behind the commands.
