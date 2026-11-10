# Protobuf & Code Generation Reference

## Table of Contents

- [Directory layout](#directory-layout)
- [Proto file conventions](#proto-file-conventions)
- [`go_package` conventions](#go_package-conventions)
- [Generation with `protoc`](#generation-with-protoc)
  - [Common `protoc` flags](#common-protoc-flags)
- [Generation with `buf`](#generation-with-buf)
  - [`buf.gen.yaml`](#bufgenyaml)
  - [`buf.yaml`](#bufyaml)
  - [Common `buf` commands](#common-buf-commands)
- [Using the generated code](#using-the-generated-code)

## Directory layout

Organize protos by domain, squeezed by API version. Wrap every operation in `Request`/`Response` messages — a bare `string` argument cannot gain fields later.

```
proto/
├── user/v1/
│   ├── user.proto          # Messages
│   └── user_service.proto  # Service RPCs
├── order/v1/
│   ├── order.proto
│   └── order_service.proto
└── shared/v1/
    └── common.proto        # Pagination, timestamps, shared enums
```

## Proto file conventions

```protobuf
syntax = "proto3";
package mycompany.user.v1;
option go_package = "github.com/mycompany/myservice/gen/user/v1;userv1";

service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
  rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);
  rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);
}

message GetUserRequest {
  string user_id = 1;
}

message GetUserResponse {
  User user = 1;
}

message ListUsersRequest {
  int32 page_size = 1;
  string page_token = 2;
}

message ListUsersResponse {
  repeated User users = 1;
  string next_page_token = 2;
}
```

## `go_package` conventions

- Format: `"import/path;alias"` — the alias becomes the Go package name.
- Keep a lowercase version suffix (`userv1`, `orderv1`).
- Emit into a separate `gen/` tree so generated code never mixes with hand-written sources.

## Generation with `protoc`

```bash
# Basic generation
protoc --go_out=gen --go_opt=paths=source_relative \
       --go-grpc_out=gen --go-grpc_opt=paths=source_relative \
       proto/user/v1/*.proto

# With buf-validate codegen
protoc --go_out=gen --go_opt=paths=source_relative \
       --go-grpc_out=gen --go-grpc_opt=paths=source_relative \
       --validate_out="lang=go:gen" \
       proto/user/v1/*.proto

# Include external import roots
protoc -I proto -I third_party \
       --go_out=gen --go_opt=paths=source_relative \
       --go-grpc_out=gen --go-grpc_opt=paths=source_relative \
       proto/user/v1/*.proto
```

### Common `protoc` flags

| Flag | Purpose |
| --- | --- |
| `--go_out=DIR` | output directory for message types |
| `--go-grpc_out=DIR` | output directory for service stubs |
| `--go_opt=paths=source_relative` | place output relative to the proto source |
| `-I DIR` | add an import path for proto dependencies |
| `--descriptor_set_out=FILE` | emit a binary descriptor (for reflection) |

## Generation with `buf`

`buf` is the modern alternative to hand-rolled `protoc` invocations: it resolves dependencies, lints, and generates from one config file.

### `buf.gen.yaml`

```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen
    opt: paths=source_relative
  - remote: buf.build/grpc/go
    out: gen
    opt: paths=source_relative
```

### `buf.yaml`

```yaml
version: v2
modules:
  - path: proto
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
```

### Common `buf` commands

```bash
buf generate                              # generate from buf.gen.yaml
buf lint                                  # lint proto files
buf breaking --against '.git#branch=main' # backward-compatibility gate
buf dep update                            # refresh module dependencies
buf build                                 # validate protos compile
```

## Using the generated code

```go
import userv1 "github.com/mycompany/myservice/gen/user/v1"

// Server: implement the generated interface
type userServer struct {
    userv1.UnimplementedUserServiceServer
}

// Client: use the generated client
client := userv1.NewUserServiceClient(conn)
resp, err := client.GetUser(ctx, &userv1.GetUserRequest{UserId: "123"})
```

Embed `Unimplemented*Server` (not `Unsafe*Server`) on the server side — it keeps the server compiling when the proto adds new RPCs.
