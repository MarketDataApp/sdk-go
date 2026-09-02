// Headless behavior tests for the model/Update loop, driven via
// tuitest.Drive and real fetch commands executed against httptest mocks
// (newTestClient, newMux, jsonHandler, notFoundHandler, chainPayload,
// mustCmdMsg all come from fetch_test.go — same package, same
// conventions). View output itself is out of scope here (Task 3.5); these
// tests only assert on model state and on the commands Update decides to
// issue.
package main

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MarketDataApp/sdk-go/v2/examples/tuitest"
	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

// testNow is the fixed clock injected into every model built by
// newTestModel, so expiration/DTE math in tests never depends on wall time.
var testNow = time.Date(2025, 1, 10, 9, 30, 0, 0, time.UTC)

// newTestModel builds a model backed by an httptest server running mux,
// with a fixed clock (testNow) and a long refresh interval (so the tick
// command, if a test accidentally holds onto it, is obviously never meant
// to fire during the test).
func newTestModel(t *testing.T, mux http.Handler) model {
	t.Helper()
	client := newTestClient(t, mux)
	m := newModel(client, "AAPL", time.Hour, false)
	m.now = func() time.Time { return testNow }
	return m
}

// catchAllMux returns a mux that answers every request with a bare "ok",
// for tests where no specific endpoint is expected to be called. Unlike
// newMux (fetch_test.go), it registers "/" exactly once, so it is safe to
// use when the test itself has no specific path to mock.
func catchAllMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", jsonHandler(map[string]any{"s": "ok"}))
	return mux
}

// --- newModel / demo mode ---

func TestNewModel_SymbolInputPrefilledAndFocusOnChain(t *testing.T) {
	m := newTestModel(t, catchAllMux())

	if got := m.symbolInput.Value(); got != "AAPL" {
		t.Errorf("symbolInput.Value() = %q, want AAPL", got)
	}
	if m.focus != focusChain {
		t.Errorf("focus = %v, want focusChain", m.focus)
	}
	if m.side != options.SideBoth {
		t.Errorf("side = %v, want SideBoth", m.side)
	}
	if m.window != 0.10 {
		t.Errorf("window = %v, want 0.10", m.window)
	}
}

func TestNewModel_DemoModeSetsBannerFlag(t *testing.T) {
	client := newTestClient(t, catchAllMux())
	m := newModel(client, "AAPL", 15*time.Second, true)

	if !m.demoMode {
		t.Error("demoMode = false, want true")
	}
	if got := m.symbolInput.Value(); got != "AAPL" {
		t.Errorf("symbolInput.Value() = %q, want AAPL", got)
	}
}

func TestModelViewContainsSymbol(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.symbol = "MSFT"
	got := tuitest.Drive(m, tea.WindowSizeMsg{Width: 100, Height: 40}).(model)

	if view := got.View(); !strings.Contains(view, "MSFT") {
		t.Errorf("View() = %q, want it to contain %q", view, "MSFT")
	}
}

// TestModelView_BelowMinimumSize documents View's guard for a
// too-small terminal: rather than emit a garbled or negative-width
// layout, it renders a short fixed message instead of the pane layout.
func TestModelView_BelowMinimumSize(t *testing.T) {
	m := newTestModel(t, catchAllMux())

	if view := m.View(); strings.Contains(view, "EXPIRATIONS") || view == "" {
		t.Errorf("View() before any WindowSizeMsg = %q, want the too-small fallback, not the pane layout", view)
	}
}

func TestNewModel_DefaultClockIsRealTime(t *testing.T) {
	client := newTestClient(t, catchAllMux())
	m := newModel(client, "AAPL", 15*time.Second, false)

	before := time.Now()
	got := m.now()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Errorf("now() = %v, want between %v and %v (default clock is time.Now)", got, before, after)
	}
}

// --- tea.WindowSizeMsg ---

func TestWindowSizeMsg_StoresWidthAndHeight(t *testing.T) {
	m := newTestModel(t, catchAllMux())

	got := tuitest.Drive(m, tea.WindowSizeMsg{Width: 100, Height: 40}).(model)
	if got.width != 100 || got.height != 40 {
		t.Errorf("width,height = %d,%d, want 100,40", got.width, got.height)
	}
}

// --- Init ---

func TestInit_LoadsSymbolAndSchedulesTick(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/stocks/bulkquotes/", jsonHandler(map[string]any{
		"s": "ok", "symbol": []string{"AAPL"}, "bid": []float64{184.50}, "ask": []float64{184.55},
		"mid": []float64{184.525}, "last": []float64{184.52}, "volume": []int64{1000000}, "updated": []int64{1704067200},
	}))
	mux.HandleFunc("/v1/options/expirations/AAPL/", jsonHandler(map[string]any{
		"s": "ok", "expirations": []int64{testNow.AddDate(0, 0, 7).Unix()},
	}))
	mux.HandleFunc("/", jsonHandler(map[string]any{"s": "ok"}))
	m := newTestModel(t, mux)

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() cmd = nil, want non-nil")
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok || len(batch) != 3 {
		t.Fatalf("Init() msg = %#v (%T), want tea.BatchMsg of length 3", msg, msg)
	}

	// batch[0]/batch[1] are the loadSymbolCmds fetches (underlying,
	// expirations); batch[2] is the refresh tick — asserted non-nil only,
	// never executed (it would block for refreshEvery).
	got := tuitest.Drive(m, batch[0](), batch[1]()).(model)
	if got.underlying == nil || got.underlying.Symbol != "AAPL" {
		t.Errorf("underlying = %#v, want AAPL quote", got.underlying)
	}
	if len(got.expirations) != 1 {
		t.Fatalf("expirations = %v, want len 1", got.expirations)
	}
	if batch[2] == nil {
		t.Error("batch[2] (tick cmd) = nil, want non-nil")
	}
}

// --- expirationsMsg ---

func TestExpirationsMsg_SelectsNearestFutureExpiration(t *testing.T) {
	m := newTestModel(t, newMux("/v1/options/chain/AAPL/", jsonHandler(map[string]any{"s": "ok", "optionSymbol": []string{}})))

	past := testNow.AddDate(0, 0, -5)
	nearFuture := testNow.AddDate(0, 0, 3)
	farFuture := testNow.AddDate(0, 0, 30)
	msg := expirationsMsg{symbol: "AAPL", expirations: []time.Time{past, farFuture, nearFuture}, meta: &marketdata.Response{}}

	got := tuitest.Drive(m, msg).(model)
	if got.expSelected != 2 {
		t.Errorf("expSelected = %d, want 2 (nearFuture)", got.expSelected)
	}
}

func TestExpirationsMsg_AllPastFallsBackToLast(t *testing.T) {
	m := newTestModel(t, newMux("/v1/options/chain/AAPL/", jsonHandler(map[string]any{"s": "ok", "optionSymbol": []string{}})))

	p1 := testNow.AddDate(0, 0, -10)
	p2 := testNow.AddDate(0, 0, -3)
	msg := expirationsMsg{symbol: "AAPL", expirations: []time.Time{p1, p2}, meta: &marketdata.Response{}}

	got := tuitest.Drive(m, msg).(model)
	if got.expSelected != 1 {
		t.Errorf("expSelected = %d, want 1 (last)", got.expSelected)
	}
}

func TestExpirationsMsg_TriggersFirstChainLoadWithoutStrikeFilter(t *testing.T) {
	var seen bool
	var gotStrike, gotSide string
	mux := newMux("/v1/options/chain/AAPL/", func(w http.ResponseWriter, r *http.Request) {
		seen = true
		gotStrike = r.URL.Query().Get("strike")
		gotSide = r.URL.Query().Get("side")
		jsonHandler(chainPayload())(w, r)
	})
	m := newTestModel(t, mux)

	exp := testNow.AddDate(0, 0, 5)
	_, cmd := m.Update(expirationsMsg{symbol: "AAPL", expirations: []time.Time{exp}, meta: &marketdata.Response{}})
	if cmd == nil {
		t.Fatal("cmd = nil, want non-nil (chain fetch)")
	}
	if _, ok := cmd().(chainMsg); !ok {
		t.Fatal("cmd() did not produce a chainMsg")
	}
	if !seen {
		t.Fatal("chain endpoint was not called")
	}
	if gotStrike != "" {
		t.Errorf("strike = %q, want empty (first load, underlyingPx unknown)", gotStrike)
	}
	if gotSide != "" {
		t.Errorf("side = %q, want empty (SideBoth default)", gotSide)
	}
}

func TestExpirationsMsg_EmptyExpirationsSkipsChainLoad(t *testing.T) {
	m := newTestModel(t, newMux("/v1/options/chain/AAPL/", jsonHandler(map[string]any{"s": "ok", "optionSymbol": []string{}})))

	_, cmd := m.Update(expirationsMsg{symbol: "AAPL", expirations: nil, meta: &marketdata.Response{}})
	if cmd != nil {
		t.Error("cmd != nil, want nil (no expirations to load a chain for)")
	}
}

// --- chainMsg ---

func TestChainMsg_StoresUnderlyingPriceAndSortsRows(t *testing.T) {
	m := newTestModel(t, newMux("/v1/options/chain/AAPL/", jsonHandler(chainPayload())))
	m.expirations = []time.Time{testNow.AddDate(0, 0, 5)}
	m.expSelected = 0

	cmds := m.chainCmds()
	if len(cmds) != 1 {
		t.Fatalf("len(chainCmds()) = %d, want 1 (no pins)", len(cmds))
	}
	cm, ok := cmds[0]().(chainMsg)
	if !ok {
		t.Fatal("chainCmds()[0]() did not produce a chainMsg")
	}

	got := tuitest.Drive(m, cm).(model)
	if got.underlyingPx != 155 {
		t.Errorf("underlyingPx = %v, want 155", got.underlyingPx)
	}
	if !got.lastRefresh.Equal(testNow) {
		t.Errorf("lastRefresh = %v, want %v", got.lastRefresh, testNow)
	}
	if got.statusNote != "" {
		t.Errorf("statusNote = %q, want empty", got.statusNote)
	}
	if got.lastErr != nil {
		t.Errorf("lastErr = %v, want nil", got.lastErr)
	}

	wantOrder := []struct {
		strike float64
		typ    options.OptionType
	}{
		{140, options.Put},
		{150, options.Call},
		{150, options.Put},
		{160, options.Call},
	}
	if len(got.rows) != len(wantOrder) {
		t.Fatalf("len(rows) = %d, want %d", len(got.rows), len(wantOrder))
	}
	for i, w := range wantOrder {
		if got.rows[i].Strike != w.strike || got.rows[i].Type != w.typ {
			t.Errorf("rows[%d] = {%v %v}, want {%v %v}", i, got.rows[i].Strike, got.rows[i].Type, w.strike, w.typ)
		}
	}
}

func TestChainMsg_UnderlyingPriceFallsBackToPreviousWhenZero(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.underlyingPx = 155

	payload := chainPayload()
	payload["underlyingPrice"] = []float64{0, 0, 0, 0}
	cm := chainMsg{chain: mustChain(t, payload), meta: &marketdata.Response{}}

	got := tuitest.Drive(m, cm).(model)
	if got.underlyingPx != 155 {
		t.Errorf("underlyingPx = %v, want 155 (kept previous)", got.underlyingPx)
	}
}

func TestChainMsg_NilChainWithNoDataSetsStatusNote(t *testing.T) {
	m := newTestModel(t, catchAllMux())

	cm := chainMsg{chain: nil, meta: &marketdata.Response{NoData: true}}
	got := tuitest.Drive(m, cm).(model)
	if got.statusNote != "no data for expiration" {
		t.Errorf("statusNote = %q, want %q", got.statusNote, "no data for expiration")
	}
}

func TestChainMsg_EmptyChainSetsDistinctStatusNote(t *testing.T) {
	m := newTestModel(t, catchAllMux())

	cm := chainMsg{chain: &options.OptionsChain{Underlying: "AAPL", Options: nil}, meta: &marketdata.Response{}}
	got := tuitest.Drive(m, cm).(model)
	if got.statusNote != "no contracts in window" {
		t.Errorf("statusNote = %q, want %q", got.statusNote, "no contracts in window")
	}
}

func TestChainCmds_SubsequentLoadPassesWindow(t *testing.T) {
	var gotStrike string
	mux := newMux("/v1/options/chain/AAPL/", func(w http.ResponseWriter, r *http.Request) {
		gotStrike = r.URL.Query().Get("strike")
		jsonHandler(map[string]any{"s": "ok", "optionSymbol": []string{}})(w, r)
	})
	m := newTestModel(t, mux)
	m.expirations = []time.Time{testNow.AddDate(0, 0, 5)}
	m.expSelected = 0
	m.underlyingPx = 155
	m.window = 0.10

	cmds := m.chainCmds()
	_ = cmds[0]()

	if want := "139.5-170.5"; gotStrike != want {
		t.Errorf("strike = %q, want %q", gotStrike, want)
	}
}

func TestChainCmds_IncludesFetchPinnedWhenNonEmpty(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.expirations = []time.Time{testNow.AddDate(0, 0, 5)}
	m.expSelected = 0
	m.pinned = []string{"AAPL250117C00150000"}

	cmds := m.chainCmds()
	if len(cmds) != 2 {
		t.Fatalf("len(chainCmds()) = %d, want 2 (chain + pinned)", len(cmds))
	}
}

func TestChainCmds_NoExpirationsReturnsNil(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	if cmds := m.chainCmds(); cmds != nil {
		t.Errorf("chainCmds() = %v, want nil", cmds)
	}
}

// --- expiration selection (arrow keys) ---

func TestExpirationArrowKeys_MoveSelectionAndRefetchChain(t *testing.T) {
	var chainCalls int
	mux := newMux("/v1/options/chain/AAPL/", func(w http.ResponseWriter, r *http.Request) {
		chainCalls++
		jsonHandler(map[string]any{"s": "ok", "optionSymbol": []string{}})(w, r)
	})
	m := newTestModel(t, mux)
	m.focus = focusExpirations
	m.expirations = []time.Time{testNow.AddDate(0, 0, 3), testNow.AddDate(0, 0, 10)}
	m.expSelected = 0

	newModelIface, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	got := newModelIface.(model)
	if got.expSelected != 1 {
		t.Fatalf("expSelected = %d, want 1", got.expSelected)
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want non-nil (chain refetch)")
	}
	_ = cmd()
	if chainCalls != 1 {
		t.Errorf("chainCalls = %d, want 1", chainCalls)
	}

	// Moving up from index 1 goes back to 0 and refetches again.
	newModelIface, cmd = got.Update(tea.KeyMsg{Type: tea.KeyUp})
	got = newModelIface.(model)
	if got.expSelected != 0 {
		t.Fatalf("expSelected = %d, want 0", got.expSelected)
	}
	_ = cmd()
	if chainCalls != 2 {
		t.Errorf("chainCalls = %d, want 2", chainCalls)
	}
}

func TestExpirationArrowKeys_ClampAtBoundsWithoutRefetch(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.focus = focusExpirations
	m.expirations = []time.Time{testNow.AddDate(0, 0, 3)}
	m.expSelected = 0

	newModelIface, cmd := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	got := newModelIface.(model)
	if got.expSelected != 0 {
		t.Errorf("expSelected = %d, want 0 (clamped)", got.expSelected)
	}
	if cmd != nil {
		t.Error("cmd != nil, want nil (selection did not change)")
	}
}

func TestChainRowArrowKeys_MoveSelectionWithoutFetch(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.rows = []options.OptionQuote{{OptionSymbol: "A"}, {OptionSymbol: "B"}, {OptionSymbol: "C"}}
	m.rowSelected = 0

	newModelIface, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	got := newModelIface.(model)
	if got.rowSelected != 1 {
		t.Errorf("rowSelected = %d, want 1", got.rowSelected)
	}
	if cmd != nil {
		t.Error("cmd != nil, want nil (row selection never fetches)")
	}
}

// --- tab / left / right pane switching ---

func TestTabTogglesFocusBetweenExpirationsAndChain(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	if m.focus != focusChain {
		t.Fatalf("precondition: focus = %v, want focusChain", m.focus)
	}

	got := tuitest.Drive(m, tea.KeyMsg{Type: tea.KeyTab}).(model)
	if got.focus != focusExpirations {
		t.Errorf("focus after tab = %v, want focusExpirations", got.focus)
	}

	got = tuitest.Drive(got, tea.KeyMsg{Type: tea.KeyTab}).(model)
	if got.focus != focusChain {
		t.Errorf("focus after second tab = %v, want focusChain", got.focus)
	}
}

func TestLeftRightAlsoSwitchPanes(t *testing.T) {
	m := newTestModel(t, catchAllMux())

	got := tuitest.Drive(m, tea.KeyMsg{Type: tea.KeyLeft}).(model)
	if got.focus != focusExpirations {
		t.Errorf("focus after left = %v, want focusExpirations", got.focus)
	}
	got = tuitest.Drive(got, tea.KeyMsg{Type: tea.KeyRight}).(model)
	if got.focus != focusChain {
		t.Errorf("focus after right = %v, want focusChain", got.focus)
	}
}

// --- c/p/b side, g greeks, +/- window ---

func TestSideKeys_SetSideAndRefetch(t *testing.T) {
	var gotSide string
	mux := newMux("/v1/options/chain/AAPL/", func(w http.ResponseWriter, r *http.Request) {
		gotSide = r.URL.Query().Get("side")
		jsonHandler(map[string]any{"s": "ok", "optionSymbol": []string{}})(w, r)
	})
	m := newTestModel(t, mux)
	m.expirations = []time.Time{testNow.AddDate(0, 0, 5)}

	cases := []struct {
		key  string
		want options.OptionSide
	}{
		{"c", options.SideCall},
		{"p", options.SidePut},
		{"b", options.SideBoth},
	}
	for _, tc := range cases {
		newModelIface, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)})
		got := newModelIface.(model)
		if got.side != tc.want {
			t.Errorf("key %q: side = %v, want %v", tc.key, got.side, tc.want)
		}
		if cmd == nil {
			t.Fatalf("key %q: cmd = nil, want non-nil (refetch)", tc.key)
		}
		_ = cmd()
		if want := string(tc.want); gotSide != want {
			t.Errorf("key %q: request side = %q, want %q", tc.key, gotSide, want)
		}
	}
}

func TestGreeksKey_TogglesWithoutFetch(t *testing.T) {
	m := newTestModel(t, catchAllMux())

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	gotModel := got.(model)
	if !gotModel.showGreeks {
		t.Error("showGreeks = false, want true")
	}
	if cmd != nil {
		t.Error("cmd != nil, want nil ('g' never fetches)")
	}

	got, _ = gotModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if got.(model).showGreeks {
		t.Error("showGreeks = true after second toggle, want false")
	}
}

func TestWindowKeys_ClampAndRefetchOnlyWhenUnderlyingKnown(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.window = 0.10
	m.underlyingPx = 0 // first load: no underlying price yet

	got, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})
	gotModel := got.(model)
	if !approxEqual(gotModel.window, 0.15) {
		t.Errorf("window = %v, want 0.15", gotModel.window)
	}
	if cmd != nil {
		t.Error("cmd != nil, want nil (underlyingPx == 0, no refetch)")
	}

	gotModel.underlyingPx = 155
	gotModel.expirations = []time.Time{testNow.AddDate(0, 0, 5)}
	got, cmd = gotModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("-")})
	gotModel = got.(model)
	if !approxEqual(gotModel.window, 0.10) {
		t.Errorf("window = %v, want 0.10", gotModel.window)
	}
	if cmd == nil {
		t.Error("cmd = nil, want non-nil (underlyingPx > 0, refetch)")
	}
}

func TestWindowKeys_ClampAtBounds(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.window = 0.48

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})
	got, _ = got.(model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})
	gotModel := got.(model)
	if gotModel.window != 0.50 {
		t.Errorf("window = %v, want 0.50 (clamped at max)", gotModel.window)
	}

	m2 := newTestModel(t, catchAllMux())
	m2.window = 0.04
	got2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("-")})
	got2, _ = got2.(model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("-")})
	if got2.(model).window != 0.02 {
		t.Errorf("window = %v, want 0.02 (clamped at min)", got2.(model).window)
	}
}

// --- pure helpers: sortedRows, atmIndex ---

func TestSortedRows_StrikeAscendingCallsBeforePuts(t *testing.T) {
	in := []options.OptionQuote{
		{Strike: 150, Type: options.Put},
		{Strike: 140, Type: options.Call},
		{Strike: 150, Type: options.Call},
	}
	got := sortedRows(in)
	want := []struct {
		strike float64
		typ    options.OptionType
	}{
		{140, options.Call},
		{150, options.Call},
		{150, options.Put},
	}
	for i, w := range want {
		if got[i].Strike != w.strike || got[i].Type != w.typ {
			t.Errorf("rows[%d] = {%v %v}, want {%v %v}", i, got[i].Strike, got[i].Type, w.strike, w.typ)
		}
	}
}

func TestATMIndex_NearestStrike(t *testing.T) {
	rows := []options.OptionQuote{{Strike: 140}, {Strike: 150}, {Strike: 160}}
	if got := atmIndex(rows, 152); got != 1 {
		t.Errorf("atmIndex = %d, want 1 (150 nearest to 152)", got)
	}
	if got := atmIndex(nil, 100); got != -1 {
		t.Errorf("atmIndex(nil, ...) = %d, want -1", got)
	}
}

// --- enter / contract detail modal ---

func TestEnterOnChainRow_FetchesContractAndOpensModal(t *testing.T) {
	mux := newMux("/v1/options/quotes/AAPL250117C00150000/", jsonHandler(map[string]any{
		"s": "ok", "optionSymbol": []string{"AAPL250117C00150000"}, "underlying": []string{"AAPL"},
		"strike": []float64{150}, "side": []string{"call"}, "bid": []float64{5.50}, "ask": []float64{5.60},
	}))
	m := newTestModel(t, mux)
	m.rows = []options.OptionQuote{{OptionSymbol: "AAPL250117C00150000", Strike: 150}}
	m.rowSelected = 0

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("cmd = nil, want non-nil (fetchContract)")
	}
	msg := cmd()
	cm, ok := msg.(contractMsg)
	if !ok {
		t.Fatalf("msg type = %T, want contractMsg", msg)
	}

	got := tuitest.Drive(m, cm).(model)
	if got.detail == nil || got.detail.OptionSymbol != "AAPL250117C00150000" {
		t.Errorf("detail = %#v, want the fetched contract", got.detail)
	}
}

func TestEnterWithNoRows_NoOp(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("cmd != nil, want nil (no rows to activate)")
	}
}

func TestEscClosesDetailModal(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.detail = &options.OptionQuote{OptionSymbol: "AAPL250117C00150000"}

	got := tuitest.Drive(m, tea.KeyMsg{Type: tea.KeyEsc}).(model)
	if got.detail != nil {
		t.Errorf("detail = %#v, want nil after esc", got.detail)
	}
}

// --- space: pin/unpin ---

func TestSpace_PinsAndUnpinsSelectedRow(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.rows = []options.OptionQuote{{OptionSymbol: "AAPL250117C00150000"}}
	m.rowSelected = 0
	m.pinData["AAPL250117C00150000"] = options.OptionQuote{OptionSymbol: "AAPL250117C00150000", Bid: 1}

	got := tuitest.Drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}).(model)
	if len(got.pinned) != 1 || got.pinned[0] != "AAPL250117C00150000" {
		t.Fatalf("pinned = %v, want [AAPL250117C00150000]", got.pinned)
	}

	got = tuitest.Drive(got, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}).(model)
	if len(got.pinned) != 0 {
		t.Errorf("pinned = %v, want empty after unpin", got.pinned)
	}
	if _, ok := got.pinData["AAPL250117C00150000"]; ok {
		t.Error("pinData still has the unpinned symbol, want removed")
	}
}

func TestSpace_PreservesOrderAcrossMultiplePins(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.rows = []options.OptionQuote{{OptionSymbol: "A"}, {OptionSymbol: "B"}, {OptionSymbol: "C"}}

	m.rowSelected = 0
	got := tuitest.Drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}).(model)
	got.rowSelected = 2
	got = tuitest.Drive(got, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}).(model)
	got.rowSelected = 1
	got = tuitest.Drive(got, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}).(model)

	want := []string{"A", "C", "B"}
	if len(got.pinned) != len(want) {
		t.Fatalf("pinned = %v, want %v", got.pinned, want)
	}
	for i := range want {
		if got.pinned[i] != want[i] {
			t.Errorf("pinned[%d] = %q, want %q", i, got.pinned[i], want[i])
		}
	}
}

// --- refresh tick: chain + pinned, suspension ---

func TestRefreshTick_NotSuspendedIncludesChainAndReschedules(t *testing.T) {
	var chainCalls, pinnedCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/options/chain/AAPL/", func(w http.ResponseWriter, r *http.Request) {
		chainCalls++
		jsonHandler(map[string]any{"s": "ok", "optionSymbol": []string{}})(w, r)
	})
	mux.HandleFunc("/v1/options/quotes/AAPL250117C00150000/", func(w http.ResponseWriter, r *http.Request) {
		pinnedCalls++
		jsonHandler(map[string]any{"s": "ok", "optionSymbol": []string{"AAPL250117C00150000"}, "underlying": []string{"AAPL"}})(w, r)
	})
	mux.HandleFunc("/", jsonHandler(map[string]any{"s": "ok"}))
	m := newTestModel(t, mux)
	m.expirations = []time.Time{testNow.AddDate(0, 0, 5)}
	m.pinned = []string{"AAPL250117C00150000"}

	want := m.chainCmds()
	if len(want) != 2 {
		t.Fatalf("precondition: len(chainCmds()) = %d, want 2", len(want))
	}

	_, cmd := m.Update(refreshTickMsg(testNow))
	if cmd == nil {
		t.Fatal("cmd = nil, want non-nil")
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok || len(batch) != 3 {
		t.Fatalf("msg = %#v (%T), want tea.BatchMsg of length 3 (chain, pinned, tick)", msg, msg)
	}
	// Execute the chain and pinned fetches (batch[0], batch[1]); leave the
	// trailing tick (batch[2]) unexecuted per the "never executed" rule.
	_ = batch[0]()
	_ = batch[1]()
	if chainCalls != 1 {
		t.Errorf("chainCalls = %d, want 1", chainCalls)
	}
	if pinnedCalls != 1 {
		t.Errorf("pinnedCalls = %d, want 1", pinnedCalls)
	}
	if batch[2] == nil {
		t.Error("batch[2] (tick) = nil, want non-nil")
	}
}

func TestRefreshTick_SuspendedReschedulesOnly(t *testing.T) {
	var chainCalls int
	mux := newMux("/v1/options/chain/AAPL/", func(w http.ResponseWriter, r *http.Request) {
		chainCalls++
		jsonHandler(map[string]any{"s": "ok", "optionSymbol": []string{}})(w, r)
	})
	m := newTestModel(t, mux)
	m.expirations = []time.Time{testNow.AddDate(0, 0, 5)}
	m.suspendedUntil = testNow.Add(time.Hour)
	// Shrink the refresh interval so executing the returned tick command
	// below is safe (executing a tick blocks for refreshEvery). This is
	// the one deliberate exception to the "tick cmds are never executed"
	// rule: executing it here is exactly what proves the suspended path
	// returned the tick alone.
	m.refreshEvery = time.Millisecond

	_, cmd := m.Update(refreshTickMsg(testNow))
	if cmd == nil {
		t.Fatal("cmd = nil, want non-nil (reschedule)")
	}
	// tea.Batch collapses a single remaining command to that command
	// directly rather than wrapping it in tea.BatchMsg (see commands.go's
	// Batch). So if suspension correctly skipped chainCmds, cmd IS the
	// tick and executing it yields a refreshTickMsg; had chainCmds been
	// batched in, executing cmd would yield a tea.BatchMsg instead (as in
	// TestRefreshTick_NotSuspendedIncludesChainAndReschedules above).
	msg := cmd()
	if _, ok := msg.(refreshTickMsg); !ok {
		t.Fatalf("cmd() = %#v (%T), want refreshTickMsg (a single non-Batch tick)", msg, msg)
	}
	if chainCalls != 0 {
		t.Errorf("chainCalls = %d, want 0 (suspended tick must not fetch the chain)", chainCalls)
	}
}

func TestErrMsg_RateLimitErrorSuspendsRefresh(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	resetAt := testNow.Add(90 * time.Second)
	rle := &marketdata.RateLimitError{ResetAt: resetAt}

	got := tuitest.Drive(m, errMsg{op: "chain", err: rle}).(model)
	if !got.suspendedUntil.Equal(resetAt) {
		t.Errorf("suspendedUntil = %v, want %v", got.suspendedUntil, resetAt)
	}
	if got.lastErr == nil {
		t.Error("lastErr = nil, want the rate-limit error")
	}
	if got.lastErrOp != "chain" {
		t.Errorf("lastErrOp = %q, want chain", got.lastErrOp)
	}

	var asRLE *marketdata.RateLimitError
	if !errors.As(got.lastErr, &asRLE) {
		t.Errorf("lastErr = %v, want errors.As to find *marketdata.RateLimitError", got.lastErr)
	}
}

// --- s: symbol input ---

func TestSymbolInput_EnterResetsStateAndReloads(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/stocks/bulkquotes/", jsonHandler(map[string]any{
		"s": "ok", "symbol": []string{"MSFT"}, "bid": []float64{1}, "ask": []float64{2},
		"mid": []float64{1.5}, "last": []float64{1.5}, "volume": []int64{1}, "updated": []int64{1704067200},
	}))
	mux.HandleFunc("/v1/options/expirations/MSFT/", jsonHandler(map[string]any{
		"s": "ok", "expirations": []int64{testNow.AddDate(0, 0, 7).Unix()},
	}))
	mux.HandleFunc("/", jsonHandler(map[string]any{"s": "ok"}))
	m := newTestModel(t, mux)
	m.symbol = "AAPL"
	m.underlyingPx = 155
	m.chain = &options.OptionsChain{Underlying: "AAPL"}
	m.rows = []options.OptionQuote{{OptionSymbol: "AAPL250117C00150000"}}
	m.detail = &options.OptionQuote{OptionSymbol: "AAPL250117C00150000"}
	m.statusNote = "stale note"

	m.focus = focusChain
	got := tuitest.Drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}).(model)
	if got.focus != focusSymbol {
		t.Fatalf("focus = %v, want focusSymbol", got.focus)
	}
	if got.symbolInput.Value() != "AAPL" {
		t.Fatalf("symbolInput prefilled with %q, want AAPL", got.symbolInput.Value())
	}

	// Type "msft" over the pre-filled value.
	got.symbolInput.SetValue("msft")
	newModelIface, cmd := got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = newModelIface.(model)
	if got.focus != focusChain {
		t.Errorf("focus after enter = %v, want focusChain", got.focus)
	}
	if got.symbol != "MSFT" {
		t.Errorf("symbol = %q, want MSFT (uppercased)", got.symbol)
	}
	if got.underlyingPx != 0 {
		t.Errorf("underlyingPx = %v, want reset to 0", got.underlyingPx)
	}
	if got.chain != nil || got.rows != nil {
		t.Errorf("chain/rows not reset: chain=%v rows=%v", got.chain, got.rows)
	}
	if got.detail != nil {
		t.Error("detail not reset, want nil")
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want non-nil (reload)")
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok || len(batch) != 2 {
		t.Fatalf("msg = %#v (%T), want tea.BatchMsg of length 2", msg, msg)
	}
	loaded := tuitest.Drive(got, batch[0](), batch[1]()).(model)
	if loaded.underlying == nil || loaded.underlying.Symbol != "MSFT" {
		t.Errorf("underlying = %#v, want MSFT quote", loaded.underlying)
	}
	if len(loaded.expirations) != 1 {
		t.Errorf("expirations = %v, want len 1", loaded.expirations)
	}
}

func TestSymbolInput_EscCancelsWithoutChangingSymbol(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.focus = focusSymbol
	m.symbolInput.SetValue("MSFT")

	got := tuitest.Drive(m, tea.KeyMsg{Type: tea.KeyEsc}).(model)
	if got.focus != focusChain {
		t.Errorf("focus = %v, want focusChain", got.focus)
	}
	if got.symbol != "AAPL" {
		t.Errorf("symbol = %q, want unchanged AAPL", got.symbol)
	}
}

func TestSymbolInput_OtherKeysAreInertToAppHotkeysButTypeIntoInput(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.focus = focusSymbol
	m.symbolInput.SetValue("")
	m.symbolInput.Focus()

	// Deliberately discard cmd: it may be the textinput's cursor-blink
	// command, which blocks for a real cursor.BlinkSpeed interval when
	// executed. The behavior under test — that 'q' types into the input
	// rather than acting as the quit hotkey — is fully observable from
	// the resulting model state.
	newModelIface, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	gotModel := newModelIface.(model)
	if gotModel.focus != focusSymbol {
		t.Errorf("focus = %v, want focusSymbol ('q' must not act as the quit hotkey while typing)", gotModel.focus)
	}
	if gotModel.symbolInput.Value() != "q" {
		t.Errorf("symbolInput.Value() = %q, want %q (typed, not treated as a hotkey)", gotModel.symbolInput.Value(), "q")
	}
}

func TestCtrlC_QuitsEvenWithInputFocus(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.focus = focusSymbol

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("cmd = nil, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("cmd() did not produce tea.QuitMsg")
	}
}

// --- /: lookup input ---

func TestSlashKey_OpensLookupInputWithFocus(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.focus = focusChain
	m.lookupInput.SetValue("leftover from a previous query")

	got := tuitest.Drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}).(model)
	if got.focus != focusLookup {
		t.Fatalf("focus = %v, want focusLookup", got.focus)
	}
	if !got.lookupInput.Focused() {
		t.Error("lookupInput.Focused() = false, want true")
	}
	if got.lookupInput.Value() != "" {
		t.Errorf("lookupInput.Value() = %q, want empty (cleared on open)", got.lookupInput.Value())
	}
}

func TestLookupInput_QKeyIsInertAndTypesIntoInput(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.focus = focusLookup
	m.lookupInput.SetValue("")
	m.lookupInput.Focus()

	// Deliberately discard cmd: it may be the textinput's cursor-blink
	// command, which blocks when executed (see the symbol-input inertness
	// test). The behavior under test — that 'q' types into the lookup
	// input rather than quitting — is fully observable from model state.
	newModelIface, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	got := newModelIface.(model)
	if got.focus != focusLookup {
		t.Errorf("focus = %v, want focusLookup ('q' must not act as the quit hotkey while typing)", got.focus)
	}
	if got.lookupInput.Value() != "q" {
		t.Errorf("lookupInput.Value() = %q, want %q (typed, not treated as a hotkey)", got.lookupInput.Value(), "q")
	}
}

func TestLookupInput_ParseFailureSetsUsageHintAndStaysOpen(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.focus = focusLookup
	m.lookupInput.SetValue("garbage")

	newModelIface, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := newModelIface.(model)
	if got.statusNote != "usage: SYMBOL YYYY-MM-DD STRIKE call|put" {
		t.Errorf("statusNote = %q, want usage hint", got.statusNote)
	}
	if got.focus != focusLookup {
		t.Errorf("focus = %v, want focusLookup (stays open for correction)", got.focus)
	}
	if cmd != nil {
		t.Error("cmd != nil, want nil (no fetch on parse failure)")
	}
}

func TestLookupInput_ParseSuccessIssuesLookupContract(t *testing.T) {
	mux := newMux("/v1/options/lookup/", jsonHandler(map[string]any{"s": "ok", "optionSymbol": "AAPL250117C00150000"}))
	m := newTestModel(t, mux)
	m.focus = focusLookup
	m.lookupInput.SetValue("AAPL 2025-01-17 150 call")

	newModelIface, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := newModelIface.(model)
	if got.focus != focusChain {
		t.Errorf("focus = %v, want focusChain (closed on success)", got.focus)
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want non-nil (lookupContract)")
	}
	msg := cmd()
	lm, ok := msg.(lookupMsg)
	if !ok {
		t.Fatalf("msg type = %T, want lookupMsg", msg)
	}
	if lm.occ != "AAPL250117C00150000" {
		t.Errorf("occ = %q, want AAPL250117C00150000", lm.occ)
	}
}

func TestLookupMsg_NoDataSetsStatusNote(t *testing.T) {
	m := newTestModel(t, catchAllMux())

	got := tuitest.Drive(m, lookupMsg{noData: true, meta: &marketdata.Response{NoData: true}}).(model)
	if got.statusNote != "no such contract" {
		t.Errorf("statusNote = %q, want %q", got.statusNote, "no such contract")
	}
}

func TestLookupMsg_SuccessFetchesContract(t *testing.T) {
	mux := newMux("/v1/options/quotes/AAPL250117C00150000/", jsonHandler(map[string]any{
		"s": "ok", "optionSymbol": []string{"AAPL250117C00150000"}, "underlying": []string{"AAPL"},
	}))
	m := newTestModel(t, mux)

	_, cmd := m.Update(lookupMsg{occ: "AAPL250117C00150000", meta: &marketdata.Response{}})
	if cmd == nil {
		t.Fatal("cmd = nil, want non-nil (fetchContract)")
	}
	msg := cmd()
	cm, ok := msg.(contractMsg)
	if !ok {
		t.Fatalf("msg type = %T, want contractMsg", msg)
	}
	got := tuitest.Drive(m, cm).(model)
	if got.detail == nil || got.detail.OptionSymbol != "AAPL250117C00150000" {
		t.Errorf("detail = %#v, want the fetched contract", got.detail)
	}
}

func TestLookupInput_EscCancels(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.focus = focusLookup
	m.lookupInput.SetValue("AAPL 2025-01-17 150 call")

	got := tuitest.Drive(m, tea.KeyMsg{Type: tea.KeyEsc}).(model)
	if got.focus != focusChain {
		t.Errorf("focus = %v, want focusChain", got.focus)
	}
}

// --- E: support modal, r: force refresh, q/ctrl+c: quit ---

func TestSupportKey_OpensModalAndEscCloses(t *testing.T) {
	m := newTestModel(t, catchAllMux())

	got := tuitest.Drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")}).(model)
	if !got.showSupport {
		t.Fatal("showSupport = false, want true")
	}

	got = tuitest.Drive(got, tea.KeyMsg{Type: tea.KeyEsc}).(model)
	if got.showSupport {
		t.Error("showSupport = true after esc, want false")
	}
}

func TestForceRefreshKey_TriggersChainAndClearsSuspension(t *testing.T) {
	m := newTestModel(t, newMux("/v1/options/chain/AAPL/", jsonHandler(map[string]any{"s": "ok", "optionSymbol": []string{}})))
	m.expirations = []time.Time{testNow.AddDate(0, 0, 5)}
	m.suspendedUntil = testNow.Add(time.Hour)

	newModelIface, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	got := newModelIface.(model)
	if !got.suspendedUntil.IsZero() {
		t.Errorf("suspendedUntil = %v, want zero", got.suspendedUntil)
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want non-nil (forced chain refresh)")
	}
	if _, ok := cmd().(chainMsg); !ok {
		t.Error("cmd() did not produce a chainMsg")
	}
}

func TestQKeyQuits(t *testing.T) {
	m := newTestModel(t, catchAllMux())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("cmd = nil, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("cmd() did not produce tea.QuitMsg")
	}
}

func TestUnhandledKey_IsANoOp(t *testing.T) {
	m := newTestModel(t, catchAllMux())

	newModelIface, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")})
	if cmd != nil {
		t.Errorf("cmd = %v, want nil", cmd)
	}
	if newModelIface.(model).symbol != m.symbol {
		t.Error("Update(z) mutated the model unexpectedly")
	}
}

// --- errMsg (generic) ---

func TestErrMsg_SetsLastErrAndOp(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	wantErr := errors.New("boom")

	got := tuitest.Drive(m, errMsg{op: "underlying", err: wantErr}).(model)
	if got.lastErr != wantErr {
		t.Errorf("lastErr = %v, want %v", got.lastErr, wantErr)
	}
	if got.lastErrOp != "underlying" {
		t.Errorf("lastErrOp = %q, want underlying", got.lastErrOp)
	}
}

func TestSuccessfulDataMsg_ClearsLastErr(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.lastErr = errors.New("boom")
	m.lastErrOp = "chain"

	got := tuitest.Drive(m, chainMsg{chain: mustChain(t, chainPayload()), meta: &marketdata.Response{}}).(model)
	if got.lastErr != nil {
		t.Errorf("lastErr = %v, want nil (cleared by successful chainMsg)", got.lastErr)
	}
	if got.lastErrOp != "" {
		t.Errorf("lastErrOp = %q, want empty (cleared by successful chainMsg)", got.lastErrOp)
	}
}

// --- credits from every non-nil meta ---

func TestCredits_UpdatedFromEveryNonNilMeta(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	meta := &marketdata.Response{RateLimit: marketdata.RateLimitMeta{Limit: 100, Remaining: 42}}

	got := tuitest.Drive(m, underlyingMsg{symbol: "AAPL", meta: meta}).(model)
	if got.credits != meta.RateLimit {
		t.Errorf("credits = %+v, want %+v", got.credits, meta.RateLimit)
	}
}

// --- stale-response guards (F1): Bubble Tea runs commands concurrently, so
// switching symbols mid-flight can let a response for the old symbol land
// after the new symbol's own fetches have already started. Each handler
// must drop a message whose symbol (or, for chainMsg, echoed Underlying)
// doesn't match the model's current symbol, and still apply one that
// matches. ---

func TestUnderlyingMsg_StaleSymbolDropped(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.symbol = "MSFT"
	stale := &stocks.Quote{Symbol: "AAPL", Last: 1}

	got := tuitest.Drive(m, underlyingMsg{symbol: "AAPL", quote: stale, meta: &marketdata.Response{}}).(model)
	if got.underlying != nil {
		t.Errorf("underlying = %+v, want nil (stale AAPL response dropped for MSFT model)", got.underlying)
	}
}

func TestUnderlyingMsg_MatchingSymbolApplies(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.symbol = "MSFT"
	quote := &stocks.Quote{Symbol: "MSFT", Last: 1}

	got := tuitest.Drive(m, underlyingMsg{symbol: "MSFT", quote: quote, meta: &marketdata.Response{}}).(model)
	if got.underlying != quote {
		t.Errorf("underlying = %+v, want %+v (matching symbol must still apply)", got.underlying, quote)
	}
}

func TestExpirationsMsg_StaleSymbolDropped(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.symbol = "MSFT"
	m.expirations = nil

	got, cmd := m.Update(expirationsMsg{symbol: "AAPL", expirations: []time.Time{testNow.AddDate(0, 0, 5)}, meta: &marketdata.Response{}})
	gm := got.(model)
	if gm.expirations != nil {
		t.Errorf("expirations = %v, want nil (stale AAPL response dropped for MSFT model)", gm.expirations)
	}
	if cmd != nil {
		t.Error("cmd != nil, want nil (a dropped stale message issues no chain load)")
	}
}

func TestExpirationsMsg_MatchingSymbolApplies(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.symbol = "MSFT"
	exp := testNow.AddDate(0, 0, 5)

	got, _ := m.Update(expirationsMsg{symbol: "MSFT", expirations: []time.Time{exp}, meta: &marketdata.Response{}})
	gm := got.(model)
	if len(gm.expirations) != 1 || !gm.expirations[0].Equal(exp) {
		t.Errorf("expirations = %v, want [%v] (matching symbol must still apply)", gm.expirations, exp)
	}
}

func TestChainMsg_StaleUnderlyingDropped(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.symbol = "MSFT"

	stale := chainMsg{chain: mustChain(t, chainPayload()), meta: &marketdata.Response{}} // mustChain builds for "AAPL"
	got := tuitest.Drive(m, stale).(model)
	if got.chain != nil {
		t.Errorf("chain = %+v, want nil (stale AAPL response dropped for MSFT model)", got.chain)
	}
	if len(got.rows) != 0 {
		t.Errorf("rows = %v, want empty (stale response dropped)", got.rows)
	}
}

func TestChainMsg_MatchingUnderlyingApplies(t *testing.T) {
	m := newTestModel(t, catchAllMux()) // symbol AAPL

	matching := chainMsg{chain: mustChain(t, chainPayload()), meta: &marketdata.Response{}}
	got := tuitest.Drive(m, matching).(model)
	if got.chain == nil {
		t.Error("chain = nil, want non-nil (matching underlying must still apply)")
	}
}

func TestChainMsg_EmptyUnderlyingAccepted(t *testing.T) {
	m := newTestModel(t, catchAllMux())
	m.symbol = "MSFT"

	// No Underlying to compare against (e.g. hand-built in a test, or an
	// unusual API response): accept rather than guess it's stale.
	cm := chainMsg{chain: &options.OptionsChain{Underlying: "", Options: nil}, meta: &marketdata.Response{}}
	got := tuitest.Drive(m, cm).(model)
	if got.chain == nil {
		t.Error("chain = nil, want non-nil (empty Underlying can't be stale-checked, so it's accepted)")
	}
}

// --- mustChain: builds an *options.OptionsChain via fetchChain against a
// mock server, for tests that need a chain payload without going through
// chainCmds. ---

func mustChain(t *testing.T, payload map[string]any) *options.OptionsChain {
	t.Helper()
	mux := newMux("/v1/options/chain/AAPL/", jsonHandler(payload))
	client := newTestClient(t, mux)
	msg := mustCmdMsg(t, fetchChain(client, "AAPL", testNow.AddDate(0, 0, 5), 0, 0, options.SideBoth))
	cm, ok := msg.(chainMsg)
	if !ok {
		t.Fatalf("mustChain: msg type = %T, want chainMsg", msg)
	}
	return cm.chain
}
