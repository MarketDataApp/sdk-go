# ADR-008: Context and Cancellation

## Status
Accepted

## Context
Go's `context.Context` is the standard mechanism for:
- Request cancellation
- Timeout propagation
- Request-scoped values (trace IDs, auth tokens)

The SDK needs a clear strategy for context usage that follows Go conventions.

## Decision

### Context as First Parameter
All public methods that perform I/O accept `context.Context` as the first parameter:

```go
func (s *Service) Quote(ctx context.Context, symbol string) (*Quote, error)
func (s *Service) Candles(ctx context.Context, symbol string, opts ...CandleOption) ([]Candle, error)
```

### Context Propagation
- Context is passed through the entire call chain
- HTTP requests use `http.NewRequestWithContext(ctx, ...)`
- Retries respect context cancellation between attempts

### Timeout Handling
The SDK enforces a fixed 99-second request timeout (and 2-second connection timeout when supported). These are not configurable via client options. Users can still set **per-request** timeouts via context, which take precedence when shorter than the fixed timeout:

```go
// Per-request timeout via context (overrides fixed timeout when shorter)
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
quote, err := client.Stocks.Quote(ctx, "AAPL")
```

### Cancellation Behavior
- Cancelled context returns `context.Canceled` error
- Timed-out context returns `context.DeadlineExceeded` error
- In-flight HTTP requests are cancelled via `http.Client` transport
- Retry loops check context before each attempt

### Context Values
The SDK does not store values in context. Configuration is passed via:
- Client options (client-level)
- Functional options (per-request)

This keeps the API explicit and testable.

## Consequences

### Positive
- Standard Go pattern familiar to developers
- Clean cancellation semantics
- Composable with user's existing context chains
- Enables distributed tracing integration

### Negative
- Every method requires context parameter (verbosity)
- Users must understand context lifecycle

## Implementation Notes
- Never store `context.Context` in structs
- Always accept context as first parameter, not via options
- Use `context.Background()` only in examples/tests, never in library code
