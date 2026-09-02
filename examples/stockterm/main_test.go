package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// withArgs runs fn with os.Args set to argv (args[0] is the program name),
// restoring the original os.Args afterward.
func withArgs(t *testing.T, argv []string, fn func()) {
	t.Helper()
	orig := os.Args
	os.Args = argv
	defer func() { os.Args = orig }()
	fn()
}

func TestParseFlags_Defaults(t *testing.T) {
	withArgs(t, []string{"stockterm"}, func() {
		cfg := parseFlags()

		wantSymbols := []string{"AAPL", "MSFT", "META", "SPY", "VFINX"}
		if !reflect.DeepEqual(cfg.symbols, wantSymbols) {
			t.Errorf("symbols = %v, want %v", cfg.symbols, wantSymbols)
		}

		wantFunds := map[string]bool{"VFINX": true}
		if !reflect.DeepEqual(cfg.funds, wantFunds) {
			t.Errorf("funds = %v, want %v", cfg.funds, wantFunds)
		}

		if cfg.refresh != 5*time.Second {
			t.Errorf("refresh = %v, want 5s", cfg.refresh)
		}
		if cfg.usePrices {
			t.Error("usePrices = true, want false")
		}
		if cfg.once {
			t.Error("once = true, want false")
		}
		if cfg.baseURL != "" {
			t.Errorf("baseURL = %q, want empty", cfg.baseURL)
		}
	})
}

func TestParseFlags_FundsSplitsCommaList(t *testing.T) {
	withArgs(t, []string{"stockterm", "-funds", "VFINX,FXAIX,vtsax"}, func() {
		cfg := parseFlags()

		want := map[string]bool{"VFINX": true, "FXAIX": true, "VTSAX": true}
		if !reflect.DeepEqual(cfg.funds, want) {
			t.Errorf("funds = %v, want %v", cfg.funds, want)
		}
	})
}

func TestParseFlags_PositionalArgsReplaceDefaultWatchlist(t *testing.T) {
	withArgs(t, []string{"stockterm", "TSLA", "NVDA"}, func() {
		cfg := parseFlags()

		want := []string{"TSLA", "NVDA"}
		if !reflect.DeepEqual(cfg.symbols, want) {
			t.Errorf("symbols = %v, want %v", cfg.symbols, want)
		}
	})
}

func TestParseFlags_RefreshParses(t *testing.T) {
	withArgs(t, []string{"stockterm", "-refresh", "10s"}, func() {
		cfg := parseFlags()

		if cfg.refresh != 10*time.Second {
			t.Errorf("refresh = %v, want 10s", cfg.refresh)
		}
	})
}

func TestParseFlags_PricesAndOnceAndBaseURL(t *testing.T) {
	withArgs(t, []string{"stockterm", "-prices", "-once", "-base-url", "http://127.0.0.1:9999"}, func() {
		cfg := parseFlags()

		if !cfg.usePrices {
			t.Error("usePrices = false, want true")
		}
		if !cfg.once {
			t.Error("once = false, want true")
		}
		if cfg.baseURL != "http://127.0.0.1:9999" {
			t.Errorf("baseURL = %q, want http://127.0.0.1:9999", cfg.baseURL)
		}
	})
}

// --- runOnce ---
//
// These tests exercise runOnce exactly as -once does in production: they
// call newClient(cfg) (via runOnce) rather than the test-only
// newTestClient helper, and they never set MARKETDATA_TOKEN, so the
// client resolves no token from the environment and runs in demo mode
// (mirroring the grader's "token-less CWD" scenario in contracts.md).
// That's fine for these two tests: AAPL, the sole demo-mode symbol, is
// present in every route fullRoutes() serves, so every fetch still
// succeeds and the frame still carries real mock values. See
// TestRunOnce_FullFetchSurface_NonDemo below for a test that forces a
// token so the complete 13-command fetch surface (including fund
// candles and /user/) actually runs.

// onceCfg builds the appConfig runOnce sees for a -base-url run: the
// default watchlist and funds set, pointed at srv.
func onceCfg(srv string) appConfig {
	return appConfig{
		symbols: append([]string(nil), defaultSymbols...),
		funds:   map[string]bool{defaultFunds: true},
		refresh: defaultRefresh,
		once:    true,
		baseURL: srv,
	}
}

func TestRunOnce_Success(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()

	var out bytes.Buffer
	code := runOnce(onceCfg(srv.URL), &out)

	if code != 0 {
		t.Fatalf("runOnce() = %d, want 0; output:\n%s", code, out.String())
	}
	got := out.String()
	if !strings.Contains(got, "AAPL") {
		t.Errorf("output does not contain AAPL:\n%s", got)
	}
	if !strings.Contains(got, "150.22") {
		t.Errorf("output does not contain mock quote value 150.22:\n%s", got)
	}
	if !strings.Contains(got, "goroutines: clean") {
		t.Errorf("output does not contain \"goroutines: clean\":\n%s", got)
	}
	if strings.Contains(got, "SUPPORT INFO:") {
		t.Errorf("output contains SUPPORT INFO: on an all-success run:\n%s", got)
	}
}

func TestRunOnce_ErrorPath(t *testing.T) {
	srv := statusServer(t, http.StatusInternalServerError, nil,
		map[string]string{"s": "error", "errmsg": "boom"})
	defer srv.Close()

	var out bytes.Buffer
	code := runOnce(onceCfg(srv.URL), &out)

	if code != 3 {
		t.Fatalf("runOnce() = %d, want 3; output:\n%s", code, out.String())
	}
	got := out.String()
	if !strings.Contains(got, "SUPPORT INFO:") {
		t.Errorf("output does not contain \"SUPPORT INFO:\":\n%s", got)
	}
	if !strings.Contains(got, "goroutines: clean") {
		t.Errorf("output does not contain \"goroutines: clean\":\n%s", got)
	}
}

// TestRunOnce_PartialFailure_StillPrintsSupportInfo locks in the fix for
// a bug the 2.6 review caught: runOnce originally gated the SUPPORT INFO
// block on m.lastErr, but Update clears lastErr on the next successful
// data message of any kind — so an early error followed by later
// successes exited 3 with no SUPPORT INFO block at all. Here only the
// bulkquotes path (hit first, by fetchQuotes) returns 500; every other
// route succeeds, so by the end of the run m.lastErr is nil again. The
// contract still requires exit 3 AND the SUPPORT INFO block.
func TestRunOnce_PartialFailure_StillPrintsSupportInfo(t *testing.T) {
	routes := fullRoutes()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/v1")

		if path == "/stocks/bulkquotes/" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"s": "error", "errmsg": "boom"})
			return
		}

		w.Header().Set("X-Api-Ratelimit-Limit", "100000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "99999")
		w.Header().Set("X-Api-Ratelimit-Consumed", "1")
		w.Header().Set("X-Api-Ratelimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()))
		if payload, ok := routes[path]; ok {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(payload)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	var out bytes.Buffer
	code := runOnce(onceCfg(srv.URL), &out)

	if code != 3 {
		t.Fatalf("runOnce() = %d, want 3; output:\n%s", code, out.String())
	}
	got := out.String()
	if !strings.Contains(got, "SUPPORT INFO:") {
		t.Errorf("output does not contain \"SUPPORT INFO:\" after a partial failure:\n%s", got)
	}
	if !strings.Contains(got, "InternalError") {
		t.Errorf("SUPPORT INFO block does not carry the failing request's InternalError:\n%s", got)
	}
	if !strings.Contains(got, "goroutines: clean") {
		t.Errorf("output does not contain \"goroutines: clean\":\n%s", got)
	}
}

// TestRunOnce_FullFetchSurface_NonDemo forces a token via the environment
// (t.Setenv, auto-restored) so the client is not in demo mode, then
// confirms every endpoint the -once contract lists actually gets hit:
// the full 5-symbol watchlist, both bulk quotes and prices, both stock
// and fund candles, and — unlike the demo-mode tests above — /user/.
func TestRunOnce_FullFetchSurface_NonDemo(t *testing.T) {
	t.Setenv("MARKETDATA_TOKEN", "test-token")

	var mu sync.Mutex
	hit := map[string]bool{}
	routes := fullRoutes()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Versioned endpoints (stocks, funds, markets) are requested with
		// a "/v1" prefix; unversioned ones (utilities: /user/, /status/,
		// /headers/) are not. Normalize it away so wantPaths below can
		// use one bare form for both.
		mu.Lock()
		hit[strings.TrimPrefix(r.URL.Path, "/v1")] = true
		mu.Unlock()

		w.Header().Set("X-Api-Ratelimit-Limit", "100000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "99999")
		w.Header().Set("X-Api-Ratelimit-Consumed", "1")
		w.Header().Set("X-Api-Ratelimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()))

		for path, payload := range routes {
			if r.URL.Path == path || r.URL.Path == "/v1"+path {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(payload)
				return
			}
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	var out bytes.Buffer
	code := runOnce(onceCfg(srv.URL), &out)

	if code != 0 {
		t.Fatalf("runOnce() = %d, want 0; output:\n%s", code, out.String())
	}

	wantPaths := []string{
		"/stocks/bulkquotes/",     // fetchQuotes + fetchDetailQuote
		"/stocks/prices/",         // fetchPrices
		"/stocks/candles/D/AAPL/", // fetchCandles, selected symbol
		"/funds/candles/D/VFINX/", // fetchFundCandles, the watchlist's fund symbol
		"/stocks/earnings/AAPL/",
		"/stocks/news/AAPL/",
		"/markets/status/", // fetchMarketStatus + fetchStatusHistory
		"/user/",           // fetchUser: only reachable when not in demo mode
		"/status/",         // fetchAPIStatus
		"/headers/",
		"/stocks/bulkcandles/D/",
	}
	mu.Lock()
	defer mu.Unlock()
	for _, p := range wantPaths {
		if !hit[p] {
			t.Errorf("runOnce() did not hit %s; hit = %v", p, hit)
		}
	}
}
