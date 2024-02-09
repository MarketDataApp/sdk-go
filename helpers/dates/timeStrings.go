package dates

import "time"

// PreferredTimezone represents the preferred timezone for time display.
var PreferredTimeStringTimezone = "America/New_York"

// timeStringTimezone is the variable that stores the PreferredTimezone as a *time.Location.
var timeStringTimezone *time.Location

// TimeFormat specifies the format for displaying full datetime information.
var TimeStringTimeFormat = "2006-01-02 15:04:05 Z07:00"

// DateFormat specifies the format for displaying date-only information.
var TimeStringDateFormat = "2006-01-02"

func init() {
	// Initialize defaultTimezone to UTC as a fallback
	timeStringTimezone = time.UTC

	// Attempt to load the preferred timezone
	if loc, err := time.LoadLocation(PreferredTimeStringTimezone); err == nil {
		timeStringTimezone = loc // Use preferred timezone if available
	}
}

// TimeString converts a time.Time object to a string representation using a predefined timezone and format. 
// This method is primarily used in for string methods that need a standardized string representation of time.Time, 
// such as for logging, displaying to users, or when performing date-time comparisons in a specific timezone.
//
// # Parameters
//
//   - time.Time: The time object to be formatted.
//
// # Returns
//
//   - string: The formatted time as a string in the predefined timezone and format. Returns 'nil' if the time object is a zero value.
//
// # Notes
//
//   - This method utilizes the global PreferredTimezone variable, which can be set to the preferred timezone or falls back to UTC if the preferred timezone is not set.
func TimeString(t time.Time) string {
	if t.IsZero() {
		return "nil"
	}
	tInLoc := t.In(timeStringTimezone)
	// Use IsStartOfDay from helpers/dates to check if the time is at the start of the day
	if IsStartOfDay(tInLoc, timeStringTimezone) {
		return tInLoc.Format(TimeStringDateFormat)
	}
	return tInLoc.Format(TimeStringTimeFormat)
}
