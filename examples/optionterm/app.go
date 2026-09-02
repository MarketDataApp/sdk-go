package main

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

// Message catalog. Every SDK call the program makes (see fetch.go) resolves
// to exactly one of these tea.Msg types: a typed success message carrying
// the SDK result plus response metadata, or errMsg on failure. Update (below)
// holds the cases that consume them.

// underlyingMsg carries the result of fetchUnderlying: the current quote
// for the active underlying symbol. symbol is the symbol the fetch was
// issued for — Update compares it against m.symbol to drop a stale
// response that lands after the user has already switched symbols (Bubble
// Tea runs commands concurrently, so an in-flight fetch for the old symbol
// can resolve after a newer one has already started).
type underlyingMsg struct {
	symbol string
	quote  *stocks.Quote
	meta   *marketdata.Response
}

// expirationsMsg carries the result of fetchExpirations: the expiration
// dates with listed contracts for symbol. Update compares symbol against
// m.symbol to drop a stale response the same way underlyingMsg does.
type expirationsMsg struct {
	symbol      string
	expirations []time.Time
	meta        *marketdata.Response
}

// chainMsg carries the result of fetchChain: the option chain for one
// expiration, already filtered server-side by strike window and side.
// Unlike underlyingMsg/expirationsMsg there is no separate symbol field:
// the API echoes the requested underlying on chain.Underlying, so Update
// uses that directly to drop a stale response (when chain is non-nil and
// Underlying is set — a nil chain or an empty Underlying can't be stale-
// checked, so those are always accepted).
type chainMsg struct {
	chain *options.OptionsChain
	meta  *marketdata.Response
}

// contractMsg carries the result of fetchContract: a single option
// contract's quote, used for the detail modal.
type contractMsg struct {
	quote *options.OptionQuote
	meta  *marketdata.Response
}

// pinnedMsg carries the result of fetchPinned: quotes for every pinned OCC
// symbol, fetched concurrently through the client's shared pool.
type pinnedMsg struct {
	quotes []options.OptionQuote
	meta   *marketdata.Response
}

// lookupMsg carries the result of lookupContract: the resolved OCC option
// symbol, or noData true when the API could not resolve the contract.
type lookupMsg struct {
	occ    string
	noData bool
	meta   *marketdata.Response
}

// errMsg carries a failed SDK call. op names the operation ("underlying",
// "expirations", "chain", "contract", "pinned", or "lookup") so the status
// line and diagnostics modal can attribute the failure.
type errMsg struct {
	op  string
	err error
}

// refreshTickMsg fires on the chain refresh cadence (default 15s, -refresh).
type refreshTickMsg time.Time

// focus names which pane or input currently receives key input.
type focus int

const (
	// focusSymbol is active while the symbol-entry input ('s') is open.
	focusSymbol focus = iota
	// focusExpirations is active while the expirations sidebar has
	// keyboard focus: up/down move expSelected.
	focusExpirations
	// focusChain is active while the chain table has keyboard focus:
	// up/down move rowSelected. This is the default focus.
	focusChain
	// focusLookup is active while the lookup input ('/') is open.
	focusLookup
)

// model is the Bubble Tea model for optionterm: an underlying symbol, its
// expirations, the option chain for the selected expiration (filtered to a
// strike window and side), pinned contracts, and the transient UI state
// (focus, modals, status line) layered on top.
type model struct {
	// client is the Market Data client used for every SDK call the
	// program makes. Owned by main, which closes it on exit.
	client *marketdata.Client

	// symbol is the active underlying stock symbol.
	symbol      string
	symbolInput textinput.Model
	underlying  *stocks.Quote

	// expirations is the set of expiration dates for symbol, and
	// expSelected indexes the one currently shown in the chain pane.
	expirations []time.Time
	expSelected int

	// chain is the raw chain response for the selected expiration; rows
	// is its Options filtered+sorted for display (currently just sorted:
	// filtering already happens server-side via the strike window).
	chain        *options.OptionsChain
	rows         []options.OptionQuote
	rowSelected  int
	side         options.OptionSide // c/p/b
	window       float64            // strike window, 0.10 start, +/- adjust, min 0.02 max 0.50
	underlyingPx float64            // from previous chain's UnderlyingPrice
	showGreeks   bool               // g toggle

	// pinned is the OCC-symbol pin list in pin order; pinData holds the
	// last known quote for each, refreshed by fetchPinned alongside every
	// chain reload.
	pinned  []string
	pinData map[string]options.OptionQuote

	detail      *options.OptionQuote // Enter modal
	lookupInput textinput.Model
	showSupport bool // E modal

	credits     marketdata.RateLimitMeta
	lastRefresh time.Time
	lastErr     error
	lastErrOp   string
	statusNote  string // transient status-line note ("no contracts in window", ...)

	// suspendedUntil holds the reset time of the most recent rate-limit
	// error. While m.now() is before it, refreshTickMsg reschedules
	// without re-fetching. Not part of the binding skeleton; added per
	// the errMsg design decision (a *marketdata.RateLimitError sets it
	// from rle.ResetAt).
	suspendedUntil time.Time

	focus        focus
	demoMode     bool
	refreshEvery time.Duration
	width        int
	height       int
	now          func() time.Time // injectable clock for tests
}

// newModel constructs the initial model for the given client, underlying
// symbol, refresh cadence, and demo-mode flag. The symbol input is
// pre-filled with symbol (the caller has already forced it to AAPL for
// demo mode); focus starts on the chain pane, and now defaults to
// time.Now (tests inject a fixed clock by overwriting the field directly).
func newModel(client *marketdata.Client, symbol string, refresh time.Duration, demoMode bool) model {
	symbolInput := textinput.New()
	symbolInput.Placeholder = "symbol"
	symbolInput.SetValue(symbol)
	symbolInput.CharLimit = 16

	lookupInput := textinput.New()
	lookupInput.Placeholder = "SYMBOL YYYY-MM-DD STRIKE call|put"
	lookupInput.CharLimit = 64

	return model{
		client:       client,
		symbol:       symbol,
		symbolInput:  symbolInput,
		lookupInput:  lookupInput,
		side:         options.SideBoth,
		window:       0.10,
		pinData:      map[string]options.OptionQuote{},
		focus:        focusChain,
		demoMode:     demoMode,
		refreshEvery: refresh,
		now:          time.Now,
	}
}

// loadSymbolCmds returns the pair of commands that load everything known
// about m.symbol: the underlying quote and its expiration dates. Init uses
// it for the first load; the symbol-input Enter handler uses it again for
// every subsequent symbol change. Exposed as a method (rather than inlined)
// so tests can execute the two fetches individually without unbatching
// Init's combined tea.Cmd.
func (m model) loadSymbolCmds() []tea.Cmd {
	return []tea.Cmd{
		fetchUnderlying(m.client, m.symbol),
		fetchExpirations(m.client, m.symbol),
	}
}

// chainCmds returns the commands to (re)load the chain pane for the
// current symbol/expiration/window/side, plus a refresh of any pinned
// contracts. It is a pure decision method — reading model state, never
// mutating it or touching the network itself — so tests and Update's
// several call sites (expiration change, side/window change, refresh tick,
// forced refresh) share one implementation of "what should be fetched right
// now." It returns nil when there is no selected expiration yet (before the
// first expirationsMsg arrives).
//
// The strike window is omitted on the very first load: strikeWindow returns
// (0, 0) whenever underlyingPx is still zero, and fetchChain reads that as
// "no filter."
func (m model) chainCmds() []tea.Cmd {
	if len(m.expirations) == 0 || m.expSelected < 0 || m.expSelected >= len(m.expirations) {
		return nil
	}
	exp := m.expirations[m.expSelected]
	lo, hi := strikeWindow(m.underlyingPx, m.window)

	cmds := []tea.Cmd{fetchChain(m.client, m.symbol, exp, lo, hi, m.side)}
	if len(m.pinned) > 0 {
		cmds = append(cmds, fetchPinned(m.client, m.pinned))
	}
	return cmds
}

// scheduleRefreshTick returns a command that fires refreshTickMsg after
// refreshEvery. Tests assert it is non-nil without calling it — calling it
// blocks for refreshEvery, so it is never executed outside the real
// Bubble Tea event loop.
func (m model) scheduleRefreshTick() tea.Cmd {
	return tea.Tick(m.refreshEvery, func(t time.Time) tea.Msg {
		return refreshTickMsg(t)
	})
}

// Init implements tea.Model: it loads the initial symbol (the underlying
// quote and its expirations) and schedules the first refresh tick.
func (m model) Init() tea.Cmd {
	cmds := append(m.loadSymbolCmds(), m.scheduleRefreshTick())
	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case underlyingMsg:
		if msg.symbol != m.symbol {
			return m, nil
		}
		m.underlying = msg.quote
		m.applyMeta(msg.meta)
		m.clearErr()
		return m, nil

	case expirationsMsg:
		if msg.symbol != m.symbol {
			return m, nil
		}
		m.expirations = msg.expirations
		m.expSelected = selectNearestExpiration(m.expirations, m.now())
		if m.expSelected < 0 {
			m.expSelected = 0
		}
		m.applyMeta(msg.meta)
		m.clearErr()
		return m, tea.Batch(m.chainCmds()...)

	case chainMsg:
		if msg.chain != nil && msg.chain.Underlying != "" && msg.chain.Underlying != m.symbol {
			return m, nil
		}
		m.applyMeta(msg.meta)
		m.clearErr()
		m.lastRefresh = m.now()
		m.chain = msg.chain
		switch {
		case msg.meta != nil && msg.meta.NoData:
			// NoData, not a nil chain, is the signal: the SDK answers a
			// valid query that matched nothing with an empty chain, and
			// reserves the error path for a symbol it does not recognize.
			m.rows = nil
			m.rowSelected = 0
			m.statusNote = "no data for expiration"
		case msg.chain == nil:
			m.rows = nil
			m.rowSelected = 0
		case len(msg.chain.Options) == 0:
			m.rows = nil
			m.rowSelected = 0
			m.statusNote = "no contracts in window"
		default:
			if px := msg.chain.Options[0].UnderlyingPrice; px > 0 {
				m.underlyingPx = px
			}
			m.rows = sortedRows(msg.chain.Options)
			m.rowSelected = clampIndex(m.rowSelected, len(m.rows))
			m.statusNote = ""
		}
		return m, nil

	case contractMsg:
		m.detail = msg.quote
		m.applyMeta(msg.meta)
		m.clearErr()
		return m, nil

	case pinnedMsg:
		for _, q := range msg.quotes {
			m.pinData[q.OptionSymbol] = q
		}
		m.applyMeta(msg.meta)
		m.clearErr()
		return m, nil

	case lookupMsg:
		m.applyMeta(msg.meta)
		m.clearErr()
		if msg.noData {
			m.statusNote = "no such contract"
			return m, nil
		}
		return m, fetchContract(m.client, msg.occ)

	case errMsg:
		m.lastErr = msg.err
		m.lastErrOp = msg.op
		var rle *marketdata.RateLimitError
		if errors.As(msg.err, &rle) {
			m.suspendedUntil = rle.ResetAt
		}
		return m, nil

	case refreshTickMsg:
		var cmds []tea.Cmd
		if !m.now().Before(m.suspendedUntil) {
			cmds = append(cmds, m.chainCmds()...)
		}
		cmds = append(cmds, m.scheduleRefreshTick())
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// applyMeta folds a response's rate-limit metadata into credits. Every
// success message carries meta, but it can be nil in hand-built test
// messages, so this always nil-guards.
func (m *model) applyMeta(meta *marketdata.Response) {
	if meta != nil {
		m.credits = meta.RateLimit
	}
}

// clearErr clears the last-error state. Called by every successful message
// handler in Update: a new success means whatever failed before no longer
// applies to the status line.
func (m *model) clearErr() {
	m.lastErr = nil
	m.lastErrOp = ""
}

// handleKey dispatches a key press. ctrl+c always quits, even while an
// input has focus. While the symbol or lookup input has focus, every other
// key is either consumed by the input (typing) or handled by that input's
// own enter/esc cases — the chain/expirations hotkeys below are inert.
func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.focus {
	case focusSymbol:
		return m.handleSymbolInputKey(msg)
	case focusLookup:
		return m.handleLookupInputKey(msg)
	}

	switch msg.String() {
	case "tab", "left", "right":
		m.focus = togglePaneFocus(m.focus)
		return m, nil
	case "up":
		return m.moveSelection(-1)
	case "down":
		return m.moveSelection(1)
	case "enter":
		return m.activateSelectedRow()
	case "esc":
		return m.closeTopModal()
	case "q":
		return m, tea.Quit
	case "c":
		m.side = options.SideCall
		return m, tea.Batch(m.chainCmds()...)
	case "p":
		m.side = options.SidePut
		return m, tea.Batch(m.chainCmds()...)
	case "b":
		m.side = options.SideBoth
		return m, tea.Batch(m.chainCmds()...)
	case "g":
		m.showGreeks = !m.showGreeks
		return m, nil
	case "+":
		m.window = clampWindow(m.window + windowStep)
		if m.underlyingPx > 0 {
			return m, tea.Batch(m.chainCmds()...)
		}
		return m, nil
	case "-":
		m.window = clampWindow(m.window - windowStep)
		if m.underlyingPx > 0 {
			return m, tea.Batch(m.chainCmds()...)
		}
		return m, nil
	case "s":
		m.symbolInput.SetValue(m.symbol)
		m.symbolInput.CursorEnd()
		m.focus = focusSymbol
		return m, m.symbolInput.Focus()
	case "/":
		m.lookupInput.SetValue("")
		m.focus = focusLookup
		return m, m.lookupInput.Focus()
	case " ":
		m.togglePin()
		return m, nil
	case "E":
		m.showSupport = true
		return m, nil
	case "r":
		m.suspendedUntil = time.Time{}
		return m, tea.Batch(m.chainCmds()...)
	}
	return m, nil
}

// handleSymbolInputKey handles a key press while focus == focusSymbol.
// Enter uppercases and trims the input, resets the chain-derived state, and
// reloads the new symbol; esc cancels without changing m.symbol. Every
// other key is forwarded to the textinput so the user can type.
func (m model) handleSymbolInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.symbolInput.Blur()
		m.focus = focusChain
		return m, nil
	case "enter":
		sym := strings.ToUpper(strings.TrimSpace(m.symbolInput.Value()))
		m.symbolInput.Blur()
		m.focus = focusChain
		if sym == "" {
			return m, nil
		}
		m = m.resetForNewSymbol(sym)
		return m, tea.Batch(m.loadSymbolCmds()...)
	}
	var cmd tea.Cmd
	m.symbolInput, cmd = m.symbolInput.Update(msg)
	return m, cmd
}

// handleLookupInputKey handles a key press while focus == focusLookup.
// Enter parses the query with parseLookup; a parse error sets a usage-hint
// statusNote and leaves the input open for correction (no fetch), while a
// parse success closes the input and issues lookupContract. esc cancels.
func (m model) handleLookupInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.lookupInput.Blur()
		m.lookupInput.SetValue("")
		m.focus = focusChain
		return m, nil
	case "enter":
		underlying, exp, strike, typ, err := parseLookup(m.lookupInput.Value(), m.now())
		if err != nil {
			m.statusNote = "usage: SYMBOL YYYY-MM-DD STRIKE call|put"
			return m, nil
		}
		m.lookupInput.Blur()
		m.lookupInput.SetValue("")
		m.focus = focusChain
		return m, lookupContract(m.client, underlying, exp, strike, typ)
	}
	var cmd tea.Cmd
	m.lookupInput, cmd = m.lookupInput.Update(msg)
	return m, cmd
}

// resetForNewSymbol clears everything derived from the previous symbol's
// chain (expirations, chain, rows, underlying price, open detail modal,
// status note, last error) and sets symbol to sym. Side, window, showGreeks,
// and the pin list are user preferences that survive a symbol change.
func (m model) resetForNewSymbol(sym string) model {
	m.symbol = sym
	m.underlying = nil
	m.expirations = nil
	m.expSelected = 0
	m.chain = nil
	m.rows = nil
	m.rowSelected = 0
	m.underlyingPx = 0
	m.detail = nil
	m.statusNote = ""
	m.clearErr()
	return m
}

// moveSelection moves the selection in the currently focused pane by delta
// (-1 or +1), clamped to the pane's bounds. Moving the expiration selection
// triggers a chain reload for the newly selected expiration (using the
// current window); moving the chain-row selection does not fetch anything.
func (m model) moveSelection(delta int) (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusExpirations:
		if len(m.expirations) == 0 {
			return m, nil
		}
		next := clampIndex(m.expSelected+delta, len(m.expirations))
		if next == m.expSelected {
			return m, nil
		}
		m.expSelected = next
		return m, tea.Batch(m.chainCmds()...)
	case focusChain:
		if len(m.rows) == 0 {
			return m, nil
		}
		m.rowSelected = clampIndex(m.rowSelected+delta, len(m.rows))
		return m, nil
	}
	return m, nil
}

// activateSelectedRow issues fetchContract for the currently selected chain
// row's OCC symbol, regardless of which pane has focus (the selection
// itself is shared state, not pane-local): this is what Enter does outside
// the symbol/lookup inputs.
func (m model) activateSelectedRow() (tea.Model, tea.Cmd) {
	if len(m.rows) == 0 || m.rowSelected < 0 || m.rowSelected >= len(m.rows) {
		return m, nil
	}
	occ := m.rows[m.rowSelected].OptionSymbol
	return m, fetchContract(m.client, occ)
}

// closeTopModal closes whichever modal is open, in priority order: the
// contract detail modal, then the support-info modal. A no-op if neither is
// open.
func (m model) closeTopModal() (tea.Model, tea.Cmd) {
	switch {
	case m.detail != nil:
		m.detail = nil
	case m.showSupport:
		m.showSupport = false
	}
	return m, nil
}

// togglePin pins the selected row's OCC symbol if it is not already pinned,
// or unpins it (removing it from both pinned and pinData) if it is. Pin
// order is preserved: unpinning removes exactly one entry, appending always
// adds to the end.
func (m *model) togglePin() {
	if len(m.rows) == 0 || m.rowSelected < 0 || m.rowSelected >= len(m.rows) {
		return
	}
	occ := m.rows[m.rowSelected].OptionSymbol
	for i, sym := range m.pinned {
		if sym == occ {
			m.pinned = append(m.pinned[:i], m.pinned[i+1:]...)
			delete(m.pinData, occ)
			return
		}
	}
	m.pinned = append(m.pinned, occ)
}

// togglePaneFocus swaps between the two non-input panes. tab, left, and
// right all invoke it: with exactly two panes, "switch panes" has one
// meaning regardless of which of the three keys triggered it.
func togglePaneFocus(f focus) focus {
	if f == focusExpirations {
		return focusChain
	}
	return focusExpirations
}

// selectNearestExpiration returns the index of the expiration with the
// smallest non-negative dte(exp, now) — the nearest one that has not yet
// passed. If every expiration has already passed, it falls back to the
// last one in exps. It returns -1 for an empty exps.
func selectNearestExpiration(exps []time.Time, now time.Time) int {
	best := -1
	bestDTE := 0
	for i, exp := range exps {
		d := dte(exp, now)
		if d >= 0 && (best == -1 || d < bestDTE) {
			best = i
			bestDTE = d
		}
	}
	if best == -1 && len(exps) > 0 {
		return len(exps) - 1
	}
	return best
}

// sortedRows returns a copy of opts sorted by strike ascending, with calls
// ordered before puts at equal strike.
func sortedRows(opts []options.OptionQuote) []options.OptionQuote {
	rows := make([]options.OptionQuote, len(opts))
	copy(rows, opts)
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Strike != rows[j].Strike {
			return rows[i].Strike < rows[j].Strike
		}
		return rows[i].Type == options.Call && rows[j].Type == options.Put
	})
	return rows
}

// atmIndex returns the index into rows whose Strike is nearest underlyingPx
// (the at-the-money contract) — used by the view (Task 3.5) to highlight
// that row. It returns -1 for an empty rows. Ties (equal distance) resolve
// to the earlier index, i.e. the lower strike given sortedRows' ordering.
func atmIndex(rows []options.OptionQuote, underlyingPx float64) int {
	best := -1
	bestDist := 0.0
	for i, r := range rows {
		dist := r.Strike - underlyingPx
		if dist < 0 {
			dist = -dist
		}
		if best == -1 || dist < bestDist {
			best = i
			bestDist = dist
		}
	}
	return best
}

// clampIndex bounds i to [0, n-1], or 0 when n <= 0.
func clampIndex(i, n int) int {
	if n <= 0 {
		return 0
	}
	if i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}
