package dates

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// GenerateDateKeys generates a list of date keys between the provided start and end dates based on the provided keyType.
// The keyType can be "days", "weeks", "months", or "years".
// If the keyType is not recognized, the function returns an error.
// The function returns a slice of strings, each representing a date key, and an error.
func GenerateDateKeys(start, end time.Time, keyType string) ([]string, error) {
	var keys []string // Declare keys here

	switch keyType {
	case "days":
		keys = GenerateDailyDateKeys(start, end)
	case "weeks":
		keys = GenerateWeeklyDateKeys(start, end)
	case "months":
		keys = GenerateMonthlyDateKeys(start, end)
	case "years":
		keys = GenerateYearlyDateKeys(start, end)
	default:
		// If the keyType is not recognized, return an error
		return nil, fmt.Errorf("invalid keyType: %s", keyType)
	}

	return keys, nil
}

// GenerateDailyDateKeys generates a list of date keys for each day between the start and end dates.
// The date keys are in the format "YYYY-MM-DD".
// The function includes the end date in the list of keys.
// The start and end dates should be in the same timezone.
func GenerateDailyDateKeys(startDate, endDate time.Time) []string {
	keys := []string{}

	// Loop over each day between the start and end dates, inclusive.
	for d := startDate; d.Before(endDate) || d.Equal(endDate); d = d.AddDate(0, 0, 1) {
		year, month, day := d.Date()
		// Generate a date key for the day and add it to the list.
		keys = append(keys, fmt.Sprintf("%d-%02d-%02d", year, month, day))
	}

	// Return the list of date keys.
	return keys
}

// GenerateWeeklyDateKeys generates a list of date keys for each week between the start and end dates.
// The date keys are in the format "YYYY-Www", where YYYY is the year and ww is the week number.
// The function includes the end date in the list of keys.
// The start and end dates should be in the same timezone.
// The start date is adjusted to the start of its week (Sunday) and the end date is adjusted to the end of its week (Saturday).
// If the start date is in the first week of the year, it is set to January 1st.
// If the end date is in the last week of the year, it is set to December 31st.
func GenerateWeeklyDateKeys(startDate, endDate time.Time) []string {
	keys := []string{}

	start := startDate.Add(-time.Duration(startDate.Weekday()) * 24 * time.Hour)
	if start.Year() < startDate.Year() {
		start = time.Date(startDate.Year(), time.January, 1, 0, 0, 0, 0, startDate.Location())
	}

	end := endDate
	if endDate.Weekday() != time.Saturday {
		end = endDate.Add(time.Duration(6-int(endDate.Weekday())) * 24 * time.Hour)
	}
	if end.Year() > endDate.Year() {
		end = time.Date(endDate.Year(), time.December, 31, 23, 59, 59, 999999999, endDate.Location())
	}

	for d := start; !d.After(end); d = d.AddDate(0, 0, 7) {
		year, week := d.Year(), d.YearDay()/7+1
		keys = append(keys, fmt.Sprintf("%d-W%02d", year, week))
	}

	return keys
}

// GenerateMonthlyDateKeys generates a list of date keys for each month between the start and end dates.
// The date keys are in the format "YYYY-MM".
// The function includes the end date's month in the list of keys.
// The start and end dates should be in the same timezone.
func GenerateMonthlyDateKeys(startDate, endDate time.Time) []string {
	// Initialize an empty slice to hold the keys.
	keys := []string{}

	// Set the start date to the first day of the start month and the end date to the first day of the month following the end date.
	start := time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, startDate.Location())
	end := time.Date(endDate.Year(), endDate.Month()+1, 1, 0, 0, 0, 0, endDate.Location())

	// Loop over each month between the start and end dates, inclusive.
	for d := start; d.Before(end); d = d.AddDate(0, 1, 0) {
		// Extract the year and month from the current date.
		year, month, _ := d.Date()
		// Generate a date key for the month and add it to the list.
		keys = append(keys, fmt.Sprintf("%d-%02d", year, month))
	}

	// Return the list of date keys.
	return keys
}

// GenerateYearlyDateKeys generates a list of date keys for each year between the start and end dates.
// The date keys are in the format "YYYY", where YYYY is the year.
// The function includes the end date in the list of keys.
// The start and end dates should be in the same timezone.
// The start date is adjusted to the start of its year (January 1st) and the end date is adjusted to the start of the next year.
// The function returns a slice of strings, each representing a date key for a year.
func GenerateYearlyDateKeys(startDate, endDate time.Time) []string {
	// Initialize an empty slice to hold the date keys
	keys := []string{}

	// Adjust the start date to the start of its year and the end date to the start of the next year
	start := time.Date(startDate.Year(), 1, 1, 0, 0, 0, 0, startDate.Location())
	end := time.Date(endDate.Year()+1, 1, 1, 0, 0, 0, 0, endDate.Location())

	// Generate a date key for each year between the start and end dates
	for d := start; d.Before(end); d = d.AddDate(1, 0, 0) {
		year, _, _ := d.Date()
		keys = append(keys, fmt.Sprintf("%d", year))
	}

	// Return the list of date keys
	return keys
}

// FromDailyDateKey converts a date key into a DateRange.
// The date key is expected to be in the format "YYYY-MM-DD".
// The function returns a DateRange with the StartDate and EndDate being the same day.
// If the key is not in the expected format or contains invalid date components, an error is returned.
func FromDailyDateKey(key string, loc *time.Location) (DateRange, error) {
	parts := strings.Split(key, "-")
	if len(parts) != 3 {
		return DateRange{}, fmt.Errorf("invalid key format")
	}

	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return DateRange{}, fmt.Errorf("invalid year in key: %v", err)
	}

	month, err := strconv.Atoi(parts[1])
	if err != nil {
		return DateRange{}, fmt.Errorf("invalid month in key: %v", err)
	}

	day, err := strconv.Atoi(parts[2])
	if err != nil {
		return DateRange{}, fmt.Errorf("invalid day in key: %v", err)
	}

	startDate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc)
	endDate := time.Date(year, time.Month(month), day, 23, 59, 59, 999999999, loc)

	return DateRange{StartDate: startDate, EndDate: endDate}, nil
}

// FromWeeklyDateKey takes a date key and a location as input and returns a DateRange and an error.
// The date key should be in the format "YYYY-Www", where YYYY is the year and ww is the week number (e.g., 2020-W01).
// The location is used to set the timezone for the start and end dates in the DateRange.
// The function calculates the start and end dates of the week specified in the date key.
// The start date is the Sunday of the week and the end date is the Saturday of the week.
// If the date key is not in the correct format or if the week number is not between 1 and 53, the function returns an error.
func FromWeeklyDateKey(key string, loc *time.Location) (DateRange, error) {
	parts := strings.Split(key, "-W")
	if len(parts) != 2 {
		return DateRange{}, fmt.Errorf("invalid key format")
	}

	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return DateRange{}, fmt.Errorf("invalid year in key: %v", err)
	}

	week, err := strconv.Atoi(parts[1])
	if err != nil {
		return DateRange{}, fmt.Errorf("invalid week in key: %v", err)
	}

	if week < 1 || week > 53 {
		return DateRange{}, fmt.Errorf("week must be between 1 and 53")
	}

	// Calculate the start date of the year
	startDate := time.Date(year, time.January, 1, 0, 0, 0, 0, loc)

	// If January 1st is not a Sunday, adjust the start date to the next Sunday
	if startDate.Weekday() != time.Sunday {
		daysToNextSunday := 7 - int(startDate.Weekday())
		startDate = startDate.AddDate(0, 0, daysToNextSunday)
	}

	// Calculate the start date of the week
	startDate = startDate.AddDate(0, 0, (week-1)*7)

	// Calculate the end date of the week
	endDate := startDate.AddDate(0, 0, 6)
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, loc)
	return DateRange{StartDate: startDate, EndDate: endDate}, nil
}

// FromMonthlyDateKey converts a date key into a DateRange.
// The date key is expected to be in the format "YYYY-MM".
// The function returns a DateRange with the StartDate being the first day of the month and the EndDate being the last day of the same month.
// If the key is not in the expected format or contains invalid date components, an error is returned.
func FromMonthlyDateKey(key string, loc *time.Location) (DateRange, error) {
	parts := strings.Split(key, "-")
	if len(parts) != 2 {
		return DateRange{}, fmt.Errorf("invalid key format")
	}

	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return DateRange{}, fmt.Errorf("invalid year in key: %v", err)
	}

	month, err := strconv.Atoi(parts[1])
	if err != nil {
		return DateRange{}, fmt.Errorf("invalid month in key: %v", err)
	}

	if month < 1 || month > 12 {
		return DateRange{}, fmt.Errorf("month must be between 1 and 12")
	}

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, loc)
	endDate := time.Date(year, time.Month(month+1), 0, 23, 59, 59, 999999999, loc)

	return DateRange{StartDate: startDate, EndDate: endDate}, nil
}

// FromYearlyDateKey converts a date key into a DateRange.
// The date key is expected to be in the format "YYYY".
// The function returns a DateRange with the StartDate being the first day of the year and the EndDate being the last day of the same year.
// If the key is not in the expected format or contains an invalid year, an error is returned.
func FromYearlyDateKey(key string, loc *time.Location) (DateRange, error) {
	year, err := strconv.Atoi(key)
	if err != nil {
		return DateRange{}, fmt.Errorf("invalid year in key: %v", err)
	}

	startDate := time.Date(year, time.January, 1, 0, 0, 0, 0, loc)
	endDate := time.Date(year, time.December, 31, 23, 59, 59, 999999999, loc)

	return DateRange{StartDate: startDate, EndDate: endDate}, nil
}

// DateKeyToDateRange creates a new DateRange instance from a date key.
// It accepts a date key of type string and an optional time zone.
// If the time zone is not provided, the default time zone is used.
// The function returns a pointer to the new DateRange instance and any errors that occurred during the creation.
func DateKeyToDateRange(dateKey string, tz ...*time.Location) (*DateRange, error) {
	// Define the location
	var loc *time.Location
	if len(tz) > 0 {
		loc = tz[0]
	} else {
		loc = DefaultTZ // replace with your default time zone
	}

	// Create a new DateRange instance
	dateRange := &DateRange{}
	// Set the DateRange from the date key
	err := dateRange.SetFromDateKey(dateKey, loc)
	if err != nil {
		return nil, err
	}

	return dateRange, nil
}
