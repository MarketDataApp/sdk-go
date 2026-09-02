package funds

import (
	"context"

	"github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/response"
)

// CSVService is the CSV facet of funds, reached through [Service.AsCSV].
// [Service.Candles] returns a [response.CSVResponse] carrying the API's raw
// CSV text — see ADR-018.
type CSVService struct {
	http *http.Client
}

// AsCSV returns the CSV facet of this service.
func (s *Service) AsCSV() *CSVService {
	return &CSVService{http: s.http}
}

// Candles fetches historical OHLC candles for a mutual fund as CSV. See
// [Service.Candles] for parameter and validation details.
func (s *CSVService) Candles(ctx context.Context, symbol string, opts ...CandleOption) (*response.CSVResponse, error) {
	path, params, err := candlesPath(symbol, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormatted(ctx, s.http, path, params, "csv", response.NewCSV)
}
