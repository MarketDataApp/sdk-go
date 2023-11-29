package dates

import (
	"testing"
	"reflect"
)



func TestFilterTimeFormats(t *testing.T) {
	timeFormats := []TimeFormats{
		{"2006-01-02", "day", false},
		{"2006-01-02T15:04", "minute", false},
		{"2006-01-02T15:04:05", "second", false},
		{"2006-01-02T15:04:05.999", "millisecond", false},
		{"2006-01-02T15:04:05.999999", "microsecond", false},
		{"2006-01-02T15:04:05.999999999", "nanosecond", false},
		{"2006-01-02T15:04Z", "minute", true},
		{"2006-01-02T15:04:05Z", "second", true},
		{"2006-01-02T15:04:05.999Z", "millisecond", true},
		{"2006-01-02T15:04:05.999999Z", "microsecond", true},
		{"2006-01-02T15:04:05.999999999Z", "nanosecond", true},
	}

	tests := []struct {
		name      string
		precision string
		timezone  bool
		want      []TimeFormats
	}{
		{
			name:      "Filter by day precision without timezone",
			precision: "day",
			timezone:  false,
			want:      []TimeFormats{{"2006-01-02", "day", false}},
		},
		{
			name:      "Filter by minute precision with timezone",
			precision: "minute",
			timezone:  true,
			want:      []TimeFormats{{"2006-01-02T15:04Z", "minute", true}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterTimeFormats(timeFormats, tt.precision, tt.timezone); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterTimeFormats() = %v, want %v", got, tt.want)
			}
		})
	}
}