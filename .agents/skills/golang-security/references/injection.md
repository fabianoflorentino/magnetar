# Injection Security Rules

Injection lets an attacker run code, queries, or commands the developer never intended. Rules:

1. SQL queries must use parameterized placeholders - never concatenate user input.
2. Command execution must use `exec.Command` with separate arguments - never shell interpolation.
3. HTML output must go through `html/template` for automatic escaping.
4. Outbound URLs for SSRF must be validated against an allowlist.

## SQL injection - Critical

String-concatenated SQL is the canonical Go injection. The driver's placeholder syntax differs (`$1` for pgx/lib/pq, `?` for MySQL/SQLite), but the rules are identical:

```go
// Bad
query := fmt.Sprintf("SELECT * FROM users WHERE name = '%s'", input)

// Good - placeholders keep data out of the statement structure
db.QueryRow("SELECT * FROM users WHERE name = $1", input)
db.Exec("DELETE FROM orders WHERE id = $1", orderID)
```

### Dynamic IN clauses

Never join user-provided values into `IN (...)`. Generate numbered placeholders and pass values as arguments:

```go
placeholders := make([]string, len(ids))
args := make([]any, len(ids))
for i, id := range ids {
    placeholders[i] = fmt.Sprintf("$%d", i+1)
    args[i] = id
}
query := fmt.Sprintf("SELECT * FROM users WHERE id IN (%s)", strings.Join(placeholders, ","))
rows, err := db.Query(query, args...)
```

`sqlx` expands and rebinds for you: `sqlx.In("... IN (?)", ids)` followed by `db.Rebind(query)` for PostgreSQL-style `$N`.

### Dynamic identifiers and ORDER BY

Placeholders bind only values. Table names, column names, and SQL keywords stay in the statement text, so a `sort` parameter must never be interpolated directly. Allowlist the identifiers instead:

```go
allowed := map[string]string{
    "created": "created_at",
    "name":    "name",
    "email":   "email",
}
col, ok := allowed[sortCol]
if !ok {
    col = "created_at"
}
query := fmt.Sprintf("SELECT * FROM users ORDER BY %s", col)
```

### Dynamic WHERE filters

Build the filter incrementally, assigning a numbered placeholder to every user-supplied value:

```go
var conditions []string
var args []any
idx := 1

if name != "" {
    conditions = append(conditions, fmt.Sprintf("name = $%d", idx))
    args = append(args, name)
    idx++
}
if minAge > 0 {
    conditions = append(conditions, fmt.Sprintf("age >= $%d", idx))
    args = append(args, minAge)
    idx++
}

query := "SELECT * FROM users"
if len(conditions) > 0 {
    query += " WHERE " + strings.Join(conditions, " AND ")
}
rows, err := db.Query(query, args...)
```

### Prefer sqlx or pgx over raw database/sql

`sqlx` and `pgx` build on prepared statements while adding named parameters, IN expansion, and struct scanning - ergonomics that remove the temptation to fall back to concatenation on complex queries.

## Command injection - Critical

Anything a shell interprets is an injection primitive: pipes, redirects, `&&`, globbing, quoting. Passing arguments separately to `exec.Command` keeps them inert:

```go
// Bad - the shell sees filename as script
exec.Command("sh", "-c", "rm -f /tmp/"+filename)

// Good - rm receives literal arguments, no shell involved
exec.Command("rm", "-f", filepath.Join("/tmp", filename))
```

Apply the same discipline to inputs bound for connection strings, URLs, and template-generated SQL: never place unchecked input inside a string that a parser treats as code.

## Template and code injection - High

`html/template` escapes output by context; raw string formatting and hand-rolled code generation do not:

```go
// Bad - data reaches the browser unescaped
w.Write([]byte(fmt.Sprintf("<div>%s</div>", data)))

// Good - auto-escaped in HTML context
t := template.Must(template.New("safe").Parse("<div>{{.}}</div>"))
t.Execute(w, data)
```

Never generate source from input - whitelist the identifiers (resource names, option keys) and map them onto predefined code paths.

## XSS and HTML tag injection - High

Anything echoed into a page from user input is a script-execution vector. Escape before writing, or let `html/template` do it. As a fallback for hand-built markup, `html.EscapeString` neutralizes tags:

```go
fmt.Fprintf(w, "<div>Welcome, %s!</div>", html.EscapeString(input))
```

Escaping is context-sensitive: an escape for HTML text is not an escape for an attribute, a URL, or a `<script>` block - which is why `html/template` exists.

## Server-Side Request Forgery - High

A server that fetches operator-supplied URLs becomes a foothold into the internal network, including cloud metadata endpoints. Validate scheme, reject internal addresses, and allowlist:

```go
u, err := url.Parse(targetURL)
if u.Scheme != "http" && u.Scheme != "https" {
    return errors.New("invalid scheme")
}
if isInternalIP(u.Hostname()) {
    return errors.New("internal host not allowed")
}
if !allowedHosts[u.Host] {
    return errors.New("domain not allowed")
}
```

Block the classic metadata hosts (`169.254.169.254`, `metadata.google.internal`) explicitly; resolve hostnames through allowlists rather than trusting DNS.

## Unsafe deserialization - Critical

`gob`, `encoding/binary`, and similar binary formats were not hardened against adversarial payloads - they can set arbitrary fields, allocate unbounded memory, or trigger type confusion. JSON is the safe decoding choice for network input because it cannot execute code or build arbitrary objects:

```go
// Bad - gob is not safe for untrusted data
dec := gob.NewDecoder(r.Body)
var user interface{}
dec.Decode(&user)

// Good - JSON, decoded into a typed struct, then validated
dec := json.NewDecoder(r.Body)
var user User
if err := dec.Decode(&user); err != nil { ... }
```

Always decode into a concrete struct and validate the result; reject unknown fields with `DisallowUnknownFields()`.

## XPath injection - High

XPath queries built from input are subject to the same metacharacter problem as SQL. Prefer numeric ids or parse without a query language; at minimum, never embed strings the user controls directly into `//user[@username='...']` style predicates.

## CWE references

- CWE-78 - OS command injection
- CWE-89 - SQL injection
- CWE-94 - code injection
- CWE-79 - cross-site scripting
- CWE-918 - server-side request forgery
- CWE-502 - deserialization of untrusted data
- CWE-20 - improper input validation
