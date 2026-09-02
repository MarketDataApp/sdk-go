package options

import (
	"context"
	"net/url"

	"github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/response"
	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

// htmlService is the HTML facet of options — built but not exposed to
// consumers (the backend serves no HTML for any data endpoint today,
// verified live: a format=html request 404s). Package-private so it can be
// exercised by tests; export asHTML and this type when the API adds
// format=html support. See ADR-018. Mirrors [CSVService] exactly.
type htmlService struct {
	http *http.Client
}

// asHTML returns the HTML facet of this service. Unexported until the API
// supports format=html — see ADR-018.
func (s *Service) asHTML() *htmlService {
	return &htmlService{http: s.http}
}

func (s *htmlService) Chain(ctx context.Context, symbol string, opts ...ChainOption) (*response.HTMLResponse, error) {
	path, params, err := chainPath(symbol, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormatted(ctx, s.http, path, params, "html", response.NewHTML)
}

func (s *htmlService) Expirations(ctx context.Context, symbol string, opts ...ExpirationOption) (*response.HTMLResponse, error) {
	path, params, err := expirationsPath(symbol, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormatted(ctx, s.http, path, params, "html", response.NewHTML)
}

func (s *htmlService) Quote(ctx context.Context, optionSymbol string, opts ...QuoteOption) (*response.HTMLResponse, error) {
	path, params, err := quotePath(optionSymbol, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormatted(ctx, s.http, path, params, "html", response.NewHTML)
}

func (s *htmlService) Quotes(ctx context.Context, optionSymbols []string, opts ...QuoteOption) (map[string]*response.HTMLResponse, error) {
	if len(optionSymbols) == 0 {
		return nil, &sdkerrors.ValidationError{Field: "optionSymbols", Message: "at least one option symbol is required"}
	}
	return response.FetchFormattedMap(ctx, s.http, optionSymbols,
		func(sym string) (string, url.Values, error) { return quotePath(sym, opts) },
		"html", response.NewHTML)
}
