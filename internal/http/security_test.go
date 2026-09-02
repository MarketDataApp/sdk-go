package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/MarketDataApp/sdk-go/v2/internal/ratelimit"
	"github.com/MarketDataApp/sdk-go/v2/internal/retry"
	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

// recordingTransport captures the outgoing request and returns a canned
// response, letting tests inspect exactly what the SDK put on the wire without
// a live server (needed to assert scheme-dependent header behavior).
type recordingTransport struct {
	mu      sync.Mutex
	lastReq *http.Request
	status  int
	body    string
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.mu.Lock()
	rt.lastReq = req.Clone(req.Context())
	rt.mu.Unlock()
	st := rt.status
	if st == 0 {
		st = 200
	}
	body := rt.body
	if body == "" {
		body = `{"s":"ok"}`
	}
	return &http.Response{
		StatusCode: st,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

func (rt *recordingTransport) authHeader() string {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.lastReq == nil {
		return "<no request>"
	}
	return rt.lastReq.Header.Get("Authorization")
}

func newRecordingClient(baseURL, token string, demo bool) (*Client, *recordingTransport) {
	rt := &recordingTransport{}
	c := New(Config{
		HTTPClient: &http.Client{Transport: rt},
		BaseURL:    baseURL,
		APIVersion: "v1",
		Token:      token,
		DemoMode:   demo,
		RetryCfg:   retry.Config{MaxRetries: 0, InitialBackoff: 1, Multiplier: 2.0},
		RateLimits: ratelimit.New(),
	})
	return c, rt
}

// TestResponseSizeCap proves a hostile oversized body is refused instead of
// being read into memory.
func TestResponseSizeCap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(bytes.Repeat([]byte("a"), 4096)) // far larger than the 16-byte cap below
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:      server.URL,
		APIVersion:   "v1",
		Token:        "test-key",
		RetryCfg:     retry.Config{MaxRetries: 0, InitialBackoff: 1, Multiplier: 2.0},
		RateLimits:   ratelimit.New(),
		MaxRespBytes: 16,
	})

	_, err := client.Get(context.Background(), "/test/", nil, nil)
	var tooLarge *sdkerrors.ResponseTooLargeError
	if !errors.As(err, &tooLarge) {
		t.Fatalf("error = %v (%T), want *ResponseTooLargeError", err, err)
	}
	if !errors.Is(err, sdkerrors.ErrResponseTooLarge) {
		t.Errorf("errors.Is(err, ErrResponseTooLarge) = false")
	}
	if tooLarge.Limit != 16 {
		t.Errorf("Limit = %d, want 16", tooLarge.Limit)
	}
}

// TestResponseSizeCap_UnderLimit proves a body at/below the cap is read fine.
func TestResponseSizeCap_UnderLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok"}`))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:      server.URL,
		APIVersion:   "v1",
		Token:        "test-key",
		RetryCfg:     retry.Config{MaxRetries: 0, InitialBackoff: 1, Multiplier: 2.0},
		RateLimits:   ratelimit.New(),
		MaxRespBytes: 1024,
	})
	var out map[string]string
	if _, err := client.Get(context.Background(), "/test/", nil, &out); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if out["s"] != "ok" {
		t.Errorf("s = %q, want ok", out["s"])
	}
}

// TestToken_SetOverHTTPS proves the bearer token is attached on an https base.
func TestToken_SetOverHTTPS(t *testing.T) {
	c, rt := newRecordingClient("https://api.example.com", "SECRET-TOKEN-XYZ", false)
	if _, err := c.Get(context.Background(), "/test/", nil, nil); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got := rt.authHeader(); got != "Bearer SECRET-TOKEN-XYZ" {
		t.Errorf("Authorization = %q, want bearer token set over https", got)
	}
}

// TestToken_RefusedOverInsecureHTTP proves the token is NOT shipped to a
// plain-http, non-loopback host — the request fails before it is sent.
func TestToken_RefusedOverInsecureHTTP(t *testing.T) {
	c, rt := newRecordingClient("http://api.example.com", "SECRET-TOKEN-XYZ", false)
	_, err := c.Get(context.Background(), "/test/", nil, nil)
	var insecure *sdkerrors.InsecureTokenError
	if !errors.As(err, &insecure) {
		t.Fatalf("error = %v (%T), want *InsecureTokenError", err, err)
	}
	if !errors.Is(err, sdkerrors.ErrInsecureToken) {
		t.Errorf("errors.Is(err, ErrInsecureToken) = false")
	}
	// The request must never have reached the transport with the token.
	if strings.Contains(rt.authHeader(), "SECRET-TOKEN-XYZ") {
		t.Errorf("token leaked to transport over insecure http: %q", rt.authHeader())
	}
}

// TestDemoMode_NoAuthOverInsecure proves demo mode (no token) is allowed over
// plain http and never sets an Authorization header.
func TestDemoMode_NoAuthOverInsecure(t *testing.T) {
	c, rt := newRecordingClient("http://api.example.com", "", true)
	if _, err := c.Get(context.Background(), "/test/", nil, nil); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if h := rt.authHeader(); h != "" {
		t.Errorf("Authorization = %q, want empty in demo mode", h)
	}
}

// TestCrossHostRedirectRefused proves a redirect to a different host is refused
// so the token can never follow a redirect to an unintended origin.
func TestCrossHostRedirectRefused(t *testing.T) {
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"s":"ok"}`))
	}))
	defer other.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+"/leak/", http.StatusFound)
	}))
	defer redirector.Close()

	client := New(Config{
		BaseURL:    redirector.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.Config{MaxRetries: 0, InitialBackoff: 1, Multiplier: 2.0},
		RateLimits: ratelimit.New(),
	})
	_, err := client.Get(context.Background(), "/test/", nil, nil)
	if err == nil {
		t.Fatal("expected error on cross-host redirect, got nil")
	}
	if !strings.Contains(err.Error(), "cross-host redirect") {
		t.Errorf("error = %v, want it to mention cross-host redirect", err)
	}
}

// TestGet_204NoContent proves a 204 (mode=cached cache miss, empty body) is
// returned without an error and without a spurious decode failure.
func TestGet_204NoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent) // 204, empty body
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.Config{MaxRetries: 0, InitialBackoff: 1, Multiplier: 2.0},
		RateLimits: ratelimit.New(),
	})
	var out map[string]any
	resp, err := client.Get(context.Background(), "/test/", nil, &out)
	if err != nil {
		t.Fatalf("Get() on 204 returned error = %v, want nil (no decode of empty body)", err)
	}
	if resp.StatusCode != 204 {
		t.Errorf("StatusCode = %d, want 204", resp.StatusCode)
	}
}

func TestTokenSafeForURL(t *testing.T) {
	cases := []struct {
		raw  string
		want bool
	}{
		{"https://api.marketdata.app", true},
		{"HTTPS://api.marketdata.app", true},
		{"http://localhost:8080", true},
		{"http://127.0.0.1:8080", true},
		{"http://[::1]:8080", true},
		{"http://api.marketdata.app", false},
		{"http://192.168.1.10", false},
		{"http://evil.example.com", false},
	}
	for _, tc := range cases {
		u, err := url.Parse(tc.raw)
		if err != nil {
			t.Fatalf("parse %q: %v", tc.raw, err)
		}
		if got := tokenSafeForURL(u); got != tc.want {
			t.Errorf("tokenSafeForURL(%q) = %v, want %v", tc.raw, got, tc.want)
		}
	}
	if tokenSafeForURL(nil) {
		t.Error("tokenSafeForURL(nil) = true, want false")
	}
}

func TestSecureCheckRedirect(t *testing.T) {
	mk := func(host string) *http.Request {
		return &http.Request{URL: &url.URL{Scheme: "https", Host: host}}
	}
	// No prior hops: allowed.
	if err := secureCheckRedirect(mk("a.example"), nil); err != nil {
		t.Errorf("first hop should be allowed, got %v", err)
	}
	// Same host: allowed.
	via := []*http.Request{mk("a.example")}
	if err := secureCheckRedirect(mk("a.example"), via); err != nil {
		t.Errorf("same-host redirect should be allowed, got %v", err)
	}
	// Different host: refused.
	if err := secureCheckRedirect(mk("b.example"), via); err == nil {
		t.Error("cross-host redirect should be refused")
	}
	// Too many hops: refused.
	long := make([]*http.Request, maxRedirects)
	for i := range long {
		long[i] = mk("a.example")
	}
	if err := secureCheckRedirect(mk("a.example"), long); err == nil {
		t.Error("exceeding max redirects should be refused")
	}
}

// TestToken_NeverLeaks drives a success path and an error path with debug
// logging enabled, then scans all captured output and error strings for the
// token value — there must be zero hits.
func TestToken_NeverLeaks(t *testing.T) {
	const secret = "tok_SUPERSECRET_9f8e7d6c5b4a"

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Server echoes the token if it were ever in the URL/query (it must not be)
	// and returns a 401 error body to exercise the error path.
	var sawTokenServerSide bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), secret) {
			sawTokenServerSide = true
		}
		if r.URL.Path == "/v1/err/" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"s":"error","errmsg":"invalid token"}`))
			return
		}
		_, _ = w.Write([]byte(`{"s":"ok"}`))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      secret,
		Debug:      true,
		Logger:     logger,
		RetryCfg:   retry.Config{MaxRetries: 0, InitialBackoff: 1, Multiplier: 2.0},
		RateLimits: ratelimit.New(),
	})

	// Success path.
	if _, err := client.Get(context.Background(), "/ok/", nil, nil); err != nil {
		t.Fatalf("Get() ok path error = %v", err)
	}
	// Error path (401) — capture the error's full string forms.
	_, err := client.Get(context.Background(), "/err/", nil, nil)
	if err == nil {
		t.Fatal("expected 401 error")
	}

	haystacks := map[string]string{
		"debug logs": logBuf.String(),
		"error()":    err.Error(),
	}
	if se, ok := err.(sdkerrors.Error); ok {
		haystacks["supportinfo"] = se.SupportInfo()
	}
	for name, h := range haystacks {
		if strings.Contains(h, secret) {
			t.Errorf("token leaked in %s: %q", name, h)
		}
	}
	if sawTokenServerSide {
		t.Error("token appeared in request URL/query server-side (must be header-only)")
	}
}
