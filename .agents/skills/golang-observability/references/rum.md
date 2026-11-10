# Real User Monitoring (RUM)

Backend signals (logs, metrics, traces, profiles) describe the *system*. RUM describes the *user's experience* — and the Go backend owns the server side of it: business events, Customer Data Platform (CDP) feeds, and correlating sessions with traces.

## Capabilities

| Capability | What it reveals | Tools |
| --- | --- | --- |
| Product analytics | page views, clicks, adoption, retention | PostHog, Amplitude, Mixpanel |
| Funnel analysis | where users drop off (signup → checkout) | PostHog, Amplitude, Mixpanel |
| CDP | unified cross-source user profile | Segment, RudderStack |

## Identity key: `user_id`, never email

The identity key (PostHog `DistinctId`, Segment `UserId`, Amplitude `user_id`) must be the immutable internal `user_id`:

```go
posthogClient.Enqueue(posthog.Capture{
    DistinctId: user.ID, // "usr_a1b2c3" — stable, not PII
    Event:      "order_completed",
})
```

Why email fails as an identity key:

- **Mutable** — a changed email splits one user's history into two, breaking funnels, retention, and cohorts.
- **PII** — the entire analytics pipeline (including third-party vendors) ends up holding personal data, complicating GDPR/CCPA.
- **Non-unique** — the same address can map to different accounts across systems and environments.

Store email as a display property, never as the primary key.

## The backend's role

1. **Server-side events** — business events the backend alone knows about (payment completed, plan upgraded, email sent) are tracked from Go so they join the frontend pipeline:

```go
posthogClient.Enqueue(posthog.Capture{
    DistinctId: order.UserID,
    Event:      "order_completed",
    Properties: posthog.NewProperties().
        Set("order_id", order.ID).
        Set("amount", order.Total).
        Set("payment_method", order.PaymentMethod).
        Set("item_count", len(order.Items)),
})
```

2. **Session ↔ trace correlation** — the frontend ships `X-Session-ID`/`X-Distinct-ID` headers; middleware copies them onto the backend span so "the page was slow" maps from a session recording to the exact trace:

```go
if sessionID := r.Header.Get("X-Session-ID"); sessionID != "" {
    span.SetAttributes(attribute.String("rum.session_id", sessionID))
}
if distinctID := r.Header.Get("X-Distinct-ID"); distinctID != "" {
    span.SetAttributes(attribute.String("rum.distinct_id", distinctID))
}
```

3. **CDP ingestion** — for a Customer Data Platform (Segment, RudderStack), send server-side `analytics.Track` and `analytics.Identify` calls so frontend and backend events merge into one user profile.

## GDPR and CCPA

Behavioral data triggers privacy law, and the fines are steep (GDPR up to 4% of global turnover; CCPA up to $7,500 per intentional violation). Compliance is not optional.

- **Consent first** — no analytics script loads and no server event fires until the user consents. Check the flag before tracking:

```go
consent := auth.ConsentFromContext(ctx)
if consent.Analytics {
    posthogClient.Enqueue(posthog.Capture{...})
}
```

- **Data subject rights** — implement endpoints that reach every holder of user data:
  - `GET /api/users/{id}/data` — right of access/export (Article 15).
  - `DELETE /api/users/{id}/data` — right to erasure (Article 17): delete from your database, then call `DeleteUser` on the analytics platform and the CDP.

- **Privacy checklist:**
  - [ ] opt-in consent per purpose (analytics vs marketing vs functional)
  - [ ] data minimization — no PII in events
  - [ ] retention limits on aggregated analytics
  - [ ] signed DPAs with every third-party vendor
  - [ ] privacy policy listing tools, data collected, retention period
  - [ ] identity key is a non-PII `user_id`
  - [ ] consider self-hosting to keep data in your infrastructure

## Self-hosted vs SaaS

| | Self-hosted (PostHog, Matomo) | SaaS (Amplitude, Mixpanel) |
| --- | --- | --- |
| Data residency | full control | vendor's servers |
| GDPR | no cross-border transfer concerns | DPA + SCC or adequacy needed |
| Cost | infrastructure, scales with volume | per-event / per-seat |
| Operations | you own upgrades, scaling, backups | vendor handles everything |

For EU-focused products or strict residency requirements, self-hosted PostHog removes most cross-border friction.

## Cost of RUM

RUM pricing is event-based: every page view, click, and custom event counts, and CDPs charge per tracked user and per event. At scale, a CDP can out-cost your backend infrastructure.

Mitigations: drop low-value events server-side before they ship, self-host to convert per-event pricing into fixed infrastructure cost, and cap retention on aggregated analytics.
