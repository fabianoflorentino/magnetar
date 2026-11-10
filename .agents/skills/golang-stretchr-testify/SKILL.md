---
name: golang-stretchr-testify
description: "Comprehensive guide to stretchr/testify for Golang testing. Covers assert, require, mock, and suite packages in depth. Use when writing tests with testify, creating mocks, setting up test suites, or choosing between assert and require. Covers testify assertions, mock expectations, argument matchers, call verification, suite lifecycle, and advanced patterns like Eventually, JSONEq, and custom matchers. Apply when the codebase imports github.com/stretchr/testify."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "✅"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
        - gotests
    install:
      - kind: go
        package: github.com/cweill/gotests/...@latest
        bins: [gotests]
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent WebFetch mcp__context7__resolve-library-id mcp__context7__query-docs Bash(gotests:*) AskUserQuestion Bash(godig:*) Bash(gopls:*) LSP mcp__gopls__*
paths:
  - "**/*.go"
---

**Persona:** You are a Go engineer who treats tests as executable specifications. You write tests to pin behavior and make failures self-explanatory — coverage is a byproduct, not the goal.

**Modes:**

- **Write** — adding tests or mocks to a codebase.
- **Review** — auditing existing tests for assertion misuse, missing mock verification, and `assert`/`require` confusion.

**When to use:** any task involving stretchr/testify. testify wraps, never replaces, the standard `testing` package — `*testing.T` remains the entry point. General testing habits live in `golang-testing`.

## assert vs require

Both expose the same assertions; they differ only in failure handling:

- `assert.*` records the failure and keeps going — one run shows every broken assertion.
- `require.*` calls `t.FailNow()` — for preconditions where continuing would panic or mislead.

Bind them once and name them by role:

```go
is := assert.New(t)
must := require.New(t)

cfg, err := ParseConfig("testdata/valid.yaml")
must.NoError(err) // stop: cfg would be unusable otherwise
is.Equal("production", cfg.Environment)
is.Equal(8080, cfg.Port)
```

Rule: `require` for setup and error checks, `assert` for verification — never interleaved at random.

## Core assertions

```go
is.Equal(expected, actual)                 // DeepEqual + exact types
is.EqualValues(expected, actual)           // coerce before comparing
is.NotEqual(a, b)
is.Nil(x) / is.NotNil(x)
is.Empty(coll) / is.NotEmpty(coll) / is.Len(coll, n)
is.True(cond) / is.False(cond)
is.Contains(text, sub)                     // strings, slices, map keys
is.Greater(a, b) / is.Less(a, b) / is.InDelta(x, y, tol)
is.Error(err) / is.NoError(err)
is.ErrorIs(err, ErrNotFound)               // walks the chain
is.ErrorAs(err, &target)
is.Zero(v) / is.Positive(v)
is.IsType(&User{}, v) / is.Implements((*io.Reader)(nil), v)
```

Argument order is always `(expected, actual)` — swap it and the diff output reads backwards.

## Advanced assertions

```go
is.ElementsMatch([]string{"b", "a"}, result)        // order-insensitive
is.JSONEq(`{"name":"alice"}`, `{ "name": "alice" }`) // whitespace/key-order agnostic
is.WithinDuration(t0, t1, 5*time.Second)
is.Regexp(`^user-[a-f0-9]+$`, id)

is.Eventually(func() bool {
    s, _ := client.JobStatus(jobID)
    return s == "completed"
}, 5*time.Second, 100*time.Millisecond)

is.EventuallyWithT(func(c *assert.CollectT) {
    resp, err := client.Order(orderID)
    assert.NoError(c, err)
    assert.Equal(c, "shipped", resp.Status)
}, 10*time.Second, 500*time.Millisecond)
```

## testify/mock

Mocks isolate the unit under test. Embed `mock.Mock`, implement methods by delegating to `m.Called(...)`, and always verify with `AssertExpectations(t)`:

```go
type TokenStoreMock struct{ mock.Mock }

func (m *TokenStoreMock) Save(id string, tok Token) error {
    args := m.Called(id, tok)
    return args.Error(0)
}
```

Matchers: `mock.Anything`, `mock.AnythingOfType("int")`, `mock.MatchedBy(func(v any) bool)`. Modifiers: `.Once()`, `.Times(n)`, `.Maybe()`, `.Run(func)` for side effects. Full definition, matcher, and verification material: [mock reference](references/mock.md).

## testify/suite

Suites group tests with shared lifecycle:

```text
SetupSuite → (SetupTest → TestXxx → TearDownTest)* → TearDownSuite
```

```go
type TokenSuite struct {
    suite.Suite
    store   *TokenStoreMock
    service *TokenService
}

func (s *TokenSuite) SetupTest() {
    s.store = new(TokenStoreMock)
    s.service = NewTokenService(s.store)
}

func (s *TokenSuite) TestGenerate() {
    s.store.On("Save", mock.Anything, mock.Anything).Return(nil)
    token, err := s.service.Generate("user-42")
    s.NoError(err)
    s.NotEmpty(token)
    s.store.AssertExpectations(s.T())
}

func TestTokenSuite(t *testing.T) {
    suite.Run(t, new(TokenSuite))
}
```

Suite methods behave like `assert`; escalate specific checks via `s.Require().NotNil(obj)`.

## Common mistakes

| Mistake | Failure mode | Fix |
| --- | --- | --- |
| Missing `AssertExpectations` | expectations pass silently | call it at the end of each test |
| `is.Equal(ErrNotFound, err)` | fails on wrapped errors | `ErrorIs` to walk the chain |
| Swapped `(expected, actual)` | backwards diff output | keep `(expected, actual)` |
| `assert` for guards | nil dereference after failure | `require` |
| No `suite.Run(...)` launcher | runs zero tests silently | add the runner function |
| Comparing pointer values | compares addresses | dereference or `EqualExportedValues` |

## Linters

Run `testifylint` to catch argument-order inversions and assert/require misuse; configuration lives in `golang-lint`.

## Cross-references

- `golang-testing` — table-driven tests, golden files, and CI integration.
- `golang-lint` — testifylint configuration.
- `golang-pkg-go-dev` / `golang-gopls` — package facts and call-site navigation.
