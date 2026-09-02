// The lipgloss style palette for optionterm's View. Centralizing every
// style here (rather than scattering lipgloss.NewStyle() calls through
// views.go) keeps the color/emphasis choices in one place and mirrors the
// package layout the design doc calls for.
//
// Every style here wraps text that is already padded/truncated to its
// final column width by views.go — lipgloss.Style.Render just adds escape
// codes around already-fixed-width plain text, so applying a style never
// changes the visible width of a cell. That matters for golden tests:
// tuitest.Frame strips ANSI escape sequences before comparing, so styling
// choices here are invisible to the golden files and free to change
// without touching testdata. The plain-text structure (markers, column
// alignment, labels) is what the goldens actually pin down.
package main

import "github.com/charmbracelet/lipgloss"

var (
	// plainStyle is the explicit "no styling" style: used instead of a
	// bare lipgloss.Style{} zero value so every style in this file goes
	// through the same NewStyle() construction path.
	plainStyle = lipgloss.NewStyle()

	// borderStyle renders the frame's box-drawing lines (top/mid/bottom
	// borders and the sidebar/chain divider).
	borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// symbolStyle renders the whole top border line — the one that carries
	// the underlying symbol and last price — set apart (bold) from the
	// plain borderStyle used for the mid/bottom borders and the divider.
	symbolStyle = lipgloss.NewStyle().Bold(true)

	// demoBannerStyle renders the "DEMO MODE" line shown when the client
	// has no token and is serving fixed demo data.
	demoBannerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))

	// headerRowStyle renders the chain table's column header row.
	headerRowStyle = lipgloss.NewStyle().Bold(true).Underline(true)

	// selectedRowStyle renders the chain row under keyboard focus
	// (m.rowSelected).
	selectedRowStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))

	// atmRowStyle renders the at-the-money row (atmIndex) when it is not
	// also the selected row.
	atmRowStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))

	// itmDimStyle renders in-the-money rows that are neither selected nor
	// ATM: a subtle tint rather than the bold treatment reserved for
	// selection/ATM.
	itmDimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	// expSelectedStyle renders the selected entry (and its DTE line) in
	// the expirations sidebar.
	expSelectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))

	// scrollHintStyle renders the chain viewport's "▲ N more" / "▼ N more"
	// truncation-hint lines.
	scrollHintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	// creditsStyle renders the footer's credit meter.
	creditsStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	// keyHintStyle renders the footer's key-binding reference.
	keyHintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	// statusErrStyle renders the status line when lastErr is set.
	statusErrStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))

	// statusOKStyle renders the status line when there is no error (the
	// "[no error]" default or a transient statusNote).
	statusOKStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	// modalTitleStyle renders a modal's title text within the header bar.
	modalTitleStyle = lipgloss.NewStyle().Bold(true)

	// inputPromptStyle renders the "lookup"/"symbol" label in front of an
	// open text input.
	inputPromptStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
)

// rowStyle picks the style for a chain row given its selection/ATM/ITM
// state, in priority order: a selected row always wins (it is what the
// user's cursor is on right now), then ATM, then a plain ITM tint. A row
// that is none of these renders unstyled.
func rowStyle(selected, atm, itm bool) lipgloss.Style {
	switch {
	case selected:
		return selectedRowStyle
	case atm:
		return atmRowStyle
	case itm:
		return itmDimStyle
	default:
		return plainStyle
	}
}
