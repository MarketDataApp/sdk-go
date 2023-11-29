package dates

import (
	"time"
	"os"
	"regexp"
)

// Constants for different time precisions
const (
	YearPrecision        = "year"        // Represents year precision
	MonthPrecision       = "month"       // Represents month precision
	DayPrecision         = "day"         // Represents day precision
	HourPrecision        = "hour"        // Represents hour precision
	MinutePrecision      = "minute"      // Represents minute precision
	SecondPrecision      = "second"      // Represents second precision
	MillisecondPrecision = "millisecond" // Represents millisecond precision
	MicrosecondPrecision = "microsecond" // Represents microsecond precision
	NanosecondPrecision  = "nanosecond"  // Represents nanosecond precision
)

// excelEpoch represents the start date for Excel's epoch which is December 30, 1899
var excelEpoch = time.Date(1899, time.December, 30, 0, 0, 0, 0, time.UTC)

// timeFormatsNoTZ is a slice of TimeFormats that stores the date formats without timezone.
var timeFormatsNoTZ []TimeFormats

// timeFormatsWithTZ is a slice of TimeFormats that stores the date formats with timezone.
var timeFormatsWithTZ []TimeFormats

// DefaultTZ is a pointer to a time.Location that stores the default timezone.
var DefaultTZ *time.Location

// RegexType is used for storing a regular expression and the timeUnit it parses to.
type RegexType struct {
	regex    *regexp.Regexp // The regular expression to match a time string
	timeUnit string			// The time unit that the regular expression represents
}

// regexes is a slice of RegexType structs that store regular expressions and their corresponding time units.
var regexes []RegexType

// setRegexes initializes the regexes slice with regular expressions and their corresponding time units.
func setRegexes() {
	regexes = []RegexType{
		{regexp.MustCompile(`^(\d+)\s*$`), "minute"}, // Matches a string of digits followed by optional whitespace
		{regexp.MustCompile(`^(\d+)\s*(min|minutes)?$`), "minute"}, // Matches a string of digits followed by optional "min" or "minutes"
		{regexp.MustCompile(`^(\d+)\s*s$`), "second"}, // Matches a string of digits followed by a "s"
		{regexp.MustCompile(`(?i)^(\d+)\s*(sec|second|seconds)?$`), "second"}, // Matches a string of digits followed by optional "sec", "second", or "seconds"
		{regexp.MustCompile(`^(\d+)\s*H$`), "hour"}, // Matches a string of digits followed by a "H"
		{regexp.MustCompile(`(?i)^(\d+)\s*hour(s)?$`), "hour"}, // Matches a string of digits followed by optional "hour" or "hours"
		{regexp.MustCompile(`^(?i)d$`), "day"}, // Matches a "d" or "D"
		{regexp.MustCompile(`(?i)^(\d+)\s*D$`), "day"}, // Matches a string of digits followed by a "D"
		{regexp.MustCompile(`^(\d+)\s*D$`), "day"}, // Matches a string of digits followed by a "D"
		{regexp.MustCompile(`(?i)^(\d+)\s*day(s)?$`), "day"}, // Matches a string of digits followed by optional "day" or "days"
		{regexp.MustCompile(`^(\d+)\s*W$`), "week"}, // Matches a string of digits followed by a "W"
		{regexp.MustCompile(`(?i)^(\d+)\s*week(s)?$`), "week"}, // Matches a string of digits followed by optional "week" or "weeks"
		{regexp.MustCompile(`(?i)^(\d+)\s*M$`), "month"}, // Matches a string of digits followed by a "M"
		{regexp.MustCompile(`(?i)^(\d+)\s*month(s)?$`), "month"}, // Matches a string of digits followed by optional "month" or "months"
		{regexp.MustCompile(`(?i)^(\d+)\s*Y$`), "year"}, // Matches a string of digits followed by a "Y"
		{regexp.MustCompile(`(?i)^(\d+)\s*year(s)?$`), "year"}, // Matches a string of digits followed by optional "year" or "years"
	}
}

// TimeFormats is a struct that holds information about a time format.
// Format is a string that represents the time format.
// Precision is a string that represents the precision of the time format.
// Timezone is a boolean that indicates whether the time format includes a timezone.
type TimeFormats struct {
	Format    string // The time format
	Precision string // The precision of the time format
	Timezone  bool   // Whether the time format includes a timezone
}

var timeFormats = []TimeFormats{
	// Dates without timezones are set to false
	{"3:04PM",                          "minute", false},
	{"Monday, January 2, 2006 3:04 PM", "minute", false},
	{"1/2/06",                          "day", false},
	{"1/2/2006",                        "day", false},
	{"01/02/2006",                      "day", false},
	{"2006-01-02",                      "day", false},
	{"2006-1-2",                        "day", false},
	{"January 2, 2006",                 "day", false},
	{"Jan 2, 2006",                     "day", false},
	{"Monday, Jan 2, 2006",             "day", false},
	{"02-Jan-2006",                     "day", false},
	{"02 January 2006",                 "day", false},
	{"01/02/2006 3:04 PM",              "minute", false},
	{"01/02/2006 15:04",                "minute", false},
	{"2006-01-02T15:04",                "minute", false},
	{"Mon Jan _2 15:04:05 2006",        "second", false},
	{"01/02/2006 3:04:05 PM",           "second", false},
	{"01/02/2006 15:04:05",             "second", false},
	{"2006-01-02 15:04:05",             "second", false},
	{"2006-01-02T15:04:05",             "second", false},
	{"20060102T150405",                 "second", false},
	{"Jan 2, 2006 15:04:05",            "second", false},
	{"January 2, 2006 15:04:05.999",    "milisecond", false},
	{"01/02/2006 3:04:05.999 PM",       "millisecond", false},
	{"01/02/2006 15:04:05.999",         "millisecond", false},
	{"2006-01-02 15:04:05.999",         "millisecond", false},
	{"2006-01-02T15:04:05.999",         "millisecond", false},
	{"01/02/2006 3:04:05.999999 PM",    "microsecond", false},
	{"01/02/2006 15:04:05.999999",      "microsecond", false},
	{"2006-01-02 15:04:05.999999",      "microsecond", false},
	{"2006-01-02T15:04:05.999999",      "microsecond", false},
	{"01/02/2006 3:04:05.999999999 PM", "nanosecond", false},
	{"01/02/2006 15:04:05.999999999",   "nanosecond", false},
	{"2006-01-02 15:04:05.999999999",   "nanosecond", false},
	{"2006-01-02T15:04:05.999999999",   "nanosecond", false},

	// Dates with timezones are set to true
	{"01/02/2006 3:04 PM MST",             "minute", true},
	{"2006-01-02T15:04Z0700",              "minute", true},
	{"20060102T1504Z0700",                 "minute", true},
	{"2006-01-02T15:04-07:00",             "minute", true},
	{"2006-01-02T15:04Z",                  "minute", true},
	{"Mon Jan _2 15:04 MST 2006",          "minute", true},     // time.UnixDate
	{"Mon Jan 02 15:04 -0700 2006",        "minute", true},     // time.RubyDate
	{"02 Jan 06 15:04 MST",                "minute", true},     // time.RFC822
	{"02 Jan 06 15:04 -0700",              "minute", true},     // time.RFC822Z
	{"Monday, 02-Jan-06 15:04 MST",        "minute", true},     // time.RFC850
	{"Mon, 02 Jan 2006 15:04 MST",         "minute", true},     // time.RFC1123
	{"Mon, 02 Jan 2006 15:04 -0700",       "minute", true},     // time.RFC1123Z
	{"2006-01-02T15:04Z-07:00",            "minute", true},     // time.RFC3339
	{"01/02/2006 3:04:05 MST",             "second", true},
	{"2006-01-02T15:04:05Z0700",           "second", true},
	{"20060102T150405Z0700",               "second", true},
	{"2006-01-02T15:04:05-07:00",          "second", true},
	{"2006-01-02T15:04:05Z",               "second", true},
	{"Mon Jan _2 15:04:05 MST 2006",       "second", true},     // time.UnixDate
	{"Mon Jan 02 15:04:05 -0700 2006",     "second", true},     // time.RubyDate
	{"02 Jan 06 15:04 MST",                "second", true},     // time.RFC822
	{"02 Jan 06 15:04 -0700",              "second", true},     // time.RFC822Z
	{"Monday, 02-Jan-06 15:04:05 MST",     "second", true},     // time.RFC850
	{"Mon, 02 Jan 2006 15:04:05 MST",      "second", true},     // time.RFC1123
	{"Mon, 02 Jan 2006 15:04:05 -0700",    "second", true},     // time.RFC1123Z
	{"2006-01-02T15:04:05Z-07:00",         "second", true},     // time.RFC3339
	{"2006-01-02T15:04:05.999999999Z07:00","nanosecond", true}, // time.RFC3339Nano
}

// filterTimeFormats is a function that filters the provided time formats based on the provided precision and timezone.
// The function takes in a slice of TimeFormats, a precision string, and an optional timezone boolean.
// It returns a slice of TimeFormats that match the provided precision and timezone.
// If no timezone is provided, it will return formats regardless of their timezone.
// If no precision is provided, it will return formats regardless of their precision.
func filterTimeFormats(formatStrings []TimeFormats, precision string, timezone ...bool) []TimeFormats {
	// Initialize an empty slice to store the filtered time formats
	var filteredTimeFormats []TimeFormats
	
	// Iterate over the provided time formats
	for _, timeFormats := range formatStrings {
		// If no timezone is provided or if the timezone matches the provided timezone
		if len(timezone) == 0 || timeFormats.Timezone == timezone[0] {
			// If no precision is provided or if the precision matches the provided precision
			if precision == "" || timeFormats.Precision == precision {
				// Append the time format to the filtered time formats
				filteredTimeFormats = append(filteredTimeFormats, timeFormats)
			}
		}
	}
	
	// Return the filtered time formats
	return filteredTimeFormats
}

// init is a special function in Go that is called when the package is initialized.
// This function initializes several package-level variables.
func init() {
	timeFormatsNoTZ = filterTimeFormats(timeFormats, "", false)
	timeFormatsWithTZ = filterTimeFormats(timeFormats, "", true)

	// Load the default timezone from an environment variable
	// If there is no environment variable, use "America/New_York"
	tz := os.Getenv("TZ")
	if tz == "" {
		tz = "America/New_York"
	}

	// Set the regular expressions for precision parsing
	setRegexes()
	
	// Load the timezone
	var err error
	DefaultTZ, err = time.LoadLocation(tz)
	if err != nil {
		// If there is an error loading the timezone, panic
		panic(err)
	}
}


