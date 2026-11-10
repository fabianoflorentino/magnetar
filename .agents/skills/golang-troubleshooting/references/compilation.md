# Compilation Issues

## Module problems

When an import resolves oddly or the tree is inconsistent, reset the module metadata:

```bash
go mod tidy             # reconcile go.mod and go.sum with imports
go mod download         # re-fetch declared dependencies
go mod verify           # confirm the module cache is intact
go clean -modcache      # wipe the shared cache (last resort)
go mod why <package>    # which dependency pulls this in?
```

## CGO issues

Cgo links against host C toolchains; failures usually mean a missing compiler, missing pkg-config, or wrong include paths:

```bash
go env CGO_ENABLED            # check cgo is on
export CGO_CFLAGS="-I/usr/local/include"
# pkg-config: brew install pkg-config (macOS) / apt install pkg-config (Ubuntu)
```

## Toolchain and go.mod mismatch

A module declares its minimum Go version in `go.mod`; building with an incompatible toolchain fails early:

```bash
go version            # toolchain in use
go mod edit -go=1.21  # set the module's minimum version
```

A `go` directive newer than the installed toolchain is the most common cause of a "module requires go 1.xx" failure.

## Build tags, files, and packaging

Symptoms like "undefined" symbols or missing functions are often not what they look like:

- **Build tags** — a file constrained by build tags silently drops out of `-tags`-less builds; `go list -tags X -deps` shows which files are in play.
- **OS/arch suffixes** — `foo_windows.go` is invisible on Linux regardless of correctness.
- **Unexported across packages** — a lowercase identifier is simply not visible anywhere else.