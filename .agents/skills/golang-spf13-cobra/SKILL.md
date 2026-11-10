---
name: golang-spf13-cobra
description: "Golang CLI command tree library using spf13/cobra — cobra.Command, RunE vs Run, PersistentPreRunE hook chain, Args validators (NoArgs, ExactArgs, MatchAll, custom), persistent vs local flags, command groups, ValidArgsFunction, RegisterFlagCompletionFunc, ShellCompDirective, usage/help template customization, man-page and markdown doc generation, and testing with SetArgs/SetOut/SetErr. Apply when using or adopting spf13/cobra, or when the codebase imports `github.com/spf13/cobra`. For configuration layering alongside cobra, see the `fabianoflorentino/golang-agent-skills@golang-spf13-viper` skill. For general CLI architecture (project layout, exit codes, signal handling, I/O patterns), see `fabianoflorentino/golang-agent-skills@golang-cli`."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🐍"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch mcp__context7__resolve-library-id mcp__context7__query-docs Bash(godig:*) Bash(gopls:*) LSP mcp__gopls__*
paths:
  - "**/*.go"
---

**Persona:** You are a Go CLI engineer building command trees that feel native to the Unix shell. You design the user-facing surface first, then wire behavior into the right hook.

**Modes:**

- **Build** — a new CLI from scratch: command tree, hooks, flags, then completions.
- **Extend** — adding subcommands or flags to an existing CLI: read the command tree first, then stay consistent with it.
- **Review** — auditing a CLI for `RunE` misuse, hard-coded `os.Stdout`, hook-order bugs, and missing args validation.

**When to use:** any task centered on spf13/cobra. cobra owns commands/flags only; configuration layering is `golang-spf13-viper`, and platform concerns (exit codes, signals, I/O layout) are `golang-cli`.

## What cobra gives you

The command/subcommand tree, `pflag` parsing, positional-arg validation, generated shell completions, and doc generation — all in one library. It deliberately does not resolve config files or environment variables; that is viper's job.

## Command tree basics

The root command's `Use` is the binary name; every level registers children with `AddCommand`. On the root, silence the framework's own error noise and keep error-format control:

```go
rootCmd := &cobra.Command{
    Use:           "myapp",
    Short:         "manage everything",
    SilenceUsage:  true,
    SilenceErrors: true,
}
```

`AddGroup` attaches a group label to subcommands in help output — call it before the `AddCommand` calls that reference the group, because cobra does not retroactively assign commands.

## The hook chain

Five hooks fire in this order:

```text
PersistentPreRunE → PreRunE → RunE → PostRunE → PersistentPostRunE
```

Rules that matter:

- Prefer the `*E` forms everywhere; the non-`E` variants cannot return an error.
- A root `PersistentPreRunE` runs before every subcommand — the natural home for config init, viper binding, and auth setup.
- A child's `PersistentPreRunE` **replaces** the parent's; re-invoke the parent explicitly when both are needed.
- `PostRunE` runs only when `RunE` succeeded.

Details and inheritance nuances: [commands and args](references/commands-and-args.md).

## Validating positional args

Declare an `Args` validator instead of counting `len(args)` inside `RunE` — the latter bypasses cobra's standard messages and its tracking:

```go
cmd := &cobra.Command{
    Use: "serve <dir>",
    Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
    RunE: func(cmd *cobra.Command, args []string) error {
        return runServer(cmd, args[0])
    },
}
```

Built-ins: `NoArgs`, `ExactArgs`, `MinimumNArgs`, `MaximumNArgs`, `RangeArgs`, `OnlyValidArgs`, `ExactValidArgs`; compose with `MatchAll`, or write `func(cmd *cobra.Command, args []string) error` for custom logic. Full catalog in [commands and args](references/commands-and-args.md).

## Flags

`pflag` backs cobra. Persistent flags are inherited by every subcommand; local flags belong to the declaring command only:

```go
rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
serveCmd.Flags().IntVar(&port, "port", 8080, "listen port")
serveCmd.MarkFlagRequired("port")
serveCmd.MarkFlagsMutuallyExclusive("json", "yaml")
```

pflag types, custom flag values, flag groups, and viper binding: [flags](references/flags.md).

## Completions

Cobra generates shell completion out of the box. Extend it:

- `ValidArgs []string` for static positional completion.
- `ValidArgsFunction` for dynamic: return candidates plus a `ShellCompDirective` (`NoFileComp` suppresses the file fallback).
- `RegisterFlagCompletionFunc(name, fn)` for flag-value completion.

Directive values, annotation tricks, and end-to-end tests: [completions](references/completions.md).

## Output: never `os.Stdout` directly

Handlers must write through `cmd.OutOrStdout()` / `cmd.ErrOrStderr()`, otherwise tests cannot capture the output:

```go
func TestServeCmd(t *testing.T) {
    var buf bytes.Buffer
    rootCmd.SetOut(&buf)
    rootCmd.SetArgs([]string{"serve", "--port", "9090"})
    require.NoError(t, rootCmd.Execute())
    assert.Contains(t, buf.String(), "listening on :9090")
}
```

Cobra accumulates flag state across `Execute()` calls, so build a fresh command tree per test. Isolation and golden-file patterns: [testing](references/testing.md).

## Best practices

1. Always `RunE`, never `Run` — the errorless form leaves only `os.Exit` or a panic, skipping defers.
2. Init config and bindings in `PersistentPreRunE` of the root.
3. Validate positionals with `Args`, not hand-rolled counts in `RunE`.
4. Route all output through `OutOrStdout()`/`ErrOrStderr()`.
5. Fresh command tree per test; never share flag state.

## Common mistakes

| Mistake | Failure | Fix |
| --- | --- | --- |
| `Run` instead of `RunE` | errors escape via `os.Exit`; defers skipped | `RunE`, return the error |
| `len(args)` checks in `RunE` | loses cobra's "accepts 1 arg" messages | `Args: cobra.ExactArgs(1)` |
| Writing to `os.Stdout` | tests can't capture it | `cmd.OutOrStdout()` |
| Child `PersistentPreRunE` | parent hook silently dropped | call `parent.PersistentPreRunE(cmd, args)` |
| Reusing one root across tests | leaked flag state between `Execute()`s | build a fresh tree per test |

## Further reading

- [generators](references/generators.md) — man-page/markdown/YAML/RST output and the `cobra-cli` scaffolder.

## Cross-references

- `golang-cli` — exit codes, signals, project layout, general CLI architecture.
- `golang-spf13-viper` — layering flag → env → file → default for the config surface.
- `golang-testing` — general Go test structure.
- `golang-pkg-go-dev` / `golang-gopls` — package facts and call-site navigation.

Library bugs: [spf13/cobra issues](https://github.com/spf13/cobra/issues).
