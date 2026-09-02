# Migration Guide: v1 to v2

This guide helps you migrate from the MarketData Go SDK v1 to v2.

## Overview

v2 is a complete rewrite following idiomatic Go patterns. The API has changed significantly.

## Import Path

```go
// v1
import "github.com/MarketDataApp/sdk-go"

// v2
import "github.com/MarketDataApp/sdk-go/v2/marketdata"
```

## Client Creation

### v1 (Singleton Pattern)
```go
// v1: Global singleton initialized from environment
client, _ := api.GetClient()

// Or create with explicit token
client, _ := api.NewClient("your-token")

// init() runs on import, making network calls automatically
```

### v2 (Explicit Construction)
```go
// v2: Explicit client construction
client, err := marketdata.NewClient(
    marketdata.WithToken("your-token"),
)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Or use environment variable (MARKETDATA_TOKEN)
client, err := marketdata.NewClient()
```

**Key changes:**
- No global state
- No network calls on import
- Explicit error handling on construction
- Environment variable fallback built-in
- Call `Close()` when done to release resources

## Making Requests

### v1
```go
// v1: Builder pattern with method chaining
quote, err := api.StockQuote().Symbol("AAPL").Get(ctx)

candles, err := api.StockCandles().
    Symbol("AAPL").
    Resolution("D").
    From("2024-01-01").
    To("2024-01-31").
    Get(ctx)
```

### v2
```go
// v2: Context-first, resource namespaced, functional options
quote, _, err := client.Stocks.Quote(ctx, "AAPL")

candles, _, err := client.Stocks.Candles(ctx, "AAPL",
    stocks.WithResolution(stocks.ResolutionDaily),
    stocks.WithCandleWindow(stocks.Between(startDate, endDate)),
)
```

**Key changes:**
- All methods require `context.Context` as the first parameter
- Resources are namespaced: `client.Stocks`, `client.Options`, etc.
- Functional options replace builder pattern for optional parameters
- Methods return `(data, *Response, error)` — the `*Response` provides access to HTTP metadata and rate limit info
- Type-safe resolution constants replace string values
- `time.Time` replaces string dates

## Execution Methods

v1 provided three execution methods per endpoint:
- `.Get(ctx)` — unpacked data
- `.Packed(ctx)` — wire-format response
- `.Raw(ctx)` — raw `*resty.Response`

v2 has a single return pattern: `(data, *response.Response, error)`. The `*Response` embeds `*http.Response` for raw access and includes rate limit metadata. If you need the raw body, use `response.Body`.

## Error Handling

### v1
```go
// v1: Unstructured string errors
if err != nil {
    // "received non-OK status: 429 Too Many Requests, error message: ..."
    log.Println(err)
}
```

### v2
```go
// v2: Rich error types
if err != nil {
    var apiErr *marketdata.APIError
    if errors.As(err, &apiErr) {
        fmt.Printf("API error %d: %s (request_id=%s)\n",
            apiErr.StatusCode, apiErr.Message, apiErr.RequestID)
    }

    // Check specific conditions
    if errors.Is(err, marketdata.ErrRateLimited) {
        // Handle rate limiting
    }
}
```

**Key changes:**
- Custom error types with full context
- Support for `errors.Is` and `errors.As`
- Request ID included for support tickets
- `Retryable()` method on all error types

## Resource Mapping

| v1 | v2 | Notes |
|----|-----|-------|
| `api.StockQuote().Symbol("AAPL").Get(ctx)` | `client.Stocks.Quote(ctx, "AAPL")` | |
| `api.BulkStockQuotes().Symbols(syms).Get(ctx)` | `client.Stocks.Quotes(ctx, syms)` | |
| `api.StockCandles().Symbol("AAPL").Get(ctx)` | `client.Stocks.Candles(ctx, "AAPL")` | |
| `api.BulkStockCandles().Symbols(syms).Get(ctx)` | `client.Stocks.BulkCandles(ctx, syms)` | |
| `api.StockEarnings().Symbol("AAPL").Get(ctx)` | `client.Stocks.Earnings(ctx, "AAPL")` | |
| `api.StockNews().Symbol("AAPL").Get(ctx)` | `client.Stocks.News(ctx, "AAPL")` | |
| `api.OptionChain().UnderlyingSymbol("AAPL").Get(ctx)` | `client.Options.Chain(ctx, "AAPL")` | |
| `api.OptionExpirations().UnderlyingSymbol("AAPL").Get(ctx)` | `client.Options.Expirations(ctx, "AAPL")` | |
| `api.OptionStrikes().UnderlyingSymbol("AAPL").Get(ctx)` | `client.Options.Chain(ctx, "AAPL", ...)` | Strikes endpoint removed in v2; read strikes from the chain |
| `api.OptionQuote().OptionSymbol("...").Get(ctx)` | `client.Options.Quote(ctx, "...")` | |
| `api.OptionLookup()` | `client.Options.Lookup(ctx, ...)` | |
| `api.FundCandles().Symbol("SPY").Get(ctx)` | `client.Funds.Candles(ctx, "SPY")` | |
| `api.MarketStatus().Get(ctx)` | `client.Markets.Status(ctx)` | |
| N/A | `client.Stocks.Prices(ctx, syms)` | New in v2 |
| N/A | `client.Markets.StatusHistory(ctx)` | New in v2 |
| N/A | `client.Utilities.Status/Headers/User(ctx)` | New in v2 (v1 had no Utilities resource) |

### Removed from v2

The following v1 features have been removed:
- **`Packed()` / `Raw()` execution methods** — Replaced by the `*Response` return value (see "Execution Methods" above)
- **In-memory request logging** (`GetLogs()`) — Replaced by standard `slog` integration via `WithLogger()`

## Configuration Options

| v1 | v2 | Notes |
|----|-----|-------|
| `MARKETDATA_TOKEN` env var | `MARKETDATA_TOKEN` env var or `WithToken()` | Same env var, new option |
| `client.Timeout(seconds)` | `context.WithTimeout(ctx, duration)` | Per-request via context |
| `client.Debug(true)` | `WithDebug(true)` | Now a constructor option |
| N/A | `WithHTTPClient(client)` | New: custom HTTP client |
| N/A | `WithMaxRetries(n)` | New: configurable retries (default 3) |
| N/A | `WithLogger(logger)` | New: custom slog logger |
| N/A | `WithoutStartupValidation()` | New: skip token check at startup |

## Concurrency

### v1
```go
// v1: Unclear thread safety with global singleton
```

### v2
```go
// v2: Designed for concurrent use
var wg sync.WaitGroup
for _, symbol := range symbols {
    wg.Add(1)
    go func(s string) {
        defer wg.Done()
        quote, _, _ := client.Stocks.Quote(ctx, s)
        // process quote
    }(symbol)
}
wg.Wait()
```

**Key changes:**
- Client is safe for concurrent use
- No global state to coordinate
- Built-in 50-request concurrency pool
- Auto date-range splitting for large candle requests
- Each request can have its own context/timeout

## Rate Limits

### v1
```go
// v1: Boolean check only
if client.RateLimitExceeded() {
    // wait...
}
```

### v2
```go
// v2: Proactive rate limit tracking with full state
limits := client.RateLimits()
fmt.Printf("Used %d/%d, resets at %v\n",
    limits.Consumed, limits.Limit, limits.ResetAt)
```

## Dependencies

v1 required several external dependencies (resty, godotenv, color, orderedmap, testify). v2 uses only the Go standard library at runtime.

## Quick Migration Checklist

- [ ] Update import path to `github.com/MarketDataApp/sdk-go/v2/marketdata`
- [ ] Replace global client with explicit `marketdata.NewClient()`
- [ ] Add `defer client.Close()` after client creation
- [ ] Add `context.Context` to all API calls
- [ ] Update method names to use resource namespaces (e.g., `client.Stocks.Quote`)
- [ ] Replace builder pattern with functional options
- [ ] Update error handling to use `errors.As` and `errors.Is`
- [ ] Replace string dates with `time.Time`
- [ ] Replace `client.Timeout(n)` with `context.WithTimeout()`
- [ ] Update code that used `Packed()` or `Raw()` to use `*Response`
- [ ] Remove any `GetLogs()` usage (use `WithLogger()` instead)
- [ ] Run `go build` to catch any remaining issues
