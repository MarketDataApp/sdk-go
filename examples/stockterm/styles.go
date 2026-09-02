// styles.go is the one place stockterm's views reach for color: every
// lipgloss style and the palette it draws from lives here, so a reader
// auditing "what can this app render in what color" never has to grep
// views.go. Golden frames are ANSI-stripped (tuitest.Frame), so none of
// this affects test outcomes — it only affects what a real terminal
// shows. Every style here is applied to text that has already been
// padded/aligned to its final width (see views.go's padRight/padLeft);
// wrapping a string in a lipgloss style only adds invisible escape
// sequences around it, so styling always happens last, after layout.
package main

import "github.com/charmbracelet/lipgloss"

// Palette. Adaptive colors pick a light- or dark-terminal variant
// automatically; the exact values are aesthetic choices, not contracts.
var (
	colorPositive = lipgloss.AdaptiveColor{Light: "#116329", Dark: "#3fb950"}
	colorNegative = lipgloss.AdaptiveColor{Light: "#b91c1c", Dark: "#f85149"}
	colorAccent   = lipgloss.AdaptiveColor{Light: "#0550ae", Dark: "#58a6ff"}
	colorMuted    = lipgloss.AdaptiveColor{Light: "#6e7781", Dark: "#8b949e"}
	colorWarning  = lipgloss.AdaptiveColor{Light: "#9a6700", Dark: "#d29922"}
	colorBorder   = lipgloss.AdaptiveColor{Light: "#57606a", Dark: "#6e7681"}
)

// Styles, one per semantic role. Names describe what they're for, not
// what they look like, so the palette above can change without renaming
// every call site.
var (
	// borderStyle colors box-drawing characters (borders, dividers).
	borderStyle = lipgloss.NewStyle().Foreground(colorBorder)

	// titleStyle colors header/divider title text (market status +
	// clock, "AAPL" in the detail divider, modal titles).
	titleStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)

	// tableHeadStyle colors the watchlist column header row.
	tableHeadStyle = lipgloss.NewStyle().Foreground(colorMuted).Bold(true)

	// positiveStyle and negativeStyle color signed numbers (CHG, CHG%)
	// by direction; see signStyle.
	positiveStyle = lipgloss.NewStyle().Foreground(colorPositive)
	negativeStyle = lipgloss.NewStyle().Foreground(colorNegative)

	// mutedStyle colors placeholders ("—") and secondary text (labels,
	// range captions).
	mutedStyle = lipgloss.NewStyle().Foreground(colorMuted)

	// markerStyle colors the "◀" selection marker.
	markerStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)

	// warningStyle colors the demo-mode banner and error/status text.
	warningStyle = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)

	// errorStyle colors the status line when lastErr is set.
	errorStyle = lipgloss.NewStyle().Foreground(colorNegative).Bold(true)
)

// signStyle returns positiveStyle, negativeStyle, or mutedStyle for v,
// so callers can color any signed quantity (price change, day
// performance) consistently with one call.
func signStyle(v float64) lipgloss.Style {
	switch {
	case v > 0:
		return positiveStyle
	case v < 0:
		return negativeStyle
	default:
		return mutedStyle
	}
}
