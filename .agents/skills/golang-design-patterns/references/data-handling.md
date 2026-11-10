# Data Handling: Iterators and Streaming

## Iterate instead of materialize

Loading a million rows into a slice is a memory decision you usually want reversed. Go 1.23+ `range`-able iterators give lazy, one-at-a-time processing:

```go
func AllUsers(ctx context.Context, db *sql.DB) iter.Seq2[User, error] {
    return func(yield func(User, error) bool) {
        rows, err := db.QueryContext(ctx, "SELECT id, name, email FROM users")
        if err != nil {
            yield(User{}, err)
            return
        }
        defer rows.Close()

        for rows.Next() {
            var u User
            if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
                yield(User{}, err)
                return
            }
            if !yield(u, nil) {
                return
            }
        }
    }
}
```

The consumer stops pulling, the iterator returns, and the rows close — memory stays flat no matter how many entries the query returns.

## Stream large transfers

When data travels straight from storage to an HTTP response, stream both ends so the working set never grows with the payload. Writing the JSON envelope by hand keeps it constant-memory:

```go
func (h *Handler) ExportUsers(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte("["))

    first := true
    for user, err := range h.repo.AllUsers(r.Context()) {
        if err != nil {
            slog.Error("streaming user", "error", err)
            return
        }
        if !first {
            w.Write([]byte(","))
        }
        json.NewEncoder(w).Encode(user)
        first = false
    }

    w.Write([]byte("]"))
}
```

The principle extends to copying a remote file to disk or piping database rows through a transform: pick the lazy shape and let backpressure be the load control.