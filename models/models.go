// Package models defines the data structures used to represent the responses from the Market Data API.
// It includes models for stock quotes, options quotes, stock candles, market status, and more.
// Each model is designed to parse and validate the JSON responses from the API, providing a structured
// way to interact with market data. The package also includes methods for unpacking these responses
// into more usable Go data types, such as converting timestamps to time.Time objects.
package models

import (
	"time"

	"github.com/MarketDataApp/sdk-go/helpers/dates"
)

// PreferredTimezone represents the preferred timezone for time display.
var PreferredTimezone = "America/New_York"

// defaultTimezone is a fallback timezone used when the preferred timezone is not available.
var defaultTimezone *time.Location

// TimeFormat specifies the format for displaying full datetime information.
var TimeFormat = "2006-01-02 15:04:05 Z07:00"

// DateFormat specifies the format for displaying date-only information.
var DateFormat = "2006-01-02"

func init() {
	// Initialize defaultTimezone to UTC as a fallback
	defaultTimezone = time.UTC

	// Attempt to load the preferred timezone
	if loc, err := time.LoadLocation(PreferredTimezone); err == nil {
		defaultTimezone = loc // Use preferred timezone if available
	}
}

// formatTime formats a given time.Time object into a string representation based on a pre-defined timezone and format.
//
// Parameters:
//   - t: The time.Time object to be formatted.
//
// Returns:
//   - A string representing the formatted time in the pre-defined timezone and format, or 'nil' if the time is a zero value.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "nil"
	}
	tInLoc := t.In(defaultTimezone)
	// Use IsStartOfDay from helpers/dates to check if the time is at the start of the day
	if dates.IsStartOfDay(tInLoc, defaultTimezone) {
		return tInLoc.Format(DateFormat)
	}
	return tInLoc.Format(TimeFormat)
}
