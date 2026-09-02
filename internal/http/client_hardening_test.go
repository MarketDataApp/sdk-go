package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/ratelimit"
	"github.com/MarketDataApp/sdk-go/v2/internal/retry"
	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

func TestClient_Do_RetryAfterHonored(t *testing.T) {
	attempts := 0
	var timestamps []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		timestamps = append(timestamps, time.Now())
		if attempts < 2 {
			w.Header().Set("Retry-After", "1")
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
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	// The server-supplied 1s Retry-After must override the 1ms calculated backoff
	if gap := timestamps[1].Sub(timestamps[0]); gap < 900*time.Millisecond {
		t.Errorf("retry gap = %v, want >= ~1s (Retry-After honored)", gap)
	}
}

func TestClient_Do_RetryAfterCapped(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			// A hostile Retry-After must not park the retry loop
			w.Header().Set("Retry-After", "9999999999")
			w.WriteHeader(http.StatusServiceUnavailable)
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

	start := time.Now()
	var result map[string]string
	_, err := client.Get(context.Background(), "/test/", nil, &result)

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	// Above the cap the calculated backoff (1ms) applies instead
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("elapsed = %v, want fast retry (hostile Retry-After ignored)", elapsed)
	}
}

func TestParseAPIError_SanitizesMessage(t *testing.T) {
	resp := &Response{
		StatusCode: 400,
		Headers:    http.Header{},
		Body:       []byte("{\"s\":\"error\",\"errmsg\":\"bad\\nrequest\\r\x1b[31minjected\"}"),
	}

	err := parseAPIError(resp, "https://example.com/v1/test/")
	var badReq *sdkerrors.BadRequestError
	if !errors.As(err, &badReq) {
		t.Fatalf("error type = %T, want *BadRequestError", err)
	}
	msg := badReq.Message
	if strings.ContainsAny(msg, "\n\r\x1b") {
		t.Errorf("Message = %q, control characters not sanitized", msg)
	}
}

func TestClient_Do_PreflightReservesCredits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
	defer server.Close()

	tracker := ratelimit.New()
	tracker.Update(&http.Response{
		Header: http.Header{
			"X-Api-Ratelimit-Remaining": []string{"0"},
			"X-Api-Ratelimit-Reset":     []string{formatUnixTS(time.Now().Add(time.Hour))},
		},
	})

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.Config{MaxRetries: 0, InitialBackoff: time.Millisecond, Multiplier: 2.0},
		RateLimits: tracker,
	})

	_, err := client.Get(context.Background(), "/test/", nil, nil)
	var rle *sdkerrors.RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("error = %v, want *RateLimitError from pre-flight reservation", err)
	}
}

func formatUnixTS(t time.Time) string {
	return strconv.FormatInt(t.Unix(), 10)
}

func TestClient_Do_SemaphoreAcquired(t *testing.T) {
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
		Sem:        make(chan struct{}, 1),
	})

	if _, err := client.Get(context.Background(), "/test/", nil, nil); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

// demoHeadersServer returns a test server whose responses carry the rate-limit
// headers the live API sends on anonymous (demo) requests: limit=0 and
// remaining=0 with a reset a day away.
func demoHeadersServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Api-Ratelimit-Limit", "0")
		w.Header().Set("X-Api-Ratelimit-Remaining", "0")
		w.Header().Set("X-Api-Ratelimit-Consumed", "0")
		w.Header().Set("X-Api-Ratelimit-Reset", formatUnixTS(time.Now().Add(24*time.Hour)))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok"})
	}))
}

// TestClient_Do_DemoModeNeverPreflightBlocked reproduces QA finding C-1: after
// the first anonymous call records the API's limit=0/remaining=0 demo headers,
// every subsequent call was rejected by the pre-flight check without touching
// the network. Demo mode must stay usable for the life of the process.
func TestClient_Do_DemoModeNeverPreflightBlocked(t *testing.T) {
	server := demoHeadersServer()
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		DemoMode:   true,
		RetryCfg:   retry.Config{MaxRetries: 0, InitialBackoff: time.Millisecond, Multiplier: 2.0},
		RateLimits: ratelimit.New(),
	})

	for i := 0; i < 3; i++ {
		if _, err := client.Get(context.Background(), "/test/", nil, nil); err != nil {
			t.Fatalf("Get() call %d error = %v, want success (demo mode must not be pre-flight blocked)", i+1, err)
		}
	}
}

// TestClient_Do_DemoModeConcurrent exercises the same scenario under
// concurrency: parallel demo-mode callers must all succeed while the tracker
// keeps absorbing limit=0 snapshots.
func TestClient_Do_DemoModeConcurrent(t *testing.T) {
	server := demoHeadersServer()
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		DemoMode:   true,
		RetryCfg:   retry.Config{MaxRetries: 0, InitialBackoff: time.Millisecond, Multiplier: 2.0},
		RateLimits: ratelimit.New(),
		Sem:        make(chan struct{}, 50),
	})

	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := client.Get(context.Background(), "/test/", nil, nil); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent demo Get() error = %v, want success", err)
	}
}

// TestClient_Do_DeadlineDuringBackoffIsNetworkError pins the context-error
// taxonomy (QA finding C-3): wherever the context fires — pool wait, backoff
// sleep, or mid-request — Do returns a *NetworkError that unwraps to the
// context error, with Timeout set for deadlines. A deadline expiring during
// the retry backoff previously escaped as a bare context.DeadlineExceeded.
func TestClient_Do_DeadlineDuringBackoffIsNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable) // 503 - retryable
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg: retry.Config{
			MaxRetries:     3,
			InitialBackoff: time.Second,
			Multiplier:     2.0,
		},
		RateLimits: ratelimit.New(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.Get(ctx, "/test/", nil, nil)

	var netErr *sdkerrors.NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("Get() error = %T (%v), want *NetworkError wrapping the deadline error", err, err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Error("errors.Is(err, context.DeadlineExceeded) = false, want true (Cause must unwrap)")
	}
	if !netErr.Timeout {
		t.Error("Timeout = false, want true for a deadline error")
	}
}
