# swag CLI Reference

## Table of Contents

- [`swag init` — generate the spec](#swag-init--generate-the-spec)
- [`swag fmt` — format annotations](#swag-fmt--format-annotations)
- [Swagger UI integration packages](#swagger-ui-integration-packages)
- [Dynamic configuration](#dynamic-configuration)
- [Generics (swag v2)](#generics-swag-v2)
- [Nested composition](#nested-composition)
- [Response headers](#response-headers)
- [Function-scoped structs](#function-scoped-structs)
- [MIME type aliases](#mime-type-aliases)

## `swag init` — generate the spec

```bash
swag init                                      # parse main.go, generate docs/
swag init -g cmd/api/main.go                   # general-info block lives elsewhere
swag init -d ./handlers,./models               # extra directories to parse
swag init --exclude ./vendor,./internal/gen    # skip directories
swag init -ot go,json                          # output only Go and JSON (no YAML)
swag init -q                                   # quiet mode
swag init --parseInternal                      # include internal/ packages
swag init --parseDependency                    # parse vendor/module dependencies
swag init --requiredByDefault                  # mark all struct fields required
swag init -p camelcase                         # property naming: snakecase | camelcase | pascalcase
swag init --tags Users,Products                # only these tags
swag init --tags '!Internal'                   # everything but this tag (! prefix)
swag init --td "[[,]]"                         # custom template delimiters
```

## `swag fmt` — format annotations

```bash
swag fmt                                       # format all annotation comments
swag fmt -d ./handlers                         # format one directory
swag fmt --exclude ./vendor                    # skip directories
```

`swag fmt` needs a standard Go doc comment (`// FuncName godoc`) directly above the first `@` line — without it, indentation cannot be derived and formatting is skipped.

## Swagger UI integration packages

| Framework | Package |
| --- | --- |
| Gin | `github.com/swaggo/gin-swagger` |
| Echo | `github.com/swaggo/echo-swagger` |
| Fiber | `github.com/swaggo/fiber-swagger` |
| Chi / net/http / Gorilla | `github.com/swaggo/http-swagger` |
| Buffalo | `github.com/swaggo/buffalo-swagger` |
| Hertz | `github.com/hertz-contrib/swagger` |

Every integration also pulls in the shared static assets from `github.com/swaggo/files`.

## Dynamic configuration

Override spec values at runtime so one binary serves multiple environments:

```go
import docs "yourmodule/docs" // named import required to reach docs.SwaggerInfo

func main() {
    docs.SwaggerInfo.Title       = "My API"
    docs.SwaggerInfo.Description = "Production API"
    docs.SwaggerInfo.Version     = "2.0"
    docs.SwaggerInfo.Host        = os.Getenv("API_HOST")
    docs.SwaggerInfo.BasePath    = "/api/v1"
    docs.SwaggerInfo.Schemes     = []string{"https"}
}
```

## Generics (swag v2)

Single type parameter:

```go
// @Success 200 {object} api.Response[model.User]
// @Success 200 {array}  api.Response[model.User]
```

Multiple type parameters:

```go
// @Success 200 {object} api.Response[model.User, model.Meta]
```

## Nested composition

Pin or override fields of a documented struct without new Go types:

```go
// @Success 200 {object} api.Envelope{data=model.User}
// @Success 200 {object} api.Envelope{data=[]model.User}
// @Success 200 {object} api.Envelope{data=model.User,meta=api.Pagination}
```

## Response headers

```go
// @Header 200       {string} X-Request-ID "Unique request identifier"
// @Header 200,400   {string} X-Request-ID "Unique request identifier"
// @Header all       {string} X-Request-ID "Present on every response"
```

## Function-scoped structs

Structs declared inside handlers are parseable — reference them by their qualified, function-scoped path:

```go
// @Param req body main.CreateUser.request true "Create user input"
func CreateUser(c *gin.Context) {
    type request struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    }
}
```

## MIME type aliases

| Alias | Content-Type |
| --- | --- |
| `json` | application/json |
| `xml` | application/xml |
| `plain` | text/plain |
| `html` | text/html |
| `mpfd` | multipart/form-data |
| `x-www-form-urlencoded` | application/x-www-form-urlencoded |
| `octet-stream` | application/octet-stream |
| `png` / `jpeg` / `gif` | image/png, image/jpeg, image/gif |
| `event-stream` | text/event-stream |
