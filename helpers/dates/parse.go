package dates

import (
	"errors"
	"time"
)

// ToTime takes an interface and returns just the time.Time and an error
func ToTime(dateInput interface{}, tz ...*time.Location) (time.Time, error) {
	parsedDate, _, err := ParseDateInput(dateInput, tz...)
	return parsedDate, err
}

// ParseDateInput tries to parse the dateInput in various formats.
// It returns the parsed time, the precision, and any error encountered.
func ParseDateInput(dateInput interface{}, tz ...*time.Location) (time.Time, string, error) {
	var loc *time.Location

	// If the user has not given us a timezone, we use the default timezone
	if len(tz) > 0 {
		loc = tz[0]
	} else {
		loc = DefaultTZ
	}

	// If the user has provided us with a string input of a number, convert to int64 or float64, as appropriate.
	dateInput = convertStringToNumber(dateInput)

	// Use parsePrimitives to parse the dateInput
	parsedDate, precision, err := parsePrimitives(dateInput, loc)
	if err != nil {
		return time.Time{}, "", err
	}

	return parsedDate, precision, nil
}

// IdentifyDateInputPrimitiveType identifies the primitive type of the date input.
func parsePrimitives(dateInput interface{}, tz *time.Location) (time.Time, string, error) {
	switch v := dateInput.(type) {
	case int:
		return int64DateParser(int64(v), tz)
	case *int:
		if v == nil {
			return time.Time{}, "", errors.New("nil pointer passed for int")
		}
		return int64DateParser(int64(*v), tz)
	case int32:
		return parsePrimitives(int64(v), tz)
	case *int32:
		if v == nil {
			return time.Time{}, "", errors.New("nil pointer passed for int32")
		}
		return parsePrimitives(int64(*v), tz)
	case int64:
		return int64DateParser(v, tz)
	case *int64:
		if v == nil {
			return time.Time{}, "", errors.New("nil pointer passed for int64")
		}
		return int64DateParser(*v, tz)
	case string:
		return stringDateParser(v, tz)
	case *string:
		if v == nil {
			return time.Time{}, "", errors.New("nil pointer passed for string")
		}
		return stringDateParser(*v, tz)
	case float32:
		return parsePrimitives(float64(v), tz)
	case *float32:
		if v == nil {
			return time.Time{}, "", errors.New("nil pointer passed for float32")
		}
		return parsePrimitives(float64(*v), tz)
	case float64:
		return float64DateParser(v, tz)
	case *float64:
		if v == nil {
			return time.Time{}, "", errors.New("nil pointer passed for float64")
		}
		return float64DateParser(*v, tz)
	case bool, *bool:
		return time.Time{}, "", errors.New("boolean values cannot be parsed as dates")
	case time.Time:
		return timeDateParser(v, tz)
	case *time.Time:
		if v == nil {
			return time.Time{}, "", errors.New("nil pointer passed for time.Time")
		}
		return timeDateParser(*v, tz)
	case DateRange:
		return dateRangeParser(v, tz)
	case *DateRange:
		if v == nil {
			return time.Time{}, "", errors.New("nil pointer passed for DateRange")
		}
		return dateRangeParser(*v, tz)
	default:
		return time.Time{}, "", errors.New("value cannot be parsed as a date")
	}
}
