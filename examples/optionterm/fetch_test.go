package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
)

// newTestClient builds a marketdata.Client backed by an httptest server
// running mux. A fake token keeps the client out of demo mode (so output
// stays pristine, with no demo-mode log noise) and
// WithoutStartupValidation skips the synchronous token check. NewClient
// still fires a background "/user/" request to seed rate-limit tracking,
// so mux must tolerate unrecognized paths.
func newTestClient(t *testing.T, mux http.Handler, extra ...marketdata.Option) *marketdata.Client {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	opts := []marketdata.Option{
		marketdata.WithToken("test-token"),
		marketdata.WithBaseURL(srv.URL),
		marketdata.WithoutStartupValidation(),
	}
	opts = append(opts, extra...)

	client, err := marketdata.NewClient(opts...)
	if err != nil {
		t.Fatalf("marketdata.NewClient() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// newMux builds a ServeMux that handles path with handler and answers every
// other path (notably the background "/user/" request) with a bare "ok"
// so it never shows up as noise in test failures.
func newMux(path string, handler http.HandlerFunc) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(path, handler)
	mux.HandleFunc("/", jsonHandler(map[string]any{"s": "ok"}))
	return mux
}

// jsonHandler responds to any request with 200 and body encoded as JSON.
func jsonHandler(body any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}
}

// notFoundHandler responds with the SDK's no-data signal: the API's real
// markerless 404 body, meaning "your query was valid and matched nothing".
// An errmsg here would mean the API rejected the question itself (an
// unknown symbol), which the SDK reports as a NotFoundError instead.
func notFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"s":"no_data","nextTime":null,"prevTime":null}`))
	}
}

// mustCmdMsg runs cmd and returns the tea.Msg it produces, failing the
// test if cmd is nil.
func mustCmdMsg(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("fetch function returned nil tea.Cmd")
	}
	return cmd()
}

// --- success: one per fetch function ---

func TestFetchUnderlying_Success(t *testing.T) {
	mux := newMux("/v1/stocks/bulkquotes/", jsonHandler(map[string]any{
		"s":       "ok",
		"symbol":  []string{"AAPL"},
		"bid":     []float64{184.50},
		"ask":     []float64{184.55},
		"mid":     []float64{184.525},
		"last":    []float64{184.52},
		"volume":  []int64{1000000},
		"updated": []int64{1704067200},
	}))
	client := newTestClient(t, mux)

	msg := mustCmdMsg(t, fetchUnderlying(client, "AAPL"))
	um, ok := msg.(underlyingMsg)
	if !ok {
		t.Fatalf("msg type = %T, want underlyingMsg", msg)
	}
	if um.quote == nil {
		t.Fatal("quote = nil, want non-nil")
	}
	if um.quote.Symbol != "AAPL" {
		t.Errorf("Symbol = %q, want AAPL", um.quote.Symbol)
	}
	if um.meta == nil {
		t.Error("meta = nil, want non-nil")
	}
}

func TestFetchExpirations_Success(t *testing.T) {
	var gotDateformat string
	mux := newMux("/v1/options/expirations/AAPL/", func(w http.ResponseWriter, r *http.Request) {
		gotDateformat = r.URL.Query().Get("dateformat")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":           "ok",
			"expirations": []int64{1737158400, 1739750400},
		})
	})
	client := newTestClient(t, mux)

	msg := mustCmdMsg(t, fetchExpirations(client, "AAPL"))
	em, ok := msg.(expirationsMsg)
	if !ok {
		t.Fatalf("msg type = %T, want expirationsMsg", msg)
	}
	if em.symbol != "AAPL" {
		t.Errorf("symbol = %q, want AAPL", em.symbol)
	}
	if len(em.expirations) != 2 {
		t.Fatalf("len(expirations) = %d, want 2", len(em.expirations))
	}
	if em.meta == nil {
		t.Error("meta = nil, want non-nil")
	}
	// Expirations always sends dateformat=unix, regardless of caller options.
	if gotDateformat != "unix" {
		t.Errorf("dateformat = %q, want unix", gotDateformat)
	}
}

func chainPayload() map[string]any {
	return map[string]any{
		"s": "ok",
		"optionSymbol": []string{
			"AAPL250117C00150000", "AAPL250117P00150000",
			"AAPL250117C00160000", "AAPL250117P00140000",
		},
		"underlying":      []string{"AAPL", "AAPL", "AAPL", "AAPL"},
		"expiration":      []int64{1737158400, 1737158400, 1737158400, 1737158400},
		"side":            []string{"call", "put", "call", "put"},
		"strike":          []float64{150, 150, 160, 140},
		"bid":             []float64{5.50, 4.50, 2.00, 1.00},
		"ask":             []float64{5.60, 4.60, 2.10, 1.10},
		"underlyingPrice": []float64{155, 155, 155, 155},
		"inTheMoney":      []bool{true, false, false, true},
		"updated":         []int64{1737000000, 1737000000, 1737000000, 1737000000},
	}
}

func TestFetchChain_Success(t *testing.T) {
	mux := newMux("/v1/options/chain/AAPL/", jsonHandler(chainPayload()))
	client := newTestClient(t, mux)

	exp := time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)
	msg := mustCmdMsg(t, fetchChain(client, "AAPL", exp, 0, 0, options.SideBoth))
	cm, ok := msg.(chainMsg)
	if !ok {
		t.Fatalf("msg type = %T, want chainMsg", msg)
	}
	if cm.chain == nil {
		t.Fatal("chain = nil, want non-nil")
	}
	if len(cm.chain.Options) < 4 {
		t.Fatalf("len(chain.Options) = %d, want >= 4", len(cm.chain.Options))
	}
	var sawCall, sawPut, sawITM, sawNotITM bool
	for _, o := range cm.chain.Options {
		switch o.Type {
		case options.Call:
			sawCall = true
		case options.Put:
			sawPut = true
		}
		if o.InTheMoney {
			sawITM = true
		} else {
			sawNotITM = true
		}
		if o.UnderlyingPrice == 0 {
			t.Errorf("OptionSymbol %s: UnderlyingPrice = 0, want set", o.OptionSymbol)
		}
	}
	if !sawCall || !sawPut {
		t.Error("chain should contain both call and put sides")
	}
	if !sawITM || !sawNotITM {
		t.Error("chain should contain both ITM and non-ITM contracts")
	}
	if cm.meta == nil {
		t.Error("meta = nil, want non-nil")
	}
}

func TestFetchChain_OptionsApplied(t *testing.T) {
	var strike, side, expiration string
	mux := newMux("/v1/options/chain/AAPL/", func(w http.ResponseWriter, r *http.Request) {
		strike = r.URL.Query().Get("strike")
		side = r.URL.Query().Get("side")
		expiration = r.URL.Query().Get("expiration")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"s": "ok", "optionSymbol": []string{}})
	})
	client := newTestClient(t, mux)

	exp := time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)
	_ = mustCmdMsg(t, fetchChain(client, "AAPL", exp, 150, 160, options.SideCall))

	if expiration != "2025-01-17" {
		t.Errorf("expiration = %q, want 2025-01-17", expiration)
	}
	if strike != "150-160" {
		t.Errorf("strike = %q, want 150-160", strike)
	}
	if side != "call" {
		t.Errorf("side = %q, want call", side)
	}
}

func TestFetchChain_NoStrikeOrSideWhenUnset(t *testing.T) {
	var strike, side string
	seen := false
	mux := newMux("/v1/options/chain/AAPL/", func(w http.ResponseWriter, r *http.Request) {
		seen = true
		q := r.URL.Query()
		strike = q.Get("strike")
		side = q.Get("side")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"s": "ok", "optionSymbol": []string{}})
	})
	client := newTestClient(t, mux)

	exp := time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)
	// lo/hi both zero and side is SideBoth: neither WithStrikeRange nor
	// WithSide should be added.
	_ = mustCmdMsg(t, fetchChain(client, "AAPL", exp, 0, 0, options.SideBoth))

	if !seen {
		t.Fatal("chain endpoint was not called")
	}
	if strike != "" {
		t.Errorf("strike = %q, want empty (unset)", strike)
	}
	if side != "" {
		t.Errorf("side = %q, want empty (unset)", side)
	}
}

func TestFetchContract_Success(t *testing.T) {
	mux := newMux("/v1/options/quotes/AAPL250117C00150000/", jsonHandler(map[string]any{
		"s":            "ok",
		"optionSymbol": []string{"AAPL250117C00150000"},
		"underlying":   []string{"AAPL"},
		"strike":       []float64{150},
		"side":         []string{"call"},
		"bid":          []float64{5.50},
		"ask":          []float64{5.60},
	}))
	client := newTestClient(t, mux)

	msg := mustCmdMsg(t, fetchContract(client, "AAPL250117C00150000"))
	cm, ok := msg.(contractMsg)
	if !ok {
		t.Fatalf("msg type = %T, want contractMsg", msg)
	}
	if cm.quote == nil {
		t.Fatal("quote = nil, want non-nil")
	}
	if cm.quote.OptionSymbol != "AAPL250117C00150000" {
		t.Errorf("OptionSymbol = %q, want AAPL250117C00150000", cm.quote.OptionSymbol)
	}
	if cm.meta == nil {
		t.Error("meta = nil, want non-nil")
	}
}

func TestFetchPinned_Success(t *testing.T) {
	syms := []string{"AAPL250117C00150000", "AAPL250117P00150000", "AAPL250117C00160000"}
	mux := http.NewServeMux()
	for _, sym := range syms {
		sym := sym
		mux.HandleFunc("/v1/options/quotes/"+sym+"/", jsonHandler(map[string]any{
			"s":            "ok",
			"optionSymbol": []string{sym},
			"underlying":   []string{"AAPL"},
			"bid":          []float64{1.0},
			"ask":          []float64{1.1},
		}))
	}
	mux.HandleFunc("/", jsonHandler(map[string]any{"s": "ok"}))
	client := newTestClient(t, mux)

	msg := mustCmdMsg(t, fetchPinned(client, syms))
	pm, ok := msg.(pinnedMsg)
	if !ok {
		t.Fatalf("msg type = %T, want pinnedMsg", msg)
	}
	if len(pm.quotes) != len(syms) {
		t.Fatalf("len(quotes) = %d, want %d", len(pm.quotes), len(syms))
	}
	if pm.meta == nil {
		t.Error("meta = nil, want non-nil")
	}
}

func TestLookupContract_Success(t *testing.T) {
	wantPath := "/v1/options/lookup/AAPL%202025-01-17%20150%20call/"
	var gotPath string
	mux := newMux("/v1/options/lookup/", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"s": "ok", "optionSymbol": "AAPL250117C00150000"})
	})
	client := newTestClient(t, mux)

	exp := time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)
	msg := mustCmdMsg(t, lookupContract(client, "AAPL", exp, 150, options.Call))
	lm, ok := msg.(lookupMsg)
	if !ok {
		t.Fatalf("msg type = %T, want lookupMsg", msg)
	}
	if lm.occ != "AAPL250117C00150000" {
		t.Errorf("occ = %q, want AAPL250117C00150000", lm.occ)
	}
	if lm.noData {
		t.Error("noData = true, want false")
	}
	if lm.meta == nil {
		t.Error("meta = nil, want non-nil")
	}
	if gotPath != wantPath {
		t.Errorf("request path = %q, want %q", gotPath, wantPath)
	}
}

// --- no-data (404): lookup and chain ---

func TestFetchChain_NoData(t *testing.T) {
	mux := newMux("/v1/options/chain/NOPE/", notFoundHandler())
	client := newTestClient(t, mux)

	exp := time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)
	msg := mustCmdMsg(t, fetchChain(client, "NOPE", exp, 0, 0, options.SideBoth))
	cm, ok := msg.(chainMsg)
	if !ok {
		t.Fatalf("msg type = %T, want chainMsg (404 is success, not errMsg)", msg)
	}
	// The SDK answers a markerless 404 with an empty chain, not a nil one,
	// so the app must read NoData from the metadata rather than from a nil
	// pointer it would otherwise dereference.
	if cm.chain == nil {
		t.Fatal("chain = nil, want an empty chain for no-data")
	}
	if len(cm.chain.Options) != 0 {
		t.Errorf("chain options = %d, want 0", len(cm.chain.Options))
	}
	if cm.meta == nil || !cm.meta.NoData {
		t.Error("meta.NoData = false, want true")
	}
}

func TestLookupContract_NoData(t *testing.T) {
	mux := newMux("/v1/options/lookup/", notFoundHandler())
	client := newTestClient(t, mux)

	exp := time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)
	msg := mustCmdMsg(t, lookupContract(client, "NOPE", exp, 999, options.Call))
	lm, ok := msg.(lookupMsg)
	if !ok {
		t.Fatalf("msg type = %T, want lookupMsg (404 is success, not errMsg)", msg)
	}
	if lm.occ != "" {
		t.Errorf("occ = %q, want empty for no-data", lm.occ)
	}
	if !lm.noData {
		t.Error("noData = false, want true")
	}
	if lm.meta == nil || !lm.meta.NoData {
		t.Error("meta.NoData = false, want true")
	}
}

// --- error classification: representative ops ---

func TestFetchChain_RateLimited(t *testing.T) {
	mux := newMux("/v1/options/chain/AAPL/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "0")
		w.Header().Set("X-Api-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10))
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{"s": "error", "errmsg": "rate limit exceeded"})
	})
	client := newTestClient(t, mux)

	exp := time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)
	msg := mustCmdMsg(t, fetchChain(client, "AAPL", exp, 0, 0, options.SideBoth))
	em, ok := msg.(errMsg)
	if !ok {
		t.Fatalf("msg type = %T, want errMsg", msg)
	}
	if em.op != "chain" {
		t.Errorf("op = %q, want chain", em.op)
	}
	var rle *marketdata.RateLimitError
	if !errors.As(em.err, &rle) {
		t.Fatalf("err = %v (%T), want *marketdata.RateLimitError", em.err, em.err)
	}
}

func TestFetchContract_AuthenticationError(t *testing.T) {
	mux := newMux("/v1/options/quotes/AAPL250117C00150000/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"s": "error", "errmsg": "invalid token"})
	})
	client := newTestClient(t, mux)

	msg := mustCmdMsg(t, fetchContract(client, "AAPL250117C00150000"))
	em, ok := msg.(errMsg)
	if !ok {
		t.Fatalf("msg type = %T, want errMsg", msg)
	}
	if em.op != "contract" {
		t.Errorf("op = %q, want contract", em.op)
	}
	var ae *marketdata.AuthenticationError
	if !errors.As(em.err, &ae) {
		t.Fatalf("err = %v (%T), want *marketdata.AuthenticationError", em.err, em.err)
	}
}

// failingRoundTripper always fails, simulating a network-level failure
// (DNS/connection error) rather than an HTTP error response.
type failingRoundTripper struct{}

func (failingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("simulated network failure")
}

func TestFetchUnderlying_NetworkError(t *testing.T) {
	client, err := marketdata.NewClient(
		marketdata.WithToken("test-token"),
		marketdata.WithBaseURL("http://127.0.0.1:9"),
		marketdata.WithoutStartupValidation(),
		marketdata.WithMaxRetries(0), // avoid the 1s+2s+4s retry backoff
		marketdata.WithHTTPClient(&http.Client{Transport: failingRoundTripper{}}),
	)
	if err != nil {
		t.Fatalf("marketdata.NewClient() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	msg := mustCmdMsg(t, fetchUnderlying(client, "AAPL"))
	em, ok := msg.(errMsg)
	if !ok {
		t.Fatalf("msg type = %T, want errMsg", msg)
	}
	if em.op != "underlying" {
		t.Errorf("op = %q, want underlying", em.op)
	}
	var ne *marketdata.NetworkError
	if !errors.As(em.err, &ne) {
		t.Fatalf("err = %v (%T), want *marketdata.NetworkError", em.err, em.err)
	}
}
