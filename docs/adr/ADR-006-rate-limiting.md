# ADR-006: Rate Limiting Strategy

## Status
Accepted

## Context

The MarketData API enforces rate limits via HTTP headers. The SDK needs to:
- Track rate limit state from response headers
- Prevent requests when limits are exhausted
- Provide visibility to users
- Handle rate limit errors gracefully

A proactive approach (check before request) is better than reactive (fail then retry).

## Decision Drivers

- **Proactive Prevention**: Check before request, not after failure
- **Transparency**: Users can see their quota
- **Automatic Tracking**: Update from every response
- **Thread Safety**: Safe for concurrent use

## Decision

### Rate Limit Tracking

```go
type RateLimits struct {
    mu sync.RWMutex

    Limit     int       // Total requests allowed
    Remaining int       // Requests remaining
    Consumed  int       // Requests consumed
    ResetAt   time.Time // When limit resets
}

// Thread-safe updates
func (r *RateLimits) Update(resp *http.Response) {
    r.mu.Lock()
    defer r.mu.Unlock()

    r.Limit = parseHeader(resp, "X-Api-Ratelimit-Limit")
    r.Remaining = parseHeader(resp, "X-Api-Ratelimit-Remaining")
    r.Consumed = parseHeader(resp, "X-Api-Ratelimit-Consumed")
    r.ResetAt = parseResetHeader(resp, "X-Api-Ratelimit-Reset")
}

// Check before request
func (r *RateLimits) Exceeded() bool {
    r.mu.RLock()
    defer r.mu.RUnlock()

    return r.Remaining <= 0 && time.Now().Before(r.ResetAt)
}
```

### Pre-request Checking

```go
func (c *Client) do(ctx context.Context, req *http.Request) (*http.Response, error) {
    // Check rate limits before making request
    if c.rateLimits.Exceeded() {
        return nil, &RateLimitError{
            Limit:     c.rateLimits.Limit,
            Remaining: 0,
            ResetAt:   c.rateLimits.ResetAt,
        }
    }

    resp, err := c.doWithRetry(ctx, req)
    if err != nil {
        return nil, err
    }

    // Update rate limits from response
    c.rateLimits.Update(resp)

    return resp, nil
}
```

### User Access

```go
// Users can check their rate limits
limits := client.RateLimits()
fmt.Printf("Used %d/%d, resets at %v\n",
    limits.Consumed, limits.Limit, limits.ResetAt)

// Check if quota available
if limits.Remaining > 0 {
    // Safe to make request
}
```

### Initialization

```go
func NewClient(opts ...Option) (*Client, error) {
    // ... create client ...

    // Initialize rate limits from /user endpoint
    if err := c.initRateLimits(context.Background()); err != nil {
        // Log warning but don't fail - limits will be populated on first request
        c.logger.Warn("failed to initialize rate limits", "error", err)
    }

    return c, nil
}
```

## Consequences

### Positive
- Fail fast when quota exhausted
- No wasted requests
- User visibility into quota
- Thread-safe concurrent access

### Negative
- Extra request at initialization
- Stale data possible (mitigated by updates)

### Mitigations
- Optional initialization (can skip with option)
- Update on every response
- Rate limits cached on client, not global

## References

- [HTTP Rate Limiting Headers](https://tools.ietf.org/html/draft-polli-ratelimit-headers)
- [Stripe Go SDK Rate Limiting](https://github.com/stripe/stripe-go)
