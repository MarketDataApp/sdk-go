package main

import (
	"errors"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/funds"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/markets"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/utilities"
)

// marketTickInterval is the fixed cadence for marketTickMsg, independent of
// -refresh (which only controls the watchlist cadence via refreshTickMsg).
const marketTickInterval = 60 * time.Second

// candleRange selects which candle window the detail pane requests for
// the selected symbol. The zero value, rangeDaily, is the default shown
// on startup; the '1' and 'w' keys (task 2.4) switch between the three.
type candleRange int

const (
	// rangeDaily requests daily candles over the trailing 3 months. It is
	// the zero value and default.
	rangeDaily candleRange = iota // 'd' default: daily, 3 months

	// rangeIntraday requests 5-minute candles, the most recent 78 (one
	// trading day at 5-minute resolution).
	rangeIntraday // '1': 5-min, countback 78

	// rangeWeekly requests weekly candles, the most recent 52 (one year).
	rangeWeekly // 'w': weekly, countback 52
)

// Message catalog. Every fetch function in fetch.go resolves to exactly
// one of these on success and to errMsg on failure. Each success message
// carries the decoded data plus the per-request *marketdata.Response
// (meta), which holds rate-limit state and the NoData flag for the
// 404-as-success case. See fetch.go's file comment for the full pattern.

// quotesMsg carries the result of fetchQuotes (client.Stocks.Quotes),
// used for the watchlist refresh when -prices is not set.
type quotesMsg struct {
	quotes []stocks.Quote
	meta   *marketdata.Response
}

// pricesMsg carries the result of fetchPrices (client.Stocks.Prices),
// used for the watchlist refresh when -prices is set.
type pricesMsg struct {
	prices []stocks.Price
	meta   *marketdata.Response
}

// candlesMsg carries the result of fetchCandles (client.Stocks.Candles)
// for the detail pane's candle window, identified by rng.
type candlesMsg struct {
	symbol  string
	rng     candleRange
	candles []stocks.Candle
	meta    *marketdata.Response
}

// fundCandlesMsg carries the result of fetchFundCandles
// (client.Funds.Candles), used instead of candlesMsg when the selected
// symbol is in the model's funds set.
type fundCandlesMsg struct {
	symbol  string
	candles []funds.Candle
	meta    *marketdata.Response
}

// detailQuoteMsg carries the result of fetchDetailQuote
// (client.Stocks.Quote with WithFiftyTwoWeek), the 52-week high/low shown
// in the detail pane.
type detailQuoteMsg struct {
	symbol string
	quote  *stocks.Quote // 52-week
	meta   *marketdata.Response
}

// earningsMsg carries the result of fetchEarnings (client.Stocks.Earnings).
type earningsMsg struct {
	symbol   string
	earnings []stocks.Earning
	meta     *marketdata.Response
}

// newsMsg carries the result of fetchNews (client.Stocks.News).
type newsMsg struct {
	symbol   string
	articles []stocks.NewsArticle
	meta     *marketdata.Response
}

// marketStatusMsg carries the result of fetchMarketStatus
// (client.Markets.Status), refreshed on the 60s market tick.
type marketStatusMsg struct {
	status *markets.MarketStatus
	meta   *marketdata.Response
}

// statusHistoryMsg carries the result of fetchStatusHistory
// (client.Markets.StatusHistory), shown in the status-history modal.
type statusHistoryMsg struct {
	statuses []markets.MarketStatus
	meta     *marketdata.Response
}

// userMsg carries the result of fetchUser (client.Utilities.User).
type userMsg struct {
	user *utilities.UserInfo
	meta *marketdata.Response
}

// apiStatusMsg carries the result of fetchAPIStatus
// (client.Utilities.Status), shown in the diagnostics modal.
type apiStatusMsg struct {
	status *utilities.APIStatus
	meta   *marketdata.Response
}

// headersMsg carries the result of fetchHeaders (client.Utilities.Headers),
// shown in the diagnostics modal.
type headersMsg struct {
	headers *utilities.Headers
	meta    *marketdata.Response
}

// bulkCandlesMsg carries the result of fetchBulkCandles
// (client.Stocks.BulkCandles), shown in the day-performance modal.
type bulkCandlesMsg struct {
	candles []stocks.BulkCandle
	meta    *marketdata.Response
}

// addValidatedMsg carries the result of validateSymbol
// (client.Stocks.Quote), the add-symbol modal's validation step. noData
// is true both for an ordinary 404 and for the SDK's
// *stocks.QuoteNotFoundError — either way the API answered but has no
// quote for the symbol, so it is not added to the watchlist.
//
// Contract note: on the QuoteNotFoundError path (API answered 200 with an
// empty result set) meta is nil, because SDK errors carry no Response.
// Unlike every other message in this catalog, consumers of
// addValidatedMsg must therefore nil-guard meta before reading it. On the
// 404 path meta is non-nil with NoData set to true, as usual.
type addValidatedMsg struct {
	symbol string
	quote  *stocks.Quote
	noData bool
	meta   *marketdata.Response
}

// errMsg reports that a fetch function's SDK call returned an error. op
// identifies which fetch function failed (each fetch function's godoc-
// visible op string is listed in fetch.go's file-top comment); err is the
// SDK error, typically classified with errors.As against the
// *marketdata.RateLimitError / AuthenticationError / NetworkError family.
type errMsg struct {
	op  string
	err error
}

// refreshTickMsg fires on the watchlist refresh cadence (default 5s, set
// by -refresh).
type refreshTickMsg time.Time

// marketTickMsg fires on the market status refresh cadence (60s, fixed).
type marketTickMsg time.Time

// modal identifies which full-screen overlay, if any, is currently shown
// over the watchlist. Only one can be open at a time; opening a modal makes
// the base watchlist keys (arrows, a, x, 1, d, w) inert until esc closes it.
type modal int

const (
	// modalNone means no modal is open; the watchlist keys are active.
	modalNone modal = iota

	// modalStatusHistory shows the 5-day market status history (key 'm').
	modalStatusHistory

	// modalDiagnostics shows API status and outbound request headers
	// (key 'D').
	modalDiagnostics

	// modalError shows lastErr's SupportInfo() (key 'E'; rendered in
	// task 2.5).
	modalError

	// modalBulk shows today's day-performance candle for every watchlist
	// symbol (key 'o').
	modalBulk

	// modalAdd shows a text input for adding a new symbol (key 'a').
	modalAdd
)

// model is the Bubble Tea model backing the stockterm watchlist: the
// watchlist itself, the detail pane for the selected symbol, modal overlay
// state, and the bookkeeping (credits, last refresh, last error) shown in
// the status line. Fields mirror the design's model skeleton exactly.
type model struct {
	// client is the SDK client used for every fetch. Owned by main, which
	// closes it on exit.
	client *marketdata.Client

	// symbols is the watchlist, in display order.
	symbols []string

	// funds marks which symbols are treated as mutual funds (candles come
	// from client.Funds instead of client.Stocks).
	funds map[string]bool

	// quotes holds the latest full quote per symbol, keyed by symbol.
	// Populated by quotesMsg (and by a successful addValidatedMsg).
	quotes map[string]stocks.Quote

	// prices holds the latest lightweight price per symbol, keyed by
	// symbol. Populated by pricesMsg when usePrices is set.
	prices map[string]stocks.Price

	// usePrices selects the lightweight prices endpoint instead of full
	// quotes for the watchlist refresh, set by -prices.
	usePrices bool

	// selected is the index into symbols of the row shown in the detail
	// pane.
	selected int

	// --- detail pane state for symbols[selected] ---

	// rng selects which candle window the detail pane requests: daily (the
	// default), intraday, or weekly. Switched by the '1'/'d'/'w' keys.
	rng candleRange

	// candles holds the selected symbol's candles when it is not a fund.
	candles []stocks.Candle

	// fundCandles holds the selected symbol's candles when it is a fund
	// (m.funds[selected symbol] is true).
	fundCandles []funds.Candle

	// detail holds the selected symbol's 52-week quote.
	detail *stocks.Quote

	// earnings holds the selected symbol's upcoming earnings reports.
	earnings []stocks.Earning

	// news holds the selected symbol's most recent news articles.
	news []stocks.NewsArticle

	// detailNoData records, per detail-pane operation, whether the last
	// fetch for the selected symbol came back with no data. Keys are
	// exactly "candles", "earnings", "news", "quote".
	detailNoData map[string]bool

	// --- cross-cutting state ---

	market      *markets.MarketStatus
	history     []markets.MarketStatus
	user        *utilities.UserInfo
	apiStatus   *utilities.APIStatus
	headers     *utilities.Headers
	bulk        []stocks.BulkCandle
	credits     marketdata.RateLimitMeta // updated from every message with non-nil meta
	lastRefresh time.Time
	lastErr     error
	lastErrOp   string
	statusNote  string // transient status-line note ("no data for XYZ")

	// suspendedUntil is set from a *marketdata.RateLimitError's ResetAt
	// when a fetch is rate-limited. While m.now() is before this time, the
	// watchlist refresh tick skips its fetch (but keeps rescheduling); the
	// 'r' key clears it as an explicit user override. It is not part of
	// the design's published model skeleton but is required to implement
	// the rate-limit-suspension behavior the design calls for.
	suspendedUntil time.Time

	modal        modal
	input        textinput.Model
	demoMode     bool
	refreshEvery time.Duration
	width        int
	height       int
	now          func() time.Time // injectable clock for tests
}

// newModel builds the initial model from a client and the parsed CLI
// configuration.
func newModel(client *marketdata.Client, cfg appConfig, demoMode bool) model {
	return model{
		client:       client,
		symbols:      cfg.symbols,
		funds:        cfg.funds,
		quotes:       make(map[string]stocks.Quote),
		prices:       make(map[string]stocks.Price),
		usePrices:    cfg.usePrices,
		detailNoData: make(map[string]bool),
		input:        textinput.New(),
		refreshEvery: cfg.refresh,
		demoMode:     demoMode,
		now:          time.Now,
	}
}

// refreshCmds decides the watchlist refresh fetch: a single fetchQuotes (or
// fetchPrices under -prices) call covering every non-fund symbol. Fund
// symbols are excluded because the bulk quote/price endpoints don't cover
// them; their live data comes only from the detail pane's fund candles.
// Kept pure (no I/O, no clock reads) so tests can inspect the decision and
// execute the returned commands directly against a mock server.
func (m model) refreshCmds() []tea.Cmd {
	var syms []string
	for _, s := range m.symbols {
		if !m.funds[s] {
			syms = append(syms, s)
		}
	}
	if len(syms) == 0 {
		return nil
	}
	if m.usePrices {
		return []tea.Cmd{fetchPrices(m.client, syms)}
	}
	return []tea.Cmd{fetchQuotes(m.client, syms)}
}

// detailCmds decides the detail-pane cascade for symbol: candles (or fund
// candles, by m.funds), the 52-week quote, earnings, and news, in that
// order. Kept pure for the same reason as refreshCmds.
func (m model) detailCmds(symbol string) []tea.Cmd {
	var cmds []tea.Cmd
	if m.funds[symbol] {
		cmds = append(cmds, fetchFundCandles(m.client, symbol))
	} else {
		cmds = append(cmds, fetchCandles(m.client, symbol, m.rng))
	}
	cmds = append(cmds, fetchDetailQuote(m.client, symbol))
	cmds = append(cmds, fetchEarnings(m.client, symbol))
	cmds = append(cmds, fetchNews(m.client, symbol))
	return cmds
}

// scheduleRefreshTick returns the command that fires the next
// refreshTickMsg after m.refreshEvery. It is a real tea.Tick, so tests must
// never invoke it directly (it blocks until the duration elapses) — only
// assert it is non-nil.
func (m model) scheduleRefreshTick() tea.Cmd {
	return tea.Tick(m.refreshEvery, func(t time.Time) tea.Msg {
		return refreshTickMsg(t)
	})
}

// scheduleMarketTick returns the command that fires the next
// marketTickMsg after the fixed marketTickInterval. Same tea.Tick caveat
// as scheduleRefreshTick.
func (m model) scheduleMarketTick() tea.Cmd {
	return tea.Tick(marketTickInterval, func(t time.Time) tea.Msg {
		return marketTickMsg(t)
	})
}

// refreshTickCmds decides what a refreshTickMsg produces: the watchlist
// refetch (skipped while m.now() is before m.suspendedUntil — an active
// rate-limit suspension) plus the rescheduled tick, always last. Kept pure,
// like refreshCmds/detailCmds, specifically so tests can assert the skip
// decision without invoking the reschedule command.
func (m model) refreshTickCmds() []tea.Cmd {
	var cmds []tea.Cmd
	if !m.now().Before(m.suspendedUntil) {
		cmds = append(cmds, m.refreshCmds()...)
	}
	cmds = append(cmds, m.scheduleRefreshTick())
	return cmds
}

// marketTickCmds decides what a marketTickMsg produces: fetchMarketStatus
// plus the rescheduled tick, always last. Unlike refreshTickCmds this never
// skips — market status isn't subject to the watchlist's rate-limit
// suspension — but it's kept as its own pure method for the same
// testability reason.
func (m model) marketTickCmds() []tea.Cmd {
	return []tea.Cmd{fetchMarketStatus(m.client), m.scheduleMarketTick()}
}

// resetDetailState clears every detail-pane field for symbols[selected],
// used whenever the selection changes so stale data from the previously
// selected symbol never lingers on screen while the new fetches are in
// flight.
func (m model) resetDetailState() model {
	m.candles = nil
	m.fundCandles = nil
	m.detail = nil
	m.earnings = nil
	m.news = nil
	m.detailNoData = make(map[string]bool)
	return m
}

// containsString reports whether s appears in list.
func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// metaOf extracts the *marketdata.Response carried by msg, if msg is one of
// the fetch-result message types, so Update can apply the "every incoming
// msg with non-nil meta updates m.credits" rule generically. addValidatedMsg
// is included even though its meta may be nil (see its doc comment); nil
// metas are simply not applied by the caller.
func metaOf(msg tea.Msg) *marketdata.Response {
	switch t := msg.(type) {
	case quotesMsg:
		return t.meta
	case pricesMsg:
		return t.meta
	case candlesMsg:
		return t.meta
	case fundCandlesMsg:
		return t.meta
	case detailQuoteMsg:
		return t.meta
	case earningsMsg:
		return t.meta
	case newsMsg:
		return t.meta
	case marketStatusMsg:
		return t.meta
	case statusHistoryMsg:
		return t.meta
	case userMsg:
		return t.meta
	case apiStatusMsg:
		return t.meta
	case headersMsg:
		return t.meta
	case bulkCandlesMsg:
		return t.meta
	case addValidatedMsg:
		return t.meta
	default:
		return nil
	}
}

// isDataMsg reports whether msg is one of the fetch-result message types
// (as opposed to a tick, key press, or window resize). Every such message
// represents a completed fetch — including the SDK's no-data convention,
// which is success, not failure — so Update uses it to clear a stale
// lastErr/lastErrOp on the next successful fetch of any kind.
func isDataMsg(msg tea.Msg) bool {
	switch msg.(type) {
	case quotesMsg, pricesMsg, candlesMsg, fundCandlesMsg, detailQuoteMsg,
		earningsMsg, newsMsg, marketStatusMsg, statusHistoryMsg, userMsg,
		apiStatusMsg, headersMsg, bulkCandlesMsg, addValidatedMsg:
		return true
	default:
		return false
	}
}

// Init starts the model's background commands: the watchlist refresh, the
// selected symbol's detail cascade, market status, the user's credit state
// (skipped in demo mode, since /user/ requires a token), and the two
// recurring ticks (watchlist refresh, market status).
func (m model) Init() tea.Cmd {
	cmds := append(m.refreshCmds(), m.detailCmds(m.symbols[0])...)
	cmds = append(cmds, fetchMarketStatus(m.client))
	if !m.demoMode {
		cmds = append(cmds, fetchUser(m.client))
	}
	cmds = append(cmds, m.scheduleRefreshTick(), m.scheduleMarketTick())
	return tea.Batch(cmds...)
}

// Update handles incoming messages: fetch results (populating watchlist and
// detail-pane state), the two ticks, key presses (selection, range
// switching, add/remove, modals, quit), and window resizes.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if meta := metaOf(msg); meta != nil {
		m.credits = meta.RateLimit
	}
	if isDataMsg(msg) {
		m.lastErr = nil
		m.lastErrOp = ""
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case quotesMsg:
		if msg.meta != nil {
			m.lastRefresh = m.now()
		}
		if msg.meta != nil && msg.meta.NoData {
			m.statusNote = "no data"
		} else {
			for _, q := range msg.quotes {
				m.quotes[q.Symbol] = q
			}
			m.statusNote = ""
		}
		return m, nil

	case pricesMsg:
		if msg.meta != nil {
			m.lastRefresh = m.now()
		}
		if msg.meta != nil && msg.meta.NoData {
			m.statusNote = "no data"
		} else {
			for _, p := range msg.prices {
				m.prices[p.Symbol] = p
			}
			m.statusNote = ""
		}
		return m, nil

	case candlesMsg:
		if len(m.symbols) > 0 && msg.symbol == m.symbols[m.selected] && msg.rng == m.rng {
			if msg.meta != nil && msg.meta.NoData {
				m.detailNoData["candles"] = true
			} else {
				m.candles = msg.candles
				delete(m.detailNoData, "candles")
			}
		}
		return m, nil

	case fundCandlesMsg:
		if len(m.symbols) > 0 && msg.symbol == m.symbols[m.selected] {
			if msg.meta != nil && msg.meta.NoData {
				m.detailNoData["candles"] = true
			} else {
				m.fundCandles = msg.candles
				delete(m.detailNoData, "candles")
			}
		}
		return m, nil

	case detailQuoteMsg:
		if len(m.symbols) > 0 && msg.symbol == m.symbols[m.selected] {
			if msg.meta != nil && msg.meta.NoData {
				m.detailNoData["quote"] = true
			} else {
				m.detail = msg.quote
				delete(m.detailNoData, "quote")
			}
		}
		return m, nil

	case earningsMsg:
		if len(m.symbols) > 0 && msg.symbol == m.symbols[m.selected] {
			if msg.meta != nil && msg.meta.NoData {
				m.detailNoData["earnings"] = true
			} else {
				m.earnings = msg.earnings
				delete(m.detailNoData, "earnings")
			}
		}
		return m, nil

	case newsMsg:
		if len(m.symbols) > 0 && msg.symbol == m.symbols[m.selected] {
			if msg.meta != nil && msg.meta.NoData {
				m.detailNoData["news"] = true
			} else {
				m.news = msg.articles
				delete(m.detailNoData, "news")
			}
		}
		return m, nil

	case marketStatusMsg:
		if msg.meta == nil || !msg.meta.NoData {
			m.market = msg.status
		}
		return m, nil

	case statusHistoryMsg:
		if msg.meta == nil || !msg.meta.NoData {
			m.history = msg.statuses
		}
		return m, nil

	case userMsg:
		if msg.meta == nil || !msg.meta.NoData {
			m.user = msg.user
		}
		return m, nil

	case apiStatusMsg:
		if msg.meta == nil || !msg.meta.NoData {
			m.apiStatus = msg.status
		}
		return m, nil

	case headersMsg:
		if msg.meta == nil || !msg.meta.NoData {
			m.headers = msg.headers
		}
		return m, nil

	case bulkCandlesMsg:
		if msg.meta == nil || !msg.meta.NoData {
			m.bulk = msg.candles
		}
		return m, nil

	case addValidatedMsg:
		if msg.noData {
			m.statusNote = "no data for " + msg.symbol
		} else if msg.quote != nil {
			if !containsString(m.symbols, msg.symbol) {
				m.symbols = append(m.symbols, msg.symbol)
			}
			m.quotes[msg.symbol] = *msg.quote
		}
		return m, nil

	case errMsg:
		m.lastErr = msg.err
		m.lastErrOp = msg.op
		var rle *marketdata.RateLimitError
		if errors.As(msg.err, &rle) {
			m.suspendedUntil = rle.ResetAt
		}
		return m, nil

	case refreshTickMsg:
		return m, tea.Batch(m.refreshTickCmds()...)

	case marketTickMsg:
		return m, tea.Batch(m.marketTickCmds()...)
	}
	return m, nil
}

// handleKey implements every key binding: global quit, the add-symbol text
// input while modalAdd is open, esc closing any other open modal, and
// (only while no modal is open) selection, range switching, add/remove, and
// modal-opening keys.
//
// Quit-key scoping: ctrl+c quits unconditionally, everywhere. 'q' quits
// everywhere except while the add-symbol input is open — there it is
// forwarded to the textinput like any other rune, so tickers containing a
// 'q' (QQQ, most notably) remain typeable.
func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()

	if s == "ctrl+c" {
		return m, tea.Quit
	}

	if m.modal == modalAdd {
		switch s {
		case "esc":
			m.modal = modalNone
			m.input.Blur()
			m.input.SetValue("")
			return m, nil
		case "enter":
			symbol := strings.ToUpper(strings.TrimSpace(m.input.Value()))
			m.modal = modalNone
			m.input.Blur()
			m.input.SetValue("")
			if symbol == "" {
				return m, nil
			}
			return m, validateSymbol(m.client, symbol)
		default:
			// Deliberately includes "q": while typing a symbol it is
			// input, not a quit request.
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
	}

	if s == "q" {
		return m, tea.Quit
	}

	if m.modal != modalNone {
		if s == "esc" {
			m.modal = modalNone
		}
		return m, nil
	}

	switch s {
	case "up":
		if m.selected > 0 {
			m.selected--
			m = m.resetDetailState()
			return m, tea.Batch(m.detailCmds(m.symbols[m.selected])...)
		}

	case "down":
		if m.selected < len(m.symbols)-1 {
			m.selected++
			m = m.resetDetailState()
			return m, tea.Batch(m.detailCmds(m.symbols[m.selected])...)
		}

	case "1", "d", "w":
		sym := m.symbols[m.selected]
		if m.funds[sym] {
			// Fund candles always come from one fixed call
			// (fetchFundCandles takes no range argument: daily,
			// countback 63), so switching m.rng here would relabel that
			// same data with a range caption that isn't what was
			// actually fetched. Honest behavior is a no-op: the fund
			// caption stays the daily NAV label.
			return m, nil
		}
		switch s {
		case "1":
			m.rng = rangeIntraday
		case "d":
			m.rng = rangeDaily
		case "w":
			m.rng = rangeWeekly
		}
		return m, fetchCandles(m.client, sym, m.rng)

	case "a":
		m.modal = modalAdd
		m.input = textinput.New()
		m.input.Placeholder = "SYMBOL"
		m.input.CharLimit = 10
		cmd := m.input.Focus()
		return m, cmd

	case "x":
		if len(m.symbols) <= 1 {
			m.statusNote = "cannot remove last symbol"
			return m, nil
		}
		sym := m.symbols[m.selected]
		newSymbols := make([]string, 0, len(m.symbols)-1)
		newSymbols = append(newSymbols, m.symbols[:m.selected]...)
		newSymbols = append(newSymbols, m.symbols[m.selected+1:]...)
		m.symbols = newSymbols
		delete(m.funds, sym)
		delete(m.quotes, sym)
		delete(m.prices, sym)
		if m.selected >= len(m.symbols) {
			m.selected = len(m.symbols) - 1
		}
		m = m.resetDetailState()
		return m, tea.Batch(m.detailCmds(m.symbols[m.selected])...)

	case "r":
		m.suspendedUntil = time.Time{}
		return m, tea.Batch(m.refreshCmds()...)

	case "m":
		m.modal = modalStatusHistory
		return m, fetchStatusHistory(m.client)

	case "D":
		m.modal = modalDiagnostics
		return m, tea.Batch(fetchAPIStatus(m.client), fetchHeaders(m.client))

	case "E":
		m.modal = modalError
		return m, nil

	case "o":
		m.modal = modalBulk
		return m, fetchBulkCandles(m.client, m.symbols)
	}

	return m, nil
}
