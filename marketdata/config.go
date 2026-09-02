package marketdata

import (
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

// Environment selects which Market Data API deployment the client talks
// to. Pass one of the predefined values to [WithEnvironment]; each maps to
// a base URL that can be further overridden with [WithBaseURL].
type Environment string

const (
	// Production is the live API environment at https://api.marketdata.app.
	// It is the default.
	Production Environment = "production"

	// Test is the test/sandbox API environment at
	// https://test.api.marketdata.app.
	Test Environment = "test"

	// Development is the local development environment at
	// http://localhost:8080.
	Development Environment = "development"

	// fixedTimeout is the fixed HTTP request timeout (not configurable).
	fixedTimeout = 99 * time.Second

	// fixedConnTimeout is the fixed connection timeout (not configurable).
	// This covers the TCP dial (DNS + SYN/SYN-ACK), not the full request.
	fixedConnTimeout = 2 * time.Second

	// defaultMaxRetries is the default maximum number of retry attempts.
	defaultMaxRetries = 3

	// fixedInitialBackoff is the fixed initial backoff duration.
	fixedInitialBackoff = 1 * time.Second

	// fixedBackoffMultiplier is the fixed backoff multiplier.
	fixedBackoffMultiplier = 2.0
)

// baseURLs maps environments to their API base URLs
var baseURLs = map[Environment]string{
	Production:  "https://api.marketdata.app",
	Test:        "https://test.api.marketdata.app",
	Development: "http://localhost:8080",
}

// Token environment variable names in priority order.
// The first non-empty value found will be used.
var tokenEnvVars = []string{
	"MARKETDATA_TOKEN", // Primary (matches Python SDK)
}

// envSource is the .env layer of the configuration cascade (requirements
// §4: .env < environment < client options < method params). Lookups check
// the real process environment first and fall back to the values parsed
// from the .env file; the process environment itself is never modified. A
// nil source reads only the real environment.
type envSource map[string]string

func (e envSource) get(key string) string {
	// LookupEnv, not Getenv: a real environment variable explicitly set to
	// "" (e.g. `MARKETDATA_TOKEN=` to force demo mode in CI despite a .env
	// file with a token) must win over the .env fallback. Getenv's plain
	// non-empty check can't tell "unset" from "set to empty" and silently
	// preferred the .env value in the latter case.
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return e[key]
}

// getTokenFromEnv returns the API token from environment variables.
// It checks environment variables in priority order and returns
// the first non-empty value found.
//
// Priority order:
//  1. MARKETDATA_TOKEN (primary, matches Python SDK)
//
// Returns empty string if no token is found.
func getTokenFromEnv(env envSource) string {
	for _, envVar := range tokenEnvVars {
		if token := env.get(envVar); token != "" {
			return token
		}
	}
	return ""
}

// redactToken returns a redacted version of the token showing only the last 4 characters.
// Returns an empty string if the token is too short.
func redactToken(token string) string {
	if len(token) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(token)-4) + token[len(token)-4:]
}

// universalParamsFromEnv reads universal parameter environment variables and
// returns them split the same way the client options are: params sent on
// every request, and params that only cohere with the formatted facets.
// MARKETDATA_USE_HUMAN_READABLE is in the second group for the reason
// [WithHumanReadable] documents — it renames every field in the response.
func universalParamsFromEnv(env envSource) (defaults, formatOnly url.Values) {
	defaults, formatOnly = url.Values{}, url.Values{}

	envMap := map[string]string{
		"MARKETDATA_DATE_FORMAT": "dateformat",
		"MARKETDATA_COLUMNS":     "columns",
		"MARKETDATA_ADD_HEADERS": "headers",
		"MARKETDATA_MODE":        "mode",
	}

	for envVar, param := range envMap {
		if v := env.get(envVar); v != "" {
			defaults.Set(param, v)
		}
	}
	if v := env.get("MARKETDATA_USE_HUMAN_READABLE"); v != "" {
		formatOnly.Set("human", v)
	}

	return defaults, formatOnly
}

// Config holds the client configuration assembled by [NewClient] from
// defaults, environment variables, and [Option] values. It is unexported
// field by field and immutable after client creation; use the With*
// functional options to influence it.
type Config struct {
	// API authentication
	token string

	// API settings
	baseURL     string
	apiVersion  string
	environment Environment

	// HTTP settings
	httpClient *http.Client
	maxRetries *int // nil means use default (3)

	// Logging
	logger *slog.Logger
	debug  bool
	// logLevel drives the default canonical logger's threshold at runtime
	// (WithDebug / Client.Debug raise it to DEBUG). It is nil when the
	// caller injected a logger with WithLogger — their handler owns the
	// level then.
	logLevel *slog.LevelVar
	// baseLogLevel is the level to restore when Client.Debug(false) is
	// called on the default logger.
	baseLogLevel slog.Level

	// Startup behavior
	skipValidation bool
	demoMode       bool
	noDotEnv       bool

	// Universal default params from env vars / client options
	defaultParams url.Values

	// formatOnlyParams are universal parameters that only cohere with the
	// CSV/HTML facets and would break the typed JSON methods. They are
	// merged by the formatted request path alone.
	formatOnlyParams url.Values
}

// logLevelFromEnv returns a slog.Level based on the MARKETDATA_LOGGING_LEVEL environment variable.
// Returns the level and true if set, or slog.LevelInfo and false if not set or invalid.
func logLevelFromEnv(env envSource) (slog.Level, bool) {
	level := env.get("MARKETDATA_LOGGING_LEVEL")
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug, true
	case "INFO":
		return slog.LevelInfo, true
	case "WARNING", "WARN":
		return slog.LevelWarn, true
	case "ERROR":
		return slog.LevelError, true
	default:
		return slog.LevelInfo, false
	}
}

// defaultConfig returns a Config with sensible defaults. Key settings fall
// back to the environment through env (requirements §4 cascade: .env <
// environment < client options < method params — env.get checks the real
// environment before the parsed .env values, and With* options are applied
// after these defaults, so the cascade holds).
func defaultConfig(env envSource) *Config {
	baseURL := baseURLs[Production]
	if v := env.get("MARKETDATA_BASE_URL"); v != "" {
		baseURL = v
	}
	apiVersion := "v1"
	if v := env.get("MARKETDATA_API_VERSION"); v != "" {
		apiVersion = v
	}

	// Default logger: the canonical cross-SDK format (requirements §7) at
	// INFO, or the MARKETDATA_LOGGING_LEVEL level when set. The dynamic
	// LevelVar lets WithDebug / Client.Debug raise it to DEBUG later.
	level, levelSet := logLevelFromEnv(env)
	levelVar := new(slog.LevelVar)
	levelVar.Set(level)

	envDefaults, envFormatOnly := universalParamsFromEnv(env)

	cfg := &Config{
		// Token from environment (priority order)
		token: getTokenFromEnv(env),

		// Default to production; overridable via environment
		environment: Production,
		baseURL:     baseURL,
		apiVersion:  apiVersion,

		// Logging
		logger:       slog.New(newCanonicalHandler(os.Stderr, levelVar)),
		debug:        levelSet && level == slog.LevelDebug,
		logLevel:     levelVar,
		baseLogLevel: level,

		// Universal params from environment
		defaultParams:    envDefaults,
		formatOnlyParams: envFormatOnly,
	}

	return cfg
}

// validate checks the configuration for required fields.
func (c *Config) validate() error {
	if c.token == "" {
		c.demoMode = true
	}

	// A token with non-printable or non-ASCII characters (e.g. a stray CR
	// from a .env file) would otherwise fail cryptically inside the HTTP
	// stack on the first request.
	for i := 0; i < len(c.token); i++ {
		if c.token[i] < 0x20 || c.token[i] > 0x7e {
			return &sdkerrors.ValidationError{
				Field:   "token",
				Message: "token contains non-printable or non-ASCII characters (check for stray whitespace or line endings in your token source)",
			}
		}
	}

	u, err := url.Parse(c.baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return &sdkerrors.ValidationError{
			Field:   "baseURL",
			Message: "base URL must be a valid http(s) URL with a host",
		}
	}

	// apiVersion is interpolated directly into the request path
	// ("baseURL/apiVersion/path"); reject anything that could escape the
	// intended path segment (e.g. "../.." from a mistyped env var or option).
	if c.apiVersion == "" || strings.ContainsAny(c.apiVersion, "/\\") || strings.Contains(c.apiVersion, "..") {
		return &sdkerrors.ValidationError{
			Field:   "apiVersion",
			Message: "API version must be a single path segment without \"/\" or \"..\"",
		}
	}

	return nil
}

// Option configures the client during [NewClient]. Options are applied in
// the order given, after defaults and environment variables, so an
// explicit option always wins over the environment and a later option
// wins over an earlier one.
type Option interface {
	apply(*Config)
}

// optionFunc is a function adapter for Option.
type optionFunc func(*Config)

func (f optionFunc) apply(c *Config) {
	f(c)
}

// WithToken sets the API token used to authenticate every request. An
// explicit token has the highest priority in the configuration cascade: it
// overrides the MARKETDATA_TOKEN environment variable and any value loaded
// from a .env file. When neither WithToken nor the environment supplies a
// token, the client runs in demo mode with access limited to
// unauthenticated endpoints.
func WithToken(token string) Option {
	return optionFunc(func(c *Config) {
		c.token = token
	})
}

// WithAPIKey sets the API token for authentication.
//
// Deprecated: WithAPIKey is an alias for [WithToken] kept for backwards
// compatibility. Use WithToken instead.
func WithAPIKey(key string) Option {
	return WithToken(key)
}

// WithEnvironment selects the API environment ([Production], [Test], or
// [Development]) and sets the corresponding base URL. The default is
// [Production]. To point the client at an arbitrary URL instead, use
// [WithBaseURL].
//
// Like all options, [WithEnvironment] and [WithBaseURL] are applied in the
// order passed to [NewClient]: whichever runs last wins the base URL. Pass
// [WithBaseURL] after [WithEnvironment] to override an environment's
// default URL (the common case — an arbitrary override); the reverse
// order lets [WithEnvironment] override an earlier [WithBaseURL].
func WithEnvironment(env Environment) Option {
	return optionFunc(func(c *Config) {
		c.environment = env
		if url, ok := baseURLs[env]; ok {
			c.baseURL = url
		}
	})
}

// WithBaseURL sets a custom base URL for the API, such as a proxy or a
// mock server in tests. The URL must be a valid http or https URL with a
// host, or [NewClient] returns a [ValidationError].
//
// Like all options, [WithBaseURL] and [WithEnvironment] are applied in the
// order passed to [NewClient]: whichever runs last wins. Pass [WithBaseURL]
// after [WithEnvironment] — the common case — to override an environment's
// default URL with an arbitrary one.
func WithBaseURL(url string) Option {
	return optionFunc(func(c *Config) {
		c.baseURL = url
	})
}

// WithAPIVersion sets the API version segment used in request paths
// (default "v1"). It overrides the MARKETDATA_API_VERSION environment
// variable, completing the configuration cascade for this setting. An
// empty value is ignored.
func WithAPIVersion(version string) Option {
	return optionFunc(func(c *Config) {
		if version != "" {
			c.apiVersion = version
		}
	})
}

// WithHTTPClient supplies a custom *http.Client for the SDK to send
// requests through, which is useful for custom transports, proxies, or
// instrumentation. The SDK operates on its own shallow copy of the
// supplied client — the caller's object is never modified — sharing its
// Transport. On that copy the SDK's fixed 99-second request timeout and
// secure redirect policy apply; dial-level settings (such as the SDK's
// default 2-second connect timeout) are the supplied Transport's own
// responsibility.
func WithHTTPClient(client *http.Client) Option {
	return optionFunc(func(c *Config) {
		c.httpClient = client
	})
}

// WithLogger sets the *slog.Logger the SDK logs through, replacing
// slog.Default(). The SDK never logs the full API token: in debug output
// the token is redacted to its last four characters.
func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(c *Config) {
		c.logger = logger
		// The injected handler owns level filtering from here on.
		c.logLevel = nil
	})
}

// WithoutStartupValidation skips the synchronous token validation call
// that [NewClient] normally makes during client creation. This saves one
// round trip when startup latency matters, but an invalid token then goes
// undetected until the first API call fails with an
// [AuthenticationError].
func WithoutStartupValidation() Option {
	return optionFunc(func(c *Config) {
		c.skipValidation = true
	})
}

// WithoutDotEnv disables loading a .env file from the working directory
// (ADR-012), for controlled environments — tests, containers, CI — where
// implicit file-based configuration is unwanted. Real environment
// variables and the other With* options are unaffected. The SDK never
// modifies the process environment either way; .env values only feed its
// own configuration cascade.
func WithoutDotEnv() Option {
	return optionFunc(func(c *Config) {
		c.noDotEnv = true
	})
}

// WithDebug enables debug logging of requests and responses on the
// configured logger. Setting the MARKETDATA_LOGGING_LEVEL environment
// variable to DEBUG has the same effect. The API token never appears in
// debug output; it is redacted to its last four characters.
func WithDebug(enabled bool) Option {
	return optionFunc(func(c *Config) {
		c.debug = enabled
	})
}

// setDefaultParam records a universal default query parameter, initializing
// the map on first use. These defaults are sent on every request unless a
// method-level parameter overrides them.
func (c *Config) setDefaultParam(key, value string) {
	if c.defaultParams == nil {
		c.defaultParams = url.Values{}
	}
	c.defaultParams.Set(key, value)
}

func (c *Config) setFormatOnlyParam(key, value string) {
	if c.formatOnlyParams == nil {
		c.formatOnlyParams = url.Values{}
	}
	c.formatOnlyParams.Set(key, value)
}

// WithColumns restricts responses to the named columns (the API's universal
// "columns" parameter), which can reduce payload size. Missing columns simply
// decode as zero values, so filtering never causes a decode error. Passing no
// columns is a no-op.
//
// On the typed JSON methods the SDK also requests the two kinds of column a
// decode cannot do without and a caller does not think to name: the "s"
// envelope, which every typed method checks, and the array each response
// keys its row count off (symbol, t, optionSymbol, date, depending on the
// endpoint). The API drops both whenever a column filter is set, and
// without them a present row decodes to nothing and the method reports
// not-found for data that is there. The formatted facets get exactly the
// columns asked for, since there the column list is the output.
func WithColumns(columns ...string) Option {
	return optionFunc(func(c *Config) {
		if len(columns) == 0 {
			return
		}
		c.setDefaultParam("columns", strings.Join(columns, ","))
	})
}

// WithDateFormat sets the API's universal "dateformat" parameter for every
// request ("timestamp", "unix", or "spreadsheet").
//
// Advanced use only. The SDK decodes dates from the API's default numeric
// (unix) representation; overriding the format globally can change the wire
// representation of date fields and cause typed responses (candles, quotes,
// earnings) to fail decoding. Endpoints that require a specific format for
// correct decoding set it themselves at the method level, which takes
// precedence over this default.
func WithDateFormat(format string) Option {
	return optionFunc(func(c *Config) {
		if format == "" {
			return
		}
		c.setDefaultParam("dateformat", format)
	})
}

// WithHumanReadable sets the API's universal "human" parameter, which requests
// human-readable field names and values.
//
// It applies to the CSV and HTML facets only ([stocks.Service.AsCSV] and its
// siblings) and is never sent on a typed JSON request. This is not a policy
// choice: the parameter renames every key in the response — "askSize" becomes
// "Ask Size", "changepct" becomes "Change %", and the "s" envelope field
// disappears — so a typed method receiving it fails outright. Unlike
// [WithColumns], which the SDK can repair by also requesting the envelope,
// there is nothing to salvage here.
func WithHumanReadable(enabled bool) Option {
	return optionFunc(func(c *Config) {
		c.setFormatOnlyParam("human", strconv.FormatBool(enabled))
	})
}

// WithAddHeaders sets the API's universal "headers" parameter, which controls
// whether a header row is included in CSV output. It has no effect on the JSON
// responses the SDK decodes and is provided for completeness.
func WithAddHeaders(enabled bool) Option {
	return optionFunc(func(c *Config) {
		c.setDefaultParam("headers", strconv.FormatBool(enabled))
	})
}

// Mode selects how the API fulfills every request, trading data freshness
// against credit cost (the API's universal "mode" parameter). It is a premium
// parameter: free and trial plans always receive delayed data.
type Mode string

const (
	// ModeLive returns real-time data. It is the default for paid plans.
	ModeLive Mode = "live"
	// ModeCached returns recently cached data at reduced credit cost. On a
	// cache miss the API returns HTTP 204, which the SDK surfaces as a no-data
	// response (nil result, Response.NoData true, nil error).
	ModeCached Mode = "cached"
	// ModeDelayed returns data delayed at least 15 minutes. It is the default
	// for free and trial plans.
	ModeDelayed Mode = "delayed"
)

// WithMode sets the API's universal "mode" parameter ([ModeLive], [ModeCached],
// or [ModeDelayed]) for every request. Because it is a client-level default,
// callers that need different modes per request should use separate clients
// (for example a cached client for bulk quotes and a live client for
// time-sensitive calls). See docs/RESIDUALS.md.
func WithMode(mode Mode) Option {
	return optionFunc(func(c *Config) {
		if mode == "" {
			return
		}
		c.setDefaultParam("mode", string(mode))
	})
}

// WithMaxAge sets the API's universal "maxage" parameter, the maximum age of
// cached data accepted when [ModeCached] is in effect. It accepts an absolute
// datetime or a relative duration such as "5min" or "1h"; if no cached data is
// within the window, the API returns a no-data response at no credit cost. It
// has no effect unless the mode is cached.
func WithMaxAge(maxAge string) Option {
	return optionFunc(func(c *Config) {
		if maxAge == "" {
			return
		}
		c.setDefaultParam("maxage", maxAge)
	})
}

// WithLimit sets the API's universal "limit" parameter, capping the number of
// results (overriding an endpoint's default). Values less than or equal to zero
// are ignored.
//
// "Universal" describes the parameter, not its reach: it is honored per
// endpoint. options/chain, options/expirations, stocks/news and
// markets/status apply it; the candles endpoints ignore it (verified live
// 2026-08-20 — a 13-candle range requested with limit=3 still returned 13).
// Tracked in integration/discrepancy_test.go.
func WithLimit(n int) Option {
	return optionFunc(func(c *Config) {
		if n <= 0 {
			return
		}
		c.setDefaultParam("limit", strconv.Itoa(n))
	})
}

// WithOffset sets the API's universal "offset" parameter for pagination, used
// together with [WithLimit]. Values less than or equal to zero are ignored
// (offset zero is the default first page).
//
// Like [WithLimit] it is ignored by the candles endpoints, which makes paging
// over candles unsafe rather than merely ineffective: every page returns the
// identical full set instead of advancing, so a loop either duplicates every
// row indefinitely or stops on a condition that was never true. Tracked in
// integration/discrepancy_test.go.
func WithOffset(n int) Option {
	return optionFunc(func(c *Config) {
		if n <= 0 {
			return
		}
		c.setDefaultParam("offset", strconv.Itoa(n))
	})
}

// WithMaxRetries sets the maximum number of retry attempts for failed
// requests, replacing the default of 3. Set it to 0 to disable retries
// entirely; negative values are treated as 0.
//
// Only the retry count is configurable. The retry conditions and backoff
// are fixed: a request is retried only on 501-599 status codes and
// transient network errors (never on 4xx or 500), with exponential backoff
// starting at 1 second and doubling each attempt, so the default schedule
// waits 1s, 2s, and 4s. A server-supplied Retry-After header takes
// precedence over the calculated backoff, and retries stop early if the
// API status endpoint reports the service offline.
func WithMaxRetries(n int) Option {
	return optionFunc(func(c *Config) {
		if n < 0 {
			n = 0
		}
		c.maxRetries = &n
	})
}
