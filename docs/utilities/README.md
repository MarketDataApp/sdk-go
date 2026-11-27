# Utilities (Go SDK)

Report on the Market Data API itself rather than on market data: check API health and uptime, echo back your request headers for debugging, inspect the authenticated account's API credit state, and revoke the current token. The Go SDK exposes these endpoints through the `client.Utilities` service.

## Utilities Endpoints

- [API Status (Go SDK)](./api-status.md) — Check the Market Data API status and its historical uptime with the Go SDK Status and GetStatus methods on the Utilities service, with no parameters.
- [Headers (Go SDK)](./headers.md)
- [User (Go SDK)](./user.md)
- [Revoke Token (Go SDK)](./revoke-token.md)
