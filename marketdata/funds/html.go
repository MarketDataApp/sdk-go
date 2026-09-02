package funds

import (
	"context"

	"github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/response"
)

// htmlService is the HTML facet of funds — built but not exposed to
// consumers (the backend serves no HTML for any data endpoint today,
// verified live: a format=html request 404s). Package-private so it can be
// exercised by tests; export asHTML and this type when the API adds
// format=html support. See ADR-018.
type htmlService struct {
	http *http.Client
}

// asHTML returns the HTML facet of this service. Unexported until the API
// supports format=html — see ADR-018.
func (s *Service) asHTML() *htmlService {
	return &htmlService{http: s.http}
}

func (s *htmlService) Candles(ctx context.Context, symbol string, opts ...CandleOption) (*response.HTMLResponse, error) {
	path, params, err := candlesPath(symbol, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormatted(ctx, s.http, path, params, "html", response.NewHTML)
}
