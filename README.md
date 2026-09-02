<div align="center">

# Market Data Go SDK v2
### Access Financial Data with Ease

> This is the official Go SDK for [Market Data](https://www.marketdata.app/). It provides developers with a powerful, easy-to-use interface to obtain real-time and historical financial data. Ideal for building financial applications, trading bots, and investment strategies.

[![Tests](https://github.com/MarketDataApp/sdk-go/actions/workflows/tests.yml/badge.svg)](https://github.com/MarketDataApp/sdk-go/actions/workflows/tests.yml)
[![Coverage](https://codecov.io/gh/MarketDataApp/sdk-go/graph/badge.svg)](https://codecov.io/gh/MarketDataApp/sdk-go)
[![License](https://img.shields.io/github/license/MarketDataApp/sdk-go.svg)](https://github.com/MarketDataApp/sdk-go/blob/main/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/MarketDataApp/sdk-go/v2.svg)](https://pkg.go.dev/github.com/MarketDataApp/sdk-go/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/MarketDataApp/sdk-go/v2)](https://goreportcard.com/report/github.com/MarketDataApp/sdk-go/v2)
[![Go Version](https://img.shields.io/github/go-mod/go-version/MarketDataApp/sdk-go)](https://go.dev/)

#### Connect With The Market Data Community

[![Website](https://img.shields.io/badge/Website-marketdata.app-blue)](https://www.marketdata.app/)
[![Discord](https://img.shields.io/badge/Discord-join%20chat-7389D8.svg?logo=discord&logoColor=ffffff)](https://discord.com/invite/GmdeAVRtnT)
[![Twitter](https://img.shields.io/twitter/follow/MarketDataApp?style=social)](https://twitter.com/MarketDataApp)
[![Helpdesk](https://img.shields.io/badge/Support-Ticketing-ff69b4.svg?logo=TicketTailor&logoColor=white)](https://www.marketdata.app/dashboard/)

</div>

## Features

- **Idiomatic Go** - Follows Go best practices and conventions (service structs, `context` first, `(T, *Response, error)` returns — like `google/go-github`)
- **Compile-time parameter safety** - Mutually-exclusive parameters are *unrepresentable*: illegal combinations (e.g. a date range plus a countback) do not compile. See [ADR-017](docs/adr/ADR-017-compile-time-exclusivity.md).
- **Context Support** - All methods accept `context.Context` for cancellation
- **Functional Options** - Clean, extensible configuration with sealed-union values
- **Automatic Retries** - Exponential backoff for transient errors
- **Rate Limit Tracking** - Proactive rate limit management
- **Hardened** - Untrusted symbols safely encoded; token never sent in cleartext or leaked; response-size bounded; `govulncheck`/`staticcheck` clean
- **Type Safety** - Strongly typed responses

## Installation

```bash
go get github.com/MarketDataApp/sdk-go/v2
```

Requires Go 1.22 or later.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/MarketDataApp/sdk-go/v2/marketdata"
    "github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

func main() {
    // Create client (reads MARKETDATA_TOKEN from environment)
    client, err := marketdata.NewClient()
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Get a stock quote
    quote, _, err := client.Stocks.Quote(ctx, "AAPL")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("AAPL: $%.2f\n", quote.Last)

    // Get multiple quotes
    quotes, _, err := client.Stocks.Quotes(ctx, []string{"AAPL", "MSFT", "GOOG"})
    if err != nil {
        log.Fatal(err)
    }
    for _, q := range quotes {
        fmt.Printf("%s: $%.2f (%.2f%%)\n", q.Symbol, q.Last, q.ChangePercent)
    }

    // Get historical candles
    candles, _, err := client.Stocks.Candles(ctx, "AAPL",
        stocks.WithResolution(stocks.ResolutionDaily),
        stocks.WithCandleWindow(stocks.LastN(30)),
    )
    if err != nil {
        log.Fatal(err)
    }
    for _, c := range candles {
        fmt.Printf("%s: O=%.2f C=%.2f\n", c.Time.Format("2006-01-02"), c.Open, c.Close)
    }
}
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MARKETDATA_TOKEN` | API authentication token | (required) |
| `MARKETDATA_BASE_URL` | API base URL | `https://api.marketdata.app` |
| `MARKETDATA_API_VERSION` | API version path segment | `v1` |
| `MARKETDATA_MODE` | Default data mode (`live`/`cached`/`delayed`) | (unset) |
| `MARKETDATA_DATE_FORMAT` | Default `dateformat` for responses | (unset) |
| `MARKETDATA_COLUMNS` | Default `columns` selection | (unset) |
| `MARKETDATA_ADD_HEADERS` | Add a header row to CSV output (`true`/`false`) | (unset) |
| `MARKETDATA_USE_HUMAN_READABLE` | Return human-readable values (`true`/`false`) | (unset) |
| `MARKETDATA_LOGGING_LEVEL` | Log level (e.g. `debug`, `info`) | (unset) |

All variables are also loaded from a project-level `.env` file if present
(real environment variables win over `.env`; client options like
`WithBaseURL`/`WithEnvironment` win over both, and method parameters win over
everything — the standard configuration cascade).

### Client Options

```go
client, err := marketdata.NewClient(
    marketdata.WithToken("your-token"),
    marketdata.WithMaxRetries(3),
    marketdata.WithDebug(true),
    marketdata.WithHTTPClient(customHTTPClient),
)
```

Other options include `WithBaseURL`, `WithEnvironment`, `WithLogger`,
`WithoutStartupValidation`, and the universal-parameter options
(`WithDateFormat`, `WithColumns`, `WithAddHeaders`, `WithHumanReadable`,
`WithMode`, `WithMaxAge`, `WithLimit`, `WithOffset`). Request timeouts are
fixed (99s request, 2s connect) and are not configurable.

## API Reference

### Stocks

```go
// Single quote
quote, _, err := client.Stocks.Quote(ctx, "AAPL")

// Multiple quotes
quotes, _, err := client.Stocks.Quotes(ctx, []string{"AAPL", "MSFT"})

// With 52-week high/low
quote, _, err := client.Stocks.Quote(ctx, "AAPL", stocks.WithFiftyTwoWeek(true))

// Historical candles
candles, _, err := client.Stocks.Candles(ctx, "AAPL",
    stocks.WithResolution(stocks.ResolutionDaily),
    stocks.WithCandleWindow(stocks.Between(startDate, endDate)),
)
```

### Options

Mutually-exclusive filters are single sealed-union values, so incompatible
combinations cannot be written:

```go
// Chain: pick one strike selector, one expiry selector, one (optional) as-of date.
chain, _, err := client.Options.Chain(ctx, "AAPL",
    options.WithExpiry(options.OnExpiration(exp)),  // or InDTE(30) / InMonth(12) / InMonthOfYear(12, 2026)
    options.WithStrike(options.StrikeRange(140, 160)), // or Strike(150) / MinStrike / MaxStrike / StrikeExpr / ByDelta
    options.WithSide(options.SideCall),
)

// Without an expiry filter the API returns only the front-month expiration.
// AllExpirations() is what asks for the whole chain (and costs accordingly).
full, _, err := client.Options.Chain(ctx, "AAPL",
    options.WithExpiry(options.AllExpirations()),
)

// Strike and delta also take lists — both legs of a spread in one request.
spread, _, err := client.Options.Chain(ctx, "AAPL",
    options.WithStrike(options.Strikes(300, 310)), // or ByDeltas(0.16, 0.30)
)

// QuotesBySymbol keeps one entry per symbol, nil where the API had no data;
// Quotes just omits them, so its slice can be shorter than the input.
quotes, _, err := client.Options.QuotesBySymbol(ctx, []string{"AAPL260821C00300000", "AAPL260821P00300000"})

// A historical window selects one quote per day. Quote returns the first;
// QuoteHistory returns them all.
series, _, err := client.Options.QuoteHistory(ctx, "AAPL260821C00300000",
    options.WithOptionQuoteWindow(options.QuoteRange(from, to)), // or QuoteLastNUntil(5, to)
)

// A historical single-contract quote: one date, or one range — never both.
q, _, err := client.Options.Quote(ctx, "AAPL260717C00150000",
    options.WithOptionQuoteWindow(options.QuoteOnDate(day)), // or QuoteRange(from, to)
)
```

### Compile-time exclusivity

The redesign makes the API's mutually-exclusive parameters impossible to combine.
For example, these do **not compile** — the symbols do not exist / the types do
not match:

```go
// date range AND countback — no such combination is expressible:
client.Stocks.Candles(ctx, "AAPL", stocks.WithFrom(a), stocks.WithCountback(5)) // ✗ won't build

// a Status-only date on the ranged history method:
client.Markets.StatusHistory(ctx, markets.WithDate(day)) // ✗ won't build
```

The date range is one value instead:

```go
client.Stocks.Candles(ctx, "AAPL", stocks.WithCandleWindow(stocks.LastN(5))) // ✓
```

The negative-compile test suite (`internal/negcompile`) proves every illegal
combination fails to build and every legal one compiles.

### Resolutions

| Resolution | Constant |
|------------|----------|
| 1 minute | `stocks.Resolution1Min` |
| 3 minutes | `stocks.Resolution3Min` |
| 5 minutes | `stocks.Resolution5Min` |
| 15 minutes | `stocks.Resolution15Min` |
| 30 minutes | `stocks.Resolution30Min` |
| 45 minutes | `stocks.Resolution45Min` |
| 1 hour | `stocks.Resolution1Hour` |
| 2 hours | `stocks.Resolution2Hour` |
| 4 hours | `stocks.Resolution4Hour` |
| Daily | `stocks.ResolutionDaily` |
| Weekly | `stocks.ResolutionWeekly` |
| Monthly | `stocks.ResolutionMonthly` |
| Yearly | `stocks.ResolutionYearly` |

Funds and market-status history use their own package's resolution/window
constants; see the [documentation](https://www.marketdata.app/docs/sdk/go).

## Error Handling

```go
quote, _, err := client.Stocks.Quote(ctx, "INVALID")
if err != nil {
    // Check for specific error types
    var apiErr *marketdata.APIError
    if errors.As(err, &apiErr) {
        fmt.Printf("API error: %s\n", apiErr.Message)
    }

    // Check for rate limiting
    if errors.Is(err, marketdata.ErrRateLimited) {
        fmt.Println("Rate limited!")
    }

    // Check if error is retryable
    var sdkErr marketdata.Error
    if errors.As(err, &sdkErr) && sdkErr.Retryable() {
        // Could retry the request
    }
}
```

## String Conversion

Every response type implements `fmt.Stringer` with a readable one-line
summary, so values can be printed directly:

```go
quote, _, _ := client.Stocks.Quote(ctx, "AAPL")
fmt.Println(quote)
// AAPL Last: $302.77 Bid: 302.75 (2) Ask: 302.79 (3) Mid: 302.77 Chg: -0.65 (-0.21%) Vol: 41203110 Updated: 2026-08-04 10:15:04

status, _, _ := client.Markets.Status(ctx)
fmt.Println(status)
// 2026-08-04 open Open: true
```

Percentage fields print multiplied by 100 (the API sends fractions); nil
pointer fields print as n/a.

## Rate Limits

```go
// Check current rate limits
limits := client.RateLimits()
fmt.Printf("Used: %d/%d\n", limits.Consumed, limits.Limit)
fmt.Printf("Remaining: %d\n", limits.Remaining)
```

## Examples

The [examples/](examples/) directory contains nine runnable programs — from a
copy-paste [quick start](examples/basic/) to full-screen terminal apps
([stockterm](examples/stockterm/), [optionterm](examples/optionterm/)) that
between them exercise every SDK method. See the
[examples index](examples/README.md) for what each one shows and how to run it.

## Architecture

The SDK follows idiomatic Go patterns:

- **Functional Options** for configuration
- **Context-First** method signatures
- **Interface-Based** design for testability
- **Modular Resources** for organization

See the [Architecture Decision Records](docs/adr/) for detailed design decisions.

## Migration from v1

If you're migrating from the v1 SDK, see the [Migration Guide](docs/MIGRATION.md).

Key changes:
- No global singleton - create clients explicitly
- No `init()` side effects
- Context required for all API methods
- Functional options instead of method chaining
- Structured error types

## License

MIT - See [LICENSE](LICENSE) for details.

## Support

- [Documentation](https://www.marketdata.app/docs/sdk/go)
- [API Reference](https://www.marketdata.app/docs/api)
- [GitHub Issues](https://github.com/MarketDataApp/sdk-go/issues)
