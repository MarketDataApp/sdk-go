// View rendering: the pane layout (symbol bar, expirations sidebar, chain
// table, pinboard strip, footer) and the two modals (contract detail,
// support info). Every function here is pure given the model — no I/O, no
// time.Now() (DTE and duration math always goes through m.now(), matching
// the rest of the app) — so the golden-frame tests in views_test.go can
// pin exact output for a fixed clock and a fixed tea.WindowSizeMsg.
//
// Layout strategy: every line this file produces is built as fixed-width
// plain text first (padded/truncated with padRight/padLeft/truncate, which
// operate on rune counts) and only then wrapped, whole, in at most one
// lipgloss style. That order matters: styling before measuring would let
// invisible escape codes count as visible width and break alignment. Since
// each line is a leaf — nothing downstream re-measures it — wrapping a
// complete, already-fixed-width line in a style is always safe.
package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
)

// Layout constants. sidebarColumnWidth is the fixed content width of the
// expirations sidebar; everything else in the frame (the chain/pinned pane,
// the footer, the modals) sizes itself from m.width.
//
// minFrameHeight guarantees the chain pane's viewport math never
// degenerates: the tallest chrome (6 lines, demo banner included) plus the
// chain header (1), the reserved pinboard block (3), and the smallest
// useful viewport (1 row between 2 truncation hints) needs 13 lines; 14
// leaves one spare.
const (
	sidebarColumnWidth = 13
	minFrameWidth      = 40
	minFrameHeight     = 14
)

// View implements tea.Model.
func (m model) View() string {
	if m.width < minFrameWidth || m.height < minFrameHeight {
		return "optionterm: terminal too small (need at least 40x14)\n"
	}

	title := "options chain"
	var body []string
	switch {
	case m.detail != nil:
		title = "contract detail"
		body = m.detailModalLines()
	case m.showSupport:
		title = "support info"
		body = m.supportModalLines()
	default:
		body = m.bodyLines()
	}

	var b strings.Builder
	b.WriteString(m.topBorder(title))
	b.WriteString("\n")
	if m.demoMode {
		b.WriteString(m.demoBannerLine())
		b.WriteString("\n")
	}
	for _, line := range body {
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString(m.midBorder())
	b.WriteString("\n")
	b.WriteString(m.footerLine1())
	b.WriteString("\n")
	b.WriteString(m.footerLine2())
	b.WriteString("\n")
	b.WriteString(m.bottomBorder())
	return b.String()
}

// innerWidth is the content width available between the frame's left and
// right border characters.
func (m model) innerWidth() int {
	return max0(m.width - 2)
}

// --- frame chrome: borders, symbol bar, demo banner ---

// topBorder renders the frame's top edge, embedding the underlying symbol,
// its last price (from m.underlying, "—" before the first load), and a
// title naming the current view ("options chain", "contract detail",
// "support info").
func (m model) topBorder(title string) string {
	last := "—"
	if m.underlying != nil {
		last = fmt.Sprintf("%.2f", m.underlying.Last)
	}
	left := fmt.Sprintf("┌ %s  %s  ─  %s ", m.symbol, last, title)
	n := max0(m.width - len([]rune(left)) - 1)
	return symbolStyle.Render(left + strings.Repeat("─", n) + "┐")
}

// midBorder renders the separator between the body and the footer.
func (m model) midBorder() string {
	return borderStyle.Render("├" + strings.Repeat("─", max0(m.width-2)) + "┤")
}

// bottomBorder renders the frame's bottom edge.
func (m model) bottomBorder() string {
	return borderStyle.Render("└" + strings.Repeat("─", max0(m.width-2)) + "┘")
}

// demoBannerLine renders the persistent demo-mode banner shown when the
// client has no token: a fixed dataset keyed to AAPL.
func (m model) demoBannerLine() string {
	return m.contentLine(" DEMO MODE — AAPL data only", demoBannerStyle)
}

// contentLine wraps text as a single full-width bordered row: "│" + text
// padded/truncated to innerWidth + "│". It is used by both modals for
// every body line, and by the footer's credits/status lines. text must be
// plain (no embedded ANSI) — it goes through truncate, which is unsafe on
// styled text; see contentLineRaw for text that might not be plain.
func (m model) contentLine(text string, style lipgloss.Style) string {
	w := m.innerWidth()
	return "│" + style.Render(padRight(truncate(text, w), w)) + "│"
}

// contentLineRaw is contentLine without the truncate step: it pads but
// never slices. Use it for text that may already carry ANSI escape codes
// — namely a textinput.View(), which styles its cursor even when the rest
// of the line is plain. On an unusually narrow terminal where the input's
// text would overflow, the row is left to overflow the right border
// rather than risk cutting an escape sequence in half.
func (m model) contentLineRaw(text string) string {
	w := m.innerWidth()
	return "│" + padRight(text, w) + "│"
}

// --- footer: credits, key hints, status line, and the lookup/symbol inputs ---

// footerLine1 renders the credit meter and key-binding hints — or, while
// the lookup or symbol input has focus, that input in place of the hints.
func (m model) footerLine1() string {
	switch m.focus {
	case focusLookup:
		// inputPromptStyle wraps only the label: a self-terminated styled
		// span (open code, text, reset) concatenated *before* the input's
		// own (possibly styled) View() output, never nested inside it —
		// sequential styled spans compose safely; nesting one inside
		// another does not (an inner style's reset can clobber the
		// outer one). See contentLineRaw's doc comment for why the input
		// text itself never goes through truncate.
		return m.contentLineRaw(" " + inputPromptStyle.Render("lookup") + " " + m.lookupInput.View())
	case focusSymbol:
		return m.contentLineRaw(" " + inputPromptStyle.Render("symbol") + " " + m.symbolInput.View())
	}

	// m.credits is this response's exact rate-limit snapshot (Limit,
	// Remaining, Consumed, ResetAt), refreshed from the RateLimit field of
	// every fetch's *marketdata.Response (see applyMeta in app.go).
	// client.RateLimits() is a different, client-level view: an
	// eventually-consistent aggregate the client maintains across all
	// requests, useful for a global "how much budget is left" readout but
	// not tied to any one response. The footer intentionally uses the
	// per-response value so the number on screen always matches the data
	// that produced it.
	// Rendered as USED/limit, matching stockterm. The two apps used to print
	// the same "credits N/M" label with opposite meanings — this one showed
	// Remaining, so a full account read as "9,997 of 10,000 used".
	credits := fmt.Sprintf(" credits %s/%s used", commaInt(m.credits.Limit-m.credits.Remaining), commaInt(m.credits.Limit))
	hints := "[/] lookup  [g] greeks  [c/p/b] side  [+/-] rng"
	return m.contentLine(credits+"   "+hints, creditsStyle)
}

// footerLine2 renders the status line: the most recent error (classified
// via classify), or the transient statusNote, or "[no error]" — in that
// priority order, matching the design's "classify(lastErr, m.now()) |
// statusNote | [no error]" rule.
func (m model) footerLine2() string {
	if m.lastErr != nil {
		return m.contentLine(" "+classify(m.lastErr, m.now()), statusErrStyle)
	}
	if m.statusNote != "" {
		return m.contentLine(" "+m.statusNote, statusOKStyle)
	}
	return m.contentLine(" [no error]", statusOKStyle)
}

// --- body: expirations sidebar + chain table + pinboard, side by side ---

// styledLine pairs a line of plain text with the style it should be
// rendered in — used so sidebarLines/chainTableLines/pinnedLines can tag
// each line's emphasis (selected, ATM, ITM, header, ...) without applying
// a style before the line's final width is fixed.
type styledLine struct {
	text  string
	style lipgloss.Style
}

func plainLine(s string) styledLine { return styledLine{text: s, style: plainStyle} }

// bodyLines composes the two-pane body: the expirations sidebar on the
// left, the chain table + pinboard strip on the right, one "│ left │ right
// │" row per line. The two sides are unrelated content lists that happen
// to share vertical position by row index; the shorter one is padded with
// blank cells so the frame's right border stays straight.
//
// The pinboard strip (blank spacer + separator + entries) is reserved
// before the chain table is laid out: the chain gets only what remains of
// the height budget, as a scrolling viewport (see chainViewport), so a
// live-sized unfiltered chain — ~100+ rows on a first load — can never
// push the pinboard off-screen or scroll the selected row out of view.
func (m model) bodyLines() []string {
	innerW := m.innerWidth()
	leftW := sidebarColumnWidth
	chainW := max0(innerW - leftW - 1)
	budget := max0(m.height - m.chromeLines())

	left := m.sidebarLines()

	pinned := m.pinnedLines(chainW)
	reserved := len(pinned) + 1 // + 1: the blank spacer above the strip
	chainAvail := budget - 1 - reserved
	right := m.chainTableLines(chainAvail)
	right = append(right, plainLine(""))
	right = append(right, pinned...)

	n := len(left)
	if len(right) > n {
		n = len(right)
	}
	if budget > 0 && n > budget {
		n = budget
	}

	lines := make([]string, n)
	for i := 0; i < n; i++ {
		l, r := plainLine(""), plainLine("")
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		leftCell := l.style.Render(padRight(truncate(l.text, leftW), leftW))
		rightCell := r.style.Render(padRight(truncate(r.text, chainW), chainW))
		lines[i] = "│" + leftCell + "│" + rightCell + "│"
	}
	return lines
}

// chromeLines counts the frame rows outside the body: the top border, the
// optional demo banner, the mid border, the two footer lines, and the
// bottom border. bodyLines uses it to cap how many body rows it emits so
// the whole frame fits within m.height.
func (m model) chromeLines() int {
	n := 5 // top + mid + footer(2) + bottom
	if m.demoMode {
		n++
	}
	return n
}

// sidebarLines renders the expirations list: one line per expiration
// ("▶" on the selected one), with a "(NN DTE)" line inserted beneath the
// selected entry, per the design's mock.
func (m model) sidebarLines() []styledLine {
	// The leading space lines EXPIRATIONS up with the date column below it:
	// every entry line starts with a one-character marker (" " or "▶")
	// before its date text, so the header needs the same one-character
	// lead-in to stay aligned.
	header := styledLine{text: " EXPIRATIONS", style: headerRowStyle}
	if len(m.expirations) == 0 {
		return []styledLine{header, plainLine("(loading...)")}
	}
	lines := make([]styledLine, 0, len(m.expirations)+2)
	lines = append(lines, header)
	for i, exp := range m.expirations {
		marker := " "
		style := plainStyle
		if i == m.expSelected {
			marker = "▶"
			style = expSelectedStyle
		}
		lines = append(lines, styledLine{text: marker + exp.Format("2006-01-02"), style: style})
		if i == m.expSelected {
			lines = append(lines, styledLine{
				text:  fmt.Sprintf("  (%d DTE)", dte(exp, m.now())),
				style: expSelectedStyle,
			})
		}
	}
	return lines
}

// chainTableLines renders the chain table: a header row for the active
// column set (market data or greeks, per m.showGreeks), then a viewport of
// m.rows sized to avail lines (see chainViewport), or an explanatory note
// when there are no rows. When rows are hidden above or below the viewport,
// a "▲ N more" / "▼ N more" hint line marks each hidden side; the hint
// lines are counted against avail, so the returned slice never exceeds
// 1 (header) + avail lines.
//
// Each data row's marker column is three characters:
//
//	[1] "▶" when this is the keyboard-selected row (m.rowSelected)
//	[2] "A" when this is the at-the-money row (atmIndex)
//	[3] "•" when the contract is in-the-money (InTheMoney)
//
// These are independent conditions (a row can be selected, ATM, and ITM
// all at once, or none of them) and are rendered as plain text precisely
// so the state survives tuitest.Frame's ANSI stripping — the row's color
// style (rowStyle) is the same information again, for a real terminal.
func (m model) chainTableLines(avail int) []styledLine {
	cols := marketColumns()
	if m.showGreeks {
		cols = greekColumns()
	}

	headerCells := make([]string, len(cols))
	for i, c := range cols {
		headerCells[i] = c.header
	}
	header := "    " + formatColumns(cols, headerCells) // 3 marker cols + 1 gap, blank
	lines := []styledLine{{text: header, style: headerRowStyle}}

	if len(m.rows) == 0 {
		note := m.statusNote
		if note == "" {
			note = "no contracts"
		}
		lines = append(lines, plainLine(""), plainLine("  ("+note+")"))
		return lines
	}

	atm := atmIndex(m.rows, m.underlyingPx)
	start, end := chainViewport(len(m.rows), m.rowSelected, avail)
	if end == start {
		// avail <= 0: no room for any row (unreachable while minFrameHeight
		// holds, but kept so the "never exceeds 1+avail lines" contract
		// stays true) — emit no rows and no hints.
		return lines
	}
	if start > 0 {
		lines = append(lines, styledLine{
			text:  fmt.Sprintf("  ▲ %d more", start),
			style: scrollHintStyle,
		})
	}
	for i := start; i < end; i++ {
		row := m.rows[i]
		cells := make([]string, len(cols))
		for c, col := range cols {
			cells[c] = col.format(row)
		}
		selected := i == m.rowSelected
		isATM := i == atm
		text := rowMarker(selected, isATM, row.InTheMoney) + " " + formatColumns(cols, cells)
		lines = append(lines, styledLine{text: text, style: rowStyle(selected, isATM, row.InTheMoney)})
	}
	if end < len(m.rows) {
		lines = append(lines, styledLine{
			text:  fmt.Sprintf("  ▼ %d more", len(m.rows)-end),
			style: scrollHintStyle,
		})
	}
	return lines
}

// chainViewport computes the half-open range [start, end) of chain rows to
// render in a pane with room for avail lines, keeping the selected row
// visible and roughly centered, clamped at both ends of the list. The
// caller renders a truncation-hint line for each hidden side (rows above
// when start > 0, rows below when end < total); those hint lines count
// against avail, so the window shrinks by one row per hint it forces —
// rows + hints never exceed avail, for avail >= 3. (Below that the
// invariant doesn't actually hold: e.g. avail == 2 with both sides
// truncated still forces 1 row + 2 hints == 3 lines. avail 1 or 2 is
// unreachable in practice, though — View() refuses to render below
// minFrameHeight (14), which guarantees avail >= 4 even with the demo
// banner's extra chrome line — so the shortfall never surfaces.)
//
// This exists because a live first-load chain is unfiltered (commonly
// 100+ rows): rendering from the top would bury the at-the-money region
// and let the keyboard selection wander below the visible window.
func chainViewport(total, selected, avail int) (start, end int) {
	if avail <= 0 || total <= 0 {
		return 0, 0
	}
	if total <= avail {
		return 0, total
	}
	selected = clampIndex(selected, total)

	// The list is longer than the pane, so at least one hint line shows.
	// Size the row window assuming both hints (avail-2 rows) and center it
	// on the selection; if that window runs off either end, clamp flush to
	// that end — the hint on that side disappears, freeing its line for
	// one more row (avail-1).
	rows := max(avail-2, 1)
	start = selected - rows/2
	if start <= 0 {
		return 0, max(avail-1, 1)
	}
	if start+rows >= total {
		rows = max(avail-1, 1)
		return max0(total - rows), total
	}
	return start, start + rows
}

// pinnedLines renders the pinboard strip: a "── PINNED ──…" separator
// (headed, like the chain table, in headerRowStyle) followed by one line
// listing every pinned contract's OCC symbol, mid price, and delta — or
// the "(empty — space pins)" hint when nothing is pinned yet.
func (m model) pinnedLines(width int) []styledLine {
	label := " ── PINNED "
	sep := label + strings.Repeat("─", max0(width-len([]rune(label))))
	lines := []styledLine{{text: truncate(sep, width), style: headerRowStyle}}

	if len(m.pinned) == 0 {
		return append(lines, plainLine("  (empty — space pins)"))
	}

	entries := make([]string, len(m.pinned))
	for i, occ := range m.pinned {
		q, ok := m.pinData[occ]
		entries[i] = pinnedEntry(occ, q, ok)
	}
	return append(lines, plainLine(truncate("  "+strings.Join(entries, "   "), width)))
}

// pinnedEntry formats one pinboard entry: "OCC  mid  Δdelta", or "OCC  —"
// when no quote has arrived for it yet (freshly pinned, before the next
// fetchPinned batch resolves it).
func pinnedEntry(occ string, q options.OptionQuote, ok bool) string {
	if !ok {
		return occ + "  —"
	}
	return fmt.Sprintf("%s  %.2f  Δ%.2f", occ, q.Mid, q.Delta)
}

// --- chain table columns ---

type colAlign int

const (
	alignLeft colAlign = iota
	alignRight
)

// chainColumn describes one column of the chain table: its header label,
// fixed display width, alignment, and how to render one row's cell.
type chainColumn struct {
	header string
	width  int
	align  colAlign
	format func(options.OptionQuote) string
}

// marketColumns is the default column set: strike+side, then the market
// data a trader watches most (bid/ask/mid/last/volume/open interest/IV).
func marketColumns() []chainColumn {
	return []chainColumn{
		{"STRIKE", 8, alignLeft, formatStrikeSide},
		{"BID", 6, alignRight, func(o options.OptionQuote) string { return fmt.Sprintf("%.2f", o.Bid) }},
		{"ASK", 6, alignRight, func(o options.OptionQuote) string { return fmt.Sprintf("%.2f", o.Ask) }},
		{"MID", 6, alignRight, func(o options.OptionQuote) string { return fmt.Sprintf("%.2f", o.Mid) }},
		{"LAST", 6, alignRight, func(o options.OptionQuote) string { return fmt.Sprintf("%.2f", o.Last) }},
		{"VOL", 7, alignRight, func(o options.OptionQuote) string { return comma(o.Volume) }},
		{"OI", 7, alignRight, func(o options.OptionQuote) string { return comma(o.OpenInterest) }},
		{"IV", 5, alignRight, func(o options.OptionQuote) string { return fmt.Sprintf("%.2f", o.IV) }},
	}
}

// greekColumns is the 'g'-toggled column set: strike+side, then the
// Greeks and IV.
func greekColumns() []chainColumn {
	return []chainColumn{
		{"STRIKE", 8, alignLeft, formatStrikeSide},
		{"Δ", 6, alignRight, func(o options.OptionQuote) string { return fmt.Sprintf("%.2f", o.Delta) }},
		{"Γ", 6, alignRight, func(o options.OptionQuote) string { return fmt.Sprintf("%.4f", o.Gamma) }},
		{"Θ", 6, alignRight, func(o options.OptionQuote) string { return fmt.Sprintf("%.2f", o.Theta) }},
		{"V", 6, alignRight, func(o options.OptionQuote) string { return fmt.Sprintf("%.2f", o.Vega) }},
		{"IV", 5, alignRight, func(o options.OptionQuote) string { return fmt.Sprintf("%.2f", o.IV) }},
	}
}

// formatColumns renders cells (one per column, already formatted by the
// caller — the column's own format func, or the header label) into a
// single space-joined row, aligned per column.
func formatColumns(cols []chainColumn, cells []string) string {
	parts := make([]string, len(cols))
	for i, c := range cols {
		if c.align == alignLeft {
			parts[i] = padRight(cells[i], c.width)
		} else {
			parts[i] = padLeft(cells[i], c.width)
		}
	}
	return strings.Join(parts, " ")
}

// formatStrikeSide renders a contract's strike and side as one cell, e.g.
// "233 C" or "232.5 P" — matching the design mock's combined STRIKE
// column.
func formatStrikeSide(o options.OptionQuote) string {
	side := "C"
	if o.Type == options.Put {
		side = "P"
	}
	return formatStrike(o.Strike) + " " + side
}

// rowMarker renders a chain row's three-character marker column: see
// chainTableLines for what each position means.
func rowMarker(selected, atm, itm bool) string {
	c1, c2, c3 := " ", " ", " "
	if selected {
		c1 = "▶"
	}
	if atm {
		c2 = "A"
	}
	if itm {
		c3 = "•"
	}
	return c1 + c2 + c3
}

// --- modals: contract detail (Enter), support info (E) ---

// detailModalLines renders the contract detail modal: the OCC symbol,
// bid/ask with sizes, mid/last, the Greeks and IV, intrinsic/extrinsic
// value, DTE, and the contract's last-updated time.
func (m model) detailModalLines() []string {
	q := m.detail
	if q == nil {
		return []string{m.contentLine(" (no contract selected)", plainStyle)}
	}
	return []string{
		m.contentLine(" "+q.OptionSymbol, modalTitleStyle),
		m.contentLine("", plainStyle),
		m.contentLine(fmt.Sprintf(" Bid  %.2f x %d   Ask  %.2f x %d   Mid  %.2f   Last %.2f",
			q.Bid, q.BidSize, q.Ask, q.AskSize, q.Mid, q.Last), plainStyle),
		m.contentLine(fmt.Sprintf(" Delta %.2f   Gamma %.4f   Theta %.2f   Vega %.2f   IV %.2f",
			q.Delta, q.Gamma, q.Theta, q.Vega, q.IV), plainStyle),
		m.contentLine(fmt.Sprintf(" Intrinsic %.2f   Extrinsic %.2f   DTE %d   Updated %s",
			q.IntrinsicValue, q.ExtrinsicValue, q.DTE, q.Updated.Format("2006-01-02 15:04")), plainStyle),
		m.contentLine("", plainStyle),
		m.contentLine(" [esc] close", keyHintStyle),
	}
}

// supportModalLines renders the support-info modal: lastErr.SupportInfo()
// verbatim when lastErr is a marketdata.Error (via errors.As against the
// alias), or the plain error string for any other error, or "(no error)"
// when there is none.
func (m model) supportModalLines() []string {
	var body string
	switch {
	case m.lastErr == nil:
		body = "(no error)"
	default:
		var apiErr marketdata.Error
		if errors.As(m.lastErr, &apiErr) {
			body = apiErr.SupportInfo()
		} else {
			body = m.lastErr.Error()
		}
	}

	lines := make([]string, 0, strings.Count(body, "\n")+3)
	for _, l := range strings.Split(body, "\n") {
		lines = append(lines, m.contentLine(" "+l, plainStyle))
	}
	lines = append(lines, m.contentLine("", plainStyle))
	lines = append(lines, m.contentLine(" [esc] close", keyHintStyle))
	return lines
}

// --- formatting primitives ---

// padRight pads s with spaces to width (measured by lipgloss.Width, so it
// is correct whether or not s carries ANSI styling already). It never
// truncates: if s is already at or beyond width, it is returned unchanged
// — slicing possibly-styled text is unsafe (it can cut an escape sequence
// in half), so callers that need truncation call truncate first, on text
// they know is plain.
func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// padLeft is padRight's mirror, for right-aligned cells.
func padLeft(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", width-w) + s
}

// truncate shortens plain (unstyled) text to at most width runes, adding
// an ellipsis when it cuts anything off. Never call this on text that may
// already carry ANSI escape codes (e.g. a textinput.View()) — rune-slicing
// such text can corrupt an escape sequence.
func truncate(s string, width int) string {
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width <= 1 {
		return string(r[:max0(width)])
	}
	return string(r[:width-1]) + "…"
}

// formatStrike renders a strike price without a spurious trailing ".00":
// 225 -> "225", 232.5 -> "232.5". Strikes are rounded to cents first so a
// float64 value like 233.09999999999999 renders as "233.1", not a long
// run of nines.
func formatStrike(f float64) string {
	rounded := float64(int64(f*100+0.5)) / 100
	if f < 0 {
		rounded = float64(int64(f*100-0.5)) / 100
	}
	return strconv.FormatFloat(rounded, 'f', -1, 64)
}

// comma renders n with thousands separators: 12410 -> "12,410".
func comma(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// commaInt is comma for an int (RateLimitMeta's Limit/Remaining fields).
func commaInt(n int) string {
	return comma(int64(n))
}

// max0 clamps n to be at least 0, guarding strings.Repeat and slice
// bounds against negative widths (a too-small window/terminal) without a
// panic.
func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
