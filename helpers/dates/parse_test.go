package dates

import (
	"testing"
	"time"
)

func TestToTime(t *testing.T) {
	// Define a slice of test cases
	testCases := []struct {
		name      string
		dateInput interface{}
		tz        *time.Location
		expected  time.Time
		wantErr   bool
	}{
		{
			name:      "Test with specific time",
			dateInput: "2006-01-02T15:04:05Z",
			tz:        time.UTC,
			expected:  time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
			wantErr:   false,
		},
		{
			name:      "Test with Unix timestamp",
			dateInput: int64(1136214245), // corresponds to "2006-01-02T15:04:05Z"
			tz:        time.UTC,
			expected:  time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
			wantErr:   false,
		},
		{
			name:      "Test with string date",
			dateInput: "2022-01-01",
			tz:        time.UTC,
			expected:  time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			wantErr:   false,
		},
		{
			name:      "Test with invalid string",
			dateInput: "invalid",
			tz:        time.UTC,
			expected:  time.Time{},
			wantErr:   true,
		},
		{
			name:      "Test with nil input",
			dateInput: nil,
			tz:        time.UTC,
			expected:  time.Time{},
			wantErr:   true,
		},
	}

	// Loop over test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ToTime(tc.dateInput, tc.tz)
			if (err != nil) != tc.wantErr {
				t.Errorf("ToTime() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !got.Equal(tc.expected) {
				t.Errorf("ToTime() = %v, want %v", got, tc.expected)
			}
		})
	}
}
