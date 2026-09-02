package funds

import (
	"context"

	"github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/response"
	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

// Service provides methods for accessing fund data.
type Service struct {
	http *http.Client
}

// NewService creates a new funds service.
func NewService(httpClient *http.Client) *Service {
	return &Service{
		http: httpClient,
	}
}

// Candles fetches historical OHLC candles for a mutual fund from the
// /v1/funds/candles/{resolution}/{symbol}/ endpoint. Each [Candle]
// reports the fund's net asset value (NAV) for one period, with its
// Time normalized to US Eastern.
//
// The symbol parameter is the fund's ticker (for example "VFINX") and
// is required; an empty symbol returns a validation error without
// making a request. The candle timeframe defaults to [ResolutionDaily]
// and can be changed with [WithResolution]. The date range is selected
// with [WithCandleWindow] using a single [DateWindow] value, so the
// mutually-exclusive date parameters cannot be combined by mistake. Dates
// are sent to the API in YYYY-MM-DD form, so any time-of-day component is
// ignored.
//
// When the API responds 404 because no data exists for the requested
// symbol or range, Candles returns a nil slice and a nil error; the
// returned Response has its NoData field set to true. Other failures
// are reported through the error return.
//
// API documentation: https://www.marketdata.app/docs/api/funds/candles
//
// Example:
//
//	candles, _, err := client.Funds.Candles(ctx, "VFINX",
//	    funds.WithResolution(funds.ResolutionDaily),
//	    funds.WithCandleWindow(funds.Since(time.Now().AddDate(0, -1, 0))),
//	)
func (s *Service) Candles(ctx context.Context, symbol string, opts ...CandleOption) ([]Candle, *response.Response, error) {
	if symbol == "" {
		return nil, nil, &sdkerrors.ValidationError{Field: "symbol", Message: "symbol is required"}
	}

	path, params, err := candlesPath(symbol, opts)
	if err != nil {
		return nil, nil, err
	}

	var resp candlesResponse
	httpResp, err := s.http.Get(ctx, path, params, &resp)
	if err != nil {
		return nil, nil, err
	}

	// Handle 404 as no-data (not an error)
	if response.IsNoData(httpResp.StatusCode) {
		return nil, response.NewNoData(httpResp), nil
	}

	if resp.Status != "ok" {
		return nil, nil, httpResp.StatusError(resp.Status)
	}

	return resp.toCandles(), response.New(httpResp), nil
}

// --- Convenience methods (no context, no *Response) ---

// GetCandles is a convenience wrapper around [Service.Candles] that uses
// context.Background() and discards the response metadata. It accepts the
// same symbol and functional options and has the same no-data behavior:
// when the API returns 404 for the requested symbol or range, GetCandles
// returns a nil slice and a nil error. Use Candles directly when you need
// request cancellation, deadlines, or access to the Response.
//
// API documentation: https://www.marketdata.app/docs/api/funds/candles
func (s *Service) GetCandles(symbol string, opts ...CandleOption) ([]Candle, error) {
	c, _, err := s.Candles(context.Background(), symbol, opts...)
	return c, err
}
