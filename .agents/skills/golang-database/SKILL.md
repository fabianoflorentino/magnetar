---
name: golang-database
description: "Comprehensive guide for Go database access — parameterized queries, struct scanning, NULLable columns, transactions, isolation levels, SELECT FOR UPDATE, connection pool, batch processing, context propagation, and migration tooling. Use when writing, reviewing, or debugging Golang code that interacts with PostgreSQL, MariaDB, MySQL, or SQLite; for database testing; or for questions about database/sql, sqlx, or pgx. Does NOT generate database schemas or migration SQL."
user-invocable: true
license: MIT
compatibility: Designed for Claude Code, Codex or similar harness, and for projects using Golang.
metadata:
  author: fabianoflorentino
  version: "1.0.0"
  openclaw:
    emoji: "🗄"
    homepage: https://github.com/fabianoflorentino/golang-agent-skills
    requires:
      bins:
        - go
    install: []
allowed-tools: Read Edit Write Glob Grep Bash(go:*) Bash(golangci-lint:*) Bash(git:*) Agent AskUserQuestion
paths:
  - "**/*.go"
---

# Database access in Go

**Persona:** You are a Go backend engineer writing safe, explicit, observable database code. SQL is a first-class language — no ORMs, no magic — and integrity problems are caught at the boundary, not deep in the application.

**Modes:**

- **Write** — new repository functions, query helpers, transaction wrappers. First grep the codebase for existing query patterns and naming, then follow the patterns in the skill. Sequential.
- **Review/Debug** — audit or fix existing DB code. One sub-agent scans for `rows.Close` misses, un-parameterized queries, missing `*Context` variants, and skipped error checks while you read the business logic. Parallel.

**When to use:** any Go code touching Postgres/MySQL/MariaDB/SQLite, `database/sql`/`sqlx`/pgx, database testing, or transaction design. It does not write schemas or migration SQL. Injection protection lives in `golang-security`; context rules in `golang-context`.

## Library choice

| Library | Strengths | Scanning | Postgres-specific |
| --- | --- | --- | --- |
| `database/sql` | Portability, minimal deps | Manual `Scan` | No |
| `sqlx` | Multi-database ergonomics | `StructScan` | No |
| `pgx` | Postgres (faster), COPY/LISTEN/arrays | `pgx.RowToStructByName` | Yes |
| GORM/ent | — | **Avoid** | Abstracted away |

Why not ORMs: query generation you cannot predict (invisible N+1), magic hooks (`BeforeCreate`, `AfterUpdate`) that defeat debugging, migrations coupled to app code, and an API that really does cost more to learn than SQL. The abstraction leaks anyway.

## Queries

**Parameterized placeholders, always.** Never concatenate user input into a SQL string.

```go
// PostgreSQL
err := db.GetContext(ctx, &user, "SELECT id, name, email FROM users WHERE email = $1", email)
// MySQL
err := db.GetContext(ctx, &user, "SELECT id, name, email FROM users WHERE email = ?", email)
```

- **Dynamic `IN` clauses:** `sqlx.In("SELECT ... WHERE id IN (?)", ids)` then `db.Rebind(query)`.
- **Dynamic identifiers** (sort columns, table names): never interpolate from user input — validate against an allowlist first, then interpolate a known-good value.
- **No rows returned:** use `Exec`, not `Query` — `Query` hands back a `*Rows` that must be closed, and forgetting leaks the connection back to the pool.

## Scanning and NULL columns

Tag exported struct fields (`db:"column_name"`, `json:"..."`), and for NULLable columns use pointer fields (`*string`, `*time.Time`) — they scan and JSON-marshal cleanly with both sqlx tags and pgx `RowToStructByName`. See the scanning reference for every approach.

## Error handling

Distinguish "not found" from real failure early, at the repository boundary:

```go
err := db.GetContext(ctx, &user, "SELECT id, name FROM users WHERE id = $1", id)
if err != nil {
    if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrUserNotFound // domain error, not a technical one
    }
    return nil, fmt.Errorf("querying user %s: %w", id, err)
}
```

| Symptom | Detect | Action |
| --- | --- | --- |
| Row missing | `errors.Is(err, sql.ErrNoRows)` | Translate to a domain error |
| Unique violation | driver-specific code | Conflict error |
| Connection refused | `db.PingContext` fails | Fail fast, log, backoff retry |
| Serialization failure | PG code `40001` | Retry the whole transaction |
| Canceled | `errors.Is(err, context.Canceled)` | Stop and propagate |

Always close rows and always check `rows.Err()` after iteration:

```go
rows, err := db.QueryContext(ctx, "SELECT id, name FROM users")
if err != nil { return fmt.Errorf("querying users: %w", err) }
defer rows.Close()

for rows.Next() { /* ... */ }
if err := rows.Err(); err != nil { return fmt.Errorf("iterating users: %w", err) }
```

## Context everywhere

Always the `*Context` variants — `QueryContext`, `ExecContext`, `GetContext`. A non-context query runs to completion even after the client disconnects.

## Transactions, isolation, locking

Multi-statement writes are transactions. Default `READ COMMITTED` is usually not enough for money or inventory; that is what isolation levels and `SELECT ... FOR UPDATE` are for. See the transactions reference for boundaries, deadlock avoidance, and the serialization-retry loop.

## Connection pool

```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(5 * time.Minute)
db.SetConnMaxIdleTime(1 * time.Minute)
```

Sizing guidance and formulas in the performance reference.

## Batch operations

Batch in reasonable sizes — not row-by-row (round-trip explosion) and not millions at once (row locks and memory). Tune the chunk size against the driver and load.

## Migrations

Schema changes need human legs: data volumes, existing indexes, foreign keys, production constraints. Version migrations in source control and apply them through CI/CD with a real tool — `golang-migrate`, Flyway, or Atlas. Never let hand-rolled or AI-generated migration SQL near your database.

## Hidden SQL features

Do not rely on triggers, views, materialized views, stored procedures, or row-level security in application code. Invisible side effects are undebuggable; keep SQL explicit, in Go, version-controlled, testable.

## Schema design

This skill deliberately does not write schemas. An AI schema that looks right on toy data creates hotspots, lock contention, and missing indexes under real load — schema design needs volume, access-pattern, and constraint knowledge the generator does not have. Use real tooling and human review.

## Reference files

- [`references/scanning.md`](./references/scanning.md) — struct tags, NULL columns, JSON marshaling.
- [`references/transactions.md`](./references/transactions.md) — boundaries, isolation, deadlock prevention, `SELECT FOR UPDATE`.
- [`references/testing.md`](./references/testing.md) — mocks, container-backed integration, fixtures, teardown.
- [`references/performance.md`](./references/performance.md) — pool sizing, batching, indexing, query tuning.

## External references

- [database/sql tutorial](https://go.dev/doc/database/)
- [sqlx](https://github.com/jmoiron/sqlx)
- [pgx](https://github.com/jackc/pgx)
- [golang-migrate](https://github.com/golang-migrate/migrate)

## Cross-references

- `golang-security` — SQL injection prevention.
- `golang-context` — cancellation/deadline propagation into DB calls.
- `golang-error-handling` — wrapping and single-handling rules for SQL errors.
- `golang-testing` — integration test patterns behind build tags.
