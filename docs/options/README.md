# Options (Go SDK)

Access options market data with the Go SDK: full option chains, expiration
dates, single and multi-contract quotes, and OCC option symbol lookup. Each
endpoint exposes a context-aware method that returns typed data plus a
`*response.Response` (raw body and rate-limit metadata) and a convenience
`Get*` wrapper that uses a background context.

## Options Endpoints

- [Lookup (Go SDK)](./lookup.md) — Resolve an option contract to its standard OCC option symbol with the Go SDK, from the underlying, expiration, strike price and option type.
- [Expirations (Go SDK)](./expirations.md) — List the expiration dates with listed option contracts for an underlying using the Go SDK Expirations method, with optional strike and date filters.
- [Option Chain (Go SDK)](./chain.md) — Fetch a current or historical option chain with the Go SDK Chain method, filtering by expiration, strike, side, moneyness and liquidity.
- [Option Quotes (Go SDK)](./quotes.md) — Fetch current or historical end-of-day option quotes with the Go SDK: Quote for one OCC contract or Quotes for many, each with a Get wrapper.
