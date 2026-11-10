# Application Documentation

## CLI help text

For a CLI, `--help` is the primary documentation. A structured framework (cobra or similar) keeps it consistent and generated:

```go
var rootCmd = &cobra.Command{
    Use:   "mytool",
    Short: "A brief description of mytool",
    Long: `A longer description that explains the tool in detail.

mytool helps you do X, Y, and Z. It connects to your
database and performs analysis on the data.

Environment variables:
  MYTOOL_DB_URL    Database connection string (required)
  MYTOOL_LOG_LEVEL Log level (default: info)
  MYTOOL_TIMEOUT   Request timeout (default: 30s)`,
    Example: `  # Basic usage
  mytool analyze --input data.csv
  mytool serve`,
}
```

See `golang-cli` for CLI application patterns and frameworks.

## Configuration documentation

Document every configuration source and its precedence — defaults, config file, environment variables, flags, with later sources overriding earlier ones:

| Variable | Description | Default | Required |
| --- | --- | --- | --- |
| `MYTOOL_DB_URL` | Database connection string | — | Yes |
| `MYTOOL_LOG_LEVEL` | Log verbosity | `info` | No |
| `MYTOOL_PORT` | HTTP server port | `8080` | No |

```yaml
database:
  url: postgres://localhost/mydb
  max_connections: 25
server:
  port: 8080
  read_timeout: 30s
logging:
  level: info
```

## Architecture decision records

For decisions that will be re-litigated, keep numbered ADRs under `docs/architecture/`:

```
docs/
  architecture/
    0001-use-postgres-as-primary-store.md
    0002-event-driven-architecture.md
    README.md
```

Each ADR records context, the decision, and its consequences:

```markdown
# Use PostgreSQL as Primary Store

## Context

We need a persistent data store that supports...

## Decision

We use PostgreSQL because...

## Consequences

- Positive: ACID transactions, rich query language
- Negative: operational overhead, connection management
```

## API documentation

### REST — OpenAPI via swaggo/swag

Annotate handlers and generate OpenAPI straight from the source:

```go
// @Summary Get user by ID
// @Description Returns a single user
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} User
// @Failure 404 {object} ErrorResponse
// @Router /users/{id} [get]
func GetUser(w http.ResponseWriter, r *http.Request) {
```

```bash
go tool swag init -g cmd/server/main.go -o docs/swagger
```

Serve `docs/swagger` with Swagger UI or Redoc.

### Event-driven — AsyncAPI

Message-based APIs (Kafka, NATS, RabbitMQ) get their contract from an AsyncAPI spec:

```yaml
asyncapi: "2.6.0"
info:
  title: Order Events
  version: "1.0.0"
channels:
  orders/created:
    publish:
      message:
        payload:
          type: object
          properties:
            orderId:
              type: string
            amount:
              type: number
```

### gRPC — protobuf as documentation

Protobuf files are the contract and the docs at once; comment messages and RPCs generously:

```protobuf
// UserService manages user accounts.
service UserService {
  // GetUser retrieves a user by id.
  // Returns NOT_FOUND if the user does not exist.
  rpc GetUser(GetUserRequest) returns (User);
}

message User {
  // Unique identifier for the user (UUID v4).
  string id = 1;
  // User's email address (must be unique).
  string email = 2;
}
```

Use [buf](https://buf.build/) to lint and detect breaking changes:

```bash
buf lint
buf breaking --against '.git#branch=main'
```

For REST plus gRPC, [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) serves both from the same definition.

### Which format fits

| API style | Format | How it is produced |
| --- | --- | --- |
| REST with Go handlers | OpenAPI 3.x | swaggo/swag annotations |
| REST with a framework | OpenAPI 3.x | Framework tooling (e.g. huma) |
| gRPC services | Protobuf | Proto files are the source of truth |
| gRPC + REST gateway | Protobuf + OpenAPI | grpc-gateway generates OpenAPI |
| Events / message queues | AsyncAPI | Manual or code generation |
| GraphQL | SDL schema | The schema is the docs |