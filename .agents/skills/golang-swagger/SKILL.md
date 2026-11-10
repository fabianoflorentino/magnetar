---
name: golang-swagger
description: "Golang OpenAPI/Swagger documentation with swaggo/swag — annotation comments (@Summary, @Param, @Success, @Router, @Security), swag init code generation, framework integrations (gin, echo, fiber, chi, net/http), security definitions (Bearer/JWT, OAuth2, API key), and struct tags (swaggertype, enums, example, swaggerignore). Apply when adding or maintaining Swagger/OpenAPI docs in a Go project, or when the codebase imports github.com/swaggo/swag, github.com/swaggo/gin-swagger, github.com/swaggo/echo-swagger, github.com/swaggo/http-swagger, or github.com/swaggo/files."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness. Requires go and swag CLI.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "📋"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
        - swag
    install:
      - kind: go
        package: github.com/swaggo/swag/cmd/swag@latest
        bins: [swag]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch mcp__context7__resolve-library-id mcp__context7__query-docs Bash(swag:*) AskUserQuestion Bash(godig:*) Bash(gopls:*) LSP mcp__gopls__*
paths:
  - "**/*.go"
---

**Persona:** You are a Go API documentation engineer. You treat the OpenAPI spec as a contract: annotations that drift from the handlers cost clients compile-time confidence and debugging time.

**Modes:**

- **Build** — adding or refreshing Swagger on a Go project: toolchain, annotations, generation, UI wiring.
- **Audit** — reviewing annotations for gaps, wrong parameter bindings, or missing security markings before they ship.

**When to use:** any task involving Swagger/OpenAPI documentation for Go HTTP APIs. The skill pairs with `golang-rest`, which defines the JSON contract and error shapes these annotations should mirror.

## Setup

Install `swag` and generate the spec from the annotated source:

```text
go install github.com/swaggo/swag/cmd/swag@latest
swag init                                   # reads annotations, writes docs/
swag init -g cmd/api/main.go                # general-info block lives elsewhere
swag fmt                                    # normalize annotation formatting
```

`swag init` walks the tree and emits `docs/docs.go`, `docs/swagger.json`, and `docs/swagger.yaml`. The generated Go file registers the spec via an init hook.

```go
import docs "yourmodule/docs"   // named import: lets you override fields at runtime
import _ "yourmodule/docs"      // blank import: just registers the spec
```

Wire the UI per framework:

| Framework | Wire-up |
| --- | --- |
| Gin | `r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))` |
| Echo | `e.GET("/swagger/*", echoSwagger.WrapHandler)` |
| Fiber | `app.Get("/swagger/*", fiberSwagger.WrapHandler(swaggerFiles.Handler))` |
| net/http | `mux.Handle("/swagger/", httpSwagger.Handler(swaggerFiles.Handler))` |
| Chi | `r.Get("/swagger/*", httpSwagger.Handler(swaggerFiles.Handler))` |

Open `/swagger/index.html`. For multi-environment hosts, use a named import and patch the info before serving:

```go
docs.SwaggerInfo.Host     = os.Getenv("API_HOST")
docs.SwaggerInfo.BasePath = "/api/v1"
```

Full CLI details: [swag CLI reference](references/swag-cli.md).

## General API info

The top-level spec block belongs wherever `swag init` starts its pass (default `main.go`, or the file named by `-g`):

```go
// @title           Payment API
// @version         1.0
// @description     Charge and refund operations.
// @host            api.example.com
// @BasePath        /v1
// @schemes         https

// @contact.name    Platform team
// @contact.email   platform@example.com
// @license.name    Apache 2.0
```

## Operation annotations

Guide every handler with a doc-comment (`// Name godoc`) directly above the swag block — the doc-comment anchors indentation so `swag fmt` stays stable:

```go
// Refund godoc
// @Summary      Refund a charge
// @Description  Reverses a settled charge.
// @Tags         payments
// @Accept       json
// @Produce      json
// @Param        id         path string true "Charge ID"
// @Param        body       body model.RefundRequest true "Refund payload"
// @Success      200 {object} model.Refund
// @Failure      400 {object} api.ErrorResponse
// @Failure      404 {object} api.ErrorResponse
// @Router       /charges/{id}/refund [post]
// @Security     Bearer
func Refund(c *gin.Context) {}
```

`@Param` grammar: `@Param <name> <in> <type> <required> "<description>" [attributes]`

| `in` | Applies to |
| --- | --- |
| `path` | URL segment, e.g. `/payments/{id}` |
| `query` | query string, e.g. `?state=pending` |
| `body` | request payload — must reference a named struct |
| `header` | an HTTP header |
| `formData` | multipart/form field |

Optional attributes: `default(v)`, `minimum(n)`, `maximum(n)`, `minLength(n)`, `maxLength(n)`, `Enums(a,b,c)`, `example(v)`, `collectionFormat(multi)`.

`@Success`/`@Failure` grammar: `@Success <code> {<kind>} <type> "<description>"`

| kind | Meaning |
| --- | --- |
| `{object}` | single named struct |
| `{array}` | slice of named structs |
| `string` / `integer` | primitive |

swag v2 supports generics — `@Success 200 {object} api.Response[model.User]` — and nested composition — `@Success 200 {object} api.Response{data=model.User}`.

## Security definitions

Declare schemes once at the API level, then mark each endpoint with `@Security`.

```go
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization

// @securityDefinitions.basic BasicAuth

// @securityDefinitions.oauth2.authorizationCode OAuth2
// @authorizationUrl https://example.com/oauth/authorize
// @tokenUrl https://example.com/oauth/token
// @scope.read Read access
```

Per endpoint:

```go
// @Security Bearer
// @Security OAuth2[read, write]
// @Security BasicAuth && ApiKeyAuth
```

`&&` means both must succeed; list multiple `@Security` lines to express alternatives.

## Struct tags

Enrich models without touching their Go types:

```go
type CreateUserRequest struct {
    Name   string `json:"name" example:"Jane Doe" minLength:"2" maxLength:"100"`
    Role   string `json:"role" enums:"admin,user,guest" example:"user"`
    Age    int    `json:"age" minimum:"18" maximum:"120"`
    Avatar []byte `json:"avatar" swaggertype:"string" format:"base64"`
    Secret string `json:"-" swaggerignore:"true"`
}
```

| Tag | Effect |
| --- | --- |
| `example` | sample value in the UI |
| `enums` | comma-separated allowed values |
| `swaggertype` | override detected type, e.g. `"primitive,integer"` for `time.Time` |
| `swaggerignore:"true"` | drop the field from the schema |
| `extensions` | OpenAPI extensions: `extensions:"x-nullable,x-deprecated=true"` |

## Common mistakes

| Mistake | Symptom | Fix |
| --- | --- | --- |
| Missing docs import | UI loads but the spec is empty | add `_ "yourmodule/docs"` |
| Stale `docs/` | spec diverges from code | re-run `swag init` with every annotation change |
| `body` param typed as primitive | generation fails; no schema | point at a named struct |
| Protected routes without `@Security` | UI shows an unauthenticated send button | annotate every guarded endpoint |
| General info in the wrong file | spec has no title/host | use `-g` or move the block to `main.go` |
| `{object}` on a `map[string]any` | schema cannot be derived | name a struct or use `swaggertype` |
| Spaces inside bare `@Tags` | tags split into broken groups | quote: `@Tags "user accounts"` |

## Cross-references

- `golang-security` — gating or disabling the Swagger UI outside development.
- `golang-rest` — the JSON/error contract these annotations must document.
- `golang-grpc` — grpc-gateway's own OpenAPI generator when the API is gRPC-first.
