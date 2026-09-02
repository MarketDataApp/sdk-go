// views.go renders the model built by app.go: a bordered dashboard box
// (header, watchlist table, detail pane, footer) when no modal is open,
// or one of five full-pane modals otherwise. Every rendering function is
// a pure function of the model (plus the width/height carried on it by
// the last tea.WindowSizeMsg) — none of them perform I/O or read the
// wall clock directly; the header clock, the "refreshed HH:MM:SS" line,
// and the rate-limit status text all go through m.now(), so tests can
// freeze time and get a byte-identical frame every run (see views_test.go
// and testdata/*.golden).
//
// Layout discipline used throughout this file: a column's text is padded
// to its final width *before* it is wrapped in a lipgloss style. Padding
// after styling would count the invisible ANSI escape bytes as part of
// the string's length and misalign the column; padding before styling
// avoids that entirely, and the few places that must measure
// already-styled text (boxLine's line padding) use lipgloss.Width, which
// is ANSI-aware.
package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
)

// Fixed watchlist column widths (plain-text, pre-styling). Shared by the
// header row and every data row so they always line up.
const (
	colSymbolW = 9
	colLastW   = 10
	colChgW    = 10
	colChgPctW = 10
	colBidAskW = 22
	colVolumeW = 15
)

// sparkWidth and rangeBarWidth bound the detail pane's two block-character
// visualizations.
const (
	sparkWidth    = 40
	rangeBarWidth = 34
)

// minWidth and minHeight are the smallest terminal size the dashboard
// will attempt to render; below this, View returns a one-line status
// instead of a garbled box.
const (
	minWidth  = 40
	minHeight = 12
)

// easternLocation is loaded once at package init for formatting
// market-facing timestamps (the header clock, the credit-reset time) in
// US/Eastern regardless of the process's local timezone. It falls back
// to UTC if the platform has no tzdata, rather than panicking — a
// reference app should degrade, not crash, when a container image is
// missing the timezone database.
var easternLocation = loadEasternLocation()

func loadEasternLocation() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return time.UTC
	}
	return loc
}

// View renders the current screen: the watchlist dashboard, or — while a
// modal is open — a full-pane swap to that modal's content. Both are
// built at the model's last-known width/height and clipped defensively
// with lipgloss's MaxWidth/MaxHeight in case a rendering helper ever
// miscalculates a line's length.
func (m model) View() string {
	width, height := m.width, m.height
	if width < minWidth || height < minHeight {
		return "stockterm — waiting for terminal size..."
	}

	var body string
	switch m.modal {
	case modalStatusHistory:
		body = m.renderStatusHistoryModal(width, height)
	case modalDiagnostics:
		body = m.renderDiagnosticsModal(width, height)
	case modalError:
		body = m.renderErrorModal(width, height)
	case modalBulk:
		body = m.renderBulkModal(width, height)
	case modalAdd:
		body = m.renderAddModal(width, height)
	default:
		body = m.renderDashboard(width, height)
	}

	return lipgloss.NewStyle().MaxWidth(width).MaxHeight(height).Render(body)
}

// --- box-drawing primitives ---
//
// Every box is built from these four pieces: a top border (optionally
// carrying a title, as the header does with market status + clock), a
// content line, a mid-box divider (used both as a titled section break
// and, with an empty title, as a plain rule), and a bottom border.

// boxTop renders "┌" + title, padded with "─" to width, + "┐". title
// should include any leading/trailing spacing the caller wants around
// the text (e.g. " Market: OPEN  ─  ... ET ").
func boxTop(width int, title string) string {
	return boxRule(width, "┌", "┐", title)
}

// boxDivider renders "├" + title + "┤", or a plain "├──...──┤" rule when
// title is empty.
func boxDivider(width int, title string) string {
	t := ""
	if title != "" {
		t = " " + title + " "
	}
	return boxRule(width, "├", "┤", t)
}

// boxRule is the shared implementation of boxTop and boxDivider: it fits
// title into the inner width (truncating if necessary) and fills the
// remainder with "─".
func boxRule(width int, left, right, title string) string {
	inner := width - 2
	t := title
	if n := len([]rune(t)); n > inner {
		t = truncate(t, inner)
	}
	fill := inner - len([]rune(t))
	if fill < 0 {
		fill = 0
	}
	return borderStyle.Render(left) + titleStyle.Render(t) +
		borderStyle.Render(strings.Repeat("─", fill)) + borderStyle.Render(right)
}

// boxBottom renders "└" + "─"*width + "┘".
func boxBottom(width int) string {
	return borderStyle.Render("└" + strings.Repeat("─", width-2) + "┘")
}

// boxLine renders "│ " + content, padded (or, if content is too wide,
// ANSI-safely clipped) to the box's content width, + " │". content may
// already carry lipgloss styling (colored numbers, a dimmed placeholder,
// ...); both the overflow check and the pad amount are computed with
// lipgloss.Width so embedded ANSI codes are never mistaken for visible
// characters.
//
// The overflow clip matters beyond goldens: every caller here formats
// its own content assuming it fits, but a couple of lines echo live API
// text of unbounded length (a support-ticket message, a request URL) —
// without this, a long one would push right past the box and the outer
// View() MaxWidth clamp would truncate the line from the *right*,
// silently eating the closing border instead of the overflowing text.
func boxLine(width int, content string) string {
	cw := width - 4
	if lipgloss.Width(content) > cw {
		content = lipgloss.NewStyle().MaxWidth(cw).Render(content)
	}
	pad := cw - lipgloss.Width(content)
	if pad < 0 {
		pad = 0
	}
	return borderStyle.Render("│") + " " + content + strings.Repeat(" ", pad) + " " + borderStyle.Render("│")
}

// --- plain-text column helpers ---
//
// These operate on unstyled text and must run before any lipgloss
// styling is applied (see the file comment).

// padRight left-justifies s within w runes, truncating (with an ellipsis)
// rather than overflowing if s is already longer than w.
func padRight(s string, w int) string {
	r := []rune(s)
	if len(r) >= w {
		return truncate(s, w)
	}
	return s + strings.Repeat(" ", w-len(r))
}

// padLeft right-justifies s within w runes, truncating (with an ellipsis)
// rather than overflowing if s is already longer than w.
func padLeft(s string, w int) string {
	r := []rune(s)
	if len(r) >= w {
		return truncate(s, w)
	}
	return strings.Repeat(" ", w-len(r)) + s
}

// padCenter centers s (which may already be lipgloss-styled) within w
// visible columns, measured with lipgloss.Width.
func padCenter(s string, w int) string {
	vis := lipgloss.Width(s)
	if vis >= w {
		return s
	}
	total := w - vis
	left := total / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", total-left)
}

// truncate hard-truncates s to at most w runes, replacing the final rune
// with "…" when a cut was needed so truncation is visible rather than
// silent.
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	return string(r[:w-1]) + "…"
}

// signedf formats v to two decimals with an explicit "+" for
// non-negative values, e.g. signedf(1.24) == "+1.24", signedf(-2.01) ==
// "-2.01".
func signedf(v float64) string {
	if v >= 0 {
		return fmt.Sprintf("+%.2f", v)
	}
	return fmt.Sprintf("%.2f", v)
}

// signedPct is signedf with a trailing "%".
func signedPct(v float64) string {
	if v >= 0 {
		return fmt.Sprintf("+%.2f%%", v)
	}
	return fmt.Sprintf("%.2f%%", v)
}

// --- dashboard (no modal open) ---

// renderDashboard builds the full watchlist box: header, table, detail
// pane, and footer. The number of watchlist rows shown is capped to
// whatever fits in height alongside the fixed-size sections around it,
// so a long watchlist truncates gracefully instead of blowing out the
// frame.
func (m model) renderDashboard(width, height int) string {
	var b strings.Builder

	b.WriteString(boxTop(width, m.renderHeaderTitle()))
	b.WriteByte('\n')

	fixed := 2 // top + bottom border
	if m.demoMode {
		b.WriteString(boxLine(width, padCenter(warningStyle.Render("DEMO MODE — AAPL data only"), width-4)))
		b.WriteByte('\n')
		fixed++
	}

	b.WriteString(boxLine(width, m.renderWatchlistHeader()))
	b.WriteByte('\n')
	fixed++ // watchlist header

	// divider + 4 detail lines + footer divider + footer = 7 more fixed
	// lines around the variable-length watchlist row block.
	fixed += 7

	rowBudget := height - fixed
	if rowBudget < 0 {
		rowBudget = 0
	}
	n := len(m.symbols)
	if n > rowBudget {
		n = rowBudget
	}
	for i := 0; i < n; i++ {
		sym := m.symbols[i]
		b.WriteString(boxLine(width, m.renderWatchlistRow(sym, i == m.selected)))
		b.WriteByte('\n')
	}

	selected := ""
	if len(m.symbols) > 0 {
		selected = m.symbols[m.selected]
	}
	isFund := m.funds[selected]

	b.WriteString(boxDivider(width, selected))
	b.WriteByte('\n')
	b.WriteString(boxLine(width, m.renderSparklineLine(isFund)))
	b.WriteByte('\n')
	b.WriteString(boxLine(width, m.render52wkLine()))
	b.WriteByte('\n')
	b.WriteString(boxLine(width, m.renderEarningsLine()))
	b.WriteByte('\n')
	b.WriteString(boxLine(width, m.renderNewsLine(width-4)))
	b.WriteByte('\n')
	b.WriteString(boxDivider(width, ""))
	b.WriteByte('\n')
	b.WriteString(boxLine(width, m.renderFooterLine()))
	b.WriteByte('\n')
	b.WriteString(boxBottom(width))

	return b.String()
}

// renderHeaderTitle builds the top border's title: market open/closed
// status and the current clock, both shown in US/Eastern (the market's
// timezone) regardless of what timezone m.now() itself carries.
func (m model) renderHeaderTitle() string {
	label := "—"
	if m.market != nil {
		label = strings.ToUpper(m.market.Status)
	}
	clock := m.now().In(easternLocation).Format("Mon 2006-01-02 15:04:05")
	return fmt.Sprintf(" Market: %s  ─  %s ET ", label, clock)
}

// renderWatchlistHeader builds the SYMBOL/LAST/CHG/CHG%/BID x
// ASK/VOLUME column header row, laid out identically to
// renderWatchlistRow's data rows.
func (m model) renderWatchlistHeader() string {
	row := layoutWatchlistRow(
		padRight("SYMBOL", colSymbolW),
		padLeft("LAST", colLastW),
		padLeft("CHG", colChgW),
		padLeft("CHG%", colChgPctW),
		padRight("BID x ASK", colBidAskW),
		padLeft("VOLUME", colVolumeW),
		" ",
	)
	return tableHeadStyle.Render(row)
}

// layoutWatchlistRow joins seven already-width-formatted columns
// (symbol, last, chg, chgPct, bidAsk, volume, marker) with the
// watchlist's fixed separators. Columns may already be lipgloss-styled;
// this function only concatenates, so it never re-measures a column's
// width — see the file comment.
func layoutWatchlistRow(symbol, last, chg, chgPct, bidAsk, volume, marker string) string {
	return symbol + last + " " + chg + " " + chgPct + "   " + bidAsk + volume + "   " + marker
}

// renderWatchlistRow builds one data row for sym: quotes (or, under
// -prices, the lighter Price type — which has no bid/ask/volume, shown
// as "—") when present, "—" placeholders for a symbol whose first
// refresh hasn't landed yet, and the "◀" marker when selected.
func (m model) renderWatchlistRow(sym string, selected bool) string {
	var lastStr, bidAskStr, volumeStr string
	var chgV, chgPctV float64
	hasData := false

	if m.usePrices {
		if p, ok := m.prices[sym]; ok {
			hasData = true
			lastStr = fmt.Sprintf("%.2f", p.Mid)
			chgV, chgPctV = p.Change, p.ChangePercent*100
		}
		bidAskStr, volumeStr = "—", "—"
	} else {
		if q, ok := m.quotes[sym]; ok {
			hasData = true
			lastStr = fmt.Sprintf("%.2f", q.Last)
			chgV, chgPctV = q.Change, q.ChangePercent*100
			bidAskStr = fmt.Sprintf("%.2f x %.2f", q.Bid, q.Ask)
			volumeStr = comma(q.Volume)
		} else {
			bidAskStr, volumeStr = "—", "—"
		}
	}

	last := padLeft("—", colLastW)
	chg := padLeft("—", colChgW)
	chgPct := padLeft("—", colChgPctW)
	chgStyled := mutedStyle.Render(chg)
	chgPctStyled := mutedStyle.Render(chgPct)
	if hasData {
		last = padLeft(lastStr, colLastW)
		chg = padLeft(signedf(chgV), colChgW)
		chgPct = padLeft(signedPct(chgPctV), colChgPctW)
		chgStyled = signStyle(chgV).Render(chg)
		chgPctStyled = signStyle(chgPctV).Render(chgPct)
	}

	marker := " "
	if selected {
		marker = markerStyle.Render("◀")
	}

	return layoutWatchlistRow(
		padRight(sym, colSymbolW),
		last,
		chgStyled,
		chgPctStyled,
		padRight(bidAskStr, colBidAskW),
		padLeft(volumeStr, colVolumeW),
		marker,
	)
}

// rangeLabelText describes rng the way the sparkline line's caption
// shows it.
func rangeLabelText(rng candleRange) string {
	switch rng {
	case rangeIntraday:
		return "intraday, 1d"
	case rangeWeekly:
		return "weekly, 1y"
	default:
		return "daily, 3mo"
	}
}

// renderSparklineLine builds the detail pane's first line: a candle-close
// sparkline flanked by its min/max labels, followed by the active range's
// caption. isFund selects fundCandles over candles and prefixes the
// labels with "NAV" (a fund's price is its net asset value, not a
// traded "last"). Renders "—" when the relevant candle fetch came back
// with no data, or hasn't landed yet.
//
// The fund caption is always the daily label, ignoring m.rng: fund
// candles come from one fixed fetch (daily, countback 63) regardless of
// range, and m.rng can legitimately be non-daily while a fund is selected
// — the range keys are fund no-ops, but rng survives selection changes as
// a stock-only preference — so labeling fund data with rangeLabelText(m.rng)
// would misdescribe what was actually fetched.
func (m model) renderSparklineLine(isFund bool) string {
	var closes []float64
	noData := false
	if isFund {
		noData = m.detailNoData["candles"] || len(m.fundCandles) == 0
		for _, c := range m.fundCandles {
			closes = append(closes, c.Close)
		}
	} else {
		noData = m.detailNoData["candles"] || len(m.candles) == 0
		for _, c := range m.candles {
			closes = append(closes, c.Close)
		}
	}
	if noData {
		return mutedStyle.Render("—")
	}

	min, max := closes[0], closes[0]
	for _, v := range closes {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	prefix := ""
	rng := m.rng
	if isFund {
		prefix = "NAV "
		rng = rangeDaily // fund candles are always daily (see doc comment)
	}
	minLabel := padLeft(prefix+fmt.Sprintf("%.2f", min), 12)
	maxLabel := padRight(fmt.Sprintf("%.2f", max), 12)
	spark := sparkline(closes, sparkWidth)

	return minLabel + " " + spark + " " + maxLabel + "   " + mutedStyle.Render(rangeLabelText(rng))
}

// render52wkLine builds the detail pane's 52-week range bar, sourced
// from m.detail (client.Stocks.Quote with WithFiftyTwoWeek). Renders
// "—" when that fetch came back with no data (routine for a symbol the
// stock quote endpoint doesn't cover, such as a mutual fund) or hasn't
// landed yet.
func (m model) render52wkLine() string {
	if m.detailNoData["quote"] || m.detail == nil {
		return "52wk: " + mutedStyle.Render("—")
	}
	low, high, last := m.detail.FiftyTwoWeekLow, m.detail.FiftyTwoWeekHigh, m.detail.Last
	bar := rangeBar(low, high, last, rangeBarWidth)
	if bar == "" {
		return "52wk: " + mutedStyle.Render("—")
	}
	lowLabel := padLeft(fmt.Sprintf("%.2f", low), 8)
	highLabel := padRight(fmt.Sprintf("%.2f", high), 8)
	return "52wk: " + lowLabel + " " + bar + " " + highLabel
}

// renderEarningsLine builds the "Next earnings: ..." line from the
// nearest future report in m.earnings. Renders "no scheduled earnings"
// — not a bare "—" — per the design's documented no-data behavior for
// this specific line.
func (m model) renderEarningsLine() string {
	if m.detailNoData["earnings"] || len(m.earnings) == 0 {
		return "Next earnings: " + mutedStyle.Render("no scheduled earnings")
	}
	next := m.earnings[0]
	for _, e := range m.earnings[1:] {
		if e.ReportDate.Before(next.ReportDate) {
			next = e
		}
	}
	if next.EstimatedEPS != nil {
		return fmt.Sprintf("Next earnings: %s (est EPS $%.2f)", next.ReportDate.Format("2006-01-02"), *next.EstimatedEPS)
	}
	return fmt.Sprintf("Next earnings: %s", next.ReportDate.Format("2006-01-02"))
}

// renderNewsLine builds the three-headline bullet line from m.news
// (already limited to 3 articles by stocks.WithNewsWindow(stocks.LastN(3))),
// truncating each headline to fit width. Renders "—" when the fetch
// came back with no data or hasn't landed yet.
func (m model) renderNewsLine(width int) string {
	if m.detailNoData["news"] || len(m.news) == 0 {
		return mutedStyle.Render("—")
	}
	n := len(m.news)
	// Every headline costs 2 runes for its "• " bullet, plus a 2-rune
	// "  " separator between items (n-1 of those). What's left, split
	// evenly across n headlines, is each one's truncation budget.
	overhead := 2*n + 2*(n-1)
	budget := max(8, (width-overhead)/n)
	parts := make([]string, 0, n)
	for _, a := range m.news {
		parts = append(parts, "• "+truncate(a.Headline, budget))
	}
	return strings.Join(parts, "  ")
}

// renderFooterLine builds the credit meter, reset time, last-refresh
// time, and status line.
//
// The credit meter reads m.credits, which Update populates from the
// *per-response* RateLimit metadata of whichever fetch most recently
// completed (see metaOf in app.go) — a request-scoped, always-exact
// snapshot. That's deliberately different from client.RateLimits(),
// which returns the client's own cached snapshot of the last completed
// request: convenient for a one-off check, but liable to lag behind
// reality under concurrent requests (several fetches in flight at once,
// as this app's Init and detail cascade both do). A dashboard that must
// show the *current* row's true credit cost — as opposed to "credits as
// of some recent request" — should always prefer the per-response value,
// which is why every fetch message in this app carries one.
func (m model) renderFooterLine() string {
	// Window total, not the per-request cost: m.credits.Consumed is what the
	// last response cost, which read as a session figure under this label.
	consumed := comma(int64(m.credits.Limit - m.credits.Remaining))
	limit := comma(int64(m.credits.Limit))

	resets := "—"
	if !m.credits.ResetAt.IsZero() {
		resets = m.credits.ResetAt.In(easternLocation).Format("15:04") + " ET"
	}

	refreshed := "—"
	if !m.lastRefresh.IsZero() {
		refreshed = m.lastRefresh.In(easternLocation).Format("15:04:05")
	}

	status := "[no error]"
	switch {
	case m.lastErr != nil:
		status = errorStyle.Render(classify(m.lastErr, m.now()))
	case m.statusNote != "":
		status = m.statusNote
	}

	return fmt.Sprintf("credits %s/%s   resets %s   refreshed %s   %s", consumed, limit, resets, refreshed, status)
}

// --- modals ---
//
// Every modal is a full-pane swap (View picks exactly one of these, or
// renderDashboard, never both): a titled box the same 100x40 size as the
// dashboard, holding a title line and a list of content lines capped to
// what fits in height. Consistent shape across all five keeps the
// golden frames — and a user's mental model of "esc always closes
// whatever's on screen" — simple.

// renderModalFrame is the shared modal shell: a titled top border, up to
// height-2 content lines, and a bottom border.
func (m model) renderModalFrame(width, height int, title string, lines []string) string {
	var b strings.Builder
	b.WriteString(boxTop(width, " "+title+" "))
	b.WriteByte('\n')

	maxLines := height - 2
	if maxLines < 0 {
		maxLines = 0
	}
	for i, l := range lines {
		if i >= maxLines {
			break
		}
		b.WriteString(boxLine(width, l))
		b.WriteByte('\n')
	}
	b.WriteString(boxBottom(width))
	return b.String()
}

// renderStatusHistoryModal ('m') lists the last five trading days' open/
// closed status from m.history (client.Markets.StatusHistory).
func (m model) renderStatusHistoryModal(width, height int) string {
	lines := []string{tableHeadStyle.Render(padRight("DATE", 14) + padRight("STATUS", 14))}
	if len(m.history) == 0 {
		lines = append(lines, mutedStyle.Render("—"))
	} else {
		for _, s := range m.history {
			lines = append(lines, padRight(s.Date.Format("2006-01-02"), 14)+padRight(strings.ToUpper(s.Status), 14))
		}
	}
	lines = append(lines, "", mutedStyle.Render("esc to close"))
	return m.renderModalFrame(width, height, "Market Status History (last 5 days)", lines)
}

// renderDiagnosticsModal ('D') shows the API's own uptime status
// (client.Utilities.Status) and the request headers the client actually
// sent (client.Utilities.Headers) — the debugging flow support asks
// users to run.
func (m model) renderDiagnosticsModal(width, height int) string {
	var lines []string

	lines = append(lines, titleStyle.Render("API Status"))
	if m.apiStatus == nil {
		lines = append(lines, mutedStyle.Render("—"))
	} else {
		lines = append(lines, fmt.Sprintf("status: %s   uptime 30d: %.2f%%   uptime 90d: %.2f%%",
			m.apiStatus.Status, m.apiStatus.Uptime30d, m.apiStatus.Uptime90d))
	}

	lines = append(lines, "", titleStyle.Render("Request Headers"))
	if m.headers == nil || len(m.headers.Headers) == 0 {
		lines = append(lines, mutedStyle.Render("—"))
	} else {
		keys := make([]string, 0, len(m.headers.Headers))
		for k := range m.headers.Headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			lines = append(lines, fmt.Sprintf("%s: %s", k, m.headers.Headers[k]))
		}
	}

	lines = append(lines, "", mutedStyle.Render("esc to close"))
	return m.renderModalFrame(width, height, "Diagnostics", lines)
}

// renderErrorModal ('E') shows the most recent error's SupportInfo()
// block verbatim when it implements marketdata.Error (every SDK-typed
// error does); for a plain error (one that never reached the SDK's HTTP
// layer, or was constructed outside it) it falls back to err.Error().
func (m model) renderErrorModal(width, height int) string {
	var lines []string
	if m.lastErr == nil {
		lines = append(lines, mutedStyle.Render("[no error]"))
	} else {
		lines = append(lines,
			"op: "+m.lastErrOp,
			classify(m.lastErr, m.now()),
			"",
		)
		var sdkErr marketdata.Error
		if errors.As(m.lastErr, &sdkErr) {
			lines = append(lines, strings.Split(sdkErr.SupportInfo(), "\n")...)
		} else {
			lines = append(lines, m.lastErr.Error())
		}
	}
	lines = append(lines, "", mutedStyle.Render("esc to close"))
	return m.renderModalFrame(width, height, "Error / Support Info", lines)
}

// renderBulkModal ('o') shows one day's OHLC + change% for every
// watchlist symbol from a single client.Stocks.BulkCandles call.
func (m model) renderBulkModal(width, height int) string {
	head := padRight("SYMBOL", colSymbolW) +
		padLeft("OPEN", 10) + padLeft("HIGH", 10) + padLeft("LOW", 10) +
		padLeft("CLOSE", 10) + padLeft("CHG%", 10) + padLeft("VOLUME", 15)
	lines := []string{tableHeadStyle.Render(head)}

	if len(m.bulk) == 0 {
		lines = append(lines, mutedStyle.Render("—"))
	} else {
		for _, c := range m.bulk {
			chgPct := 0.0
			if c.Open != 0 {
				chgPct = (c.Close - c.Open) / c.Open * 100
			}
			row := padRight(c.Symbol, colSymbolW) +
				padLeft(fmt.Sprintf("%.2f", c.Open), 10) +
				padLeft(fmt.Sprintf("%.2f", c.High), 10) +
				padLeft(fmt.Sprintf("%.2f", c.Low), 10) +
				padLeft(fmt.Sprintf("%.2f", c.Close), 10) +
				signStyle(chgPct).Render(padLeft(signedPct(chgPct), 10)) +
				padLeft(comma(c.Volume), 15)
			lines = append(lines, row)
		}
	}

	lines = append(lines, "", mutedStyle.Render("esc to close"))
	return m.renderModalFrame(width, height, "Day Performance", lines)
}

// renderAddModal ('a') shows the add-symbol text input.
func (m model) renderAddModal(width, height int) string {
	lines := []string{
		"Enter a symbol to add to the watchlist.",
		"",
		m.input.View(),
		"",
		mutedStyle.Render("enter to add, esc to cancel"),
	}
	return m.renderModalFrame(width, height, "Add Symbol", lines)
}
