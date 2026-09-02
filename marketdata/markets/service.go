// Package markets provides market status information from the Market
// Data API's /v1/markets/status/ endpoint. It reports whether a market
// was open, closed, or closed early on a given trading day.
// [Service.Status] answers the question for a single date (today by
// default), while [Service.StatusHistory] returns a [MarketStatus] for
// each day in a date range.
//
// The two methods take separate, sealed option families so that a
// method-specific parameter cannot be passed to the wrong method: Status
// takes [StatusOption] values ([WithDate], [WithCountry]) and StatusHistory
// takes [HistoryOption] values ([WithHistoryWindow], [WithCountry]). The
// history range is a single [HistoryWindow] value built with [Between],
// [Since], [Until], [LastN], or [LastNUntil], so the API's mutually-exclusive
// range parameters can never be combined by mistake. [WithCountry] applies to
// both methods.
//
// See https://www.marketdata.app/docs/api/markets/status for the
// endpoint's API documentation.
package markets

import (
	"context"

	"github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/response"
)

// Service provides methods for accessing market status data.
type Service struct {
	http *http.Client
}

// NewService creates a new markets service.
func NewService(httpClient *http.Client) *Service {
	return &Service{
		http: httpClient,
	}
}

// Status fetches the market status for a single day from the
// /v1/markets/status/ endpoint. With no options it reports today's
// status for the United States; use [WithDate] to query a different
// day and [WithCountry] to query a different country by its two-letter
// ISO code. The date is sent to the API in YYYY-MM-DD form, so any
// time-of-day component is ignored. Status accepts only single-day
// [StatusOption] values; for a range of days, use [Service.StatusHistory]
// with [WithHistoryWindow].
//
// The returned [MarketStatus] carries the date (normalized to US
// Eastern), the raw status string, and an Open flag derived from it.
// When the API responds 404 because no status exists for the requested
// date, Status returns a nil MarketStatus and a nil error; the
// returned Response has its NoData field set to true.
//
// API documentation: https://www.marketdata.app/docs/api/markets/status
//
// Example:
//
//	status, _, err := client.Markets.Status(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if status != nil && status.Open {
//	    fmt.Println("Market is open!")
//	}
func (s *Service) Status(ctx context.Context, opts ...StatusOption) (*MarketStatus, *response.Response, error) {
	path, params := statusPath(opts)

	var resp statusResponse
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

	return resp.toMarketStatus(), response.New(httpResp), nil
}

// StatusHistory fetches the market status for a range of days from the
// /v1/markets/status/ endpoint, returning one [MarketStatus] per day.
// Select the range with [WithHistoryWindow] using a single [HistoryWindow]
// value, for example markets.Between(startDate, endDate) or markets.LastN(5)
// to request a fixed number of days counting back from the end of the range.
// [WithCountry] chooses the market by its two-letter ISO code, defaulting to
// the United States. Dates are sent to the API in YYYY-MM-DD form, so any
// time-of-day component is ignored. StatusHistory accepts only range
// [HistoryOption] values; for a single day, use [Service.Status] with
// [WithDate].
//
// When the API responds 404 because no data exists for the requested
// range, StatusHistory returns a nil slice and a nil error; the
// returned Response has its NoData field set to true.
//
// API documentation: https://www.marketdata.app/docs/api/markets/status
//
// Example:
//
//	statuses, _, err := client.Markets.StatusHistory(ctx,
//	    markets.WithHistoryWindow(markets.Between(startDate, endDate)),
//	)
func (s *Service) StatusHistory(ctx context.Context, opts ...HistoryOption) ([]MarketStatus, *response.Response, error) {
	path, params, err := statusHistoryPath(opts)
	if err != nil {
		return nil, nil, err
	}

	var resp statusResponse
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

	return resp.toMarketStatuses(), response.New(httpResp), nil
}

// --- Convenience methods (no context, no *Response) ---

// GetStatus is a convenience wrapper around [Service.Status] that uses
// context.Background() and discards the response metadata. It accepts
// the same functional options and has the same no-data behavior: when
// the API returns 404 for the requested date, GetStatus returns a nil
// [MarketStatus] and a nil error. Use Status directly when you need
// request cancellation, deadlines, or access to the Response.
//
// API documentation: https://www.marketdata.app/docs/api/markets/status
func (s *Service) GetStatus(opts ...StatusOption) (*MarketStatus, error) {
	st, _, err := s.Status(context.Background(), opts...)
	return st, err
}

// GetStatusHistory is a convenience wrapper for [Service.StatusHistory] that
// uses context.Background and discards the per-request [response.Response].
// Because the response metadata is discarded, a 404 no-data result is
// returned as a nil slice with a nil error. Use StatusHistory directly when
// you need request cancellation, deadlines, or access to the Response.
//
// API documentation: https://www.marketdata.app/docs/api/markets/status
func (s *Service) GetStatusHistory(opts ...HistoryOption) ([]MarketStatus, error) {
	h, _, err := s.StatusHistory(context.Background(), opts...)
	return h, err
}
