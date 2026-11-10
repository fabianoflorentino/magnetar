# gopls settings reference

Source: [tip.golang.org/gopls/settings](https://tip.golang.org/gopls/settings); the exhaustive, always-current API is `gopls api-json`. Settings reach gopls through the client's LSP `initializationOptions` — there is no `gopls.json` the server reads from disk. Record chosen values in the project's agent config (CLAUDE.md, AGENTS.md, or equivalent) so later sessions inherit them.

## Build and layout

| Setting | Type | Default | What it controls |
| --- | --- | --- | --- |
| `buildFlags` | `[]string` | `[]` | Extra flags handed to the build system, most commonly `-tags=<tag>` to pull build-tagged files into scope |
| `env` | `map[string]string` | `{}` | Environment for external commands gopls shells out to (`go list`, and others) |
| `directoryFilters` | `[]string` | `["-**/node_modules"]` | Include/exclude workspace directories from loading and workspace-symbol search, via `+`/`-`-prefixed glob patterns |
| `expandWorkspaceToModule` | `bool` | `true` | Whether the enclosing module counts as "workspace" for diagnostic scope, not just the opened directory |
| `templateExtensions` | `[]string` | `[]` | Extensions treated as Go template files; empty because templates have no canonical extension |

**Practical impact:** a symbol behind a build tag (`//go:build integration`) stays invisible to `references`/`go_search` until `buildFlags: ["-tags=integration"]` appears — the same root cause as the References build-scoping gotcha in [features.md](features.md#navigation).

## Formatting

| Setting | Type | Default | What it controls |
| --- | --- | --- | --- |
| `local` | `string` | `""` | Import path prefix classed as "local" for grouping/ordering — the analogue of `goimports -local` |
| `gofumpt` | `bool` | `false` | Format with `mvdan.cc/gofumpt`'s stricter rules rather than plain `gofmt` |

## Diagnostics

| Setting | Type | Default | What it controls |
| --- | --- | --- | --- |
| `analyses` | `map[string]bool` | `{}` | Enable/disable individual analyzers by name (the vet-based framework plus gopls's own) |
| `staticcheck` | `bool` | `false` | Add the staticcheck.io suite on top of the built-in analyzers |
| `vulncheck` | enum: `Off`\|`Imports`\|`Prompt` | `"Prompt"` (some clients default to `"Off"`) | Whether and how `go.mod` diagnostics feed off the vulnerability database |
| `diagnosticsDelay` | `time.Duration` | `"1s"` | Idle time after an edit before workspace-wide analysis recomputes (open-file compile errors surface sooner regardless) |
| `diagnosticsTrigger` | enum: `Edit`\|`Save` | `"Edit"` | Recompute diagnostics on every edit, or only on save |
| `pullDiagnostics` | `bool` | `false` | Let the client request diagnostics on demand (`textDocument/diagnostic`) instead of receiving only pushed updates |

**Practical impact:** with the defaults, a large monorepo recomputes on every keystroke pause; switching `diagnosticsTrigger` to `"Save"` trades immediacy for fewer recomputations. Enabling `staticcheck` changes the shape of `go_diagnostics`/`gopls check` output substantially — expect more findings, not a regression.

## Documentation

| Setting | Type | Default | What it controls |
| --- | --- | --- | --- |
| `hoverKind` | enum: `FullDocumentation`\|`SingleLine`\|`Structured`\|`NoDocumentation`\|`SynopsisDocumentation` | `"FullDocumentation"` | How much doc text Hover renders |
| `linksInHover` | `bool` | `true` | Whether hover markdown includes doc-comment links |
| `linkTarget` | `string` | `"pkg.go.dev"` | Base host for generated documentation links (hover, Document Link, diagnostics) |

## Inlay hints

| Setting | Type | Default | What it controls |
| --- | --- | --- | --- |
| `hints` | `map[string]bool` | `{}` (all off) | Which hint kinds are on — keys: `parameterNames`, `assignVariableTypes`, `compositeLiteralFields`, `compositeLiteralTypes`, `constantValues`, `functionTypeParameters`, `rangeVariableTypes` |

Example:

```json
"hints": {
  "parameterNames": true,
  "assignVariableTypes": true
}
```

## Navigation

| Setting | Type | Default | What it controls |
| --- | --- | --- | --- |
| `symbolMatcher` | enum: `FastFuzzy`\|`Fuzzy`\|`CaseSensitive`\|`CaseInsensitive` | `"FastFuzzy"` | Matching algorithm for `workspace/symbol` and `go_search` |
| `symbolScope` | enum: `all`\|`workspace` | `"all"` | Whether symbol search covers only workspace packages or every loaded package including dependencies |
| `symbolStyle` | enum | — | How matched symbols are qualified in the response (package-qualified vs. bare) |
| `codelenses` | `map[string]bool` | — | Toggle individual code lenses such as `generate`, `tidy`, `vendor`, `run_tests` |

The tables above cover the knobs most likely to change day-to-day behavior; experimental and client-specific keys, plus any drift across releases, live in `gopls api-json` and the upstream settings page.
