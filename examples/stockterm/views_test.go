package main

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MarketDataApp/sdk-go/v2/examples/tuitest"
	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/funds"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/markets"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/utilities"
)

// goldenNow is the frozen clock every golden test injects via m.now, so
// the header clock, the footer's "refreshed" time, and any other
// m.now()-derived text are byte-identical on every run. It is
// constructed directly in US/Eastern (rather than converted into it) so
// the header reads exactly "Sat 2026-07-11 14:32:05 ET" as the design
// mock shows, independent of loadEasternLocation's own correctness.
var goldenNow = time.Date(2026, 7, 11, 14, 32, 5, 0, easternLocation)

// ep builds an *float64, for the pointer-typed EstimatedEPS/ReportedEPS
// fields on stocks.Earning.
func ep(v float64) *float64 { return &v }

// easternMidnight returns t's Eastern calendar day at 00:00:00 Eastern.
// t.Truncate(24*time.Hour) would round to a *UTC* midnight instead, which
// — for a location west of UTC, as US/Eastern always is — lands on the
// wrong calendar day when displayed back in Eastern; this fixture helper
// avoids that trap for the fixed-date test data below.
func easternMidnight(t time.Time) time.Time {
	y, mo, d := t.In(easternLocation).Date()
	return time.Date(y, mo, d, 0, 0, 0, 0, easternLocation)
}

// genCandles fabricates n deterministic daily/weekly-shaped candles
// riding a slow sine wave around base, so the sparkline has visible
// structure instead of a flat line.
func genCandles(n int, base, amp float64, start time.Time, step time.Duration) []stocks.Candle {
	out := make([]stocks.Candle, n)
	for i := 0; i < n; i++ {
		v := base + amp*math.Sin(float64(i)/3)
		out[i] = stocks.Candle{
			Time:   start.Add(time.Duration(i) * step),
			Open:   v - 0.4,
			High:   v + 1.1,
			Low:    v - 1.1,
			Close:  v,
			Volume: 40_000_000 + int64(i)*10_000,
		}
	}
	return out
}

// genFundCandles is genCandles for funds.Candle (no Volume field).
func genFundCandles(n int, base, amp float64, start time.Time, step time.Duration) []funds.Candle {
	out := make([]funds.Candle, n)
	for i := 0; i < n; i++ {
		v := base + amp*math.Sin(float64(i)/3)
		out[i] = funds.Candle{
			Time:  start.Add(time.Duration(i) * step),
			Open:  v - 0.2,
			High:  v + 0.5,
			Low:   v - 0.5,
			Close: v,
		}
	}
	return out
}

// baseModel builds the fully populated fixture every golden test starts
// from: a 5-symbol watchlist (AAPL, MSFT, META, SPY, and the fund
// VFINX), AAPL selected with a complete detail pane (daily candles,
// 52-week quote, next earnings, three headlines), an open market, and a
// primed credit meter — deliberately echoing the values in the design
// doc's ASCII mock (AAPL 233.10 +1.24, 52wk 168.2-245.1, next earnings
// 2026-07-30 est EPS $2.14, credits 91,201/100,000, resets 09:30 ET).
func baseModel(t *testing.T) model {
	t.Helper()

	m := model{
		symbols: []string{"AAPL", "MSFT", "META", "SPY", "VFINX"},
		funds:   map[string]bool{"VFINX": true},
		quotes: map[string]stocks.Quote{
			"AAPL": {Symbol: "AAPL", Last: 233.10, Bid: 233.08, Ask: 233.12, Change: 1.24, ChangePercent: 0.0053, Volume: 41_203_110},
			"MSFT": {Symbol: "MSFT", Last: 512.44, Bid: 512.40, Ask: 512.48, Change: -2.01, ChangePercent: -0.0039, Volume: 18_113_020},
			"META": {Symbol: "META", Last: 512.00, Bid: 511.95, Ask: 512.05, Change: 3.10, ChangePercent: 0.0061, Volume: 9_876_543},
			"SPY":  {Symbol: "SPY", Last: 560.25, Bid: 560.20, Ask: 560.30, Change: -0.85, ChangePercent: -0.0015, Volume: 55_432_111},
			// VFINX intentionally absent: it's a fund, excluded from the
			// bulk quote refresh, so its row exercises the "no data yet"
			// placeholder path.
		},
		prices:       make(map[string]stocks.Price),
		selected:     0,
		rng:          rangeDaily,
		candles:      genCandles(63, 226, 8, time.Date(2026, 4, 13, 0, 0, 0, 0, easternLocation), 24*time.Hour),
		detailNoData: make(map[string]bool),
		detail: &stocks.Quote{
			Symbol: "AAPL", Last: 233.10, FiftyTwoWeekLow: 168.20, FiftyTwoWeekHigh: 245.10,
		},
		earnings: []stocks.Earning{
			{Symbol: "AAPL", ReportDate: time.Date(2026, 7, 30, 0, 0, 0, 0, easternLocation), EstimatedEPS: ep(2.14)},
		},
		news: []stocks.NewsArticle{
			{Symbol: "AAPL", Headline: "Apple unveils new product lineup at fall event", PublicationDate: goldenNow.Add(-2 * time.Hour)},
			{Symbol: "AAPL", Headline: "Suppliers report strong Q3 shipment volumes", PublicationDate: goldenNow.Add(-5 * time.Hour)},
			{Symbol: "AAPL", Headline: "Analysts raise price targets after earnings beat", PublicationDate: goldenNow.Add(-26 * time.Hour)},
		},
		market: &markets.MarketStatus{Date: goldenNow, Open: true, Status: "open"},
		credits: marketdata.RateLimitMeta{
			Limit: 100_000, Remaining: 8_799, Consumed: 91_201,
			ResetAt: time.Date(2026, 7, 11, 9, 30, 0, 0, easternLocation),
		},
		lastRefresh:  goldenNow.Add(-2 * time.Second),
		refreshEvery: 5 * time.Second,
		input:        textinput.New(),
		now:          func() time.Time { return goldenNow },
	}
	return sized(m)
}

// sized injects the golden window size every test renders at (100x40 —
// the size the task brief and contracts.md both fix golden frames to).
func sized(m model) model {
	return tuitest.Drive(m, tea.WindowSizeMsg{Width: 100, Height: 40}).(model)
}

// TestBoxLine_OverflowingContentKeepsClosingBorder locks in boxLine's
// defensive clip: content wider than the box swallows the closing
// border if boxLine doesn't truncate it itself (see boxLine's doc
// comment) — this is a plain unit test, not a golden, because it's a
// specific invariant ("the border survives"), not a full-frame layout.
func TestBoxLine_OverflowingContentKeepsClosingBorder(t *testing.T) {
	width := 40
	longContent := strings.Repeat("x", 500)

	got := boxLine(width, longContent)
	gotRunes := []rune(got)

	if len(gotRunes) != width {
		t.Fatalf("len(boxLine(...)) = %d, want %d", len(gotRunes), width)
	}
	if last := gotRunes[len(gotRunes)-1]; last != '│' {
		t.Errorf("last rune = %q, want '│' (closing border must survive overflow)", last)
	}
	if first := gotRunes[0]; first != '│' {
		t.Errorf("first rune = %q, want '│'", first)
	}
}

func TestGolden_Main(t *testing.T) {
	m := baseModel(t)
	tuitest.Golden(t, "testdata/main.golden", tuitest.Frame(m))
}

func TestGolden_IntradayDetail(t *testing.T) {
	m := baseModel(t)
	m.rng = rangeIntraday
	m.candles = genCandles(78, 232, 1.5, goldenNow.Add(-6*time.Hour+30*time.Minute), 5*time.Minute)
	tuitest.Golden(t, "testdata/intraday.golden", tuitest.Frame(m))
}

func TestGolden_WeeklyDetail(t *testing.T) {
	m := baseModel(t)
	m.rng = rangeWeekly
	m.candles = genCandles(52, 210, 20, goldenNow.AddDate(-1, 0, 0), 7*24*time.Hour)
	tuitest.Golden(t, "testdata/weekly.golden", tuitest.Frame(m))
}

func TestGolden_ModalStatusHistory(t *testing.T) {
	m := baseModel(t)
	m.modal = modalStatusHistory
	day := easternMidnight(goldenNow)
	m.history = []markets.MarketStatus{
		{Date: day.AddDate(0, 0, -4), Open: true, Status: "open"},
		{Date: day.AddDate(0, 0, -3), Open: true, Status: "open"},
		{Date: day.AddDate(0, 0, -2), Open: true, Status: "early-close"},
		{Date: day.AddDate(0, 0, -1), Open: false, Status: "closed"},
		{Date: day, Open: true, Status: "open"},
	}
	tuitest.Golden(t, "testdata/modal_status_history.golden", tuitest.Frame(m))
}

func TestGolden_ModalDiagnostics(t *testing.T) {
	m := baseModel(t)
	m.modal = modalDiagnostics
	m.apiStatus = &utilities.APIStatus{Status: "online", Uptime30d: 99.95, Uptime90d: 99.90, Updated: goldenNow}
	m.headers = &utilities.Headers{Headers: map[string]string{
		"Authorization": "Bearer ***redacted***",
		"User-Agent":    "marketdata-go-sdk/2.0.0",
		"X-Request-Id":  "req-8f3c1e",
	}}
	tuitest.Golden(t, "testdata/modal_diagnostics.golden", tuitest.Frame(m))
}

func TestGolden_ModalError(t *testing.T) {
	m := baseModel(t)
	m.modal = modalError
	m.lastErrOp = "quotes"
	m.lastErr = &marketdata.InternalError{
		SupportContext: marketdata.SupportContext{
			RequestID:     "8f3c1e9a-ray",
			RequestURL:    "https://api.marketdata.app/v1/stocks/bulkquotes/",
			StatusCode:    500,
			Timestamp:     goldenNow,
			Message:       "unexpected server error",
			ExceptionType: "InternalServerError",
		},
	}
	tuitest.Golden(t, "testdata/modal_error.golden", tuitest.Frame(m))
}

func TestGolden_ModalBulk(t *testing.T) {
	m := baseModel(t)
	m.modal = modalBulk
	day := easternMidnight(goldenNow)
	m.bulk = []stocks.BulkCandle{
		{Symbol: "AAPL", Time: day, Open: 231.86, High: 233.50, Low: 231.20, Close: 233.10, Volume: 41_203_110},
		{Symbol: "MSFT", Time: day, Open: 514.45, High: 515.00, Low: 511.80, Close: 512.44, Volume: 18_113_020},
		{Symbol: "META", Time: day, Open: 508.90, High: 513.20, Low: 508.50, Close: 512.00, Volume: 9_876_543},
		{Symbol: "SPY", Time: day, Open: 561.10, High: 561.50, Low: 559.80, Close: 560.25, Volume: 55_432_111},
		{Symbol: "VFINX", Time: day, Open: 399.00, High: 400.50, Low: 398.50, Close: 400.10, Volume: 0},
	}
	tuitest.Golden(t, "testdata/modal_bulk.golden", tuitest.Frame(m))
}

func TestGolden_ModalAdd(t *testing.T) {
	m := baseModel(t)
	m.modal = modalAdd
	m.input = textinput.New()
	m.input.Placeholder = "SYMBOL"
	m.input.Focus()
	m.input.SetValue("TS")
	tuitest.Golden(t, "testdata/modal_add.golden", tuitest.Frame(m))
}

func TestGolden_DemoBanner(t *testing.T) {
	m := baseModel(t)
	m.demoMode = true
	m.symbols = []string{"AAPL"}
	m.quotes = map[string]stocks.Quote{
		"AAPL": {Symbol: "AAPL", Last: 233.10, Bid: 233.08, Ask: 233.12, Change: 1.24, ChangePercent: 0.0053, Volume: 41_203_110},
	}
	m.selected = 0
	m.credits = marketdata.RateLimitMeta{Limit: 100, Remaining: 88, Consumed: 12, ResetAt: goldenNow.Add(time.Hour)}
	tuitest.Golden(t, "testdata/demo_banner.golden", tuitest.Frame(m))
}

func TestGolden_NoDataDetail(t *testing.T) {
	m := baseModel(t)
	m.candles = nil
	m.detail = nil
	m.earnings = nil
	m.news = nil
	m.detailNoData = map[string]bool{"candles": true, "quote": true, "earnings": true, "news": true}
	tuitest.Golden(t, "testdata/no_data_detail.golden", tuitest.Frame(m))
}

// TestGolden_FundDetail is not on the task's required golden list, but it
// is the only frame that exercises the "Fund symbols: detail pane renders
// fund candles sparkline with NAV labels" binding decision end to end —
// VFINX selected, fund candles populated, and (realistically, since
// client.Stocks.Quote/Earnings/News don't cover mutual funds) the 52-week,
// earnings, and news sections all on their no-data fallback.
func TestGolden_FundDetail(t *testing.T) {
	m := baseModel(t)
	m.selected = 4 // VFINX
	m.fundCandles = genFundCandles(63, 398, 4, time.Date(2026, 4, 13, 0, 0, 0, 0, easternLocation), 24*time.Hour)
	m.candles = nil
	m.detail = nil
	m.earnings = nil
	m.news = nil
	m.detailNoData = map[string]bool{"quote": true, "earnings": true, "news": true}
	tuitest.Golden(t, "testdata/fund_detail.golden", tuitest.Frame(m))
}

// TestFundSparklineCaption_IgnoresRng closes the selection-change residual
// of the fund-caption honesty fix: 1/d/w are no-ops while a fund is
// selected, but m.rng can still be non-daily from a previously selected
// stock (resetDetailState deliberately doesn't touch rng — it's a user
// preference for stock symbols). Fund candles are always daily countback-63
// NAV data regardless of m.rng, so the fund caption must always be the
// daily label, never rangeLabelText(m.rng).
func TestFundSparklineCaption_IgnoresRng(t *testing.T) {
	m := baseModel(t)
	m.rng = rangeWeekly // set by 'w' while a stock was selected
	m.selected = 4      // VFINX, the fund
	m.fundCandles = genFundCandles(63, 398, 4, time.Date(2026, 4, 13, 0, 0, 0, 0, easternLocation), 24*time.Hour)

	got := m.renderSparklineLine(true)
	if !strings.Contains(got, rangeLabelText(rangeDaily)) {
		t.Errorf("fund sparkline caption = %q, want it to contain %q (fund data is always daily)", got, rangeLabelText(rangeDaily))
	}
	if strings.Contains(got, rangeLabelText(rangeWeekly)) {
		t.Errorf("fund sparkline caption = %q, must not contain %q (rng is a stock-only preference)", got, rangeLabelText(rangeWeekly))
	}
}

func TestGolden_RateLimitedStatusLine(t *testing.T) {
	m := baseModel(t)
	m.lastErrOp = "quotes"
	m.lastErr = &marketdata.RateLimitError{
		SupportContext: marketdata.SupportContext{Message: "rate limit exceeded"},
		Limit:          100_000,
		Remaining:      0,
		// classify() renders this from m.now() (goldenNow, frozen above),
		// not the real wall clock, so ResetAt can be set relative to
		// goldenNow and the golden's "4m12s" is exact on every run — no
		// fixture-to-render latency window to pad.
		ResetAt: goldenNow.Add(4*time.Minute + 12*time.Second),
	}
	tuitest.Golden(t, "testdata/rate_limited_status.golden", tuitest.Frame(m))
}
