package utilities

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/ratelimit"
	"github.com/MarketDataApp/sdk-go/v2/internal/timezone"
)

// APIStatus represents the current status of the MarketData API.
type APIStatus struct {
	// Status indicates if the API is online
	Status string `json:"s"`

	// Uptime30d is the 30-day uptime percentage
	Uptime30d float64 `json:"uptime30d"`

	// Uptime90d is the 90-day uptime percentage
	Uptime90d float64 `json:"uptime90d"`

	// Updated is when the status was last checked
	Updated time.Time `json:"updated"`
}

// String returns a summary of the API status.
func (s APIStatus) String() string {
	return fmt.Sprintf("API %s (30d: %.2f%% 90d: %.2f%%) Updated: %s", s.Status, s.Uptime30d, s.Uptime90d, s.Updated.Format("2006-01-02 15:04:05"))
}

// IsOnline returns true if the API is currently online.
func (s *APIStatus) IsOnline() bool {
	return s.Status == "online"
}

// apiStatusResponse is the API response for status.
// The API returns arrays of status info for each service.
type apiStatusResponse struct {
	ResponseStatus string    `json:"s"`
	Service        []string  `json:"service"`
	Status         []string  `json:"status"`
	Online         []bool    `json:"online"`
	UptimePct30d   []float64 `json:"uptimePct30d"`
	UptimePct90d   []float64 `json:"uptimePct90d"`
	Updated        []int64   `json:"updated"`
}

// RequiredColumns names the online array toAPIStatus derives its verdict
// from. An empty one reports the API offline — deliberately, since absent
// evidence is not evidence of health — so filtering it out would make a
// WithColumns client believe the API is down and abort its retries. See
// http.ColumnRequirer.
func (r *apiStatusResponse) RequiredColumns() []string { return []string{"online"} }

// toAPIStatus converts the API response to an APIStatus.
// Uses the aggregate status across all services.
func (r *apiStatusResponse) toAPIStatus() *APIStatus {
	if r == nil {
		return nil
	}

	// "All online" over an empty list is vacuously true, which is the wrong
	// default for a health check: a body carrying no per-service array at
	// all would report the API as up. Absent evidence is not evidence of
	// health, so an empty list reports offline.
	allOnline := len(r.Online) > 0
	for _, online := range r.Online {
		if !online {
			allOnline = false
			break
		}
	}

	// Calculate average uptimes
	var uptime30d, uptime90d float64
	if len(r.UptimePct30d) > 0 {
		for _, u := range r.UptimePct30d {
			uptime30d += u
		}
		uptime30d = (uptime30d / float64(len(r.UptimePct30d))) * 100 // Convert to percentage
	}
	if len(r.UptimePct90d) > 0 {
		for _, u := range r.UptimePct90d {
			uptime90d += u
		}
		uptime90d = (uptime90d / float64(len(r.UptimePct90d))) * 100 // Convert to percentage
	}

	// Use the latest updated timestamp
	var updated time.Time
	if len(r.Updated) > 0 {
		updated = timezone.ToEastern(r.Updated[0])
	}

	status := "offline"
	if allOnline {
		status = "online"
	}

	return &APIStatus{
		Status:    status,
		Uptime30d: uptime30d,
		Uptime90d: uptime90d,
		Updated:   updated,
	}
}

// Headers represents the headers sent by the client.
// This is useful for debugging authentication issues.
type Headers struct {
	// Headers is a map of header names to values
	Headers map[string]string `json:"headers"`
}

// String returns a concise summary of the headers.
func (h Headers) String() string {
	parts := make([]string, 0, len(h.Headers))
	for k, v := range h.Headers {
		parts = append(parts, k+": "+v)
	}
	return fmt.Sprintf("Headers{%s}", strings.Join(parts, ", "))
}

// headersResponse is the wire response for /headers/. Unlike every other
// endpoint, the API echoes the request's own headers as a flat JSON object
// — each header name is a top-level key, e.g. {"accept":"*/*","host":"..."}
// — rather than wrapping them in an envelope field (verified live
// 2026-08-05). A prior {"headers": {...}} wrapper assumption never matched
// the real response, so Headers() always decoded to an empty map.
type headersResponse map[string]string

// RequiredColumns is nil: /headers/ echoes request headers, so there is no
// envelope and no row-count column to protect. See http.ColumnRequirer.
func (r *headersResponse) RequiredColumns() []string { return nil }

// toHeaders converts the API response to Headers.
func (r *headersResponse) toHeaders() *Headers {
	if r == nil {
		return nil
	}
	return &Headers{
		Headers: map[string]string(*r),
	}
}

// UserInfo describes the authenticated account: the API credit state
// for the current window, taken from the x-api-ratelimit-* response
// headers, and the account's options data entitlement, taken from the
// response body.
type UserInfo struct {
	// CreditLimit is the account's total API credit allowance for the
	// current window.
	CreditLimit int

	// CreditsRemaining is the number of API credits left in the
	// current window.
	CreditsRemaining int

	// CreditsConsumed is the number of API credits consumed by the
	// user info request itself.
	CreditsConsumed int

	// ResetAt is when the current credit window resets.
	ResetAt time.Time

	// OptionsDataPermissions describes the account's options data
	// entitlement, e.g. "OPRA data delayed 15 minutes".
	OptionsDataPermissions string
}

// String returns a summary of the user info.
func (u UserInfo) String() string {
	return fmt.Sprintf("User{Credits: %d/%d, Resets: %s, Options: %s}", u.CreditsRemaining, u.CreditLimit, u.ResetAt.Format("2006-01-02 15:04:05"), u.OptionsDataPermissions)
}

// userResponse is the API response body for the /user/ endpoint. The
// request counts duplicate the x-api-ratelimit-* headers, which are the
// authoritative source for credit state.
type userResponse struct {
	RequestsRemaining      int    `json:"x-ratelimit-requests-remaining"`
	RequestsLimit          int    `json:"x-ratelimit-requests-limit"`
	OptionsDataPermissions string `json:"x-options-data-permissions"`
}

// RequiredColumns is nil: /user/ returns scalar fields with no envelope and
// no row-count column. See http.ColumnRequirer.
func (r *userResponse) RequiredColumns() []string { return nil }

// toUserInfo builds a UserInfo from the response body and the
// x-api-ratelimit-* response headers. The body's request counts serve
// as a fallback when the headers are absent.
func (r *userResponse) toUserInfo(headers http.Header) *UserInfo {
	if r == nil {
		return nil
	}

	info := &UserInfo{
		CreditLimit:            r.RequestsLimit,
		CreditsRemaining:       r.RequestsRemaining,
		OptionsDataPermissions: r.OptionsDataPermissions,
	}

	// One shared parser for the four rate-limit headers, so this surface
	// cannot disagree with Response.RateLimit and Client.RateLimits() about
	// the same response (it used to: a reset of 0 read as 1969-12-31 here
	// and as the zero time there).
	rl := ratelimit.ParseHeaders(headers)
	if rl.HasLimit {
		info.CreditLimit = rl.Limit
	}
	if rl.HasRemaining {
		info.CreditsRemaining = rl.Remaining
	}
	if rl.HasConsumed {
		info.CreditsConsumed = rl.Consumed
	}
	if rl.HasReset {
		info.ResetAt = rl.ResetAt
	}

	// Entitlements come from the header for the same reason the credit
	// fields above do: the body's own key is present but always empty in
	// production (verified live 2026-08-19), while the header carries the
	// real value — e.g. "delayed_quotes_permission,historical_quotes_
	// permission,real_time_quotes_permission". Reading only the body meant
	// an account with full real-time entitlements reported none, so any
	// gating built on this field took the wrong branch for every user
	// — the same class of failure as the /headers/ wrapper assumption on
	// this endpoint family: trusting the body's shape without checking it
	// against the live response.
	if perms := headers.Get("X-Options-Data-Permissions"); perms != "" {
		info.OptionsDataPermissions = perms
	}

	return info
}
