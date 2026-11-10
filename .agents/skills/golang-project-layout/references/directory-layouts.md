# Directory Layouts

Three recurring shapes cover most projects. Right-size the layout to the problem: a script stays flat, a service earns layers only when complexity justifies them.

## Universal layout (most projects)

```
project/
├── cmd/                         # entry points — one directory per main package
│   ├── server/                  # app #1
│   │   └── main.go
│   ├── client/                  # app #2
│   │   └── main.go
│   ├── migrate/                 # app #3
│   │   └── main.go
│   ├── cli/                     # app #4
│   │   └── main.go
│   └── worker/                  # app #5
│       └── main.go
├── internal/                    # private application code
│   ├── app/                     # application initialization
│   ├── config/                  # configuration loading
│   ├── handler/                 # HTTP/request handlers
│   ├── model/                   # data models / domain
│   └── service/                 # business logic
├── pkg/                         # public libraries — only if useful to others
│   └── logger/
│       └── logger.go
├── api/                         # API definitions (optional)
│   └── openapi.yaml
├── configs/                     # config files (optional)
│   └── config.yaml
├── scripts/                     # build/deploy scripts (optional)
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
├── .golangci.yml
├── LICENSE
└── README.md
```

## Small project (single binary)

For simple tools, keep it minimal:

```
my-tool/
├── cmd/
│   └── my-tool/
│       └── main.go              # single main package
├── internal/
│   └── core.go                  # application logic
├── go.mod
├── Makefile                     # optional but recommended
├── .gitignore
├── .golangci.yml                # optional
├── LICENSE                      # recommended
└── README.md
```

## Library (reusable code)

```
my-library/
├── example/                     # example
├── logger/                      # public package — root-level directory
│   ├── logger.go
│   └── logger_test.go
├── internal/
│   └── impl/                    # private implementation details
│       └── core.go
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
├── .golangci.yml
├── LICENSE
└── README.md
```

**Library key points:**

- **Public API lives in root-level directories** (e.g. `logger/`).
- **`internal/` holds private implementation only.**
- **Skip `cmd/`** unless you ship example binaries.

## The `cmd/` convention

All `main` packages live in `cmd/`, one directory per binary. Each `cmd/*/main.go` declares `package main`, has its own `func main()`, and holds minimal logic — parse flags, wire dependencies, call `Run()`. Business logic belongs in `internal/` or `pkg/`, never in `cmd/`.

```bash
go build ./cmd/...        # build every main package
go build ./cmd/server     # build one specific binary
```

## Common mistakes

```
myproject/
├── src/                # Go does not use /src (that is the Java pattern)
├── main.go             # do not put main at the root
├── utils/              # generic package name
├── helpers/            # generic package name
└── common/             # generic package name
```

Instead:

```
myproject/
├── cmd/
│   └── myapp/
│       └── main.go     # main in cmd/
├── internal/
│   ├── util/           # specific utility names
│   └── format/         # or domain-specific names
└── pkg/                # only if useful to others
```

Avoid generic catch-all names (`utils`, `helpers`, `common`); prefer names that say which concern they serve. A 100-line CLI gains nothing from this scaffolding — use the small-project shape instead.
