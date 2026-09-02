// Package http provides an HTTP client wrapper for the MarketData SDK.
package http

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/MarketDataApp/sdk-go/v2/internal/fanout"
	"github.com/MarketDataApp/sdk-go/v2/internal/ratelimit"
	"github.com/MarketDataApp/sdk-go/v2/internal/retry"
	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
	"github.com/MarketDataApp/sdk-go/v2/internal/status"
	"github.com/MarketDataApp/sdk-go/v2/internal/timezone"
)

const modulePath = "github.com/MarketDataApp/sdk-go/v2"

// defaultMaxResponseBytes caps how much of a response body the SDK will read
// into memory. It bounds memory use against a hostile or malfunctioning server
// that streams an unbounded body. It is generous enough for the largest
// legitimate responses (full option chains, multi-year bulk candles).
const defaultMaxResponseBytes int64 = 100 << 20 // 100 MiB

// maxRedirects caps how many redirects the SDK will follow before giving up.
const maxRedirects = 10

// DefaultPoolSize is the SDK's global concurrency pool size (ADR-014):
// NewClient's semaphore channel is sized to this, and this same constant
// backs the pool-saturation test, so a change to one is a change to both —
// a test asserting the wrong number silently (see T-5 in the deep review)
// can no longer happen.
const DefaultPoolSize = 50

var (
	versionOnce sync.Once
	versionStr  string
)

// Version returns the SDK version detected from Go module metadata.
// Falls back to "unknown" if build info is unavailable.
func Version() string {
	versionOnce.Do(func() {
		versionStr = detectVersion()
	})
	return versionStr
}

func detectVersion() string {
	info, ok := debug.ReadBuildInfo()
	return versionFromBuildInfo(info, ok)
}

// versionFromBuildInfo resolves the SDK version from build metadata: the
// module's version when the SDK is a dependency, the main module's stamped
// version when the SDK builds directly, "unknown" otherwise. Split from
// detectVersion so every resolution branch is testable — a test binary's own
// build info can only ever produce the fallback.
func versionFromBuildInfo(info *debug.BuildInfo, ok bool) string {
	if !ok {
		return "unknown"
	}
	for _, dep := range info.Deps {
		if dep.Path == modulePath {
			return dep.Version
		}
	}
	if info.Main.Path == modulePath && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "unknown"
}

// Client wraps http.Client with SDK-specific functionality.
type Client struct {
	http          *http.Client
	ownsTransport bool // false when http.Transport came from an injected *http.Client (WithHTTPClient)
	baseURL       string
	apiVersion    string
	token         string
	demoMode      bool
	retryCfg      retry.Config
	rateLimits    *ratelimit.Tracker
	statusCache   *status.Cache
	logger        *slog.Logger
	debug         atomic.Bool // runtime-toggleable via SetDebug
	sem           chan struct{}
	defaultParams url.Values // universal default params from env/client config

	// formatOnlyParams are merged by GetFormatted alone: they are universal
	// parameters that only cohere with a raw response body and would break
	// the typed decoders. See marketdata.WithHumanReadable.
	formatOnlyParams url.Values
	maxRespBytes     int64 // cap on response body size read into memory
}

// Config holds the HTTP client configuration.
type Config struct {
	HTTPClient       *http.Client
	BaseURL          string
	APIVersion       string
	Token            string
	Timeout          time.Duration
	ConnTimeout      time.Duration
	RetryCfg         retry.Config
	RateLimits       *ratelimit.Tracker
	Logger           *slog.Logger
	Debug            bool
	DemoMode         bool
	Sem              chan struct{}
	DefaultParams    url.Values
	FormatOnlyParams url.Values    // universal default params (from env vars / client config)
	StatusCache      *status.Cache // API status cache for retry decisions
	MaxRespBytes     int64         // cap on response body size (0 = default)
}

// New creates a new HTTP client with the given configuration.
func New(cfg Config) *Client {
	var httpClient *http.Client
	if cfg.HTTPClient != nil {
		// Work on a shallow copy of the caller's client: the SDK needs its
		// own Timeout and redirect policy, and writing them onto the
		// caller's object would both surprise the caller and race with any
		// of their in-flight requests. The Transport (connection pool,
		// proxy, TLS setup) is intentionally shared — that is what
		// injecting a client is for.
		clone := *cfg.HTTPClient
		httpClient = &clone
	}
	if httpClient == nil {
		transport := &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   cfg.ConnTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:   true,
			TLSHandshakeTimeout: 10 * time.Second,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		}
		httpClient = &http.Client{
			Transport: transport,
		}
	}
	httpClient.Timeout = cfg.Timeout
	// Enforce a secure redirect policy regardless of whether the client was
	// supplied by the caller: cap the redirect chain and refuse any redirect
	// that would send the request (and its Authorization header) to a
	// different host, so the token cannot be redirected to an unintended
	// origin. (Go's stdlib already strips Authorization cross-host; this makes
	// the guarantee explicit and testable.)
	httpClient.CheckRedirect = secureCheckRedirect

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	maxResp := cfg.MaxRespBytes
	if maxResp <= 0 {
		maxResp = defaultMaxResponseBytes
	}

	c := &Client{
		http:             httpClient,
		ownsTransport:    cfg.HTTPClient == nil,
		baseURL:          strings.TrimSuffix(cfg.BaseURL, "/"),
		apiVersion:       cfg.APIVersion,
		token:            cfg.Token,
		demoMode:         cfg.DemoMode,
		retryCfg:         cfg.RetryCfg,
		rateLimits:       cfg.RateLimits,
		statusCache:      cfg.StatusCache,
		logger:           logger,
		sem:              cfg.Sem,
		defaultParams:    cfg.DefaultParams,
		formatOnlyParams: cfg.FormatOnlyParams,
		maxRespBytes:     maxResp,
	}
	c.debug.Store(cfg.Debug)
	return c
}

// secureCheckRedirect caps the redirect chain and refuses cross-host redirects
// so the API token is never followed to an unintended origin.
func secureCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}
	if len(via) > 0 && req.URL.Host != via[0].URL.Host {
		return fmt.Errorf("refusing cross-host redirect from %q to %q", via[0].URL.Host, req.URL.Host)
	}
	return nil
}

// tokenSafeForURL reports whether the API token may be transmitted to u without
// exposing it in cleartext. HTTPS is always safe; plain HTTP is allowed only to
// loopback hosts, which never leave the machine (local development).
func tokenSafeForURL(u *url.URL) bool {
	if u == nil {
		return false
	}
	if strings.EqualFold(u.Scheme, "https") {
		return true
	}
	return isLoopbackHost(u.Hostname())
}

// isLoopbackHost reports whether host is localhost or a loopback IP literal.
func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

// Request represents an API request.
type Request struct {
	Method      string
	Path        string
	Params      url.Values
	Headers     map[string]string
	Unversioned bool // If true, don't prefix with API version

	// RawFormat marks a request whose body is handed back verbatim (the
	// CSV/HTML facets, see ADR-018) rather than decoded. It exists so
	// decoder-serving adjustments — the "columns" repair below — apply only
	// where something actually decodes.
	RawFormat bool

	// RequiredColumns names the response columns this request's decoder
	// cannot do without, on top of the "s" envelope. Get fills it from the
	// destination value; see ColumnRequirer.
	RequiredColumns []string
}

// ColumnRequirer is implemented by a wire-response type whose decode needs a
// column the caller's "columns" filter may have dropped.
//
// The rule is narrow on purpose. WithColumns promises that "missing columns
// simply decode as zero values", and for a data field that is exactly what
// happens — a filtered-out bid reads as 0. One column per response type is
// different: the array the conversion takes its row count from. Drop that
// one and every row disappears, so a present, billed quote is reported as
// not found. That is not a zero value, it is a wrong answer, and it is the
// same failure mode as the dropped "s" envelope — a field the decoder needs
// and the caller never thought to ask for.
//
// Implementations return nil when they have no such column. Every wire
// response type implements it, which TestEveryWireResponseDeclaresRequiredColumns
// enforces as new ones are added.
type ColumnRequirer interface {
	RequiredColumns() []string
}

// envelopeColumn is the status field every typed method checks. The API
// drops it whenever a column filter is set.
const envelopeColumn = "s"

// requiredColumns reads a destination value's column needs. A destination
// that does not implement ColumnRequirer — a bare map in a test, say — needs
// nothing beyond the envelope.
func requiredColumns(result any) []string {
	if cr, ok := result.(ColumnRequirer); ok {
		return cr.RequiredColumns()
	}
	return nil
}

// missingColumns returns the columns the decoder needs and filter does not
// already name, envelope first, in a stable order and without repeats.
func missingColumns(filter, required []string) []string {
	var missing []string
	for _, col := range append([]string{envelopeColumn}, required...) {
		if namesColumn(filter, col) || slices.Contains(missing, col) {
			continue
		}
		missing = append(missing, col)
	}
	return missing
}

// Response represents an API response.
type Response struct {
	Raw        *http.Response // the raw HTTP response (body already consumed)
	StatusCode int
	Headers    http.Header
	Body       []byte
	RequestID  string
}

// SupportContext builds a [sdkerrors.SupportContext] for an error tied to
// this response, filling RequestID, RequestURL, StatusCode, and Timestamp
// from the response itself so callers only need to supply the message and
// exception type. Used for errors raised outside the >=400 status mapping
// in parseAPIError — e.g. an HTTP 200 whose body reports its own failure.
//
// RequestURL goes through wireURL like the >=400 and ParseError blocks do.
// It used to report Raw.Request.URL.Path, so all fifteen StatusError sites
// produced a support block stripped of the query — and Quote is served by
// bulkquotes, which addresses by query, so the block for a failed
// Quote(ctx, "AAPL") did not even say which symbol was asked for. A block
// exists to be pasted into a ticket, and this is the error class where the
// merged universal parameters matter most: a 200 whose body reports its own
// failure is the one most likely to have been caused by them.
func (r *Response) SupportContext(message, exceptionType string) sdkerrors.SupportContext {
	return sdkerrors.SupportContext{
		RequestID:     r.RequestID,
		RequestURL:    r.wireURL(""),
		StatusCode:    r.StatusCode,
		Timestamp:     time.Now().In(timezone.Eastern),
		Message:       message,
		ExceptionType: exceptionType,
	}
}

// wireURL returns the URL the request was actually sent to, falling back to
// the supplied value when the raw request is unavailable (a hand-built
// Response in a test).
//
// The error paths used to rebuild this from the caller's own params, which
// Do never writes to — it merges the client's universal defaults into a
// copy. So every support block omitted exactly the parameters most likely
// to explain the failure: a client configured with WithColumns, WithMode or
// WithLimit reported a URL that had never been sent. The support context
// exists to be pasted into a ticket, so it has to be the real request.
func (r *Response) wireURL(fallback string) string {
	if r != nil && r.Raw != nil && r.Raw.Request != nil && r.Raw.Request.URL != nil {
		return r.Raw.Request.URL.String()
	}
	return fallback
}

// StatusError builds the [sdkerrors.APIError] every service raises when a
// 200 response carries a body whose own "s" field is not "ok" — the one
// failure mode the >=400 mapping in parseAPIError cannot see. Every such
// guard is identical apart from the status string, so they share this
// method rather than repeating the same SupportContext call at each of the
// service call sites. A guard that genuinely needs a different exception
// type still calls [Response.SupportContext] directly.
func (r *Response) StatusError(status string) error {
	return &sdkerrors.APIError{
		SupportContext: r.SupportContext("unexpected response status: "+status, "APIError"),
	}
}

// logTerminalFailure logs err at ERROR level (WARN for a pre-flight
// rate-limit rejection, see below). Called only from Get and GetUnversioned
// — the two exported methods every service call ultimately goes through —
// so it is the single choke point for every terminal request failure
// (requirements §7): a >=400 API response, a ParseError from a malformed
// body, a transport/network failure, a context cancellation while waiting
// for a pool slot or during backoff, a pre-flight rate-limit rejection,
// retries exhausted, or the offline-status abort. Do() itself does NOT log:
// several of those error kinds (>=400, ParseError) are only synthesized by
// Get/GetUnversioned AFTER Do returns a response successfully, so Do is not
// actually the outermost boundary — logging there would miss those cases
// entirely (as it did prior to this fix) while also risking a double log
// for the ones Do does originate, once Get/GetUnversioned's own wrapper
// logs them too.
//
// Every typed SDK error (requirements §6.1's taxonomy, APIError included)
// implements sdkerrors.Error and carries a
// SupportContext, whose SupportInfo() already formats request_id,
// request_url, status_code, and exception_type — logging it covers the
// fields §7 asks for in one line without hand-extracting them per error
// type. Errors synthesized locally without a SupportContext
// (retries-exhausted, the offline-status abort) fall back to their plain
// message.
//
// Two kinds of failure are demoted below ERROR because they are not
// server-reported failures:
//
// A context.Canceled failure is a request the caller — or the SDK itself —
// deliberately abandoned. Every cancel-on-first-error fan-out (options
// Quotes, stocks candlesSplit, response.FetchFormattedChunked and
// FetchFormattedMap) cancels its siblings the moment one request fails, and
// each cancelled sibling surfaces here as a NetworkError wrapping
// context.Canceled before the fan-out's own selection loop discards it as
// an echo. Logging those at ERROR means one 401 in a 50-symbol batch
// produces one legitimate ERROR plus ~49 lines of noise on stderr, which
// the default logger emits with zero configuration. They log at DEBUG
// instead.
//
// context.DeadlineExceeded is demoted only inside a fan-out, where the same
// arithmetic applies for a different reason: the caller's own
// context.WithTimeout is ONE expiry that every sibling reports, so a
// 50-symbol batch produced 50 ERROR lines for it (measured). fanout.IsChild
// identifies those requests, and the fan-out surfaces a single error to its
// caller regardless. Outside a fan-out a deadline is a genuine failure the
// caller did not ask for and keeps its ERROR line; NetworkError.Timeout
// distinguishes it either way.
//
// The demotion is deliberately limited to context failures. A 401 or a 500
// on one sibling is a distinct answer from the server, not an echo, and
// still logs at ERROR even inside a fan-out.
//
// A pre-flight RateLimitError (RateLimitError.PreFlight) is the rate
// limiter working as designed — the SDK caught it locally and never spent a
// real request — not a server-reported failure, so it logs at WARN instead
// of ERROR to avoid alarm-fatiguing operators under normal throttling.
func (c *Client) logTerminalFailure(ctx context.Context, err error) {
	if errors.Is(err, context.Canceled) {
		c.logDebug("request cancelled", "error", err.Error())
		return
	}
	if errors.Is(err, context.DeadlineExceeded) && fanout.IsChild(ctx) {
		c.logDebug("request deadline exceeded inside a fan-out", "error", err.Error())
		return
	}
	var rlErr *sdkerrors.RateLimitError
	if errors.As(err, &rlErr) && rlErr.PreFlight {
		c.logger.Warn("request rejected", "error", err.Error(), "support_info", rlErr.SupportInfo())
		return
	}
	var sdkErr sdkerrors.Error
	if errors.As(err, &sdkErr) {
		c.logger.Error("request failed", "error", err.Error(), "support_info", sdkErr.SupportInfo())
		return
	}
	c.logger.Error("request failed", "error", err.Error())
}

// Do executes the request with retry logic.
func (c *Client) Do(ctx context.Context, req Request) (*Response, error) {
	// Merge default params as fallbacks (method-level params take priority).
	// Build a fresh url.Values instead of writing into req.Params: the map
	// belongs to the caller and must not be mutated, and copying wholesale
	// also preserves multi-value defaults (the old Set-in-a-loop kept only
	// the last value).
	if len(c.defaultParams) > 0 {
		merged := make(url.Values, len(req.Params)+len(c.defaultParams))
		for key, vals := range req.Params {
			merged[key] = vals
		}
		for key, vals := range c.defaultParams {
			if _, ok := merged[key]; !ok {
				merged[key] = append([]string(nil), vals...)
			}
		}
		req.Params = merged
	}

	// A "columns" filter makes the API return only the named columns, and two
	// kinds of field the decoder needs are not ones a caller thinks to name:
	// the "s" envelope, which every typed method checks, and the array each
	// wire type takes its row count from (see ColumnRequirer). Without the
	// first, every JSON call from a client configured with WithColumns failed
	// with "unexpected response status: " (an empty status); without the
	// second it decoded to zero rows and reported not-found for data that was
	// there. Both contradict WithColumns' promise that "filtering never
	// causes a decode error". Asking for them back restores the decode
	// without changing which data columns the caller gets.
	//
	// Not applied to the formatted facets: there the column list IS the
	// output, and adding a column the caller did not ask for would corrupt
	// it. Same rule as dateformat on options expirations — a parameter that
	// serves the decoder belongs only on the decoding path.
	if !req.RawFormat {
		if cols := req.Params["columns"]; len(cols) > 0 {
			if missing := missingColumns(cols, req.RequiredColumns); len(missing) > 0 {
				req.Params = cloneValues(req.Params)
				// One joined value, never a repeated key. Prepending each
				// name as its own value looked safer — it kept a multi-value
				// default intact where rewriting only the first value would
				// have dropped the rest — but it was never checked on the
				// wire: verified against production 2026-08-26, the API reads
				// only the LAST occurrence of a repeated key, so the addition
				// was discarded and the repair returned the same stripped
				// body as no repair at all. Joining keeps every value AND
				// survives first-wins, last-wins, and merge semantics alike.
				req.Params.Set("columns", strings.Join(append(missing, cols...), ","))
			}
		}
	}

	// Acquire concurrency pool slot
	if c.sem != nil {
		select {
		case c.sem <- struct{}{}:
			defer func() { <-c.sem }()
		case <-ctx.Done():
			return nil, c.ctxNetworkError(ctx, req)
		}
	}

	// Check rate limits before making the request. Reserve accounts for
	// in-flight requests so concurrent callers cannot all pass a
	// "remaining == 1" check and overshoot the limit. Demo mode is exempt:
	// anonymous responses carry limit=0 headers (the API's marker for
	// unmetered demo access), which would otherwise wedge the tracker into
	// rejecting every call after the first.
	var lastErr error
	var lastResp *Response

	// The reservation is per ATTEMPT, not per call. Every attempt in the
	// loop below is a real, billed request, so reserving once outside it
	// under-counted actual spend by up to MaxRetries+1 — measured at four
	// billed requests against one reserved credit for a single caller-visible
	// call against a flapping API.
	reserve := func() bool {
		if c.rateLimits == nil || c.demoMode {
			return true
		}
		return c.rateLimits.Reserve()
	}
	release := func() {
		if c.rateLimits != nil && !c.demoMode {
			c.rateLimits.Release()
		}
	}
	preFlightError := func() error {
		state := c.rateLimits.State()
		return &sdkerrors.RateLimitError{
			SupportContext: sdkerrors.SupportContext{
				StatusCode:    429,
				Timestamp:     time.Now().In(timezone.Eastern),
				Message:       "rate limit exceeded (pre-flight check)",
				ExceptionType: "RateLimitError",
			},
			Limit:     state.Limit,
			Remaining: state.Remaining,
			ResetAt:   state.ResetAt,
			PreFlight: true,
		}
	}

	for attempt := 0; attempt <= c.retryCfg.MaxRetries; attempt++ {
		// Wait before retry (skip for first attempt)
		if attempt > 0 {
			// Check API status before retrying — abort if service is offline
			if c.statusCache != nil && !c.statusCache.IsOnline() {
				c.logDebug("aborting retry: API status is offline", "attempt", attempt)
				if lastResp != nil {
					return lastResp, nil
				}
				return nil, fmt.Errorf("API is offline, aborting retry: %w", lastErr)
			}

			backoff := retry.CalculateBackoff(c.retryCfg, attempt-1)

			// Honor a server-supplied Retry-After delay, capped so a
			// hostile or broken value cannot park the retry loop.
			if lastResp != nil {
				if ra := retry.ParseRetryAfter(&http.Response{Header: lastResp.Headers}); ra > 0 {
					if ra <= retry.MaxRetryAfter {
						backoff = ra
					} else {
						c.logDebug("Retry-After exceeds cap, using calculated backoff", "retry_after", ra, "cap", retry.MaxRetryAfter)
					}
				}
			}

			c.logDebug("retrying request", "attempt", attempt, "backoff", backoff)

			if err := retry.Wait(ctx, backoff); err != nil {
				return nil, c.ctxNetworkError(ctx, req)
			}
		}

		// Reserve for this attempt. Running out mid-retry is treated like
		// the offline-status abort above: stop retrying and hand back what
		// we already have, rather than replacing a real failure with a
		// pre-flight rejection the caller never saw the cause of.
		if !reserve() {
			// On the first attempt there is nothing in hand, so the caller
			// gets the pre-flight rejection. Later, prefer what we already
			// have: replacing a real failure with a pre-flight rejection
			// would hide the cause the caller never got to see.
			//
			// lastResp and lastErr are not set together. A retryable status
			// sets both, but a transport failure sets only lastErr, so
			// checking lastResp alone dropped the network error on exactly
			// the case this branch exists to protect. Wrapping keeps the
			// cause reachable via errors.As, as the offline-abort branch
			// above already does for the identical shape.
			if lastResp != nil {
				return lastResp, nil
			}
			if lastErr != nil {
				return nil, fmt.Errorf("rate limit reached during retry: %w", lastErr)
			}
			return nil, preFlightError()
		}

		// The reservation is held until the tracker has absorbed this
		// response's remaining count, so the two guards overlap with no
		// instant uncovered: while the request is in flight the reservation
		// blocks a concurrent Reserve, and from Update onwards the new
		// remaining does. Releasing first reopened the concurrent-overshoot
		// window that `reserved` exists to close.
		//
		// The release is deferred inside a closure rather than called at the
		// end of the attempt: a panic escaping doOnce — a caller-supplied
		// RoundTripper is the realistic source — would otherwise skip it and
		// leak the credit permanently, since nothing ever decrements
		// `reserved` again. The pool-slot release above has always been
		// deferred for exactly this reason; the reservation was not.
		resp, err := func() (*Response, error) {
			defer release()
			resp, err := c.doOnce(ctx, req)
			if err == nil && c.rateLimits != nil {
				c.rateLimits.Update(&http.Response{Header: resp.Headers})
			}
			return resp, err
		}()
		if err != nil {
			if retry.ShouldRetryError(err) {
				lastErr = err
				continue
			}
			return nil, err
		}

		// Only retry 501-599 status codes
		if retry.ShouldRetryStatus(resp.StatusCode) {
			lastResp = resp
			lastErr = fmt.Errorf("request failed with status %d", resp.StatusCode)
			continue
		}

		// All other status codes: return immediately (no retry)
		return resp, nil
	}

	// All retries exhausted
	if lastResp != nil {
		return lastResp, nil
	}
	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// apiError maps a >=400 response to its typed error. Logging happens once,
// centrally, in Do's logTerminalFailure — not here — so a >=400 response
// isn't logged twice.
func (c *Client) apiError(resp *Response, requestURL string) error {
	return parseAPIError(resp, requestURL)
}

// requestURL builds the full URL a Request will be sent to.
func (c *Client) requestURL(req Request) string {
	if req.Unversioned {
		return c.buildURLUnversioned(req.Path, req.Params)
	}
	return c.buildURL(req.Path, req.Params)
}

// ctxNetworkError converts a fired context (cancellation or deadline) into
// the SDK's NetworkError taxonomy, so Do returns one consistent error shape
// no matter where the context fires: waiting for a pool slot, sleeping
// between retries, or mid-request (which the transport error path already
// wraps). The context error stays reachable via errors.Is through Cause.
func (c *Client) ctxNetworkError(ctx context.Context, req Request) *sdkerrors.NetworkError {
	err := ctx.Err()
	return &sdkerrors.NetworkError{
		SupportContext: sdkerrors.SupportContext{
			RequestURL:    c.requestURL(req),
			Timestamp:     time.Now().In(timezone.Eastern),
			Message:       err.Error(),
			ExceptionType: "NetworkError",
		},
		Timeout: errors.Is(err, context.DeadlineExceeded),
		Cause:   err,
	}
}

// doOnce executes a single request without retry.
func (c *Client) doOnce(ctx context.Context, req Request) (*Response, error) {
	fullURL := c.requestURL(req)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers. Refuse to attach the bearer token to an insecure transport:
	// a caller who points the base URL at a plain-http, non-loopback host must
	// not have the token shipped in cleartext to that origin.
	if !c.demoMode {
		if !tokenSafeForURL(httpReq.URL) {
			return nil, &sdkerrors.InsecureTokenError{
				Scheme: httpReq.URL.Scheme,
				Host:   httpReq.URL.Hostname(),
			}
		}
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}
	httpReq.Header.Set("User-Agent", "marketdata-sdk-go/"+Version())
	httpReq.Header.Set("Accept", "application/json")

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	start := time.Now()
	c.logDebug("sending request", "method", req.Method, "url", redactQuery(fullURL))

	// Execute request
	httpResp, err := c.http.Do(httpReq)
	if err != nil {
		netErr := &sdkerrors.NetworkError{
			SupportContext: sdkerrors.SupportContext{
				RequestURL:    fullURL,
				Timestamp:     time.Now().In(timezone.Eastern),
				Message:       err.Error(),
				ExceptionType: "NetworkError",
			},
			Cause: err,
		}
		// Populate Timeout from the underlying net.Error
		var ne net.Error
		if errors.As(err, &ne) {
			netErr.Timeout = ne.Timeout()
		}
		return nil, netErr
	}
	defer func() { _ = httpResp.Body.Close() }()

	// Read response body, bounded by the size cap so a hostile or broken
	// server streaming an unbounded body cannot exhaust memory. Read one byte
	// past the cap to detect overflow.
	body, err := io.ReadAll(io.LimitReader(httpResp.Body, c.maxRespBytes+1))
	if err != nil {
		// A body that dies mid-read — the connection dropped after the
		// headers arrived — is a transport failure like any other, and it is
		// exactly the transient class retries exist for. It used to return a
		// bare fmt.Errorf, which put it outside the taxonomy entirely:
		// errors.As against sdkerrors.Error missed it, it carried no
		// SupportContext, and retry.ShouldRetryError said no, so the SDK
		// never retried the one failure most likely to succeed on a second
		// attempt.
		netErr := &sdkerrors.NetworkError{
			SupportContext: sdkerrors.SupportContext{
				RequestURL:    fullURL,
				StatusCode:    httpResp.StatusCode,
				Timestamp:     time.Now().In(timezone.Eastern),
				Message:       "failed to read response body: " + err.Error(),
				ExceptionType: "NetworkError",
			},
			Cause: err,
		}
		// Classify before handing the error back. Timeout used to be
		// hardcoded false, on the premise that a stalling read is surfaced
		// earlier by the request timeout and the context paths, so nothing
		// reaching here could be a timeout and the branch would be
		// untestable. The premise is false in both halves: a context
		// deadline interrupts io.ReadAll and arrives exactly here, and so
		// does an http.Client.Timeout on its own — which is how the SDK's
		// own fixed 99-second timeout is applied (see New). The one field
		// the exported doc says classifies the failure therefore pointed the
		// wrong way for the SDK's own timeout, and Error() rendered it as
		// "network error" rather than "network timeout".
		//
		// Both checks are needed: a transport-level timeout satisfies
		// net.Error without wrapping context.DeadlineExceeded.
		var ne net.Error
		netErr.Timeout = errors.Is(err, context.DeadlineExceeded) ||
			(errors.As(err, &ne) && ne.Timeout())
		return nil, netErr
	}
	if int64(len(body)) > c.maxRespBytes {
		return nil, &sdkerrors.ResponseTooLargeError{
			SupportContext: sdkerrors.SupportContext{
				RequestURL:    redactQuery(fullURL),
				StatusCode:    httpResp.StatusCode,
				Timestamp:     time.Now().In(timezone.Eastern),
				Message:       "response body exceeded size cap",
				ExceptionType: "ResponseTooLargeError",
			},
			Limit: c.maxRespBytes,
		}
	}

	requestID := httpResp.Header.Get("X-Request-Id")
	if requestID == "" {
		requestID = httpResp.Header.Get("Cf-Ray")
	}

	c.logDebug("received response",
		"status", httpResp.StatusCode,
		"request_id", requestID,
		"body_size", len(body),
		"duration", time.Since(start).Round(time.Millisecond).String(),
	)

	return &Response{
		Raw:        httpResp,
		StatusCode: httpResp.StatusCode,
		Headers:    httpResp.Header,
		Body:       body,
		RequestID:  requestID,
	}, nil
}

// buildURL constructs the full URL for a request with API version prefix.
func (c *Client) buildURL(path string, params url.Values) string {
	path = strings.TrimPrefix(path, "/")
	u := fmt.Sprintf("%s/%s/%s", c.baseURL, c.apiVersion, path)

	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	return u
}

// buildURLUnversioned constructs a full URL without the API version prefix.
// Used for endpoints like /status/ that are not versioned.
func (c *Client) buildURLUnversioned(path string, params url.Values) string {
	path = strings.TrimPrefix(path, "/")
	u := fmt.Sprintf("%s/%s", c.baseURL, path)

	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	return u
}

// CloseIdleConnections closes any idle HTTP connections on the transport
// the SDK built for itself. It is a no-op when the underlying *http.Client
// was supplied via WithHTTPClient: that transport (and its connection
// pool) is intentionally shared with the caller, so closing its idle
// connections here would reach outside the SDK and affect the caller's
// own in-flight or pooled requests on that same client.
func (c *Client) CloseIdleConnections() {
	if !c.ownsTransport {
		return
	}
	c.http.CloseIdleConnections()
}

// StatusProbe issues a minimal direct GET against the unversioned /status/
// endpoint and reports whether the API answered as online (200 or 203). It
// deliberately bypasses the concurrency pool, retry loop, and rate-limit
// accounting: the offline-gate's background probe must stay responsive
// precisely when the pool is saturated or requests are failing, and must
// not pollute credit accounting.
func (c *Client) StatusProbe(ctx context.Context) (bool, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildURLUnversioned("status/", nil), nil)
	if err != nil {
		return false, err
	}
	httpReq.Header.Set("User-Agent", "marketdata-sdk-go/"+Version())
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	return resp.StatusCode == 200 || resp.StatusCode == 203, nil
}

// SetDebug enables or disables emission of per-request debug records at
// runtime. It is safe for concurrent use with in-flight requests.
func (c *Client) SetDebug(enabled bool) { c.debug.Store(enabled) }

// logDebug logs a debug message if debug mode is enabled.
func (c *Client) logDebug(msg string, args ...any) {
	if c.debug.Load() {
		c.logger.Debug(msg, args...)
	}
}

// Get executes a GET request and decodes the JSON response, logging any
// terminal failure at ERROR (see logTerminalFailure).
func (c *Client) Get(ctx context.Context, path string, params url.Values, result any) (*Response, error) {
	resp, err := c.get(ctx, path, params, result)
	if err != nil {
		c.logTerminalFailure(ctx, err)
	}
	return resp, err
}

func (c *Client) get(ctx context.Context, path string, params url.Values, result any) (*Response, error) {
	resp, err := c.Do(ctx, Request{
		Method:          http.MethodGet,
		Path:            path,
		Params:          params,
		RequiredColumns: requiredColumns(result),
	})
	if err != nil {
		return nil, err
	}

	requestURL := resp.wireURL(c.buildURL(path, params))

	// A 404 is the API's no-data signal, but it doubles as its "the question
	// itself was invalid" signal: a nonexistent symbol answers 404 too. The
	// two are separated only by errmsg, so a 404 that names an error is
	// mapped like any other failure — waking parseAPIError's NotFoundError
	// branch — instead of being reported to the caller as an empty answer.
	// See notFoundNamesAnError.
	if resp.StatusCode == 404 && notFoundNamesAnError(resp) {
		return resp, c.apiError(resp, requestURL)
	}

	// 404 (no data) and 204 (mode=cached cache miss) return the response
	// without an error and without decoding — services handle NoData. A 204
	// body is empty, so decoding it would spuriously fail.
	if resp.StatusCode == 404 || resp.StatusCode == 204 {
		return resp, nil
	}

	if resp.StatusCode >= 400 {
		return resp, c.apiError(resp, requestURL)
	}

	if result != nil {
		if err := json.Unmarshal(resp.Body, result); err != nil {
			return resp, &sdkerrors.ParseError{
				SupportContext: sdkerrors.SupportContext{
					RequestID:     resp.RequestID,
					RequestURL:    requestURL,
					StatusCode:    resp.StatusCode,
					Timestamp:     time.Now().In(timezone.Eastern),
					Message:       err.Error(),
					ExceptionType: "ParseError",
				},
				Cause: err,
			}
		}
	}

	return resp, nil
}

// GetUnversioned executes a GET request without API version prefix.
// Used for endpoints like /status/ that are not versioned. Logs any
// terminal failure at ERROR (see logTerminalFailure).
func (c *Client) GetUnversioned(ctx context.Context, path string, params url.Values, result any) (*Response, error) {
	resp, err := c.getUnversioned(ctx, path, params, result)
	if err != nil {
		c.logTerminalFailure(ctx, err)
	}
	return resp, err
}

// GetUnversionedSilent is [GetUnversioned] without the ERROR log on
// failure. Reserved for best-effort background priming (rate-limit state
// at startup) whose caller already discards the error by design — logging
// it there would either be pure noise or duplicate an ERROR already logged
// by a synchronous call to the same endpoint moments earlier.
func (c *Client) GetUnversionedSilent(ctx context.Context, path string, params url.Values, result any) (*Response, error) {
	return c.getUnversioned(ctx, path, params, result)
}

func (c *Client) getUnversioned(ctx context.Context, path string, params url.Values, result any) (*Response, error) {
	resp, err := c.Do(ctx, Request{
		Method:          http.MethodGet,
		Path:            path,
		Params:          params,
		Unversioned:     true,
		RequiredColumns: requiredColumns(result),
	})
	if err != nil {
		return nil, err
	}

	requestURL := resp.wireURL(c.buildURLUnversioned(path, params))

	// A 404 is the API's no-data signal, but it doubles as its "the question
	// itself was invalid" signal: a nonexistent symbol answers 404 too. The
	// two are separated only by errmsg, so a 404 that names an error is
	// mapped like any other failure — waking parseAPIError's NotFoundError
	// branch — instead of being reported to the caller as an empty answer.
	// See notFoundNamesAnError.
	if resp.StatusCode == 404 && notFoundNamesAnError(resp) {
		return resp, c.apiError(resp, requestURL)
	}

	// 404 (no data) and 204 (mode=cached cache miss) return the response
	// without an error and without decoding — services handle NoData. A 204
	// body is empty, so decoding it would spuriously fail.
	if resp.StatusCode == 404 || resp.StatusCode == 204 {
		return resp, nil
	}

	if resp.StatusCode >= 400 {
		return resp, c.apiError(resp, requestURL)
	}

	if result != nil {
		if err := json.Unmarshal(resp.Body, result); err != nil {
			return resp, &sdkerrors.ParseError{
				SupportContext: sdkerrors.SupportContext{
					RequestID:     resp.RequestID,
					RequestURL:    requestURL,
					StatusCode:    resp.StatusCode,
					Timestamp:     time.Now().In(timezone.Eastern),
					Message:       err.Error(),
					ExceptionType: "ParseError",
				},
				Cause: err,
			}
		}
	}

	return resp, nil
}

// namesColumn reports whether a columns parameter already names col. Values
// may arrive as one comma-separated string (what WithColumns builds) or as
// several repeated values, so both shapes are scanned.
func namesColumn(values []string, col string) bool {
	for _, v := range values {
		for _, c := range strings.Split(v, ",") {
			if strings.TrimSpace(c) == col {
				return true
			}
		}
	}
	return false
}

// cloneValues copies a url.Values so a caller-owned map is never written to.
func cloneValues(v url.Values) url.Values {
	out := make(url.Values, len(v))
	for key, vals := range v {
		out[key] = vals
	}
	return out
}

// mediaTypeForFormat maps a wire format value (as sent in the "format" query
// parameter) to the Accept header that requests it. Only called with the
// two formats GetFormatted's own callers use — the CSV/HTML facets (see
// ADR-018) — so an unrecognized value is a programmer error, not a runtime
// one; it falls back to JSON rather than sending a bogus Accept header.
func mediaTypeForFormat(format string) string {
	switch format {
	case "csv":
		return "text/csv"
	case "html":
		return "text/html"
	default:
		return "application/json"
	}
}

// GetFormatted executes a GET request asking for a non-JSON wire format
// (format=csv or format=html, see ADR-018) and returns the raw response
// body as-is — no JSON decoding. Logs any terminal failure at ERROR (see
// logTerminalFailure), same as Get.
//
// Unlike Get, this has no NoData concept: the API's own "no data" body
// shape is not consistent between JSON (404, a typed no-data body) and
// these formats (verified live: the same no-data condition on a
// CSV-formatted candles request comes back 200 with a degenerate body, not
// 404) — so 404/204 are passed through like any other status rather than
// specially interpreted. The caller gets whatever text came back.
func (c *Client) GetFormatted(ctx context.Context, path string, params url.Values, format string) (*Response, error) {
	// Copy before setting "format": the map belongs to the caller and must
	// not be mutated (the same contract Do documents for its own default
	// merge). Without this, a caller reusing one map across a JSON Get and a
	// GetFormatted leaks format=csv into the JSON request, and two
	// concurrent GetFormatted calls sharing a map race on a concurrent map
	// write.
	merged := make(url.Values, len(params)+1)
	for key, vals := range params {
		merged[key] = vals
	}
	for key, vals := range c.formatOnlyParams {
		if _, ok := merged[key]; !ok {
			merged[key] = append([]string(nil), vals...)
		}
	}
	merged.Set("format", format)
	params = merged

	resp, err := c.Do(ctx, Request{
		Method:    http.MethodGet,
		Path:      path,
		Params:    params,
		Headers:   map[string]string{"Accept": mediaTypeForFormat(format)},
		RawFormat: true,
	})
	if err != nil {
		c.logTerminalFailure(ctx, err)
		return nil, err
	}

	// A 404 that names an error is the API rejecting the question, not an
	// empty answer, and is reported as an error here exactly as it is on
	// the JSON path (see notFoundNamesAnError). On this path the empty
	// answer does not even use 404: verified live, an impossible filter
	// with format=csv comes back 200 with a degenerate body, so in
	// practice every 404 here carries the marker.
	if resp.StatusCode == 404 && notFoundNamesAnError(resp) {
		err := c.apiError(resp, resp.wireURL(c.buildURL(path, params)))
		c.logTerminalFailure(ctx, err)
		return resp, err
	}

	// 404/204 are the API's no-data signal for JSON — Get treats them as
	// success and lets the service layer set NoData. There is no NoData
	// concept here (see the doc comment above), but the same statuses still
	// must not be treated as *errors*: they're the same "found nothing, not
	// a failure" condition, just carrying whatever body this format put in
	// a 200 would put there. Only a genuine >=400 elsewhere is an error.
	if resp.StatusCode == 404 || resp.StatusCode == 204 {
		return resp, nil
	}

	if resp.StatusCode >= 400 {
		err := c.apiError(resp, resp.wireURL(c.buildURL(path, params)))
		c.logTerminalFailure(ctx, err)
		return resp, err
	}

	return resp, nil
}

// apiErrorResponse represents the JSON error body returned by the API.
type apiErrorResponse struct {
	Status               string `json:"s"`
	ErrMsg               string `json:"errmsg"`
	Message              string `json:"message"`
	TroubleshootingGuide string `json:"troubleshootingGuide"`
	AuthorizedIP         string `json:"authorizedIP"`
	BlockedIP            string `json:"blockedIP"`
}

// parseCSVErrorFields parses an error body requested with format=csv (e.g.
// "s,errmsg\nerror,\"Bad parameters...\"") into the same field set
// apiErrorResponse's JSON tags carry, so parseAPIError extracts errmsg /
// troubleshootingGuide / etc. the same way regardless of which wire format
// the request asked for — the API serializes errors in the requested
// format, not always JSON (see ADR-018). Read by column name rather than
// position so it degrades gracefully (empty fields, handled the same as a
// JSON body missing those keys) if a status code's CSV shape omits a
// column this SDK hasn't observed. Malformed or truncated CSV yields a
// zero-value apiErrorResponse, same as a JSON parse failure does today.
func parseCSVErrorFields(body []byte) apiErrorResponse {
	r := csv.NewReader(bytes.NewReader(body))
	header, err := r.Read()
	if err != nil {
		return apiErrorResponse{}
	}
	row, err := r.Read()
	if err != nil {
		return apiErrorResponse{}
	}
	fields := make(map[string]string, len(header))
	for i, col := range header {
		if i < len(row) {
			fields[col] = row[i]
		}
	}
	return apiErrorResponse{
		Status:               fields["s"],
		ErrMsg:               fields["errmsg"],
		Message:              fields["message"],
		TroubleshootingGuide: fields["troubleshootingGuide"],
		AuthorizedIP:         fields["authorizedIP"],
		BlockedIP:            fields["blockedIP"],
	}
}

// maxErrorMessageLen bounds server-supplied error messages embedded in SDK
// errors, so a hostile or broken response body cannot balloon logs.
const maxErrorMessageLen = 500

// sanitizeMessage prepares a server-supplied string for embedding in error
// messages and logs: control characters (CR/LF/TAB/ESC, etc.) are collapsed
// to spaces to prevent log forging and ANSI terminal spoofing, and the
// result is truncated to maxErrorMessageLen.
func sanitizeMessage(s string) string {
	sanitized := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)

	if len(sanitized) > maxErrorMessageLen {
		// Back off to a rune boundary so the byte cut cannot split a
		// multi-byte UTF-8 sequence and leave invalid bytes in the message.
		cut := maxErrorMessageLen
		for cut > 0 && !utf8.RuneStart(sanitized[cut]) {
			cut--
		}
		sanitized = sanitized[:cut] + "…(truncated)"
	}
	return sanitized
}

// redactQuery strips the query string from a URL for ambient logging,
// leaving a "?…" marker so logs never persist request parameters.
func redactQuery(fullURL string) string {
	if i := strings.IndexByte(fullURL, '?'); i >= 0 {
		return fullURL[:i] + "?…"
	}
	return fullURL
}

// parseErrorBody decodes the API's error envelope from a response body,
// in whichever wire format the request asked for — the API serializes
// errors in the requested format, not always JSON (see ADR-018).
func parseErrorBody(resp *Response) apiErrorResponse {
	if strings.Contains(resp.Headers.Get("Content-Type"), "text/csv") {
		return parseCSVErrorFields(resp.Body)
	}
	var apiResp apiErrorResponse
	_ = json.Unmarshal(resp.Body, &apiResp)
	return apiResp
}

// notFoundNamesAnError reports whether a 404 body names an error message.
//
// The API answers two different things with 404: "your question was
// invalid" (a nonexistent symbol, an OCC symbol matching no contract) and
// "your valid question has an empty answer" (a filter that matched
// nothing). Both carry s:"no_data" or s:"error", so the status field does
// not separate them — the only discriminator on the wire is errmsg, which
// the invalid question carries and the empty answer omits.
//
// Verified live 2026-09-01 against production:
//
//	options/chain/ZZZZQQ/       404 {"s":"no_data","errmsg":"Symbol not found."}
//	options/chain/AAPL/ + an
//	  impossible filter         404 {"s":"no_data","nextTime":null,"prevTime":null}
//	stocks/quotes/ZZZZQQ/       404 {"s":"no_data","errmsg":"Symbol not found."}
//	options/quotes/<bogus OCC>  404 {"s":"error","errmsg":"No option found..."}
//	options/chain/ZZZZQQ/
//	  ?format=csv               404 text/csv "s,errmsg\nno_data,Symbol not found."
//
// Not every endpoint supplies the marker: options/expirations and
// stocks/candles answer a nonexistent symbol with a markerless 404,
// indistinguishable from an empty result. Those stay NoData — the wire
// gives nothing to tell them apart, and inventing a distinction the API
// does not make would be worse than reporting the empty answer it sent.
func notFoundNamesAnError(resp *Response) bool {
	return strings.TrimSpace(parseErrorBody(resp).ErrMsg) != ""
}

// parseAPIError parses an error response from the API into the appropriate
// classified error type based on HTTP status code.
func parseAPIError(resp *Response, requestURL string) error {
	apiResp := parseErrorBody(resp)

	message := apiResp.ErrMsg
	if message == "" {
		message = apiResp.Message
	}
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}
	message = sanitizeMessage(message)

	ctx := sdkerrors.SupportContext{
		RequestID:  resp.RequestID,
		RequestURL: requestURL,
		StatusCode: resp.StatusCode,
		Timestamp:  time.Now().In(timezone.Eastern),
		Message:    message,
	}

	switch {
	case resp.StatusCode == 400:
		ctx.ExceptionType = "BadRequestError"
		return &sdkerrors.BadRequestError{SupportContext: ctx}
	case resp.StatusCode == 401:
		ctx.ExceptionType = "AuthenticationError"
		return &sdkerrors.AuthenticationError{SupportContext: ctx}
	case resp.StatusCode == 402:
		ctx.ExceptionType = "PaymentRequiredError"
		return &sdkerrors.PaymentRequiredError{SupportContext: ctx}
	case resp.StatusCode == 403:
		ctx.ExceptionType = "ForbiddenError"
		return &sdkerrors.ForbiddenError{
			SupportContext:       ctx,
			AuthorizedIP:         sanitizeMessage(apiResp.AuthorizedIP),
			BlockedIP:            sanitizeMessage(apiResp.BlockedIP),
			TroubleshootingGuide: sanitizeMessage(apiResp.TroubleshootingGuide),
		}
	case resp.StatusCode == 404:
		// Reached for a 404 whose body names an error — the API's "your
		// question was invalid" answer, which Get, GetUnversioned and
		// GetFormatted route here rather than reporting as no data (see
		// notFoundNamesAnError). A markerless 404 never arrives: that is
		// the "empty answer" case, intercepted as NoData before the
		// error mapping runs.
		ctx.ExceptionType = "NotFoundError"
		return &sdkerrors.NotFoundError{SupportContext: ctx}
	case resp.StatusCode == 413:
		ctx.ExceptionType = "PayloadTooLargeError"
		return &sdkerrors.PayloadTooLargeError{SupportContext: ctx}
	case resp.StatusCode == 429:
		ctx.ExceptionType = "RateLimitError"
		rl := ratelimit.ParseHeaders(resp.Headers)
		return &sdkerrors.RateLimitError{
			SupportContext:       ctx,
			Limit:                rl.Limit,
			Remaining:            rl.Remaining,
			ResetAt:              rl.ResetAt,
			TroubleshootingGuide: sanitizeMessage(apiResp.TroubleshootingGuide),
		}
	case resp.StatusCode == 500:
		ctx.ExceptionType = "InternalError"
		return &sdkerrors.InternalError{SupportContext: ctx}
	case resp.StatusCode > 500:
		ctx.ExceptionType = "ServerError"
		return &sdkerrors.ServerError{SupportContext: ctx}
	default:
		ctx.ExceptionType = "BadRequestError"
		return &sdkerrors.BadRequestError{SupportContext: ctx}
	}
}
