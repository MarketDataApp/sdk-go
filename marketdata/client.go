package marketdata

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/dotenv"
	internalhttp "github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/ratelimit"
	"github.com/MarketDataApp/sdk-go/v2/internal/retry"
	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
	"github.com/MarketDataApp/sdk-go/v2/internal/status"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/funds"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/markets"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/utilities"
)

// Client is the entry point to the Market Data API. It holds the
// configuration, the underlying HTTP client with retry and rate limit
// tracking, and one service per API resource. Create a Client with
// [NewClient] and release its resources with [Client.Close] when done.
//
// A Client is safe for concurrent use across goroutines and limits itself
// to at most 50 concurrent in-flight requests; additional calls block until
// a slot frees.
type Client struct {
	// Configuration (immutable after creation)
	config *Config

	// HTTP client for making requests
	http *internalhttp.Client

	// Rate limit tracker
	rateLimits *ratelimit.Tracker

	// Concurrency pool (50 max concurrent requests)
	sem chan struct{}

	// Close handling
	closeOnce sync.Once

	// Stocks provides stock quotes, candles, bulk prices, earnings, and news.
	Stocks *stocks.Service

	// Options provides option chains, expiration dates, option quotes, and
	// option symbol lookup.
	Options *options.Service

	// Funds provides mutual fund candles.
	Funds *funds.Service

	// Markets provides market status (open or closed) and status history.
	Markets *markets.Service

	// Utilities provides API status, response header echo, and account
	// details for the authenticated user.
	Utilities *utilities.Service
}

// NewClient creates a new Market Data client configured by the given
// options.
//
// NewClient first loads a .env file from the working directory if one
// exists; values from .env never override variables already set in the
// process environment. The API token is then resolved in priority order:
// the [WithToken] option if provided, otherwise the MARKETDATA_TOKEN
// environment variable. If no token is found, the client starts in demo
// mode with access limited to unauthenticated endpoints, and a warning is
// logged.
//
// When a token is present, NewClient validates it against the API with a
// synchronous request and returns an error if the token is rejected;
// use [WithoutStartupValidation] to skip this check. Rate limit state is
// then initialized in the background without blocking the caller.
//
// Timeouts are fixed and cannot be overridden: 99 seconds per request and
// 2 seconds for the TCP connection dial. Failed requests are retried up to
// 3 times by default (configurable with [WithMaxRetries]) with exponential
// backoff starting at 1 second and doubling each attempt; only 501-599
// status codes and transient network errors are retried.
//
// Example:
//
//	// Using environment variable (recommended)
//	client, err := marketdata.NewClient()
//
//	// Using explicit token
//	client, err := marketdata.NewClient(
//	    marketdata.WithToken("your-token"),
//	)
func NewClient(opts ...Option) (*Client, error) {
	// Options are pure field setters, so applying them twice is harmless;
	// this first pass exists only to learn noDotEnv before the defaults
	// (which consult .env) are computed.
	peek := &Config{}
	for _, opt := range opts {
		opt.apply(peek)
	}

	// Parse .env from the working directory into the config cascade's
	// fallback layer. The process environment is never modified; a missing
	// file simply yields no fallbacks.
	var env envSource
	if !peek.noDotEnv {
		env, _ = dotenv.Parse(".env")
	}

	// Start with defaults
	cfg := defaultConfig(env)

	// Apply user options
	for _, opt := range opts {
		opt.apply(cfg)
	}

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	// WithDebug on the default logger raises its threshold so Debug
	// records actually surface; an injected WithLogger handler owns its
	// own level (logLevel is nil then).
	if cfg.debug && cfg.logLevel != nil {
		cfg.logLevel.Set(slog.LevelDebug)
	}

	// Create rate limit tracker
	rateLimits := ratelimit.New()

	// Create concurrency pool
	sem := make(chan struct{}, internalhttp.DefaultPoolSize)

	// Create the API status cache. Its fetcher calls GET /status/ through
	// the HTTP client constructed just below; the closure captures the
	// variable, which is assigned before any request can run.
	var httpClient *internalhttp.Client
	statusCache := status.New(func(ctx context.Context) (bool, error) {
		// StatusProbe bypasses the concurrency pool, retries, and
		// rate-limit accounting: the gate must stay responsive precisely
		// when the pool is saturated by a failing API.
		return httpClient.StatusProbe(ctx)
	})

	// Resolve max retries: user-configured or default
	maxRetries := defaultMaxRetries
	if cfg.maxRetries != nil {
		maxRetries = *cfg.maxRetries
	}

	// Create HTTP client with timeout and retry settings
	httpClient = internalhttp.New(internalhttp.Config{
		HTTPClient:  cfg.httpClient,
		BaseURL:     cfg.baseURL,
		APIVersion:  cfg.apiVersion,
		Token:       cfg.token,
		DemoMode:    cfg.demoMode,
		Timeout:     fixedTimeout,
		ConnTimeout: fixedConnTimeout,
		RetryCfg: retry.Config{
			MaxRetries:     maxRetries,
			InitialBackoff: fixedInitialBackoff,
			Multiplier:     fixedBackoffMultiplier,
		},
		RateLimits:       rateLimits,
		Logger:           cfg.logger,
		Debug:            cfg.debug,
		Sem:              sem,
		DefaultParams:    cfg.defaultParams,
		FormatOnlyParams: cfg.formatOnlyParams,
		StatusCache:      statusCache,
	})

	// Create client
	c := &Client{
		config:     cfg,
		http:       httpClient,
		rateLimits: rateLimits,
		sem:        sem,
	}

	// Initialize resources
	c.Stocks = stocks.NewService(httpClient)
	c.Options = options.NewService(httpClient)
	c.Funds = funds.NewService(httpClient)
	c.Markets = markets.NewService(httpClient)
	c.Utilities = utilities.NewService(httpClient)

	// Log redacted token at debug level
	if cfg.debug && cfg.token != "" {
		cfg.logger.Debug("using API token", "token", redactToken(cfg.token))
	}

	// Handle startup behavior based on mode
	if cfg.demoMode {
		// Demo mode: warn and skip rate limit initialization
		cfg.logger.Warn("No API token provided — running in demo mode with limited access")
	} else if !cfg.skipValidation {
		// Synchronous token validation. The /user/ endpoint is
		// unversioned; the versioned path would 404 and silently
		// skip validation.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err := httpClient.GetUnversioned(ctx, "user/", nil, nil)
		if err != nil {
			// Check if it's an authentication error
			var authErr *sdkerrors.AuthenticationError
			if errors.As(err, &authErr) {
				return nil, fmt.Errorf("token validation failed: %w", err)
			}
			// Non-auth errors are warnings, not failures
		}
	}

	// Initialize rate limits in background (skip in demo mode)
	if !cfg.demoMode {
		go c.initRateLimits()
	}

	// Requirements §7 log point: construction success at INFO.
	cfg.logger.Info("client initialized", "base_url", cfg.baseURL, "api_version", cfg.apiVersion)

	return c, nil
}

// initRateLimits fetches initial rate limit information.
// This runs in the background and doesn't block client creation.
func (c *Client) initRateLimits() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Make a lightweight request to populate rate limits. The /user/
	// endpoint is unversioned. Silent: this is best-effort priming whose
	// error is deliberately discarded (ADR-006 deviation — async and
	// silent), and a bad token already produced one ERROR log from the
	// synchronous startup validation above; logging here too would just
	// duplicate it for the same underlying failure.
	_, _ = c.http.GetUnversionedSilent(ctx, "user/", nil, nil)
}

// Version reports the SDK's own version as recorded in the caller's build
// info (ADR-015): the module version when this SDK is a dependency, or
// "unknown" when built from source without a stamped main-module version
// (e.g. a plain "go build" with no VCS info) or when build info is
// unavailable at all. It matches the version sent in the User-Agent header
// of every request.
func Version() string {
	return internalhttp.Version()
}

// RateLimits returns the client's snapshot of the rate limit state as of
// the most recently completed request. The snapshot is convenient for
// monitoring credit consumption, but when requests run concurrently it may
// lag behind the true server-side state; for exact, request-scoped values
// use the RateLimit field of the [Response] returned by each context-first
// method instead. Before any request has completed (or in demo mode) the
// returned state is zero-valued.
func (c *Client) RateLimits() RateLimitState {
	state := c.rateLimits.State()
	return RateLimitState{
		Limit:     state.Limit,
		Remaining: state.Remaining,
		Consumed:  state.Consumed,
		ResetAt:   state.ResetAt,
	}
}

// DemoMode reports whether the client is running in demo mode because no
// API token was provided. In demo mode access is limited to
// unauthenticated endpoints and sample data; applications can use this to
// display a demo banner or constrain their feature set.
func (c *Client) DemoMode() bool {
	return c.config.demoMode
}

// RateLimitState is the client-level snapshot of rate limit information
// returned by [Client.RateLimits]. It reflects the headers of the most
// recently completed request, not necessarily the request the caller just
// made.
type RateLimitState struct {
	// Limit is the maximum number of requests allowed in the current window.
	Limit int

	// Remaining is the number of requests remaining in the current window.
	Remaining int

	// Consumed is the number of requests consumed in the current window.
	Consumed int

	// ResetAt is when the current rate limit window resets.
	ResetAt time.Time
}

// Close releases resources held by the client by closing idle HTTP
// connections. After Close is called, the client must not be reused.
// Close is safe to call multiple times; subsequent calls are no-ops.
// It always returns nil; the error result exists to satisfy io.Closer.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		c.http.CloseIdleConnections()
	})
	return nil
}

// Debug enables or disables debug logging at runtime. It is a convenience
// method equivalent to having passed [WithDebug] to [NewClient], useful for
// turning verbose request logging on temporarily while diagnosing an issue.
// On the SDK's default logger it also adjusts the log level (to DEBUG, and
// back to the configured base level when disabled); a logger injected with
// [WithLogger] keeps its own level.
func (c *Client) Debug(enabled bool) {
	// The internal HTTP client owns the runtime debug state: without this,
	// its per-request records ("sending request", retry traces) are never
	// emitted, no matter the logger level.
	c.http.SetDebug(enabled)
	if c.config.logLevel != nil {
		if enabled {
			c.config.logLevel.Set(slog.LevelDebug)
		} else {
			c.config.logLevel.Set(c.config.baseLogLevel)
		}
	}
}
