package dates

import (
	"strings"
    "strconv"
	"time"
	"fmt"
)

// stringDateParser attempts to parse a date string without timezone, with timezone, and special cases.
// It returns the parsed time, the precision, and any error encountered.
func stringDateParser(dateStr string, tz *time.Location) (time.Time, string, error) {
    t, precision, err := tryParseAttemptNoTZ(dateStr, tz)
    if err == nil {
        return t, precision, nil
    }

    t, precision, err = tryParseAttemptWithTZ(dateStr)
    if err == nil {
        return t, precision, nil
    }

	parsedDate, precision, found := tryParseSpecialCases(dateStr)
    if found {
        return parsedDate, precision, nil
    }

    return time.Time{}, "", fmt.Errorf("unable to parse date string: %s", dateStr)
}

// tryParseSpecialCases tries to parse special date strings like "today", "yesterday", and "now".
// It returns the parsed time, the precision, and a boolean indicating if the parsing was successful.
func tryParseSpecialCases(dateStr string) (time.Time, string, bool) {
    switch strings.ToLower(dateStr) {
    case "today":
        return time.Now().Truncate(24 * time.Hour), "day", true
    case "yesterday":
        return time.Now().Truncate(24 * time.Hour).AddDate(0, 0, -1), "day", true
    case "now":
        return time.Now(), "second", true
    default:
        return time.Time{}, "", false
    }
}

// tryParseAttemptNoTZ tries to parse a date string without timezone.
// It returns the parsed time, the precision, and any error encountered.
func tryParseAttemptNoTZ(dateStr string, tz *time.Location) (time.Time, string, error) {
    var loc *time.Location
    if tz != nil {
        loc = tz
    } else {
        loc = DefaultTZ
    }

    for _, timeFormat := range timeFormatsNoTZ {
        var t time.Time
        var err error
        t, err = time.ParseInLocation(timeFormat.Format, dateStr, loc)
        if err == nil {
            return t, timeFormat.Precision, nil
        }
    }
    return time.Time{}, "", fmt.Errorf("unable to parse date string: %s", dateStr)
}

// tryParseAttemptWithTZ tries to parse a date string with timezone.
// It returns the parsed time, the precision, and any error encountered.
func tryParseAttemptWithTZ(dateInput string) (time.Time, string, error) {
    for _, timeFormat := range timeFormatsWithTZ {
        var t time.Time
        var err error
        t, err = time.Parse(timeFormat.Format, dateInput)
        if err == nil {
            return t, timeFormat.Precision, nil
        }
    }
    return time.Time{}, "", fmt.Errorf("unable to parse date string: %s", dateInput)
}

// convertStringToNumber attempts to convert a string input to a number.
// If the input is a string that can be parsed to an int64, it returns the int64.
// If the input is a string that can be parsed to a float64, it returns the float64.
// If the input is not a string or cannot be parsed to a number, it returns the original input.
func convertStringToNumber(dateInput interface{}) interface{} {
    switch v := dateInput.(type) {
    case string:
        if val, err := strconv.ParseInt(v, 10, 64); err == nil {
            return int64(val)
        } else if val, err := strconv.ParseFloat(v, 64); err == nil {
            return float64(val)
        }
    }
    return dateInput
}