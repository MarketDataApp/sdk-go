# ADR-014: Concurrency Helpers

## Status
Accepted

## Context

The SDK needs built-in concurrency support for high-throughput use cases (req 12): a global concurrency pool to prevent overwhelming the API, automatic date-range splitting for large candle requests, and batch processing for multi-symbol option quotes. Go's goroutine model makes this natural, but the pool must be enforced globally across all endpoints.

## Decision

### Global Concurrency Pool

The client enforces a **maximum of 50 concurrent in-flight requests** using a semaphore (sliding window, not batching):

```go
type Client struct {
    // ... other fields
    sem chan struct{} // buffered channel of size 50
}

func NewClient(opts ...Option) (*Client, error) {
    // ...
    c.sem = make(chan struct{}, 50)
    // ...
}
```

All HTTP requests acquire a slot before executing and release it on completion:

```go
func (c *Client) do(ctx context.Context, req *http.Request) (*http.Response, error) {
    // Acquire semaphore slot (blocks if 50 in-flight)
    select {
    case c.sem <- struct{}{}:
        defer func() { <-c.sem }()
    case <-ctx.Done():
        return nil, ctx.Err()
    }

    return c.http.Do(req)
}
```

**Sliding window behavior:**
- Maintains up to 50 active requests at all times when demand exists
- As soon as one request completes, the next waiting request starts immediately
- No batching — does not wait for all 50 to finish before starting more
- Respects context cancellation while waiting for a slot

### Date-Range Splitting for Candles

`Stocks.Candles()` automatically splits intraday requests spanning more than 1 year into year-sized chunks, fetched concurrently:

```go
// Automatically splits into yearly chunks, fetched concurrently
candles, err := client.Stocks.Candles(ctx, "AAPL",
    stocks.WithResolution(stocks.Minute),
    stocks.WithFrom(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
    stocks.WithTo(time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)),
)
// Result: single merged slice of candles, sorted chronologically
```

Implementation:
1. Detect if resolution is intraday and date range exceeds 1 year
2. Split into year-aligned chunks (e.g., 2020, 2021, 2022, 2023)
3. Launch goroutines for each chunk (respects global pool limit)
4. Collect results, sort chronologically, merge into single `[]Candle`
5. If any chunk fails, return the error (cancel remaining via context)

Daily and higher resolutions are not split (the API handles them in a single request).

### Batch Processing for Option Quotes

`Options.Quotes()` accepts multiple option symbols and fetches them concurrently:

```go
// Fetch multiple option quotes concurrently
quotes, err := client.Options.Quotes(ctx,
    []string{"AAPL230616C00150000", "AAPL230616P00150000", "MSFT230616C00300000"},
)
// Result: single merged slice of OptionQuote
```

Implementation:
1. Launch one goroutine per symbol (respects global pool limit)
2. Collect results into a single `[]OptionQuote`
3. If any request fails, return the error

### Thread Safety

All concurrency primitives are goroutine-safe:
- The semaphore is a buffered channel (inherently safe)
- Rate limit state uses `sync.RWMutex` (see ADR-006)
- The underlying `http.Client` is safe for concurrent use
- Result collection uses channels or `sync.WaitGroup` + mutex

### Error Handling in Concurrent Operations

When multiple concurrent requests are in flight:
- The first error cancels remaining requests via a shared `context.WithCancel`
- The first error is returned to the caller
- Partial results are not returned — it is all-or-nothing
- Rate limit errors (`429`) are not retried and cancel the batch immediately

## Consequences

### Positive
- Prevents overwhelming the API with unbounded concurrency
- Sliding window maximizes throughput (no idle time between batches)
- Date-range splitting is transparent to the user
- Batch option quotes reduce boilerplate for common use cases
- Goroutine model makes implementation natural in Go

### Negative
- Pool size of 50 is fixed (not configurable) — may be suboptimal for some use cases
- Date-range splitting adds complexity to candle methods
- All-or-nothing error semantics may discard successful results on partial failure

### Mitigations
- 50 is the API's recommended concurrency limit
- Date-range splitting is automatic and transparent
- Users needing partial results can make individual requests with their own error handling
- Debug logging shows when requests are queued behind the semaphore

## References

- Requirements: Section 12 (Concurrency Helpers)
- Related: ADR-006 (rate limiting), ADR-008 (context cancellation)
