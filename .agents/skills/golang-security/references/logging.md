# Logging Security Rules

Logs are stored indefinitely and shipped to aggregators - every line you write today is a future data breach or compliance finding. Rules:

1. Never log PII such as passwords, tokens, emails, and personal data in plaintext.
2. Sanitize user-controlled input before storing it, to stop log injection.
3. Return generic errors to clients; keep the detailed error server-side.

## Sensitive data in logs - Medium

Logging a whole struct (`%+v`) drags every field along, including the ones that must never persist:

```go
// Bad - password and token are now immortal
log.Printf("user logged in: %+v", user)

// Good - explicit allowlisted fields only
logger.Info("user_login",
    "user_id", user.ID,
    "username", user.Username,
)
```

Enumerate the fields you mean to log. If a struct gains a `Password` field later, a blanket format is the only way it reaches your log pipeline.

## Log injection - Low

User input can smuggle newlines and control characters into logs, forging entries or corrupting the record. Structured logging with `log/slog` treats the message as data and largely neutralizes this; when a field is user-controlled, strip control characters first:

```go
func sanitizeLogInput(input string) string {
    var b strings.Builder
    for _, r := range input {
        if !unicode.IsControl(r) || r == '\n' || r == '\t' {
            b.WriteRune(r)
        }
    }
    return b.String()
}

logger.Info("username_changed", "username", sanitizeLogInput(username))
```

## Error detail leakage - Medium

Sending `err.Error()` to the client hands out internal package paths, SQL, and stack mechanics. Log the detail, return the generic message:

```go
func handleDatabaseError(logger *slog.Logger, err error) error {
    logger.Error("database_error", "error", err.Error())
    return errors.New("database operation failed") // generic to the client
}

func writeError(w http.ResponseWriter, logger *slog.Logger, err error) {
    logger.Error("request_failed", "error", err.Error())
    http.Error(w, "Internal server error", http.StatusInternalServerError)
}
```

## General logger hygiene - Low

- Use structured `log/slog` with a JSON handler for machine-readable, filterable logs.
- Catch misconfigured verbosity early: a debug build with `fmt`-style printf logging will leak whatever it formats.
- Keep request-scoped identifiers (request id, trace id) instead of replaying payloads.
- Enforce a level policy in production (`LevelInfo` by default) so debug handlers cannot turn on PII wholesale.

## Checklist

- No passwords, tokens, or secrets anywhere in logs.
- No PII/PHI in production logs.
- User input scrubbed of control characters before logging.
- Structured JSON logging in place.
- Access logs separated from error logs.
- Log files written with restricted permissions (e.g. `0600`).
- Rotation configured so logs cannot exhaust the disk.
- Generic errors to clients; full details only in internal logs.

## CWE references

- CWE-532 - insertion of sensitive information into log file
- CWE-117 - improper output neutralization for logs
- CWE-209 - information exposure through an error message
- CWE-200 - exposure of sensitive information
- CWE-312 - cleartext storage of sensitive information
