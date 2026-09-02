package markets

import (
	"context"

	"github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/response"
)

// CSVService is the CSV facet of markets, reached through [Service.AsCSV].
// [Service.Status] returns a [response.CSVResponse] carrying the API's raw
// CSV text — see ADR-018. [Service.StatusHistory] has no CSV facet.
type CSVService struct {
	http *http.Client
}

// AsCSV returns the CSV facet of this service.
func (s *Service) AsCSV() *CSVService {
	return &CSVService{http: s.http}
}

// Status fetches the market status for a single day as CSV. See
// [Service.Status] for parameter details.
func (s *CSVService) Status(ctx context.Context, opts ...StatusOption) (*response.CSVResponse, error) {
	path, params := statusPath(opts)
	return response.FetchFormatted(ctx, s.http, path, params, "csv", response.NewCSV)
}
