# ADR-005: Response Types

## Status
Accepted (Revised)

## Context
Go SDKs need clear conventions for API response types:
- What types should methods return?
- How should collections be handled?
- Should response metadata be included?

## Decision

### Wire Format Decoding

The MarketData API returns compressed, array-keyed JSON where each field is an array of values rather than an array of objects. The SDK decodes this into user-usable typed structs before returning results.

**API wire format (compressed):**
```json
{
  "s": "ok",
  "symbol": ["AAPL"],
  "last": [150.25],
  "bid": [150.20],
  "ask": [150.30],
  "volume": [45000000],
  "updated": [1674000000]
}
```

**SDK returns (decoded):**
```go
&Quote{
    Symbol:  "AAPL",
    Last:    150.25,
    Bid:     150.20,
    Ask:     150.30,
    Volume:  45000000,
    Updated: time.Date(2023, 1, 18, 2, 40, 0, 0, easternLoc),
}
```

Each resource package defines unexported response types that mirror the wire format, with `toXxx()` conversion methods that:
1. Extract values from parallel arrays by index (using safe-access helpers to avoid panics)
2. Convert Unix timestamps to `time.Time` in US/Eastern timezone
3. Map API status strings to typed Go values
4. Return `nil` for nil responses

Raw compressed payloads are never exposed as the default SDK output. Users work exclusively with typed Go structs.

### Date and Timezone Handling

All Unix timestamps returned by the API are converted to `time.Time` and normalized to the **US/Eastern** timezone (`America/New_York`) by default, matching the MarketData API's convention:

```go
var easternLoc *time.Location

func init() {
    easternLoc, _ = time.LoadLocation("America/New_York")
}

// Used in all response conversion methods
func toEasternTime(unix int64) time.Time {
    return time.Unix(unix, 0).In(easternLoc)
}
```

This applies to:
- Quote timestamps (`Updated`, `Timestamp`)
- Candle timestamps (`Time`)
- Market status dates
- Any other time-bearing fields

The `dateformat` universal parameter (ADR-013) controls the format the API returns on the wire. Regardless of wire format, the SDK always parses to `time.Time` in US/Eastern.

### Triple-Return Pattern: `(data, *Response, error)`

All service methods return three values: the typed data, a `*Response` carrying per-request metadata, and an error:

```go
func (s *Service) Quote(ctx context.Context, symbol string) (*Quote, *Response, error)
func (s *Service) Candles(ctx context.Context, symbol string, opts ...CandleOption) ([]Candle, *Response, error)
```

Single-item responses return `*Type`, collections return `[]Type`. The `*Response` is always the second return value.

This follows the established pattern used by widely-adopted Go API client libraries:
- **go-github**: `repos, resp, err := client.Repositories.List(ctx, org, opts)` — the most popular Go API client library, used extensively across the Go ecosystem
- **DigitalOcean godo**: `droplet, resp, err := client.Droplets.Get(ctx, id)`
- **Okta Go SDK**: `user, resp, err := client.User.GetUser(ctx, userID)`
- **Buildkite Go**: `build, resp, err := client.Builds.Get(org, pipeline, num)`

#### Why Triple-Return Over Generic Wrapper

An earlier design used a `Response[T]` generic wrapper with a `.Data` field. This was reconsidered because:

1. **The common case should be simplest.** Most callers want `quote.Last`, not `resp.Data.Last`. In idiomatic Go, the most frequent operation should require the least ceremony.
2. **Go favors explicit return values over wrapper types.** Multiple return values are a core Go idiom — the `(value, error)` pattern is universal. Extending to `(value, metadata, error)` is a natural progression.
3. **Callers who don't need metadata can ignore it.** `quote, _, err := client.Stocks.Quote(ctx, "AAPL")` — the blank identifier is idiomatic and communicates intent.
4. **Generic type aliases require Go 1.23+.** The SDK targets Go 1.22, making `Response[T]` difficult to re-export cleanly across packages without generic type aliases.
5. **Precedent matters.** The triple-return pattern is proven in production across the most widely-used Go API client libraries.

#### The Response Struct

```go
// Response carries per-request metadata returned alongside typed data.
// Most callers can ignore it with a blank identifier: quote, _, err := ...
//
// It embeds *http.Response, giving callers escape-hatch access to raw
// headers, status codes, and the underlying response when needed.
// This follows the go-github pattern.
type Response struct {
    *http.Response // embedded — raw access to headers, status, etc.

    // NoData is true when a 404 indicated no data was available (not an error).
    NoData bool

    // RateLimit contains per-request rate limit metadata (request-scoped).
    RateLimit RateLimitMeta
}

type RateLimitMeta struct {
    Limit     int
    Remaining int
    Consumed  int
    ResetAt   time.Time
}

// Format detection (uses embedded Response.Header Content-Type)
func (r *Response) IsJSON() bool { ... }
func (r *Response) IsCSV() bool  { ... }
func (r *Response) IsHTML() bool { ... }

// File saving
func (r *Response) SaveToFile(path string) error { ... }
```

The embedded `*http.Response` provides `resp.StatusCode`, `resp.Header`, `resp.Body`, and all other standard fields. The SDK adds parsed convenience fields (`RateLimit`, `NoData`) so callers don't need to parse headers manually.

#### Usage

```go
// Common case — ignore metadata
quote, _, err := client.Stocks.Quote(ctx, "AAPL")
if err != nil { ... }
fmt.Printf("AAPL: $%.2f\n", quote.Last)

// When metadata matters
candles, resp, err := client.Stocks.Candles(ctx, "AAPL",
    stocks.WithResolution(stocks.ResolutionDaily),
)
if err != nil { ... }

// Check for no-data (404)
if resp.NoData {
    fmt.Println("No data available")
    return
}

// Per-request rate limit state
fmt.Printf("Credits remaining: %d\n", resp.RateLimit.Remaining)

// Save to file
resp.SaveToFile("/tmp/candles.json")

// Format detection
if resp.IsJSON() { ... }
```

**Note on concurrency:** `resp.RateLimit` is request-scoped and deterministic. The client-level `client.RateLimits()` is a convenience latest-snapshot and is non-deterministic under concurrent use.

### Type Design Principles

**Exported fields with json tags:**
```go
type Quote struct {
    Symbol        string    `json:"symbol"`
    Last          float64   `json:"last"`
    Bid           float64   `json:"bid"`
    Ask           float64   `json:"ask"`
    Volume        int64     `json:"volume"`
    Updated       time.Time `json:"updated"`
}
```

**Helper methods for computed values:**
```go
func (q *Quote) Spread() float64 {
    return q.Ask - q.Bid
}

func (q *Quote) MidPrice() float64 {
    return (q.Bid + q.Ask) / 2
}
```

**Use stdlib types:**
- `time.Time` for timestamps (not Unix epochs or strings)
- `float64` for prices (standard for financial data in Go)
- `int64` for volumes

### Zero Values Are Meaningful

Go's zero values are used intentionally:
- `0` for prices means no data available
- Empty string for optional string fields
- `time.Time{}` (zero time) indicates missing timestamp

Users check with standard Go patterns:
```go
if quote.Bid == 0 {
    // No bid available
}
if quote.Updated.IsZero() {
    // No timestamp
}
```

## Consequences

### Positive
- Typed data returned directly — `quote.Last`, not `resp.Data.Last`
- Follows proven Go API client conventions (go-github, godo, Okta)
- Callers who don't need metadata use `_, _` — idiomatic and clean
- Per-response rate limit metadata is request-scoped and deterministic
- No-data detection via `resp.NoData` without error handling
- Format detection and file saving available on the Response struct
- No generic type constraints — works with Go 1.21+

### Negative
- Three return values instead of two (slightly more verbose signatures)
- Callers must handle or explicitly ignore the `*Response` value
- Response struct is not generic — no compile-time link between data type and response

### Mitigations
- Blank identifier `_` is idiomatic Go for ignored values
- Consistent pattern across all methods — learn once, apply everywhere
- Clear documentation and examples showing both `_, _` and full metadata usage

## References

- [go-github Response Pattern](https://pkg.go.dev/github.com/google/go-github/v56/github)
- [DigitalOcean godo](https://pkg.go.dev/github.com/digitalocean/godo)
- [Okta Go SDK](https://pkg.go.dev/github.com/okta/okta-sdk-golang/v2/okta)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
