# ADR-007: Retry Strategy

## Status
Accepted

## Context

Network requests can fail due to transient issues. We need a retry strategy that:
- Automatically recovers from temporary failures
- Doesn't waste resources on permanent failures
- Respects server rate limits
- Provides visibility into retry behavior

## Decision Drivers

- **Automatic Recovery**: Retry transient errors automatically
- **Exponential Backoff**: Prevent thundering herd
- **Rate Limit Awareness**: Respect 429 responses
- **Configurable**: Users can adjust behavior
- **Observable**: Logging of retry attempts

## Decision

### Retry Configuration

```go
type RetryConfig struct {
    MaxRetries     int           // Default: 3 (configurable with WithMaxRetries)
    InitialBackoff time.Duration // Fixed: 1 second
    Multiplier     float64       // Fixed: 2.0
}
```

Users can configure the retry count via `WithMaxRetries(n)`. Set to 0 to disable retries entirely.

Backoff formula: `initial * 2^retry` (uncapped, per SDK requirements §9.3) → 1s, 2s, 4s at the default of 3 retries.

### HTTP Status Code Handling

| Status Code | Action | Retry? |
|-------------|--------|--------|
| 200 | Success - parse response | — |
| 203 | Success - non-authoritative (cached data) - parse response | — |
| 400 | Return `BadRequestError` | **No** |
| 401 | Return `AuthenticationError` - **fail immediately** | **No** |
| 404 | Return empty result with `NoData: true` (not an error) | **No** |
| 429 | Return `RateLimitError` - expose retry-after | **No** |
| 500 | Return `ServerError` | **No** |
| 501-599 | Retry with exponential backoff | **Yes** |

### Retryable Conditions

| Condition | Retry? | Reason |
|-----------|--------|--------|
| Status 501-599 | Yes | Server issues (transient) |
| Network/connection errors | Yes | Network transient |
| Status 400-404 | No | Client error |
| Status 429 (Rate Limited) | No | Expose retry-after, do not retry |
| Status 500 | No | Server error, do not retry |

### Implementation

```go
func (c *Client) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
    var lastErr error

    for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
        if attempt > 0 {
            // API status check before retry (non-blocking)
            if c.apiStatus.IsOffline() {
                return nil, fmt.Errorf("API is offline, aborting retry: %w", lastErr)
            }

            backoff := c.calculateBackoff(attempt)

            // Respect Retry-After header if present
            if retryAfter > 0 {
                backoff = retryAfter
            }

            select {
            case <-ctx.Done():
                return nil, ctx.Err()
            case <-time.After(backoff):
            }
        }

        resp, err := c.http.Do(req)
        if err != nil {
            if isRetryableNetworkError(err) {
                lastErr = err
                continue
            }
            return nil, &NetworkError{...}
        }

        // Only retry 501-599
        if resp.StatusCode > 500 && resp.StatusCode < 600 {
            lastErr = newServerError(resp)
            resp.Body.Close()
            continue
        }

        // All other status codes: return immediately (no retry)
        return resp, nil
    }

    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

### Backoff Calculation

```go
func (c *Client) calculateBackoff(attempt int) time.Duration {
    // Formula: initial * 2^attempt (uncapped, per SDK requirements §9.3)
    // With defaults: 1s, 2s, 4s
    return c.config.InitialBackoff * time.Duration(math.Pow(c.config.Multiplier, float64(attempt-1)))
}
```

### API Status Check Before Retry

Before each retry attempt for 501-599 errors, the SDK checks cached API status:

```go
type APIStatusCache struct {
    mu            sync.RWMutex
    status        string    // "online", "offline", or "unknown"
    lastFetch     time.Time
    refreshInterval time.Duration // 270 seconds (4.5 minutes)
    cacheValidity   time.Duration // 300 seconds (5 minutes)
}
```

**Cache thresholds:**
1. Cache age **< 270s**: use cached status, no refresh
2. Cache age **270s to < 300s**: use cached status, trigger **non-blocking async refresh**
3. Cache **stale or empty** (>= 300s): trigger **non-blocking async refresh**, treat as `unknown`

**Retry decision based on status:**
- `offline` → do not retry, fail immediately
- `online` or `unknown` → proceed with retry/backoff

The status refresh is always non-blocking and never performed synchronously inside the retry loop.

### Retry-After Header

If the response includes a `Retry-After` header, the server-specified delay overrides the calculated backoff.

## Consequences

### Positive
- Automatic recovery from transient failures
- Configurable for different use cases
- Rate limit awareness prevents bans
- Exponential backoff with jitter

### Negative
- Adds latency on failures
- Memory for retry state

### Mitigations
- Users can disable retries
- Clear logging of retry attempts
- Context cancellation respected

## References

- [AWS SDK Retry Behavior](https://docs.aws.amazon.com/sdkref/latest/guide/feature-retry-behavior.html)
- [Exponential Backoff and Jitter - AWS](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/)
