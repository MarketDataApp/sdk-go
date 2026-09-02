# ADR-001: Package Structure

## Status
Accepted

## Context

The MarketData Go SDK v2 needs a well-organized package structure that:
- Follows idiomatic Go conventions
- Enables independent versioning and testing
- Provides a clean, intuitive API surface
- Supports future growth without breaking changes
- Avoids the flat structure of v1 that led to maintenance issues

## Decision Drivers

- **Idiomatic Go**: Follow patterns from AWS SDK Go v2, Stripe Go SDK
- **Modularity**: Each domain should be independently maintainable
- **Discoverability**: Users should easily find functionality
- **Testability**: Easy to mock and test individual components
- **Minimal API Surface**: Export only what's necessary

## Considered Options

### Option 1: Flat Package Structure (v1 approach)
```
github.com/marketdataapp/sdk-go/v2
├── client.go
├── stocks.go
├── options.go
├── funds.go
└── ...
```

**Pros**: Simple, single import
**Cons**: Monolithic, hard to maintain, cluttered namespace, v1 failure mode

### Option 2: Service-per-Package (AWS style)
```
github.com/marketdataapp/sdk-go/v2
├── config/
├── stocks/
├── options/
├── funds/
└── markets/
```

**Pros**: Maximum modularity, independent versioning
**Cons**: Multiple imports needed, complex for small API

### Option 3: Nested Resources under Main Package (Stripe style)
```
github.com/marketdataapp/sdk-go/v2
├── marketdata/
│   ├── client.go
│   ├── config.go
│   ├── errors.go
│   ├── stocks/
│   ├── options/
│   ├── funds/
│   └── markets/
└── internal/
```

**Pros**: Single main import, organized subpackages, clean separation
**Cons**: Nested structure requires careful API design

## Decision

We adopt **Option 3: Nested Resources under Main Package** with the following structure:

```
github.com/marketdataapp/sdk-go/v2
├── marketdata/              # Main package (public API)
│   ├── client.go            # MarketDataClient type
│   ├── config.go            # Configuration and options
│   ├── errors.go            # Error types
│   ├── stocks/              # Stocks resource
│   │   ├── quotes.go
│   │   ├── candles.go
│   │   └── ...
│   ├── options/             # Options resource
│   ├── funds/               # Funds resource
│   ├── markets/             # Markets resource
│   └── utilities/           # Utilities resource (status, headers, user)
├── internal/                # Private implementation
│   ├── http/                # HTTP client wrapper
│   ├── retry/               # Retry logic
│   └── ratelimit/           # Rate limiting
├── examples/                # Usage examples
└── docs/
    └── adr/                 # Architecture Decision Records
```

### Required SDK Method Coverage

Each resource package provides first-class methods for the capabilities defined in the SDK requirements (§2.2). Method names follow Go conventions (PascalCase exported):

| Resource     | Methods                                           | Notes                                      |
|--------------|---------------------------------------------------|--------------------------------------------|
| `stocks`     | `Prices`, `Quotes`, `Candles`, `Earnings`, `News` | Full coverage of stocks endpoints           |
| `options`    | `Chain`, `Expirations`, `Quote`, `Lookup`         | Full coverage of options endpoints          |
| `funds`      | `Candles`                                         | Single endpoint per requirements            |
| `markets`    | `Status`                                          | Single endpoint per requirements            |
| `utilities`  | `Status`, `Headers`, `User`                       | Diagnostics and account info                |

### Usage Pattern

```go
import "github.com/marketdataapp/sdk-go/v2/marketdata"

// Create client
client, err := marketdata.NewClient(
    marketdata.WithToken("your-token"),
)
defer client.Close()

// Stocks
prices, err := client.Stocks.Prices(ctx, "AAPL")
quotes, err := client.Stocks.Quotes(ctx, []string{"AAPL", "MSFT"})
candles, err := client.Stocks.Candles(ctx, "AAPL", stocks.WithResolution("D"))
earnings, err := client.Stocks.Earnings(ctx, "AAPL")
news, err := client.Stocks.News(ctx, "AAPL")

// Options
chain, err := client.Options.Chain(ctx, "AAPL")
exps, err := client.Options.Expirations(ctx, "AAPL")
optQuote, err := client.Options.Quote(ctx, "AAPL230616C00150000")
symbol, err := client.Options.Lookup(ctx, "AAPL", exp, 150.0, options.Call)

// Funds
fundCandles, err := client.Funds.Candles(ctx, "SPY", funds.WithResolution("D"))

// Markets
mktStatus, err := client.Markets.Status(ctx)

// Utilities
apiStatus, err := client.Utilities.Status(ctx)
user, err := client.Utilities.User(ctx)
headers, err := client.Utilities.Headers(ctx)
```

## Consequences

### Positive
- Single import for most use cases
- Clear separation between public API and internals
- Resources are discoverable via client methods
- Follows patterns users expect from modern Go SDKs
- Easy to add new resources without breaking changes

### Negative
- Nested packages require careful consideration of what to export
- Users who want only stocks still import the entire marketdata package
- Need to ensure no circular dependencies between packages

### Mitigations
- Use interfaces for cross-package dependencies
- Keep internal package strictly private
- Comprehensive documentation and examples

## References

- [AWS SDK for Go v2 Package Structure](https://github.com/aws/aws-sdk-go-v2)
- [Stripe Go SDK Structure](https://github.com/stripe/stripe-go)
- [Go Project Layout](https://github.com/golang-standards/project-layout)
