package stocks

import (
	"context"

	"github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/response"
)

// htmlService is the HTML facet of stocks — built but not exposed to
// consumers (the backend serves no HTML for any data endpoint today,
// verified live: a format=html request 404s). Package-private so it can be
// exercised by tests; export asHTML and this type when the API adds
// format=html support. See ADR-018. Mirrors [CSVService] exactly, one
// method per endpoint, requesting format=html and returning
// [response.HTMLResponse] instead.
type htmlService struct {
	http *http.Client
}

// asHTML returns the HTML facet of this service. Unexported until the API
// supports format=html — see ADR-018.
func (s *Service) asHTML() *htmlService {
	return &htmlService{http: s.http}
}

func (s *htmlService) Quote(ctx context.Context, symbol string, opts ...QuoteOption) (*response.HTMLResponse, error) {
	path, params, err := quotePath(symbol, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormatted(ctx, s.http, path, params, "html", response.NewHTML)
}

func (s *htmlService) Quotes(ctx context.Context, symbols []string, opts ...QuotesOption) (*response.HTMLResponse, error) {
	path, params, err := quotesPath(symbols, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormatted(ctx, s.http, path, params, "html", response.NewHTML)
}

func (s *htmlService) Prices(ctx context.Context, symbols []string, opts ...PriceOption) (*response.HTMLResponse, error) {
	path, params, err := pricesPath(symbols, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormatted(ctx, s.http, path, params, "html", response.NewHTML)
}

func (s *htmlService) News(ctx context.Context, symbol string, opts ...NewsOption) (*response.HTMLResponse, error) {
	path, params, err := newsPath(symbol, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormatted(ctx, s.http, path, params, "html", response.NewHTML)
}

func (s *htmlService) Earnings(ctx context.Context, symbol string, opts ...EarningsOption) (*response.HTMLResponse, error) {
	path, params, err := earningsPath(symbol, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormatted(ctx, s.http, path, params, "html", response.NewHTML)
}

func (s *htmlService) Candles(ctx context.Context, symbol string, opts ...CandleOption) (*response.HTMLResponse, error) {
	path, chunkParams, err := candlesPath(symbol, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormattedChunked(ctx, s.http, path, chunkParams, "html", response.NewHTML)
}
