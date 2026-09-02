package stocks_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	internalhttp "github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/ratelimit"
	"github.com/MarketDataApp/sdk-go/v2/internal/retry"
	"github.com/MarketDataApp/sdk-go/v2/internal/timezone"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

// mockServer creates a test server that returns mock API responses.
func mockServer(t *testing.T, responses map[string]any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set rate limit headers
		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "9999")
		w.Header().Set("X-Api-Ratelimit-Consumed", "1")
		w.Header().Set("X-Api-Ratelimit-Reset", "1735689600")

		for path, resp := range responses {
			if r.URL.Path == path || r.URL.Path == "/v1"+path {
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(resp); err != nil {
					t.Errorf("failed to encode response: %v", err)
				}
				return
			}
		}

		http.NotFound(w, r)
	}))
}

// newTestService creates a stocks service for testing.
func newTestService(serverURL string) *stocks.Service {
	httpClient := internalhttp.New(internalhttp.Config{
		BaseURL:    serverURL,
		APIVersion: "v1",
		Token:      "test-api-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})
	return stocks.NewService(httpClient)
}

func TestQuote(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/bulkquotes/": map[string]any{
			"s":         "ok",
			"symbol":    []string{"AAPL"},
			"ask":       []float64{150.25},
			"askSize":   []int{100},
			"bid":       []float64{150.20},
			"bidSize":   []int{200},
			"mid":       []float64{150.225},
			"last":      []float64{150.22},
			"change":    []float64{1.50},
			"changepct": []float64{1.01},
			"volume":    []int64{50000000},
			"updated":   []int64{1704067200},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	ctx := context.Background()

	quote, _, err := svc.Quote(ctx, "AAPL")
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}

	if quote.Symbol != "AAPL" {
		t.Errorf("Symbol = %q, want %q", quote.Symbol, "AAPL")
	}
	if quote.Last != 150.22 {
		t.Errorf("Last = %v, want %v", quote.Last, 150.22)
	}
	if quote.Ask != 150.25 {
		t.Errorf("Ask = %v, want %v", quote.Ask, 150.25)
	}
	if quote.Bid != 150.20 {
		t.Errorf("Bid = %v, want %v", quote.Bid, 150.20)
	}
}

func TestQuotes(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/bulkquotes/": map[string]any{
			"s":         "ok",
			"symbol":    []string{"AAPL", "MSFT", "GOOG"},
			"ask":       []float64{150.25, 375.50, 140.75},
			"askSize":   []int{100, 50, 75},
			"bid":       []float64{150.20, 375.45, 140.70},
			"bidSize":   []int{200, 100, 125},
			"mid":       []float64{150.225, 375.475, 140.725},
			"last":      []float64{150.22, 375.48, 140.72},
			"change":    []float64{1.50, 2.25, -0.50},
			"changepct": []float64{1.01, 0.60, -0.35},
			"volume":    []int64{50000000, 25000000, 15000000},
			"updated":   []int64{1704067200, 1704067200, 1704067200},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	ctx := context.Background()

	quotes, _, err := svc.Quotes(ctx, []string{"AAPL", "MSFT", "GOOG"})
	if err != nil {
		t.Fatalf("Quotes() error = %v", err)
	}

	if len(quotes) != 3 {
		t.Fatalf("len(quotes) = %d, want %d", len(quotes), 3)
	}

	symbols := []string{"AAPL", "MSFT", "GOOG"}
	for i, q := range quotes {
		if q.Symbol != symbols[i] {
			t.Errorf("quotes[%d].Symbol = %q, want %q", i, q.Symbol, symbols[i])
		}
	}
}

func TestQuote_WithExtended(t *testing.T) {
	server := mockServer(t, map[string]any{
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
	defer server.Close()

	ctx := context.Background()

	// Verify extended param is sent
	server.Close()
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("extended") != "true" {
			t.Errorf("extended = %q, want true", r.URL.Query().Get("extended"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "9999")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":       "ok",
			"symbol":  []string{"AAPL"},
			"ask":     []float64{150.25},
			"bid":     []float64{150.20},
			"mid":     []float64{150.225},
			"last":    []float64{150.22},
			"updated": []int64{1704067200},
		})
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Quote(ctx, "AAPL", stocks.WithExtended(true))
	if err != nil {
		t.Fatalf("Quote() with extended error = %v", err)
	}
}

func TestQuotes_WithExtended(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("extended") != "true" {
			t.Errorf("extended = %q, want true", r.URL.Query().Get("extended"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "9999")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":       "ok",
			"symbol":  []string{"AAPL", "MSFT"},
			"ask":     []float64{150.25, 375.50},
			"bid":     []float64{150.20, 375.45},
			"mid":     []float64{150.225, 375.475},
			"last":    []float64{150.22, 375.48},
			"updated": []int64{1704067200, 1704067200},
		})
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	ctx := context.Background()

	quotes, _, err := svc.Quotes(ctx, []string{"AAPL", "MSFT"}, stocks.WithQuotesExtended(true))
	if err != nil {
		t.Fatalf("Quotes() with extended error = %v", err)
	}
	if len(quotes) != 2 {
		t.Fatalf("len(quotes) = %d, want 2", len(quotes))
	}
}

func TestCandles_204NoData(t *testing.T) {
	// A mode=cached cache miss returns 204 with an empty body; the SDK must
	// surface it as no-data (nil candles, NoData response, nil error), not a
	// decode error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	svc := newTestService(server.URL)
	candles, resp, err := svc.Candles(context.Background(), "AAPL", stocks.WithCandleWindow(stocks.LastN(5)))
	if err != nil {
		t.Fatalf("Candles() on 204 error = %v, want nil", err)
	}
	if candles != nil {
		t.Errorf("candles = %v, want nil", candles)
	}
	if resp == nil || !resp.NoData {
		t.Errorf("resp.NoData = false, want true for 204")
	}
}

func TestQuote_EmptySymbol(t *testing.T) {
	svc := newTestService("http://unused")
	ctx := context.Background()

	_, _, err := svc.Quote(ctx, "")
	if err == nil {
		t.Fatal("Quote() with empty symbol should return error")
	}
}

func TestQuote_Spread(t *testing.T) {
	quote := stocks.Quote{
		Ask: 150.25,
		Bid: 150.20,
		Mid: 150.225,
	}

	spread := quote.Spread()
	if !floatEquals(spread, 0.05, 0.0001) {
		t.Errorf("Spread() = %v, want ~0.05", spread)
	}

	spreadPct := quote.SpreadPercent()
	if spreadPct < 0.03 || spreadPct > 0.04 {
		t.Errorf("SpreadPercent() = %v, want ~0.033", spreadPct)
	}
}

// floatEquals compares two floats with a tolerance.
func floatEquals(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < tolerance
}

func TestCandle_Helpers(t *testing.T) {
	bullish := stocks.Candle{Open: 100, Close: 110, High: 115, Low: 95}
	bearish := stocks.Candle{Open: 110, Close: 100, High: 115, Low: 95}

	if !bullish.IsBullish() {
		t.Error("bullish candle should be bullish")
	}
	if bullish.IsBearish() {
		t.Error("bullish candle should not be bearish")
	}

	if bearish.IsBullish() {
		t.Error("bearish candle should not be bullish")
	}
	if !bearish.IsBearish() {
		t.Error("bearish candle should be bearish")
	}

	if bullish.Range() != 20 {
		t.Errorf("Range() = %v, want %v", bullish.Range(), 20)
	}
}

func TestCandles(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/candles/D/AAPL/": map[string]any{
			"s": "ok",
			"t": []int64{1704067200, 1704153600, 1704240000},
			"o": []float64{150.0, 151.0, 152.0},
			"h": []float64{155.0, 156.0, 157.0},
			"l": []float64{149.0, 150.0, 151.0},
			"c": []float64{154.0, 155.0, 156.0},
			"v": []int64{1000000, 1100000, 1200000},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	ctx := context.Background()

	candles, _, err := svc.Candles(ctx, "AAPL", stocks.WithResolution(stocks.ResolutionDaily))
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}

	if len(candles) != 3 {
		t.Fatalf("len(candles) = %d, want 3", len(candles))
	}

	if candles[0].Open != 150.0 {
		t.Errorf("candles[0].Open = %v, want %v", candles[0].Open, 150.0)
	}
	if candles[0].Close != 154.0 {
		t.Errorf("candles[0].Close = %v, want %v", candles[0].Close, 154.0)
	}
}

func TestCandles_EmptySymbol(t *testing.T) {
	svc := newTestService("http://unused")
	ctx := context.Background()

	_, _, err := svc.Candles(ctx, "")
	if err == nil {
		t.Fatal("Candles() with empty symbol should return error")
	}
}

func TestPrices(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/prices/AAPL/": map[string]any{
			"s":         "ok",
			"symbol":    []string{"AAPL"},
			"mid":       []float64{150.50},
			"change":    []float64{1.25},
			"changepct": []float64{0.84},
			"updated":   []int64{1704067200},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	ctx := context.Background()

	prices, _, err := svc.Prices(ctx, []string{"AAPL"})
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	if len(prices) != 1 {
		t.Fatalf("len(prices) = %d, want 1", len(prices))
	}

	if prices[0].Symbol != "AAPL" {
		t.Errorf("Symbol = %q, want %q", prices[0].Symbol, "AAPL")
	}
	if prices[0].Mid != 150.50 {
		t.Errorf("Mid = %v, want %v", prices[0].Mid, 150.50)
	}
}

func TestPrices_MultipleSymbols(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/prices/": map[string]any{
			"s":         "ok",
			"symbol":    []string{"AAPL", "MSFT"},
			"mid":       []float64{150.50, 375.25},
			"change":    []float64{1.25, 2.50},
			"changepct": []float64{0.84, 0.67},
			"updated":   []int64{1704067200, 1704067200},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	ctx := context.Background()

	prices, _, err := svc.Prices(ctx, []string{"AAPL", "MSFT"})
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}

	if len(prices) != 2 {
		t.Fatalf("len(prices) = %d, want 2", len(prices))
	}
}

func TestPrices_NoSymbols(t *testing.T) {
	svc := newTestService("http://unused")
	ctx := context.Background()

	_, _, err := svc.Prices(ctx, nil)
	if err == nil {
		t.Fatal("Prices() with no symbols should return error")
	}
}

func TestEarnings(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/earnings/AAPL/": map[string]any{
			"s":              "ok",
			"symbol":         []string{"AAPL"},
			"fiscalYear":     []int{2024},
			"fiscalQuarter":  []int{1},
			"date":           []int64{1704067200},
			"reportDate":     []int64{1704153600},
			"reportTime":     []string{"after close"},
			"currency":       []string{"USD"},
			"reportedEPS":    []float64{2.18},
			"estimatedEPS":   []float64{2.10},
			"surpriseEPS":    []float64{0.08},
			"surpriseEPSpct": []float64{3.81},
			"updated":        []int64{1704067200},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	ctx := context.Background()

	earnings, _, err := svc.Earnings(ctx, "AAPL")
	if err != nil {
		t.Fatalf("Earnings() error = %v", err)
	}

	if len(earnings) != 1 {
		t.Fatalf("len(earnings) = %d, want 1", len(earnings))
	}

	if earnings[0].Symbol != "AAPL" {
		t.Errorf("Symbol = %q, want %q", earnings[0].Symbol, "AAPL")
	}
	if earnings[0].FiscalYear != 2024 {
		t.Errorf("FiscalYear = %d, want %d", earnings[0].FiscalYear, 2024)
	}
	if earnings[0].FiscalQuarter != 1 {
		t.Errorf("FiscalQuarter = %d, want %d", earnings[0].FiscalQuarter, 1)
	}
}

func TestEarnings_NullVsZeroEPS(t *testing.T) {
	// Hand-written wire JSON with null and a true 0 inside the same SoA
	// arrays: three quarters — reported ($1.52 beat), met expectations
	// (surprise 0), and not yet reported (nulls). The decode layer must
	// keep null and $0.00 distinguishable end to end.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"s": "ok",
			"symbol": ["AAPL", "AAPL", "AAPL"],
			"fiscalYear": [2023, 2024, 2024],
			"fiscalQuarter": [4, 1, 2],
			"date": [1704067200, 1711929600, 1719792000],
			"reportDate": [1704153600, 1712016000, 1719878400],
			"reportTime": ["after close", "after close", "after close"],
			"currency": ["USD", "USD", "USD"],
			"reportedEPS": [1.52, 2.10, null],
			"estimatedEPS": [1.44, 2.10, 2.25],
			"surpriseEPS": [0.08, 0, null],
			"surpriseEPSpct": [5.56, 0, null],
			"updated": [1704067200, 1712016000, 1712016000]
		}`))
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	earnings, _, err := svc.Earnings(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Earnings() error = %v", err)
	}
	if len(earnings) != 3 {
		t.Fatalf("len(earnings) = %d, want 3", len(earnings))
	}

	beat, met, future := earnings[0], earnings[1], earnings[2]

	if beat.SurpriseEPS == nil || *beat.SurpriseEPS != 0.08 {
		t.Errorf("beat.SurpriseEPS = %v, want pointer to 0.08", beat.SurpriseEPS)
	}

	// The met-expectations quarter: a true 0 on the wire must survive as a
	// pointer to zero, not collapse to nil.
	if met.SurpriseEPS == nil || *met.SurpriseEPS != 0 {
		t.Errorf("met.SurpriseEPS = %v, want pointer to 0", met.SurpriseEPS)
	}
	if met.SurpriseEPSPercent == nil || *met.SurpriseEPSPercent != 0 {
		t.Errorf("met.SurpriseEPSPercent = %v, want pointer to 0", met.SurpriseEPSPercent)
	}

	// The unreported quarter: wire nulls must map to nil.
	if future.ReportedEPS != nil {
		t.Errorf("future.ReportedEPS = %v, want nil for wire null", future.ReportedEPS)
	}
	if future.SurpriseEPS != nil || future.SurpriseEPSPercent != nil {
		t.Errorf("future surprise fields = %v/%v, want nil for wire nulls", future.SurpriseEPS, future.SurpriseEPSPercent)
	}
	if future.EstimatedEPS == nil || *future.EstimatedEPS != 2.25 {
		t.Errorf("future.EstimatedEPS = %v, want pointer to 2.25", future.EstimatedEPS)
	}
}

// TestEarnings_LastNAnchoredToToday pins the wire contract that works
// around the API defect tracked as MarketData-App/api#283: the earnings
// endpoint silently ignores a countback that arrives without to=, so a
// bare LastN(n) must be sent as countback=n&to=<today Eastern>. Candles
// and News, where the API honors bare countback, must keep sending it
// without an anchor.
func TestEarnings_LastNAnchoredToToday(t *testing.T) {
	type captured struct {
		countback, to string
		hasTo         bool
	}
	newCaptureServer := func(c *captured) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.countback = r.URL.Query().Get("countback")
			c.to = r.URL.Query().Get("to")
			_, c.hasTo = r.URL.Query()["to"]
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"s":"ok","symbol":[]}`))
		}))
	}
	// The expected date is computed around the call so a test that happens
	// to straddle midnight Eastern accepts either day.
	todayEastern := func() string { return time.Now().In(timezone.Eastern).Format("2006-01-02") }

	t.Run("earnings LastN gains a to= anchor of today Eastern", func(t *testing.T) {
		var c captured
		server := newCaptureServer(&c)
		defer server.Close()
		svc := newTestService(server.URL)

		before := todayEastern()
		if _, _, err := svc.Earnings(context.Background(), "AAPL", stocks.WithEarningsWindow(stocks.LastN(4))); err != nil {
			t.Fatalf("Earnings() error = %v", err)
		}
		after := todayEastern()

		if c.countback != "4" {
			t.Errorf("countback = %q, want %q", c.countback, "4")
		}
		if c.to != before && c.to != after {
			t.Errorf("to = %q, want today Eastern (%q)", c.to, before)
		}
	})

	t.Run("earnings LastNUntil keeps the caller's anchor", func(t *testing.T) {
		var c captured
		server := newCaptureServer(&c)
		defer server.Close()
		svc := newTestService(server.URL)

		until := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
		if _, _, err := svc.Earnings(context.Background(), "AAPL", stocks.WithEarningsWindow(stocks.LastNUntil(4, until))); err != nil {
			t.Fatalf("Earnings() error = %v", err)
		}
		if c.countback != "4" || c.to != "2026-03-31" {
			t.Errorf("countback/to = %q/%q, want 4/2026-03-31", c.countback, c.to)
		}
	})

	t.Run("candles LastN stays unanchored", func(t *testing.T) {
		var c captured
		server := newCaptureServer(&c)
		defer server.Close()
		svc := newTestService(server.URL)

		if _, _, err := svc.Candles(context.Background(), "AAPL", stocks.WithCandleWindow(stocks.LastN(30))); err != nil {
			t.Fatalf("Candles() error = %v", err)
		}
		if c.countback != "30" || c.hasTo {
			t.Errorf("countback/to = %q/%q (hasTo=%t), want 30 with no to param", c.countback, c.to, c.hasTo)
		}
	})

	t.Run("news LastN stays unanchored", func(t *testing.T) {
		var c captured
		server := newCaptureServer(&c)
		defer server.Close()
		svc := newTestService(server.URL)

		if _, _, err := svc.News(context.Background(), "AAPL", stocks.WithNewsWindow(stocks.LastN(5))); err != nil {
			t.Fatalf("News() error = %v", err)
		}
		if c.countback != "5" || c.hasTo {
			t.Errorf("countback/to = %q/%q (hasTo=%t), want 5 with no to param", c.countback, c.to, c.hasTo)
		}
	})
}

func TestQuote_NullUpdatedIsZeroTime(t *testing.T) {
	// A wire null timestamp must surface as the zero time (IsZero-friendly),
	// not as the Unix epoch rendered in Eastern (1969-12-31 19:00 EST).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok","symbol":["AAPL"],"ask":[150.25],"askSize":[100],"bid":[150.20],"bidSize":[200],"mid":[150.225],"last":[150.22],"volume":[50000000],"updated":[null]}`))
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	quote, _, err := svc.Quote(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if !quote.Updated.IsZero() {
		t.Errorf("Updated = %v for a wire null, want the zero time", quote.Updated)
	}
}

func TestEarnings_EmptySymbol(t *testing.T) {
	svc := newTestService("http://unused")
	ctx := context.Background()

	_, _, err := svc.Earnings(ctx, "")
	if err == nil {
		t.Fatal("Earnings() with empty symbol should return error")
	}
}

func TestNews(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/news/AAPL/": map[string]any{
			"s":               "ok",
			"symbol":          []string{"AAPL"},
			"headline":        []string{"Apple announces new product"},
			"content":         []string{"Apple Inc. today announced..."},
			"source":          []string{"https://example.com/news/1"},
			"publicationDate": []int64{1704067200},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	ctx := context.Background()

	news, _, err := svc.News(ctx, "AAPL")
	if err != nil {
		t.Fatalf("News() error = %v", err)
	}

	if len(news) != 1 {
		t.Fatalf("len(news) = %d, want 1", len(news))
	}

	if news[0].Symbol != "AAPL" {
		t.Errorf("Symbol = %q, want %q", news[0].Symbol, "AAPL")
	}
	if news[0].Headline != "Apple announces new product" {
		t.Errorf("Headline = %q, want %q", news[0].Headline, "Apple announces new product")
	}
}

func TestNews_EmptySymbol(t *testing.T) {
	svc := newTestService("http://unused")
	ctx := context.Background()

	_, _, err := svc.News(ctx, "")
	if err == nil {
		t.Fatal("News() with empty symbol should return error")
	}
}

func TestBulkCandles(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/bulkcandles/D/": map[string]any{
			"s":      "ok",
			"symbol": []string{"AAPL", "MSFT"},
			"t":      []int64{1704067200, 1704067200},
			"o":      []float64{150.0, 375.0},
			"h":      []float64{155.0, 380.0},
			"l":      []float64{149.0, 374.0},
			"c":      []float64{154.0, 378.0},
			"v":      []int64{1000000, 500000},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	ctx := context.Background()

	bulkCandles, _, err := svc.BulkCandles(ctx, []string{"AAPL", "MSFT"})
	if err != nil {
		t.Fatalf("BulkCandles() error = %v", err)
	}

	if len(bulkCandles) != 2 {
		t.Fatalf("len(bulkCandles) = %d, want 2", len(bulkCandles))
	}

	if bulkCandles[0].Symbol != "AAPL" {
		t.Errorf("bulkCandles[0].Symbol = %q, want %q", bulkCandles[0].Symbol, "AAPL")
	}
	if bulkCandles[1].Symbol != "MSFT" {
		t.Errorf("bulkCandles[1].Symbol = %q, want %q", bulkCandles[1].Symbol, "MSFT")
	}
}

// TestQuote_WithCandle pins that the candle parameter reaches the wire and
// that the OHLC fields it unlocks are decoded. The API omits o/h/l/c entirely
// unless candle=true is sent (verified live 2026-08-11), so a request that
// drops the parameter silently yields zeroed OHLC.
func TestQuote_WithCandle(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		// Hand-written wire JSON in the real shape: OHLC are top-level
		// parallel arrays keyed o/h/l/c, present only with candle=true.
		_, _ = w.Write([]byte(`{"s":"ok","symbol":["AAPL"],"ask":[303.49],"askSize":[80],` +
			`"bid":[303.47],"bidSize":[80],"mid":[303.48],"last":[303.5],"change":[-4.76],` +
			`"changepct":[-0.0154],"volume":[17950994],"updated":[1786474123],` +
			`"o":[309.35],"h":[310.2],"l":[302.1],"c":[303.5]}`))
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	q, _, err := svc.Quote(context.Background(), "AAPL", stocks.WithCandle(true))
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if got := gotQuery.Get("candle"); got != "true" {
		t.Errorf("candle = %q, want true", got)
	}
	for _, c := range []struct {
		name string
		got  float64
		want float64
	}{
		{"Open", q.Open, 309.35}, {"High", q.High, 310.2},
		{"Low", q.Low, 302.1}, {"Close", q.Close, 303.5},
	} {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

// TestQuote_WithoutCandleSendsNothing pins the counterpart: the parameter is
// absent unless asked for, so the default request shape is unchanged.
func TestQuote_WithoutCandleSendsNothing(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok","symbol":["AAPL"],"last":[303.5],"updated":[1786474123]}`))
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	if _, _, err := svc.Quote(context.Background(), "AAPL"); err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if gotQuery.Has("candle") {
		t.Errorf("candle = %q, want the parameter absent", gotQuery.Get("candle"))
	}
}

// TestQuote_WithCandleFalse pins the explicit opt-out: passing false sends
// candle=false rather than omitting the parameter, so a caller can override an
// API-side default if one is ever introduced.
func TestQuote_WithCandleFalse(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok","symbol":["AAPL"],"last":[303.5],"updated":[1786474123]}`))
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	if _, _, err := svc.Quote(context.Background(), "AAPL", stocks.WithCandle(false)); err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if got := gotQuery.Get("candle"); got != "false" {
		t.Errorf("candle = %q, want false", got)
	}
}

// TestQuotes_WithQuotesCandleFalse is the bulk counterpart of
// TestQuote_WithCandleFalse.
func TestQuotes_WithQuotesCandleFalse(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok","symbol":["AAPL"],"last":[303.5],"updated":[1786474123]}`))
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Quotes(context.Background(), []string{"AAPL"}, stocks.WithQuotesCandle(false))
	if err != nil {
		t.Fatalf("Quotes() error = %v", err)
	}
	if got := gotQuery.Get("candle"); got != "false" {
		t.Errorf("candle = %q, want false", got)
	}
}

// TestQuotes_WithQuotesCandle proves the bulk endpoint honors candle too —
// unlike 52week, which it ignores (verified live 2026-08-11).
func TestQuotes_WithQuotesCandle(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok","symbol":["AAPL","MSFT"],"last":[303.5,410.1],` +
			`"updated":[1786474123,1786474123],"o":[309.35,405.0],"h":[310.2,411.0],` +
			`"l":[302.1,404.0],"c":[303.5,410.1]}`))
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	quotes, _, err := svc.Quotes(context.Background(), []string{"AAPL", "MSFT"},
		stocks.WithQuotesCandle(true))
	if err != nil {
		t.Fatalf("Quotes() error = %v", err)
	}
	if got := gotQuery.Get("candle"); got != "true" {
		t.Errorf("candle = %q, want true", got)
	}
	if len(quotes) != 2 {
		t.Fatalf("len(quotes) = %d, want 2", len(quotes))
	}
	if quotes[1].Open != 405.0 || quotes[1].Close != 410.1 {
		t.Errorf("MSFT OHLC = (%v, %v), want (405, 410.1)", quotes[1].Open, quotes[1].Close)
	}
}

func TestBulkCandles_NoSymbols(t *testing.T) {
	svc := newTestService("http://unused")
	ctx := context.Background()

	_, _, err := svc.BulkCandles(ctx, []string{})
	if err == nil {
		t.Fatal("BulkCandles() with no symbols should return error")
	}
}

// TestBulkCandles_NoSymbolsWithSnapshotFalse pins that snapshot=false does not
// unlock the empty-symbol form: only an affirmative snapshot request does.
func TestBulkCandles_NoSymbolsWithSnapshotFalse(t *testing.T) {
	svc := newTestService("http://unused")
	ctx := context.Background()

	_, _, err := svc.BulkCandles(ctx, nil, stocks.WithSnapshot(false))
	if err == nil {
		t.Fatal("BulkCandles() with no symbols and snapshot=false should return error")
	}
}

// TestBulkCandles_MarketWideSnapshot covers the one case where an empty symbol
// list is legitimate: snapshot=true asks the API for every symbol it covers.
// The request must carry snapshot=true and must NOT carry an empty symbols
// parameter, which the endpoint would read as "no symbols" rather than "all".
func TestBulkCandles_MarketWideSnapshot(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok","symbol":["AAPL","MSFT"],"o":[100,200],"h":[101,201],"l":[99,199],"c":[100.5,200.5],"v":[1000,2000],"t":[1704124800,1704124800]}`))
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	candles, _, err := svc.BulkCandles(context.Background(), nil, stocks.WithSnapshot(true))
	if err != nil {
		t.Fatalf("BulkCandles() error = %v", err)
	}
	if gotQuery.Has("symbols") {
		t.Errorf("symbols = %q, want the parameter absent for a market-wide snapshot", gotQuery.Get("symbols"))
	}
	if got := gotQuery.Get("snapshot"); got != "true" {
		t.Errorf("snapshot = %q, want true", got)
	}
	if len(candles) != 2 {
		t.Errorf("len(candles) = %d, want 2", len(candles))
	}
}

func TestResolution_String(t *testing.T) {
	tests := []struct {
		res  stocks.Resolution
		want string
	}{
		{stocks.Resolution1Min, "1"},
		{stocks.Resolution5Min, "5"},
		{stocks.Resolution15Min, "15"},
		{stocks.Resolution30Min, "30"},
		{stocks.Resolution1Hour, "60"},
		{stocks.ResolutionDaily, "D"},
		{stocks.ResolutionWeekly, "W"},
		{stocks.ResolutionMonthly, "M"},
	}

	for _, tt := range tests {
		if got := tt.res.String(); got != tt.want {
			t.Errorf("%v.String() = %q, want %q", tt.res, got, tt.want)
		}
	}
}

func TestCandle_RangePercent(t *testing.T) {
	candle := stocks.Candle{Open: 100, High: 110, Low: 90, Close: 105}

	pct := candle.RangePercent()
	// Range is 20, Open is 100, so 20%
	if pct != 20.0 {
		t.Errorf("RangePercent() = %v, want 20.0", pct)
	}
}

func TestCandle_RangePercent_ZeroOpen(t *testing.T) {
	candle := stocks.Candle{Open: 0, High: 10, Low: 5, Close: 8}

	pct := candle.RangePercent()
	if pct != 0 {
		t.Errorf("RangePercent() = %v, want 0 (zero open)", pct)
	}
}

func TestQuote_SpreadPercent_ZeroMid(t *testing.T) {
	quote := stocks.Quote{Ask: 0, Bid: 0, Mid: 0}

	pct := quote.SpreadPercent()
	if pct != 0 {
		t.Errorf("SpreadPercent() = %v, want 0 (zero mid)", pct)
	}
}

func TestQuote_NotFound(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/bulkquotes/": map[string]any{
			"s":      "ok",
			"symbol": []string{}, // Empty response
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	ctx := context.Background()

	_, _, err := svc.Quote(ctx, "INVALID")
	if err == nil {
		t.Fatal("Quote() should return error for symbol not found")
	}
}

func TestQuote_WithFiftyTwoWeek(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for 52week parameter
		if r.URL.Query().Get("52week") != "true" {
			t.Error("52week parameter should be true")
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "9999")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":          "ok",
			"symbol":     []string{"AAPL"},
			"last":       []float64{150.0},
			"52weekHigh": []float64{200.0},
			"52weekLow":  []float64{100.0},
		})
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	quote, _, err := svc.Quote(context.Background(), "AAPL", stocks.WithFiftyTwoWeek(true))
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if quote.FiftyTwoWeekHigh != 200.0 {
		t.Errorf("FiftyTwoWeekHigh = %f, want 200.0", quote.FiftyTwoWeekHigh)
	}
}

func TestQuotes_APIError(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/bulkquotes/": map[string]any{
			"s":      "error",
			"errmsg": "Something went wrong",
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Quotes(context.Background(), []string{"AAPL"})
	if err == nil {
		t.Fatal("Quotes() should return error for API error response")
	}
}

func TestCandles_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for query parameters
		if r.URL.Query().Get("from") == "" {
			t.Error("from parameter should be set")
		}
		if r.URL.Query().Get("to") == "" {
			t.Error("to parameter should be set")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s": "ok",
			"t": []int64{1704067200},
			"o": []float64{150.0},
			"h": []float64{155.0},
			"l": []float64{149.0},
			"c": []float64{154.0},
			"v": []int64{1000000},
		})
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	_, _, err := svc.Candles(context.Background(), "AAPL",
		stocks.WithResolution(stocks.ResolutionDaily),
		stocks.WithCandleWindow(stocks.Between(from, to)),
	)
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
}

func TestCandles_WithExtendedAndAdjustSplits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("extended") != "true" {
			t.Errorf("extended = %q, want true", r.URL.Query().Get("extended"))
		}
		if r.URL.Query().Get("adjustsplits") != "false" {
			t.Errorf("adjustsplits = %q, want false", r.URL.Query().Get("adjustsplits"))
		}
		if r.URL.Query().Get("adjustdividends") != "true" {
			t.Errorf("adjustdividends = %q, want true", r.URL.Query().Get("adjustdividends"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "9999")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s": "ok",
			"t": []int64{1704067200},
			"o": []float64{150.0},
			"h": []float64{155.0},
			"l": []float64{149.0},
			"c": []float64{154.0},
			"v": []int64{1000000},
		})
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Candles(context.Background(), "AAPL",
		stocks.WithResolution(stocks.ResolutionDaily),
		stocks.WithCandleExtended(true),
		stocks.WithCandleAdjustSplits(false),
		stocks.WithCandleAdjustDividends(true),
	)
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
}

func TestCandles_APIError(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/candles/D/AAPL/": map[string]any{
			"s":      "error",
			"errmsg": "Something went wrong",
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Candles(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("Candles() should return error for API error response")
	}
}

func TestPrices_APIError(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/prices/AAPL/": map[string]any{
			"s":      "error",
			"errmsg": "Something went wrong",
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Prices(context.Background(), []string{"AAPL"})
	if err == nil {
		t.Fatal("Prices() should return error for API error response")
	}
}

func TestEarnings_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for query parameters
		if r.URL.Query().Get("from") == "" {
			t.Error("from parameter should be set")
		}
		if r.URL.Query().Get("to") == "" {
			t.Error("to parameter should be set")
		}
		// countback is mutually exclusive with from/to and is no longer
		// expressible alongside a range, so it must NOT be present here.
		if r.URL.Query().Get("countback") != "" {
			t.Errorf("countback = %q, want empty (exclusive with from/to)", r.URL.Query().Get("countback"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":             "ok",
			"symbol":        []string{"AAPL"},
			"fiscalYear":    []int{2024},
			"fiscalQuarter": []int{1},
		})
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	_, _, err := svc.Earnings(context.Background(), "AAPL",
		stocks.WithEarningsWindow(stocks.Between(from, to)),
	)
	if err != nil {
		t.Fatalf("Earnings() error = %v", err)
	}
}

func TestEarnings_WithDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("date") == "" {
			t.Error("date parameter should be set")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":             "ok",
			"symbol":        []string{"AAPL"},
			"fiscalYear":    []int{2024},
			"fiscalQuarter": []int{1},
		})
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	_, _, err := svc.Earnings(context.Background(), "AAPL",
		stocks.WithEarningsWindow(stocks.OnDate(date)),
	)
	if err != nil {
		t.Fatalf("Earnings() error = %v", err)
	}
}

func TestEarnings_APIError(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/earnings/AAPL/": map[string]any{
			"s":      "error",
			"errmsg": "Something went wrong",
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Earnings(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("Earnings() should return error for API error response")
	}
}

func TestNews_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("from") == "" {
			t.Error("from parameter should be set")
		}
		if r.URL.Query().Get("to") == "" {
			t.Error("to parameter should be set")
		}
		if r.URL.Query().Get("countback") != "" {
			t.Errorf("countback = %q, want empty (exclusive with from/to)", r.URL.Query().Get("countback"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":        "ok",
			"symbol":   []string{"AAPL"},
			"headline": []string{"Test headline"},
		})
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	_, _, err := svc.News(context.Background(), "AAPL",
		stocks.WithNewsWindow(stocks.Between(from, to)),
	)
	if err != nil {
		t.Fatalf("News() error = %v", err)
	}
}

func TestNews_WithDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("date") == "" {
			t.Error("date parameter should be set")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":        "ok",
			"symbol":   []string{"AAPL"},
			"headline": []string{"Test headline"},
		})
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	_, _, err := svc.News(context.Background(), "AAPL",
		stocks.WithNewsWindow(stocks.OnDate(date)),
	)
	if err != nil {
		t.Fatalf("News() error = %v", err)
	}
}

func TestNews_APIError(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/news/AAPL/": map[string]any{
			"s":      "error",
			"errmsg": "Something went wrong",
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.News(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("News() should return error for API error response")
	}
}

func TestBulkCandles_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("date") == "" {
			t.Error("date parameter should be set")
		}
		if r.URL.Query().Get("adjustsplits") != "true" {
			t.Errorf("adjustsplits = %q, want true", r.URL.Query().Get("adjustsplits"))
		}
		if r.URL.Query().Get("adjustdividends") != "false" {
			t.Errorf("adjustdividends = %q, want false", r.URL.Query().Get("adjustdividends"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":      "ok",
			"symbol": []string{"AAPL"},
			"t":      []int64{1704067200},
			"o":      []float64{150.0},
			"h":      []float64{155.0},
			"l":      []float64{149.0},
			"c":      []float64{154.0},
			"v":      []int64{1000000},
		})
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	_, _, err := svc.BulkCandles(context.Background(), []string{"AAPL"},
		stocks.WithBulkDate(date),
		stocks.WithBulkResolution(stocks.ResolutionDaily),
		stocks.WithAdjustSplits(true),
		stocks.WithAdjustDividends(false),
	)
	if err != nil {
		t.Fatalf("BulkCandles() error = %v", err)
	}
}

func TestBulkCandles_AdjustSplitsFalse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("adjustsplits") != "false" {
			t.Errorf("adjustsplits = %q, want false", r.URL.Query().Get("adjustsplits"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":      "ok",
			"symbol": []string{"AAPL"},
		})
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.BulkCandles(context.Background(), []string{"AAPL"},
		stocks.WithAdjustSplits(false),
	)
	if err != nil {
		t.Fatalf("BulkCandles() error = %v", err)
	}
}

func TestBulkCandles_WithSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("snapshot") != "true" {
			t.Errorf("snapshot = %q, want true", r.URL.Query().Get("snapshot"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":      "ok",
			"symbol": []string{"AAPL"},
			"t":      []int64{1704067200},
			"o":      []float64{150.0},
			"h":      []float64{155.0},
			"l":      []float64{149.0},
			"c":      []float64{154.0},
			"v":      []int64{1000000},
		})
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.BulkCandles(context.Background(), []string{"AAPL"},
		stocks.WithSnapshot(true),
	)
	if err != nil {
		t.Fatalf("BulkCandles() error = %v", err)
	}
}

func TestBulkCandles_APIError(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/bulkcandles/D/": map[string]any{
			"s":      "error",
			"errmsg": "Something went wrong",
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.BulkCandles(context.Background(), []string{"AAPL"})
	if err == nil {
		t.Fatal("BulkCandles() should return error for API error response")
	}
}

// HTTP error tests
func TestQuote_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Quote(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("Quote() should return error for HTTP error")
	}
}

func TestQuotes_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Quotes(context.Background(), []string{"AAPL"})
	if err == nil {
		t.Fatal("Quotes() should return error for HTTP error")
	}
}

func TestCandles_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Candles(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("Candles() should return error for HTTP error")
	}
}

func TestPrices_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Prices(context.Background(), []string{"AAPL"})
	if err == nil {
		t.Fatal("Prices() should return error for HTTP error")
	}
}

func TestEarnings_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Earnings(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("Earnings() should return error for HTTP error")
	}
}

func TestNews_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.News(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("News() should return error for HTTP error")
	}
}

func TestBulkCandles_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.BulkCandles(context.Background(), []string{"AAPL"})
	if err == nil {
		t.Fatal("BulkCandles() should return error for HTTP error")
	}
}

func TestCandles_DateRangeSplitting(t *testing.T) {
	// Track how many requests were made
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "9999")
		// Return a single candle per chunk, timestamped at the chunk's own
		// from date so each chunk contributes a distinct candle.
		ft, err := time.Parse("2006-01-02", r.URL.Query().Get("from"))
		if err != nil {
			t.Errorf("unparseable from param: %v", err)
		}
		resp := map[string]any{
			"s": "ok",
			"t": []int64{ft.Unix()},
			"o": []float64{100.0},
			"h": []float64{105.0},
			"l": []float64{95.0},
			"c": []float64{102.0},
			"v": []int64{1000},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := newTestService(server.URL)

	// Request 2.5 years of 5-min candles — should be split into 3 chunks
	from := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 7, 1, 0, 0, 0, 0, time.UTC)
	candles, _, err := svc.Candles(context.Background(),
		"AAPL",
		stocks.WithResolution(stocks.Resolution5Min),
		stocks.WithCandleWindow(stocks.Between(from, to)),
	)
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}

	// Should have made 3 requests (3 year-sized chunks)
	if rc := atomic.LoadInt32(&requestCount); rc != 3 {
		t.Errorf("requestCount = %d, want 3", rc)
	}

	// Should have 3 candles (1 per chunk)
	if len(candles) != 3 {
		t.Errorf("len(candles) = %d, want 3", len(candles))
	}
}

// TestCandles_SplitChunksCarryEveryOption pins that a chunked request sends
// the same parameters as an unsplit one. candlesSplit used to build each
// chunk's options field by field, so any option it forgot to list was
// silently dropped from every chunk. It now copies the whole option set;
// asserting on every wire parameter makes the next dropped option fail here.
func TestCandles_SplitChunksCarryEveryOption(t *testing.T) {
	var mu sync.Mutex
	var chunks []url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		chunks = append(chunks, r.URL.Query())
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s": "ok", "t": []int64{1}, "o": []float64{1}, "h": []float64{1},
			"l": []float64{1}, "c": []float64{1}, "v": []int64{1},
		})
	}))
	defer server.Close()

	svc := newTestService(server.URL)

	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, _, err := svc.Candles(context.Background(), "RY",
		stocks.WithResolution(stocks.Resolution1Hour),
		stocks.WithCandleWindow(stocks.Between(from, to)),
		stocks.WithCandleExtended(true),
		stocks.WithCandleAdjustSplits(false),
		stocks.WithCandleAdjustDividends(false),
	); err != nil {
		t.Fatalf("Candles() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(chunks) < 2 {
		t.Fatalf("got %d requests, want a split into several chunks", len(chunks))
	}
	want := map[string]string{
		"extended":        "true",
		"adjustsplits":    "false",
		"adjustdividends": "false",
	}
	for i, got := range chunks {
		for key, val := range want {
			if got.Get(key) != val {
				t.Errorf("chunk %d: %s = %q, want %q", i, key, got.Get(key), val)
			}
		}
		// The resolution rides in the path, not the query — the API ignores
		// the query parameter entirely (verified live: /candles/D/ with
		// resolution=60 still returns daily candles).
		if _, ok := got["resolution"]; ok {
			t.Errorf("chunk %d: params = %v, want no resolution key", i, got)
		}
	}
}

func TestCandles_DailyResolution_NoSplit(t *testing.T) {
	// Daily resolution should NOT split even for >1 year ranges
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "9999")
		resp := map[string]any{
			"s": "ok",
			"t": []int64{1704067200},
			"o": []float64{100.0},
			"h": []float64{105.0},
			"l": []float64{95.0},
			"c": []float64{102.0},
			"v": []int64{1000},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := newTestService(server.URL)

	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _, err := svc.Candles(context.Background(),
		"AAPL",
		stocks.WithResolution(stocks.ResolutionDaily),
		stocks.WithCandleWindow(stocks.Between(from, to)),
	)
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}

	// Should only make 1 request (no splitting for daily)
	if rc := atomic.LoadInt32(&requestCount); rc != 1 {
		t.Errorf("requestCount = %d, want 1", rc)
	}
}

// candleChunkHandler returns a handler that records each request's from/to
// range under mu and serves the JSON produced by respond(from, to).
func candleChunkHandler(t *testing.T, mu *sync.Mutex, ranges *[][2]time.Time, respond func(from, to time.Time) map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ft, errF := time.Parse("2006-01-02", r.URL.Query().Get("from"))
		tt, errT := time.Parse("2006-01-02", r.URL.Query().Get("to"))
		if errF != nil || errT != nil {
			t.Errorf("unparseable from/to params: %v %v", errF, errT)
		}
		mu.Lock()
		*ranges = append(*ranges, [2]time.Time{ft, tt})
		mu.Unlock()

		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "9999")
		body := respond(ft, tt)
		if body == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}
}

func candleBody(timestamps ...int64) map[string]any {
	o := make([]float64, len(timestamps))
	h := make([]float64, len(timestamps))
	l := make([]float64, len(timestamps))
	c := make([]float64, len(timestamps))
	v := make([]int64, len(timestamps))
	for i := range timestamps {
		o[i], h[i], l[i], c[i], v[i] = 100, 105, 95, 102, 1000
	}
	return map[string]any{"s": "ok", "t": timestamps, "o": o, "h": h, "l": l, "c": c, "v": v}
}

func TestCandles_SplitChunksAreDisjoint(t *testing.T) {
	// The API treats date-only from/to as inclusive on both ends (verified
	// live), so consecutive chunks must never share a boundary day: sharing
	// it fetches that whole day twice and duplicates its candles.
	var mu sync.Mutex
	var ranges [][2]time.Time
	server := httptest.NewServer(candleChunkHandler(t, &mu, &ranges, func(from, _ time.Time) map[string]any {
		return candleBody(from.Unix())
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 7, 1, 0, 0, 0, 0, time.UTC)
	_, _, err := svc.Candles(context.Background(), "AAPL",
		stocks.WithResolution(stocks.Resolution1Min),
		stocks.WithCandleWindow(stocks.Between(from, to)),
	)
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}

	sort.Slice(ranges, func(i, j int) bool { return ranges[i][0].Before(ranges[j][0]) })
	if len(ranges) != 4 {
		t.Fatalf("got %d chunks, want 4: %v", len(ranges), ranges)
	}
	if !ranges[0][0].Equal(from) {
		t.Errorf("first chunk from = %v, want %v", ranges[0][0], from)
	}
	if last := ranges[len(ranges)-1][1]; !last.Equal(to) {
		t.Errorf("last chunk to = %v, want %v", last, to)
	}
	for i, r := range ranges {
		if span := r[1].Sub(r[0]); span > 366*24*time.Hour {
			t.Errorf("chunk %d spans %v, want <= 366 days", i, span)
		}
		if i == 0 {
			continue
		}
		if want := ranges[i-1][1].AddDate(0, 0, 1); !r[0].Equal(want) {
			t.Errorf("chunk %d from = %v, want %v (day after chunk %d's to=%v)", i, r[0], want, i-1, ranges[i-1][1])
		}
	}
}

// TestCandles_SplitDoesNotDropLastDayWhenFromTimeOfDayIsLater reproduces a
// regression in the disjoint-chunking fix: the loop compared full instants
// (current.After(to)), not calendar dates. When from's wall-clock time (or
// zone offset) put it later than to's, the final iteration's current ended
// up "after" to one day early — even though its calendar date matched — and
// that last calendar day was silently dropped instead of ever being
// requested. Verified failing before the fix: from 09:30 UTC to midnight UTC
// across exactly 3 years dropped 2023-01-01 entirely (0 requests for it).
func TestCandles_SplitDoesNotDropLastDayWhenFromTimeOfDayIsLater(t *testing.T) {
	var mu sync.Mutex
	var ranges [][2]time.Time
	server := httptest.NewServer(candleChunkHandler(t, &mu, &ranges, func(from, _ time.Time) map[string]any {
		return candleBody(from.Unix())
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	from := time.Date(2020, 1, 1, 9, 30, 0, 0, time.UTC) // later time-of-day than `to`
	to := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)    // exactly 3 years later, midnight
	_, _, err := svc.Candles(context.Background(), "AAPL",
		stocks.WithResolution(stocks.Resolution1Min),
		stocks.WithCandleWindow(stocks.Between(from, to)),
	)
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}

	sort.Slice(ranges, func(i, j int) bool { return ranges[i][0].Before(ranges[j][0]) })
	if len(ranges) == 0 {
		t.Fatal("no chunk requests were made")
	}

	// The last chunk's `to` must reach the requested `to` date — the exact
	// day the old instant-comparison loop dropped.
	last := ranges[len(ranges)-1][1]
	if last.Format("2006-01-02") != to.Format("2006-01-02") {
		t.Errorf("last chunk to = %s, want %s (the requested end date)", last.Format("2006-01-02"), to.Format("2006-01-02"))
	}

	// No gap between consecutive chunks: chunk[i+1].from must be exactly one
	// day after chunk[i].to, so every calendar date in [from, to] is covered.
	for i := 1; i < len(ranges); i++ {
		want := ranges[i-1][1].AddDate(0, 0, 1)
		if !ranges[i][0].Equal(want) {
			t.Errorf("chunk %d from = %v, want %v (day after chunk %d's to=%v) — a day was skipped", i, ranges[i][0], want, i-1, ranges[i-1][1])
		}
	}
}

// TestCandles_SplitDoesNotDropLastDayAcrossZones reproduces the same
// instant-comparison bug via a from/to pair in different zones where
// "midnight" isn't a simultaneous instant (midnight Eastern is later, in
// absolute time, than midnight UTC on the same calendar date).
func TestCandles_SplitDoesNotDropLastDayAcrossZones(t *testing.T) {
	var mu sync.Mutex
	var ranges [][2]time.Time
	server := httptest.NewServer(candleChunkHandler(t, &mu, &ranges, func(from, _ time.Time) map[string]any {
		return candleBody(from.Unix())
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, timezone.Eastern) // midnight ET
	to := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)           // midnight UTC, exactly 3 years later
	_, _, err := svc.Candles(context.Background(), "AAPL",
		stocks.WithResolution(stocks.Resolution1Min),
		stocks.WithCandleWindow(stocks.Between(from, to)),
	)
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}

	sort.Slice(ranges, func(i, j int) bool { return ranges[i][0].Before(ranges[j][0]) })
	if len(ranges) == 0 {
		t.Fatal("no chunk requests were made")
	}
	last := ranges[len(ranges)-1][1]
	if last.Format("2006-01-02") != to.Format("2006-01-02") {
		t.Errorf("last chunk to = %s, want %s (the requested end date)", last.Format("2006-01-02"), to.Format("2006-01-02"))
	}
}

func TestCandles_SplitDedupesBoundaryCandle(t *testing.T) {
	// Defense in depth: even if the server returns the same candle in more
	// than one chunk (the inclusive-boundary behavior of the live API under
	// the old shared-boundary chunking), the merge must not duplicate it.
	shared := time.Date(2021, 1, 1, 14, 30, 0, 0, time.UTC).Unix()
	var mu sync.Mutex
	var ranges [][2]time.Time
	server := httptest.NewServer(candleChunkHandler(t, &mu, &ranges, func(from, _ time.Time) map[string]any {
		return candleBody(shared, from.Unix())
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC)
	candles, _, err := svc.Candles(context.Background(), "AAPL",
		stocks.WithResolution(stocks.Resolution5Min),
		stocks.WithCandleWindow(stocks.Between(from, to)),
	)
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}

	// 3 chunks x 2 candles, but the shared timestamp must survive only once:
	// 3 unique per-chunk candles + 1 shared = 4.
	if len(candles) != 4 {
		t.Errorf("len(candles) = %d, want 4 (shared candle deduplicated)", len(candles))
	}
	for i := 1; i < len(candles); i++ {
		if !candles[i].Time.After(candles[i-1].Time) {
			t.Errorf("candles[%d].Time = %v not strictly after candles[%d].Time = %v", i, candles[i].Time, i-1, candles[i-1].Time)
		}
	}
}

func TestCandles_SplitTrailingNoData(t *testing.T) {
	// A trailing NoData chunk (range ending in days with no trades) must not
	// mark a result that does have candles as NoData.
	cutoff := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	var ranges [][2]time.Time
	server := httptest.NewServer(candleChunkHandler(t, &mu, &ranges, func(from, _ time.Time) map[string]any {
		if !from.Before(cutoff) {
			return nil // 404 → NoData
		}
		return candleBody(from.Unix())
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC)
	candles, resp, err := svc.Candles(context.Background(), "AAPL",
		stocks.WithResolution(stocks.Resolution5Min),
		stocks.WithCandleWindow(stocks.Between(from, to)),
	)
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
	if len(candles) != 2 {
		t.Errorf("len(candles) = %d, want 2 (chunks before the cutoff)", len(candles))
	}
	if resp == nil {
		t.Fatal("resp is nil")
	}
	if resp.NoData {
		t.Error("resp.NoData = true, want false when candles are present")
	}
}

func TestCandles_SplitAllNoData(t *testing.T) {
	var mu sync.Mutex
	var ranges [][2]time.Time
	server := httptest.NewServer(candleChunkHandler(t, &mu, &ranges, func(_, _ time.Time) map[string]any {
		return nil // every chunk 404s
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC)
	candles, resp, err := svc.Candles(context.Background(), "AAPL",
		stocks.WithResolution(stocks.Resolution5Min),
		stocks.WithCandleWindow(stocks.Between(from, to)),
	)
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
	if candles != nil {
		t.Errorf("candles = %v, want nil when every chunk is NoData", candles)
	}
	if resp == nil {
		t.Fatal("resp is nil")
	}
	if !resp.NoData {
		t.Error("resp.NoData = false, want true when every chunk is NoData")
	}
}

// --- Convenience method tests ---

func TestGetQuote(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/bulkquotes/": map[string]any{
			"s":         "ok",
			"symbol":    []string{"AAPL"},
			"ask":       []float64{150.0},
			"askSize":   []int{100},
			"bid":       []float64{149.0},
			"bidSize":   []int{200},
			"mid":       []float64{149.5},
			"last":      []float64{149.75},
			"change":    []float64{1.0},
			"changepct": []float64{0.67},
			"volume":    []int64{50000},
			"updated":   []int64{1704067200},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	quote, err := svc.GetQuote("AAPL")
	if err != nil {
		t.Fatalf("GetQuote() error = %v", err)
	}
	if quote == nil {
		t.Fatal("GetQuote() returned nil quote")
	}
	if quote.Symbol != "AAPL" {
		t.Errorf("Symbol = %q, want %q", quote.Symbol, "AAPL")
	}
	if quote.Last != 149.75 {
		t.Errorf("Last = %v, want %v", quote.Last, 149.75)
	}
}

func TestGetQuotes(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/bulkquotes/": map[string]any{
			"s":         "ok",
			"symbol":    []string{"AAPL", "MSFT"},
			"ask":       []float64{150.0, 375.0},
			"askSize":   []int{100, 50},
			"bid":       []float64{149.0, 374.0},
			"bidSize":   []int{200, 100},
			"mid":       []float64{149.5, 374.5},
			"last":      []float64{149.75, 374.75},
			"change":    []float64{1.0, 2.0},
			"changepct": []float64{0.67, 0.54},
			"volume":    []int64{50000, 30000},
			"updated":   []int64{1704067200, 1704067200},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	quotes, err := svc.GetQuotes("AAPL", "MSFT")
	if err != nil {
		t.Fatalf("GetQuotes() error = %v", err)
	}
	if len(quotes) != 2 {
		t.Fatalf("len(quotes) = %d, want 2", len(quotes))
	}
	if quotes[0].Symbol != "AAPL" {
		t.Errorf("quotes[0].Symbol = %q, want %q", quotes[0].Symbol, "AAPL")
	}
}

func TestGetCandles(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/candles/D/AAPL/": map[string]any{
			"s": "ok",
			"o": []float64{150.0},
			"h": []float64{155.0},
			"l": []float64{149.0},
			"c": []float64{153.0},
			"v": []int64{1000000},
			"t": []int64{1704067200},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	candles, err := svc.GetCandles("AAPL")
	if err != nil {
		t.Fatalf("GetCandles() error = %v", err)
	}
	if len(candles) != 1 {
		t.Fatalf("len(candles) = %d, want 1", len(candles))
	}
	if candles[0].Open != 150.0 {
		t.Errorf("Open = %v, want %v", candles[0].Open, 150.0)
	}
}

func TestGetPrices(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/prices/AAPL/": map[string]any{
			"s":       "ok",
			"symbol":  []string{"AAPL"},
			"mid":     []float64{150.0},
			"updated": []int64{1704067200},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	prices, err := svc.GetPrices("AAPL")
	if err != nil {
		t.Fatalf("GetPrices() error = %v", err)
	}
	if len(prices) != 1 {
		t.Fatalf("len(prices) = %d, want 1", len(prices))
	}
	if prices[0].Symbol != "AAPL" {
		t.Errorf("Symbol = %q, want %q", prices[0].Symbol, "AAPL")
	}
}

func TestGetEarnings(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/earnings/AAPL/": map[string]any{
			"s":              "ok",
			"symbol":         []string{"AAPL"},
			"fiscalYear":     []int{2024},
			"fiscalQuarter":  []int{1},
			"date":           []int64{1704067200},
			"reportDate":     []int64{1704067200},
			"reportTime":     []string{"after close"},
			"currency":       []string{"USD"},
			"reportedEPS":    []float64{2.18},
			"estimatedEPS":   []float64{2.10},
			"surpriseEPS":    []float64{0.08},
			"surpriseEPSpct": []float64{3.81},
			"updated":        []int64{1704067200},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	earnings, err := svc.GetEarnings("AAPL")
	if err != nil {
		t.Fatalf("GetEarnings() error = %v", err)
	}
	if len(earnings) != 1 {
		t.Fatalf("len(earnings) = %d, want 1", len(earnings))
	}
	if earnings[0].Symbol != "AAPL" {
		t.Errorf("Symbol = %q, want %q", earnings[0].Symbol, "AAPL")
	}
}

func TestGetNews(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/news/AAPL/": map[string]any{
			"s":               "ok",
			"symbol":          []string{"AAPL"},
			"headline":        []string{"Apple reports earnings"},
			"content":         []string{"Content here"},
			"source":          []string{"Reuters"},
			"publicationDate": []int64{1704067200},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	news, err := svc.GetNews("AAPL")
	if err != nil {
		t.Fatalf("GetNews() error = %v", err)
	}
	if len(news) != 1 {
		t.Fatalf("len(news) = %d, want 1", len(news))
	}
	if news[0].Headline != "Apple reports earnings" {
		t.Errorf("Headline = %q, want %q", news[0].Headline, "Apple reports earnings")
	}
}

// --- WithPriceExtended option test ---

func TestPrices_WithPriceExtended(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("extended") != "true" {
			t.Errorf("extended = %q, want true", r.URL.Query().Get("extended"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "9999")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":       "ok",
			"symbol":  []string{"AAPL"},
			"mid":     []float64{150.0},
			"updated": []int64{1704067200},
		})
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	prices, _, err := svc.Prices(context.Background(), []string{"AAPL"}, stocks.WithPriceExtended(true))
	if err != nil {
		t.Fatalf("Prices() with WithPriceExtended error = %v", err)
	}
	if len(prices) != 1 {
		t.Fatalf("len(prices) = %d, want 1", len(prices))
	}
}

// --- Empty symbols test ---

func TestQuotes_EmptySymbols(t *testing.T) {
	svc := newTestService("http://unused")
	_, _, err := svc.Quotes(context.Background(), []string{})
	if err == nil {
		t.Fatal("Quotes() with empty slice should return error")
	}
}

// --- 404 (NoData) tests ---

func TestQuote_404(t *testing.T) {
	// No matching path in responses map, so mockServer returns 404
	server := mockServer(t, map[string]any{})
	defer server.Close()

	svc := newTestService(server.URL)
	quote, resp, err := svc.Quote(context.Background(), "INVALID")
	if err != nil {
		t.Fatalf("Quote() 404 should not return error, got %v", err)
	}
	if quote != nil {
		t.Error("Quote() 404 should return nil quote")
	}
	if resp == nil {
		t.Fatal("Quote() 404 should return a response")
	}
}

func TestCandles_404(t *testing.T) {
	server := mockServer(t, map[string]any{})
	defer server.Close()

	svc := newTestService(server.URL)
	candles, resp, err := svc.Candles(context.Background(), "INVALID")
	if err != nil {
		t.Fatalf("Candles() 404 should not return error, got %v", err)
	}
	if candles != nil {
		t.Error("Candles() 404 should return nil candles")
	}
	if resp == nil {
		t.Fatal("Candles() 404 should return a response")
	}
}

func TestPrices_404(t *testing.T) {
	server := mockServer(t, map[string]any{})
	defer server.Close()

	svc := newTestService(server.URL)
	prices, resp, err := svc.Prices(context.Background(), []string{"INVALID"})
	if err != nil {
		t.Fatalf("Prices() 404 should not return error, got %v", err)
	}
	if prices != nil {
		t.Error("Prices() 404 should return nil prices")
	}
	if resp == nil {
		t.Fatal("Prices() 404 should return a response")
	}
}

func TestEarnings_404(t *testing.T) {
	server := mockServer(t, map[string]any{})
	defer server.Close()

	svc := newTestService(server.URL)
	earnings, resp, err := svc.Earnings(context.Background(), "INVALID")
	if err != nil {
		t.Fatalf("Earnings() 404 should not return error, got %v", err)
	}
	if earnings != nil {
		t.Error("Earnings() 404 should return nil earnings")
	}
	if resp == nil {
		t.Fatal("Earnings() 404 should return a response")
	}
}

func TestNews_404(t *testing.T) {
	server := mockServer(t, map[string]any{})
	defer server.Close()

	svc := newTestService(server.URL)
	news, resp, err := svc.News(context.Background(), "INVALID")
	if err != nil {
		t.Fatalf("News() 404 should not return error, got %v", err)
	}
	if news != nil {
		t.Error("News() 404 should return nil news")
	}
	if resp == nil {
		t.Fatal("News() 404 should return a response")
	}
}

func TestBulkCandles_404(t *testing.T) {
	server := mockServer(t, map[string]any{})
	defer server.Close()

	svc := newTestService(server.URL)
	candles, resp, err := svc.BulkCandles(context.Background(), []string{"INVALID"})
	if err != nil {
		t.Fatalf("BulkCandles() 404 should not return error, got %v", err)
	}
	if candles != nil {
		t.Error("BulkCandles() 404 should return nil candles")
	}
	if resp == nil {
		t.Fatal("BulkCandles() 404 should return a response")
	}
}

func TestQuote_WithExtendedFalse(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/bulkquotes/": map[string]any{
			"s":         "ok",
			"symbol":    []string{"AAPL"},
			"ask":       []float64{150.0},
			"askSize":   []int{100},
			"bid":       []float64{149.0},
			"bidSize":   []int{200},
			"mid":       []float64{149.5},
			"last":      []float64{149.75},
			"change":    []float64{1.0},
			"changepct": []float64{0.67},
			"volume":    []int64{50000},
			"updated":   []int64{1704067200},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Quote(context.Background(), "AAPL", stocks.WithExtended(false))
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
}

func TestQuotes_WithExtendedFalse(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/bulkquotes/": map[string]any{
			"s":         "ok",
			"symbol":    []string{"AAPL"},
			"ask":       []float64{150.0},
			"askSize":   []int{100},
			"bid":       []float64{149.0},
			"bidSize":   []int{200},
			"mid":       []float64{149.5},
			"last":      []float64{149.75},
			"change":    []float64{1.0},
			"changepct": []float64{0.67},
			"volume":    []int64{50000},
			"updated":   []int64{1704067200},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Quotes(context.Background(), []string{"AAPL"}, stocks.WithQuotesExtended(false))
	if err != nil {
		t.Fatalf("Quotes() error = %v", err)
	}
}

func TestCandles_WithExtendedFalseAndAdjustSplitsFalse(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/candles/D/AAPL/": map[string]any{
			"s": "ok",
			"o": []float64{150.0},
			"h": []float64{155.0},
			"l": []float64{149.0},
			"c": []float64{153.0},
			"v": []int64{1000000},
			"t": []int64{1704067200},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Candles(context.Background(), "AAPL",
		stocks.WithCandleExtended(false),
		stocks.WithCandleAdjustSplits(false),
	)
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
}

func TestPrices_WithPriceExtendedFalse(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/prices/AAPL/": map[string]any{
			"s":       "ok",
			"symbol":  []string{"AAPL"},
			"price":   []float64{150.0},
			"updated": []int64{1704067200},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Prices(context.Background(), []string{"AAPL"}, stocks.WithPriceExtended(false))
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}
}

func TestBulkCandles_WithSnapshotFalse(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/bulkcandles/D/": map[string]any{
			"s":      "ok",
			"symbol": []string{"AAPL"},
			"o":      []float64{150.0},
			"h":      []float64{155.0},
			"l":      []float64{149.0},
			"c":      []float64{153.0},
			"v":      []int64{1000000},
			"t":      []int64{1704067200},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.BulkCandles(context.Background(), []string{"AAPL"},
		stocks.WithSnapshot(false),
		stocks.WithAdjustSplits(false),
	)
	if err != nil {
		t.Fatalf("BulkCandles() error = %v", err)
	}
}

func TestCandles_SplitError(t *testing.T) {
	// Test date range splitting with an error in one chunk
	attempts := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "9999")
		atomic.AddInt32(&attempts, 1)
		count := atomic.LoadInt32(&attempts)
		if count == 1 {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"s": "ok",
				"o": []float64{150.0},
				"h": []float64{155.0},
				"l": []float64{149.0},
				"c": []float64{153.0},
				"v": []int64{1000000},
				"t": []int64{1704067200},
			})
		} else {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"s": "error", "errmsg": "bad"})
		}
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC) // > 1 year to trigger split
	_, _, err := svc.Candles(context.Background(), "AAPL",
		stocks.WithResolution(stocks.Resolution1Min),
		stocks.WithCandleWindow(stocks.Between(from, to)),
	)
	if err == nil {
		t.Fatal("Candles() should return error when a chunk fails")
	}
}

// TestCandlesSplit_FirstErrorCancelsSiblings pins ADR-014's cancellation on
// the candle chunk splitter specifically: the conformance commit that added
// first-error cancellation (see options.Quotes) touched both candlesSplit
// and options.Quotes but only got a regression test for the latter —
// deleting candlesSplit's cancel() call left the whole suite green. Once a
// chunk fails hard, in-flight sibling chunk fetches must be canceled
// instead of burning API credits, and the ROOT error (not a cancellation
// echo) must surface to the caller.
func TestCandlesSplit_FirstErrorCancelsSiblings(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("from") == "2020-01-01" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"s":"error","errmsg":"boom"}`))
			return
		}
		// Sibling chunks block until canceled by the batch, or a bounded
		// fallback fires — so a missing cancellation fails the elapsed-time
		// assertion below quickly instead of hanging until the
		// package-level test timeout (T-2's lesson, applied here too).
		select {
		case <-release:
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok","t":[1704067200],"o":[100],"h":[101],"l":[99],"c":[100.5],"v":[1000]}`))
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC) // >1yr intraday -> split into 3 chunks

	start := time.Now()
	_, _, err := svc.Candles(context.Background(), "AAPL",
		stocks.WithResolution(stocks.Resolution1Min),
		stocks.WithCandleWindow(stocks.Between(from, to)),
	)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Candles() should surface the failing chunk's error")
	}
	if errors.Is(err, context.Canceled) {
		t.Errorf("Candles() returned a cancellation echo (%v); want the root 400 error", err)
	}
	// Without cancellation, siblings block until the 3s fallback fires; with
	// it, the batch returns promptly after the first failure.
	if elapsed > 1*time.Second {
		t.Errorf("Candles() took %v; sibling chunk fetches were not canceled", elapsed)
	}
}

func TestQuote_EmptyQuotesList(t *testing.T) {
	// API returns "ok" but with empty arrays - should return QuoteNotFoundError
	server := mockServer(t, map[string]any{
		"/stocks/bulkquotes/": map[string]any{
			"s":         "ok",
			"symbol":    []string{},
			"ask":       []float64{},
			"askSize":   []int{},
			"bid":       []float64{},
			"bidSize":   []int{},
			"mid":       []float64{},
			"last":      []float64{},
			"change":    []float64{},
			"changepct": []float64{},
			"volume":    []int64{},
			"updated":   []int64{},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Quote(context.Background(), "NONEXIST")
	if err == nil {
		t.Fatal("Quote() should return error for empty quotes list")
	}
}

func TestQuote_APIError(t *testing.T) {
	// API returns non-ok status
	server := mockServer(t, map[string]any{
		"/stocks/bulkquotes/": map[string]any{
			"s": "error",
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Quote(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("Quote() should return error for non-ok status")
	}
}

func TestCandles_APIStatusError(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/candles/D/AAPL/": map[string]any{
			"s": "error",
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Candles(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("Candles() should return error for non-ok status")
	}
}

func TestQuotes_404(t *testing.T) {
	server := mockServer(t, map[string]any{})
	defer server.Close()

	svc := newTestService(server.URL)
	quotes, resp, err := svc.Quotes(context.Background(), []string{"INVALID"})
	if err != nil {
		t.Fatalf("Quotes() 404 should not return error, got %v", err)
	}
	if quotes != nil {
		t.Error("Quotes() 404 should return nil quotes")
	}
	if resp == nil {
		t.Fatal("Quotes() 404 should return a response")
	}
}

func TestQuotes_NonOkStatus(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/bulkquotes/": map[string]any{
			"s": "no_data",
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Quotes(context.Background(), []string{"AAPL"})
	if err == nil {
		t.Fatal("Quotes() should return error for non-ok status")
	}
}

func TestCandles_WithCountback(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/candles/D/AAPL/": map[string]any{
			"s": "ok",
			"o": []float64{150.0},
			"h": []float64{155.0},
			"l": []float64{149.0},
			"c": []float64{153.0},
			"v": []int64{1000000},
			"t": []int64{1704067200},
		},
	})
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Candles(context.Background(), "AAPL", stocks.WithCandleWindow(stocks.LastN(10)))
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
}

func TestEarnings_PathInjection(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"s": "ok", "symbol": []string{}})
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Earnings(context.Background(), "AAPL/../../user")
	if err != nil {
		t.Fatalf("Earnings() error = %v", err)
	}

	want := "/v1/stocks/earnings/AAPL%2F..%2F..%2Fuser/"
	if gotPath != want {
		t.Errorf("request path = %q, want %q (symbol must not escape its path segment)", gotPath, want)
	}
}

func TestCandles_InvalidWindow(t *testing.T) {
	svc := newTestService("http://unused")

	// from after to must fail validation before any network call.
	_, _, err := svc.Candles(context.Background(), "AAPL", stocks.WithCandleWindow(stocks.Between(
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))))
	if err == nil {
		t.Fatal("Candles() with from>to should return validation error")
	}
}

func TestEarnings_InvalidWindow(t *testing.T) {
	svc := newTestService("http://unused")

	_, _, err := svc.Earnings(context.Background(), "AAPL", stocks.WithEarningsWindow(stocks.Between(
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))))
	if err == nil {
		t.Fatal("Earnings() with from>to should return validation error")
	}
}

func TestNews_InvalidWindow(t *testing.T) {
	svc := newTestService("http://unused")

	_, _, err := svc.News(context.Background(), "AAPL", stocks.WithNewsWindow(stocks.Between(
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))))
	if err == nil {
		t.Fatal("News() with from>to should return validation error")
	}
}

func TestCandles_AdjustParamsFalse(t *testing.T) {
	var gotSplits, gotDividends string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSplits = r.URL.Query().Get("adjustsplits")
		gotDividends = r.URL.Query().Get("adjustdividends")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s": "ok",
			"t": []int64{1704067200},
			"o": []float64{150.0},
			"h": []float64{155.0},
			"l": []float64{149.0},
			"c": []float64{154.0},
			"v": []int64{1000000},
		})
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Candles(context.Background(), "AAPL",
		stocks.WithCandleExtended(false),
		stocks.WithCandleAdjustSplits(false),
		stocks.WithCandleAdjustDividends(false))
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
	if gotSplits != "false" {
		t.Errorf("adjustsplits = %q, want %q", gotSplits, "false")
	}
	if gotDividends != "false" {
		t.Errorf("adjustdividends = %q, want %q", gotDividends, "false")
	}

	// The opposite polarity must emit "true" on the wire.
	_, _, err = svc.Candles(context.Background(), "AAPL",
		stocks.WithCandleAdjustSplits(true))
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
	if gotSplits != "true" {
		t.Errorf("adjustsplits = %q, want %q", gotSplits, "true")
	}
}

func TestBulkCandles_AdjustParamsFalse(t *testing.T) {
	var gotSplits, gotDividends string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSplits = r.URL.Query().Get("adjustsplits")
		gotDividends = r.URL.Query().Get("adjustdividends")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":      "ok",
			"symbol": []string{"AAPL"},
			"t":      []int64{1704067200},
			"o":      []float64{150.0},
			"h":      []float64{155.0},
			"l":      []float64{149.0},
			"c":      []float64{154.0},
			"v":      []int64{1000000},
		})
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.BulkCandles(context.Background(), []string{"AAPL"},
		stocks.WithAdjustSplits(false),
		stocks.WithAdjustDividends(true))
	if err != nil {
		t.Fatalf("BulkCandles() error = %v", err)
	}
	if gotSplits != "false" {
		t.Errorf("adjustsplits = %q, want %q", gotSplits, "false")
	}
	if gotDividends != "true" {
		t.Errorf("adjustdividends = %q, want %q", gotDividends, "true")
	}
}

// TestEarnings_WithReport pins the report parameter on the wire. The API
// declares it but does not currently act on it (verified live 2026-08-11); it
// is exposed for parity with sdk-java so a caller's request survives unchanged
// once the API honors it.
func TestEarnings_WithReport(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok","symbol":["AAPL"],"fiscalYear":[2024],"fiscalQuarter":[1],` +
			`"date":[1704124800],"reportDate":[1706716800],"reportTime":["after close"],"currency":["USD"]}`))
	}))
	defer server.Close()

	svc := newTestService(server.URL)
	_, _, err := svc.Earnings(context.Background(), "AAPL", stocks.WithEarningsReport("2024-Q1"))
	if err != nil {
		t.Fatalf("Earnings() error = %v", err)
	}
	if got := gotQuery.Get("report"); got != "2024-Q1" {
		t.Errorf("report = %q, want 2024-Q1", got)
	}

	// An empty string leaves the parameter unset.
	_, _, err = svc.Earnings(context.Background(), "AAPL", stocks.WithEarningsReport(""))
	if err != nil {
		t.Fatalf("Earnings() error = %v", err)
	}
	if gotQuery.Has("report") {
		t.Errorf("report = %q, want the parameter absent for an empty value", gotQuery.Get("report"))
	}
}

func TestGetBulkCandles(t *testing.T) {
	server := mockServer(t, map[string]any{
		"/stocks/bulkcandles/D/": map[string]any{
			"s": "ok", "symbol": []string{"AAPL", "MSFT"},
			"t": []int64{1704067200, 1704067200},
			"o": []float64{170.11, 371.22}, "h": []float64{171.22, 373.44},
			"l": []float64{169.33, 370.01}, "c": []float64{170.99, 372.55},
			"v": []int64{45000111, 22000333},
		},
	})
	defer server.Close()

	candles, err := newTestService(server.URL).GetBulkCandles([]string{"AAPL", "MSFT"})
	if err != nil {
		t.Fatalf("GetBulkCandles() error = %v", err)
	}
	if len(candles) != 2 {
		t.Fatalf("len = %d, want 2", len(candles))
	}
	if candles[0].Symbol != "AAPL" || candles[1].Symbol != "MSFT" {
		t.Errorf("symbols = %q/%q, want AAPL/MSFT", candles[0].Symbol, candles[1].Symbol)
	}
}
