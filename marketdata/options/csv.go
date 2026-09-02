package options

import (
	"context"
	"net/url"

	"github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/response"
	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

// CSVService is the CSV facet of options, reached through [Service.AsCSV].
// Every endpoint here returns a [response.CSVResponse] carrying the API's
// raw CSV text — see ADR-018 for why this exists alongside the typed JSON
// methods on [Service]. The SDK does not parse the CSV into rows; callers
// get the text exactly as the API sent it.
//
// [Service.Lookup] has no CSV facet: it resolves to a single OCC symbol
// string, not tabular data.
type CSVService struct {
	http *http.Client
}

// AsCSV returns the CSV facet of this service.
func (s *Service) AsCSV() *CSVService {
	return &CSVService{http: s.http}
}

// Chain fetches the options chain for an underlying stock symbol as CSV.
// See [Service.Chain] for parameter and validation details.
func (s *CSVService) Chain(ctx context.Context, symbol string, opts ...ChainOption) (*response.CSVResponse, error) {
	path, params, err := chainPath(symbol, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormatted(ctx, s.http, path, params, "csv", response.NewCSV)
}

// Expirations fetches the expiration dates for an underlying stock symbol
// as CSV. See [Service.Expirations] for parameter and validation details.
func (s *CSVService) Expirations(ctx context.Context, symbol string, opts ...ExpirationOption) (*response.CSVResponse, error) {
	path, params, err := expirationsPath(symbol, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormatted(ctx, s.http, path, params, "csv", response.NewCSV)
}

// Quote fetches a single option contract's quote as CSV. See [Service.Quote]
// for parameter and validation details.
func (s *CSVService) Quote(ctx context.Context, optionSymbol string, opts ...QuoteOption) (*response.CSVResponse, error) {
	path, params, err := quotePath(optionSymbol, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormatted(ctx, s.http, path, params, "csv", response.NewCSV)
}

// Quotes fetches quotes for multiple option contracts as CSV, one request
// per symbol (mirroring [Service.Quotes]' own per-symbol fan-out), and
// returns a map keyed by option symbol instead of merging into one result
// — each contract's CSV text is independent, unlike candle chunks of the
// same time series. The first error cancels the remaining in-flight
// requests (ADR-014).
//
// The options apply to every symbol in the batch, so a historical window is
// expressible for a watchlist and not only for a single contract — the same
// reason [Service.Quotes] takes a slice plus options rather than a variadic
// symbol list.
//
// Unlike [Service.Quotes], there is no NoData omission: every requested
// symbol gets an entry in the map holding whatever the API returned for it
// (see ADR-018 — the CSV/HTML facets have no NoData concept).
func (s *CSVService) Quotes(ctx context.Context, optionSymbols []string, opts ...QuoteOption) (map[string]*response.CSVResponse, error) {
	if len(optionSymbols) == 0 {
		return nil, &sdkerrors.ValidationError{Field: "optionSymbols", Message: "at least one option symbol is required"}
	}
	return response.FetchFormattedMap(ctx, s.http, optionSymbols,
		func(sym string) (string, url.Values, error) { return quotePath(sym, opts) },
		"csv", response.NewCSV)
}
