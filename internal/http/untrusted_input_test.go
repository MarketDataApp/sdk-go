package http

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/MarketDataApp/sdk-go/v2/internal/ratelimit"
	"github.com/MarketDataApp/sdk-go/v2/internal/retry"
)

// TestUntrustedSymbol_EndToEnd drives hostile, attacker-controlled symbols
// through the real request-building path (PathSegment + buildURL +
// http.NewRequestWithContext) and asserts the outgoing request is safely
// encoded: no CRLF/header injection, no path escape or added segments, no
// query smuggling, and no control characters on the wire.
func TestUntrustedSymbol_EndToEnd(t *testing.T) {
	rt := &recordingTransport{}
	client := New(Config{
		HTTPClient: &http.Client{Transport: rt},
		BaseURL:    "https://api.example.test",
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.Config{MaxRetries: 0, InitialBackoff: 1, Multiplier: 2.0},
		RateLimits: ratelimit.New(),
	})

	hostile := []string{
		"AAPL\r\nX-Injected: evil",       // CRLF header injection
		"AAPL/../../../etc/passwd",       // path traversal
		"AAPL?admin=true&columns=secret", // query smuggling
		"AAPL#fragment",                  // fragment injection
		"AAPL\x00null",                   // NUL byte
		"../..",                          // pure traversal
		"日本語\u202eRTL",                   // Unicode incl. RTL override
		"AAPL%2F%00",                     // pre-encoded payload
		strings.Repeat("A", 50000),       // oversized (must not hang/panic)
	}

	for _, sym := range hostile {
		t.Run(strings.Map(func(r rune) rune {
			if r < 0x20 || r > 0x7e {
				return '.'
			}
			return r
		}, sym[:min(len(sym), 24)]), func(t *testing.T) {
			// The canonical way the services build a symbol path.
			path := "stocks/quotes/" + PathSegment(sym) + "/"
			if _, err := client.Get(context.Background(), path, nil, nil); err != nil {
				t.Fatalf("Get() error = %v (must not fail on hostile input)", err)
			}
			req := rt.lastReq
			if req == nil {
				t.Fatal("no request captured")
			}

			full := req.URL.String()
			// No raw control characters anywhere in the URL.
			for _, bad := range []string{"\r", "\n", "\x00"} {
				if strings.Contains(full, bad) {
					t.Errorf("URL contains raw control char %q: %q", bad, full)
				}
			}
			// No query was smuggled from a "?" in the symbol.
			if req.URL.RawQuery != "" {
				t.Errorf("smuggled query %q from symbol %q", req.URL.RawQuery, sym)
			}
			// The hostile segment must not add path segments: the fixed path
			// "/v1/stocks/quotes/<sym>/" has exactly 5 slashes (leading, v1,
			// stocks, quotes, trailing) regardless of the symbol content.
			if got := strings.Count(req.URL.EscapedPath(), "/"); got != 5 {
				t.Errorf("path escaped to %d segments (want 5): %q", got, req.URL.EscapedPath())
			}
			// No header was injected via CRLF.
			if v := req.Header.Get("X-Injected"); v != "" {
				t.Errorf("header injection succeeded: X-Injected=%q", v)
			}
			for k := range req.Header {
				switch k {
				case "Authorization", "User-Agent", "Accept":
				default:
					t.Errorf("unexpected header on request: %q", k)
				}
			}
		})
	}
}
