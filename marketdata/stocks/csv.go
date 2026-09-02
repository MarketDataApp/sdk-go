package stocks

import (
	"context"

	"github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/response"
)

// CSVService is the CSV facet of stocks, reached through [Service.AsCSV].
// Every endpoint here returns a [response.CSVResponse] carrying the API's
// raw CSV text — see ADR-018 for why this exists alongside the typed JSON
// methods on [Service]. The SDK does not parse the CSV into rows; callers
// get the text exactly as the API sent it.
//
// The universal params that only cohere with CSV — [github.com/MarketDataApp/sdk-go/v2/marketdata.WithColumns],
// [github.com/MarketDataApp/sdk-go/v2/marketdata.WithAddHeaders], and
// [github.com/MarketDataApp/sdk-go/v2/marketdata.WithHumanReadable] — apply
// here exactly as documented; they have no effect on the typed JSON methods.
type CSVService struct {
	http *http.Client
}

// AsCSV returns the CSV facet of this service.
func (s *Service) AsCSV() *CSVService {
	return &CSVService{http: s.http}
}

// Quote fetches a real-time quote for a single stock symbol as CSV. See
// [Service.Quote] for parameter and validation details — this method
// mirrors it exactly, only the wire format and return type differ.
func (s *CSVService) Quote(ctx context.Context, symbol string, opts ...QuoteOption) (*response.CSVResponse, error) {
	path, params, err := quotePath(symbol, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormatted(ctx, s.http, path, params, "csv", response.NewCSV)
}

// Quotes fetches real-time quotes for multiple stock symbols as CSV. See
// [Service.Quotes] for parameter and validation details.
func (s *CSVService) Quotes(ctx context.Context, symbols []string, opts ...QuotesOption) (*response.CSVResponse, error) {
	path, params, err := quotesPath(symbols, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormatted(ctx, s.http, path, params, "csv", response.NewCSV)
}

// Prices fetches SmartMid midpoint prices for one or more stock symbols as
// CSV. See [Service.Prices] for parameter and validation details.
func (s *CSVService) Prices(ctx context.Context, symbols []string, opts ...PriceOption) (*response.CSVResponse, error) {
	path, params, err := pricesPath(symbols, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormatted(ctx, s.http, path, params, "csv", response.NewCSV)
}

// News fetches news articles for a stock symbol as CSV. See [Service.News]
// for parameter and validation details.
func (s *CSVService) News(ctx context.Context, symbol string, opts ...NewsOption) (*response.CSVResponse, error) {
	path, params, err := newsPath(symbol, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormatted(ctx, s.http, path, params, "csv", response.NewCSV)
}

// Earnings fetches earnings reports for a stock symbol as CSV. See
// [Service.Earnings] for parameter and validation details.
func (s *CSVService) Earnings(ctx context.Context, symbol string, opts ...EarningsOption) (*response.CSVResponse, error) {
	path, params, err := earningsPath(symbol, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormatted(ctx, s.http, path, params, "csv", response.NewCSV)
}

// Candles fetches historical OHLCV candles for a stock symbol as CSV. See
// [Service.Candles] for parameter, validation, and auto-chunking details —
// this method applies the identical chunking decision and fetches chunks
// concurrently the same way, merging the CSV text in chronological chunk
// order (dropping the repeated header row from every chunk after the
// first) instead of merging typed candles.
func (s *CSVService) Candles(ctx context.Context, symbol string, opts ...CandleOption) (*response.CSVResponse, error) {
	path, chunkParams, err := candlesPath(symbol, opts)
	if err != nil {
		return nil, err
	}
	return response.FetchFormattedChunked(ctx, s.http, path, chunkParams, "csv", response.NewCSV)
}
