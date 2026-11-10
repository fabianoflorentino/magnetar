# Packages, Files & Import Aliasing

## Package names

A package name is **lowercase, single-word**, plural-free, and free of underscores and MixedCaps. Short, evocative, and precise; digits are fine (`oauth2`, `k8s`).

```go
package json
package http
package tabwriter
package oauth2

// Bad
package httpServer    // MixedCaps
package http_server   // underscores
package util          // too generic
package common        // meaningless
package helpers       // what does it help with?
package base          // says nothing
package model         // too vague
```

`util`, `helper`, `common`, `base`, `model` are folder-role words, not abstractions — reach for them and the function really belongs somewhere specific. Packages stay **singular**: `net/url`, not `net/urls`; `go/token`, not `go/tokens`.

### Directory vs package name

Make the directory match the package when possible. Multi-word directories use hyphens; the package drops them (packages cannot contain hyphens):

```
httputil/        → package httputil
auth/            → package auth

user-service/    → package userservice
rate-limit/      → package ratelimit

cmd/api/         → package main     (every cmd/ subdir is main)
internal/auth/   → package auth     (internal/ restricts visibility)
```

Special directories carry toolchain meaning: `cmd/` holds entry points (each subdirectory is `package main`), `internal/` limits imports to the enclosing module, `testdata/` is ignored by the Go tool, `vendor/` holds vendored dependencies.

Package names should not duplicate their exported names — the call site reads `bufio.Reader`, never `bufio.BufReader`. Think about the call site.

## File names

Lowercase, underscores between words:

```
user_handler.go
string_converter.go
http_client_test.go
```

Suffixes with meaning: `_test.go` for test files (excluded from production builds) and `_linux.go`, `_amd64.go` for OS/architecture-specific code identified by build constraints.

## Import aliasing

Alias only on collision — otherwise the import path is the name:

```go
import "github.com/go-chi/chi/v5"

// Collision: one of the two rands must be renamed
import (
    "crypto/rand"
    mrand "math/rand"
)

// Generated code keeps a conventional short alias
import pb "myapp/proto/userpb"

// Unnecessary alias
import f "fmt"
```