// Package marketdata provides the official Go SDK for the Market Data API,
// offering type-safe access to real-time and historical financial data,
// including stock quotes and candles, options chains, mutual fund prices,
// and market status. The full API reference is available at
// https://www.marketdata.app/docs/api/intro.
//
// # Quick Start
//
// With an API token in the MARKETDATA_TOKEN environment variable, fetching
// a quote takes three lines:
//
//	client, err := marketdata.NewClient()
//	defer client.Close()
//	quote, err := client.Stocks.GetQuote("AAPL")
//
// # Resource Services
//
// The [Client] groups API endpoints into one service per resource:
//
//   - Client.Stocks: quotes, bulk quotes, candles, bulk prices, earnings, and news
//   - Client.Options: option chains, expiration dates, option quotes, and symbol lookup
//   - Client.Funds: mutual fund candles
//   - Client.Markets: market status (open or closed) and status history
//   - Client.Utilities: API status, response headers, and account details
//
// Every service method has two forms: a context-first form such as
// Quote(ctx, "AAPL", opts...) that also returns per-request [Response]
// metadata, and a Get-prefixed convenience form such as GetQuote("AAPL")
// that uses a background context and returns only the decoded data.
//
// # Authentication and Configuration
//
// [NewClient] resolves the API token from, in order of priority, the
// [WithToken] option and the MARKETDATA_TOKEN environment variable. Before
// reading the environment, NewClient loads a .env file from the working
// directory if one exists; values from .env never override variables that
// are already set in the process environment. Configuration follows the
// same cascade throughout the SDK: .env file values, then environment
// variables, then client options passed to NewClient, then per-method
// options, with later tiers taking precedence.
//
// If no token is found anywhere, the client starts in demo mode: it logs a
// warning, skips token validation and rate limit initialization, and can
// access only the limited set of endpoints the API exposes without
// authentication. When a token is present, NewClient validates it against
// the API synchronously at startup unless [WithoutStartupValidation] is
// used.
//
// # Error Handling
//
// All failures are reported as typed errors that support the standard
// errors.Is and errors.As idioms. Each HTTP failure maps to a specific
// type, such as [AuthenticationError] (401), [RateLimitError] (429), or
// [ServerError] (501-599), and each type has a matching sentinel value,
// such as [ErrAuthentication] or [ErrRateLimited], for quick errors.Is
// checks. API-produced errors embed a [SupportContext] whose SupportInfo
// method formats the request ID, URL, status code, and timestamp into a
// block suitable for pasting into a Market Data support ticket:
//
//	var rlErr *marketdata.RateLimitError
//	if errors.As(err, &rlErr) {
//		fmt.Println(rlErr.SupportInfo())
//	}
//
// # Missing Data and Unknown Symbols
//
// The API answers two different situations with HTTP 404, and the SDK
// reports them differently.
//
// A question the API rejects — a symbol that does not exist, an OCC symbol
// matching no contract — comes back with an errmsg naming the problem. The
// SDK maps it to a [NotFoundError], so a typo fails loudly rather than
// reading as an empty result:
//
//	_, _, err := client.Stocks.Quote(ctx, "ZZZZQQ")
//	if errors.Is(err, marketdata.ErrNotFound) {
//		// the symbol does not exist
//	}
//
// A valid question whose answer is empty — a filter matching no contracts,
// a date window with no candles — comes back without that marker. It is not
// an error: the method returns a nil error and a [Response] whose NoData
// field is true, and the result itself is empty rather than absent wherever
// an empty value exists. Methods returning a slice return an empty slice,
// and the collection-shaped results ([options.OptionsChain],
// [options.Expirations], [utilities.Headers]) return an empty value that is
// safe to range:
//
//	chain, resp, err := client.Options.Chain(ctx, "AAPL", ...)
//	if err != nil {
//		return err
//	}
//	for _, contract := range chain.Options { // zero iterations when resp.NoData
//		...
//	}
//
// The five results that describe a single thing rather than a collection —
// [stocks.Quote], [options.OptionQuote], [markets.MarketStatus],
// [utilities.APIStatus] and [utilities.UserInfo] — have no meaningful empty
// value, since every field of a zero-valued one would read as real data
// (a price of zero, a closed market, an account with no credits). Those
// return a nil pointer, and callers should check it:
//
//	quote, resp, err := client.Stocks.Quote(ctx, "AAPL")
//	if err != nil {
//		return err
//	}
//	if quote == nil { // resp.NoData is true
//		return nil
//	}
//
// Not every endpoint supplies the marker: options/expirations and
// stocks/candles answer an unknown symbol with an unmarked 404,
// indistinguishable on the wire from an empty answer, so those report no
// data rather than an error.
//
// # Rate Limiting
//
// Rate limit information is available in two places. Every context-first
// method returns a [Response] whose RateLimit field carries the exact,
// request-scoped values from that response's headers. Separately,
// [Client.RateLimits] returns the client's running snapshot of the most
// recently observed state; it is convenient for monitoring but may lag
// behind when requests run concurrently.
//
// # Retries, Timeouts, and Concurrency
//
// Failed requests are retried with exponential backoff, but only for
// 501-599 status codes and transient network errors; 4xx responses and 500
// are never retried. The default is 3 retries with a 1s, 2s, 4s backoff
// schedule, configurable via [WithMaxRetries]. Before each retry the SDK
// consults the API status endpoint and aborts early if the service is
// reported offline.
//
// Timeouts are fixed and not configurable: 99 seconds per request and 2
// seconds for the TCP connection dial. The client also limits itself to at
// most 50 concurrent in-flight requests through an internal pool; calls
// beyond that block until a slot frees, so the client is safe to share
// across many goroutines.
package marketdata
