---
name: golang-rest
description: "Golang REST API development with net/http — resource and method design, JSON contract and encoding, RFC 9457 problem+json errors, request validation, pagination (cursor vs offset), idempotency keys, content negotiation, versioning with deprecation headers, proper http.Server timeouts, and httptest-based testing. Apply when building, reviewing or debugging a REST/JSON API in Go, or when the codebase uses net/http routing (Go 1.22+ patterns) or chi, gin, echo handlers."
user-invocable: false
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🌐"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent
paths:
  - "**/*.go"
---

# Golang REST API

**Persona:** You are a Go backend engineer. You design REST APIs that are easy to consume, predictable to debug, and safe under production load.

**Modes:**

- **Build** — greenfield API or new endpoint: resources, routing, JSON contract, error handling, testing.
- **Audit** — reviewing an existing API for consistency of status codes, error shape, validation, and safety.
- **Debug** — a failing endpoint: narrowing down the handler, data, or transport layer.

**When to use:** any task centered on a REST/JSON API. For gRPC or GraphQL transports use `golang-grpc` / `golang-graphql` instead. Always load `golang-testing`; load `golang-swagger` when the project documents its API with OpenAPI.

## Resources and methods

- Use **plural nouns** for collections (`/users`, `/users/{id}`); keep item identifiers in the path, never in the body.
- Leave **query params** for filtering, sorting and pagination (`/users?status=active&sort=-created_at`).
- Map methods canonically: `GET` read, `POST` create (or trigger an action with an explicit verb path like `/users/{id}/activate`), `PUT` replace, `PATCH` partial update, `DELETE` remove.
- Return `201 Created` with a `Location` header on resource creation; `204 No Content` for deletions and updates that return nothing; `404` only for missing resources, not for forbidden ones.
- Keep the transport free of business rules: handlers thin, logic in services/use-cases, persistence behind interfaces.

## Routing

- Prefer the **stdlib** `net/http` multiplexer with Go 1.22+ patterns (`mux.HandleFunc("GET /users/{id}", ...)` with `r.PathValue("id")`) — no dependency for simple APIs.
- Reach for a router (chi, gin, echo) only for middleware ecosystems, group/param validation shortcuts, or large handler counts; keep one router style per codebase.
- Create the `http.Server` explicitly and never start with a leaked server: set `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`, and `MaxHeaderBytes`.

## JSON contract

- Define request/response as **explicit structs** with json tags; never expose `map[string]any` (loses typing and turns numbers into `float64`).
- Handle body size and one decode once: `r.Body = http.MaxBytesReader(w, r.Body, limit)` then a single `json.Decoder`/`DisallowUnknownFields()` decode; reject trailing garbage and keep `io.Closer` closed.
- For responses prefer canonical `json.Encoder w/ SetEscapeHTML(false)` where the HTML-escaping default would corrupt payloads (`<`, `>`, `&` in URLs or user data).
- Dates/times: wire format is RFC 3339 UTC (`time.Time` marshals this by default); never send locale or timezone-ambiguous strings.
- Numeric IDs: prefer `int64`/`uint64` or string ids over `int`; document overflow and platform-width risk in an audit.

## Status codes and errors

- Use **RFC 9457 (`Content-Type: application/problem+json`)** for all non-2xx responses; one error shape everywhere, with a stable, machine-readable `code` (e.g. `validation_error`, `rate_limited`, `not_found`).
- Keep `status` echoing the HTTP code, `title` short, `detail` actionable for clients, and only the fields needed per error; never leak stack traces, SQL, or internal ids into `detail`.
- `400` malformed request, `401` unauthenticated, `403` authenticated-but-forbidden, `404` missing, `405` wrong method (set `Allow`), `409` conflict, `422` well-formed-but-invalid (validation), `429` too many requests (set `Retry-After`), `500` unexpected.
- Do not let panics reach the transport: a recovery middleware converts them to `500` problem+json and logs with `golang-observability` conventions.
- Always return after writing an error (`http.Error` is not a return; see `golang-pitfalls-standard-library` mistake #80).

## Validation

- Validate **before** touching persistence; on failure return `422` with one problem detail per offending field (`field`, `reason`).
- Prefer a validation library only when the schema is large (e.g. go-playground/validator with `binding` tags); stdlib checks suffice for most handlers.
- Normalize input early: trim strings, parse once, and never re-validate in multiple layers with divergent rules.

## Pagination, filtering, sorting

- Default to **cursor pagination** for large or high-churn collections (stable under inserts/deletes); keep offset-based for small admin/backoffice lists with capped `limit` (e.g. 100 max).
- Return pagination metadata in a consistent envelope or `Link`/`next` field: `{ "items": [...], "next": "/users?cursor=...", "has_more": bool }`; keep results ordering deterministic (stable sort key).
- Filtering and sorting should be **whitelisted** per-resource: unknown fields → `422`, never silent filter-by-original-query error.

## Idempotency and retries

- Accept `Idempotency-Key` on state-changing `POST`/`PATCH`; store the key with the operation result and reuse it for replays (return the original outcome), expiring keys after a bounded window.
- Make `PUT`/`DELETE` naturally idempotent (replace semantics; delete by id returns `204` on repeat).
- If a client retries on `429`/`503` timeouts, your endpoints must be safe to run concurrently — that is a server-side guarantee, not a client courtesy.

## Content negotiation and versioning

- Honor `Accept`: serve `application/json` by default, `406 Not Acceptable` for unsupported types; honor `Content-Type` on writes and do not guess.
- Version the URL path (`/v1/users`) rather than headers; document breaking changes and send **`Deprecation` and `Sunset` headers** before retiring a version.
- Keep deprecated routes functionally identical until `Sunset` — no silent behavior changes in a deprecated version.

## Security and operations

- Delegate authz to middleware (JWT/session verification, scopes) and keep handlers unaware of token parsing; see `golang-security` for the full checklist.
- Set CORS, `Strict-Transport-Security`, cache headers (`Cache-Control`, `ETag`/`If-None-Match` for 304s) deliberately; cookies only if you control the client, with `HttpOnly`, `Secure`, `SameSite`.
- Add the request context (`context.WithTimeout`, request `r.Context()` cancellation) to all downstream calls so a client disconnect cancels DB/HTTP work.
- Expose `health`/`ready` endpoints and structured logging/metrics per `golang-observability`.

## Testing

- Test every handler with `httptest.NewRecorder` + `httptest.NewRequest` in table-driven form: happy path, validation, 404/409/422, authz, and idempotency replay.
- Assert status code, `Content-Type`, and the problem+json shape contractually; use golden files for full response bodies only when justified.
- Add at least one test that exercises the real `http.Server` (golang.org/x/net or `httptest.NewServer`) for timeout and concurrency behavior.

## Cross-references

- `golang-error-handling` — idiomatic error wrapping and `errors.Is`/`As` for handler mappings.
- `golang-testing` — full testing approach for handlers and services.
- `golang-security` — authz, injection, secrets, cookies, transport security.
- `golang-swagger` — OpenAPI annotations matching this contract (swaggo/swag).
- `golang-observability` — structured logs, metrics, traces per endpoint.
- `golang-pitfalls-standard-library` — `http.Error` returns and HTTP client/server timeouts.
- `golang-database` — persistence behind the handler layer (if the API is data-backed).
