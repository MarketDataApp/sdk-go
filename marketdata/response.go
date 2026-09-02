package marketdata

import (
	"github.com/MarketDataApp/sdk-go/v2/internal/response"
)

// Response carries per-request metadata returned alongside typed data.
// It embeds *http.Response for raw access to headers, status, etc.
type Response = response.Response

// RateLimitMeta contains per-response rate limit information.
// This is request-scoped and deterministic, unlike client-level rate limits.
type RateLimitMeta = response.RateLimitMeta

// CSVResponse carries a raw CSV response body — see the AsCSV() facet on
// each service (e.g. [github.com/MarketDataApp/sdk-go/v2/marketdata/stocks.Service.AsCSV])
// and ADR-018 for the design rationale.
type CSVResponse = response.CSVResponse

// HTMLResponse carries a raw HTML response body. Reserved: no service
// exposes a facet returning this type yet, since the API does not serve
// HTML for data endpoints today. See ADR-018.
type HTMLResponse = response.HTMLResponse
