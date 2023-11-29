package dates

import (
	"testing"
	"time"
)
func TestTryParseStringDate(t *testing.T) {
    // Test Case 1: Designed to succeed
    // Input is a string representing a date without time
    successInput := "April 7, 2021"
    expectedTime, _ := time.Parse("January 2, 2006", successInput)

    timeResult, precision, err := ParseDateInput(successInput, time.UTC)
    if err != nil {
        t.Errorf("TestTryParseStringDate success case failed with error: %s", err)
    }
    if !timeResult.Equal(expectedTime) {
        t.Errorf("TestTryParseStringDate success case failed: expected %v, got %v", expectedTime, timeResult)
    }
    if precision != "day" {
        t.Errorf("TestTryParseStringDate success case failed: expected precision %v, got %v", "day", precision)
    }

    // Test Case 2: Designed to fail
    // Input is a string that doesn't match any format
    failInput := "unparsable date string"
    _, _, err = ParseDateInput(failInput, DefaultTZ)
    if err == nil {
        t.Errorf("TestTryParseStringDate fail case did not result in an error as expected")
    }
}

func TestTryParseStringTZDatesNoTZ(t *testing.T) {
	// Define test cases
	testCases := []struct {
		dateStr   string
		expectedTime time.Time
		expectedPrecision string
	}{
		{
			dateStr: "2022-03-15T18:30:00",
			expectedTime: time.Date(2022, 3, 15, 18, 30, 0, 0, DefaultTZ),
			expectedPrecision: "second",
		},
		{
			dateStr: "2022-07-20",
			expectedTime: time.Date(2022, 7, 20, 0, 0, 0, 0, DefaultTZ),
			expectedPrecision: "day",
		},
		{
			dateStr: "2006-01-02",
			expectedTime: time.Date(2006, 1, 2, 0, 0, 0, 0, DefaultTZ),
			expectedPrecision: "day",
		},
	}

	for _, testCase := range testCases {
		// Call the function with the test case input
		actualTime, actualPrecision, err := ParseDateInput(testCase.dateStr, DefaultTZ)

		// Check if the function returned an error
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Check if the actual output matches the expected output
		if !actualTime.Equal(testCase.expectedTime) || actualPrecision != testCase.expectedPrecision {
			t.Errorf("unexpected output: got time %v precision %s, want time %v precision %s",
				actualTime, actualPrecision, testCase.expectedTime, testCase.expectedPrecision)
		}
	}
}

func TestTryParseSpecialCases(t *testing.T) {
	tests := []struct {
		name     string
		dateStr  string
		wantTime time.Time
		wantStr  string
		wantBool bool
	}{
		{
			name:     "Test Today",
			dateStr:  "today",
			wantTime: time.Now().Truncate(24 * time.Hour),
			wantStr:  "day",
			wantBool: true,
		},
		{
			name:     "Test Yesterday",
			dateStr:  "yesterday",
			wantTime: time.Now().Truncate(24 * time.Hour).AddDate(0, 0, -1),
			wantStr:  "day",
			wantBool: true,
		},
		{
			name:     "Test Now",
			dateStr:  "now",
			wantTime: time.Now(),
			wantStr:  "second",
			wantBool: true,
		},
		{
			name:     "Test Invalid",
			dateStr:  "invalid",
			wantTime: time.Time{},
			wantStr:  "",
			wantBool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTime, gotStr, gotBool := tryParseSpecialCases(tt.dateStr)
			if gotTime.Truncate(time.Second) != tt.wantTime.Truncate(time.Second) {
				t.Errorf("tryParseSpecialCases() gotTime = %v, want %v", gotTime, tt.wantTime)
			}
			if gotStr != tt.wantStr {
				t.Errorf("tryParseSpecialCases() gotStr = %v, want %v", gotStr, tt.wantStr)
			}
			if gotBool != tt.wantBool {
				t.Errorf("tryParseSpecialCases() gotBool = %v, want %v", gotBool, tt.wantBool)
			}
		})
	}
}

func TestTryParseStringTZDates(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	az, _ := time.LoadLocation("America/Phoenix")

    testCases := []struct {
        name     string
        dateStr  string
        tz       *time.Location
        wantTime time.Time
        wantErr  bool
    }{
        {
            name:     "Valid date string with timezone",
            dateStr:  "2006-01-02T15:04:05Z-07:00",
            tz:       az,
            wantTime: time.Date(2006, 1, 2, 15, 4, 5, 0, az),
            wantErr:  false,
        },
        {
            name:     "Invalid date string",
            dateStr:  "invalid-date",
            tz:       az,
            wantTime: time.Time{},
            wantErr:  true,
        },
        {
            name:     "Valid date string with EST",
            dateStr:  "2006-01-02T15:04:05Z-05:00",
            tz:       ny,
            wantTime: time.Date(2006, 1, 2, 15, 4, 5, 0, ny),
            wantErr:  false,
        },
    }

    // Run the test cases
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            gotTime, _, err := ParseDateInput(tc.dateStr, tc.tz)
            if (err != nil) != tc.wantErr {
                t.Errorf("tryParseStringTZDates() error = %v, wantErr %v", err, tc.wantErr)
                return
            }
            if !gotTime.Equal(tc.wantTime) {
                t.Errorf("tryParseStringTZDates() = %v, want %v", gotTime, tc.wantTime)
            }
        })
    }
}

