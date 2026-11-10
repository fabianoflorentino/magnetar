# gopls CLI reference

The Go team documents the gopls command line as experimental — "not efficient, complete, flexible, or officially supported." Treat it as a debugging and one-shot-scripting fallback; when an MCP or native `LSP` surface is available, prefer those (see [mcp.md](mcp.md)).

## Locating a position

Two equivalent spellings point at a spot in a file:

- `file.go:line:column` — both 1-indexed; the column counts UTF-8 bytes, not runes or UTF-16 units, so a non-ASCII line can disagree with what an editor reports.
- `file.go:#offset` — a 0-indexed byte offset from the start of the file.

```bash
gopls definition internal/cmd/definition.go:44:47
gopls definition internal/cmd/definition.go:#1270
```

## Global flags

Flags that belong to `gopls` itself and come before the subcommand:

| Flag | Value | Purpose |
| --- | --- | --- |
| `-logfile=<path>` | a file path, or the literal string `auto` | Where logs go; `auto` picks a default output file instead of stderr |
| `-profile.cpu=<path>` | a file path | Write a CPU profile |
| `-profile.mem=<path>` | a file path | Write a memory profile |
| `-profile.alloc=<path>` | a file path | Write an allocation profile |
| `-profile.block=<path>` | a file path | Write a blocking profile |
| `-profile.trace=<path>` | a file path | Write an execution trace |
| `-v`, `-verbose` | boolean | Verbose output |
| `-vv`, `-veryverbose` | boolean | Very verbose output |

`gopls mcp` accepts a narrower set: `-listen=<addr>` (serve over SSE/HTTP instead of stdio), `-logfile=<path>` (defaults to stderr), and `-rpc.trace` (which cannot combine with `-listen`).

## Shared write flags

Every command that can modify source (`format`, `imports`, `rename`, `codeaction`, `codelens`, `execute`) accepts the same boolean options:

| Flag | Purpose |
| --- | --- |
| `-w`, `-write` | Write the edited content back to the source file(s) |
| `-d`, `-diff` | Print a unified diff instead of writing |
| `-l`, `-list` | Print only the names of the files that would be/were edited |
| `-preserve` | With `-w`, keep a copy of each original file before overwriting |

None of these exclude each other; passing none simply computes the edit without printing or writing it.

## Navigation commands

| Command | Flags | Example | Notes |
| --- | --- | --- | --- |
| `definition` | `-json`, `-markdown` | `gopls definition helper/helper.go:8:6` | `-json` gives structured output; `-markdown` renders doc comments as Markdown |
| `references` | `-d`, `-declaration` | `gopls references helper/helper.go:8:6` | Include the declaration itself in the result set |
| `implementation` | none | `gopls implementation helper/helper.go:8:6` | — |
| `call_hierarchy` | none | `gopls call_hierarchy helper/helper.go:8:6` | Static calls only |
| `symbols` | none | `gopls symbols helper/helper.go` | File-scoped outline |
| `workspace_symbol` | `-matcher=<one of fuzzy,fastfuzzy,casesensitive,caseinsensitive>` (default `caseinsensitive`) | `gopls workspace_symbol -matcher fuzzy 'wsymbols'` | Matching algorithm for the query |
| `signature` | none | `gopls signature helper/helper.go:8:6` | Function signature at position |
| `highlight` | none | `gopls highlight helper/helper.go:8:6` | Identifiers referencing the same symbol |
| `folding_ranges` | none | `gopls folding_ranges helper/helper.go` | Collapsible regions |
| `links` | `-json` | `gopls links internal/cmd/check.go` | Structured output when set |
| `prepare_rename` | none | `gopls prepare_rename helper/helper.go:8:6` | Validate that a rename is possible before attempting it |
| `semtok` | none | `gopls semtok internal/cmd/semtok.go` | Semantic token dump |

## Diagnostics

| Command | Flags | Example |
| --- | --- | --- |
| `check` | `-severity=<one of hint,info,warning,error>` (default `warning`); reports findings at or above this severity | `gopls check -severity=error internal/cmd/check.go` |

## Transformation commands

All of these also accept the [shared write flags](#shared-write-flags).

| Command | Positional args | Example | Notes |
| --- | --- | --- | --- |
| `format` | one or more `<filerange>` (a file, or a range within one) | `gopls format -w internal/cmd/check.go` | Canonical gofmt-equivalent; ignores client formatting options |
| `imports` | `<filename>` | `gopls imports -w internal/cmd/check.go` | Add, remove, sort imports |
| `rename` | `<position> <new-name>` | `gopls rename helper/helper.go:8:6 Foo` | `<new-name>` is a plain identifier — run `prepare_rename` first if unsure |

## Code actions and code lenses

`codeaction` and `codelens` additionally accept the [shared write flags](#shared-write-flags).

| Command | Extra flags | Notes |
| --- | --- | --- |
| `codeaction` | `-kind=<value>` — comma-separated kinds (see below); `-title=<regex>` — filter by title; `-exec` — execute the first match instead of only listing | Kinds are hierarchical, so `-kind=refactor` matches every `refactor.*`; only one action executes per invocation, with no conflict resolution for applying several; actions of kind `source.test` are excluded unless explicitly requested via `-kind` |
| `codelens` | `-exec` — run the first matching lens instead of only listing | Positional args: `<file>`, `<file:line>`, or `<file> <title>` |
| `execute` | none beyond the shared write flags | `<command> <json-argument>` — sends a raw LSP `ExecuteCommand` request; gopls's command set is unstable and may shift between versions |

```bash
# List the quick fixes available in a range
gopls codeaction -kind=quickfix ./gopls/main.go

# Execute the first matching action, showing the diff
gopls codeaction -kind=quickfix -exec -diff ./gopls/main.go

# Filter by title as a regex in addition to kind
gopls codeaction -kind=refactor.rewrite -title 'Fill struct' -exec -w file.go:12:3

# Code lenses: list, or run a specific one
gopls codelens a_test.go
gopls codelens a_test.go:10
gopls codelens a_test.go "run test"
gopls codelens -exec a_test.go:10 "run test"

# Raw LSP ExecuteCommand
gopls execute gopls.add_import '{"ImportPath": "fmt", "URI": "file:///hello.go"}'
gopls execute gopls.run_tests '{"URI": "file:///a_test.go", "Tests": ["Test"]}'
gopls execute gopls.list_known_packages '{"URI": "file:///hello.go"}'
```

## Introspection

| Command | Flags | Notes |
| --- | --- | --- |
| `stats` | `-anon` | JSON summary of workspace info relevant to performance; populates the file cache as a side effect. `-anon` redacts fields that could leak user/file names or source text |
| `version` | none | gopls version info |
| `api-json` | none | The full gopls API surface as JSON |
| `bug` | none | Report a gopls bug |
| `licenses` | none | Licenses of bundled software |

```bash
gopls stats
gopls stats -anon
gopls version
gopls api-json
gopls bug
gopls licenses
```

## CodeAction kinds

Passed to `-kind` on `codeaction` (comma-separated; hierarchical, so `refactor` covers all `refactor.*`):

```
gopls.doc.features
quickfix
refactor
refactor.extract
refactor.extract.constant
refactor.extract.function
refactor.extract.method
refactor.extract.toNewFile
refactor.extract.variable
refactor.inline
refactor.inline.call
refactor.rewrite
refactor.rewrite.changeQuote
refactor.rewrite.fillStruct
refactor.rewrite.fillSwitch
refactor.rewrite.invertIf
refactor.rewrite.joinLines
refactor.rewrite.removeUnusedParam
refactor.rewrite.splitLines
source
source.assembly
source.doc
source.fixAll
source.freesymbols
source.organizeImports
source.test
```

Beyond this `-kind`-documented set, a few kinds exist only behind editor UI or `execute`/code lens and cannot be filtered by name: `refactor.extract.variable-all`, `refactor.extract.constant-all`, `refactor.inline.variable`, `refactor.rewrite.moveParamLeft`, `refactor.rewrite.moveParamRight`, `refactor.rewrite.eliminateDotImport`, `refactor.rewrite.addTags`, `refactor.rewrite.removeTags`, `refactor.rewrite.implementInterface`, `source.addTest`, `source.splitPackage`, `source.toggleCompilerOptDetails`. Their behavior is covered in [features.md](features.md#transformation).
