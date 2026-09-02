package marketdata

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

// syncBuffer is a concurrency-safe writer for capturing log output from
// handlers that may be written to from multiple goroutines.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestNewClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "9999")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	client, err := NewClient(
		WithToken("test-key"),
		WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.Stocks == nil {
		t.Error("Stocks service is nil")
	}
	if client.Options == nil {
		t.Error("Options service is nil")
	}
	if client.Funds == nil {
		t.Error("Funds service is nil")
	}
	if client.Markets == nil {
		t.Error("Markets service is nil")
	}
	if client.Utilities == nil {
		t.Error("Utilities service is nil")
	}
}

func TestNewClient_NoToken_DemoMode(t *testing.T) {
	// Clear any existing environment variable
	oldToken := os.Getenv("MARKETDATA_TOKEN")
	_ = os.Unsetenv("MARKETDATA_TOKEN")
	defer func() {
		if oldToken != "" {
			_ = os.Setenv("MARKETDATA_TOKEN", oldToken)
		}
	}()

	// No token should succeed in demo mode (not return an error)
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() should succeed in demo mode, got error: %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil in demo mode")
	}
	if !client.config.demoMode {
		t.Error("client should be in demo mode when no token provided")
	}
}

// TestNewClient_DemoMode_EndToEnd exercises the demo path through the public
// constructor: NewClient must wire cfg.demoMode into the internal HTTP client
// so demo requests carry no Authorization header and skip the rate-limit
// pre-flight. The internal client's demo behavior is tested in internal/http;
// this guards the wiring itself.
func TestNewClient_DemoMode_EndToEnd(t *testing.T) {
	oldToken := os.Getenv("MARKETDATA_TOKEN")
	_ = os.Unsetenv("MARKETDATA_TOKEN")
	defer func() {
		if oldToken != "" {
			_ = os.Setenv("MARKETDATA_TOKEN", oldToken)
		}
	}()

	quotePayload := map[string]any{
		"s":       "ok",
		"symbol":  []string{"AAPL"},
		"ask":     []float64{150.25},
		"askSize": []int{100},
		"bid":     []float64{150.20},
		"bidSize": []int{200},
		"mid":     []float64{150.225},
		"last":    []float64{150.22},
		"volume":  []int64{50000000},
		"updated": []int64{1704067200},
	}

	t.Run("demo client sends no Authorization header", func(t *testing.T) {
		var requests atomic.Int64
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests.Add(1)
			if _, ok := r.Header["Authorization"]; ok {
				t.Errorf("demo request to %s carried Authorization header %q; want none", r.URL.Path, r.Header.Get("Authorization"))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(quotePayload)
		}))
		defer server.Close()

		client, err := NewClient(WithBaseURL(server.URL))
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		if !client.DemoMode() {
			t.Fatal("client should be in demo mode when no token provided")
		}

		ctx := context.Background()
		// Two consecutive calls: the second one exercises the demo
		// exemption from the rate-limit pre-flight via the public path.
		for i := 1; i <= 2; i++ {
			if _, _, err := client.Stocks.Quote(ctx, "AAPL"); err != nil {
				t.Fatalf("Quote() call %d error = %v", i, err)
			}
		}
		if got := requests.Load(); got != 2 {
			t.Errorf("server received %d requests, want 2", got)
		}
	})

	t.Run("token client sends Authorization header", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "Bearer e2e-token" {
				t.Errorf("Authorization = %q, want %q", got, "Bearer e2e-token")
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(quotePayload)
		}))
		defer server.Close()

		client, err := NewClient(WithToken("e2e-token"), WithBaseURL(server.URL))
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}
		if client.DemoMode() {
			t.Fatal("client with token must not be in demo mode")
		}
		if _, _, err := client.Stocks.Quote(context.Background(), "AAPL"); err != nil {
			t.Fatalf("Quote() error = %v", err)
		}
	})
}

func TestNewClient_WithEnvironmentToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	// Save and set environment variable
	oldToken := os.Getenv("MARKETDATA_TOKEN")
	_ = os.Setenv("MARKETDATA_TOKEN", "env-test-key")
	defer func() {
		if oldToken != "" {
			_ = os.Setenv("MARKETDATA_TOKEN", oldToken)
		} else {
			_ = os.Unsetenv("MARKETDATA_TOKEN")
		}
	}()

	client, err := NewClient(WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
}

func TestNewClient_AllOptions(t *testing.T) {
	var hits atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	httpClient := &http.Client{Timeout: 60 * time.Second}

	// WithEnvironment before WithBaseURL: options apply in the order given
	// (standard functional-options semantics — see WithBaseURL's godoc), so
	// the mock URL set by WithBaseURL must win. The reverse order is its
	// own regression test below.
	client, err := NewClient(
		WithToken("test-key"),
		WithEnvironment(Test),
		WithBaseURL(server.URL),
		WithHTTPClient(httpClient),
		WithLogger(logger),
		WithDebug(true),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	// A real request must land on the mock, proving the client actually
	// uses the constructed configuration — not just that construction
	// returned no error (NewClient treats non-auth startup validation
	// failures as warnings, so err == nil alone proves very little).
	if _, _, err := client.Utilities.Status(context.Background()); err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if hits.Load() == 0 {
		t.Error("request never reached the mock server — WithBaseURL did not take effect")
	}
}

// TestNewClient_LaterOptionWins is a regression test for a real
// documentation bug found in a mutation-testing sweep: WithBaseURL's
// godoc claimed it "overrides ... any URL chosen by WithEnvironment"
// unconditionally, but options are plain sequential field setters — the
// LAST one applied wins, regardless of which function it came from.
// TestNewClient_AllOptions previously passed WithBaseURL(mock) before
// WithEnvironment(Test), so WithEnvironment silently won and the mock was
// never actually exercised — the test still passed because NewClient
// swallows non-auth startup-validation failures, so it never contacted
// the mock server it thought it was testing against, and no assertion
// caught the discrepancy since only err == nil was checked.
func TestNewClient_LaterOptionWins(t *testing.T) {
	client, err := NewClient(
		WithToken("test-key"),
		WithBaseURL("https://example.test"),
		WithEnvironment(Test), // applied after WithBaseURL: this must win
		WithoutStartupValidation(),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if got, want := client.config.baseURL, baseURLs[Test]; got != want {
		t.Errorf("baseURL = %q, want %q (the later-applied WithEnvironment)", got, want)
	}
}

func TestClient_RateLimits(t *testing.T) {
	// Deterministic, no sleeps: a foreground request updates the tracker
	// synchronously before RateLimits() is read. The background init hits
	// the same mock with the same headers, so it cannot change the outcome.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "9500")
		w.Header().Set("X-Api-Ratelimit-Consumed", "500")
		w.Header().Set("X-Api-Ratelimit-Reset", "1899999999")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	client, err := NewClient(
		WithToken("test-key"),
		WithBaseURL(server.URL),
		WithoutStartupValidation(),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if _, _, err := client.Utilities.Status(context.Background()); err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	limits := client.RateLimits()
	if limits.Limit != 10000 {
		t.Errorf("Limit = %d, want 10000", limits.Limit)
	}
	if limits.Remaining != 9500 {
		t.Errorf("Remaining = %d, want 9500", limits.Remaining)
	}
	if limits.ResetAt.Unix() != 1899999999 {
		t.Errorf("ResetAt = %v, want unix 1899999999", limits.ResetAt)
	}
}

// debugTestClient builds a client against a quote-serving mock with a
// DEBUG-capable logger writing to out, without startup debug enabled.
func debugTestClient(t *testing.T, out *syncBuffer) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s": "ok", "symbol": []string{"AAPL"}, "ask": []float64{150.25},
			"askSize": []int{100}, "bid": []float64{150.20}, "bidSize": []int{200},
			"mid": []float64{150.225}, "last": []float64{150.22},
			"volume": []int64{50000000}, "updated": []int64{1704067200},
		})
	}))
	logger := slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client, err := NewClient(
		WithToken("test-key"),
		WithBaseURL(server.URL),
		WithLogger(logger),
		WithoutStartupValidation(),
	)
	if err != nil {
		server.Close()
		t.Fatalf("NewClient() error = %v", err)
	}
	return client, server
}

// TestClientDebug_RuntimeTogglesHTTPLogs pins the promise of Client.Debug:
// enabling it at runtime must make the internal HTTP client emit its
// per-request records, even on a client constructed without WithDebug.
func TestClientDebug_RuntimeTogglesHTTPLogs(t *testing.T) {
	out := &syncBuffer{}
	client, server := debugTestClient(t, out)
	defer server.Close()
	ctx := context.Background()

	quote := func() {
		if _, _, err := client.Stocks.Quote(ctx, "AAPL"); err != nil {
			t.Fatalf("Quote() error = %v", err)
		}
	}

	quote()
	if strings.Contains(out.String(), "sending request") {
		t.Fatal("HTTP debug records emitted before Debug(true)")
	}

	client.Debug(true)
	quote()
	if !strings.Contains(out.String(), "sending request") {
		t.Error("Debug(true) did not enable HTTP request logging at runtime")
	}

	client.Debug(false)
	before := strings.Count(out.String(), "sending request")
	quote()
	if after := strings.Count(out.String(), "sending request"); after != before {
		t.Errorf("Debug(false) did not disable HTTP request logging: %d records grew to %d", before, after)
	}
}

// TestClientDebug_ConcurrentToggle exercises Debug toggling while requests
// are in flight; the race detector guards the shared debug state.
func TestClientDebug_ConcurrentToggle(t *testing.T) {
	out := &syncBuffer{}
	client, server := debugTestClient(t, out)
	defer server.Close()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if i%2 == 0 {
					client.Debug(j%2 == 0)
				} else if _, _, err := client.Stocks.Quote(ctx, "AAPL"); err != nil {
					t.Errorf("Quote() error = %v", err)
				}
			}
		}(i)
	}
	wg.Wait()
}

// chdirTemp switches the CWD to a fresh temp dir and restores it on
// cleanup.
//
// WARNING (T-5): os.Chdir mutates process-global state. Any test using
// this helper — and any test using saveEnv below, which mutates the
// process environment the same way — must never call t.Parallel(),
// directly or via a parent that does, or it will race with every other
// test's view of the working directory / environment. None of the tests
// in this file are parallel today; if one ever needs to be, give it its
// own subprocess-level isolation instead of reusing these helpers.
func chdirTemp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	return dir
}

// saveEnv unsets a variable for the test and restores its prior state.
func saveEnv(t *testing.T, key string) {
	t.Helper()
	old, had := os.LookupEnv(key)
	_ = os.Unsetenv(key)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func TestNewClient_DotEnv_FeedsConfigWithoutTouchingProcessEnv(t *testing.T) {
	saveEnv(t, "MARKETDATA_TOKEN")
	saveEnv(t, "MARKETDATA_BASE_URL")
	saveEnv(t, "SENTINEL_FROM_DOTENV")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"s": "ok"})
	}))
	defer server.Close()

	dir := chdirTemp(t)
	env := "MARKETDATA_TOKEN=dotenv-token\nMARKETDATA_BASE_URL=" + server.URL + "\nSENTINEL_FROM_DOTENV=leaked\n"
	if err := os.WriteFile(dir+"/.env", []byte(env), 0644); err != nil {
		t.Fatal(err)
	}

	client, err := NewClient(WithoutStartupValidation())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// The .env fed the SDK's own cascade...
	if client.DemoMode() {
		t.Error("DemoMode() = true; the .env token should have been picked up")
	}
	if _, _, err := client.Utilities.Status(context.Background()); err != nil {
		t.Errorf("request through .env base URL failed: %v", err)
	}

	// ...without leaking a single variable into the process environment.
	for _, key := range []string{"SENTINEL_FROM_DOTENV", "MARKETDATA_TOKEN", "MARKETDATA_BASE_URL"} {
		if v, ok := os.LookupEnv(key); ok {
			t.Errorf("process env %s=%q was set by NewClient; the environment must never be mutated", key, v)
		}
	}
}

func TestNewClient_DotEnv_RealEnvWins(t *testing.T) {
	saveEnv(t, "MARKETDATA_TOKEN")
	saveEnv(t, "MARKETDATA_BASE_URL")

	var hitsA, hitsB atomic.Int64
	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitsA.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"s": "ok"})
	}))
	defer serverA.Close()
	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitsB.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"s": "ok"})
	}))
	defer serverB.Close()

	dir := chdirTemp(t)
	env := "MARKETDATA_TOKEN=dotenv-token\nMARKETDATA_BASE_URL=" + serverA.URL + "\n"
	if err := os.WriteFile(dir+"/.env", []byte(env), 0644); err != nil {
		t.Fatal(err)
	}
	_ = os.Setenv("MARKETDATA_BASE_URL", serverB.URL)

	client, err := NewClient(WithoutStartupValidation())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if _, _, err := client.Utilities.Status(context.Background()); err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if hitsB.Load() == 0 {
		t.Error("real env var MARKETDATA_BASE_URL did not win over the .env value")
	}
	if hitsA.Load() != 0 {
		t.Errorf("the .env base URL received %d requests despite a real env override", hitsA.Load())
	}
}

func TestNewClient_WithoutDotEnv(t *testing.T) {
	saveEnv(t, "MARKETDATA_TOKEN")

	dir := chdirTemp(t)
	if err := os.WriteFile(dir+"/.env", []byte("MARKETDATA_TOKEN=dotenv-token\n"), 0644); err != nil {
		t.Fatal(err)
	}

	client, err := NewClient(WithoutDotEnv())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if !client.DemoMode() {
		t.Error("DemoMode() = false; WithoutDotEnv must prevent the .env token from loading")
	}
}

func TestClientDemoMode(t *testing.T) {
	t.Setenv("MARKETDATA_TOKEN", "")
	c, err := NewClient(WithoutStartupValidation())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = c.Close() }()
	if !c.DemoMode() {
		t.Error("DemoMode() = false, want true without a token")
	}
}

func TestClientDemoModeWithToken(t *testing.T) {
	c, err := NewClient(WithToken("test-token"), WithoutStartupValidation())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = c.Close() }()
	if c.DemoMode() {
		t.Error("DemoMode() = true, want false with a token")
	}
}

func TestClient_Debug(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	client, err := NewClient(
		WithToken("test-key"),
		WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Should not panic
	client.Debug(true)
	client.Debug(false)
}

// Test config options
func TestWithEnvironment(t *testing.T) {
	tests := []struct {
		env     Environment
		wantURL string
	}{
		{Production, "https://api.marketdata.app"},
		{Test, "https://test.api.marketdata.app"},
		{Development, "http://localhost:8080"},
	}

	for _, tt := range tests {
		t.Run(string(tt.env), func(t *testing.T) {
			cfg := defaultConfig(nil)
			WithEnvironment(tt.env).apply(cfg)
			if cfg.baseURL != tt.wantURL {
				t.Errorf("baseURL = %q, want %q", cfg.baseURL, tt.wantURL)
			}
		})
	}
}

func TestWithBaseURL(t *testing.T) {
	cfg := defaultConfig(nil)
	customURL := "https://custom.api.example.com"
	WithBaseURL(customURL).apply(cfg)
	if cfg.baseURL != customURL {
		t.Errorf("baseURL = %q, want %q", cfg.baseURL, customURL)
	}
}

func TestWithHTTPClient(t *testing.T) {
	cfg := defaultConfig(nil)
	httpClient := &http.Client{Timeout: 120 * time.Second}
	WithHTTPClient(httpClient).apply(cfg)
	if cfg.httpClient != httpClient {
		t.Error("httpClient not set correctly")
	}
}

func TestWithLogger(t *testing.T) {
	cfg := defaultConfig(nil)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	WithLogger(logger).apply(cfg)
	if cfg.logger != logger {
		t.Error("logger not set correctly")
	}
}

func TestWithDebug(t *testing.T) {
	cfg := defaultConfig(nil)

	WithDebug(true).apply(cfg)
	if !cfg.debug {
		t.Error("debug should be true")
	}

	WithDebug(false).apply(cfg)
	if cfg.debug {
		t.Error("debug should be false")
	}
}

func TestWithMaxRetries(t *testing.T) {
	cfg := defaultConfig(nil)

	// Default: nil (uses defaultMaxRetries)
	if cfg.maxRetries != nil {
		t.Error("maxRetries should be nil by default")
	}

	// Set to 5
	WithMaxRetries(5).apply(cfg)
	if cfg.maxRetries == nil || *cfg.maxRetries != 5 {
		t.Errorf("maxRetries = %v, want 5", cfg.maxRetries)
	}

	// Set to 0 (disable retries)
	WithMaxRetries(0).apply(cfg)
	if cfg.maxRetries == nil || *cfg.maxRetries != 0 {
		t.Errorf("maxRetries = %v, want 0", cfg.maxRetries)
	}

	// Negative clamped to 0
	WithMaxRetries(-1).apply(cfg)
	if cfg.maxRetries == nil || *cfg.maxRetries != 0 {
		t.Errorf("maxRetries = %v, want 0 (clamped from -1)", cfg.maxRetries)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig(nil)

	if cfg.environment != Production {
		t.Errorf("environment = %v, want Production", cfg.environment)
	}
	if cfg.apiVersion != "v1" {
		t.Errorf("apiVersion = %q, want v1", cfg.apiVersion)
	}
	if cfg.debug {
		t.Error("debug should be false by default")
	}
}

func TestFixedConstants(t *testing.T) {
	if fixedTimeout != 99*time.Second {
		t.Errorf("fixedTimeout = %v, want 99s", fixedTimeout)
	}
	if fixedConnTimeout != 2*time.Second {
		t.Errorf("fixedConnTimeout = %v, want 2s", fixedConnTimeout)
	}
	if defaultMaxRetries != 3 {
		t.Errorf("defaultMaxRetries = %d, want 3", defaultMaxRetries)
	}
	if fixedInitialBackoff != 1*time.Second {
		t.Errorf("fixedInitialBackoff = %v, want 1s", fixedInitialBackoff)
	}
	if fixedBackoffMultiplier != 2.0 {
		t.Errorf("fixedBackoffMultiplier = %v, want 2.0", fixedBackoffMultiplier)
	}
}

func TestConfig_validate(t *testing.T) {
	// Valid config
	cfg := defaultConfig(nil)
	cfg.token = "test-key"
	if err := cfg.validate(); err != nil {
		t.Errorf("validate() should not return error for valid config: %v", err)
	}

	// No token — enters demo mode (not an error)
	cfg2 := defaultConfig(nil)
	cfg2.token = ""
	if err := cfg2.validate(); err != nil {
		t.Errorf("validate() should not return error for demo mode: %v", err)
	}
	if !cfg2.demoMode {
		t.Error("validate() should set demoMode when no token provided")
	}

	// Token with control characters (e.g. stray CR from a .env file) — error
	cfg3 := defaultConfig(nil)
	cfg3.token = "test-key\r"
	if err := cfg3.validate(); err == nil {
		t.Error("validate() should return error for token with control characters")
	}

	// Token with non-ASCII characters — error
	cfg4 := defaultConfig(nil)
	cfg4.token = "test-kéy"
	if err := cfg4.validate(); err == nil {
		t.Error("validate() should return error for non-ASCII token")
	}

	// Invalid base URL — error
	for _, bad := range []string{"", "not-a-url", "ftp://example.com", "https://"} {
		cfgB := defaultConfig(nil)
		cfgB.token = "test-key"
		cfgB.baseURL = bad
		if err := cfgB.validate(); err == nil {
			t.Errorf("validate() should return error for baseURL %q", bad)
		}
	}
}

// Test Environment constants
func TestEnvironmentConstants(t *testing.T) {
	if Production != "production" {
		t.Errorf("Production = %q, want production", Production)
	}
	if Test != "test" {
		t.Errorf("Test = %q, want test", Test)
	}
	if Development != "development" {
		t.Errorf("Development = %q, want development", Development)
	}
}

// Test unknown environment doesn't set URL
func TestWithEnvironment_Unknown(t *testing.T) {
	cfg := defaultConfig(nil)
	originalURL := cfg.baseURL
	WithEnvironment(Environment("unknown")).apply(cfg)
	if cfg.baseURL != originalURL {
		t.Errorf("baseURL should not change for unknown environment")
	}
}

func TestWithoutStartupValidation(t *testing.T) {
	cfg := defaultConfig(nil)
	WithoutStartupValidation().apply(cfg)
	if !cfg.skipValidation {
		t.Error("skipValidation should be true after WithoutStartupValidation()")
	}
}

func TestNewClient_WithoutStartupValidation(t *testing.T) {
	// Server that would fail validation (returns 401)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "error", "errmsg": "Unauthorized"})
	}))
	defer server.Close()

	// With skipValidation, client creation should succeed even with bad token
	client, err := NewClient(
		WithToken("bad-token"),
		WithBaseURL(server.URL),
		WithoutStartupValidation(),
	)
	if err != nil {
		t.Fatalf("NewClient() with WithoutStartupValidation() should not error, got: %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
}

func TestNewClient_InvalidToken(t *testing.T) {
	// Server that returns 401 for validation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "error", "errmsg": "Unauthorized"})
	}))
	defer server.Close()

	// Without skipValidation, client creation should fail with invalid token
	_, err := NewClient(
		WithToken("bad-token"),
		WithBaseURL(server.URL),
	)
	if err == nil {
		t.Fatal("NewClient() should return error for invalid token")
	}
	if !strings.Contains(err.Error(), "token validation failed") {
		t.Errorf("error = %q, want to contain 'token validation failed'", err.Error())
	}
}

func TestUniversalParamsFromEnv(t *testing.T) {
	t.Run("reads all env vars", func(t *testing.T) {
		t.Setenv("MARKETDATA_DATE_FORMAT", "unix")
		t.Setenv("MARKETDATA_COLUMNS", "open,high,low,close")
		t.Setenv("MARKETDATA_ADD_HEADERS", "false")
		t.Setenv("MARKETDATA_USE_HUMAN_READABLE", "true")

		params, formatOnly := universalParamsFromEnv(nil)

		if got := params.Get("dateformat"); got != "unix" {
			t.Errorf("dateformat = %q, want %q", got, "unix")
		}
		if got := params.Get("columns"); got != "open,high,low,close" {
			t.Errorf("columns = %q, want %q", got, "open,high,low,close")
		}
		if got := params.Get("headers"); got != "false" {
			t.Errorf("headers = %q, want %q", got, "false")
		}
		// human lands in the format-only bucket, not the request defaults: it
		// renames every field in the response, so it must never reach a typed
		// JSON call (see WithHumanReadable).
		if got := params.Get("human"); got != "" {
			t.Errorf("human = %q in the request defaults, want it format-only", got)
		}
		if got := formatOnly.Get("human"); got != "true" {
			t.Errorf("format-only human = %q, want %q", got, "true")
		}
	})

	t.Run("skips unset vars", func(t *testing.T) {
		// Clear all relevant env vars
		t.Setenv("MARKETDATA_DATE_FORMAT", "")
		t.Setenv("MARKETDATA_COLUMNS", "")
		t.Setenv("MARKETDATA_ADD_HEADERS", "")
		t.Setenv("MARKETDATA_USE_HUMAN_READABLE", "")

		params, _ := universalParamsFromEnv(nil)
		if len(params) != 0 {
			t.Errorf("expected empty params, got %v", params)
		}
	})
}

func TestLogLevelFromEnv(t *testing.T) {
	tests := []struct {
		name   string
		envVal string
		want   slog.Level
		wantOK bool
	}{
		{"debug", "DEBUG", slog.LevelDebug, true},
		{"info", "INFO", slog.LevelInfo, true},
		{"warning", "WARNING", slog.LevelWarn, true},
		{"warn alias", "WARN", slog.LevelWarn, true},
		{"error", "ERROR", slog.LevelError, true},
		{"lowercase", "debug", slog.LevelDebug, true},
		{"mixed case", "Debug", slog.LevelDebug, true},
		{"empty", "", slog.LevelInfo, false},
		{"invalid", "TRACE", slog.LevelInfo, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MARKETDATA_LOGGING_LEVEL", tt.envVal)
			got, ok := logLevelFromEnv(nil)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("logLevelFromEnv(nil) = (%v, %v), want (%v, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestRedactToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{"normal token", "abcdefghijklmnopYKT0", "****************YKT0"},
		{"short token", "ab", "****"},
		{"exactly 4 chars", "abcd", "****"},
		{"5 chars", "abcde", "*bcde"},
		{"empty", "", "****"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactToken(tt.token)
			if got != tt.want {
				t.Errorf("redactToken(%q) = %q, want %q", tt.token, got, tt.want)
			}
		})
	}
}

// Test ValidationError
func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Field:   "token",
		Message: "token is required",
	}
	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() returned empty string")
	}
}

func TestClient_Close(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	client, err := NewClient(
		WithToken("test-key"),
		WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// First close should succeed
	if err := client.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Second close should be a no-op (idempotent)
	if err := client.Close(); err != nil {
		t.Errorf("Close() second call error = %v", err)
	}
}

func TestWithAPIKey(t *testing.T) {
	cfg := defaultConfig(nil)
	WithAPIKey("test-api-key").apply(cfg)
	if cfg.token != "test-api-key" {
		t.Errorf("token = %q, want %q", cfg.token, "test-api-key")
	}
}

func TestDefaultConfig_WithDebugLoggingLevel(t *testing.T) {
	t.Setenv("MARKETDATA_LOGGING_LEVEL", "DEBUG")

	cfg := defaultConfig(nil)
	if !cfg.debug {
		t.Error("debug should be true when MARKETDATA_LOGGING_LEVEL=DEBUG")
	}
}

func TestDefaultConfig_WithNonDebugLoggingLevel(t *testing.T) {
	t.Setenv("MARKETDATA_LOGGING_LEVEL", "INFO")

	cfg := defaultConfig(nil)
	if cfg.debug {
		t.Error("debug should be false when MARKETDATA_LOGGING_LEVEL=INFO")
	}
}

func TestNewClient_ValidationNonAuthError(t *testing.T) {
	// Server that returns a 500 during validation (non-auth error = warning, not failure)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "error", "errmsg": "Server Error"})
	}))
	defer server.Close()

	// Non-auth errors during validation should NOT fail client creation
	client, err := NewClient(
		WithToken("test-key"),
		WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("NewClient() should succeed for non-auth validation errors, got: %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
}

func TestNewClient_WithDebugAndToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	// Test that debug mode with token logs the redacted token
	client, err := NewClient(
		WithToken("test-key-12345"),
		WithBaseURL(server.URL),
		WithDebug(true),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
}

func TestGetTokenFromEnv(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		t.Setenv("MARKETDATA_TOKEN", "env-token")
		got := getTokenFromEnv(nil)
		if got != "env-token" {
			t.Errorf("getTokenFromEnv(nil) = %q, want %q", got, "env-token")
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Setenv("MARKETDATA_TOKEN", "")
		got := getTokenFromEnv(nil)
		if got != "" {
			t.Errorf("getTokenFromEnv(nil) = %q, want empty", got)
		}
	})
}

func TestEnvSource_Get_ExplicitEmptyWinsOverDotEnv(t *testing.T) {
	// A real environment variable explicitly set to "" (the CI pattern for
	// forcing demo mode despite a .env file with a token) must win over the
	// .env fallback, not be treated as "unset".
	saveEnv(t, "MARKETDATA_TOKEN")
	t.Setenv("MARKETDATA_TOKEN", "")
	src := envSource{"MARKETDATA_TOKEN": "dotenv-token"}
	if got := src.get("MARKETDATA_TOKEN"); got != "" {
		t.Errorf("get(%q) = %q, want empty string (explicit env override), not the .env fallback", "MARKETDATA_TOKEN", got)
	}
}

func TestEnvSource_Get_UnsetFallsBackToDotEnv(t *testing.T) {
	saveEnv(t, "MARKETDATA_TOKEN")
	src := envSource{"MARKETDATA_TOKEN": "dotenv-token"}
	if got := src.get("MARKETDATA_TOKEN"); got != "dotenv-token" {
		t.Errorf("get(%q) = %q, want the .env fallback %q when the real env var is unset", "MARKETDATA_TOKEN", got, "dotenv-token")
	}
}

func TestNewClient_InvalidTokenCharset(t *testing.T) {
	_, err := NewClient(WithToken("bad\x01token"))
	if err == nil {
		t.Fatal("NewClient() with control characters in token should return validation error")
	}
}

func TestNewClient_MaxRetriesAndStatusFetcher(t *testing.T) {
	var statusCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/status"):
			statusCalls.Add(1)
			_, _ = w.Write([]byte(`{"s":"ok","status":["online"]}`))
		case strings.HasPrefix(r.URL.Path, "/user"):
			_, _ = w.Write([]byte(`{"s":"ok"}`))
		default:
			w.WriteHeader(http.StatusServiceUnavailable) // 503 - retryable
		}
	}))
	defer server.Close()

	client, err := NewClient(
		WithToken("test-token"),
		WithBaseURL(server.URL),
		WithoutStartupValidation(),
		WithMaxRetries(1),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	// A retryable 503 makes the retry loop consult the API status cache,
	// which drives the real fetcher through GET /status/.
	_, _, err = client.Stocks.Quote(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("Quote() against an all-503 server should return an error")
	}
	if statusCalls.Load() == 0 {
		t.Error("status fetcher was never invoked during retry")
	}
}

// TestSpecLogPoints pins the requirements-§7 log points: INFO on client
// construction, ERROR on terminally failed requests, and a duration field
// on the DEBUG response record.
func TestSpecLogPoints(t *testing.T) {
	var status atomic.Int64
	status.Store(200)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := int(status.Load())
		w.Header().Set("Content-Type", "application/json")
		if code != 200 {
			w.WriteHeader(code)
			_, _ = w.Write([]byte(`{"s":"error","errmsg":"boom"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"s": "ok"})
	}))
	defer server.Close()

	out := &syncBuffer{}
	logger := slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client, err := NewClient(
		WithToken("test-key"),
		WithBaseURL(server.URL),
		WithLogger(logger),
		WithoutStartupValidation(),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if !strings.Contains(out.String(), "client initialized") {
		t.Error("missing INFO 'client initialized' log point")
	}
	if !strings.Contains(out.String(), "base_url="+server.URL) {
		t.Errorf("'client initialized' log point missing base_url=%s, got %q", server.URL, out.String())
	}
	if !strings.Contains(out.String(), "api_version=v1") {
		t.Errorf("'client initialized' log point missing api_version=v1, got %q", out.String())
	}

	client.Debug(true)
	if _, _, err := client.Utilities.Status(context.Background()); err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !strings.Contains(out.String(), "duration=") {
		t.Error("DEBUG 'received response' record is missing the duration field")
	}

	status.Store(400)
	if _, _, err := client.Utilities.Status(context.Background()); err == nil {
		t.Fatal("Status() should fail on 400")
	}
	if !strings.Contains(out.String(), "level=ERROR") || !strings.Contains(out.String(), "request failed") {
		t.Error("terminal request failure was not logged at ERROR")
	}
}

// TestInitRateLimits_DoesNotLogOnFailure pins the fix for the startup
// double-log: initRateLimits is best-effort background priming whose error
// is deliberately discarded (ADR-006 deviation — async and silent), so it
// must never emit the ERROR log that a foreground call to the same failing
// endpoint (the synchronous startup validation) already produces. NewClient
// itself also starts this same background goroutine; calling initRateLimits
// again here directly is what makes the assertion deterministic instead of
// racing that goroutine's completion — either way, no call to it should log.
func TestInitRateLimits_DoesNotLogOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"s":"error","errmsg":"boom"}`))
	}))
	defer server.Close()

	out := &syncBuffer{}
	logger := slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client, err := NewClient(
		WithToken("test-key"),
		WithBaseURL(server.URL),
		WithLogger(logger),
		WithoutStartupValidation(),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	before := out.String()
	client.initRateLimits()
	after := out.String()

	if after != before {
		t.Errorf("initRateLimits() logged on failure, want no additional output (best-effort background priming): %q", strings.TrimPrefix(after, before))
	}
}

func TestWithAPIVersion(t *testing.T) {
	var gotPath atomic.Value
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ignore the background /user/ init request; capture the quote call.
		if strings.Contains(r.URL.Path, "bulkquotes") {
			gotPath.Store(r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"s": "ok", "symbol": []string{"AAPL"}, "last": []float64{1}})
	}))
	defer server.Close()

	client, err := NewClient(
		WithToken("test-key"),
		WithBaseURL(server.URL),
		WithAPIVersion("v2"),
		WithoutStartupValidation(),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if _, _, err := client.Stocks.Quote(context.Background(), "AAPL"); err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if p, _ := gotPath.Load().(string); !strings.HasPrefix(p, "/v2/") {
		t.Errorf("request path = %q, want a /v2/ prefix", p)
	}
}

func TestWithAPIVersion_RejectsPathTraversal(t *testing.T) {
	for _, version := range []string{"../..", "v1/../../admin", "a/b", `a\b`} {
		_, err := NewClient(
			WithToken("test-key"),
			WithAPIVersion(version),
			WithoutStartupValidation(),
		)
		var valErr *sdkerrors.ValidationError
		if !errors.As(err, &valErr) || valErr.Field != "apiVersion" {
			t.Errorf("WithAPIVersion(%q): err = %v, want a ValidationError on field apiVersion", version, err)
		}
	}
}

func TestVersion_Public(t *testing.T) {
	if Version() == "" {
		t.Error("Version() returned an empty string")
	}
}

// TestClient_RateLimitMetadataIsRequestScopedUnderConcurrency closes the last
// unticked item of the requirements §13 unit-test checklist: "request-scoped
// rate limit metadata under concurrency".
//
// The failure it guards against is not hypothetical. Both sdk-py and sdk-php
// ship a RATE_LIMITS_CONCURRENCY_ISSUE.md documenting exactly this bug in
// production: the per-request x-api-ratelimit-* headers are stored on a
// client-level snapshot, so under concurrency every caller reads the last
// response's numbers instead of their own. Go avoids it by construction —
// response.New parses the headers of the response it was handed, and
// Client.RateLimits() is a separate convenience snapshot — but nothing until
// now pinned that, and a refactor pointing Response.RateLimit at the shared
// tracker would restore the bug with every existing test still green.
//
// The mock answers each symbol with a distinct set of headers. If the metadata
// were read from shared state, all responses would carry identical values and
// the per-symbol assertions below would fail.
func TestClient_RateLimitMetadataIsRequestScopedUnderConcurrency(t *testing.T) {
	const workers = 40

	// remainingFor derives a unique, recognizable credit count per symbol so a
	// mismatched response can be traced back to the request that produced it.
	remainingFor := func(i int) int { return 9000 + i }
	symbolFor := func(i int) string { return fmt.Sprintf("SYM%03d", i) }

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbols")
		var idx int
		if _, err := fmt.Sscanf(symbol, "SYM%03d", &idx); err != nil {
			// The background /user/ priming call and the status probe share
			// this handler; answer them without rate-limit headers so they
			// cannot be mistaken for one of the probes.
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", strconv.Itoa(remainingFor(idx)))
		w.Header().Set("X-Api-Ratelimit-Consumed", strconv.Itoa(idx))
		w.Header().Set("X-Api-Ratelimit-Reset", "1899999999")
		// Force the responses to genuinely overlap rather than serialize.
		time.Sleep(2 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s": "ok", "symbol": []string{symbol}, "last": []float64{1.0},
			"updated": []int64{1704067200},
		})
	}))
	defer server.Close()

	client, err := NewClient(
		WithToken("test-key"),
		WithBaseURL(server.URL),
		WithoutStartupValidation(),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	type observation struct {
		symbol    string
		remaining int
		consumed  int
	}
	results := make([]observation, workers)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, resp, err := client.Stocks.Quote(context.Background(), symbolFor(i))
			if err != nil {
				t.Errorf("Quote(%s) error = %v", symbolFor(i), err)
				return
			}
			results[i] = observation{
				symbol:    symbolFor(i),
				remaining: resp.RateLimit.Remaining,
				consumed:  resp.RateLimit.Consumed,
			}
		}(i)
	}
	wg.Wait()

	// Every response must carry the numbers from its OWN request.
	for i, got := range results {
		want := remainingFor(i)
		if got.remaining != want {
			t.Errorf("%s: RateLimit.Remaining = %d, want %d — metadata is not request-scoped "+
				"(it looks like shared client state won)", symbolFor(i), got.remaining, want)
		}
		if got.consumed != i {
			t.Errorf("%s: RateLimit.Consumed = %d, want %d", symbolFor(i), got.consumed, i)
		}
	}

	// A last-write-wins bug would make every response identical. Assert the
	// observed values are actually distinct, so the check above cannot pass by
	// coincidence if remainingFor ever became a constant.
	distinct := make(map[int]struct{}, workers)
	for _, got := range results {
		distinct[got.remaining] = struct{}{}
	}
	if len(distinct) != workers {
		t.Errorf("saw %d distinct Remaining values across %d concurrent responses, want %d",
			len(distinct), workers, workers)
	}
}
