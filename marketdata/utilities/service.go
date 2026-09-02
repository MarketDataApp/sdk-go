// Package utilities provides access to the Market Data API's utility
// endpoints, which report on the API itself rather than on market
// data. [Service.Status] returns an [APIStatus] with the API's current
// availability and historical uptime statistics, [Service.Headers]
// echoes back the request headers your application sends (useful for
// debugging authentication problems), and [Service.User] returns a
// [UserInfo] describing the authenticated account's API credit state
// and options data entitlement.
//
// All three endpoints are served unversioned: /status/, /headers/, and
// /user/. See https://www.marketdata.app/docs/api/utilities/status for
// the API documentation.
package utilities

import (
	"context"

	"github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/response"
)

// Service provides methods for accessing API utilities.
type Service struct {
	http *http.Client
}

// NewService creates a new utilities service.
func NewService(httpClient *http.Client) *Service {
	return &Service{
		http: httpClient,
	}
}

// Status fetches the current status of the Market Data API from the
// unversioned /status/ endpoint. It takes no parameters and no
// functional options.
//
// The API reports per-service status; the returned [APIStatus]
// aggregates it, with Status set to "online" only when every service
// is online, and Uptime30d and Uptime90d holding the average 30-day
// and 90-day uptime percentages across services. The status is updated
// every 5 minutes and remains accessible even when the API is
// experiencing issues. In the unlikely event the endpoint responds
// 404, Status returns a nil APIStatus and a nil error, and the
// returned Response has its NoData field set to true.
//
// API documentation: https://www.marketdata.app/docs/api/utilities/status
//
// Example:
//
//	status, _, err := client.Utilities.Status(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if status != nil && status.IsOnline() {
//	    fmt.Printf("API is online! 30-day uptime: %.2f%%\n", status.Uptime30d)
//	}
func (s *Service) Status(ctx context.Context) (*APIStatus, *response.Response, error) {
	var resp apiStatusResponse
	// Status endpoint is not versioned (no /v1/ prefix)
	httpResp, err := s.http.GetUnversioned(ctx, "status/", nil, &resp)
	if err != nil {
		return nil, nil, err
	}

	// Handle 404 as no-data (not an error)
	if response.IsNoData(httpResp.StatusCode) {
		return nil, response.NewNoData(httpResp), nil
	}

	// The same 200-but-not-ok guard every other decoding method applies.
	// This one was missing, and it is the endpoint where it matters most:
	// without it a body of {"s":"error"} decoded to an APIStatus reporting
	// IsOnline() == true with a nil error, so the one method whose purpose
	// is monitoring told callers the API was up while the API said it was
	// not.
	if resp.ResponseStatus != "ok" {
		return nil, nil, httpResp.StatusError(resp.ResponseStatus)
	}

	data := resp.toAPIStatus()
	return data, response.New(httpResp), nil
}

// Headers fetches the request headers that your application is sending
// to the API, echoed back by the unversioned /headers/ endpoint. It
// takes no parameters and no functional options.
//
// The returned [Headers] maps header names to the values the server
// received, which is useful for debugging authentication and proxy
// issues. Sensitive header values (like Authorization) are partially
// redacted by the server for security. In the unlikely event the
// endpoint responds 404, Headers returns an empty result (the map is
// safe to range) and a nil error, and the returned Response has its
// NoData field set to true.
//
// API documentation: https://www.marketdata.app/docs/api/utilities/headers
//
// Example:
//
//	headers, _, err := client.Utilities.Headers(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if headers != nil {
//	    for name, value := range headers.Headers {
//	        fmt.Printf("%s: %s\n", name, value)
//	    }
//	}
func (s *Service) Headers(ctx context.Context) (*Headers, *response.Response, error) {
	var resp headersResponse
	// Headers endpoint is not versioned (no /v1/ prefix)
	httpResp, err := s.http.GetUnversioned(ctx, "headers/", nil, &resp)
	if err != nil {
		return nil, nil, err
	}

	// Empty answer, not a nil one — same reasoning as options.Service.Chain:
	// an empty map ranges zero times, a nil *Headers panics.
	if response.IsNoData(httpResp.StatusCode) {
		return resp.toHeaders(), response.NewNoData(httpResp), nil
	}

	data := resp.toHeaders()
	return data, response.New(httpResp), nil
}

// User fetches information about the authenticated user and their
// account from the /user/ endpoint (unversioned, like /status/ and
// /headers/). It takes no parameters and no functional options, but it
// requires a valid API token.
//
// The returned [UserInfo] carries the account's API credit state for
// the current window (limit, remaining, consumed, and reset time, from
// the x-api-ratelimit-* response headers) and the account's options
// data entitlement. In the unlikely event the endpoint responds 404,
// User returns a nil result and a nil error, and the returned Response
// has its NoData field set to true.
//
// API documentation: https://www.marketdata.app/docs/api/utilities/user
//
// Example:
//
//	user, _, err := client.Utilities.User(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if user != nil {
//	    fmt.Printf("Credits: %d/%d\n", user.CreditsRemaining, user.CreditLimit)
//	}
func (s *Service) User(ctx context.Context) (*UserInfo, *response.Response, error) {
	var resp userResponse
	httpResp, err := s.http.GetUnversioned(ctx, "user/", nil, &resp)
	if err != nil {
		return nil, nil, err
	}

	// Handle 404 as no-data (not an error)
	if response.IsNoData(httpResp.StatusCode) {
		return nil, response.NewNoData(httpResp), nil
	}

	data := resp.toUserInfo(httpResp.Headers)
	return data, response.New(httpResp), nil
}

// --- Convenience methods (no context, no *Response) ---

// GetStatus is a convenience wrapper around [Service.Status] that uses
// context.Background() and discards the response metadata. Use Status
// directly when you need request cancellation, deadlines, or access to
// the Response.
//
// API documentation: https://www.marketdata.app/docs/api/utilities/status
func (s *Service) GetStatus() (*APIStatus, error) {
	st, _, err := s.Status(context.Background())
	return st, err
}

// GetHeaders is a convenience wrapper around [Service.Headers] that
// uses context.Background() and discards the response metadata. Use
// Headers directly when you need request cancellation, deadlines, or
// access to the Response.
//
// API documentation: https://www.marketdata.app/docs/api/utilities/headers
func (s *Service) GetHeaders() (*Headers, error) {
	h, _, err := s.Headers(context.Background())
	return h, err
}

// GetUser is a convenience wrapper around [Service.User] that uses
// context.Background() and discards the response metadata. Like User,
// it requires a valid API token. Use User directly when you need
// request cancellation, deadlines, or access to the Response.
//
// API documentation: https://www.marketdata.app/docs/api/utilities/user
func (s *Service) GetUser() (*UserInfo, error) {
	u, _, err := s.User(context.Background())
	return u, err
}
