package dates

import (
	"testing"
	"time"
	"reflect"
)

func TestSetInterval(t *testing.T) {
	tests := []struct {
		name        string
		interval    string
		expectedDur time.Duration
		expectErr   bool
	}{
		{"seconds", "10s", 10 * time.Second, false},
		{"seconds with text", "10 seconds", 10 * time.Second, false},
		{"minutes", "10min", 10 * time.Minute, false},
		{"minutes with text", "10 minutes", 10 * time.Minute, false},
		{"hours", "10H", 10 * time.Hour, false},
		{"hours with text", "10 hours", 10 * time.Hour, false},
		{"weeks", "2W", 2 * 7 * 24 * time.Hour, false},
		{"weeks with text", "2 weeks", 2 * 7 * 24 * time.Hour, false},
		{"months", "2M", 2 * 30 * 24 * time.Hour, false},
		{"months with text", "2 months", 2 * 30 * 24 * time.Hour, false},
		{"years", "2Y", 2 * 365 * 24 * time.Hour, false},
		{"years with text", "2 years", 2 * 365 * 24 * time.Hour, false},
		{"invalid interval", "10x", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ti := &TimeInterval{}
			err := ti.SetInterval(tt.interval)
			if (err != nil) != tt.expectErr {
				t.Errorf("SetInterval() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if err == nil && ti.Duration != tt.expectedDur {
				t.Errorf("SetInterval() = %v, want %v", ti.Duration, tt.expectedDur)
			}
		})
	}
}

func TestSetFromDateKey(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	dr := &DateRange{}

	// Define test cases
	testCases := []struct {
		cacheKey     string
		expectedStart time.Time
		expectedEnd   time.Time
	}{
		{"2022-12-01", time.Date(2022, 12, 1, 0, 0, 0, 0, loc), time.Date(2022, 12, 1, 23, 59, 59, 999999999, loc)},
		{"2022-W01", time.Date(2022, 1, 2, 0, 0, 0, 0, loc), time.Date(2022, 1, 8, 23, 59, 59, 999999999, loc)},
		{"2022-01", time.Date(2022, 1, 1, 0, 0, 0, 0, loc), time.Date(2022, 1, 31, 23, 59, 59, 999999999, loc)},
		{"2022", time.Date(2022, 1, 1, 0, 0, 0, 0, loc), time.Date(2022, 12, 31, 23, 59, 59, 999999999, loc)},
	}

	for _, tc := range testCases {
		err := dr.SetFromDateKey(tc.cacheKey, loc)
		if err != nil {
			t.Errorf("Failed to set DateRange from cache key %s: %v", tc.cacheKey, err)
		}
		if !dr.StartDate.Equal(tc.expectedStart) {
			t.Errorf("Incorrect start date for cache key %s. Expected: %s, Got: %s", tc.cacheKey, tc.expectedStart.String(), dr.StartDate.String())
		}
		if !dr.EndDate.Equal(tc.expectedEnd) {
			t.Errorf("Incorrect end date for cache key %s. Expected: %s, Got: %s", tc.cacheKey, tc.expectedEnd.String(), dr.EndDate.String())
		}
	}
}

func TestNewDateRange(t *testing.T) {
	// Define a timezone
	location, _ := time.LoadLocation("America/New_York")

	// Define the expected start and end dates as time.Time
	expectedStartDate := time.Date(2022, time.January, 1, 0, 0, 0, 0, location)
	expectedEndDate := time.Date(2022, time.December, 31, 0, 0, 0, 0, location)

	// Define test cases
	testCases := []struct {
		startDate interface{}
		endDate   interface{}
	}{
		{expectedStartDate, expectedEndDate},
		{"01/01/2022", "12/31/2022"},
		{"2022-01-01", "2022-12-31"},
		// Add more date formats here
	}

	for _, tc := range testCases {
		dr, err := NewDateRange(tc.startDate, tc.endDate, location)

		// Check if there was an error
		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}

		// Check if the start and end dates were set correctly
		if !dr.StartDate.Equal(expectedStartDate) {
			t.Errorf("Expected start date to be %v, but got %v", expectedStartDate, dr.StartDate)
		}
		if !dr.EndDate.Equal(expectedEndDate) {
			t.Errorf("Expected end date to be %v, but got %v", expectedEndDate, dr.EndDate)
		}
	}
}

func TestSetDatesWithMultipleFormats(t *testing.T) {
	// Define a timezone
	location, _ := time.LoadLocation("America/New_York")

	// Define the expected start and end dates as time.Time
	expectedStartDate := time.Date(2022, time.January, 1, 0, 0, 0, 0, location)
	expectedEndDate := time.Date(2022, time.December, 31, 0, 0, 0, 0, location)

	// Define test cases
	testCases := []struct {
		startDate string
		endDate   string
	}{
		{"01/01/2022", "12/31/2022"},
		{"2022-01-01", "2022-12-31"},
		// Add more date formats here
	}

	for _, tc := range testCases {
		dr := &DateRange{}

		// Call the SetDates method
		err := dr.SetDates(tc.startDate, tc.endDate, location)

		// Check if there was an error
		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}

		// Check if the start and end dates were set correctly
		if !dr.StartDate.Equal(expectedStartDate) {
			t.Errorf("Expected start date to be %v, but got %v", expectedStartDate, dr.StartDate)
		}
		if !dr.EndDate.Equal(expectedEndDate) {
			t.Errorf("Expected end date to be %v, but got %v", expectedEndDate, dr.EndDate)
		}
	}
}

func TestSetDatesUTC(t *testing.T) {
	dr := &DateRange{}

	// Define a start and end date
	startDate := time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2022, time.December, 31, 0, 0, 0, 0, time.UTC)

	// Call the SetDates method
	err := dr.SetDates(startDate, endDate)

	// Check if there was an error
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	// Check if the start and end dates were set correctly
	if !dr.StartDate.Equal(startDate) {
		t.Errorf("Expected start date to be %v, but got %v", startDate, dr.StartDate)
	}
	if !dr.EndDate.Equal(endDate) {
		t.Errorf("Expected end date to be %v, but got %v", endDate, dr.EndDate)
	}
}

func TestSetDatesNY(t *testing.T) {
	// Define the start and end times
	location, _ := time.LoadLocation("America/New_York")
	startTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 9, 30, 0, 0, location)
	endTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 16, 0, 0, 0, location)

	// Create a new DateRange
	dr := &DateRange{}

	// Set the dates
	err := dr.SetDates(startTime, endTime, location)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check if the start and end dates are set correctly
	if !dr.StartDate.Equal(startTime) {
		t.Errorf("Expected start time to be %v, but got %v", startTime, dr.StartDate)
	}
	if !dr.EndDate.Equal(endTime) {
		t.Errorf("Expected end time to be %v, but got %v", endTime, dr.EndDate)
	}
}

func TestGetDateRange(t *testing.T) {
	location, _ := time.LoadLocation("UTC")
	tests := []struct {
		name      string
		tz        *time.Location
		dateInput string
		fromInput string
		toInput   string
		wantErr   bool
	}{
		{
			name:      "Single date",
			tz:        location,
			dateInput: "2022-01-01",
			wantErr:   false,
		},
		{
			name:      "Date range",
			tz:        location,
			fromInput: "2022-01-01",
			toInput:   "2022-01-31",
			wantErr:   false,
		},
		{
			name:      "Invalid date",
			tz:        location,
			dateInput: "invalid-date",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetDateRange(tt.tz, tt.dateInput, tt.fromInput, tt.toInput)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDateRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDateRange_Contains(t *testing.T) {
	// Initialize a new DateRange
	dr := &DateRange{}

	// Use SetDates to set the StartDate and EndDate
	err := dr.SetDates(1699462680, 1699462800)
	if err != nil {
		t.Fatalf("Failed to set dates: %v", err)
	}

	// Define test cases
	testCases := []struct {
		input    int64
		expected bool
	}{
		{1699462680, true},
		{1699462740, true},
		{1699462800, true},
		{1699462860, false},
		{1699462920, false},
		{1699462980, false},
		{1699463040, false},
		{1699463100, false},
		{1699463160, false},
		{1699463220, false},
	}

	// Run test cases
	for _, tc := range testCases {
		result := dr.Contains(time.Unix(tc.input, 0))
		if result != tc.expected {
			t.Errorf("Expected %v for input %v, got %v", tc.expected, tc.input, result)
		}
	}
}

func TestDateRange_ValidateTimestamps(t *testing.T) {
	// Initialize a new DateRange
	dr := &DateRange{}

	// Use SetDates to set the StartDate and EndDate
	err := dr.SetDates(1699462680, 1699462800)
	if err != nil {
		t.Fatalf("Failed to set dates: %v", err)
	}

	// Define test cases
	testCases := []struct {
		input          []int64
		expectedValid  []int64
		expectedInvalid []int64
	}{
		{
			input:          []int64{1699462680, 1699462740, 1699462800, 1699462860},
			expectedValid:  []int64{1699462680, 1699462740, 1699462800},
			expectedInvalid: []int64{1699462860},
		},
		{
			input:          []int64{1699462920, 1699462980, 1699463040},
			expectedValid:  []int64{},
			expectedInvalid: []int64{1699462920, 1699462980, 1699463040},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		valid, invalid := dr.ValidateTimestamps(tc.input...)
		if (valid == nil && len(tc.expectedValid) != 0) || (valid != nil && !reflect.DeepEqual(valid, tc.expectedValid)) {
			t.Errorf("Expected valid timestamps %v, got %v", tc.expectedValid, valid)
		}
		if !reflect.DeepEqual(invalid, tc.expectedInvalid) {
			t.Errorf("Expected invalid timestamps %v, got %v", tc.expectedInvalid, invalid)
		}
	}
}

func TestInside(t *testing.T) {
	// Define a DateRange from 2022-01-01 to 2022-12-31
	dr := DateRange{
		StartDate: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2022, time.December, 31, 23, 59, 59, 999999999, time.UTC),
	}

	// Define a time within the DateRange
	tm := time.Date(2022, time.June, 15, 0, 0, 0, 0, time.UTC)

	// Test if the time is inside the DateRange
	if !dr.Contains(tm) {
		t.Errorf("Expected Inside to return true for time %v, but it returned false", tm)
	}

	// Define a DateRange within the original DateRange
	dr2 := DateRange{
		StartDate: time.Date(2022, time.February, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2022, time.March, 31, 23, 59, 59, 999999999, time.UTC),
	}

	// Test if the second DateRange is inside the first DateRange
	if !dr.Contains(dr2) {
		t.Errorf("Expected Inside to return true for DateRange %v, but it returned false", dr2)
	}

	// Define a time outside the DateRange
	tm2 := time.Date(2023, time.June, 15, 0, 0, 0, 0, time.UTC)

	// Test if the time is inside the DateRange
	if dr.Contains(tm2) {
		t.Errorf("Expected Inside to return false for time %v, but it returned true", tm2)
	}

	// Define a DateRange outside the original DateRange
	dr3 := DateRange{
		StartDate: time.Date(2023, time.February, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2023, time.March, 31, 23, 59, 59, 999999999, time.UTC),
	}

	// Test if the third DateRange is inside the first DateRange
	if dr.Contains(dr3) {
		t.Errorf("Expected Inside to return false for DateRange %v, but it returned true", dr3)
	}

	// Define a DateRange that partially intersects the original DateRange
	dr4 := DateRange{
		StartDate: time.Date(2022, time.December, 15, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2023, time.January, 15, 23, 59, 59, 999999999, time.UTC),
	}

	// Test if the fourth DateRange is inside the first DateRange
	if dr.Contains(dr4) {
		t.Errorf("Expected Inside to return false for DateRange %v, but it returned true", dr4)
	}
}

func TestOutside(t *testing.T) {
	// Define a DateRange from 2022-01-01 to 2022-12-31
	dr := DateRange{
		StartDate: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2022, time.December, 31, 23, 59, 59, 999999999, time.UTC),
	}

	// Define a time within the DateRange
	tm := time.Date(2022, time.June, 15, 0, 0, 0, 0, time.UTC)

	// Test if the time is outside the DateRange
	if dr.DoesNotContain(tm) {
		t.Errorf("Expected Outside to return false for time %v, but it returned true", tm)
	}

	// Define a DateRange within the original DateRange
	dr2 := DateRange{
		StartDate: time.Date(2022, time.February, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2022, time.March, 31, 23, 59, 59, 999999999, time.UTC),
	}

	// Test if the second DateRange is outside the first DateRange
	if dr.DoesNotContain(dr2) {
		t.Errorf("Expected Outside to return false for DateRange %v, but it returned true", dr2)
	}

	// Define a time outside the DateRange
	tm2 := time.Date(2023, time.June, 15, 0, 0, 0, 0, time.UTC)

	// Test if the time is outside the DateRange
	if !dr.DoesNotContain(tm2) {
		t.Errorf("Expected Outside to return true for time %v, but it returned false", tm2)
	}

	// Define a DateRange outside the original DateRange
	dr3 := DateRange{
		StartDate: time.Date(2023, time.February, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2023, time.March, 31, 23, 59, 59, 999999999, time.UTC),
	}

	// Test if the third DateRange is outside the first DateRange
	if !dr.DoesNotContain(dr3) {
		t.Errorf("Expected Outside to return true for DateRange %v, but it returned false", dr3)
	}

	// Define a DateRange that partially intersects the original DateRange
	dr4 := DateRange{
		StartDate: time.Date(2022, time.December, 15, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2023, time.January, 15, 23, 59, 59, 999999999, time.UTC),
	}

	// Test if the fourth DateRange is outside the first DateRange
	if dr.DoesNotContain(dr4) {
		t.Errorf("Expected Outside to return false for DateRange %v, but it returned true", dr4)
	}
}

func TestConverges(t *testing.T) {
	location := time.UTC
	baseStart := time.Date(2022, 1, 1, 0, 0, 0, 0, location)
	baseEnd := time.Date(2022, 1, 31, 0, 0, 0, 0, location)
	baseRange := DateRange{StartDate: baseStart, EndDate: baseEnd}

	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected bool
	}{
		{
			name:     "Inside",
			start:    time.Date(2022, 1, 10, 0, 0, 0, 0, location),
			end:      time.Date(2022, 1, 20, 0, 0, 0, 0, location),
			expected: false,
		},
		{
			name:     "Outside",
			start:    time.Date(2022, 2, 1, 0, 0, 0, 0, location),
			end:      time.Date(2022, 2, 28, 0, 0, 0, 0, location),
			expected: false,
		},
		{
			name:     "Converges",
			start:    time.Date(2022, 1, 20, 0, 0, 0, 0, location),
			end:      time.Date(2022, 2, 10, 0, 0, 0, 0, location),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRange := DateRange{StartDate: tt.start, EndDate: tt.end}
			if got := baseRange.PartiallyContains(testRange); got != tt.expected {
				t.Errorf("Converges() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsEarlierThan(t *testing.T) {
	// Define a DateRange from 2022-01-01 to 2022-12-31
	dr := DateRange{
		StartDate: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2022, time.December, 31, 23, 59, 59, 999999999, time.UTC),
	}

	// Define test cases
	testCases := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{"Earlier time", time.Date(2021, time.December, 31, 0, 0, 0, 0, time.UTC), false},
		{"Later time", time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC), true},
		{"Same time", time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC), false},
		{"Invalid type", "invalid", false},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, _ := dr.IsEarlierThan(tc.input)
			if result != tc.expected {
				t.Errorf("Testcase '%s' failed. Expected %v, got %v", tc.name, tc.expected, result)
			}
		})
	}
}

func TestIsLaterThan(t *testing.T) {
	// Define a DateRange from 2022-01-01 to 2022-12-31
	dr := DateRange{
		StartDate: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2022, time.December, 31, 23, 59, 59, 999999999, time.UTC),
	}

	// Define test cases
	testCases := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{"Earlier time", time.Date(2021, time.December, 31, 0, 0, 0, 0, time.UTC), true},
		{"Later time", time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC), false},
		{"Same time", time.Date(2022, time.December, 31, 23, 59, 59, 999999999, time.UTC), false},
		{"Invalid type", "invalid", false},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, _ := dr.IsLaterThan(tc.input)
			if result != tc.expected {
				t.Errorf("Testcase '%s' failed. Expected %v, got %v", tc.name, tc.expected, result)
			}
		})
	}
}

func TestEarliest(t *testing.T) {
	tz, _ := time.LoadLocation("UTC")
	date1 := time.Date(2022, 1, 1, 0, 0, 0, 0, tz)
	date2 := time.Date(2022, 1, 2, 0, 0, 0, 0, tz)
	date3 := time.Date(2022, 1, 3, 0, 0, 0, 0, tz)
	unixTimestamp := int64(1672444800) // Represents 2023-01-01 00:00:00 in Unix timestamp

	dr := DateRange{
		StartDate: time.Date(2022, 1, 4, 0, 0, 0, 0, tz),
		EndDate:   time.Date(2022, 1, 5, 0, 0, 0, 0, tz),
	}

	earliest, err := Earliest(date1, date2, date3, dr, unixTimestamp)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !earliest.Equal(date1) {
		t.Errorf("Expected %v, got %v", date1, earliest)
	}
}

func TestLatest(t *testing.T) {
	tz, _ := time.LoadLocation("UTC")
	date1 := time.Date(2022, 1, 1, 0, 0, 0, 0, tz)
	date2 := time.Date(2022, 1, 2, 0, 0, 0, 0, tz)
	date3 := time.Date(2022, 1, 3, 0, 0, 0, 0, tz)

	dr := DateRange{
		StartDate: time.Date(2022, 1, 4, 0, 0, 0, 0, tz),
		EndDate:   time.Date(2022, 1, 5, 0, 0, 0, 0, tz),
	}

	unixTimestamp := int64(1672444800) // Represents 2023-01-01 00:00:00 in Unix timestamp

	latest, err := Latest(date1, date2, date3, dr, unixTimestamp)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expectedDate := time.Unix(unixTimestamp, 0)
	if !latest.Equal(expectedDate) {
		t.Errorf("Expected %v, got %v", expectedDate, latest)
	}
}

func TestCombineDateRanges(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")

	dr1 := DateRange{
		StartDate: time.Date(2022, 1, 1, 0, 0, 0, 0, loc),
		EndDate:   time.Date(2022, 1, 31, 0, 0, 0, 0, loc),
	}

	dr2 := DateRange{
		StartDate: time.Date(2022, 2, 1, 0, 0, 0, 0, loc),
		EndDate:   time.Date(2022, 2, 28, 0, 0, 0, 0, loc),
	}

	dr3 := DateRange{
		StartDate: time.Date(2022, 3, 1, 0, 0, 0, 0, loc),
		EndDate:   time.Date(2022, 3, 31, 0, 0, 0, 0, loc),
	}

	// Test with individual DateRange instances
	combined, err := CombineDateRanges(dr1, dr2, dr3)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if combined.StartDate != dr1.StartDate {
		t.Errorf("Expected start date to be %v, but got %v", dr1.StartDate, combined.StartDate)
	}

	if combined.EndDate != dr3.EndDate {
		t.Errorf("Expected end date to be %v, but got %v", dr3.EndDate, combined.EndDate)
	}

	// Test with a slice of DateRange instances
	dateRanges := []DateRange{dr1, dr2, dr3}
	combined, err = CombineDateRanges(dateRanges)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if combined.StartDate != dr1.StartDate {
		t.Errorf("Expected start date to be %v, but got %v", dr1.StartDate, combined.StartDate)
	}

	if combined.EndDate != dr3.EndDate {
		t.Errorf("Expected end date to be %v, but got %v", dr3.EndDate, combined.EndDate)
	}
}

func TestGetDateKeyType(t *testing.T) {
	tests := []struct {
		name     string
		dateKey  string
		want     string
		wantErr  bool
	}{
		{
			name:     "Test daily date key",
			dateKey:  "2022-01-01",
			want:     "days",
			wantErr:  false,
		},
		{
			name:     "Test weekly date key",
			dateKey:  "2022-W01",
			want:     "weeks",
			wantErr:  false,
		},
		{
			name:     "Test monthly date key",
			dateKey:  "2022-01",
			want:     "months",
			wantErr:  false,
		},
		{
			name:     "Test yearly date key",
			dateKey:  "2022",
			want:     "years",
			wantErr:  false,
		},
		{
			name:     "Test invalid date key",
			dateKey:  "invalid",
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetDateKeyType(tt.dateKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDateKeyType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetDateKeyType() = %v, want %v", got, tt.want)
			}
		})
	}
}