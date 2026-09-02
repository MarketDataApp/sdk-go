package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MarketDataApp/sdk-go/v2/examples/tuitest"
	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

// fixedNow is the frozen clock value used by every test that injects
// m.now, so lastRefresh/suspendedUntil comparisons are exact rather than
// racing wall-clock time.
var fixedNow = time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)

// fullRoutes returns a route map covering every endpoint the model's fetch
// commands touch, so Init() and the various key-driven fetches all resolve
// against one shared mock server. Payload shapes mirror fetch_test.go.
func fullRoutes() map[string]any {
	return map[string]any{
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
		"/stocks/prices/": map[string]any{
			"s":         "ok",
			"symbol":    []string{"AAPL", "MSFT"},
			"mid":       []float64{150.225, 375.475},
			"change":    []float64{1.50, 2.25},
			"changepct": []float64{1.01, 0.60},
			"updated":   []int64{1704067200, 1704067200},
		},
		"/stocks/candles/D/AAPL/": candlePayload(),
		"/stocks/candles/5/AAPL/": candlePayload(),
		"/stocks/candles/W/AAPL/": candlePayload(),
		"/funds/candles/D/VFINX/": map[string]any{
			"s": "ok",
			"t": []int64{1704067200, 1704153600},
			"o": []float64{398.0, 399.0},
			"h": []float64{400.0, 401.0},
			"l": []float64{397.0, 398.5},
			"c": []float64{399.5, 400.5},
		},
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
		"/stocks/news/AAPL/": map[string]any{
			"s":               "ok",
			"symbol":          []string{"AAPL", "AAPL"},
			"headline":        []string{"Headline one", "Headline two"},
			"content":         []string{"Content one", "Content two"},
			"source":          []string{"https://example.com/1", "https://example.com/2"},
			"publicationDate": []int64{1704067200, 1704153600},
		},
		"/markets/status/": map[string]any{
			"s":      "ok",
			"date":   []int64{1704067200},
			"status": []string{"open"},
		},
		"/status/": map[string]any{
			"s":            "ok",
			"service":      []string{"api"},
			"status":       []string{"online"},
			"online":       []bool{true},
			"uptimePct30d": []float64{0.999},
			"uptimePct90d": []float64{0.998},
			"updated":      []int64{1704067200},
		},
		// The /headers/ endpoint echoes the request's own headers as a flat
		// object (no "headers" envelope key).
		"/headers/": map[string]any{"Authorization": "Bearer ***"},
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
		"/user/": map[string]any{
			"x-ratelimit-requests-remaining": 9999,
			"x-ratelimit-requests-limit":     100000,
			"x-options-data-permissions":     "OPRA data delayed 15 minutes",
		},
	}
}

// baseCfg builds an appConfig for tests that don't care about refresh
// cadence beyond it being non-zero (Init/tick tests override refresh
// directly where the value matters).
func baseCfg(symbols []string, funds map[string]bool) appConfig {
	if funds == nil {
		funds = map[string]bool{}
	}
	return appConfig{symbols: symbols, funds: funds, refresh: 5 * time.Second}
}

// --- newModel / demo mode ---

func TestNewModel_DemoMode(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)

	m := newModel(client, baseCfg([]string{"AAPL"}, nil), true)

	if !m.demoMode {
		t.Error("demoMode = false, want true")
	}
}

func TestNewModel_NonDemoMode(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)

	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	if m.demoMode {
		t.Error("demoMode = true, want false")
	}
}

// --- Init ---

func TestInit_NonDemoMode_BatchesFetchesAndTicks(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)

	m := newModel(client, baseCfg([]string{"AAPL", "VFINX"}, map[string]bool{"VFINX": true}), false)
	m.now = func() time.Time { return fixedNow }

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil cmd")
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Init() cmd produced %T, want tea.BatchMsg", msg)
	}

	// refreshCmds(1: AAPL only) + detailCmds(4: candles, quote, earnings, news)
	// + marketStatus(1) + user(1) + ticks(2) = 9
	wantLen := 9
	if len(batch) != wantLen {
		t.Fatalf("len(batch) = %d, want %d", len(batch), wantLen)
	}

	fetchCmds := batch[:len(batch)-2]
	tickCmds := batch[len(batch)-2:]
	for i, c := range tickCmds {
		if c == nil {
			t.Errorf("tickCmds[%d] = nil, want non-nil (never executed in tests)", i)
		}
	}

	var sawQuotes, sawCandles, sawDetailQuote, sawEarnings, sawNews, sawMarket, sawUser bool
	for _, c := range fetchCmds {
		switch got := c().(type) {
		case quotesMsg:
			sawQuotes = true
		case candlesMsg:
			sawCandles = true
			if got.symbol != "AAPL" {
				t.Errorf("candlesMsg.symbol = %q, want AAPL", got.symbol)
			}
		case detailQuoteMsg:
			sawDetailQuote = true
		case earningsMsg:
			sawEarnings = true
		case newsMsg:
			sawNews = true
		case marketStatusMsg:
			sawMarket = true
		case userMsg:
			sawUser = true
		default:
			t.Errorf("unexpected fetch msg type %T", got)
		}
	}
	if !sawQuotes || !sawCandles || !sawDetailQuote || !sawEarnings || !sawNews || !sawMarket || !sawUser {
		t.Errorf("Init() batch missing an expected fetch: quotes=%v candles=%v detailQuote=%v earnings=%v news=%v market=%v user=%v",
			sawQuotes, sawCandles, sawDetailQuote, sawEarnings, sawNews, sawMarket, sawUser)
	}
}

func TestInit_DemoMode_SkipsFetchUser(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)

	m := newModel(client, baseCfg([]string{"AAPL"}, nil), true)
	m.now = func() time.Time { return fixedNow }

	msg := m.Init()()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Init() cmd produced %T, want tea.BatchMsg", msg)
	}

	// refreshCmds(1) + detailCmds(4) + marketStatus(1) + ticks(2), no user = 8
	wantLen := 8
	if len(batch) != wantLen {
		t.Fatalf("len(batch) = %d, want %d (fetchUser skipped in demo mode)", len(batch), wantLen)
	}
	for _, c := range batch[:len(batch)-2] {
		if _, ok := c().(userMsg); ok {
			t.Error("Init() in demo mode included a userMsg-producing command")
		}
	}
}

func TestInit_UsePrices_FetchesPricesForRefresh(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)

	cfg := baseCfg([]string{"AAPL", "MSFT"}, nil)
	cfg.usePrices = true
	m := newModel(client, cfg, false)

	cmds := m.refreshCmds()
	if len(cmds) != 1 {
		t.Fatalf("len(refreshCmds()) = %d, want 1", len(cmds))
	}
	msg := cmds[0]()
	if _, ok := msg.(pricesMsg); !ok {
		t.Fatalf("refreshCmds()[0]() = %T, want pricesMsg", msg)
	}
}

// --- refreshCmds fund exclusion ---

func TestRefreshCmds_AllFundSymbols_ReturnsNoCommand(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"VFINX"}, map[string]bool{"VFINX": true}), false)

	cmds := m.refreshCmds()

	if cmds != nil {
		t.Errorf("refreshCmds() = %v, want nil (no non-fund symbols to fetch)", cmds)
	}
}

func TestRefreshCmds_ExcludesFundSymbols(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/stocks/bulkquotes/" || r.URL.Path == "/v1/stocks/bulkquotes/" {
			gotQuery = r.URL.Query()
		}
		w.Header().Set("X-Api-Ratelimit-Limit", "100000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "99999")
		w.Header().Set("X-Api-Ratelimit-Consumed", "1")
		w.Header().Set("X-Api-Ratelimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s": "ok", "symbol": []string{"AAPL"}, "ask": []float64{1}, "bid": []float64{1},
			"mid": []float64{1}, "last": []float64{1}, "updated": []int64{1704067200},
		})
	}))
	defer srv.Close()
	client := newTestClient(t, srv)

	m := newModel(client, baseCfg([]string{"AAPL", "VFINX"}, map[string]bool{"VFINX": true}), false)

	cmds := m.refreshCmds()
	if len(cmds) != 1 {
		t.Fatalf("len(refreshCmds()) = %d, want 1", len(cmds))
	}
	_ = cmds[0]()

	if gotQuery.Get("symbols") != "AAPL" {
		t.Errorf("bulkquotes symbols query = %q, want %q (VFINX excluded)", gotQuery.Get("symbols"), "AAPL")
	}
}

// --- tick decision methods (pure, never execute the reschedule cmd) ---

func TestRefreshTickCmds_NotSuspended_FetchesAndReschedules(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)

	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)
	m.now = func() time.Time { return fixedNow }

	cmds := m.refreshTickCmds()
	if len(cmds) != 2 {
		t.Fatalf("len(refreshTickCmds()) = %d, want 2 (fetch + reschedule)", len(cmds))
	}
	msg := cmds[0]()
	if _, ok := msg.(quotesMsg); !ok {
		t.Errorf("refreshTickCmds()[0]() = %T, want quotesMsg", msg)
	}
	if cmds[1] == nil {
		t.Error("refreshTickCmds()[1] = nil, want the reschedule tick command (never executed)")
	}
}

func TestRefreshTickCmds_Suspended_SkipsFetchButReschedules(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)

	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)
	m.now = func() time.Time { return fixedNow }
	m.suspendedUntil = fixedNow.Add(time.Hour)

	cmds := m.refreshTickCmds()
	if len(cmds) != 1 {
		t.Fatalf("len(refreshTickCmds()) = %d, want 1 (reschedule only, fetch skipped while suspended)", len(cmds))
	}
	if cmds[0] == nil {
		t.Error("refreshTickCmds()[0] = nil, want the reschedule tick command (never executed)")
	}
}

func TestRefreshTickCmds_SuspensionExpired_Fetches(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)

	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)
	m.now = func() time.Time { return fixedNow }
	m.suspendedUntil = fixedNow.Add(-time.Minute) // in the past: no longer suspended

	cmds := m.refreshTickCmds()
	if len(cmds) != 2 {
		t.Fatalf("len(refreshTickCmds()) = %d, want 2 (suspension expired)", len(cmds))
	}
}

func TestUpdate_RefreshTickMsg_ReturnsNonNilCmd(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)
	m.now = func() time.Time { return fixedNow }

	_, cmd := m.Update(refreshTickMsg(fixedNow))
	if cmd == nil {
		t.Fatal("Update(refreshTickMsg) returned nil cmd")
	}
}

func TestMarketTickCmds_FetchesAndReschedules(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	cmds := m.marketTickCmds()
	if len(cmds) != 2 {
		t.Fatalf("len(marketTickCmds()) = %d, want 2", len(cmds))
	}
	msg := cmds[0]()
	if _, ok := msg.(marketStatusMsg); !ok {
		t.Errorf("marketTickCmds()[0]() = %T, want marketStatusMsg", msg)
	}
	if cmds[1] == nil {
		t.Error("marketTickCmds()[1] = nil, want reschedule tick (never executed)")
	}
}

func TestUpdate_MarketTickMsg_ReturnsNonNilCmd(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	_, cmd := m.Update(marketTickMsg(fixedNow))
	if cmd == nil {
		t.Fatal("Update(marketTickMsg) returned nil cmd")
	}
}

// --- quotesMsg ---

func TestQuotesMsg_PopulatesRowsSetsLastRefreshAndCredits(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL", "MSFT"}, nil), false)
	m.now = func() time.Time { return fixedNow }

	msg := fetchQuotes(client, []string{"AAPL", "MSFT"})()
	qm, ok := msg.(quotesMsg)
	if !ok {
		t.Fatalf("fetch produced %T, want quotesMsg", msg)
	}

	got := tuitest.Drive(m, qm).(model)

	if q, ok := got.quotes["AAPL"]; !ok || q.Symbol != "AAPL" {
		t.Errorf("quotes[AAPL] = %+v, ok=%v, want populated", q, ok)
	}
	if !got.lastRefresh.Equal(fixedNow) {
		t.Errorf("lastRefresh = %v, want %v", got.lastRefresh, fixedNow)
	}
	if got.credits != qm.meta.RateLimit {
		t.Errorf("credits = %+v, want %+v", got.credits, qm.meta.RateLimit)
	}
}

func TestQuotesMsg_SuccessClearsStatusNote(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL", "MSFT"}, nil), false)
	m.statusNote = "no data" // stale note from an earlier failed refresh

	msg := fetchQuotes(client, []string{"AAPL", "MSFT"})()
	qm, ok := msg.(quotesMsg)
	if !ok {
		t.Fatalf("fetch produced %T, want quotesMsg", msg)
	}

	got := tuitest.Drive(m, qm).(model)
	if got.statusNote != "" {
		t.Errorf("statusNote = %q, want empty (cleared by successful quotesMsg)", got.statusNote)
	}
}

func TestQuotesMsg_NoData_LeavesRowsIntactSetsStatusNote(t *testing.T) {
	srv := mockServer(t, map[string]any{}) // no route registered -> 404 -> NoData
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)
	m.now = func() time.Time { return fixedNow }
	existing := stocks.Quote{Symbol: "AAPL", Last: 42}
	m.quotes["AAPL"] = existing

	msg := fetchQuotes(client, []string{"AAPL"})()
	qm, ok := msg.(quotesMsg)
	if !ok {
		t.Fatalf("fetch produced %T, want quotesMsg", msg)
	}
	if qm.meta == nil || !qm.meta.NoData {
		t.Fatalf("meta.NoData = %v, want true", qm.meta)
	}

	got := tuitest.Drive(m, qm).(model)

	if got.quotes["AAPL"] != existing {
		t.Errorf("quotes[AAPL] changed on NoData: got %+v, want unchanged %+v", got.quotes["AAPL"], existing)
	}
	if got.statusNote == "" {
		t.Error("statusNote = empty, want a no-data note")
	}
}

func TestPricesMsg_PopulatesRowsSetsLastRefreshAndCredits(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL", "MSFT"}, nil), false)
	m.now = func() time.Time { return fixedNow }

	msg := fetchPrices(client, []string{"AAPL", "MSFT"})()
	pm, ok := msg.(pricesMsg)
	if !ok {
		t.Fatalf("fetch produced %T, want pricesMsg", msg)
	}

	got := tuitest.Drive(m, pm).(model)

	if p, ok := got.prices["AAPL"]; !ok || p.Symbol != "AAPL" {
		t.Errorf("prices[AAPL] = %+v, ok=%v, want populated", p, ok)
	}
	if !got.lastRefresh.Equal(fixedNow) {
		t.Errorf("lastRefresh = %v, want %v", got.lastRefresh, fixedNow)
	}
	if got.credits != pm.meta.RateLimit {
		t.Errorf("credits = %+v, want %+v", got.credits, pm.meta.RateLimit)
	}
}

func TestPricesMsg_SuccessClearsStatusNote(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL", "MSFT"}, nil), false)
	m.statusNote = "no data" // stale note from an earlier failed refresh

	msg := fetchPrices(client, []string{"AAPL", "MSFT"})()
	pm, ok := msg.(pricesMsg)
	if !ok {
		t.Fatalf("fetch produced %T, want pricesMsg", msg)
	}

	got := tuitest.Drive(m, pm).(model)
	if got.statusNote != "" {
		t.Errorf("statusNote = %q, want empty (cleared by successful pricesMsg)", got.statusNote)
	}
}

func TestPricesMsg_NoData_LeavesRowsIntactSetsStatusNote(t *testing.T) {
	srv := mockServer(t, map[string]any{})
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL", "MSFT"}, nil), false)
	existing := stocks.Price{Symbol: "AAPL", Mid: 42}
	m.prices["AAPL"] = existing

	msg := fetchPrices(client, []string{"AAPL", "MSFT"})()
	pm, ok := msg.(pricesMsg)
	if !ok || pm.meta == nil || !pm.meta.NoData {
		t.Fatalf("fetch produced %+v (%T), want pricesMsg with NoData", msg, msg)
	}

	got := tuitest.Drive(m, pm).(model)

	if got.prices["AAPL"] != existing {
		t.Errorf("prices[AAPL] changed on NoData: got %+v, want unchanged %+v", got.prices["AAPL"], existing)
	}
	if got.statusNote == "" {
		t.Error("statusNote = empty, want a no-data note")
	}
}

// --- detail pane and cross-cutting message handlers ---

func TestCandlesMsg_StoresOnMatchingSymbolAndRng(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)
	m.detailNoData["candles"] = true

	msg := fetchCandles(client, "AAPL", rangeDaily)()
	cm, ok := msg.(candlesMsg)
	if !ok {
		t.Fatalf("fetch produced %T, want candlesMsg", msg)
	}

	got := tuitest.Drive(m, cm).(model)

	if len(got.candles) != 2 {
		t.Errorf("candles = %v, want 2 populated", got.candles)
	}
	if got.detailNoData["candles"] {
		t.Error("detailNoData[candles] still true after a successful fetch")
	}
}

func TestCandlesMsg_NoData_SetsDetailNoData(t *testing.T) {
	srv := mockServer(t, map[string]any{}) // no route -> 404 -> NoData
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	msg := fetchCandles(client, "AAPL", rangeDaily)()
	cm, ok := msg.(candlesMsg)
	if !ok || cm.meta == nil || !cm.meta.NoData {
		t.Fatalf("fetch produced %+v (%T), want candlesMsg with NoData", msg, msg)
	}

	got := tuitest.Drive(m, cm).(model)

	if !got.detailNoData["candles"] {
		t.Error("detailNoData[candles] = false, want true")
	}
	if got.candles != nil {
		t.Errorf("candles = %v, want nil (no data)", got.candles)
	}
}

func TestCandlesMsg_StaleSymbolIgnored(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL", "MSFT"}, nil), false)
	m.selected = 1 // MSFT now selected

	// A stale response for AAPL (the previously selected symbol) must not
	// overwrite the detail pane now showing MSFT.
	msg := fetchCandles(client, "AAPL", rangeDaily)()
	cm, ok := msg.(candlesMsg)
	if !ok {
		t.Fatalf("fetch produced %T, want candlesMsg", msg)
	}

	got := tuitest.Drive(m, cm).(model)

	if got.candles != nil {
		t.Errorf("candles = %v, want nil (stale symbol response ignored)", got.candles)
	}
}

func TestCandlesMsg_StaleRngIgnored(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)
	m.rng = rangeWeekly // user has since switched away from daily

	msg := fetchCandles(client, "AAPL", rangeDaily)()
	cm, ok := msg.(candlesMsg)
	if !ok {
		t.Fatalf("fetch produced %T, want candlesMsg", msg)
	}

	got := tuitest.Drive(m, cm).(model)

	if got.candles != nil {
		t.Errorf("candles = %v, want nil (stale rng response ignored)", got.candles)
	}
}

func TestFundCandlesMsg_StoresOnMatchingSymbol(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"VFINX"}, map[string]bool{"VFINX": true}), false)

	msg := fetchFundCandles(client, "VFINX")()
	fcm, ok := msg.(fundCandlesMsg)
	if !ok {
		t.Fatalf("fetch produced %T, want fundCandlesMsg", msg)
	}

	got := tuitest.Drive(m, fcm).(model)

	if len(got.fundCandles) != 2 {
		t.Errorf("fundCandles = %v, want 2 populated", got.fundCandles)
	}
}

func TestFundCandlesMsg_NoData_SetsDetailNoData(t *testing.T) {
	srv := mockServer(t, map[string]any{})
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"VFINX"}, map[string]bool{"VFINX": true}), false)

	msg := fetchFundCandles(client, "VFINX")()
	fcm, ok := msg.(fundCandlesMsg)
	if !ok || fcm.meta == nil || !fcm.meta.NoData {
		t.Fatalf("fetch produced %+v (%T), want fundCandlesMsg with NoData", msg, msg)
	}

	got := tuitest.Drive(m, fcm).(model)

	if !got.detailNoData["candles"] {
		t.Error("detailNoData[candles] = false, want true")
	}
}

func TestDetailQuoteMsg_StoresOnMatchingSymbol(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)
	m.detailNoData["quote"] = true

	msg := fetchDetailQuote(client, "AAPL")()
	dm, ok := msg.(detailQuoteMsg)
	if !ok {
		t.Fatalf("fetch produced %T, want detailQuoteMsg", msg)
	}

	got := tuitest.Drive(m, dm).(model)

	if got.detail == nil || got.detail.Symbol != "AAPL" {
		t.Errorf("detail = %+v, want populated for AAPL", got.detail)
	}
	if got.detailNoData["quote"] {
		t.Error("detailNoData[quote] still true after a successful fetch")
	}
}

func TestDetailQuoteMsg_NoData_SetsDetailNoData(t *testing.T) {
	srv := mockServer(t, map[string]any{})
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	msg := fetchDetailQuote(client, "AAPL")()
	dm, ok := msg.(detailQuoteMsg)
	if !ok || dm.meta == nil || !dm.meta.NoData {
		t.Fatalf("fetch produced %+v (%T), want detailQuoteMsg with NoData", msg, msg)
	}

	got := tuitest.Drive(m, dm).(model)

	if !got.detailNoData["quote"] {
		t.Error("detailNoData[quote] = false, want true")
	}
}

func TestEarningsMsg_StoresOnMatchingSymbol(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)
	m.detailNoData["earnings"] = true

	msg := fetchEarnings(client, "AAPL")()
	em, ok := msg.(earningsMsg)
	if !ok {
		t.Fatalf("fetch produced %T, want earningsMsg", msg)
	}

	got := tuitest.Drive(m, em).(model)

	if len(got.earnings) != 1 {
		t.Errorf("earnings = %v, want 1 populated", got.earnings)
	}
	if got.detailNoData["earnings"] {
		t.Error("detailNoData[earnings] still true after a successful fetch")
	}
}

func TestEarningsMsg_NoData_SetsDetailNoData(t *testing.T) {
	srv := mockServer(t, map[string]any{})
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	msg := fetchEarnings(client, "AAPL")()
	em, ok := msg.(earningsMsg)
	if !ok || em.meta == nil || !em.meta.NoData {
		t.Fatalf("fetch produced %+v (%T), want earningsMsg with NoData", msg, msg)
	}

	got := tuitest.Drive(m, em).(model)

	if !got.detailNoData["earnings"] {
		t.Error("detailNoData[earnings] = false, want true")
	}
}

func TestNewsMsg_StoresOnMatchingSymbol(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)
	m.detailNoData["news"] = true

	msg := fetchNews(client, "AAPL")()
	nm, ok := msg.(newsMsg)
	if !ok {
		t.Fatalf("fetch produced %T, want newsMsg", msg)
	}

	got := tuitest.Drive(m, nm).(model)

	if len(got.news) != 2 {
		t.Errorf("news = %v, want 2 populated", got.news)
	}
	if got.detailNoData["news"] {
		t.Error("detailNoData[news] still true after a successful fetch")
	}
}

func TestNewsMsg_NoData_SetsDetailNoData(t *testing.T) {
	srv := mockServer(t, map[string]any{})
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	msg := fetchNews(client, "AAPL")()
	nm, ok := msg.(newsMsg)
	if !ok || nm.meta == nil || !nm.meta.NoData {
		t.Fatalf("fetch produced %+v (%T), want newsMsg with NoData", msg, msg)
	}

	got := tuitest.Drive(m, nm).(model)

	if !got.detailNoData["news"] {
		t.Error("detailNoData[news] = false, want true")
	}
}

func TestMarketStatusMsg_StoresStatus(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	msg := fetchMarketStatus(client)()
	got := tuitest.Drive(m, msg).(model)

	if got.market == nil || !got.market.Open {
		t.Errorf("market = %+v, want populated and open", got.market)
	}
}

func TestStatusHistoryMsg_StoresHistory(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	msg := fetchStatusHistory(client)()
	got := tuitest.Drive(m, msg).(model)

	if len(got.history) == 0 {
		t.Error("history = empty, want populated")
	}
}

func TestUserMsg_StoresUser(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	msg := fetchUser(client)()
	got := tuitest.Drive(m, msg).(model)

	if got.user == nil {
		t.Error("user = nil, want populated")
	}
}

func TestAPIStatusMsg_StoresStatus(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	msg := fetchAPIStatus(client)()
	got := tuitest.Drive(m, msg).(model)

	if got.apiStatus == nil || !got.apiStatus.IsOnline() {
		t.Errorf("apiStatus = %+v, want populated and online", got.apiStatus)
	}
}

func TestHeadersMsg_StoresHeaders(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	msg := fetchHeaders(client)()
	got := tuitest.Drive(m, msg).(model)

	if got.headers == nil {
		t.Error("headers = nil, want populated")
	}
}

func TestBulkCandlesMsg_StoresCandles(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL", "MSFT"}, nil), false)

	msg := fetchBulkCandles(client, []string{"AAPL", "MSFT"})()
	got := tuitest.Drive(m, msg).(model)

	if len(got.bulk) != 2 {
		t.Errorf("bulk = %v, want 2 populated", got.bulk)
	}
}

// --- selection ---

func TestSelection_DownMovesAndTriggersDetailCascade(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL", "VFINX"}, map[string]bool{"VFINX": true}), false)
	m.candles = []stocks.Candle{{}} // pre-existing detail state to prove it gets reset
	m.detailNoData["quote"] = true

	newModelI, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	nm := newModelI.(model)

	if nm.selected != 1 {
		t.Fatalf("selected = %d, want 1", nm.selected)
	}
	if nm.candles != nil {
		t.Errorf("candles not reset on selection change: %+v", nm.candles)
	}
	if len(nm.detailNoData) != 0 {
		t.Errorf("detailNoData not reset on selection change: %+v", nm.detailNoData)
	}
	if cmd == nil {
		t.Fatal("Update(KeyDown) returned nil cmd")
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want tea.BatchMsg", msg)
	}
	if len(batch) != 4 {
		t.Fatalf("len(batch) = %d, want 4 (detailCmds for VFINX)", len(batch))
	}
	var sawFundCandles bool
	for _, c := range batch {
		if fc, ok := c().(fundCandlesMsg); ok {
			sawFundCandles = true
			if fc.symbol != "VFINX" {
				t.Errorf("fundCandlesMsg.symbol = %q, want VFINX", fc.symbol)
			}
		}
	}
	if !sawFundCandles {
		t.Error("detail cascade for VFINX did not include fetchFundCandles")
	}
}

func TestSelection_UpMovesAndTriggersDetailCascade(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL", "MSFT"}, nil), false)
	m.selected = 1

	newModelI, cmd := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	nm := newModelI.(model)

	if nm.selected != 0 {
		t.Fatalf("selected = %d, want 0", nm.selected)
	}
	if cmd == nil {
		t.Fatal("Update(KeyUp) returned nil cmd")
	}
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want tea.BatchMsg", cmd())
	}
	if len(batch) != 4 {
		t.Fatalf("len(batch) = %d, want 4 (detailCmds for AAPL)", len(batch))
	}
}

func TestSelection_UpAtTopIsNoOp(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL", "MSFT"}, nil), false)

	newModelI, cmd := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	nm := newModelI.(model)
	if nm.selected != 0 {
		t.Errorf("selected = %d, want 0 (already at top)", nm.selected)
	}
	if cmd != nil {
		t.Error("Update(KeyUp) at top returned a non-nil cmd, want nil (no-op)")
	}
}

func TestSelection_DownAtBottomIsNoOp(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL", "MSFT"}, nil), false)
	m.selected = 1

	newModelI, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	nm := newModelI.(model)
	if nm.selected != 1 {
		t.Errorf("selected = %d, want 1 (already at bottom)", nm.selected)
	}
	if cmd != nil {
		t.Error("Update(KeyDown) at bottom returned a non-nil cmd, want nil (no-op)")
	}
}

// --- range keys ---

func TestRangeKeys_StockSymbol_SwitchesRngAndFetchesCandles(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	newModelI, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	nm := newModelI.(model)
	if nm.rng != rangeIntraday {
		t.Errorf("rng = %v, want rangeIntraday", nm.rng)
	}
	if cmd == nil {
		t.Fatal("Update('1') returned nil cmd")
	}
	msg := cmd()
	cm, ok := msg.(candlesMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want candlesMsg", msg)
	}
	if cm.rng != rangeIntraday {
		t.Errorf("candlesMsg.rng = %v, want rangeIntraday", cm.rng)
	}
}

// TestRangeKeys_FundSymbol_AreNoOps locks in the fix for the fund-caption
// honesty bug: fund symbols always fetch daily countback-63 NAV candles
// regardless of m.rng (fetchFundCandles takes no range argument), but the
// caption used to render rangeLabelText(m.rng) anyway — so pressing '1' or
// 'w' relabeled the still-daily data as "intraday, 1d" or "weekly, 1y"
// without changing what was actually fetched or displayed. Since funds have
// only one real range, 1/d/w are no-ops for a fund-selected row: m.rng
// doesn't change and no fetch is issued.
func TestRangeKeys_FundSymbol_AreNoOps(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"VFINX"}, map[string]bool{"VFINX": true}), false)
	wantRng := m.rng

	for _, key := range []string{"1", "w", "d"} {
		newModelI, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		nm := newModelI.(model)
		if cmd != nil {
			t.Errorf("Update(%q) cmd = %v, want nil (fund symbols: range keys are no-ops)", key, cmd)
		}
		if nm.rng != wantRng {
			t.Errorf("Update(%q) rng = %v, want %v (unchanged for a fund symbol)", key, nm.rng, wantRng)
		}
		m = nm
	}
}

// TestRangeKeys_StockSymbol_UnaffectedBySiblingFund is a regression guard
// for the fix above: the fund no-op must be scoped to the selected symbol,
// not applied whenever any fund exists in the watchlist.
func TestRangeKeys_StockSymbol_UnaffectedBySiblingFund(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL", "VFINX"}, map[string]bool{"VFINX": true}), false)
	// selected defaults to 0 (AAPL), a non-fund symbol.

	newModelI, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	nm := newModelI.(model)
	if nm.rng != rangeIntraday {
		t.Errorf("rng = %v, want rangeIntraday (AAPL is not a fund)", nm.rng)
	}
	if cmd == nil {
		t.Fatal("Update('1') returned nil cmd, want a candles fetch")
	}
	if _, ok := cmd().(candlesMsg); !ok {
		t.Errorf("cmd() = %T, want candlesMsg", cmd())
	}
}

// --- add symbol ---

func TestAddSymbol_OpensModalAndSubmitsUppercased(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	m = tuitest.Drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}).(model)
	if m.modal != modalAdd {
		t.Fatalf("modal = %v, want modalAdd", m.modal)
	}

	m = tuitest.Drive(m,
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")},
	).(model)

	if m.input.Value() != "tsla" {
		t.Errorf("input.Value() = %q, want %q (raw input keeps case; uppercased on submit)", m.input.Value(), "tsla")
	}

	newModelI, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	nm := newModelI.(model)
	if nm.modal != modalNone {
		t.Errorf("modal after Enter = %v, want modalNone (input closes immediately)", nm.modal)
	}
	if cmd == nil {
		t.Fatal("Update(Enter) returned nil cmd, want validateSymbol cmd")
	}
	msg := cmd()
	got, ok := msg.(addValidatedMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want addValidatedMsg", msg)
	}
	if got.symbol != "TSLA" {
		t.Errorf("validated symbol = %q, want %q (uppercased)", got.symbol, "TSLA")
	}
}

func TestAddSymbol_EscClosesInputWithoutValidating(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)
	m.modal = modalAdd
	m.input.SetValue("tsla")

	newModelI, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	nm := newModelI.(model)
	if nm.modal != modalNone {
		t.Errorf("modal = %v, want modalNone after esc", nm.modal)
	}
	if nm.input.Value() != "" {
		t.Errorf("input.Value() = %q, want empty after esc", nm.input.Value())
	}
	if cmd != nil {
		t.Error("Update(Esc) in modalAdd returned a non-nil cmd, want nil")
	}
}

func TestAddSymbol_SubmittingBlankInputClosesWithoutValidating(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)
	m.modal = modalAdd
	m.input.SetValue("   ") // whitespace-only, trims to empty

	newModelI, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	nm := newModelI.(model)

	if nm.modal != modalNone {
		t.Errorf("modal = %v, want modalNone (input still closes)", nm.modal)
	}
	if cmd != nil {
		t.Error("Update(Enter) with blank input returned a non-nil cmd, want nil (nothing to validate)")
	}
}

func TestAddValidatedMsg_AlreadyOnWatchlist_NotDuplicated(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/stocks/bulkquotes/": map[string]any{
			"s": "ok", "symbol": []string{"AAPL"}, "ask": []float64{150},
			"bid": []float64{149}, "mid": []float64{149.5}, "last": []float64{149.8},
			"updated": []int64{1704067200},
		},
	})
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	msg := validateSymbol(client, "AAPL")()
	got, ok := msg.(addValidatedMsg)
	if !ok || got.quote == nil {
		t.Fatalf("validateSymbol result = %+v (%T), want addValidatedMsg{quote: non-nil}", msg, msg)
	}

	nm := tuitest.Drive(m, got).(model)

	if len(nm.symbols) != 1 {
		t.Errorf("symbols = %v, want unchanged (AAPL already on watchlist, not duplicated)", nm.symbols)
	}
}

// TestAddSymbol_QIsTypedNotQuit locks the orchestrator's override of the
// original "q always quits" rule: while the add-symbol input is open, 'q'
// must be forwarded to the textinput like any other rune — otherwise the
// user could never type QQQ. ctrl+c remains the universal quit.
func TestAddSymbol_QIsTypedNotQuit(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	// Open via the 'a' key so the input is focused (textinput ignores keys
	// when blurred).
	m = tuitest.Drive(m,
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")},
	).(model)

	if m.modal != modalAdd {
		t.Fatalf("modal = %v, want modalAdd still open ('q' must not quit while typing)", m.modal)
	}
	if m.input.Value() != "qqq" {
		t.Fatalf("input.Value() = %q, want %q (raw input keeps case; uppercased on submit)", m.input.Value(), "qqq")
	}

	// And the submitted symbol uppercases to QQQ.
	newModelI, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	nm := newModelI.(model)
	if nm.modal != modalNone {
		t.Errorf("modal after Enter = %v, want modalNone", nm.modal)
	}
	if cmd == nil {
		t.Fatal("Update(Enter) returned nil cmd, want validateSymbol cmd")
	}
	got, ok := cmd().(addValidatedMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want addValidatedMsg", cmd())
	}
	if got.symbol != "QQQ" {
		t.Errorf("validated symbol = %q, want QQQ", got.symbol)
	}
}

func TestAddSymbol_CtrlCStillQuitsWhileInputOpen(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)
	m = tuitest.Drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}).(model)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("Update(ctrl+c) in modalAdd returned nil cmd, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("cmd() = %T, want tea.QuitMsg", cmd())
	}
}

func TestModal_QStillQuitsInNonInputModals(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)
	m.modal = modalDiagnostics

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("Update('q') in modalDiagnostics returned nil cmd, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("cmd() = %T, want tea.QuitMsg", cmd())
	}
}

func TestAddValidatedMsg_NoData_SetsStatusNoteAndDoesNotAdd(t *testing.T) {
	srv := mockServer(t, map[string]any{}) // no route -> 404
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	msg := validateSymbol(client, "BOGUS")()
	got, ok := msg.(addValidatedMsg)
	if !ok || !got.noData {
		t.Fatalf("validateSymbol result = %+v (%T), want addValidatedMsg{noData: true}", msg, msg)
	}

	nm := tuitest.Drive(m, got).(model)

	if len(nm.symbols) != 1 {
		t.Errorf("symbols = %v, want unchanged (not added)", nm.symbols)
	}
	if nm.statusNote != "no data for BOGUS" {
		t.Errorf("statusNote = %q, want %q", nm.statusNote, "no data for BOGUS")
	}
}

func TestAddValidatedMsg_Success_AppendsAndStoresQuote(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/stocks/bulkquotes/": map[string]any{
			"s": "ok", "symbol": []string{"TSLA"}, "ask": []float64{250},
			"bid": []float64{249}, "mid": []float64{249.5}, "last": []float64{249.8},
			"updated": []int64{1704067200},
		},
	})
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	msg := validateSymbol(client, "TSLA")()
	got, ok := msg.(addValidatedMsg)
	if !ok || got.noData || got.quote == nil {
		t.Fatalf("validateSymbol result = %+v (%T), want addValidatedMsg{quote: non-nil}", msg, msg)
	}

	nm := tuitest.Drive(m, got).(model)

	if len(nm.symbols) != 2 || nm.symbols[1] != "TSLA" {
		t.Errorf("symbols = %v, want [AAPL TSLA]", nm.symbols)
	}
	if q, ok := nm.quotes["TSLA"]; !ok || q.Symbol != "TSLA" {
		t.Errorf("quotes[TSLA] = %+v, ok=%v, want stored", q, ok)
	}
}

// TestAddValidatedMsg_NilMeta_NoDataStillHandled locks the addValidatedMsg
// nil-meta contract documented in app.go: the QuoteNotFoundError path
// carries meta == nil, and Update must not panic dereferencing it.
func TestAddValidatedMsg_NilMeta_NoDataStillHandled(t *testing.T) {
	srv := mockServer(t, map[string]any{
		"/stocks/bulkquotes/": map[string]any{"s": "ok", "symbol": []string{}},
	})
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	msg := validateSymbol(client, "ZZZZ")()
	got, ok := msg.(addValidatedMsg)
	if !ok || got.meta != nil {
		t.Fatalf("validateSymbol result = %+v (%T), want addValidatedMsg with nil meta", msg, msg)
	}

	nm := tuitest.Drive(m, got).(model) // must not panic on nil meta

	if nm.statusNote != "no data for ZZZZ" {
		t.Errorf("statusNote = %q, want %q", nm.statusNote, "no data for ZZZZ")
	}
}

// --- remove symbol ---

func TestRemoveSymbol_RemovesSelectedAndClampsSelection(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL", "MSFT", "VFINX"}, map[string]bool{"VFINX": true}), false)
	m.quotes["MSFT"] = stocks.Quote{Symbol: "MSFT"}
	m.selected = 2 // VFINX, last position

	newModelI, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	nm := newModelI.(model)

	want := []string{"AAPL", "MSFT"}
	if len(nm.symbols) != 2 || nm.symbols[0] != want[0] || nm.symbols[1] != want[1] {
		t.Errorf("symbols = %v, want %v", nm.symbols, want)
	}
	if nm.funds["VFINX"] {
		t.Error("funds[VFINX] still true after removal")
	}
	if _, ok := nm.quotes["VFINX"]; ok {
		t.Error("quotes[VFINX] still present after removal")
	}
	if nm.selected != 1 {
		t.Errorf("selected = %d, want 1 (clamped to new last index)", nm.selected)
	}
	if cmd == nil {
		t.Error("Update('x') returned nil cmd, want detail cascade for new selection")
	}
}

func TestRemoveSymbol_NonLastPosition_KeepsSelectionIndex(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL", "MSFT", "VFINX"}, map[string]bool{"VFINX": true}), false)
	m.quotes["MSFT"] = stocks.Quote{Symbol: "MSFT"}
	m.selected = 1 // MSFT, middle position: no clamping needed

	newModelI, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	nm := newModelI.(model)

	want := []string{"AAPL", "VFINX"}
	if len(nm.symbols) != 2 || nm.symbols[0] != want[0] || nm.symbols[1] != want[1] {
		t.Errorf("symbols = %v, want %v", nm.symbols, want)
	}
	if _, ok := nm.quotes["MSFT"]; ok {
		t.Error("quotes[MSFT] still present after removal")
	}
	if nm.selected != 1 {
		t.Errorf("selected = %d, want 1 (index unchanged, now pointing at VFINX)", nm.selected)
	}
	if cmd == nil {
		t.Fatal("Update('x') returned nil cmd, want detail cascade for the new selection")
	}
	// The new selection is VFINX, a fund: the cascade must fetch fund candles.
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want tea.BatchMsg", cmd())
	}
	var sawFundCandles bool
	for _, c := range batch {
		if fc, ok := c().(fundCandlesMsg); ok {
			sawFundCandles = true
			if fc.symbol != "VFINX" {
				t.Errorf("fundCandlesMsg.symbol = %q, want VFINX", fc.symbol)
			}
		}
	}
	if !sawFundCandles {
		t.Error("detail cascade for VFINX did not include fetchFundCandles")
	}
}

func TestRemoveSymbol_GuardsLastRemaining(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	newModelI, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	nm := newModelI.(model)

	if len(nm.symbols) != 1 {
		t.Errorf("symbols = %v, want unchanged (guard against removing last)", nm.symbols)
	}
	if nm.statusNote != "cannot remove last symbol" {
		t.Errorf("statusNote = %q, want %q", nm.statusNote, "cannot remove last symbol")
	}
}

// --- modals ---

func TestModal_StatusHistory_OpensAndFetches(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	newModelI, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	nm := newModelI.(model)
	if nm.modal != modalStatusHistory {
		t.Fatalf("modal = %v, want modalStatusHistory", nm.modal)
	}
	if cmd == nil {
		t.Fatal("Update('m') returned nil cmd")
	}
	if _, ok := cmd().(statusHistoryMsg); !ok {
		t.Errorf("cmd() type = %T, want statusHistoryMsg", cmd())
	}
}

func TestModal_Diagnostics_OpensAndFetchesStatusAndHeaders(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	newModelI, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	nm := newModelI.(model)
	if nm.modal != modalDiagnostics {
		t.Fatalf("modal = %v, want modalDiagnostics", nm.modal)
	}
	if cmd == nil {
		t.Fatal("Update('D') returned nil cmd")
	}
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("cmd() = %T, want tea.BatchMsg", cmd())
	}
	var sawStatus, sawHeaders bool
	for _, c := range batch {
		switch c().(type) {
		case apiStatusMsg:
			sawStatus = true
		case headersMsg:
			sawHeaders = true
		}
	}
	if !sawStatus || !sawHeaders {
		t.Errorf("diagnostics batch missing fetch: status=%v headers=%v", sawStatus, sawHeaders)
	}
}

func TestModal_Error_OpensWithoutFetch(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	newModelI, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	nm := newModelI.(model)
	if nm.modal != modalError {
		t.Fatalf("modal = %v, want modalError", nm.modal)
	}
	if cmd != nil {
		t.Error("Update('E') returned a non-nil cmd, want nil (no fetch)")
	}
}

func TestModal_Bulk_OpensAndFetchesAllSymbols(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL", "VFINX"}, map[string]bool{"VFINX": true}), false)

	newModelI, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	nm := newModelI.(model)
	if nm.modal != modalBulk {
		t.Fatalf("modal = %v, want modalBulk", nm.modal)
	}
	if cmd == nil {
		t.Fatal("Update('o') returned nil cmd")
	}
	if _, ok := cmd().(bulkCandlesMsg); !ok {
		t.Errorf("cmd() = %T, want bulkCandlesMsg", cmd())
	}
}

func TestModal_EscClosesAnyOpenModal(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)
	m.modal = modalDiagnostics

	newModelI, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	nm := newModelI.(model)
	if nm.modal != modalNone {
		t.Errorf("modal = %v, want modalNone after esc", nm.modal)
	}
}

func TestModal_WatchlistKeysInertWhileModalOpen(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL", "MSFT"}, nil), false)
	m.modal = modalDiagnostics

	for _, msg := range []tea.Msg{
		tea.KeyMsg{Type: tea.KeyDown},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")},
	} {
		newModelI, _ := m.Update(msg)
		nm := newModelI.(model)
		if nm.selected != 0 || len(nm.symbols) != 2 || nm.rng != rangeDaily {
			t.Errorf("msg %v changed watchlist state while modal open: selected=%d symbols=%v rng=%v", msg, nm.selected, nm.symbols, nm.rng)
		}
		if nm.modal != modalDiagnostics {
			t.Errorf("msg %v changed modal state: %v", msg, nm.modal)
		}
	}
}

func TestModal_QuitKeysStillWorkWhileModalOpen(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)
	m.modal = modalDiagnostics

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("Update('q') while modal open returned nil cmd, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("cmd() = %T, want tea.QuitMsg", cmd())
	}
}

// --- errMsg / rate limit suspension ---

func TestErrMsg_SetsLastErrAndOp(t *testing.T) {
	srv := statusServer(t, http.StatusUnauthorized, nil, map[string]string{"s": "error", "errmsg": "invalid token"})
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	msg := fetchQuotes(client, []string{"AAPL"})()
	em, ok := msg.(errMsg)
	if !ok {
		t.Fatalf("fetch produced %T, want errMsg", msg)
	}

	nm := tuitest.Drive(m, em).(model)

	if nm.lastErr == nil {
		t.Fatal("lastErr = nil, want set")
	}
	if nm.lastErrOp != "quotes" {
		t.Errorf("lastErrOp = %q, want %q", nm.lastErrOp, "quotes")
	}
}

func TestErrMsg_RateLimitError_SuspendsRefreshUntilResetAt(t *testing.T) {
	resetAt := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	srv := statusServer(t, http.StatusTooManyRequests, map[string]string{
		"X-Api-Ratelimit-Limit":     "100000",
		"X-Api-Ratelimit-Remaining": "0",
		"X-Api-Ratelimit-Reset":     fmt.Sprintf("%d", resetAt.Unix()),
	}, map[string]string{"s": "error", "errmsg": "rate limit exceeded"})
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	msg := fetchQuotes(client, []string{"AAPL"})()
	em, ok := msg.(errMsg)
	if !ok {
		t.Fatalf("fetch produced %T, want errMsg", msg)
	}
	var rle *marketdata.RateLimitError
	if !errors.As(em.err, &rle) {
		t.Fatalf("err = %v, want *marketdata.RateLimitError", em.err)
	}

	nm := tuitest.Drive(m, em).(model)

	if !nm.suspendedUntil.Equal(rle.ResetAt) {
		t.Errorf("suspendedUntil = %v, want %v", nm.suspendedUntil, rle.ResetAt)
	}

	// And the suspension actually skips the next refresh tick's fetch.
	nm.now = func() time.Time { return resetAt.Add(-time.Minute) }
	cmds := nm.refreshTickCmds()
	if len(cmds) != 1 {
		t.Errorf("len(refreshTickCmds()) = %d, want 1 (still suspended)", len(cmds))
	}
}

func TestLaterSuccessClearsLastErr(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)
	m.lastErr = errors.New("boom")
	m.lastErrOp = "quotes"

	msg := fetchQuotes(client, []string{"AAPL"})()
	nm := tuitest.Drive(m, msg).(model)

	if nm.lastErr != nil {
		t.Errorf("lastErr = %v, want nil after a later successful fetch", nm.lastErr)
	}
	if nm.lastErrOp != "" {
		t.Errorf("lastErrOp = %q, want empty after a later successful fetch", nm.lastErrOp)
	}
}

// --- force refresh / quit ---

func TestForceRefresh_ClearsSuspensionAndFetches(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)
	m.suspendedUntil = time.Now().Add(time.Hour)

	newModelI, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	nm := newModelI.(model)

	if !nm.suspendedUntil.IsZero() {
		t.Errorf("suspendedUntil = %v, want zero (cleared by force refresh)", nm.suspendedUntil)
	}
	if cmd == nil {
		t.Fatal("Update('r') returned nil cmd")
	}
	if _, ok := cmd().(quotesMsg); !ok {
		t.Errorf("cmd() = %T, want quotesMsg", cmd())
	}
}

func TestQuitKeys(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	for _, msg := range []tea.Msg{
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")},
		tea.KeyMsg{Type: tea.KeyCtrlC},
	} {
		_, cmd := m.Update(msg)
		if cmd == nil {
			t.Fatalf("Update(%v) returned nil cmd, want tea.Quit", msg)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("Update(%v) cmd() = %T, want tea.QuitMsg", msg, cmd())
		}
	}
}

// --- window size ---

func TestWindowSizeMsg_StoresDimensions(t *testing.T) {
	srv := mockServer(t, fullRoutes())
	defer srv.Close()
	client := newTestClient(t, srv)
	m := newModel(client, baseCfg([]string{"AAPL"}, nil), false)

	nm := tuitest.Drive(m, tea.WindowSizeMsg{Width: 120, Height: 40}).(model)

	if nm.width != 120 || nm.height != 40 {
		t.Errorf("width,height = %d,%d, want 120,40", nm.width, nm.height)
	}
}
