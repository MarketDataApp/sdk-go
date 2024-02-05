package dates

import (
	"testing"
	"time"
)

func TestDateRangeParser(t *testing.T) {
	// Define a struct for holding each test case
	type testCase struct {
		dr        DateRange
		tz        *time.Location
		expectedT time.Time
		expectedP string
	}

	// Define your test cases
	testCases := []testCase{
		{
			dr: DateRange{
				StartDate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2022, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			tz:        time.UTC,
			expectedT: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedP: DayPrecision,
		},
		// Add more test cases here
	}

	// Iterate over test cases
	for _, tc := range testCases {
		// Run the function with the test case input
		actualT, actualP, err := ParseDateInput(tc.dr, tc.tz)

		// Check for unexpected errors
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Check the function's output against the expected output
		if !actualT.Equal(tc.expectedT) || actualP != tc.expectedP {
			t.Errorf("dateRangeParser(%v, %v) = %v, %v; want %v, %v", tc.dr, tc.tz, actualT, actualP, tc.expectedT, tc.expectedP)
		}
	}
}

func TestInferPrecision(t *testing.T) {
	// Define a struct for holding each test case
	type testCase struct {
		t        time.Time
		expected string
	}

	// Define your test cases
	testCases := []testCase{
		{
			t:        time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: YearPrecision,
		},
		{
			t:        time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC),
			expected: MonthPrecision,
		},
		{
			t:        time.Date(2022, 1, 2, 0, 0, 0, 0, time.UTC),
			expected: DayPrecision,
		},
		{
			t:        time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC),
			expected: HourPrecision,
		},
		{
			t:        time.Date(2022, 1, 1, 0, 1, 0, 0, time.UTC),
			expected: MinutePrecision,
		},
		{
			t:        time.Date(2022, 1, 1, 0, 0, 1, 0, time.UTC),
			expected: SecondPrecision,
		},
		{
			t:        time.Date(2022, 1, 1, 0, 0, 0, 1, time.UTC),
			expected: NanosecondPrecision,
		},
	}

	// Iterate over test cases
	for _, tc := range testCases {
		// Run the function with the test case input
		actual, err := inferPrecision(tc.t)

		// Check for unexpected errors
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Check the function's output against the expected output
		if actual != tc.expected {
			t.Errorf("inferPrecision(%v) = %v; want %v", tc.t, actual, tc.expected)
		}
	}
}
