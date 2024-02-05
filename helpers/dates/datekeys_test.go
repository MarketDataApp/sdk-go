package dates

import (
	"testing"
	"time"
)

func TestGenerateDateKeys(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")

	// Define test cases
	testCases := []struct {
		name       string
		keyType    string
		startDate  time.Time
		endDate    time.Time
		expectKeys []string
	}{
		{
			name:      "Test daily keys",
			keyType:   "days",
			startDate: time.Date(2022, 1, 1, 0, 0, 0, 0, ny),
			endDate:   time.Date(2022, 1, 3, 0, 0, 0, 0, ny),
			expectKeys: []string{
				"2022-01-01",
				"2022-01-02",
				"2022-01-03",
			},
		},
		{
			name:      "Test weekly keys",
			keyType:   "weeks",
			startDate: time.Date(2022, 1, 1, 0, 0, 0, 0, ny),
			endDate:   time.Date(2022, 1, 15, 0, 0, 0, 0, ny),
			expectKeys: []string{
				"2022-W01",
				"2022-W02",
				"2022-W03",
			},
		},
		{
			name:      "Test monthly keys",
			keyType:   "months",
			startDate: time.Date(2022, 1, 1, 0, 0, 0, 0, ny),
			endDate:   time.Date(2022, 4, 1, 0, 0, 0, 0, ny),
			expectKeys: []string{
				"2022-01",
				"2022-02",
				"2022-03",
				"2022-04",
			},
		},
		{
			name:      "Test yearly keys",
			keyType:   "years",
			startDate: time.Date(2020, 1, 1, 0, 0, 0, 0, ny),
			endDate:   time.Date(2023, 1, 1, 0, 0, 0, 0, ny),
			expectKeys: []string{
				"2020",
				"2021",
				"2022",
				"2023",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dr := &DateRange{
				StartDate: tc.startDate,
				EndDate:   tc.endDate,
			}

			keys, err := dr.GenerateDateKeys(tc.keyType)
			if err != nil {
				t.Fatalf("Failed to generate date keys: %v", err)
			}

			if len(keys) != len(tc.expectKeys) {
				t.Errorf("Expected %d keys, got %d keys", len(tc.expectKeys), len(keys))
			}

			for i, key := range keys {
				if key != tc.expectKeys[i] {
					t.Errorf("Expected key '%s', got '%s'", tc.expectKeys[i], key)
				}
			}
		})
	}
}

func TestIsValidDateKey(t *testing.T) {
	tests := []struct {
		name     string
		dateKey  string
		expected bool
	}{
		{
			name:     "Valid daily date key",
			dateKey:  "2022-01-01",
			expected: true,
		},
		{
			name:     "Valid weekly date key",
			dateKey:  "2022-W01",
			expected: true,
		},
		{
			name:     "Valid monthly date key",
			dateKey:  "2022-01",
			expected: true,
		},
		{
			name:     "Valid yearly date key",
			dateKey:  "2022",
			expected: true,
		},
		{
			name:     "Invalid date key",
			dateKey:  "invalid-key",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidDateKey(tt.dateKey); got != tt.expected {
				t.Errorf("IsValidDateKey() = %v, want %v", got, tt.expected)
			}
		})
	}
}
