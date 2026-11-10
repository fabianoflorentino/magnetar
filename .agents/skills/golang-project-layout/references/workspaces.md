# Go Workspaces for Multi-Package Repositories

`go.work` ties several local modules into one development unit. It mirrors the shape of a released monorepo — modules that would resolve against a proxy resolve against sibling directories instead — without a `replace` directive in every `go.mod`.

## When to use a workspace

Use `go.work` when:

- Several related modules import each other and you develop them together.
- A monorepo holds multiple separate Go modules.
- You want local changes to take effect across module boundaries immediately.

Do not use one for a single-module project, an application that only pulls external dependencies, or a simple standalone tool — `go.work` adds indirection with nothing to coordinate.

## Workspace structure

```
my-monorepo/
├── go.work                # the workspace file
├── pkg/
│   ├── auth/              # module: github.com/user/my-monorepo/pkg/auth
│   │   ├── go.mod
│   │   ├── cmd/
│   │   │   └── auth-server/
│   │   │       └── main.go
│   │   └── internal/
│   │       └── handler/
│   │           └── auth.go
│   └── user/              # module: github.com/user/my-monorepo/pkg/user
│       ├── go.mod
│       ├── cmd/
│       │   └── user-server/
│       │       └── main.go
│       └── internal/
│           └── handler/
│               └── user.go
├── cmd/
│   └── api/               # module: github.com/user/my-monorepo/cmd/api
│       ├── go.mod
│       └── main.go
└── tools/
    └── cli/               # module: github.com/user/my-monorepo/tools/cli
        ├── go.mod
        └── cmd/
            └── mycli/
                └── main.go
```

## Creating a workspace

**Initialize** — `go work init` writes `go.work` with the current toolchain version:

```go
go 1.21

use (
    ./services/auth
    ./services/user
    ./shared/libs
    ./tools/cli
)
```

**Add modules** — `go work use` appends each module directory to the `use` block:

```bash
go work use ./services/auth
go work use ./services/user
go work use ./shared/libs
```

**Import across modules with no `replace` directives** — with `shared/libs` in the workspace, a module requiring it resolves to the local directory automatically:

```go
module github.com/user/my-monorepo/services/user

go 1.21

require github.com/user/my-monorepo/shared/libs v0.0.0
```

## Workspace commands

```bash
go work init              # initialize a new workspace
go work use ./path/to/mod # add a module to the workspace
go work use -rm ./path    # remove a module from the workspace
go work sync              # sync the workspace with module changes
```
