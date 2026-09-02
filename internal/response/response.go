// Package response provides the Response type for SDK results.
package response

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	internalhttp "github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/ratelimit"
)

// Response carries per-request metadata returned alongside typed data.
// It embeds *http.Response, giving callers access to raw headers, status, etc.
//
// Most callers can ignore it with a blank identifier:
//
//	quote, _, err := client.Stocks.Quote(ctx, "AAPL")
type Response struct {
	*http.Response

	// NoData is true when a 404 indicated no data was available (not an error).
	NoData bool

	// RateLimit contains per-request rate limit metadata (request-scoped).
	RateLimit RateLimitMeta

	// body is the cached response body (since http.Response.Body is consumed).
	body []byte
}

// RateLimitMeta contains per-response rate limit information.
type RateLimitMeta struct {
	Limit     int
	Remaining int
	Consumed  int
	ResetAt   time.Time
}

// IsJSON returns true if the response Content-Type is JSON.
func (r *Response) IsJSON() bool {
	return strings.Contains(r.Header.Get("Content-Type"), "application/json")
}

// IsCSV returns true if the response Content-Type is CSV.
func (r *Response) IsCSV() bool {
	return strings.Contains(r.Header.Get("Content-Type"), "text/csv")
}

// IsHTML returns true if the response Content-Type is HTML.
func (r *Response) IsHTML() bool {
	return strings.Contains(r.Header.Get("Content-Type"), "text/html")
}

// SaveToFile saves the response body to a file.
func (r *Response) SaveToFile(path string) error {
	if r.body == nil {
		return fmt.Errorf("no response body available")
	}
	return os.WriteFile(path, r.body, 0644)
}

// Body returns the raw response body exactly as the API sent it, for logging,
// debugging, or decoding a payload the typed models do not cover. The SDK reads
// http.Response.Body to completion when the request is made, so the embedded
// field is already drained; this is the only way to reach the bytes.
//
// The returned slice is a copy: mutating it cannot corrupt a later
// [Response.SaveToFile] or a second call. It is nil when no body was captured.
func (r *Response) Body() []byte {
	if r.body == nil {
		return nil
	}
	out := make([]byte, len(r.body))
	copy(out, r.body)
	return out
}

// String returns a one-line summary of the response — status, no-data flag, body
// size, and the remaining rate-limit credits — for logs and quick debugging. It
// never includes the body itself, which may be large and may carry data the
// caller would not expect in a log line.
func (r *Response) String() string {
	status := "<no http response>"
	if r.Response != nil {
		status = r.Status
	}
	return fmt.Sprintf("Response{Status: %s, NoData: %t, Body: %d bytes, Credits: %d/%d}",
		status, r.NoData, len(r.body), r.RateLimit.Remaining, r.RateLimit.Limit)
}

// New creates a Response from an internal HTTP response.
func New(httpResp *internalhttp.Response) *Response {
	return &Response{
		Response:  httpResp.Raw,
		RateLimit: parseRateLimitMeta(httpResp.Headers),
		body:      httpResp.Body,
	}
}

// IsNoData reports whether an HTTP status code means the request succeeded but
// returned no data: 404 (no data for the request) or 204 (a mode=cached cache
// miss). Both are surfaced to callers as a no-data Response, not an error.
func IsNoData(statusCode int) bool {
	return statusCode == 404 || statusCode == 204
}

// NewNoData creates a Response indicating no data was available (404 or 204).
func NewNoData(httpResp *internalhttp.Response) *Response {
	return &Response{
		Response:  httpResp.Raw,
		NoData:    true,
		RateLimit: parseRateLimitMeta(httpResp.Headers),
		body:      httpResp.Body,
	}
}

// CSVResponse carries a raw CSV response body (requested with format=csv;
// see ADR-018) alongside the same per-request metadata as Response. Unlike
// Response, it has no NoData field: the API's own no-data signal is not
// consistent between JSON and CSV (verified live — see ADR-018), so the raw
// body is handed to the caller exactly as the API sent it, whatever it is.
type CSVResponse struct {
	Response
}

// CSV returns the raw CSV response text, exactly as the API sent it.
func (r *CSVResponse) CSV() string {
	return string(r.body)
}

// NewCSV creates a CSVResponse from an internal HTTP response requested
// with format=csv.
func NewCSV(httpResp *internalhttp.Response) *CSVResponse {
	return &CSVResponse{Response: *New(httpResp)}
}

// HTMLResponse carries a raw HTML response body (requested with
// format=html; see ADR-018) alongside the same per-request metadata as
// Response. The facet that reaches this type is deliberately unexported
// today — the API does not serve HTML for any data endpoint yet (verified
// live: format=html 404s). The type exists so enabling it later is a
// small, low-risk change instead of new design.
type HTMLResponse struct {
	Response
}

// HTML returns the raw HTML response text, exactly as the API sent it.
func (r *HTMLResponse) HTML() string {
	return string(r.body)
}

// NewHTML creates an HTMLResponse from an internal HTTP response requested
// with format=html.
func NewHTML(httpResp *internalhttp.Response) *HTMLResponse {
	return &HTMLResponse{Response: *New(httpResp)}
}

func parseRateLimitMeta(headers http.Header) RateLimitMeta {
	if headers == nil {
		return RateLimitMeta{}
	}

	h := ratelimit.ParseHeaders(headers)
	return RateLimitMeta{
		Limit:     h.Limit,
		Remaining: h.Remaining,
		Consumed:  h.Consumed,
		ResetAt:   h.ResetAt,
	}
}
