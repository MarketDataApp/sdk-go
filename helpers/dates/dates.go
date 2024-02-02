// Package dates provides utilities for parsing, formatting, and manipulating dates and times.
// It includes functions to parse dates with or without time zones, handle date ranges,
// and convert dates to various formats. It also defines constants for different time precisions
// and initializes default time zones and date formats.
package dates

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TimeInterval represents a duration object
type TimeInterval struct {
	Duration      time.Duration
	IntervalType  string
	IntervalValue int
}

func (ti *TimeInterval) SetInterval(interval string) error {
	// Check if interval matches any of the regular expressions
	for _, regexType := range regexes {
		if matches := regexType.regex.FindStringSubmatch(interval); matches != nil {
			intervalValue, err := strconv.Atoi(matches[1]) // Get the numeric part of the interval
			if err != nil {
				return fmt.Errorf("invalid interval value: %v", err)
			}

			// Set the IntervalValue field
			ti.IntervalValue = intervalValue

			switch regexType.timeUnit {
			case "second":
				ti.Duration = time.Duration(intervalValue) * time.Second
			case "minute":
				ti.Duration = time.Duration(intervalValue) * time.Minute
			case "hour":
				ti.Duration = time.Duration(intervalValue) * time.Hour
			case "day":
				if matches[1] == "" {
					ti.Duration = 24 * time.Hour
				} else {
					intervalValue, err := strconv.Atoi(matches[1])
					if err != nil {
						return fmt.Errorf("invalid interval value: %v", err)
					}
					ti.Duration = time.Duration(intervalValue) * 24 * time.Hour
				}
			case "week":
				ti.Duration = time.Duration(intervalValue) * 7 * 24 * time.Hour
			case "month":
				ti.Duration = time.Duration(intervalValue) * 30 * 24 * time.Hour // Approximation
			case "year":
				ti.Duration = time.Duration(intervalValue) * 365 * 24 * time.Hour // Approximation
			}
			ti.IntervalType = regexType.timeUnit
			return nil
		}
	}

	// If no match found, return an error
	return fmt.Errorf("invalid interval parameter: %s", interval)
}

// DateRange represents a range of dates with a start and end date.
type DateRange struct {
	StartDate time.Time
	EndDate   time.Time
}

// UnixTimestamps returns the start and end dates as Unix timestamps.
func (dr *DateRange) UnixTimestamps() (int64, int64) {
	return dr.StartDate.Unix(), dr.EndDate.Unix()
}

func (dr *DateRange) StopwatchStart() {
	dr.StartDate = time.Now()
}

func (dr *DateRange) StopwatchEnd() {
	dr.EndDate = time.Now()
}

func StopwatchStart() *DateRange {
	return &DateRange{
		StartDate: time.Now(),
	}
}

// SetDates sets the start and end dates of the DateRange.
// It accepts two parameters for the start and end dates, which can be of type time.Time or string.
// If the dates are strings, they are parsed using the ParseDateInput function.
// An optional time zone can be provided as the third parameter. If not provided, the default time zone is used.
// The function returns an error if the start or end date is invalid or if they cannot be parsed.
func (dr *DateRange) SetDates(startDate, endDate interface{}, defaultTZ ...*time.Location) error {
	var start, end time.Time
	var err error
	var tz *time.Location

	if len(defaultTZ) > 0 {
		tz = defaultTZ[0]
	} else {
		tz = DefaultTZ
	}

	if startDate != nil {
		switch v := startDate.(type) {
		case time.Time:
			start = v
		default:
			start, _, err = ParseDateInput(startDate, tz)
			if err != nil {
				return fmt.Errorf("invalid start date: %v", err)
			}
		}
	} else {
		return fmt.Errorf("start date cannot be nil")
	}

	if endDate != nil {
		switch v := endDate.(type) {
		case time.Time:
			end = v
		default:
			end, _, err = ParseDateInput(endDate, tz)
			if err != nil {
				return fmt.Errorf("invalid end date: %v", err)
			}
		}
	} else {
		return fmt.Errorf("end date cannot be nil")
	}

	dr.StartDate = start
	dr.EndDate = end

	return nil
}

// DurationInMs returns the duration of the DateRange in milliseconds.
// It throws an error if either the start or end date is null.
func (dr *DateRange) DurationInMs() (int, error) {
	if dr.StartDate.IsZero() || dr.EndDate.IsZero() {
		return 0, fmt.Errorf("start date and end date cannot be nil")
	}

	duration := dr.EndDate.Sub(dr.StartDate)
	return int(duration.Milliseconds()), nil
}

// Duration returns the duration of the DateRange.
func (dr *DateRange) Duration() time.Duration {
	return dr.EndDate.Sub(dr.StartDate)
}

// ValidateTimestamps checks if the provided timestamps are contained within the current DateRange.
// It returns two slices of timestamps: the first one contains valid timestamps, the second one contains invalid timestamps.
func (dr *DateRange) ValidateTimestamps(timestamps ...int64) (validTimestamps, invalidTimestamps []int64) {
	for _, ts := range timestamps {
		t, _, err := ParseDateInput(ts)
		if err != nil {
			invalidTimestamps = append(invalidTimestamps, ts)
			continue
		}
		if dr.Contains(t) {
			validTimestamps = append(validTimestamps, ts)
		} else {
			invalidTimestamps = append(invalidTimestamps, ts)
		}
	}
	return validTimestamps, invalidTimestamps
}


// GenerateDateKeys generates a list of date keys for a DateRange based on the provided keyType.
// The keyType can be "days", "weeks", "months", or "years".
// The function uses the StartDate and EndDate of the DateRange to generate the keys.
// If the keyType is not recognized, the function returns an empty slice.
// The function returns a slice of strings, each representing a date key.
func (dr *DateRange) GenerateDateKeys(keyType string) ([]string, error) {
	return GenerateDateKeys(dr.StartDate, dr.EndDate, keyType)
}


// Contains checks if the provided time or DateRange (the argument) is wholly contained within the DateRange on which the method is called (the reference DateRange).
func (dr *DateRange) Contains(t interface{}) bool {
	if t == nil {
		return false
	}

	var timeToCheckStart, timeToCheckEnd time.Time

	switch v := t.(type) {
	case time.Time:
		timeToCheckStart = v
		timeToCheckEnd = v
	case DateRange:
		timeToCheckStart = v.StartDate
		timeToCheckEnd = v.EndDate
	default:
		var err error
		timeToCheckStart, _, err = ParseDateInput(v)
		if err != nil {
			return false
		}
		timeToCheckEnd = timeToCheckStart
	}

	return !dr.StartDate.After(timeToCheckStart) && !dr.EndDate.Before(timeToCheckEnd)
}

// DoesNotContain checks if the provided time or DateRange (the argument) is entirely outside the DateRange on which the method is called (the reference DateRange).
func (dr *DateRange) DoesNotContain(t interface{}) bool {
	if t == nil {
		return false
	}

	var timeToCheckStart, timeToCheckEnd time.Time

	switch v := t.(type) {
	case time.Time:
		timeToCheckStart = v
		timeToCheckEnd = v
	case DateRange:
		timeToCheckStart = v.StartDate
		timeToCheckEnd = v.EndDate
	default:
		var err error
		timeToCheckStart, _, err = ParseDateInput(v)
		if err != nil {
			return false
		}
		timeToCheckEnd = timeToCheckStart
	}

	return dr.StartDate.After(timeToCheckEnd) || dr.EndDate.Before(timeToCheckStart)
}

// PartiallyContains checks if the provided DateRange (the argument) partially intersects with the DateRange on which the method is called (the reference DateRange).
func (dr *DateRange) PartiallyContains(other DateRange) bool {
	// Check if the times are entirely within or entirely without one another
	if dr.Contains(other) || dr.DoesNotContain(other) {
		return false
	}

	// If neither of the above conditions is met, the times partially intersect
	return true
}

// NewDateRange is a function that creates a new DateRange instance.
// It accepts two parameters for the start and end dates, which can be of type time.Time or string.
// An optional time zone can be provided as the third parameter. If not provided, the default time zone is used.
// The function returns a pointer to the new DateRange instance and any errors that occurred during the creation.
func NewDateRange(startDate, endDate interface{}, defaultTZ ...*time.Location) (*DateRange, error) {
	dr := &DateRange{}
	err := dr.SetDates(startDate, endDate, defaultTZ...)
	if err != nil {
		return nil, err
	}
	return dr, nil
}

// SetFromDateKey sets the DateRange from a date key.
// The date key can be in the format "YYYY-Www" for weekly, "YYYY-MM-DD" for daily, "YYYY-MM" for monthly, or "YYYY" for yearly.
// The function uses the provided location to set the timezone for the start and end dates in the DateRange.
// If the date key is not in the correct format or contains invalid date components, the function returns an error.
func (dr *DateRange) SetFromDateKey(key string, loc *time.Location) error {
	var err error
	if strings.Contains(key, "-W") {
		*dr, err = FromWeeklyDateKey(key, loc)
	} else if strings.Count(key, "-") == 2 {
		*dr, err = FromDailyDateKey(key, loc)
	} else if strings.Count(key, "-") == 1 {
		*dr, err = FromMonthlyDateKey(key, loc)
	} else {
		*dr, err = FromYearlyDateKey(key, loc)
	}
	return err
}

func (dr DateRange) String() string {
	return fmt.Sprintf("StartDate: RFC3339: %s\nEndDate: RFC3339: %s\n",
		dr.StartDate.Format(time.RFC3339),
		dr.EndDate.Format(time.RFC3339))
}

// IsEarlierThan is a method that takes a parameter of type interface.
// It converts the parameter to a Time when necessary and compares if the start date of the value is earlier than the start date of our reference DateRange.
func (dr *DateRange) IsEarlierThan(date interface{}) (bool, error) {
	var compareDate time.Time
	switch date := date.(type) {
	case int, int64, string, float32, float64:
		parsedDate, _, err := ParseDateInput(date)
		if err != nil {
			return false, err
		}
		compareDate = parsedDate
	case time.Time:
		compareDate = date
	case *DateRange:
		compareDate = date.StartDate
	default:
		return false, fmt.Errorf("invalid type for date: IsEarlierThan")
	}
	return dr.StartDate.Before(compareDate), nil
}

// IsLaterThan is a method that takes a parameter of type interface.
// It converts the parameter to a Time when necessary and compares if the end date of the DateRange is later than the end date of the paramter passed.
func (dr *DateRange) IsLaterThan(date interface{}) (bool, error) {
	var compareDate time.Time
	switch date := date.(type) {
	case int, int64, string, float32, float64:
		parsedDate, _, err := ParseDateInput(date)
		if err != nil {
			return false, err
		}
		compareDate = parsedDate
	case time.Time:
		compareDate = date
	case *DateRange:
		compareDate = date.EndDate
	default:
		return false, fmt.Errorf("invalid type for date: IsLaterThan")
	}
	return dr.EndDate.After(compareDate), nil
}

// GetDateRange is a function that takes in a timezone, dateInput, fromInput, and toInput as strings.
// It returns a DateRange and an error.
// The function converts the date inputs to a DateRange,
// and returns the DateRange and any errors that occurred during the conversion.
func GetDateRange(defaultTZ *time.Location, dateInput, fromInput, toInput string) (DateRange, error) {
	var dr DateRange

	// If a single date is provided, convert it to a DateRange
	if dateInput != "" {
		singleDate, _, err := ParseDateInput(dateInput, defaultTZ)
		if err != nil {
			return dr, err
		}
		// Set the start and end dates of the DateRange to the single date
		dr.StartDate = singleDate
		dr.EndDate = time.Date(singleDate.Year(), singleDate.Month(), singleDate.Day(), 23, 59, 59, 999999999, defaultTZ)
	} else if fromInput != "" && toInput != "" {
		// If a date range is provided, convert the start and end dates to a DateRange
		fromDate, _, err := ParseDateInput(fromInput, defaultTZ)
		if err != nil {
			return dr, err
		}
		toDate, _, err := ParseDateInput(toInput, defaultTZ)
		if err != nil {
			return dr, err
		}
		// Set the start and end dates of the DateRange to the provided range
		dr.StartDate = fromDate
		dr.EndDate = toDate
	} else {
		// If no valid date or date range is provided, return an error
		return dr, fmt.Errorf("invalid date input provided")
	}

	// Return the DateRange and any errors
	return dr, nil
}

// CombineDateRanges is a function that takes multiple DateRange objects or a slice of DateRange objects
// and uses the Earliest and Latest functions to generate a new DateRange object that covers the entire time range of the original objects.
func CombineDateRanges(dates ...interface{}) (DateRange, error) {
	var combined DateRange

	// Check if the first argument is a slice of DateRange
	if len(dates) == 1 {
		if dateRanges, ok := dates[0].([]DateRange); ok {
			// Convert the slice of DateRange to a slice of interface{}
			dates = make([]interface{}, len(dateRanges))
			for i, dr := range dateRanges {
				dates[i] = dr
			}
		}
	}

	earliestDate, err1 := Earliest(dates...)
	latestDate, err2 := Latest(dates...)
	if err1 != nil || err2 != nil {
		return combined, fmt.Errorf("error calculating date ranges: %v, %v", err1, err2)
	}

	combined.StartDate = earliestDate
	combined.EndDate = latestDate

	return combined, nil
}

func Earliest(dates ...interface{}) (time.Time, error) {
	var earliest time.Time

	// Check if the first argument is a slice of DateRange or a slice of int64
	if len(dates) == 1 {
		switch v := dates[0].(type) {
		case []DateRange:
			// Convert the slice of DateRange to a slice of interface{}
			dates = make([]interface{}, len(v))
			for i, dr := range v {
				dates[i] = dr
			}
		case []int64:
			// Convert the slice of int64 to a slice of time.Time
			dates = make([]interface{}, len(v))
			for i, timestamp := range v {
				dates[i] = time.Unix(timestamp, 0)
			}
		}
	}

	for _, date := range dates {
		var t time.Time
		var err error
		switch v := date.(type) {
		case time.Time:
			t = v
		case DateRange:
			t = v.StartDate
		default:
			t, _, err = ParseDateInput(v)
			if err != nil {
				return time.Time{}, err
			}
		}
		if earliest.IsZero() || t.Before(earliest) {
			earliest = t
		}
	}
	if earliest.IsZero() {
		return time.Time{}, errors.New("no valid dates provided")
	}
	return earliest, nil
}

func Latest(dates ...interface{}) (time.Time, error) {
	var latest time.Time

	// Check if the first argument is a slice of DateRange or a slice of int64
	if len(dates) == 1 {
		switch v := dates[0].(type) {
		case []DateRange:
			// Convert the slice of DateRange to a slice of interface{}
			dates = make([]interface{}, len(v))
			for i, dr := range v {
				dates[i] = dr
			}
		case []int64:
			// Convert the slice of int64 to a slice of time.Time
			dates = make([]interface{}, len(v))
			for i, timestamp := range v {
				dates[i] = time.Unix(timestamp, 0)
			}
		}
	}

	for _, date := range dates {
		var t time.Time
		var err error
		switch v := date.(type) {
		case time.Time:
			t = v
		case DateRange:
			t = v.EndDate
		default:
			t, _, err = ParseDateInput(v)
			if err != nil {
				return time.Time{}, err
			}
		}
		if latest.IsZero() || t.After(latest) {
			latest = t
		}
	}
	if latest.IsZero() {
		return time.Time{}, errors.New("no valid dates provided")
	}
	return latest, nil
}
