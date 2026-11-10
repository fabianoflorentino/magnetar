# Dependency Conflicts and Resolution

Most apparent conflicts are misunderstandings of Minimal Version Selection: two modules requiring different versions are not a clash, MVS simply selects the highest required. Reach for `replace`/`exclude` only when a specific version actually breaks the build.

## Diagnosing

```bash
go mod why -m github.com/some/module     # why is this in the build
go list -m github.com/some/module        # which version got selected
go mod graph                             # the full requirement graph
go list -m all                           # what the build resolves to
```

`go mod why -m` names the chain from your code to the module; `go list -m all` shows the selected versions. If the selected version is not the one a dependency asked for, that is MVS doing its job - the question is whether the chosen version actually compiles.

## Resolution strategies

**Force a pattern upgrade of a transitive dep** - add an explicit requirement so the chain cannot drag the version back down:

```bash
go get github.com/transitive/dep@v1.5.0
```

**Block a broken version** - `exclude` redirects any requirement on that version to the next higher available:

```bash
go mod edit -exclude=example.com/pkg@v1.3.0
```

**Use a local fork** - for debugging, or to hold an unmerged patch while upstream sorts itself out:

```go
replace example.com/pkg => ../my-local-fork
```

**Force a version at a path** - when one dependency pins something incompatible, re-point it:

```bash
go mod edit -replace=example.com/pkg@v1.2.0=example.com/pkg@v1.3.1
```

`replace` and `exclude` act only in the main module's `go.mod`. They are ignored when your module is consumed as a dependency, so a published library must never carry a `replace` - resolve the real upgrade before releasing.

## Workflow

1. `go mod graph` and `go mod why -m <module>` to understand the chain.
2. Identify which direct dependency drags in the conflicting version.
3. Try upgrading that direct dependency first: `go get github.com/direct/dep@latest`.
4. Only if that fails, apply `replace` or `exclude` as a temporary, commented fix.
5. `go mod tidy`, then verify with `go build ./...` and `go test ./...`.

## retract - for module authors

Mark bad releases so `go get` will not select them and `go list -m -u` warns:

```go
retract v1.0.0           // contains a critical bug
retract [v1.1.0, v1.2.0] // a range of broken versions
```

Retracted versions stay downloadable - `retract` steers selection, it does not delete history.
