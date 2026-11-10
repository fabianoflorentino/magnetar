# Go Workspaces (go.work)

Workspaces bundle several local modules into one development build, replacing the `replace`-directive dance across independent repos.

## go.work vs go.mod

| Scenario | Use |
| --- | --- |
| Single module project | `go.mod` |
| Developing multiple related local modules | `go.work` |
| Monorepo with separate Go modules | `go.work` |
| Testing local changes across module boundaries | `go.work` |
| Published library consumed by others | `go.mod` |

## Commands

```bash
go work init                    # create a workspace
go work use ./services/auth     # add a module
go work use -rm ./old-module    # drop a module
go work sync                    # reconcile with module changes
```

`go work use` records each module's path in `go.work`; `go work sync` aligns the workspace's requirements with the modules it contains.

## Key points

- Workspaces resolve local modules automatically, so during development you skip `replace` directives entirely - edits in `./services/auth` are immediately visible to consumers in the same workspace.
- `go.work` is a development convenience only; it does not affect how external consumers resolve a published module.
- Do not commit `go.work.sum` - keep it out of version control and out of CI.
- Structure: a single multi-module repo (monorepo) or several checkouts wired together both work; see `golang-project-layout` for directory layout examples.
