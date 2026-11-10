# Test Naming

## Test functions

`Test` + what is being tested, with underscores separating method and subcase:

```go
func TestParseToken(t *testing.T) { ... }
func TestServer_Handle(t *testing.T) { ... }            // method test
func TestParseToken_InvalidInput(t *testing.T) { ... }  // subcase
```

## Table-driven cases

Subcase names are fully lowercase, descriptive phrases — acronyms included. `input` marks the input, `expected` the outcome:

```go
tests := []struct {
    name         string
    input        string
    expectedCode int
    expectedErr  bool
}{
    {name: "empty input", input: "", expectedCode: 400, expectedErr: true},
    {name: "valid token", input: "abc123", expectedCode: 200},
    {name: "expired token", input: "exp", expectedCode: 401, expectedErr: true},
    {name: "invalid id", input: "???", expectedCode: 400, expectedErr: true},
}

// Against conventions
{name: "valid ID", ...}     // "id", not "ID"
{name: "Empty Input", ...}  // lowercase
```

## Helpers

Test helpers that panic on failure use the `must` prefix: `mustLoadFixture()`, `mustParseURL()`.