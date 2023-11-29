// Package dates provides helper functions for date and time parsing.
package dates

import (
	"time"
)

// precisionOrder is a map that assigns an order of precision to date and time components.
var precisionOrder = map[string]int{
	YearPrecision:        1,
	MonthPrecision:       2,
	DayPrecision:         3,
	HourPrecision:        4,
	MinutePrecision:      5,
	SecondPrecision:      6,
	MillisecondPrecision: 7,
	MicrosecondPrecision: 8,
	NanosecondPrecision:  9,
}

// dateRangeParser parses the start and end dates of a DateRange and returns the start time, precision, and any error that occurred.
func dateRangeParser(dr DateRange, tz *time.Location) (time.Time, string, error) {
    // Parse the StartDate of the DateRange
    parsedStartTime, startPrecision, err := timeDateParser(dr.StartDate, tz)
    if err != nil {
        return time.Time{}, "", err
    }

    // Parse the EndDate of the DateRange
    _, endPrecision, err := timeDateParser(dr.EndDate, tz)
    if err != nil {
        return time.Time{}, "", err
    }

    // Determine the smallest precision
    precision := startPrecision
    if precisionOrder[startPrecision] < precisionOrder[endPrecision] {
        precision = endPrecision
    }

    // Return the parsed start and end times, precision, and no error
    return parsedStartTime, precision, nil
}

// timeDateParser converts a time to a specified timezone and infers its precision.
func timeDateParser(t time.Time, tz *time.Location) (time.Time, string, error) {
    // Convert the time to the specified timezone
    t = t.In(tz)

    // Infer the precision
    precision, err := inferPrecision(t)
    if err != nil {
        return time.Time{}, "", err
    }

    // Return the time, precision, and no error
    return t, precision, nil
}

// inferPrecision infers the precision of a time by checking its components from smallest to largest.
func inferPrecision(t time.Time) (string, error) {
	if t.Nanosecond() != 0 {
		return NanosecondPrecision, nil
	} else if t.Second() != 0 {
		return SecondPrecision, nil
	} else if t.Minute() != 0 {
		return MinutePrecision, nil
	} else if t.Hour() != 0 {
		return HourPrecision, nil
	} else {
		// The time is at 00:00:00, so check the day, month, and year
		_, month, day := t.Date()
		if day != 1 {
			return DayPrecision, nil
		} else if month != time.January {
			return MonthPrecision, nil
		} else {
			return YearPrecision, nil
		}
	}
}