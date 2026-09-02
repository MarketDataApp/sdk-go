package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/MarketDataApp/sdk-go/v2/internal/fanout"
	"github.com/MarketDataApp/sdk-go/v2/internal/ratelimit"
	"github.com/MarketDataApp/sdk-go/v2/internal/retry"
	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
	"github.com/MarketDataApp/sdk-go/v2/internal/status"
)

func TestVersion_ReturnsNonEmpty(t *testing.T) {
	v := Version()
	if v == "" {
		t.Error("Version() returned empty string")
	}
}

func TestVersion_ConsistentAcrossCalls(t *testing.T) {
	v1 := Version()
	v2 := Version()
	if v1 != v2 {
		t.Errorf("Version() not consistent: %q vs %q", v1, v2)
	}
}

func TestDetectVersion(t *testing.T) {
	// detectVersion should return a string (likely "unknown" in test context
	// since the module is not consumed as a dependency)
	v := detectVersion()
	if v == "" {
		t.Error("detectVersion() returned empty string")
	}
}

func TestUserAgent_ContainsVersion(t *testing.T) {
	var receivedUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})

	_, _ = client.Get(context.Background(), "/test/", nil, nil)

	expected := "marketdata-sdk-go/" + Version()
	if receivedUA != expected {
		t.Errorf("User-Agent = %q, want %q", receivedUA, expected)
	}
}

func TestNew(t *testing.T) {
	cfg := Config{
		BaseURL:     "https://api.example.com/",
		APIVersion:  "v1",
		Token:       "test-key",
		Timeout:     99 * time.Second,
		ConnTimeout: 2 * time.Second,
		RetryCfg:    retry.DefaultConfig(),
		RateLimits:  ratelimit.New(),
	}

	client := New(cfg)

	if client == nil {
		t.Fatal("New() returned nil")
	}
	if client.baseURL != "https://api.example.com" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "https://api.example.com")
	}
	if client.apiVersion != "v1" {
		t.Errorf("apiVersion = %q, want %q", client.apiVersion, "v1")
	}
	if client.token != "test-key" {
		t.Errorf("apiKey = %q, want %q", client.token, "test-key")
	}
}

func TestNew_DefaultHTTPClient(t *testing.T) {
	cfg := Config{
		BaseURL:     "https://api.example.com",
		APIVersion:  "v1",
		Token:       "test-key",
		Timeout:     99 * time.Second,
		ConnTimeout: 2 * time.Second,
	}

	client := New(cfg)

	if client.http == nil {
		t.Error("http client should not be nil")
	}
}

func TestClient_Get_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Method = %q, want GET", r.Method)
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("Authorization header missing")
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("User-Agent header missing")
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "9999")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok", "message": "success"})
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})

	var result struct {
		Status  string `json:"s"`
		Message string `json:"message"`
	}

	_, err := client.Get(context.Background(), "/test/", nil, &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if result.Status != "ok" {
		t.Errorf("Status = %q, want %q", result.Status, "ok")
	}
	if result.Message != "success" {
		t.Errorf("Message = %q, want %q", result.Message, "success")
	}
}

func TestClient_GetFormatted_CSV_Success(t *testing.T) {
	const csvBody = "symbol,last\nAAPL,150.22\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("format"); got != "csv" {
			t.Errorf("format param = %q, want %q", got, "csv")
		}
		if got := r.Header.Get("Accept"); got != "text/csv" {
			t.Errorf("Accept header = %q, want %q", got, "text/csv")
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		_, _ = w.Write([]byte(csvBody))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "test-key",
		RetryCfg: retry.DefaultConfig(),
	})

	resp, err := client.GetFormatted(context.Background(), "stocks/quotes/AAPL/", nil, "csv")
	if err != nil {
		t.Fatalf("GetFormatted() error = %v", err)
	}
	if string(resp.Body) != csvBody {
		t.Errorf("Body = %q, want %q", resp.Body, csvBody)
	}
}

func TestClient_GetFormatted_HTML_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("format"); got != "html" {
			t.Errorf("format param = %q, want %q", got, "html")
		}
		if got := r.Header.Get("Accept"); got != "text/html" {
			t.Errorf("Accept header = %q, want %q", got, "text/html")
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html></html>"))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "test-key",
		RetryCfg: retry.DefaultConfig(),
	})

	resp, err := client.GetFormatted(context.Background(), "stocks/quotes/AAPL/", nil, "html")
	if err != nil {
		t.Fatalf("GetFormatted() error = %v", err)
	}
	if string(resp.Body) != "<html></html>" {
		t.Errorf("Body = %q, want %q", resp.Body, "<html></html>")
	}
}

func TestClient_GetFormatted_ErrorLogsAndReturnsClassifiedError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(400)
		_, _ = w.Write([]byte("s,errmsg\nerror,\"bad request\"\n"))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "test-key",
		RetryCfg: retry.Config{MaxRetries: 0, InitialBackoff: time.Millisecond, Multiplier: 2},
		Logger:   logger,
	})

	_, err := client.GetFormatted(context.Background(), "stocks/quotes/AAPL/", nil, "csv")
	badReq, ok := err.(*sdkerrors.BadRequestError)
	if !ok {
		t.Fatalf("error type = %T, want *sdkerrors.BadRequestError", err)
	}
	if badReq.Message != "bad request" {
		t.Errorf("Message = %q, want the CSV errmsg column", badReq.Message)
	}
	if !containsString(buf.String(), "level=ERROR") {
		t.Errorf("terminal failure was not logged at ERROR: %q", buf.String())
	}
}

func TestMediaTypeForFormat(t *testing.T) {
	cases := map[string]string{"csv": "text/csv", "html": "text/html", "bogus": "application/json"}
	for format, want := range cases {
		if got := mediaTypeForFormat(format); got != want {
			t.Errorf("mediaTypeForFormat(%q) = %q, want %q", format, got, want)
		}
	}
}

func TestClient_GetFormatted_NetworkError(t *testing.T) {
	client := New(Config{
		BaseURL: "http://127.0.0.1:1", APIVersion: "v1", Token: "test-key",
		RetryCfg: retry.Config{MaxRetries: 0, InitialBackoff: time.Millisecond, Multiplier: 2},
	})

	_, err := client.GetFormatted(context.Background(), "stocks/quotes/AAPL/", nil, "csv")
	var netErr *sdkerrors.NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("error type = %T, want *sdkerrors.NetworkError", err)
	}
}

func TestClient_GetFormatted_404IsNotAnError(t *testing.T) {
	// Verified live: a genuine no-data condition on a format=csv request can
	// come back 200 with a degenerate body rather than 404 — but when the
	// API *does* send a 404 for this format, it must not be treated as a
	// failure either (Get's own NoData carve-out treats 404 as success, and
	// GetFormatted must not regress to erroring where Get wouldn't).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"s":"no_data"}`))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "test-key",
		RetryCfg: retry.DefaultConfig(),
	})

	resp, err := client.GetFormatted(context.Background(), "stocks/candles/D/AAPL/", nil, "csv")
	if err != nil {
		t.Fatalf("GetFormatted() error = %v, want nil (404 is not an error)", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", resp.StatusCode)
	}
}

func TestClient_Get_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "test-request-123")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "error", "errmsg": "Invalid symbol"})
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})

	var result map[string]string
	_, err := client.Get(context.Background(), "/test/", nil, &result)

	if err == nil {
		t.Fatal("Get() should return error for 400 status")
	}

	apiErr, ok := err.(*sdkerrors.BadRequestError)
	if !ok {
		t.Fatalf("error type = %T, want *sdkerrors.BadRequestError", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusBadRequest)
	}
	if apiErr.Message != "Invalid symbol" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "Invalid symbol")
	}
	if apiErr.RequestID != "test-request-123" {
		t.Errorf("RequestID = %q, want %q", apiErr.RequestID, "test-request-123")
	}
}

func TestClient_Get_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable) // 503 - retryable
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg: retry.Config{
			MaxRetries:     3,
			InitialBackoff: 1 * time.Millisecond,
			Multiplier:     2.0,
		},
		RateLimits: ratelimit.New(),
	})

	var result map[string]string
	_, err := client.Get(context.Background(), "/test/", nil, &result)

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestClient_Get_500_NotRetried(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError) // 500 - NOT retried
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "error", "errmsg": "Internal Server Error"})
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg: retry.Config{
			MaxRetries:     3,
			InitialBackoff: 1 * time.Millisecond,
			Multiplier:     2.0,
		},
		RateLimits: ratelimit.New(),
	})

	_, err := client.Get(context.Background(), "/test/", nil, nil)

	if err == nil {
		t.Fatal("Get() should return error for 500")
	}
	// 500 should NOT be retried — only 1 attempt
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (500 should not be retried)", attempts)
	}
}

func TestClient_Get_429_NotRetried(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests) // 429 - NOT retried
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "error", "errmsg": "Rate limit exceeded"})
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg: retry.Config{
			MaxRetries:     3,
			InitialBackoff: 1 * time.Millisecond,
			Multiplier:     2.0,
		},
		RateLimits: ratelimit.New(),
	})

	_, err := client.Get(context.Background(), "/test/", nil, nil)

	if err == nil {
		t.Fatal("Get() should return error for 429")
	}
	// 429 should NOT be retried — only 1 attempt
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (429 should not be retried)", attempts)
	}
}

func TestClient_RateLimitExceeded(t *testing.T) {
	tracker := ratelimit.New()

	// Simulate exhausted rate limit
	resp := &http.Response{
		Header: http.Header{
			"X-Api-Ratelimit-Remaining": []string{"0"},
			"X-Api-Ratelimit-Reset":     []string{"9999999999"}, // Far future
		},
	}
	tracker.Update(resp)

	client := New(Config{
		BaseURL:    "http://unused",
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: tracker,
	})

	_, err := client.Get(context.Background(), "/test/", nil, nil)

	if err == nil {
		t.Fatal("Get() should return error when rate limit exceeded")
	}

	_, ok := err.(*sdkerrors.RateLimitError)
	if !ok {
		t.Errorf("error type = %T, want *sdkerrors.RateLimitError", err)
	}
}

func TestClient_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		Timeout:    1 * time.Second,
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Get(ctx, "/test/", nil, nil)

	if err == nil {
		t.Fatal("Get() should return error when context canceled")
	}
}

func TestBadRequestError_Error(t *testing.T) {
	err := &sdkerrors.BadRequestError{
		SupportContext: sdkerrors.SupportContext{
			StatusCode: 400,
			Message:    "Bad request",
			RequestID:  "req-123",
		},
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() returned empty string")
	}
	if !containsString(errStr, "req-123") {
		t.Errorf("Error() = %q, should contain request ID", errStr)
	}
}

func TestBadRequestError_Error_NoRequestID(t *testing.T) {
	err := &sdkerrors.BadRequestError{
		SupportContext: sdkerrors.SupportContext{
			StatusCode: 400,
			Message:    "Bad request",
		},
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() returned empty string")
	}
}

func TestNetworkError_Error(t *testing.T) {
	err := &sdkerrors.NetworkError{
		SupportContext: sdkerrors.SupportContext{
			Message: "connection refused",
		},
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() returned empty string")
	}
}

func TestNetworkError_Unwrap(t *testing.T) {
	cause := errors.New("test")
	err := &sdkerrors.NetworkError{
		SupportContext: sdkerrors.SupportContext{
			Message: "network error",
		},
		Cause: cause,
	}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestRateLimitError_Error(t *testing.T) {
	err := &sdkerrors.RateLimitError{
		SupportContext: sdkerrors.SupportContext{
			StatusCode: 429,
			Message:    "rate limit exceeded",
		},
		Limit:     10000,
		Remaining: 0,
		ResetAt:   time.Now().Add(1 * time.Hour),
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() returned empty string")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestClient_Get_MaxRetriesExhausted(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable) // 503 - retryable
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg: retry.Config{
			MaxRetries:     2,
			InitialBackoff: 1 * time.Millisecond,
			Multiplier:     2.0,
		},
		RateLimits: ratelimit.New(),
	})

	_, err := client.Get(context.Background(), "/test/", nil, nil)

	// Should return ServerError after max retries (status 503 is still an error)
	if err == nil {
		t.Fatal("Get() should return error for 503 status")
	}
	serverErr, ok := err.(*sdkerrors.ServerError)
	if !ok {
		t.Fatalf("error type = %T, want *sdkerrors.ServerError", err)
	}
	if serverErr.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("StatusCode = %d, want %d", serverErr.StatusCode, http.StatusServiceUnavailable)
	}
	// Verify we did retry: 1 initial + 2 retries = 3
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestClient_Get_WithDebug(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
		Debug:      true,
	})

	var result map[string]string
	_, err := client.Get(context.Background(), "/test/", nil, &result)

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestClient_Get_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})

	var result map[string]string
	_, err := client.Get(context.Background(), "/test/", nil, &result)

	if err == nil {
		t.Fatal("Get() should return error for invalid JSON")
	}
}

func TestClient_Get_WithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("foo") != "bar" {
			t.Errorf("foo = %q, want bar", r.URL.Query().Get("foo"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})

	params := make(map[string][]string)
	params["foo"] = []string{"bar"}

	var result map[string]string
	_, err := client.Get(context.Background(), "/test/", params, &result)

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestParseAPIError_AllFields(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "errmsg field",
			body: `{"s":"error","errmsg":"Test error message"}`,
			want: "Test error message",
		},
		{
			name: "message field",
			body: `{"s":"error","message":"Test message"}`,
			want: "Test message",
		},
		{
			name: "empty fields",
			body: `{"s":"error"}`,
			want: "Bad Request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &Response{
				StatusCode: http.StatusBadRequest,
				Body:       []byte(tt.body),
			}
			err := parseAPIError(resp, "https://api.example.com/v1/test/")
			apiErr, ok := err.(*sdkerrors.BadRequestError)
			if !ok {
				t.Fatalf("error type = %T, want *sdkerrors.BadRequestError", err)
			}
			if apiErr.Message != tt.want {
				t.Errorf("Message = %q, want %q", apiErr.Message, tt.want)
			}
		})
	}
}

func TestClient_Get_NilResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})

	_, err := client.Get(context.Background(), "/test/", nil, nil)

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestClient_Get_NoRateLimiter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
	})

	var result map[string]string
	_, err := client.Get(context.Background(), "/test/", nil, &result)

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestClient_Get_CfRayRequestID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cf-Ray", "cf-ray-id-123")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})

	var result map[string]string
	resp, err := client.Get(context.Background(), "/test/", nil, &result)

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if resp.RequestID != "cf-ray-id-123" {
		t.Errorf("RequestID = %q, want cf-ray-id-123", resp.RequestID)
	}
}

func TestClient_Do_ContextCanceledDuringBackoff(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable) // 503 - retryable
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg: retry.Config{
			MaxRetries:     3,
			InitialBackoff: 500 * time.Millisecond,
			Multiplier:     2.0,
		},
		RateLimits: ratelimit.New(),
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := client.Get(ctx, "/test/", nil, nil)

	var netErr *sdkerrors.NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("Get() error = %T (%v), want *NetworkError wrapping the context error", err, err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("errors.Is(err, context.Canceled) = false, want true (Cause must unwrap)")
	}
	if netErr.Timeout {
		t.Error("Timeout = true, want false for cancellation")
	}
}

func TestClient_Do_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom-Header") != "custom-value" {
			t.Errorf("X-Custom-Header = %q, want custom-value", r.Header.Get("X-Custom-Header"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})

	resp, err := client.Do(context.Background(), Request{
		Method:  http.MethodGet,
		Path:    "/test/",
		Headers: map[string]string{"X-Custom-Header": "custom-value"},
	})

	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

// testTransport is a custom http.RoundTripper that can simulate network errors
type testTransport struct {
	attempts  int
	failUntil int
	serverURL string
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.attempts++
	if t.attempts <= t.failUntil {
		return nil, &timeoutError{msg: "connection timeout"}
	}
	req.URL.Host = t.serverURL[7:] // Strip "http://"
	req.URL.Scheme = "http"
	return http.DefaultTransport.RoundTrip(req)
}

type timeoutError struct {
	msg string
}

func (e *timeoutError) Error() string   { return e.msg }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return false }

func TestClient_Do_RetryOnNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	transport := &testTransport{
		failUntil: 1,
		serverURL: server.URL,
	}

	client := New(Config{
		HTTPClient: &http.Client{Transport: transport},
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg: retry.Config{
			MaxRetries:     3,
			InitialBackoff: 1 * time.Millisecond,
			Multiplier:     2.0,
		},
		RateLimits: ratelimit.New(),
	})

	var result map[string]string
	_, err := client.Get(context.Background(), "/test/", nil, &result)

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if transport.attempts != 2 {
		t.Errorf("attempts = %d, want 2", transport.attempts)
	}
}

func TestClient_Do_MaxRetriesExhausted_NetworkError(t *testing.T) {
	transport := &testTransport{
		failUntil: 100, // Always fail
	}

	client := New(Config{
		HTTPClient: &http.Client{Transport: transport},
		BaseURL:    "http://localhost:9999",
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg: retry.Config{
			MaxRetries:     2,
			InitialBackoff: 1 * time.Millisecond,
			Multiplier:     2.0,
		},
		RateLimits: ratelimit.New(),
	})

	_, err := client.Get(context.Background(), "/test/", nil, nil)

	if err == nil {
		t.Fatal("Get() should return error after max retries exhausted")
	}
	if !containsString(err.Error(), "max retries") {
		t.Errorf("error = %v, should contain 'max retries'", err)
	}
}

func TestClient_Do_NonRetryableNetworkError(t *testing.T) {
	transport := &nonRetryableTransport{}

	client := New(Config{
		HTTPClient: &http.Client{Transport: transport},
		BaseURL:    "http://localhost:9999",
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg: retry.Config{
			MaxRetries:     3,
			InitialBackoff: 1 * time.Millisecond,
			Multiplier:     2.0,
		},
		RateLimits: ratelimit.New(),
	})

	_, err := client.Get(context.Background(), "/test/", nil, nil)

	if err == nil {
		t.Fatal("Get() should return error")
	}
	if transport.attempts != 1 {
		t.Errorf("attempts = %d, want 1 (no retry for non-retryable error)", transport.attempts)
	}
}

type nonRetryableTransport struct {
	attempts int
}

func (t *nonRetryableTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.attempts++
	return nil, &plainNetworkError{msg: "connection refused"}
}

type plainNetworkError struct {
	msg string
}

func (e *plainNetworkError) Error() string { return e.msg }

func TestClient_Do_InvalidMethod(t *testing.T) {
	client := New(Config{
		BaseURL:    "http://localhost:9999",
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})

	_, err := client.Do(context.Background(), Request{
		Method: "INVALID METHOD",
		Path:   "/test/",
	})

	if err == nil {
		t.Fatal("Do() should return error for invalid method")
	}
}

type errorReadCloser struct{}

func (e *errorReadCloser) Read(p []byte) (int, error) {
	return 0, &plainNetworkError{msg: "read error"}
}

func (e *errorReadCloser) Close() error { return nil }

type errorBodyTransport struct{}

func (t *errorBodyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       &errorReadCloser{},
	}, nil
}

func TestClient_Do_ReadBodyError(t *testing.T) {
	client := New(Config{
		HTTPClient: &http.Client{Transport: &errorBodyTransport{}},
		BaseURL:    "http://localhost:9999",
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})

	_, err := client.Do(context.Background(), Request{
		Method: http.MethodGet,
		Path:   "/test/",
	})

	if err == nil {
		t.Fatal("Do() should return error when reading body fails")
	}
}

func TestClient_Do_StatusCacheAbortsRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable) // 503 - retryable
	}))
	defer server.Close()

	// Create a status cache that reports offline
	cache := status.New(func(ctx context.Context) (bool, error) {
		return false, nil
	})
	cache.Update(false) // Mark as offline

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg: retry.Config{
			MaxRetries:     3,
			InitialBackoff: 1 * time.Millisecond,
			Multiplier:     2.0,
		},
		RateLimits:  ratelimit.New(),
		StatusCache: cache,
	})

	_, err := client.Get(context.Background(), "/test/", nil, nil)

	// Should have only made 1 attempt (no retries since status is offline)
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (no retry when API is offline)", attempts)
	}
	// Should still get a ServerError from the final response
	if err == nil {
		t.Fatal("Get() should return error for 503 status")
	}
}

func TestClient_CloseIdleConnections(t *testing.T) {
	client := New(Config{
		BaseURL:    "http://localhost:9999",
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
	})
	// Should not panic
	client.CloseIdleConnections()
}

// spyTransport tracks whether CloseIdleConnections was invoked on it, to
// prove the SDK client did (or did not) reach into a transport's connection
// pool without needing a real network stack.
type spyTransport struct {
	closeIdleCalled bool
}

func (t *spyTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("spyTransport: RoundTrip not implemented")
}

func (t *spyTransport) CloseIdleConnections() {
	t.closeIdleCalled = true
}

func TestClient_CloseIdleConnections_NoOpWhenHTTPClientInjected(t *testing.T) {
	spy := &spyTransport{}
	client := New(Config{
		BaseURL:    "http://localhost:9999",
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		HTTPClient: &http.Client{Transport: spy},
	})

	client.CloseIdleConnections()

	if spy.closeIdleCalled {
		t.Error("CloseIdleConnections() reached into a caller-injected *http.Client's shared transport, want no-op")
	}
}

func TestClient_GetUnversioned_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Unversioned should NOT have /v1/ prefix
		if r.URL.Path == "/status/" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok", "status": "online"})
			return
		}
		t.Errorf("unexpected path: %s", r.URL.Path)
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})

	var result map[string]string
	_, err := client.GetUnversioned(context.Background(), "status/", nil, &result)
	if err != nil {
		t.Fatalf("GetUnversioned() error = %v", err)
	}
	if result["status"] != "online" {
		t.Errorf("status = %q, want online", result["status"])
	}
}

func TestClient_GetUnversioned_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})

	resp, err := client.GetUnversioned(context.Background(), "status/", nil, nil)
	if err != nil {
		t.Fatalf("GetUnversioned() should not error on 404, got: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", resp.StatusCode)
	}
}

func TestClient_GetUnversioned_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "error", "errmsg": "bad"})
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})

	_, err := client.GetUnversioned(context.Background(), "status/", nil, nil)
	if err == nil {
		t.Fatal("GetUnversioned() should return error for 400")
	}
}

func TestClient_GetUnversioned_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})

	var result map[string]string
	_, err := client.GetUnversioned(context.Background(), "status/", nil, &result)
	if err == nil {
		t.Fatal("GetUnversioned() should return error for invalid JSON")
	}
	_, ok := err.(*sdkerrors.ParseError)
	if !ok {
		t.Errorf("error type = %T, want *sdkerrors.ParseError", err)
	}
}

func TestClient_GetUnversioned_NetworkError(t *testing.T) {
	transport := &testTransport{failUntil: 100}
	client := New(Config{
		HTTPClient: &http.Client{Transport: transport},
		BaseURL:    "http://localhost:9999",
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg: retry.Config{
			MaxRetries:     0,
			InitialBackoff: 1 * time.Millisecond,
			Multiplier:     2.0,
		},
		RateLimits: ratelimit.New(),
	})

	_, err := client.GetUnversioned(context.Background(), "status/", nil, nil)
	if err == nil {
		t.Fatal("GetUnversioned() should return error for network failure")
	}
}

func TestClient_Do_DefaultParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify default params are merged
		if r.URL.Query().Get("dateformat") != "unix" {
			t.Errorf("dateformat = %q, want unix", r.URL.Query().Get("dateformat"))
		}
		// Method-level params should take priority
		if r.URL.Query().Get("foo") != "bar" {
			t.Errorf("foo = %q, want bar", r.URL.Query().Get("foo"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	defaultParams := url.Values{}
	defaultParams.Set("dateformat", "unix")

	client := New(Config{
		BaseURL:       server.URL,
		APIVersion:    "v1",
		Token:         "test-key",
		RetryCfg:      retry.DefaultConfig(),
		RateLimits:    ratelimit.New(),
		DefaultParams: defaultParams,
	})

	params := url.Values{}
	params.Set("foo", "bar")

	_, err := client.Get(context.Background(), "/test/", params, nil)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestClient_Do_DefaultParamsNoOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Method-level params should NOT be overridden by default params
		if r.URL.Query().Get("dateformat") != "iso" {
			t.Errorf("dateformat = %q, want iso (method-level should win)", r.URL.Query().Get("dateformat"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	defaultParams := url.Values{}
	defaultParams.Set("dateformat", "unix")

	client := New(Config{
		BaseURL:       server.URL,
		APIVersion:    "v1",
		Token:         "test-key",
		RetryCfg:      retry.DefaultConfig(),
		RateLimits:    ratelimit.New(),
		DefaultParams: defaultParams,
	})

	params := url.Values{}
	params.Set("dateformat", "iso")

	_, err := client.Get(context.Background(), "/test/", params, nil)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestClient_Do_DefaultParamsNilRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("dateformat") != "unix" {
			t.Errorf("dateformat = %q, want unix", r.URL.Query().Get("dateformat"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	defaultParams := url.Values{}
	defaultParams.Set("dateformat", "unix")

	client := New(Config{
		BaseURL:       server.URL,
		APIVersion:    "v1",
		Token:         "test-key",
		RetryCfg:      retry.DefaultConfig(),
		RateLimits:    ratelimit.New(),
		DefaultParams: defaultParams,
	})

	// nil params should still get default params
	_, err := client.Get(context.Background(), "/test/", nil, nil)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestClient_Do_SemaphoreContextCanceled(t *testing.T) {
	sem := make(chan struct{}, 1)
	// Fill the semaphore
	sem <- struct{}{}

	client := New(Config{
		BaseURL:    "http://localhost:9999",
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
		Sem:        sem,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Do(ctx, Request{
		Method: http.MethodGet,
		Path:   "/test/",
	})

	var netErr *sdkerrors.NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("Do() error = %T (%v), want *NetworkError wrapping the context error", err, err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("errors.Is(err, context.Canceled) = false, want true (Cause must unwrap)")
	}
}

func TestClient_Do_DemoMode(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		DemoMode:   true,
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})

	_, err := client.Get(context.Background(), "/test/", nil, nil)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if receivedAuth != "" {
		t.Errorf("Authorization header = %q, should be empty in demo mode", receivedAuth)
	}
}

func TestClient_Get_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})

	resp, err := client.Get(context.Background(), "/test/", nil, nil)
	if err != nil {
		t.Fatalf("Get() should not error on 404, got: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", resp.StatusCode)
	}
}

func TestParseAPIError_AllStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantType   string
	}{
		{"400 bad request", 400, "*sdkerrors.BadRequestError"},
		{"401 auth error", 401, "*sdkerrors.AuthenticationError"},
		{"402 payment required", 402, "*sdkerrors.PaymentRequiredError"},
		{"403 forbidden", 403, "*sdkerrors.ForbiddenError"},
		{"404 not found", 404, "*sdkerrors.NotFoundError"},
		{"413 payload too large", 413, "*sdkerrors.PayloadTooLargeError"},
		{"429 rate limit", 429, "*sdkerrors.RateLimitError"},
		{"500 internal error", 500, "*sdkerrors.InternalError"},
		{"502 server error", 502, "*sdkerrors.ServerError"},
		{"503 server error", 503, "*sdkerrors.ServerError"},
		{"509 server error", 509, "*sdkerrors.ServerError"},
		{"524 server error", 524, "*sdkerrors.ServerError"},
		{"529 server error", 529, "*sdkerrors.ServerError"},
		{"530 server error", 530, "*sdkerrors.ServerError"},
		{"540 server error", 540, "*sdkerrors.ServerError"},
		{"598 server error", 598, "*sdkerrors.ServerError"},
		{"418 default bad request", 418, "*sdkerrors.BadRequestError"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &Response{
				StatusCode: tt.statusCode,
				Headers:    http.Header{},
				Body:       []byte(`{"s":"error","errmsg":"test error"}`),
				RequestID:  "test-id",
			}
			err := parseAPIError(resp, "https://api.example.com/test")

			gotType := fmt.Sprintf("%T", err)
			if gotType != tt.wantType {
				t.Errorf("parseAPIError() type = %s, want %s", gotType, tt.wantType)
			}
		})
	}
}

func TestParseAPIError_MessageField(t *testing.T) {
	resp := &Response{
		StatusCode: 400,
		Body:       []byte(`{"s":"error","message":"from message field"}`),
	}
	err := parseAPIError(resp, "https://api.example.com/test")
	badReq, _ := err.(*sdkerrors.BadRequestError)
	if badReq.Message != "from message field" {
		t.Errorf("Message = %q, want 'from message field'", badReq.Message)
	}
}

func TestParseAPIError_FallbackStatusText(t *testing.T) {
	resp := &Response{
		StatusCode: 400,
		Body:       []byte(`{}`),
	}
	err := parseAPIError(resp, "https://api.example.com/test")
	badReq, _ := err.(*sdkerrors.BadRequestError)
	if badReq.Message != "Bad Request" {
		t.Errorf("Message = %q, want 'Bad Request'", badReq.Message)
	}
}

func TestParseAPIError_InvalidJSON(t *testing.T) {
	resp := &Response{
		StatusCode: 400,
		Body:       []byte("not json"),
	}
	err := parseAPIError(resp, "https://api.example.com/test")
	// Should still return a BadRequestError with fallback status text
	badReq, _ := err.(*sdkerrors.BadRequestError)
	if badReq.Message != "Bad Request" {
		t.Errorf("Message = %q, want 'Bad Request'", badReq.Message)
	}
}

// TestParseAPIError_CSVBody pins the finding (verified live against the
// API) that error bodies are serialized in the requested wire format, not
// always JSON: a 400 from a format=csv request comes back as
// "s,errmsg\nerror,\"message\"", not a JSON object. parseAPIError must
// extract the same errmsg either way.
func TestParseAPIError_CSVBody(t *testing.T) {
	resp := &Response{
		StatusCode: 400,
		Headers:    http.Header{"Content-Type": []string{"text/csv; charset=utf-8"}},
		Body:       []byte("s,errmsg\r\nerror,\"Bad parameters, please check API documentation.\"\r\n"),
	}
	err := parseAPIError(resp, "https://api.example.com/test")
	badReq, ok := err.(*sdkerrors.BadRequestError)
	if !ok {
		t.Fatalf("error type = %T, want *sdkerrors.BadRequestError", err)
	}
	if badReq.Message != "Bad parameters, please check API documentation." {
		t.Errorf("Message = %q, want the errmsg column's value", badReq.Message)
	}
}

func TestParseAPIError_CSVBody_Malformed(t *testing.T) {
	resp := &Response{
		StatusCode: 400,
		Headers:    http.Header{"Content-Type": []string{"text/csv"}},
		Body:       []byte("not,even,csv\"unterminated"),
	}
	err := parseAPIError(resp, "https://api.example.com/test")
	badReq, ok := err.(*sdkerrors.BadRequestError)
	if !ok {
		t.Fatalf("error type = %T, want *sdkerrors.BadRequestError", err)
	}
	if badReq.Message != "Bad Request" {
		t.Errorf("Message = %q, want the generic HTTP status text fallback", badReq.Message)
	}
}

func TestParseCSVErrorFields_HeaderOnly(t *testing.T) {
	// A header row with no data row (truncated body) should degrade to a
	// zero-value result, not panic or return a partial/misaligned row.
	fields := parseCSVErrorFields([]byte("s,errmsg\n"))
	if fields != (apiErrorResponse{}) {
		t.Errorf("parseCSVErrorFields(header-only) = %+v, want zero value", fields)
	}
}

func TestParseAPIError_403_ExtendedFields(t *testing.T) {
	body := `{
		"s": "error",
		"errmsg": "Access denied. Only one device is permitted.",
		"authorizedIP": "107.178.202.2",
		"blockedIP": "44.116.21.32",
		"troubleshootingGuide": "https://www.marketdata.app/docs/api/troubleshooting/multiple-ip-addresses"
	}`
	resp := &Response{
		StatusCode: 403,
		Headers:    http.Header{},
		Body:       []byte(body),
		RequestID:  "ray-123",
	}
	err := parseAPIError(resp, "https://api.example.com/test")
	forbidden, ok := err.(*sdkerrors.ForbiddenError)
	if !ok {
		t.Fatalf("error type = %T, want *sdkerrors.ForbiddenError", err)
	}
	if forbidden.AuthorizedIP != "107.178.202.2" {
		t.Errorf("AuthorizedIP = %q, want %q", forbidden.AuthorizedIP, "107.178.202.2")
	}
	if forbidden.BlockedIP != "44.116.21.32" {
		t.Errorf("BlockedIP = %q, want %q", forbidden.BlockedIP, "44.116.21.32")
	}
	if forbidden.TroubleshootingGuide == "" {
		t.Error("TroubleshootingGuide should not be empty")
	}
}

func TestParseAPIError_429_RateLimitHeaders(t *testing.T) {
	body := `{
		"s": "error",
		"errmsg": "API credit limit reached for current window.",
		"troubleshootingGuide": "https://www.marketdata.app/docs/api/troubleshooting/running-out-of-credits"
	}`
	headers := http.Header{}
	headers.Set("X-Api-Ratelimit-Limit", "10000")
	headers.Set("X-Api-Ratelimit-Remaining", "0")
	headers.Set("X-Api-Ratelimit-Reset", "1711500600")

	resp := &Response{
		StatusCode: 429,
		Headers:    headers,
		Body:       []byte(body),
		RequestID:  "ray-456",
	}
	err := parseAPIError(resp, "https://api.example.com/test")
	rateErr, ok := err.(*sdkerrors.RateLimitError)
	if !ok {
		t.Fatalf("error type = %T, want *sdkerrors.RateLimitError", err)
	}
	if rateErr.Limit != 10000 {
		t.Errorf("Limit = %d, want 10000", rateErr.Limit)
	}
	if rateErr.Remaining != 0 {
		t.Errorf("Remaining = %d, want 0", rateErr.Remaining)
	}
	if rateErr.ResetAt.IsZero() {
		t.Error("ResetAt should not be zero")
	}
	if rateErr.TroubleshootingGuide == "" {
		t.Error("TroubleshootingGuide should not be empty")
	}
}

func TestParseAPIError_429_NoHeaders(t *testing.T) {
	resp := &Response{
		StatusCode: 429,
		Headers:    http.Header{},
		Body:       []byte(`{"s":"error","errmsg":"rate limited"}`),
	}
	err := parseAPIError(resp, "https://api.example.com/test")
	rateErr, ok := err.(*sdkerrors.RateLimitError)
	if !ok {
		t.Fatalf("error type = %T, want *sdkerrors.RateLimitError", err)
	}
	if rateErr.Limit != 0 {
		t.Errorf("Limit = %d, want 0 when header missing", rateErr.Limit)
	}
}

func TestParseAPIError_500_InternalError(t *testing.T) {
	resp := &Response{
		StatusCode: 500,
		Headers:    http.Header{},
		Body:       []byte(`{"s":"error","errmsg":"unknown failure"}`),
		RequestID:  "ray-789",
	}
	err := parseAPIError(resp, "https://api.example.com/test")
	internal, ok := err.(*sdkerrors.InternalError)
	if !ok {
		t.Fatalf("error type = %T, want *sdkerrors.InternalError", err)
	}
	if internal.Retryable() {
		t.Error("InternalError (500) should not be retryable")
	}
	if internal.Message != "unknown failure" {
		t.Errorf("Message = %q, want 'unknown failure'", internal.Message)
	}
}

func TestClient_Do_StatusCacheAbortsRetry_NoResponse(t *testing.T) {
	// Test the case where status cache aborts retry and there's no lastResp (only error)
	transport := &testTransport{failUntil: 100}

	cache := status.New(func(ctx context.Context) (bool, error) {
		return false, nil
	})
	cache.Update(false) // Mark as offline

	client := New(Config{
		HTTPClient: &http.Client{Transport: transport},
		BaseURL:    "http://localhost:9999",
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg: retry.Config{
			MaxRetries:     3,
			InitialBackoff: 1 * time.Millisecond,
			Multiplier:     2.0,
		},
		RateLimits:  ratelimit.New(),
		StatusCache: cache,
	})

	_, err := client.Get(context.Background(), "/test/", nil, nil)
	if err == nil {
		t.Fatal("Get() should return error when API is offline and no response")
	}
	if !containsString(err.Error(), "API is offline") {
		t.Errorf("error = %v, should contain 'API is offline'", err)
	}
}

func TestClient_logDebug_NoDebug(t *testing.T) {
	client := New(Config{
		BaseURL:    "http://localhost:9999",
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		Debug:      false,
	})
	// Should not panic
	client.logDebug("test message", "key", "value")
}

func TestClient_logDebug_WithDebug(t *testing.T) {
	client := New(Config{
		BaseURL:    "http://localhost:9999",
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		Debug:      true,
	})
	// Should not panic
	client.logDebug("test message", "key", "value")
}

func TestClient_SetDebug_TogglesEmission(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := New(Config{
		BaseURL:    "http://localhost:9999",
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		Logger:     logger,
	})

	client.logDebug("off-before")
	client.SetDebug(true)
	client.logDebug("on")
	client.SetDebug(false)
	client.logDebug("off-after")

	out := buf.String()
	if !containsString(out, "on") {
		t.Error("SetDebug(true) did not enable debug record emission")
	}
	if containsString(out, "off-before") || containsString(out, "off-after") {
		t.Errorf("debug records emitted while disabled: %q", out)
	}
}

func TestBuildURL(t *testing.T) {
	client := New(Config{
		BaseURL:    "https://api.example.com",
		APIVersion: "v1",
		Token:      "test-key",
	})

	// Without params
	u := client.buildURL("stocks/candles/", nil)
	if u != "https://api.example.com/v1/stocks/candles/" {
		t.Errorf("buildURL() = %q, want https://api.example.com/v1/stocks/candles/", u)
	}

	// With leading slash
	u = client.buildURL("/stocks/candles/", nil)
	if u != "https://api.example.com/v1/stocks/candles/" {
		t.Errorf("buildURL() = %q, want https://api.example.com/v1/stocks/candles/", u)
	}
}

func TestBuildURLUnversioned(t *testing.T) {
	client := New(Config{
		BaseURL:    "https://api.example.com",
		APIVersion: "v1",
		Token:      "test-key",
	})

	// Without params
	u := client.buildURLUnversioned("status/", nil)
	if u != "https://api.example.com/status/" {
		t.Errorf("buildURLUnversioned() = %q, want https://api.example.com/status/", u)
	}

	// With leading slash
	u = client.buildURLUnversioned("/status/", nil)
	if u != "https://api.example.com/status/" {
		t.Errorf("buildURLUnversioned() = %q, want https://api.example.com/status/", u)
	}

	// With params
	params := url.Values{}
	params.Set("key", "value")
	u = client.buildURLUnversioned("status/", params)
	if u != "https://api.example.com/status/?key=value" {
		t.Errorf("buildURLUnversioned() = %q, want https://api.example.com/status/?key=value", u)
	}
}

func TestClient_New_CustomHTTPClient(t *testing.T) {
	// The SDK must use the caller's Transport (connection pool, proxy, TLS)
	// but operate on its own shallow copy: writing Timeout/CheckRedirect
	// onto the caller's object would surprise them and race with their
	// in-flight requests.
	callerRedirect := func(req *http.Request, via []*http.Request) error { return nil }
	customClient := &http.Client{
		Transport:     &http.Transport{},
		Timeout:       30 * time.Second,
		CheckRedirect: callerRedirect,
	}
	client := New(Config{
		HTTPClient: customClient,
		BaseURL:    "https://api.example.com",
		APIVersion: "v1",
		Token:      "test-key",
		Timeout:    99 * time.Second,
	})

	if client.http == customClient {
		t.Error("SDK must not adopt the caller's *http.Client object itself")
	}
	if client.http.Transport != customClient.Transport {
		t.Error("SDK copy must share the caller's Transport")
	}
	if client.http.Timeout != 99*time.Second {
		t.Errorf("SDK copy Timeout = %v, want the SDK's own 99s", client.http.Timeout)
	}
	// The caller's object is untouched.
	if customClient.Timeout != 30*time.Second {
		t.Errorf("caller's Timeout mutated to %v", customClient.Timeout)
	}
	if fmt.Sprintf("%p", customClient.CheckRedirect) != fmt.Sprintf("%p", callerRedirect) {
		t.Error("caller's CheckRedirect mutated")
	}
}

// TestClient_New_DoesNotRaceWithInjectedClientInFlight reproduces the CI
// race caught on the PR #14 merge: constructing a second SDK client around
// the same injected *http.Client used to write Timeout/CheckRedirect onto
// it while the first client's requests were reading them in the stdlib.
func TestClient_New_DoesNotRaceWithInjectedClientInFlight(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok"}`))
	}))
	defer server.Close()

	shared := &http.Client{}
	cfg := Config{
		HTTPClient: shared,
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	}
	a := New(cfg)

	done := make(chan struct{})
	go func() {
		defer close(done)
		var out map[string]any
		for i := 0; i < 30; i++ {
			_, _ = a.Get(context.Background(), "stocks/quotes/", nil, &out)
		}
	}()
	for i := 0; i < 30; i++ {
		_ = New(cfg)
	}
	<-done
}

func TestStatusProbe(t *testing.T) {
	for _, tc := range []struct {
		status int
		online bool
	}{{200, true}, {203, true}, {500, false}, {503, false}} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/status/" {
				t.Errorf("probe path = %q, want /status/", r.URL.Path)
			}
			w.WriteHeader(tc.status)
		}))
		client := New(Config{BaseURL: server.URL, APIVersion: "v1", Token: "k", RetryCfg: retry.DefaultConfig()})
		online, err := client.StatusProbe(context.Background())
		if err != nil {
			t.Errorf("status %d: StatusProbe error = %v", tc.status, err)
		}
		if online != tc.online {
			t.Errorf("status %d: online = %t, want %t", tc.status, online, tc.online)
		}
		server.Close()
	}
}

func TestStatusProbe_BypassesSaturatedPool(t *testing.T) {
	// The probe must answer while every pool slot is taken — the exact
	// situation of a real outage, when the gate needs it most.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	sem := make(chan struct{}, 1)
	sem <- struct{}{} // saturate the pool
	client := New(Config{BaseURL: server.URL, APIVersion: "v1", Token: "k", RetryCfg: retry.DefaultConfig(), Sem: sem})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	online, err := client.StatusProbe(ctx)
	if err != nil || !online {
		t.Fatalf("StatusProbe with saturated pool = (%t, %v), want (true, nil)", online, err)
	}
}

func TestStatusProbe_InvalidBaseURL(t *testing.T) {
	client := New(Config{BaseURL: "http://exa mple.com", APIVersion: "v1", Token: "k", RetryCfg: retry.DefaultConfig()})
	if _, err := client.StatusProbe(context.Background()); err == nil {
		t.Fatal("StatusProbe with an unparsable base URL should error")
	}
}

func TestStatusProbe_Unreachable(t *testing.T) {
	client := New(Config{BaseURL: "http://127.0.0.1:1", APIVersion: "v1", Token: "k", RetryCfg: retry.DefaultConfig()})
	if _, err := client.StatusProbe(context.Background()); err == nil {
		t.Fatal("StatusProbe against an unreachable host should error")
	}
}

func TestClient_Do_DefaultParams_DoNotMutateCaller(t *testing.T) {
	var gotColumns []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotColumns = r.URL.Query()["columns"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok"}`))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		DefaultParams: url.Values{
			"columns": {"bid", "ask"}, // multi-value default
		},
	})

	callerParams := url.Values{"symbol": {"AAPL"}}
	var out map[string]any
	if _, err := client.Get(context.Background(), "stocks/quotes/", callerParams, &out); err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// The caller's map must be untouched by the defaults merge.
	if len(callerParams) != 1 || callerParams.Get("symbol") != "AAPL" {
		t.Errorf("caller's params mutated: %v", callerParams)
	}
	// The wire must carry every value of a multi-value default, joined into
	// the single value the API actually reads, with the "s" envelope column
	// prepended: a columns filter makes the API drop the status field that
	// every typed method checks, so the decoding path asks for it back (the
	// formatted facets do not — see Do).
	if len(gotColumns) != 1 || gotColumns[0] != "s,bid,ask" {
		t.Errorf("columns on the wire = %v, want [s,bid,ask]", gotColumns)
	}
}

func TestSanitizeMessage_TruncatesOnRuneBoundary(t *testing.T) {
	// A two-byte rune straddling the 500-byte cut must not be split.
	msg := strings.Repeat("a", maxErrorMessageLen-1) + "é" + strings.Repeat("b", 50)
	out := sanitizeMessage(msg)
	if !utf8.ValidString(out) {
		t.Errorf("sanitizeMessage produced invalid UTF-8: %q", out[maxErrorMessageLen-5:])
	}
	if !strings.HasSuffix(out, "…(truncated)") {
		t.Errorf("missing truncation marker: %q", out)
	}
	if prefix := strings.TrimSuffix(out, "…(truncated)"); len(prefix) != maxErrorMessageLen-1 {
		t.Errorf("cut at %d bytes, want %d (backed off before the split rune)", len(prefix), maxErrorMessageLen-1)
	}
}

func TestClient_Do_StatusCacheAbortsRetry_WithLastResp(t *testing.T) {
	// First request returns 503 (retryable), then status cache says offline.
	// Should return lastResp (not error) because we have a response.
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"s":"error","errmsg":"unavailable"}`))
	}))
	defer server.Close()

	cache := status.New(func(ctx context.Context) (bool, error) {
		return false, nil
	})
	cache.Update(false) // Mark as offline

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg: retry.Config{
			MaxRetries:     3,
			InitialBackoff: 1 * time.Millisecond,
			Multiplier:     2.0,
		},
		RateLimits:  ratelimit.New(),
		StatusCache: cache,
	})

	resp, err := client.Do(context.Background(), Request{
		Method: http.MethodGet,
		Path:   "/test/",
	})

	// Should return the response (not nil) since we got a 503 response
	if resp == nil {
		t.Fatal("Do() should return lastResp when API is offline")
	}
	if resp.StatusCode != 503 {
		t.Errorf("StatusCode = %d, want 503", resp.StatusCode)
	}
	// err should be nil since we return the response
	if err != nil {
		t.Errorf("Do() error = %v, want nil (response returned)", err)
	}
}

func TestVersionFromBuildInfo(t *testing.T) {
	tests := []struct {
		name string
		info *debug.BuildInfo
		ok   bool
		want string
	}{
		{
			name: "no build info",
			info: nil,
			ok:   false,
			want: "unknown",
		},
		{
			name: "SDK as dependency",
			info: &debug.BuildInfo{
				Deps: []*debug.Module{
					{Path: "github.com/other/mod", Version: "v1.0.0"},
					{Path: modulePath, Version: "v2.1.3"},
				},
			},
			ok:   true,
			want: "v2.1.3",
		},
		{
			name: "SDK as stamped main module",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: modulePath, Version: "v2.0.0"},
			},
			ok:   true,
			want: "v2.0.0",
		},
		{
			name: "SDK as devel main module",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: modulePath, Version: "(devel)"},
			},
			ok:   true,
			want: "unknown",
		},
		{
			name: "unrelated main module",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "github.com/other/app", Version: "v9.9.9"},
			},
			ok:   true,
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionFromBuildInfo(tt.info, tt.ok); got != tt.want {
				t.Errorf("versionFromBuildInfo() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestClient_Do_ReleasesReservationOnFailure pins the credit accounting
// contract behaviorally: with remaining=1, a FAILED request must release
// its pre-flight reservation, or the phantom reservation would block the
// next request before it reaches the wire.
func TestClient_Do_ReleasesReservationOnFailure(t *testing.T) {
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "1")
		w.Header().Set("X-Api-Ratelimit-Reset", "1899999999")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"s":"error","errmsg":"boom"}`))
	}))
	defer server.Close()

	// The first failed response seeds the tracker with remaining=1 via its
	// rate-limit headers; the second request then depends on the first
	// having released its reservation.
	tracker := ratelimit.New()
	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.Config{MaxRetries: 0, InitialBackoff: time.Millisecond, Multiplier: 2},
		RateLimits: tracker,
	})

	var out map[string]any
	for i := 1; i <= 2; i++ {
		if _, err := client.Get(context.Background(), "stocks/quotes/", nil, &out); err == nil {
			t.Fatalf("request %d: expected the 500 to surface as an error", i)
		}
	}
	// Both requests must have reached the server: if the first failure
	// leaked its reservation, the second would be pre-flight blocked.
	if got := calls.Load(); got != 2 {
		t.Errorf("server received %d requests, want 2 (reservation leaked on failure?)", got)
	}
}

// TestClient_Do_PoolCapsConcurrency drives more requests than pool slots
// through Do and asserts the semaphore actually bounds in-flight requests
// (ADR-014's pool-size claim, exercised at total requests over the real
// DefaultPoolSize). Sizing the pool from the same constant NewClient uses
// — rather than a test-local literal — means this test genuinely fails if
// the production default ever changes without anyone updating it here.
func TestClient_Do_PoolCapsConcurrency(t *testing.T) {
	const slots, total = DefaultPoolSize, DefaultPoolSize*2 + 20
	var inFlight, maxInFlight atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := inFlight.Add(1)
		for {
			max := maxInFlight.Load()
			if cur <= max || maxInFlight.CompareAndSwap(max, cur) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		inFlight.Add(-1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok"}`))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
		Sem:        make(chan struct{}, slots),
	})

	var wg sync.WaitGroup
	errs := make(chan error, total)
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var out map[string]any
			if _, err := client.Get(context.Background(), "stocks/quotes/", nil, &out); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("request error: %v", err)
	}
	if got := maxInFlight.Load(); got > slots {
		t.Errorf("max in-flight = %d, want <= %d", got, slots)
	}
}

// TestResponse_SupportContext verifies Response.SupportContext directly —
// not just transitively through marketdata/* packages' APIError call
// sites — so this package's own coverage is self-contained regardless of
// cross-package attribution (the project's coverage gate runs `go test
// ./marketdata/... ./internal/...` without -coverpkg, so a function only
// exercised from a caller package's tests reports as uncovered here).
func TestResponse_SupportContext(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://api.example.com/v1/stocks/quotes/?symbol=AAPL", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp := &Response{
		Raw:        &http.Response{Request: req},
		StatusCode: 200,
		RequestID:  "req-abc",
	}

	ctx := resp.SupportContext("unexpected response status: error", "APIError")

	if ctx.RequestID != "req-abc" {
		t.Errorf("RequestID = %q, want req-abc", ctx.RequestID)
	}
	// The full wire URL, query included — the same contract the >=400 and
	// ParseError blocks follow. Reporting the path alone stripped the very
	// parameters most likely to explain a 200-with-error-body, and on the
	// query-addressed endpoints it dropped the symbol too.
	if ctx.RequestURL != "https://api.example.com/v1/stocks/quotes/?symbol=AAPL" {
		t.Errorf("RequestURL = %q, want the full wire URL with its query", ctx.RequestURL)
	}
	if ctx.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", ctx.StatusCode)
	}
	if ctx.Message != "unexpected response status: error" {
		t.Errorf("Message = %q, want the given message", ctx.Message)
	}
	if ctx.ExceptionType != "APIError" {
		t.Errorf("ExceptionType = %q, want APIError", ctx.ExceptionType)
	}
	if ctx.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestResponse_SupportContext_NilRequest(t *testing.T) {
	// A Response with no underlying Raw.Request (e.g. hand-built in a test)
	// must not panic and should leave RequestURL empty.
	resp := &Response{Raw: &http.Response{}, StatusCode: 500}
	ctx := resp.SupportContext("boom", "ServerError")
	if ctx.RequestURL != "" {
		t.Errorf("RequestURL = %q, want empty when Raw.Request is nil", ctx.RequestURL)
	}
}

// TestResponse_StatusError verifies the helper the service packages' 200-
// but-not-ok guards share, directly rather than transitively, for the same
// coverage-attribution reason as TestResponse_SupportContext above. It
// pins both halves of what the call sites rely on: the error is an
// *sdkerrors.APIError reachable through errors.As, and it carries the
// response's own support context with the status folded into the message.
func TestResponse_StatusError(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://api.example.com/v1/options/chain/AAPL/?side=call", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp := &Response{
		Raw:        &http.Response{Request: req},
		StatusCode: 200,
		RequestID:  "req-xyz",
	}

	var apiErr *sdkerrors.APIError
	if !errors.As(resp.StatusError("error"), &apiErr) {
		t.Fatalf("StatusError() should return an *sdkerrors.APIError")
	}
	if apiErr.Message != "unexpected response status: error" {
		t.Errorf("Message = %q, want unexpected response status: error", apiErr.Message)
	}
	if apiErr.ExceptionType != "APIError" {
		t.Errorf("ExceptionType = %q, want APIError", apiErr.ExceptionType)
	}
	if apiErr.RequestID != "req-xyz" {
		t.Errorf("RequestID = %q, want req-xyz", apiErr.RequestID)
	}
	if apiErr.RequestURL != "https://api.example.com/v1/options/chain/AAPL/?side=call" {
		t.Errorf("RequestURL = %q, want the full wire URL with its query", apiErr.RequestURL)
	}
}

// midReadTransport scripts a response per call.
type midReadTransport func(*http.Request) (*http.Response, error)

func (f midReadTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// timeoutReadErr is a read failure that satisfies net.Error and reports a
// timeout, without wrapping context.DeadlineExceeded.
type timeoutReadErr struct{}

func (timeoutReadErr) Error() string   { return "read tcp: i/o timeout" }
func (timeoutReadErr) Timeout() bool   { return true }
func (timeoutReadErr) Temporary() bool { return true }

// erroringBody yields some bytes and then fails, standing in for a body that
// dies partway through io.ReadAll.
type erroringBody struct {
	prefix []byte
	err    error
}

func (b *erroringBody) Read(p []byte) (int, error) {
	if len(b.prefix) > 0 {
		n := copy(p, b.prefix)
		b.prefix = b.prefix[n:]
		return n, nil
	}
	return 0, b.err
}

func (b *erroringBody) Close() error { return nil }

// TestMidReadFailureClassifiesTimeout pins that the mid-read branch reads
// the error rather than assuming. The context-deadline half is covered end
// to end by marketdata.TestMidReadTimeoutSetsTimeoutFlag; this covers the
// other two cases, which share
// one statement with it and would otherwise be executed but never compared:
// a transport timeout that satisfies net.Error without wrapping
// context.DeadlineExceeded must set the flag, and a body that merely ended
// early must not.
func TestMidReadFailureClassifiesTimeout(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"a net.Error timeout sets the flag", timeoutReadErr{}, true},
		{"a body that ended early does not", io.ErrUnexpectedEOF, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := midReadTransport(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Header:     http.Header{"Content-Type": {"application/json"}},
					Body:       &erroringBody{prefix: []byte(`{"s":`), err: tt.err},
					Request:    r,
				}, nil
			})

			client := New(Config{
				HTTPClient: &http.Client{Transport: transport},
				BaseURL:    "https://api.test",
				APIVersion: "v1",
				Token:      "k",
				RetryCfg:   retry.Config{MaxRetries: 0, InitialBackoff: time.Millisecond, Multiplier: 2},
			})

			var out map[string]any
			_, err := client.Get(context.Background(), "stocks/quotes/", nil, &out)
			var netErr *sdkerrors.NetworkError
			if !errors.As(err, &netErr) {
				t.Fatalf("want a NetworkError, got %T: %v", err, err)
			}
			if netErr.Timeout != tt.want {
				t.Errorf("Timeout = %v, want %v — the mid-read branch is not classifying %v", netErr.Timeout, tt.want, tt.err)
			}
			if netErr.StatusCode != 200 {
				t.Errorf("StatusCode = %d, want 200 carried over from the interrupted response", netErr.StatusCode)
			}
		})
	}
}

// TestGet_LogsTerminalFailures pins C-1: every terminal failure Get can
// produce — not just the >=400 case apiError used to log directly — must
// appear in operator logs exactly once. In particular, a ParseError (a
// 200 response with a malformed body) was never logged before this fix:
// it's synthesized by Get AFTER Do already returned successfully, so it
// never passed through Do's own error path.
func TestGet_LogsTerminalFailures(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		body       string
		wantType   string
	}{
		{"api error (>=400)", 400, `{"s":"error","errmsg":"boom"}`, "BadRequestError"},
		{"parse error (malformed body)", 200, `not json`, "ParseError"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			client := New(Config{
				BaseURL: server.URL, APIVersion: "v1", Token: "k",
				RetryCfg: retry.DefaultConfig(), Logger: logger,
			})

			var out map[string]any
			_, err := client.Get(context.Background(), "stocks/quotes/", nil, &out)
			if err == nil {
				t.Fatal("Get() should return an error")
			}

			logged := buf.String()
			if !containsString(logged, "level=ERROR") || !containsString(logged, "request failed") {
				t.Errorf("terminal failure was not logged at ERROR: %q", logged)
			}
			if !containsString(logged, tc.wantType) {
				t.Errorf("log line missing exception type %q: %q", tc.wantType, logged)
			}
			// Exactly one ERROR line — no double-logging.
			if n := strings.Count(logged, "level=ERROR"); n != 1 {
				t.Errorf("logged %d ERROR lines, want exactly 1: %q", n, logged)
			}
		})
	}
}

// TestGetUnversioned_LogsTerminalFailure closes the same gap for
// GetUnversioned (used by the offline-status probe's sibling, /user/,
// /headers/) — the >=400 case previously logged only through Get's path.
func TestGetUnversioned_LogsTerminalFailure(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"s":"error","errmsg":"boom"}`))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "k",
		RetryCfg: retry.Config{MaxRetries: 0, InitialBackoff: time.Millisecond, Multiplier: 2},
		Logger:   logger,
	})

	var out map[string]any
	_, err := client.GetUnversioned(context.Background(), "status/", nil, &out)
	if err == nil {
		t.Fatal("GetUnversioned() should return an error")
	}
	if !containsString(buf.String(), "level=ERROR") || !containsString(buf.String(), "request failed") {
		t.Errorf("terminal failure was not logged at ERROR: %q", buf.String())
	}
}

// TestGetUnversionedSilent_DoesNotLog verifies the silent variant reserved
// for best-effort background priming (startup rate-limit init) never emits
// the ERROR log that GetUnversioned would, even though it returns the same
// error — the caller already discards it by design.
func TestGetUnversionedSilent_DoesNotLog(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"s":"error","errmsg":"boom"}`))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "k",
		RetryCfg: retry.Config{MaxRetries: 0, InitialBackoff: time.Millisecond, Multiplier: 2},
		Logger:   logger,
	})

	var out map[string]any
	_, err := client.GetUnversionedSilent(context.Background(), "status/", nil, &out)
	if err == nil {
		t.Fatal("GetUnversionedSilent() should return an error")
	}
	if buf.Len() != 0 {
		t.Errorf("GetUnversionedSilent() logged output, want none: %q", buf.String())
	}
}

// TestGet_NoDoubleLogForDoOriginatedError verifies an error that Do()
// itself originates (a pre-flight rate-limit rejection, in this case) is
// logged exactly once by Get's wrapper — not once inside Do and again in
// Get, which centralizing logging at Get/GetUnversioned instead of Do
// avoids by construction. A pre-flight rejection logs at WARN, not ERROR
// (see TestLogTerminalFailure_PreFlightRateLimitLogsWarnNotError) — this
// test only pins the "exactly once" part.
func TestGet_NoDoubleLogForDoOriginatedError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tracker := ratelimit.New()
	tracker.Update(&http.Response{Header: http.Header{
		"X-Api-Ratelimit-Limit":     []string{"100"},
		"X-Api-Ratelimit-Remaining": []string{"0"},
		"X-Api-Ratelimit-Reset":     []string{fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix())},
	}})
	client := New(Config{
		BaseURL: "http://127.0.0.1:1", APIVersion: "v1", Token: "k",
		RetryCfg: retry.DefaultConfig(), RateLimits: tracker, Logger: logger,
	})

	var out map[string]any
	_, err := client.Get(context.Background(), "stocks/quotes/", nil, &out)
	if err == nil {
		t.Fatal("Get() should return the pre-flight rate limit error")
	}
	if n := strings.Count(buf.String(), "level=WARN"); n != 1 {
		t.Errorf("logged %d WARN lines for a Do-originated error, want exactly 1: %q", n, buf.String())
	}
	if strings.Contains(buf.String(), "level=ERROR") {
		t.Errorf("a pre-flight rate-limit rejection should not log at ERROR: %q", buf.String())
	}
}

// TestLogTerminalFailure_PreFlightRateLimitLogsWarnNotError pins the level
// distinction directly: a pre-flight RateLimitError (the SDK's own
// reservation catching the request before it was ever sent) is expected
// throttling, not a server-reported failure — WARN, not ERROR.
func TestLogTerminalFailure_PreFlightRateLimitLogsWarnNotError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := New(Config{
		BaseURL: "http://127.0.0.1:1", APIVersion: "v1", Token: "k",
		RetryCfg: retry.DefaultConfig(), Logger: logger,
	})

	client.logTerminalFailure(context.Background(), &sdkerrors.RateLimitError{
		SupportContext: sdkerrors.SupportContext{Message: "rate limit exceeded (pre-flight check)"},
		PreFlight:      true,
	})

	if !containsString(buf.String(), "level=WARN") {
		t.Errorf("pre-flight RateLimitError should log at WARN: %q", buf.String())
	}
	if containsString(buf.String(), "level=ERROR") {
		t.Errorf("pre-flight RateLimitError should not log at ERROR: %q", buf.String())
	}
}

// TestLogTerminalFailure_ServerRateLimitLogsError verifies a genuine
// server-reported 429 (PreFlight false) still logs at ERROR like any other
// terminal failure — only the SDK's own pre-flight rejection is demoted.
func TestLogTerminalFailure_ServerRateLimitLogsError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := New(Config{
		BaseURL: "http://127.0.0.1:1", APIVersion: "v1", Token: "k",
		RetryCfg: retry.DefaultConfig(), Logger: logger,
	})

	client.logTerminalFailure(context.Background(), &sdkerrors.RateLimitError{
		SupportContext: sdkerrors.SupportContext{Message: "rate limit exceeded"},
	})

	if !containsString(buf.String(), "level=ERROR") {
		t.Errorf("server-reported RateLimitError should log at ERROR: %q", buf.String())
	}
}

// TestClient_GetFormatted_DoesNotMutateCallerParams pins the contract Do
// documents for its own default merge: the params map belongs to the caller.
// Before this was fixed, GetFormatted set "format" directly into the
// caller's map, so reusing one map across a JSON Get and a GetFormatted
// leaked format=csv into the JSON request, and two concurrent GetFormatted
// calls sharing a map raced on a concurrent map write.
func TestClient_GetFormatted_DoesNotMutateCallerParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		_, _ = w.Write([]byte("symbol,last\nAAPL,150.22\n"))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "test-key",
		RetryCfg: retry.DefaultConfig(),
	})

	params := url.Values{"symbols": []string{"AAPL"}}
	if _, err := client.GetFormatted(context.Background(), "stocks/quotes/", params, "csv"); err != nil {
		t.Fatalf("GetFormatted() error = %v", err)
	}
	if _, ok := params["format"]; ok {
		t.Errorf("caller params were mutated: %v, want no format key", params)
	}
	if len(params) != 1 {
		t.Errorf("caller params = %v, want exactly the one key the caller set", params)
	}
}

// TestLogTerminalFailure_ContextCanceledLogsDebugNotError pins the level
// distinction for a cancelled request: it is a request the caller or the
// SDK deliberately abandoned, not a server-reported failure, so it logs at
// DEBUG rather than ERROR.
func TestLogTerminalFailure_ContextCanceledLogsDebugNotError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := New(Config{
		BaseURL: "http://127.0.0.1:1", APIVersion: "v1", Token: "k",
		RetryCfg: retry.DefaultConfig(), Logger: logger, Debug: true,
	})

	client.logTerminalFailure(context.Background(), &sdkerrors.NetworkError{
		SupportContext: sdkerrors.SupportContext{Message: context.Canceled.Error()},
		Cause:          context.Canceled,
	})

	if !containsString(buf.String(), "level=DEBUG") {
		t.Errorf("a cancelled request should log at DEBUG: %q", buf.String())
	}
	if containsString(buf.String(), "level=ERROR") {
		t.Errorf("a cancelled request should not log at ERROR: %q", buf.String())
	}
}

// TestLogTerminalFailure_ContextCanceledSilentWithoutDebug is the property
// that actually matters to operators: with debug off (the default), a
// cancellation emits nothing at all. The cancel-on-first-error fan-outs
// cancel every sibling on the first failure, so without this one 401 in a
// 50-symbol batch would put ~49 lines on stderr with zero configuration.
func TestLogTerminalFailure_ContextCanceledSilentWithoutDebug(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := New(Config{
		BaseURL: "http://127.0.0.1:1", APIVersion: "v1", Token: "k",
		RetryCfg: retry.DefaultConfig(), Logger: logger,
	})

	client.logTerminalFailure(context.Background(), &sdkerrors.NetworkError{
		SupportContext: sdkerrors.SupportContext{Message: context.Canceled.Error()},
		Cause:          context.Canceled,
	})

	if buf.Len() != 0 {
		t.Errorf("a cancelled request should log nothing with debug off, got %q", buf.String())
	}
}

// TestLogTerminalFailure_DeadlineExceededStillLogsError guards the inverse
// for a STANDALONE request: a timeout the caller did not ask for stays at
// ERROR. The fan-out case is different and is covered by
// TestLogTerminalFailure_DeadlineInsideFanOutIsDebug — this test used to be
// the only one, and pinning the behavior at N=1 is exactly why the N-line
// fan-out noise went unnoticed.
func TestLogTerminalFailure_DeadlineExceededStillLogsError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := New(Config{
		BaseURL: "http://127.0.0.1:1", APIVersion: "v1", Token: "k",
		RetryCfg: retry.DefaultConfig(), Logger: logger,
	})

	client.logTerminalFailure(context.Background(), &sdkerrors.NetworkError{
		SupportContext: sdkerrors.SupportContext{Message: context.DeadlineExceeded.Error()},
		Cause:          context.DeadlineExceeded,
		Timeout:        true,
	})

	if !containsString(buf.String(), "level=ERROR") {
		t.Errorf("a deadline-exceeded request should still log at ERROR: %q", buf.String())
	}
}

// TestGet_CancelledContextDoesNotLogError drives the demotion through the
// real Get path rather than calling logTerminalFailure directly: the caller
// cancels, Get still returns the error, and nothing reaches the log.
func TestGet_CancelledContextDoesNotLogError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := New(Config{
		BaseURL: "http://127.0.0.1:1", APIVersion: "v1", Token: "k",
		RetryCfg: retry.DefaultConfig(), Logger: logger,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var out map[string]any
	_, err := client.Get(ctx, "stocks/quotes/", nil, &out)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Get() error = %v, want one wrapping context.Canceled", err)
	}
	if containsString(buf.String(), "level=ERROR") {
		t.Errorf("a caller's own cancellation should not log at ERROR: %q", buf.String())
	}
}

// TestLogTerminalFailure_DeadlineInsideFanOutIsDebug covers the case a
// single-request test cannot see: when the caller's own context expires,
// every sibling of a fan-out reports the same one expiry, so logging each at
// ERROR turns one logical failure into N lines.
func TestLogTerminalFailure_DeadlineInsideFanOutIsDebug(t *testing.T) {
	deadlineErr := &sdkerrors.NetworkError{
		SupportContext: sdkerrors.SupportContext{Message: context.DeadlineExceeded.Error()},
		Cause:          context.DeadlineExceeded,
		Timeout:        true,
	}

	newClient := func(buf *bytes.Buffer) *Client {
		return New(Config{
			BaseURL: "http://127.0.0.1:1", APIVersion: "v1", Token: "k",
			RetryCfg: retry.DefaultConfig(),
			Logger:   slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})),
		})
	}

	// Inside a fan-out: silent with debug off.
	var quiet bytes.Buffer
	var inFanOut context.Context
	_, _ = fanout.Run(context.Background(), 1, func(ctx context.Context, _ int) (int, error) {
		inFanOut = ctx
		return 0, nil
	})
	newClient(&quiet).logTerminalFailure(inFanOut, deadlineErr)
	if quiet.Len() != 0 {
		t.Errorf("a deadline inside a fan-out should not log with debug off, got %q", quiet.String())
	}

	// Outside a fan-out: still ERROR.
	var loud bytes.Buffer
	newClient(&loud).logTerminalFailure(context.Background(), deadlineErr)
	if !containsString(loud.String(), "level=ERROR") {
		t.Errorf("a standalone deadline should still log at ERROR: %q", loud.String())
	}

	// A real server answer inside a fan-out is not an echo and still logs.
	var apiErr bytes.Buffer
	newClient(&apiErr).logTerminalFailure(inFanOut, &sdkerrors.AuthenticationError{
		SupportContext: sdkerrors.SupportContext{Message: "unauthorized"},
	})
	if !containsString(apiErr.String(), "level=ERROR") {
		t.Errorf("a 401 inside a fan-out must still log at ERROR: %q", apiErr.String())
	}
}

// TestColumnsEnvelopeRepair covers both halves of the columns adjustment: a
// typed request gets the "s" envelope column back (the API drops it whenever
// a column filter is set, and every typed method checks it), and a formatted
// request does not, because there the column list is the user's output.
func TestColumnsEnvelopeRepair(t *testing.T) {
	// The envelope column is folded into a single comma-joined value, and a
	// multi-value columns default is joined with it rather than dropped.
	// The repeated form was tried first and is a no-op: verified against
	// production 2026-08-26, the API reads only the last occurrence of a
	// repeated key, so the prepended "s" never arrived.
	tests := []struct {
		name      string
		columns   []string
		formatted bool
		want      []string
	}{
		{"typed request gains the envelope", []string{"symbol,last"}, false, []string{"s,symbol,last"}},
		{"multi-value default survives", []string{"bid", "ask"}, false, []string{"s,bid,ask"}},
		{"already asked for, left alone", []string{"s,symbol"}, false, []string{"s,symbol"}},
		{"asked for in another position", []string{"symbol,s,last"}, false, []string{"symbol,s,last"}},
		{"whitespace around the name still counts", []string{"symbol, s ,last"}, false, []string{"symbol, s ,last"}},
		{"formatted request is untouched", []string{"symbol,last"}, true, []string{"symbol,last"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.URL.Query()["columns"]
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"s":"ok"}`))
			}))
			defer server.Close()

			client := New(Config{
				BaseURL: server.URL, APIVersion: "v1", Token: "k",
				RetryCfg: retry.DefaultConfig(),
			})
			params := url.Values{"columns": tt.columns}

			var err error
			if tt.formatted {
				_, err = client.GetFormatted(context.Background(), "stocks/quotes/", params, "csv")
			} else {
				var out map[string]any
				_, err = client.Get(context.Background(), "stocks/quotes/", params, &out)
			}
			if err != nil {
				t.Fatalf("request error = %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("columns on the wire = %v, want %v", got, tt.want)
			}
		})
	}
}

// redcolumnsDest is a destination that declares decoder-required columns,
// standing in for the wire response types in the resource packages.
type redcolumnsDest struct {
	Status string   `json:"s"`
	Symbol []string `json:"symbol"`
	needs  []string
}

func (d *redcolumnsDest) RequiredColumns() []string { return d.needs }

// TestColumnsRepairAddsDecoderRequiredColumns covers the second half of the
// columns repair: beyond the "s" envelope, the destination's own row-count
// column is asked back. Without it a filter like "bid,ask" returns a body
// whose symbol array is absent, the conversion reads zero rows, and the
// typed method reports not-found for a quote that is present and billed.
func TestColumnsRepairAddsDecoderRequiredColumns(t *testing.T) {
	tests := []struct {
		name    string
		columns []string
		needs   []string
		want    []string
	}{
		{"row-count column added after the envelope", []string{"bid,ask"}, []string{"symbol"}, []string{"s,symbol,bid,ask"}},
		{"a multi-value filter is joined with both", []string{"bid", "ask"}, []string{"symbol"}, []string{"s,symbol,bid,ask"}},
		{"already named, not repeated", []string{"symbol,bid"}, []string{"symbol"}, []string{"s,symbol,bid"}},
		{"a destination needing nothing gets the envelope only", []string{"bid,ask"}, nil, []string{"s,bid,ask"}},
		{"a duplicate requirement is named once", []string{"bid"}, []string{"symbol", "symbol"}, []string{"s,symbol,bid"}},
		{"nothing to add leaves the filter untouched", []string{"s,symbol,bid"}, []string{"symbol"}, []string{"s,symbol,bid"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.URL.Query()["columns"]
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"s":"ok","symbol":["AAPL"]}`))
			}))
			defer server.Close()

			client := New(Config{
				BaseURL: server.URL, APIVersion: "v1", Token: "k",
				RetryCfg: retry.DefaultConfig(),
			})

			dest := &redcolumnsDest{needs: tt.needs}
			if _, err := client.Get(context.Background(), "stocks/quotes/", url.Values{"columns": tt.columns}, dest); err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("columns on the wire = %v, want %v", got, tt.want)
			}
			if len(dest.Symbol) != 1 {
				t.Errorf("the destination decoded %d rows; the repair exists so this is never zero", len(dest.Symbol))
			}
		})
	}
}

// TestGetFormatted_MergesFormatOnlyParams verifies the params that only cohere
// with a raw body reach the formatted path — and that an explicit method-level
// value still wins, matching how Do treats the request defaults.
func TestGetFormatted_MergesFormatOnlyParams(t *testing.T) {
	var got url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("a,b\n1,2\n"))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "k",
		RetryCfg:         retry.DefaultConfig(),
		FormatOnlyParams: url.Values{"human": {"true"}},
	})

	if _, err := client.GetFormatted(context.Background(), "stocks/quotes/", nil, "csv"); err != nil {
		t.Fatalf("GetFormatted() error = %v", err)
	}
	if got.Get("human") != "true" {
		t.Errorf("human = %q, want true on the formatted path", got.Get("human"))
	}

	caller := url.Values{"human": {"false"}}
	if _, err := client.GetFormatted(context.Background(), "stocks/quotes/", caller, "csv"); err != nil {
		t.Fatalf("GetFormatted() error = %v", err)
	}
	if got.Get("human") != "false" {
		t.Errorf("human = %q, want the caller's explicit false to win", got.Get("human"))
	}
}

// TestPolicyConstants pins the configuration values that encode a policy
// decision rather than an implementation detail.
//
// Every one of these is executed by the suite, so they were all inside the
// 100% statement-coverage figure — but their VALUES were compared nowhere,
// and a mutation battery confirmed it: changing DefaultPoolSize to 5000,
// defaultMaxResponseBytes to 100 GiB and maxRedirects to 100 all left the
// suite green. Coverage measures which lines ran, not which decisions are
// still the ones we made.
func TestPolicyConstants(t *testing.T) {
	// ADR-014: one global pool bounds concurrent in-flight requests.
	if DefaultPoolSize != 50 {
		t.Errorf("DefaultPoolSize = %d, want 50 (ADR-014's global concurrency contract)", DefaultPoolSize)
	}
	// Memory-exhaustion ceiling: a hostile or broken server streaming an
	// unbounded body must not be able to take the caller down.
	if defaultMaxResponseBytes != 100<<20 {
		t.Errorf("defaultMaxResponseBytes = %d, want 100 MiB", defaultMaxResponseBytes)
	}
	// Redirect-loop bound.
	if maxRedirects != 10 {
		t.Errorf("maxRedirects = %d, want 10", maxRedirects)
	}
}

// TestTransportPolicy pins the settings of the transport the SDK builds for
// itself. A caller-supplied transport keeps its own (see WithHTTPClient).
func TestTransportPolicy(t *testing.T) {
	client := New(Config{
		BaseURL: "https://api.example.test", APIVersion: "v1", Token: "k",
		RetryCfg: retry.DefaultConfig(),
	})
	transport, ok := client.http.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Transport = %T, want *http.Transport", client.http.Transport)
	}

	// Idle connections per host is deliberately below DefaultPoolSize: the
	// two numbers are in tension — a 50-way fan-out retires roughly 40
	// connections per batch — and this test exists so changing either is a
	// visible decision rather than a silent one.
	if transport.MaxIdleConnsPerHost != 10 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 10", transport.MaxIdleConnsPerHost)
	}
	if transport.MaxIdleConns != 100 {
		t.Errorf("MaxIdleConns = %d, want 100", transport.MaxIdleConns)
	}
	if transport.TLSHandshakeTimeout != 10*time.Second {
		t.Errorf("TLSHandshakeTimeout = %v, want 10s", transport.TLSHandshakeTimeout)
	}
	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("IdleConnTimeout = %v, want 90s", transport.IdleConnTimeout)
	}
	if !transport.ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2 = false, want true")
	}
}

// TestTruncatedBodyIsTypedAndRetried covers the failure that used to escape
// the taxonomy: a server that announces a Content-Length and then drops the
// connection. It returned a bare fmt.Errorf, so errors.As missed it, it
// carried no support context, and retry.ShouldRetryError said no — leaving
// the one transient class retries exist for unretried.
func TestTruncatedBodyIsTypedAndRetried(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			// Promise more than we send, then hang up mid-body.
			w.Header().Set("Content-Length", "512")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"s":"ok"`))
			if hj, ok := w.(http.Hijacker); ok {
				conn, _, _ := hj.Hijack()
				_ = conn.Close()
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok"}`))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "k",
		RetryCfg: retry.Config{MaxRetries: 2, InitialBackoff: time.Millisecond, Multiplier: 2},
	})

	var out map[string]any
	_, err := client.Get(context.Background(), "stocks/quotes/", nil, &out)
	if err != nil {
		t.Fatalf("Get() error = %v, want the retry to recover", err)
	}
	if got := attempts.Load(); got < 2 {
		t.Errorf("attempts = %d, want at least 2 — the truncated body was not retried", got)
	}
}

// TestTruncatedBodyErrorIsInTheTaxonomy pins the classification directly,
// without depending on whether a retry happens to recover.
func TestTruncatedBodyErrorIsInTheTaxonomy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "512")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"s":"ok"`))
		if hj, ok := w.(http.Hijacker); ok {
			conn, _, _ := hj.Hijack()
			_ = conn.Close()
		}
	}))
	defer server.Close()

	client := New(Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "k",
		RetryCfg: retry.Config{MaxRetries: 0},
	})

	var out map[string]any
	_, err := client.Get(context.Background(), "stocks/quotes/", nil, &out)
	if err == nil {
		t.Fatal("Get() error = nil, want a network error")
	}

	var netErr *sdkerrors.NetworkError
	if !errors.As(err, &netErr) {
		t.Errorf("errors.As(*NetworkError) = false for %v", err)
	}
	var sdkErr sdkerrors.Error
	if !errors.As(err, &sdkErr) {
		t.Errorf("errors.As(sdkerrors.Error) = false: the error carries no support context")
	}
	if !retry.ShouldRetryError(err) {
		t.Errorf("ShouldRetryError = false for a truncated body, want true")
	}
}

// TestErrorRequestURLIncludesMergedDefaults pins that the URL in an error's
// support context is the request that was actually sent. It used to be
// rebuilt from the caller's own params, which Do never writes to, so the
// client's universal defaults — the parameters most likely to explain the
// failure — were missing from the exact string meant to be pasted into a
// support ticket.
func TestErrorRequestURLIncludesMergedDefaults(t *testing.T) {
	var wire string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wire = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"s":"error","errmsg":"bad"}`))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "k",
		RetryCfg: retry.DefaultConfig(),
		DefaultParams: url.Values{
			"mode":  {"cached"},
			"limit": {"10"},
		},
	})

	var out map[string]any
	_, err := client.Get(context.Background(), "stocks/quotes/", url.Values{"symbols": {"AAPL"}}, &out)
	if err == nil {
		t.Fatal("Get() error = nil, want the 400")
	}

	var sdkErr sdkerrors.Error
	if !errors.As(err, &sdkErr) {
		t.Fatalf("errors.As(sdkerrors.Error) = false for %v", err)
	}
	info := sdkErr.SupportInfo()
	for _, want := range []string{"mode=cached", "limit=10", "symbols=AAPL"} {
		if !containsString(info, want) {
			t.Errorf("support info is missing %q; the wire query was %q\n%s", want, wire, info)
		}
	}
}

// TestResponse_wireURL_Fallback covers the defensive path: a Response with
// no raw request (hand-built in a test, or a failure before the request was
// issued) falls back to the rebuilt URL rather than reporting an empty one.
func TestResponse_wireURL_Fallback(t *testing.T) {
	const fallback = "https://api.example.test/v1/stocks/quotes/?symbols=AAPL"

	if got := (*Response)(nil).wireURL(fallback); got != fallback {
		t.Errorf("nil Response wireURL = %q, want the fallback", got)
	}
	if got := (&Response{}).wireURL(fallback); got != fallback {
		t.Errorf("Response with no Raw wireURL = %q, want the fallback", got)
	}
	if got := (&Response{Raw: &http.Response{}}).wireURL(fallback); got != fallback {
		t.Errorf("Response with no Raw.Request wireURL = %q, want the fallback", got)
	}
}

// TestRetriesReserveEveryAttempt pins that the pre-flight reservation is per
// attempt, not per call. Every attempt is a real, billed request, so
// reserving once outside the retry loop under-counted actual spend by up to
// MaxRetries+1 — one caller-visible call could bill four requests against a
// single reserved credit.
func TestRetriesReserveEveryAttempt(t *testing.T) {
	var billed atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		billed.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(503)
		_, _ = w.Write([]byte(`{"s":"error","errmsg":"unavailable"}`))
	}))
	defer server.Close()

	tracker := ratelimit.New()
	// A budget big enough that the pre-flight never rejects; we are counting
	// reservations, not testing rejection.
	tracker.Update(&http.Response{Header: http.Header{
		"X-Api-Ratelimit-Limit":     []string{"1000"},
		"X-Api-Ratelimit-Remaining": []string{"1000"},
		"X-Api-Ratelimit-Reset":     []string{fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix())},
	}})

	client := New(Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "k",
		RetryCfg:   retry.Config{MaxRetries: 3, InitialBackoff: time.Millisecond, Multiplier: 1},
		RateLimits: tracker,
	})

	var out map[string]any
	_, _ = client.Get(context.Background(), "stocks/quotes/", nil, &out)

	// Four billed requests (1 + 3 retries) means four reservations were
	// taken and released; a reservation leak would leave the tracker holding
	// some, which State() would show.
	if got := billed.Load(); got != 4 {
		t.Fatalf("billed requests = %d, want 4", got)
	}
	if got := tracker.State().Remaining; got == 0 {
		t.Errorf("tracker remaining = 0; reservations were not released")
	}
	// Every reservation must have been released, so a fresh Reserve succeeds
	// even at the boundary.
	if !tracker.Reserve() {
		t.Error("Reserve() = false after the retries completed; reservations leaked")
	}
	tracker.Release()
}

// TestRetryStopsWhenTheBudgetRunsOut verifies that exhausting the budget
// mid-retry hands back the failure already in hand rather than replacing it
// with a pre-flight rejection whose cause the caller never saw.
func TestRetryStopsWhenTheBudgetRunsOut(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Api-Ratelimit-Limit", "1")
		w.Header().Set("X-Api-Ratelimit-Remaining", "0")
		w.Header().Set("X-Api-Ratelimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()))
		w.WriteHeader(503)
		_, _ = w.Write([]byte(`{"s":"error","errmsg":"unavailable"}`))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "k",
		RetryCfg:   retry.Config{MaxRetries: 3, InitialBackoff: time.Millisecond, Multiplier: 1},
		RateLimits: ratelimit.New(),
	})

	var out map[string]any
	resp, err := client.Get(context.Background(), "stocks/quotes/", nil, &out)

	// The first response set remaining=0, so the second attempt cannot
	// reserve: the loop stops with the 503 already in hand.
	if got := hits.Load(); got != 1 {
		t.Errorf("requests = %d, want 1 — the retry should not have been attempted", got)
	}
	if err == nil && resp == nil {
		t.Error("Get() returned neither the original failure nor a response")
	}
	var rlErr *sdkerrors.RateLimitError
	if errors.As(err, &rlErr) && rlErr.PreFlight {
		t.Error("Get() replaced the original 503 with a pre-flight rejection")
	}
}
