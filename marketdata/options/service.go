package options

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/fanout"
	"github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/response"
	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

// Service provides access to the Market Data options endpoints. It is not
// meant to be constructed directly; use the Options field of the marketdata
// client, which shares the client's HTTP transport, rate-limit tracking, and
// concurrency pool.
type Service struct {
	http *http.Client
}

// NewService creates a new options service backed by the given HTTP client.
// Most callers should use the Options service on a marketdata client rather
// than calling NewService directly.
func NewService(httpClient *http.Client) *Service {
	return &Service{
		http: httpClient,
	}
}

// Chain fetches the options chain for an underlying stock symbol, returning
// a real-time or historical [OptionsChain] with one [OptionQuote] per
// contract. The symbol is required; all other parameters are optional and
// supplied as [ChainOption] values.
//
// Without an expiry filter the API returns only the front-month (nearest)
// expiration, not the whole chain. To fetch every listed expiration, pass
// options.AllExpirations() to [WithExpiry] — and expect a much larger
// response, billed accordingly.
//
// The two groups of mutually-exclusive API parameters are each expressed as a
// single sealed-union option, so the API's silently-conflicting combinations
// cannot be written: [WithStrike] selects contracts by strike or delta, and
// [WithExpiry] selects expirations by date, dte, month, year, or an expiration
// range. [WithChainDate] requests a historical snapshot as of a day and is
// independent. Free filters such as [WithSide], [WithStrikeLimit], [WithRange],
// the bid/ask and liquidity filters, and the expiration-type filters combine
// freely. See the ChainOption constructors in this package for the complete set.
//
// Option values that cannot be encoded in the type system (a strike or delta
// out of range, a month outside 1..12, a malformed date range) are rejected
// with a [sdkerrors.ValidationError] before any request is made.
//
// A symbol the API does not recognize is reported as a [NotFoundError]. A
// valid request that simply matches no contracts is not an error: Chain
// returns an empty chain (safe to range), a nil error, and a response whose
// NoData field is true. See the "Missing Data and Unknown Symbols" section
// of the marketdata package documentation.
//
// API documentation: https://www.marketdata.app/docs/api/options/chain
//
// Example:
//
//	chain, _, err := client.Options.Chain(ctx, "AAPL",
//	    options.WithExpiry(options.OnExpiration(time.Now().AddDate(0, 1, 0))),
//	    options.WithStrike(options.StrikeRange(150, 160)),
//	)
func (s *Service) Chain(ctx context.Context, symbol string, opts ...ChainOption) (*OptionsChain, *response.Response, error) {
	path, params, err := chainPath(symbol, opts)
	if err != nil {
		return nil, nil, err
	}

	var resp quoteResponse
	httpResp, err := s.http.Get(ctx, path, params, &resp)
	if err != nil {
		return nil, nil, err
	}

	// A markerless 404 means the query was valid and matched nothing (a
	// symbol that does not exist is an error now, see
	// http.notFoundNamesAnError). The correct answer to "which contracts
	// match?" is then the empty set, so return the same empty-but-usable
	// chain a 200 with no rows already produces — the caller can range it.
	// resp is the zero value here: Get returns a 404 without decoding.
	if response.IsNoData(httpResp.StatusCode) {
		return resp.toOptionsChain(), response.NewNoData(httpResp), nil
	}

	if resp.Status != "ok" {
		return nil, nil, httpResp.StatusError(resp.Status)
	}

	return resp.toOptionsChain(), response.New(httpResp), nil
}

// Expirations fetches the expiration dates with listed option contracts for
// an underlying stock symbol. The returned [Expirations] carries the dates
// in Eastern time plus the server's response-level update time, mirroring
// the API response. The symbol is required. The optional [ExpirationOption]
// values [WithExpirationStrike] (limit to expirations offering a given
// strike) and [WithExpirationDate] (list the expirations that were available
// on a past date) refine the request.
//
// A request with no listed expirations is not an error: Expirations returns
// an empty result (Dates is safe to range), a nil error, and a response
// whose NoData field is true. Note that options/expirations answers an
// unknown symbol with an unmarked 404 too, so an unrecognized symbol is
// reported the same way rather than as a [NotFoundError] — the API sends
// nothing that tells the two apart.
//
// API documentation: https://www.marketdata.app/docs/api/options/expirations
//
// Example:
//
//	exps, _, err := client.Options.Expirations(ctx, "AAPL")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if exps != nil {
//	    for _, d := range exps.Dates { ... }
//	}
func (s *Service) Expirations(ctx context.Context, symbol string, opts ...ExpirationOption) (*Expirations, *response.Response, error) {
	path, params, err := expirationsPath(symbol, opts)
	if err != nil {
		return nil, nil, err
	}
	// Serves this method's decoder, not the request: the wire dates have to
	// be unix for expirationsResponse to parse them. The builder leaves it
	// out on purpose so the formatted facets, whose output IS the wire text,
	// keep the user's own date format. See expirationsPath.
	params.Set("dateformat", "unix")

	var resp expirationsResponse
	httpResp, err := s.http.Get(ctx, path, params, &resp)
	if err != nil {
		return nil, nil, err
	}

	// Empty answer, not a nil one — same reasoning as Service.Chain: an
	// empty Dates slice ranges zero times, a nil *Expirations panics.
	if response.IsNoData(httpResp.StatusCode) {
		return resp.toExpirations(), response.NewNoData(httpResp), nil
	}

	if resp.Status != "ok" {
		return nil, nil, httpResp.StatusError(resp.Status)
	}

	return resp.toExpirations(), response.New(httpResp), nil
}

// Quote fetches a real-time or historical [OptionQuote] for a single option
// contract, identified by its OCC option symbol (for example
// "AAPL250117C00150000"). The option symbol is required; if it is not known,
// [Service.Lookup] can resolve one from the underlying, expiration, strike,
// and type. The optional [WithOptionQuoteWindow] carries a single
// [OptionQuoteWindow] value — options.QuoteOnDate(t), options.QuoteRange(from,
// to), options.QuoteLastN(n), or options.QuoteLastNUntil(n, to) — to request a
// historical quote; because it is one value, the API's mutually-exclusive date,
// from/to, and countback parameters cannot be combined (the API returns HTTP
// 400 when both a date and a from/to are sent).
//
// A malformed quote window (a zero date, a from after its to, or a countback
// at or below zero) is rejected with a [sdkerrors.ValidationError] before any
// request is made.
//
// Quote returns exactly one quote, as its name says. A window that selects
// several days — a range, or a countback greater than one — makes the API
// return one row per day, of which Quote keeps only the first; use
// [Service.QuoteHistory] to receive all of them.
//
// An OCC symbol matching no contract is reported as a [NotFoundError]. If
// the contract exists but the API has no data for the requested window,
// Quote returns a nil quote, a response whose NoData field is true, and a
// nil error — an OptionQuote has no meaningful empty value, so callers must
// check for nil. See the "Missing Data and Unknown Symbols" section of the
// marketdata package documentation.
//
// API documentation: https://www.marketdata.app/docs/api/options/quotes
//
// Example:
//
//	quote, _, err := client.Options.Quote(ctx, "AAPL230120C00150000")
func (s *Service) Quote(ctx context.Context, optionSymbol string, opts ...QuoteOption) (*OptionQuote, *response.Response, error) {
	resp, httpResp, err := s.quoteRaw(ctx, optionSymbol, opts...)
	if err != nil {
		return nil, nil, err
	}
	if response.IsNoData(httpResp.StatusCode) {
		return nil, response.NewNoData(httpResp), nil
	}
	return resp.toOptionQuote(), response.New(httpResp), nil
}

// quoteRaw issues the single-contract quote request shared by [Service.Quote]
// and [Service.QuoteHistory]: it validates the arguments, applies the quote
// window, and returns the decoded wire response together with the HTTP
// response. Callers must check the HTTP status for no-data before reading the
// payload. A non-ok API status is reported as an error here, so callers never
// see one.
func (s *Service) quoteRaw(ctx context.Context, optionSymbol string, opts ...QuoteOption) (*quoteResponse, *http.Response, error) {
	path, params, err := quotePath(optionSymbol, opts)
	if err != nil {
		return nil, nil, err
	}

	var resp quoteResponse
	httpResp, err := s.http.Get(ctx, path, params, &resp)
	if err != nil {
		// The HTTP response goes back even on the error path: a 404 naming
		// an unknown contract carries rate-limit metadata a caller that
		// recovers from it still needs. See quoteAllowMissing.
		return nil, httpResp, err
	}

	// Handle 404 as no-data (not an error); the caller turns it into a
	// no-data response of the right shape.
	if response.IsNoData(httpResp.StatusCode) {
		return &resp, httpResp, nil
	}

	if resp.Status != "ok" {
		return nil, nil, httpResp.StatusError(resp.Status)
	}

	return &resp, httpResp, nil
}

// quoteAllowMissing is [Service.Quote] with the unknown-contract case folded
// back into the no-data shape.
//
// It exists for the batch methods. In a batch, "this contract does not
// exist" is information about one symbol, not a failure of the request:
// [Service.Quotes] documents that such symbols are omitted and
// [Service.QuotesBySymbol] that they come back as a nil entry, and letting
// one bad OCC symbol cancel its siblings would break both promises while
// throwing away the credits already spent on them. The single-contract
// [Service.Quote] keeps reporting it as an error, because there the caller
// asked about exactly that contract and nothing else.
func (s *Service) quoteAllowMissing(ctx context.Context, optionSymbol string, opts ...QuoteOption) (*OptionQuote, *response.Response, error) {
	resp, httpResp, err := s.quoteRaw(ctx, optionSymbol, opts...)
	if err != nil {
		if httpResp == nil || !errors.Is(err, sdkerrors.ErrNotFound) {
			return nil, nil, err
		}
		return nil, response.NewNoData(httpResp), nil
	}
	if response.IsNoData(httpResp.StatusCode) {
		return nil, response.NewNoData(httpResp), nil
	}
	return resp.toOptionQuote(), response.New(httpResp), nil
}

// QuoteHistory fetches every quote the API returns for a single option
// contract, as a slice in the order the API sent them. It takes the same
// arguments and options as [Service.Quote].
//
// It exists because a historical window can select more than one quote: a
// [QuoteRange] spanning several days, or a [QuoteLastN]/[QuoteLastNUntil]
// countback, returns one row per day. Quote returns only the first of those
// rows, since its contract is a single quote; QuoteHistory returns all of
// them. For a current quote, or a window that selects a single day, the two
// are equivalent and QuoteHistory returns a one-element slice.
//
// An OCC symbol matching no contract is reported as a [NotFoundError]. If
// the contract exists but the window selects nothing, QuoteHistory returns
// a nil slice (safe to range), a response whose NoData field is true, and a
// nil error.
//
// API documentation: https://www.marketdata.app/docs/api/options/quotes
//
// Example:
//
//	quotes, _, err := client.Options.QuoteHistory(ctx, "AAPL260821C00300000",
//	    options.WithOptionQuoteWindow(options.QuoteLastN(5)))
func (s *Service) QuoteHistory(ctx context.Context, optionSymbol string, opts ...QuoteOption) ([]OptionQuote, *response.Response, error) {
	resp, httpResp, err := s.quoteRaw(ctx, optionSymbol, opts...)
	if err != nil {
		return nil, nil, err
	}
	if response.IsNoData(httpResp.StatusCode) {
		return nil, response.NewNoData(httpResp), nil
	}
	return resp.toOptionQuotes(), response.New(httpResp), nil
}

// GetQuoteHistory is a convenience wrapper for [Service.QuoteHistory] that
// uses context.Background() and discards the response metadata. When the API
// returns 404, GetQuoteHistory returns a nil slice and a nil error.
func (s *Service) GetQuoteHistory(optionSymbol string, opts ...QuoteOption) ([]OptionQuote, error) {
	q, _, err := s.QuoteHistory(context.Background(), optionSymbol, opts...)
	return q, err
}

// Lookup resolves an option contract to its OCC option symbol. It takes the
// underlying stock symbol, the expiration date, the strike price, and an
// [OptionType] ([Call] or [Put]), all of which are required, and returns the
// OCC symbol string (for example "AAPL250117C00150000") suitable for use
// with [Service.Quote] and [Service.Quotes].
//
// All four arguments are validated before any request is made: a missing
// underlying, a zero expiration, a strike at or below zero, or an option type
// other than [Call] or [Put] is rejected with a [sdkerrors.ValidationError].
// Without those checks an unset argument would be interpolated into the query
// verbatim, and the resulting 404 would be reported as no-data — making a
// malformed call indistinguishable from a contract that does not exist.
//
// If the API cannot resolve the contract (HTTP 404), Lookup returns an empty
// string, a response whose NoData field is true, and a nil error.
//
// API documentation: https://www.marketdata.app/docs/api/options/lookup
//
// Example:
//
//	symbol, _, err := client.Options.Lookup(ctx, "AAPL", expDate, 150.0, options.Call)
func (s *Service) Lookup(ctx context.Context, underlying string, expiration time.Time, strike float64, optionType OptionType) (string, *response.Response, error) {
	if underlying == "" {
		return "", nil, &sdkerrors.ValidationError{Field: "underlying", Message: "underlying symbol is required"}
	}
	if expiration.IsZero() {
		return "", nil, &sdkerrors.ValidationError{Field: "expiration", Message: "expiration date is required"}
	}
	if strike <= 0 {
		return "", nil, &sdkerrors.ValidationError{Field: "strike", Message: "strike must be greater than zero"}
	}
	if optionType != Call && optionType != Put {
		return "", nil, &sdkerrors.ValidationError{Field: "optionType", Message: "option type must be options.Call or options.Put"}
	}

	// The lookup endpoint takes the whole human-readable query as a
	// single URL path segment, e.g. "AAPL 2026-08-21 220 Call".
	// Query parameters are not supported by the API.
	query := underlying + " " + expiration.Format("2006-01-02") + " " + formatFloat(strike) + " " + string(optionType)

	return s.LookupQuery(ctx, query)
}

// LookupQuery resolves an option contract to its OCC option symbol from a
// free-form, human-readable description, exactly as the endpoint accepts it —
// for example "AAPL 7/26/23 $200 Call". The query is required and is sent as a
// single URL path segment; the endpoint does its own parsing, so the accepted
// phrasings are the API's, not the SDK's.
//
// [Service.Lookup] is the typed alternative for the common case where the
// underlying, expiration, strike, and type are already known separately; it
// validates those four and assembles the query. Use LookupQuery when the
// description arrives as text — from a user, a spreadsheet cell, or a broker
// export — and cannot be decomposed first.
//
// If the API cannot resolve the contract (HTTP 404), LookupQuery returns an
// empty string, a response whose NoData field is true, and a nil error.
//
// API documentation: https://www.marketdata.app/docs/api/options/lookup
//
// Example:
//
//	symbol, _, err := client.Options.LookupQuery(ctx, "AAPL 7/26/23 $200 Call")
func (s *Service) LookupQuery(ctx context.Context, query string) (string, *response.Response, error) {
	if strings.TrimSpace(query) == "" {
		return "", nil, &sdkerrors.ValidationError{Field: "query", Message: "lookup query is required"}
	}

	var resp lookupResponse
	httpResp, err := s.http.Get(ctx, "options/lookup/"+http.PathSegment(query)+"/", nil, &resp)
	if err != nil {
		return "", nil, err
	}

	if response.IsNoData(httpResp.StatusCode) {
		return "", response.NewNoData(httpResp), nil
	}

	if resp.Status != "ok" {
		return "", nil, httpResp.StatusError(resp.Status)
	}

	return resp.OptionSymbol, response.New(httpResp), nil
}

// Quotes fetches quotes for multiple option contracts identified by their
// OCC option symbols. At least one symbol is required. The optional
// [QuoteOption] values are the same ones [Service.Quote] accepts and
// apply to every symbol in the batch, so [WithOptionQuoteWindow] requests the
// same historical window across the whole set. The method fans out
// one concurrent [Service.Quote] request per symbol; the goroutines draw
// slots from the client's shared concurrency pool (at most 50 in-flight
// requests per client across all services), so passing many symbols will
// not exceed that limit. The results are merged into a single slice in the
// order the symbols were given.
//
// Symbols the API has no data for, and symbols naming a contract it does not
// recognize, are both omitted from the result rather than producing an error,
// so the returned slice may be shorter than the input and gives no indication
// of which symbols were dropped; use [Service.QuotesBySymbol] when that
// distinction matters. This is where the batch methods differ from
// [Service.Quote], which reports an unknown contract as a [NotFoundError]:
// in a batch a bad symbol is information about that symbol, not a failure of
// the whole request.
// If any request fails with a real error, Quotes returns that error and no
// quotes. The returned response corresponds to one of the individual
// requests, not an aggregate.
//
// API documentation: https://www.marketdata.app/docs/api/options/quotes
//
// Example:
//
//	quotes, _, err := client.Options.Quotes(ctx, []string{"AAPL230120C00150000", "AAPL230120P00150000"})
func (s *Service) Quotes(ctx context.Context, optionSymbols []string, opts ...QuoteOption) ([]OptionQuote, *response.Response, error) {
	quotes, resp, err := s.quoteEach(ctx, optionSymbols, opts...)
	if err != nil {
		return nil, nil, err
	}

	var out []OptionQuote
	for _, q := range quotes {
		if q != nil {
			out = append(out, *q)
		}
	}
	return out, resp, nil
}

// QuotesBySymbol fetches quotes for multiple option contracts and returns them
// keyed by the OCC option symbol that was requested. At least one symbol is
// required. It fetches exactly like [Service.Quotes] — one concurrent request
// per symbol, drawn from the client's shared 50-slot pool, with the first hard
// error cancelling its siblings.
//
// It accepts the same [QuoteOption] values as [Service.Quote], applied
// to every symbol.
//
// The difference is what happens to contracts the API has no data for, or
// does not recognize at all. Quotes omits them, so the returned slice is
// simply shorter than the input and the caller cannot tell which symbols were
// dropped. QuotesBySymbol returns one map entry for every requested symbol,
// with a nil value for those, so a missing contract is distinguishable from
// one that was never asked for. Repeated symbols collapse into a single
// entry. Neither method reports an unknown contract as an error the way
// [Service.Quote] does — see Quotes for why.
//
// If any request fails with a real error, QuotesBySymbol returns that error and
// no quotes. The returned response corresponds to one of the individual
// requests, not an aggregate.
//
// API documentation: https://www.marketdata.app/docs/api/options/quotes
//
// Example:
//
//	quotes, _, err := client.Options.QuotesBySymbol(ctx, []string{"AAPL230120C00150000", "AAPL230120P00150000"})
//	for symbol, q := range quotes {
//	    if q == nil {
//	        log.Printf("%s: no data", symbol)
//	        continue
//	    }
//	    ...
//	}
func (s *Service) QuotesBySymbol(ctx context.Context, optionSymbols []string, opts ...QuoteOption) (map[string]*OptionQuote, *response.Response, error) {
	quotes, resp, err := s.quoteEach(ctx, optionSymbols, opts...)
	if err != nil {
		return nil, nil, err
	}

	out := make(map[string]*OptionQuote, len(optionSymbols))
	for i, sym := range optionSymbols {
		// A duplicate symbol is fetched twice but keyed once; keep the first
		// non-nil result so a repeated symbol never downgrades to nil.
		if existing, ok := out[sym]; ok && existing != nil {
			continue
		}
		out[sym] = quotes[i]
	}
	return out, resp, nil
}

// quoteEach fetches one quote per symbol and returns a slice positionally
// parallel to optionSymbols, holding nil where the API reported no data. It
// carries the fan-out, cancellation, and error-selection policy shared by
// [Service.Quotes] and [Service.QuotesBySymbol].
func (s *Service) quoteEach(ctx context.Context, optionSymbols []string, opts ...QuoteOption) ([]*OptionQuote, *response.Response, error) {
	if len(optionSymbols) == 0 {
		return nil, nil, &sdkerrors.ValidationError{Field: "optionSymbols", Message: "at least one option symbol is required"}
	}

	// Single symbol: no concurrency needed
	if len(optionSymbols) == 1 {
		q, resp, err := s.quoteAllowMissing(ctx, optionSymbols[0], opts...)
		if err != nil {
			return nil, nil, err
		}
		return []*OptionQuote{q}, resp, nil
	}

	// Multiple symbols: fetch concurrently, abandoning the rest on the first
	// failure (ADR-014, see fanout.Run): once the batch is going to fail
	// there is no reason to keep spending API credits on the remaining
	// symbols.
	type symbolResult struct {
		quote *OptionQuote
		resp  *response.Response
	}
	results, err := fanout.Run(ctx, len(optionSymbols), func(ctx context.Context, i int) (symbolResult, error) {
		q, r, err := s.quoteAllowMissing(ctx, optionSymbols[i], opts...)
		return symbolResult{quote: q, resp: r}, err
	})
	if err != nil {
		return nil, nil, err
	}

	// Keep each quote at its symbol's index so callers can map results back
	// to inputs.
	quotes := make([]*OptionQuote, len(optionSymbols))
	var lastResp *response.Response
	for i, r := range results {
		quotes[i] = r.quote
		if r.resp != nil {
			lastResp = r.resp
		}
	}

	return quotes, lastResp, nil
}

// --- Convenience methods (no context, no *Response) ---

// GetChain is a convenience wrapper for [Service.Chain] that uses
// context.Background() and discards the response metadata. It accepts the
// same required symbol and optional [ChainOption] filters, and shares
// Chain's no-data behavior: when the API returns 404, GetChain returns a
// nil chain and a nil error.
func (s *Service) GetChain(symbol string, opts ...ChainOption) (*OptionsChain, error) {
	c, _, err := s.Chain(context.Background(), symbol, opts...)
	return c, err
}

// GetExpirations is a convenience wrapper for [Service.Expirations] that
// uses context.Background() and discards the response metadata. When the
// API returns 404, GetExpirations returns nil and a nil error.
func (s *Service) GetExpirations(symbol string, opts ...ExpirationOption) (*Expirations, error) {
	e, _, err := s.Expirations(context.Background(), symbol, opts...)
	return e, err
}

// GetQuote is a convenience wrapper for [Service.Quote] that uses
// context.Background() and discards the response metadata. When the API
// returns 404, GetQuote returns a nil quote and a nil error.
func (s *Service) GetQuote(optionSymbol string, opts ...QuoteOption) (*OptionQuote, error) {
	q, _, err := s.Quote(context.Background(), optionSymbol, opts...)
	return q, err
}

// GetQuotes is a convenience wrapper for [Service.Quotes] that uses
// context.Background() and discards the response metadata. Like Quotes, it
// fetches the symbols concurrently and omits symbols with no data from the
// result.
func (s *Service) GetQuotes(optionSymbols []string, opts ...QuoteOption) ([]OptionQuote, error) {
	q, _, err := s.Quotes(context.Background(), optionSymbols, opts...)
	return q, err
}

// GetQuotesBySymbol is a convenience wrapper for [Service.QuotesBySymbol] that
// uses context.Background() and discards the response metadata. Like
// QuotesBySymbol, it returns one entry per requested symbol, with a nil value
// where the API had no data.
func (s *Service) GetQuotesBySymbol(optionSymbols []string, opts ...QuoteOption) (map[string]*OptionQuote, error) {
	q, _, err := s.QuotesBySymbol(context.Background(), optionSymbols, opts...)
	return q, err
}

// GetLookupQuery is a convenience wrapper for [Service.LookupQuery] that uses
// context.Background() and discards the response metadata. When the API cannot
// resolve the contract (404), GetLookupQuery returns an empty string and a nil
// error.
func (s *Service) GetLookupQuery(query string) (string, error) {
	sym, _, err := s.LookupQuery(context.Background(), query)
	return sym, err
}

// GetLookup is a convenience wrapper for [Service.Lookup] that uses
// context.Background() and discards the response metadata. When the API
// cannot resolve the contract (404), GetLookup returns an empty string and
// a nil error.
func (s *Service) GetLookup(underlying string, expiration time.Time, strike float64, optionType OptionType) (string, error) {
	sym, _, err := s.Lookup(context.Background(), underlying, expiration, strike, optionType)
	return sym, err
}
