# Options (Go SDK)

Access options market data with the Go SDK: full option chains, expiration
dates, single and multi-contract quotes, and OCC option symbol lookup. Each
endpoint exposes a context-aware method that returns typed data plus a
`*response.Response` (raw body and rate-limit metadata) and a convenience
`Get*` wrapper that uses a background context.

## Options Endpoints

- [Lookup (Go SDK)](./lookup.md) — Resolve an option contract to its standard OCC option symbol with the Go SDK, from the underlying, expiration, strike price and option type.
- [Expirations (Go SDK)](./expirations.md)
- [Option Chain (Go SDK)](./chain.md)
- [Option Quotes (Go SDK)](./quotes.md)
