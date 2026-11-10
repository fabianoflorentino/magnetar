---
name: golang-spf13-viper
description: "Golang configuration library using spf13/viper — layered precedence (flag > env > file > KV > default), BindPFlag/BindPFlags, SetEnvPrefix + SetEnvKeyReplacer + AutomaticEnv, ReadInConfig + ConfigFileNotFoundError, Unmarshal + mapstructure struct tags, Sub for sub-trees, WatchConfig + OnConfigChange for hot reload, viper.New() for test isolation, and remote KV integration. Apply when using or adopting spf13/viper, or when the codebase imports `github.com/spf13/viper`. For CLI command structure alongside viper, see the `fabianoflorentino/golang-agent-skills@golang-spf13-cobra` skill. For general CLI architecture, see `fabianoflorentino/golang-agent-skills@golang-cli`."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🔧"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch mcp__context7__resolve-library-id mcp__context7__query-docs Bash(godig:*) Bash(gopls:*) LSP mcp__gopls__*
paths:
  - "**/*.go"
---

**Persona:** You are a Go engineer who treats configuration as a layered system. Flag beats env beats file beats default — and every key stays reachable through one API.

**Modes:**

- **Build** — wiring viper into a service or CLI.
- **Review** — auditing precedence, env key formatting, and global-vs-instance state.

**When to use:** any task centered on spf13/viper. The library has no user-facing surface: it answers "what is the value of key X right now?" by walking its sources in a fixed order. Keep the command tree across the hall in `golang-spf13-cobra`.

## The precedence pipeline

Resolution never changes order; the first source that has the key wins:

```text
1. explicit Set()
2. bound flag (BindPFlag)
3. env var (BindEnv / AutomaticEnv)
4. config file (ReadInConfig)
5. remote KV (etcd, Consul)
6. default (SetDefault)
```

Most viper bugs are precedence bugs: a key that "should" come from a file is shadowed by an env var, or a flag default silently beats a file value. Keep the order in mind when diagnosing.

## Reading config files

```go
viper.SetConfigName("config")
viper.AddConfigPath("$HOME/.myapp")
if err := viper.ReadInConfig(); err != nil {
    var notFound *viper.ConfigFileNotFoundError
    if !errors.As(err, &notFound) {
        return fmt.Errorf("reading config: %w", err)
    }
}
```

A missing file is usually fine — the app may be running on flags and env alone — so absorb `ConfigFileNotFoundError` and propagate only real failures. Formats (JSON, TOML, YAML, HCL, INI, properties), `MergeInConfig`, and remote KV: [sources and formats](references/sources-and-formats.md).

## Env binding: the bug factory

Three settings must be wired together; drop any one and nested keys silently break:

```go
viper.SetEnvPrefix("MYAPP")                            // PORT → MYAPP_PORT
viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // database.host → MYAPP_DATABASE_HOST
viper.AutomaticEnv()
```

Without the replacer viper hunts for `MYAPP_DATABASE.HOST` — dot preserved — and never finds it. `BindEnv`, `AllowEmptyEnv`, and env-vs-default interactions: [binding and env](references/binding-and-env.md).

## Binding cobra flags (the seam)

Bind in `init()` or root `PersistentPreRunE` — never in `RunE`, which runs after config loading already happened:

```go
func init() {
    rootCmd.PersistentFlags().Int("port", 8080, "listen port")
    viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
}
```

`BindPFlags(cmd.Flags())` binds a whole FlagSet at once.

## Unmarshalling into structs

`Unmarshal` maps the resolved tree into a struct via `mapstructure` — and that mapping needs explicit tags:

```go
type Config struct {
    Port int `mapstructure:"port"`
    DB   struct {
        MaxConn int `mapstructure:"max_conn"`
    } `mapstructure:"db"`
}
var cfg Config
viper.Unmarshal(&cfg)
```

Without `mapstructure:"max_conn"` the library never converts underscore keys. Prefer `UnmarshalKey("db", &cfg)` over `Sub("db").Unmarshal(...)` — `Sub` returns nil for missing keys and forces a nil-check. Decoder hooks for `time.Duration`, `net.IP`, and slices: [unmarshal](references/unmarshal.md).

## Sub-trees and hot reload

`viper.Sub("database")` scopes a new instance under a prefix but returns **nil** when the prefix is absent — nil-check before use, or skip it via `UnmarshalKey`. For live config:

```go
viper.WatchConfig()
viper.OnConfigChange(func(e fsnotify.Event) {
    // re-read and re-apply changed values
})
```

`WatchConfig` rides fsnotify on inodes, so editors that save atomically via rename (vim, neovim) replace the inode and the callback may never fire — test with `echo >> config.yaml`, not an editor save. Race-safe reload patterns: [watch and reload](references/watch-and-reload.md).

## Test isolation

Never test against the package-global viper — its state leaks across cases. One instance per test:

```go
v := viper.New()
v.SetConfigFile("testdata/config.yaml")
require.NoError(t, v.ReadInConfig())
```

`t.Setenv` interactions and `Reset()` caveats: [testing and isolation](references/testing-and-isolation.md).

## Best practices

1. `SetEnvPrefix` + `SetEnvKeyReplacer` + `AutomaticEnv` always together.
2. Swallow `ConfigFileNotFoundError`; propagate everything else.
3. `mapstructure` tags on every config-struct field.
4. `viper.New()` per test; never the global.
5. Bind flags before `Execute()` — `RunE` is too late.

## Common mistakes

| Mistake | Failure | Fix |
| --- | --- | --- |
| `AutomaticEnv` without the replacer | nested keys resolve as `MYAPP_DATABASE.HOST` | `NewReplacer(".", "_")` first |
| Missing `mapstructure` tags | nested/underscore fields silently skipped | tag every field |
| Global viper in tests | cross-test contamination, flaky order | `viper.New()` per test |
| Ignoring `ConfigFileNotFoundError` | crashes on flags/env-only runs | `errors.As` and continue |

## Further reading

- [binding and env](references/binding-and-env.md) — env binding, timing rules.
- [unmarshal](references/unmarshal.md) — custom decode hooks.
- [watch and reload](references/watch-and-reload.md) — fsnotify caveats.
- [testing and isolation](references/testing-and-isolation.md) — snapshot/restore patterns.

## Cross-references

- `golang-spf13-cobra` — defining and binding the flags viper overrides.
- `golang-cli` — CLI architecture and cobra+viper integration.
- `golang-testing` — general test structure.
- `golang-pkg-go-dev` / `golang-gopls` — package facts and call-site navigation.

Library bugs: [spf13/viper issues](https://github.com/spf13/viper/issues).
