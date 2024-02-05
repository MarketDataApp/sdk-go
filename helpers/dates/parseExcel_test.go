package dates

import (
	"testing"
	"time"
)

func TestFromExcelTime(t *testing.T) {
	tests := []struct {
		name    string
		input   float64
		want    time.Time
		wantErr bool
	}{
		{
			name:    "Test with float64 input",
			input:   float64(44197), // Represents 2021-01-01 in Excel
			want:    time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "Test Case 1: Start of 1800",
			input:   float64(-36522),
			want:    time.Date(1800, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "Test Case 2: Start of 1900",
			input:   float64(2),
			want:    time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "Test Case 3: Start of 2000",
			input:   float64(36526),
			want:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromExcelTime(tt.input, time.UTC)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromExcelTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("FromExcelTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExcelParsingUTC(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    time.Time
		wantErr bool
	}{
		{
			name:    "Test with float64 input",
			input:   float64(44197), // Represents 2021-01-01 in Excel
			want:    time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "Test with string input",
			input:   "44197", // Represents 2021-01-01 in Excel
			want:    time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "Test with invalid string input",
			input:   "invalid",
			want:    time.Time{},
			wantErr: true,
		},
		{
			name:    "Test with invalid type input",
			input:   []int{1, 2, 3},
			want:    time.Time{},
			wantErr: true,
		},
		{
			name:    "Test Case 1: Noon of Jan 1, 2018",
			input:   float64(43101.5),
			want:    time.Date(2018, time.January, 1, 12, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "Test Case 2: Start of 1900",
			input:   float64(2),
			want:    time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "Test Case 3: Start of 2000",
			input:   float64(36526),
			want:    time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToTime(tt.input, time.UTC)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("ToTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExcelParsingNewYork(t *testing.T) {
	location, _ := time.LoadLocation("America/New_York")

	tests := []struct {
		name    string
		input   interface{}
		want    time.Time
		wantErr bool
	}{
		{
			name:    "Test with float64 input",
			input:   float64(44197), // Represents 2021-01-01 in Excel
			want:    time.Date(2021, time.January, 1, 0, 0, 0, 0, location),
			wantErr: false,
		},
		{
			name:    "Test with string input",
			input:   "44197", // Represents 2021-01-01 in Excel
			want:    time.Date(2021, time.January, 1, 0, 0, 0, 0, location),
			wantErr: false,
		},
		{
			name:    "Test with invalid string input",
			input:   "invalid",
			want:    time.Time{},
			wantErr: true,
		},
		{
			name:    "Test with invalid type input",
			input:   []int{1, 2, 3},
			want:    time.Time{},
			wantErr: true,
		},
		{
			name:    "Test Case 1: Noon of Jan 1, 2018",
			input:   float64(43101.5),
			want:    time.Date(2018, time.January, 1, 12, 0, 0, 0, location),
			wantErr: false,
		},
		{
			name:    "Test Case 2: Start of 1900",
			input:   float64(2),
			want:    time.Date(1900, time.January, 1, 0, 0, 0, 0, location),
			wantErr: false,
		},
		{
			name:    "Test Case 3: Start of 2000",
			input:   float64(36526),
			want:    time.Date(2000, time.January, 1, 0, 0, 0, 0, location),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToTime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("ToTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTryExcelTimestamps(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    time.Time
		wantErr bool
	}{
		{
			name:    "Test with string input",
			input:   "36540.00",
			want:    time.Date(2000, 1, 15, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "Test with int input",
			input:   36540,
			want:    time.Date(2000, 1, 15, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "Test with float64 input",
			input:   float64(36540.00),
			want:    time.Date(2000, 1, 15, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToTime(tt.input, time.UTC)
			if (err != nil) != tt.wantErr {
				t.Errorf("tryParseExcelTimestamp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("tryParseExcelTimestamp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToExcelTime(t *testing.T) {
	location, _ := time.LoadLocation("America/New_York")

	tests := []struct {
		name     string
		input    time.Time
		expected float64
	}{
		{
			name:     "Test case 1: 1/1/1800",
			input:    time.Date(1800, 1, 1, 0, 0, 0, 0, location),
			expected: -36522.00,
		},
		{
			name:     "Test case 2: 1/1/1900",
			input:    time.Date(1900, 1, 1, 0, 0, 0, 0, location),
			expected: 2.00,
		},
		{
			name:     "Test case 3: 1/1/2018 12:00:00",
			input:    time.Date(2018, 1, 1, 12, 0, 0, 0, location),
			expected: 43101.50,
		},
		{
			name:     "Test case 4: 10/15/2023",
			input:    time.Date(2023, 10, 15, 0, 0, 0, 0, location),
			expected: 45214.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToExcelTime(tt.input, location)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}
