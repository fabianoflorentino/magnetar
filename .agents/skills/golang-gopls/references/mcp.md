# gopls MCP server and native LSP tool

Both surfaces talk to the same language server; they differ in how an agent addresses it. The MCP server takes names, paths, and queries; the native `LSP` tool takes editor-style `line`/`character` positions and pushes diagnostics for free. Wire up whichever your harness supports — both, if you can.

## Launching the MCP server

`gopls mcp` runs a dedicated gopls instance that speaks MCP over stdin/stdout, started fresh each session, with no editor attached:

```bash
gopls mcp
```

The server reads files **as they exist on disk**. Edits staged through another tool but not yet saved are invisible to it, so flush buffers before querying. This is the correct setup for a pure agent workflow that never opens an editor.

## Registering the server

`gopls mcp` is a plain executable, so any MCP-capable host can configure it as a command-based server. With Claude Code, register it via the CLI:

```bash
claude mcp add gopls -- gopls mcp
```

Other harnesses (Cursor, Windsurf, and similar) each keep their own MCP settings file; the entry always points the launch command at `gopls mcp` — there is no shared config format.

## MCP tools

Eight tools, all keyed by identifier, path, or query rather than cursor coordinates — the main ergonomic difference from the native `LSP` tool.

| Tool | Purpose | Example |
| --- | --- | --- |
| `go_workspace` | Overall shape of the workspace: module, multi-module workspace, or GOPATH layout. Call first, once per session. | `go_workspace({})` |
| `go_vulncheck` | Whether the current build reaches any known vulnerability. Run right after `go_workspace` in a workspace and again after any `go.mod` change. | `go_vulncheck({"pattern":"./..."})` |
| `go_search` | Fuzzy lookup of a type, function, or variable by name when the location is unknown. | `go_search({"query":"server"})` |
| `go_file_context` | What a file draws in from the same package. Run immediately after first reading any Go file. | `go_file_context({"file":"/path/to/server.go"})` |
| `go_package_api` | A package's public API surface — most valuable for dependencies or sibling packages you have not read file-by-file. | `go_package_api({"packagePaths":["example.com/internal/storage"]})` |
| `go_symbol_references` | All references to a symbol, to size up blast radius before editing a definition. | `go_symbol_references({"file":"/path/to/server.go","symbol":"Server.Run"})` |
| `go_diagnostics` | Build/analysis problems in the given files — required after every edit. | `go_diagnostics({"files":["/path/to/server.go"]})` |
| `go_rename_symbol` | Rename a symbol and every reference workspace-wide with the same guardrails as LSP rename (refuses changes that would break interface satisfaction). | — |

The chained Read/Edit ordering these tools are built for is in [SKILL.md](../SKILL.md#read-workflow-understand-first) and the edit loop it pairs with.

## Native LSP tool

Claude Code's built-in editor-style integration — a distinct mechanism from the MCP server, worth having in addition to it.

**Enabling:**

1. Export `ENABLE_LSP_TOOL=1` (the feature is off by default).
2. Ensure `gopls` is installed (`go install golang.org/x/tools/gopls@latest`).
3. Install the official `gopls-lsp@claude-plugins-official` marketplace plugin so the tool knows to back Go with gopls.

**Operations** — all addressed by `line`/`character` rather than symbol:

- `goToDefinition`
- `findReferences`
- `hover`
- `documentSymbol`
- `workspaceSymbol`
- `goToImplementation`
- call hierarchy

`goToTypeDefinition` is deliberately missing from that fixed set; type-navigation has no agent-invocable path here (see [features.md](features.md#navigation)). Because these operations need a location up front, they pay off after you already have one — following up a grep hit or a file read — rather than as the opening move of an investigation (that is what `go_search` is for).

**What only this surface gives you:** compiler diagnostics are injected into context automatically after every edit, with no explicit request. The MCP server's `go_diagnostics` requires an explicit call each time.

## Scope and boundaries of the MCP server

The server wraps LSP with explicit limits:

- **Can:** read files and return their contents; run `go` commands for package metadata (these may hit `proxy.golang.org` and write the local module/build caches); write to gopls's own cache and config files; upload telemetry if the user has opted in.
- **Cannot:** write to the source tree except for edits a tool call explicitly returns; make network requests beyond what `go` itself needs to resolve the build.

Either surface reasons only about code present and resolvable in the local build: the workspace plus every dependency pinned in `go.sum`, `replace` directives included. Everything outside that boundary belongs to the `golang-pkg-go-dev` skill (`godig`).
