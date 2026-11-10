# Third-Party Data Leak Rules

Error trackers, analytics, and observability pipes are external systems too - data shipped to them leaves your trust boundary. Rules:

1. Filter PII before transmitting anything to a third-party service.
2. Error-tracking hooks must redact request data (`BeforeSend`), never pass it through untouched.

## Common exposure points

| Service | Risk |
| --- | --- |
| Sentry, Airbrake, Bugsnag, Rollbar, Honeybadger | error events carry request context and breadcrumbs |
| New Relic, Datadog, OpenTelemetry | traces and metrics can embed attribute values |
| Google Analytics, Segment | event properties may be PII |
| Algolia, Elasticsearch | query payloads can leak sensitive documents |
| BigQuery, ClickHouse | ingest pipelines may receive unfiltered rows |

The concern is uniform: whatever function call looks convenient to pass "the whole object" to is where the leak lives.

## Error tracking - Medium

`CaptureException(err)` without configuration sends request headers and context along with the stack:

```go
// Bad - full request context leaves the building
sentry.CaptureException(err)

// Good - a BeforeSend hook strips sensitive headers before upload
sentry.Init(sentry.ClientOptions{
    Dsn:         os.Getenv("SENTRY_DSN"),
    BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
        if event.Request != nil {
            delete(event.Request.Headers, "Authorization")
            delete(event.Request.Headers, "Cookie")
        }
        return event
    },
})
```

Apply the same hook logic wherever the SDK allows scrubbing: age off breadcrumbs, mask URL query strings, and never attach the raw request body.

## Analytics and monitoring - Medium

Analytics events are not free-form logging. An email, phone number, or street address in an event property is PII shipped to a data processor:

```go
// Bad - personally identifying fields
analytics.Track("user_signed_up", analytics.Properties{
    "email":   user.Email,
    "phone":   user.Phone,
    "address": user.Address,
})

// Good - identifiers and business facts only
analytics.Track("user_signed_up", analytics.Properties{
    "user_id":        user.ID,
    "plan":           user.Plan,
    "country":        user.CountryCode,
})
```

When correlation matters, send non-reversible identifiers (a keyed hash) instead of the raw value.

## Data filtering layer

Centralize the scrub list so every export path passes through the same decisions:

```go
type DataFilter struct {
    sensitive []string
}

func NewDataFilter() *DataFilter {
    return &DataFilter{sensitive: []string{
        "password", "token", "secret", "key", "email",
        "phone", "address", "ssn", "credit_card", "bank_account",
    }}
}

func (f *DataFilter) Apply(data map[string]any) map[string]any {
    out := make(map[string]any, len(data))
    for k, v := range data {
        if f.matches(strings.ToLower(k)) {
            out[k] = "[REDACTED]"
            continue
        }
        out[k] = v
    }
    return out
}
```

## Integration checklist

- [ ] Identify every field a new integration will receive.
- [ ] Enforce a redaction pass on all PII before transmission.
- [ ] Confirm data residency, retention, and deletion commitments.
- [ ] Audit data flows on a schedule; review terms for how payloads are used.
- [ ] Record consent where the law requires it and support deletion requests.

## CWE references

- CWE-200 - exposure of sensitive information
- CWE-359 - exposure of private personal information
- CWE-201 - information exposure through sent data
