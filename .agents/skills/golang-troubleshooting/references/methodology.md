# Systematic Debugging Methodology

Debug with evidence, not intuition. Every bug yields to the same sequence: pin down the expectation, capture the complete failure, isolate, then trace backward to where the bad value is created before writing any fix.

## Step 1 — State expected vs actual

Before touching code, write down three answers:

- What **should** happen?
- What **actually** happens?
- What **changed** recently?

The third answer is usually the fastest lead. Get it from git:

```bash
git log --oneline -20
git diff HEAD~5
```

When a known-good ancestor exists, binary-search for the breaking commit:

```bash
git bisect start
git bisect bad          # current commit is broken
git bisect good abc123  # known-good commit
```

`git bisect` walks you to the first bad commit.

## Step 2 — Surface the full error

Collect every diagnostic the toolchain offers before reasoning:

```bash
go build ./... 2>&1
go test ./... -v 2>&1
go vet ./...
golangci-lint run ./...
```

Run `golangci-lint` early: it flags unchecked errors and suspicious constructs that are easy to miss on read-through. See the `fabianoflorentino/golang-agent-skills@golang-lint` skill for setup.

## Step 3 — Isolate the failure

Narrow the blast radius before investigating deeper:

```bash
go test -run TestSpecificName -v ./pkg/...      # a single test
go test -count=1 -run TestSpecificName ./pkg/...  # bypass the cache
go build ./pkg/suspect/...
go test -count=10 -run TestSuspect ./pkg/...    # flake check
```

If coverage looks thin, write a test that pins the suspicious behavior first — the test is the reproduction (see [testing-debug.md](./testing-debug.md)).

## Step 4 — Rule out external dependencies

Before blaming your code, verify the surrounding system behaves:

```bash
# Reproduce an API call outside the app
curl -v -X POST https://api.example.com/endpoint \
  -H "Content-Type: application/json" \
  -d '{"key": "value"}'

# Inspect authoritative state (psql, mysql, mongosh, redis-cli, ...)
psql -h localhost -U myuser -d mydb -c "SELECT * FROM orders WHERE id = 123"

# Connectivity and DNS
dig api.example.com
nc -zv api.example.com 443

# Certificates
openssl s_client -connect api.example.com:443 -brief

# Config and credentials
env | grep DATABASE
env | grep API_KEY
```

Common external culprits to check off:

| Area | Typical failure |
| --- | --- |
| API contract | new required field, changed response shape, deprecated endpoint |
| Database | schema drift, missing migration, connection-pool exhaustion |
| Credentials | expired or rotated tokens, keys, certificates |
| DNS | resolution failure, stale cache |
| Load | rate limiting, quota exhaustion |
| Upstream | degraded service, 5xx responses, stale cached data |
| Queues | full queue, consumer lag, rebalancing |
| Configuration | staging-vs-prod differences, feature flags |
| Infrastructure | disk full or read-only, OOM-killed dependencies, firewall or load-balancer changes |
| Time | clock skew breaking JWT validation, TTLs, or scheduled jobs |
| Encoding | locale, timezone, or charset mismatches |
| Containers/K8s | wrong image tag, missing env var, resource limits, misconfigured probes |

## Step 5 — Start from observability

Production debugging begins with whatever the project already exports. Look for `prometheus`, `opentelemetry`, `datadog`, `sentry`, or `elastic/apm` in the code or deployment; if nothing shows up, ask the user what monitoring they run. Useful leads:

- **Prometheus + Grafana** — error-rate spikes, p99 latency, and goroutine or heap growth over time:

  ```promql
  rate(http_requests_total{status=~"5.."}[5m])
  histogram_quantile(0.99, rate(http_duration_seconds_bucket[5m]))
  go_goroutines
  ```

- **Sentry** — grouped exceptions with the full stack trace and breadcrumbs from the first occurrence.
- **Datadog** — APM traces, error tracking, and infrastructure metrics.
- **ELK / Loki** — search structured logs in the failure window, e.g. `level:error AND service:myapp`.
- **Jaeger / Zipkin / OpenTelemetry** — span breakdowns name the slow or failed upstream.

An MCP server wired to any of these enables interactive queries.

## Step 6 — Compare against working code

Find analogous functionality that does **not** exhibit the bug and read it completely — do not skim. Then list every difference: dependencies, configuration, initialization order, error handling, control flow. The divergence usually names the defect.

## Step 7 — Test one hypothesis at a time

- Form a **single, specific** hypothesis with its reasoning attached.
- Add targeted logging or a focused test.
- Change **one thing**, observe, then accept or reject.
- On rejection, revert the change. Never stack fixes on top of failed attempts.

## Step 8 — Trace to the root cause

Fix the bug where the bad value is **created**, not where it surfaces. A nil dereference deep in a handler is usually a constructor that admitted an invalid state:

```go
// Wrong: mask the symptom at the use site
func (s *Server) Handle(w http.ResponseWriter, r *http.Request) {
    if s.db == nil { // hides the real defect
        http.Error(w, "db unavailable", 500)
        return
    }
    // ...
}

// Right: fail fast where the invariant is established
func NewServer(db *sql.DB) *Server {
    if db == nil {
        panic("NewServer: db must not be nil")
    }
    return &Server{db: db}
}
```

When manual tracing stalls, instrument the chain:

```go
func suspectFunction(val string) {
    fmt.Fprintf(os.Stderr, "DEBUG suspectFunction: val=%q\n%s\n", val, debug.Stack())
    // ...
}
```

## Step 9 — Fix and verify

Apply the fix at the source, confirm the failing test passes, then run the full suite to catch regressions.

## Step 10 — Make the bug structurally impossible

After fixing, ask how the same bug could recur and push validation to the layer that owns each invariant:

1. **Entry points** — `New*` constructors and exported functions reject invalid input.
2. **Business logic** — internal functions assert their preconditions.
3. **Runtime guards** — build tags or env checks catch dangerous operations during tests.
4. **Observability** — logs or metrics make the failure class visible immediately if it recurs.

Not every fix needs all four layers; spend them where the bug risks data loss, corruption, or security.

## When stuck — escalation

- **Fewer than three failed attempts** — you misidentified the root cause. Return to step 1 and gather more evidence.
- **Three or more failed attempts** — stop patching. Step back and question the design: is the abstraction itself broken?
- **Every fix reveals a new problem** — you are chasing symptoms. See the red-flag list in [./SKILL.md](./SKILL.md).