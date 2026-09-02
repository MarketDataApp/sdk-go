package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
)

// mockServer starts an httptest.Server that serves routes keyed by
// request path (e.g. "/stocks/bulkquotes/"). Both the unversioned path
// and its "/v1"-prefixed form are matched, since versioned endpoints
// (stocks, funds, markets) and unversioned ones (utilities) share this
// helper. Every response carries baseline rate-limit headers, and any
// path not present in routes (including the client's background
// /user/ priming request) falls through to a plain 404 — the same
// signal the SDK treats as "no data".
func mockServer(t *testing.T, routes map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Api-Ratelimit-Limit", "100000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "99999")
		w.Header().Set("X-Api-Ratelimit-Consumed", "1")
		w.Header().Set("X-Api-Ratelimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()))

		for path, payload := range routes {
			if r.URL.Path == path || r.URL.Path == "/v1"+path {
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(payload); err != nil {
					t.Errorf("mockServer: encode response for %s: %v", path, err)
				}
				return
			}
		}
		http.NotFound(w, r)
	}))
}

// statusServer starts an httptest.Server that returns the given HTTP
// status, headers, and JSON body for every request, regardless of
// path. It's used to force a specific SDK error classification (429,
// 401, ...) from any endpoint.
func statusServer(t *testing.T, status int, headers map[string]string, body any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body)
		}
	}))
}

// newTestClient builds a client pointed at srv with a fake token (so the
// pristine-output rule isn't tripped by demo-mode warnings) and startup
// validation disabled (the mock server doesn't implement every endpoint
// the validation check might hit).
func newTestClient(t *testing.T, srv *httptest.Server, opts ...marketdata.Option) *marketdata.Client {
	t.Helper()
	all := append([]marketdata.Option{
		marketdata.WithToken("test-token"),
		marketdata.WithBaseURL(srv.URL),
		marketdata.WithoutStartupValidation(),
	}, opts...)
	client, err := marketdata.NewClient(all...)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func TestFetchQuotes_Success(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/stocks/bulkquotes/": map[string]any{
			"s":         "ok",
			"symbol":    []string{"AAPL", "MSFT"},
			"ask":       []float64{150.25, 375.50},
			"askSize":   []int{100, 50},
			"bid":       []float64{150.20, 375.45},
			"bidSize":   []int{200, 100},
			"mid":       []float64{150.225, 375.475},
			"last":      []float64{150.22, 375.48},
			"change":    []float64{1.50, 2.25},
			"changepct": []float64{1.01, 0.60},
			"volume":    []int64{50000000, 25000000},
			"updated":   []int64{1704067200, 1704067200},
		},
	})
	defer srv.Close()
	client := newTestClient(t, srv)

	msg := fetchQuotes(client, []string{"AAPL", "MSFT"})()

	got, ok := msg.(quotesMsg)
	if !ok {
		t.Fatalf("msg type = %T, want quotesMsg", msg)
	}
	if len(got.quotes) != 2 {
		t.Fatalf("len(quotes) = %d, want 2", len(got.quotes))
	}
	if got.quotes[0].Symbol != "AAPL" {
		t.Errorf("quotes[0].Symbol = %q, want AAPL", got.quotes[0].Symbol)
	}
	if got.meta == nil {
		t.Fatal("meta = nil, want non-nil")
	}
}

func TestFetchPrices_Success(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/stocks/prices/": map[string]any{
			"s":         "ok",
			"symbol":    []string{"AAPL", "MSFT"},
			"mid":       []float64{150.225, 375.475},
			"change":    []float64{1.50, 2.25},
			"changepct": []float64{1.01, 0.60},
			"updated":   []int64{1704067200, 1704067200},
		},
	})
	defer srv.Close()
	client := newTestClient(t, srv)

	msg := fetchPrices(client, []string{"AAPL", "MSFT"})()

	got, ok := msg.(pricesMsg)
	if !ok {
		t.Fatalf("msg type = %T, want pricesMsg", msg)
	}
	if len(got.prices) != 2 {
		t.Fatalf("len(prices) = %d, want 2", len(got.prices))
	}
	if got.prices[0].Symbol != "AAPL" {
		t.Errorf("prices[0].Symbol = %q, want AAPL", got.prices[0].Symbol)
	}
	if got.meta == nil {
		t.Fatal("meta = nil, want non-nil")
	}
}

func candlePayload() map[string]any {
	return map[string]any{
		"s": "ok",
		"t": []int64{1704067200, 1704153600},
		"o": []float64{148.0, 150.0},
		"h": []float64{151.0, 152.0},
		"l": []float64{147.5, 149.5},
		"c": []float64{150.0, 151.5},
		"v": []int64{1000000, 1100000},
	}
}

func TestFetchCandles_Daily_Success(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/stocks/candles/D/AAPL/": candlePayload(),
	})
	defer srv.Close()
	client := newTestClient(t, srv)

	msg := fetchCandles(client, "AAPL", rangeDaily)()

	got, ok := msg.(candlesMsg)
	if !ok {
		t.Fatalf("msg type = %T, want candlesMsg", msg)
	}
	if got.symbol != "AAPL" {
		t.Errorf("symbol = %q, want AAPL", got.symbol)
	}
	if got.rng != rangeDaily {
		t.Errorf("rng = %v, want rangeDaily", got.rng)
	}
	if len(got.candles) != 2 {
		t.Fatalf("len(candles) = %d, want 2", len(got.candles))
	}
	if got.meta == nil {
		t.Fatal("meta = nil, want non-nil")
	}
}

func TestFetchCandles_Intraday_Success(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/stocks/candles/5/AAPL/": candlePayload(),
	})
	defer srv.Close()
	client := newTestClient(t, srv)

	msg := fetchCandles(client, "AAPL", rangeIntraday)()

	got, ok := msg.(candlesMsg)
	if !ok {
		t.Fatalf("msg type = %T, want candlesMsg", msg)
	}
	if got.rng != rangeIntraday {
		t.Errorf("rng = %v, want rangeIntraday", got.rng)
	}
	if len(got.candles) != 2 {
		t.Fatalf("len(candles) = %d, want 2", len(got.candles))
	}
}

func TestFetchCandles_Weekly_Success(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/stocks/candles/W/AAPL/": candlePayload(),
	})
	defer srv.Close()
	client := newTestClient(t, srv)

	msg := fetchCandles(client, "AAPL", rangeWeekly)()

	got, ok := msg.(candlesMsg)
	if !ok {
		t.Fatalf("msg type = %T, want candlesMsg", msg)
	}
	if got.rng != rangeWeekly {
		t.Errorf("rng = %v, want rangeWeekly", got.rng)
	}
	if len(got.candles) != 2 {
		t.Fatalf("len(candles) = %d, want 2", len(got.candles))
	}
}

func TestFetchFundCandles_Success(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/funds/candles/D/VFINX/": map[string]any{
			"s": "ok",
			"t": []int64{1704067200, 1704153600},
			"o": []float64{398.0, 399.0},
			"h": []float64{400.0, 401.0},
			"l": []float64{397.0, 398.5},
			"c": []float64{399.5, 400.5},
		},
	})
	defer srv.Close()
	client := newTestClient(t, srv)

	msg := fetchFundCandles(client, "VFINX")()

	got, ok := msg.(fundCandlesMsg)
	if !ok {
		t.Fatalf("msg type = %T, want fundCandlesMsg", msg)
	}
	if got.symbol != "VFINX" {
		t.Errorf("symbol = %q, want VFINX", got.symbol)
	}
	if len(got.candles) != 2 {
		t.Fatalf("len(candles) = %d, want 2", len(got.candles))
	}
	if got.meta == nil {
		t.Fatal("meta = nil, want non-nil")
	}
}

func TestFetchDetailQuote_Success(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/stocks/bulkquotes/": map[string]any{
			"s":          "ok",
			"symbol":     []string{"AAPL"},
			"ask":        []float64{150.25},
			"askSize":    []int{100},
			"bid":        []float64{150.20},
			"bidSize":    []int{200},
			"mid":        []float64{150.225},
			"last":       []float64{150.22},
			"change":     []float64{1.50},
			"changepct":  []float64{1.01},
			"volume":     []int64{50000000},
			"updated":    []int64{1704067200},
			"52weekHigh": []float64{198.23},
			"52weekLow":  []float64{124.17},
		},
	})
	defer srv.Close()
	client := newTestClient(t, srv)

	msg := fetchDetailQuote(client, "AAPL")()

	got, ok := msg.(detailQuoteMsg)
	if !ok {
		t.Fatalf("msg type = %T, want detailQuoteMsg", msg)
	}
	if got.symbol != "AAPL" {
		t.Errorf("symbol = %q, want AAPL", got.symbol)
	}
	if got.quote == nil {
		t.Fatal("quote = nil, want non-nil")
	}
	if got.quote.FiftyTwoWeekHigh != 198.23 {
		t.Errorf("FiftyTwoWeekHigh = %v, want 198.23", got.quote.FiftyTwoWeekHigh)
	}
	if got.quote.FiftyTwoWeekLow != 124.17 {
		t.Errorf("FiftyTwoWeekLow = %v, want 124.17", got.quote.FiftyTwoWeekLow)
	}
	if got.meta == nil {
		t.Fatal("meta = nil, want non-nil")
	}
}

func TestFetchEarnings_Success(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/stocks/earnings/AAPL/": map[string]any{
			"s":              "ok",
			"symbol":         []string{"AAPL"},
			"fiscalYear":     []int{2026},
			"fiscalQuarter":  []int{3},
			"date":           []int64{1704067200},
			"reportDate":     []int64{1704067200},
			"reportTime":     []string{"After Market"},
			"currency":       []string{"USD"},
			"reportedEPS":    []float64{0},
			"estimatedEPS":   []float64{1.5},
			"surpriseEPS":    []float64{0},
			"surpriseEPSpct": []float64{0},
			"updated":        []int64{1704067200},
		},
	})
	defer srv.Close()
	client := newTestClient(t, srv)

	msg := fetchEarnings(client, "AAPL")()

	got, ok := msg.(earningsMsg)
	if !ok {
		t.Fatalf("msg type = %T, want earningsMsg", msg)
	}
	if got.symbol != "AAPL" {
		t.Errorf("symbol = %q, want AAPL", got.symbol)
	}
	if len(got.earnings) != 1 {
		t.Fatalf("len(earnings) = %d, want 1", len(got.earnings))
	}
	if got.meta == nil {
		t.Fatal("meta = nil, want non-nil")
	}
}

func TestFetchNews_Success(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/stocks/news/AAPL/": map[string]any{
			"s":               "ok",
			"symbol":          []string{"AAPL", "AAPL"},
			"headline":        []string{"Headline one", "Headline two"},
			"content":         []string{"Content one", "Content two"},
			"source":          []string{"https://example.com/1", "https://example.com/2"},
			"publicationDate": []int64{1704067200, 1704153600},
		},
	})
	defer srv.Close()
	client := newTestClient(t, srv)

	msg := fetchNews(client, "AAPL")()

	got, ok := msg.(newsMsg)
	if !ok {
		t.Fatalf("msg type = %T, want newsMsg", msg)
	}
	if got.symbol != "AAPL" {
		t.Errorf("symbol = %q, want AAPL", got.symbol)
	}
	if len(got.articles) != 2 {
		t.Fatalf("len(articles) = %d, want 2", len(got.articles))
	}
	if got.meta == nil {
		t.Fatal("meta = nil, want non-nil")
	}
}

func TestFetchMarketStatus_Success(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/markets/status/": map[string]any{
			"s":      "ok",
			"date":   []int64{1704067200},
			"status": []string{"open"},
		},
	})
	defer srv.Close()
	client := newTestClient(t, srv)

	msg := fetchMarketStatus(client)()

	got, ok := msg.(marketStatusMsg)
	if !ok {
		t.Fatalf("msg type = %T, want marketStatusMsg", msg)
	}
	if got.status == nil {
		t.Fatal("status = nil, want non-nil")
	}
	if !got.status.Open {
		t.Error("status.Open = false, want true")
	}
	if got.meta == nil {
		t.Fatal("meta = nil, want non-nil")
	}
}

func TestFetchStatusHistory_Success(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/markets/status/": map[string]any{
			"s":      "ok",
			"date":   []int64{1703980800, 1704067200},
			"status": []string{"open", "closed"},
		},
	})
	defer srv.Close()
	client := newTestClient(t, srv)

	msg := fetchStatusHistory(client)()

	got, ok := msg.(statusHistoryMsg)
	if !ok {
		t.Fatalf("msg type = %T, want statusHistoryMsg", msg)
	}
	if len(got.statuses) != 2 {
		t.Fatalf("len(statuses) = %d, want 2", len(got.statuses))
	}
	if got.meta == nil {
		t.Fatal("meta = nil, want non-nil")
	}
}

func TestFetchUser_Success(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/user/": map[string]any{
			"x-ratelimit-requests-remaining": 9999,
			"x-ratelimit-requests-limit":     100000,
			"x-options-data-permissions":     "OPRA data delayed 15 minutes",
		},
	})
	defer srv.Close()
	client := newTestClient(t, srv)

	msg := fetchUser(client)()

	got, ok := msg.(userMsg)
	if !ok {
		t.Fatalf("msg type = %T, want userMsg", msg)
	}
	if got.user == nil {
		t.Fatal("user = nil, want non-nil")
	}
	if got.meta == nil {
		t.Fatal("meta = nil, want non-nil")
	}
}

func TestFetchAPIStatus_Success(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/status/": map[string]any{
			"s":            "ok",
			"service":      []string{"api"},
			"status":       []string{"online"},
			"online":       []bool{true},
			"uptimePct30d": []float64{0.999},
			"uptimePct90d": []float64{0.998},
			"updated":      []int64{1704067200},
		},
	})
	defer srv.Close()
	client := newTestClient(t, srv)

	msg := fetchAPIStatus(client)()

	got, ok := msg.(apiStatusMsg)
	if !ok {
		t.Fatalf("msg type = %T, want apiStatusMsg", msg)
	}
	if got.status == nil {
		t.Fatal("status = nil, want non-nil")
	}
	if !got.status.IsOnline() {
		t.Error("status.IsOnline() = false, want true")
	}
	if got.meta == nil {
		t.Fatal("meta = nil, want non-nil")
	}
}

func TestFetchHeaders_Success(t *testing.T) {
	// The /headers/ endpoint echoes the request's own headers as a flat
	// object (no "headers" envelope key).
	srv := mockServer(t, map[string]any{
		"/headers/": map[string]any{"Authorization": "Bearer ***"},
	})
	defer srv.Close()
	client := newTestClient(t, srv)

	msg := fetchHeaders(client)()

	got, ok := msg.(headersMsg)
	if !ok {
		t.Fatalf("msg type = %T, want headersMsg", msg)
	}
	if got.headers == nil {
		t.Fatal("headers = nil, want non-nil")
	}
	if got.meta == nil {
		t.Fatal("meta = nil, want non-nil")
	}
}

func TestFetchBulkCandles_Success(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/stocks/bulkcandles/D/": map[string]any{
			"s":      "ok",
			"symbol": []string{"AAPL", "MSFT"},
			"t":      []int64{1704067200, 1704067200},
			"o":      []float64{148.0, 370.0},
			"h":      []float64{151.0, 378.0},
			"l":      []float64{147.5, 369.0},
			"c":      []float64{150.0, 375.0},
			"v":      []int64{1000000, 900000},
		},
	})
	defer srv.Close()
	client := newTestClient(t, srv)

	msg := fetchBulkCandles(client, []string{"AAPL", "MSFT"})()

	got, ok := msg.(bulkCandlesMsg)
	if !ok {
		t.Fatalf("msg type = %T, want bulkCandlesMsg", msg)
	}
	if len(got.candles) != 2 {
		t.Fatalf("len(candles) = %d, want 2", len(got.candles))
	}
	if got.meta == nil {
		t.Fatal("meta = nil, want non-nil")
	}
}

func TestValidateSymbol_Success(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/stocks/bulkquotes/": map[string]any{
			"s":       "ok",
			"symbol":  []string{"AAPL"},
			"ask":     []float64{150.25},
			"bid":     []float64{150.20},
			"mid":     []float64{150.225},
			"last":    []float64{150.22},
			"updated": []int64{1704067200},
		},
	})
	defer srv.Close()
	client := newTestClient(t, srv)

	msg := validateSymbol(client, "AAPL")()

	got, ok := msg.(addValidatedMsg)
	if !ok {
		t.Fatalf("msg type = %T, want addValidatedMsg", msg)
	}
	if got.symbol != "AAPL" {
		t.Errorf("symbol = %q, want AAPL", got.symbol)
	}
	if got.noData {
		t.Error("noData = true, want false")
	}
	if got.quote == nil {
		t.Fatal("quote = nil, want non-nil")
	}
	if got.meta == nil {
		t.Fatal("meta = nil, want non-nil")
	}
}

// TestValidateSymbol_NoData_404 covers the ordinary no-data path: the API
// answers 404 for the symbol.
func TestValidateSymbol_NoData_404(t *testing.T) {
	srv := mockServer(t, map[string]any{})
	defer srv.Close()
	client := newTestClient(t, srv)

	msg := validateSymbol(client, "BOGUS")()

	if _, ok := msg.(errMsg); ok {
		t.Fatalf("got errMsg for 404, want addValidatedMsg: %v", msg)
	}
	got, ok := msg.(addValidatedMsg)
	if !ok {
		t.Fatalf("msg type = %T, want addValidatedMsg", msg)
	}
	if !got.noData {
		t.Error("noData = false, want true")
	}
	if got.quote != nil {
		t.Errorf("quote = %+v, want nil", got.quote)
	}
	if got.meta == nil || !got.meta.NoData {
		t.Errorf("meta = %v, want non-nil with NoData=true", got.meta)
	}
}

// TestValidateSymbol_NoData_QuoteNotFound covers the SDK-level
// QuoteNotFoundError path: the API answers 200 with an empty result set,
// which the SDK reports as a *stocks.QuoteNotFoundError rather than a
// 404. validateSymbol must still treat this as noData, not errMsg.
func TestValidateSymbol_NoData_QuoteNotFound(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/stocks/bulkquotes/": map[string]any{
			"s":      "ok",
			"symbol": []string{},
		},
	})
	defer srv.Close()
	client := newTestClient(t, srv)

	msg := validateSymbol(client, "ZZZZ")()

	if _, ok := msg.(errMsg); ok {
		t.Fatalf("got errMsg for QuoteNotFoundError, want addValidatedMsg: %v", msg)
	}
	got, ok := msg.(addValidatedMsg)
	if !ok {
		t.Fatalf("msg type = %T, want addValidatedMsg", msg)
	}
	if !got.noData {
		t.Error("noData = false, want true")
	}
	if got.quote != nil {
		t.Errorf("quote = %+v, want nil", got.quote)
	}
	// Contract: meta is nil on this path. The SDK returns
	// (nil, nil, *QuoteNotFoundError) — its errors never carry a
	// Response — so consumers of addValidatedMsg must nil-guard meta.
	if got.meta != nil {
		t.Errorf("meta = %v, want nil (SDK returns no Response with QuoteNotFoundError)", got.meta)
	}
}

// assertNoData checks that msg is a typed success message reporting no
// data (empty data, meta.NoData == true) rather than an errMsg — a 404
// for a valid request is not an error.
func assertNoData(t *testing.T, msg tea.Msg) {
	t.Helper()
	switch m := msg.(type) {
	case quotesMsg:
		if m.meta == nil || !m.meta.NoData {
			t.Errorf("quotesMsg.meta.NoData = %v, want true", m.meta)
		}
		if len(m.quotes) != 0 {
			t.Errorf("quotes = %v, want empty", m.quotes)
		}
	case candlesMsg:
		if m.meta == nil || !m.meta.NoData {
			t.Errorf("candlesMsg.meta.NoData = %v, want true", m.meta)
		}
		if len(m.candles) != 0 {
			t.Errorf("candles = %v, want empty", m.candles)
		}
	case earningsMsg:
		if m.meta == nil || !m.meta.NoData {
			t.Errorf("earningsMsg.meta.NoData = %v, want true", m.meta)
		}
		if len(m.earnings) != 0 {
			t.Errorf("earnings = %v, want empty", m.earnings)
		}
	case userMsg:
		if m.meta == nil || !m.meta.NoData {
			t.Errorf("userMsg.meta.NoData = %v, want true", m.meta)
		}
		if m.user != nil {
			t.Errorf("user = %+v, want nil", m.user)
		}
	default:
		t.Fatalf("msg type = %T, want a typed no-data msg", msg)
	}
}

// TestFetch_ErrorHandling exercises the four error/no-data classes the
// brief requires on representative endpoints (quotes, candles, earnings,
// user): 404 (no-data, not an error), 429 (RateLimitError), 401
// (AuthenticationError), and an unreachable server (NetworkError).
func TestFetch_ErrorHandling(t *testing.T) {
	cases := []struct {
		name string
		op   string
		call func(client *marketdata.Client) tea.Cmd
	}{
		{"quotes", "quotes", func(c *marketdata.Client) tea.Cmd {
			return fetchQuotes(c, []string{"AAPL"})
		}},
		{"candles", "candles", func(c *marketdata.Client) tea.Cmd {
			return fetchCandles(c, "AAPL", rangeDaily)
		}},
		{"earnings", "earnings", func(c *marketdata.Client) tea.Cmd {
			return fetchEarnings(c, "AAPL")
		}},
		{"user", "user", func(c *marketdata.Client) tea.Cmd {
			return fetchUser(c)
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/404NoData", func(t *testing.T) {
			srv := mockServer(t, map[string]any{})
			defer srv.Close()
			client := newTestClient(t, srv)

			msg := tc.call(client)()

			if _, ok := msg.(errMsg); ok {
				t.Fatalf("got errMsg for 404, want typed no-data msg: %v", msg)
			}
			assertNoData(t, msg)
		})

		t.Run(tc.name+"/429RateLimited", func(t *testing.T) {
			srv := statusServer(t, http.StatusTooManyRequests, map[string]string{
				"X-Api-Ratelimit-Limit":     "100000",
				"X-Api-Ratelimit-Remaining": "0",
				"X-Api-Ratelimit-Reset":     fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()),
			}, map[string]string{"s": "error", "errmsg": "rate limit exceeded"})
			defer srv.Close()
			client := newTestClient(t, srv)

			msg := tc.call(client)()

			em, ok := msg.(errMsg)
			if !ok {
				t.Fatalf("msg type = %T, want errMsg", msg)
			}
			if em.op != tc.op {
				t.Errorf("op = %q, want %q", em.op, tc.op)
			}
			var rle *marketdata.RateLimitError
			if !errors.As(em.err, &rle) {
				t.Errorf("err = %v (%T), want *marketdata.RateLimitError", em.err, em.err)
			}
		})

		t.Run(tc.name+"/401Auth", func(t *testing.T) {
			srv := statusServer(t, http.StatusUnauthorized, nil,
				map[string]string{"s": "error", "errmsg": "invalid token"})
			defer srv.Close()
			client := newTestClient(t, srv)

			msg := tc.call(client)()

			em, ok := msg.(errMsg)
			if !ok {
				t.Fatalf("msg type = %T, want errMsg", msg)
			}
			if em.op != tc.op {
				t.Errorf("op = %q, want %q", em.op, tc.op)
			}
			var ae *marketdata.AuthenticationError
			if !errors.As(em.err, &ae) {
				t.Errorf("err = %v (%T), want *marketdata.AuthenticationError", em.err, em.err)
			}
		})

		t.Run(tc.name+"/NetworkError", func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			srv.Close() // closed before use: connections are refused

			client := newTestClient(t, srv, marketdata.WithMaxRetries(0))

			msg := tc.call(client)()

			em, ok := msg.(errMsg)
			if !ok {
				t.Fatalf("msg type = %T, want errMsg", msg)
			}
			if em.op != tc.op {
				t.Errorf("op = %q, want %q", em.op, tc.op)
			}
			var ne *marketdata.NetworkError
			if !errors.As(em.err, &ne) {
				t.Errorf("err = %v (%T), want *marketdata.NetworkError", em.err, em.err)
			}
		})
	}
}
