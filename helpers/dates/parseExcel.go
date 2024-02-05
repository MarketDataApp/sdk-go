package dates

import (
	"math"
	"time"
)

// float64DateParser attempts to parse an Excel timestamp and determine its precision.
// The function accepts a float64 and a timezone as input.
func float64DateParser(excelTimestampFloat float64, tz *time.Location) (time.Time, string, error) {
	// Use the FromExcelTime helper function to convert the Excel timestamp to a time.Time
	parsedDate, err := FromExcelTime(excelTimestampFloat, tz)
	if err != nil {
		return time.Time{}, "", err
	}

	// Calculate the number of days and the fraction of a day
	_, fraction := math.Modf(excelTimestampFloat)

	// Determine the smallest unit of precision based on the fraction.
	precision := DayPrecision // Default precision if there's no fractional part.
	if fraction > 0 {
		// Determine the precision of the timestamp.
		if fraction*math.Pow(10, 9) == math.Floor(fraction*math.Pow(10, 9)) {
			precision = SecondPrecision
		} else if fraction*math.Pow(10, 6) == math.Floor(fraction*math.Pow(10, 6)) {
			precision = MillisecondPrecision
		} else if fraction*math.Pow(10, 3) == math.Floor(fraction*math.Pow(10, 3)) {
			precision = MicrosecondPrecision
		} else {
			precision = NanosecondPrecision
		}
	}

	return parsedDate, precision, nil
}

// FromExcelTime converts Excel time, which is the number of days since 1900-01-01, to a time.Time.
// The function takes the Excel time as a float64 and an optional timezone. If no timezone is provided, it uses the default timezone.
func FromExcelTime(excelTime float64, tz *time.Location) (time.Time, error) {
	days, fracDays := math.Modf(excelTime)

	duration := time.Duration(fracDays * 24 * float64(time.Hour))
	date := excelEpoch.AddDate(0, 0, int(days)).Add(duration)

	// Get the offset from UTC
	offset := getTimezoneOffset(date, tz)

	// Adjust the date by the offset
	date = utcToTZ(date, offset)

	// Set the timezone of the Time object
	date = date.In(tz)

	return date, nil
}

// ToExcelTime converts a time.Time to Excel time, which is the number of days since 1900-01-01.
// The function returns the Excel time as a float64.
// The function now takes a timezone as input and adjusts the Excel timestamp based on the timezone of the time.Time object.
func ToExcelTime(t time.Time, tz *time.Location) float64 {
	// Adjust the time by the timezone offset
	offset := getTimezoneOffset(t, tz)
	t = tzToUTC(t, offset)

	durationSinceExcelEpoch := t.Sub(excelEpoch)
	excelTime := float64(durationSinceExcelEpoch.Hours() / 24)

	return excelTime
}

func getTimezoneOffset(t time.Time, tz *time.Location) time.Duration {
	// Get the time in the specified timezone
	t = t.In(tz)

	// Get the name of the time zone and its offset from UTC in seconds
	_, offset := t.Zone()

	// Convert the offset to a time.Duration and return it
	return time.Duration(offset) * time.Second
}

func utcToTZ(utcTime time.Time, offset time.Duration) time.Time {
	// Adjust the time by the offset
	adjustedTime := utcTime.Add(-offset)

	return adjustedTime
}

func tzToUTC(utcTime time.Time, offset time.Duration) time.Time {
	// Adjust the time by the offset
	adjustedTime := utcTime.Add(offset)

	return adjustedTime
}
