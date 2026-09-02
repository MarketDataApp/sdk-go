# ADR-004: Request/Response Pattern

## Status
Accepted

## Context

We need to define how API methods are called and how responses are returned. This affects:
- API ergonomics (how users call methods)
- Type safety (compile-time vs runtime checks)
- Flexibility (optional parameters)
- Go idioms (context handling)

## Decision Drivers

- **Context-First**: All methods accept context.Context
- **Type Safety**: Compile-time validation where possible
- **Go Idioms**: Follow standard patterns
- **Flexibility**: Support optional parameters without pain

## Decision

### Method Signatures

All API methods follow this pattern (the `*response.Response` carries
per-request metadata such as status and rate-limit state):

```go
func (s *Service) Method(ctx context.Context, required params, opts ...Option) (Result, *response.Response, error)
```

Examples:
```go
// Simple case - just symbol
quote, _, err := client.Stocks.Quote(ctx, "AAPL")

// With options — the date range is a single sealed-union value
candles, _, err := client.Stocks.Candles(ctx, "AAPL",
    stocks.WithResolution(stocks.ResolutionDaily),
    stocks.WithCandleWindow(stocks.Between(startDate, endDate)),
)

// Multiple symbols
quotes, _, err := client.Stocks.Quotes(ctx, []string{"AAPL", "MSFT", "GOOG"})
```

### Mutually-exclusive parameters

Where the API has mutually-exclusive parameters, the option carries a **sealed
union value** so incompatible modes cannot be combined at compile time (for
example the candle date window is one `stocks.DateWindow`, not separate
`WithFrom`/`WithTo`/`WithCountback` options). See
[ADR-017](ADR-017-compile-time-exclusivity.md) for the full mechanism and the
negative-compile enforcement.

### Context Usage

Context is **always** the first parameter:
- Enables timeout/cancellation
- Can carry request-scoped values
- Required by http.Request
- Follows Go convention

```go
// Set timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

quote, err := client.Stocks.Quote(ctx, "AAPL")
```

### Response Types

Responses are concrete structs with JSON tags:

```go
type Quote struct {
    Symbol     string    `json:"symbol"`
    Last       float64   `json:"last"`
    Change     float64   `json:"change"`
    ChangePct  float64   `json:"changepct"`
    Volume     int64     `json:"volume"`
    Updated    time.Time `json:"updated"`
}

type Candle struct {
    Time   time.Time `json:"t"`
    Open   float64   `json:"o"`
    High   float64   `json:"h"`
    Low    float64   `json:"l"`
    Close  float64   `json:"c"`
    Volume int64     `json:"v"`
}
```

### Optional Parameters

Use functional options for optional parameters:

```go
// Option function type
type QuoteOption func(*quoteOptions)

// Option implementations
func WithFiftyTwoWeek(enabled bool) QuoteOption {
    return func(o *quoteOptions) {
        o.fiftyTwoWeek = enabled
    }
}

// Internal options struct
type quoteOptions struct {
    fiftyTwoWeek bool
    extended     bool
}
```

## Consequences

### Positive
- Consistent, predictable API
- Type-safe parameters
- Context propagation
- Extensible without breaking changes

### Negative
- Options pattern adds some verbosity
- Multiple options structs to maintain

## References

- [AWS SDK Go v2 Patterns](https://github.com/aws/aws-sdk-go-v2)
- [Stripe Go SDK](https://github.com/stripe/stripe-go)
