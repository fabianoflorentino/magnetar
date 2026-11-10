# Capability to tool-path mapping

Every gopls capability below is cross-referenced against the three surfaces available to an agent: the CLI, the MCP server, and the native `LSP` tool. A `—` in a cell means that surface has no route to that capability. Deep-dives for the refactor and transform rows live in [features.md](features.md).

## Navigation

| Capability | CLI | MCP tool | Native `LSP` op |
| --- | --- | --- | --- |
| Workspace layout (module / workspace / GOPATH) | `gopls stats` | `go_workspace` | — |
| Fuzzy symbol lookup across the workspace | `gopls workspace_symbol <query>` | `go_search` | `workspaceSymbol` |
| Jump to a symbol's declaration | `gopls definition f:l:c` | — (use `go_file_context`/`go_package_api`) | `goToDefinition` |
| Underlying type definition | — (unsupported) | — | — (type navigation is absent from the fixed op list) |
| Every reference to a symbol | `gopls references f:l:c` | `go_symbol_references` | `findReferences` |
| Implementations / satisfied interfaces | `gopls implementation f:l:c` | — | `goToImplementation` |
| Full subtype/supertype graph | — (not implemented) | — | Type Hierarchy |
| Static callers/callees of a function | `gopls call_hierarchy f:l:c` | — | Call Hierarchy |
| Syntactic selection expand/shrink | — (unsupported) | — | `selectionRange` (editor gesture only, no agent hook) |

## Discovery

| Capability | CLI | MCP tool | Native `LSP` op |
| --- | --- | --- | --- |
| Outline of one file's declarations | `gopls symbols <file>` | `go_file_context` | `documentSymbol` |
| Public API surface of a package | — | `go_package_api` | (hover per symbol) |
| A file's in-package dependencies | — | `go_file_context` | — |
| Hover type, doc text, size/offset | — | — | `hover` |
| Signature help | `gopls signature f:l:c` | — | signature help |
| Identifier highlight set | `gopls highlight f:l:c` | — | `documentHighlight` (not in the fixed op list) |
| Semantic token stream | `gopls semtok <file>` | — | `semanticTokens` (editor-automatic only) |
| Collapsible regions | `gopls folding_ranges <file>` | — | `foldingRange` (editor-automatic only) |
| Links extracted from docs/imports | `gopls links <file>` | — | `documentLink` (editor-automatic only) |

## Rigor: diagnostics and safety

| Capability | CLI | MCP tool | Native `LSP` op |
| --- | --- | --- | --- |
| Compiler + analyzer findings | `gopls check <file>` | `go_diagnostics` | pushed automatically after each edit |
| Reachability of known vulnerabilities in the current build | — | `go_vulncheck` | — |
| Safe rename (symbol, receiver, package move, signature) | `gopls rename -w f:l:c NewName` | `go_rename_symbol` | rename |

## Editing

| Capability | CLI | MCP tool | Native `LSP` op |
| --- | --- | --- | --- |
| Import fix/org | `gopls imports -w <file>` | — | `source.organizeImports` code action |
| Formatting | `gopls format -w <file>` | — | `textDocument/formatting` |
| Refactors (extract/inline/fill/rewrite, see [features.md](features.md#transformation)) | `gopls codeaction -kind=<kind> -exec -w <file>` | — | code action |
| Generate a test for a function | `gopls codelens -exec <file:line> "..."` (via `source.addTest`) | — | code action / code lens |
| Rendered package documentation (incl. internal packages) | — | — | `source.doc` code action → browser report |
| Free symbols of a selection (extract inputs) | — | — | `source.freesymbols` code action → browser report |
| Assembly listing for a function | — | — | `source.assembly` code action → browser report |
| Split-package dependency planning | — | — | `source.splitPackage` code action → browser report |
