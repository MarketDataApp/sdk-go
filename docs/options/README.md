# Options

Access options market data with the Go SDK: full option chains, expiration
dates, single and multi-contract quotes, and OCC option symbol lookup. Each
endpoint exposes a context-aware method that returns typed data plus a
`*response.Response` (raw body and rate-limit metadata) and a convenience
`Get*` wrapper that uses a background context.

## Options Endpoints

- [Lookup](./lookup.md)
- [Expirations](./expirations.md)
- [Option Chain](./chain.md)
- [Option Quotes](./quotes.md)
