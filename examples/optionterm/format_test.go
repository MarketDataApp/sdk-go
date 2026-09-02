package main

import (
	"errors"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
)

// approxEqual reports whether a and b differ by no more than 1e-9,
// tolerating the float64 rounding noise inherent in px*(1±window)
// (e.g. 100*1.10 does not land on an exact 110.0 bit pattern).
func approxEqual(a, b float64) bool {
	const epsilon = 1e-9
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= epsilon
}

func TestStrikeWindow(t *testing.T) {
	lo, hi := strikeWindow(100, 0.10)
	if !approxEqual(lo, 90) {
		t.Errorf("lo = %v, want 90", lo)
	}
	if !approxEqual(hi, 110) {
		t.Errorf("hi = %v, want 110", hi)
	}
}

func TestStrikeWindow_ZeroOrNegativePrice(t *testing.T) {
	for _, px := range []float64{0, -5} {
		lo, hi := strikeWindow(px, 0.10)
		if lo != 0 || hi != 0 {
			t.Errorf("strikeWindow(%v, 0.10) = (%v, %v), want (0, 0)", px, lo, hi)
		}
	}
}

func TestClampWindow(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{"below floor", 0.01, 0.02},
		{"at floor", 0.02, 0.02},
		{"identity", 0.20, 0.20},
		{"at cap", 0.50, 0.50},
		{"above cap", 0.60, 0.50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clampWindow(tt.in); got != tt.want {
				t.Errorf("clampWindow(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestDTE_SameDay(t *testing.T) {
	// Different hours on the same calendar day: dte is 0, not a fraction
	// truncated from an hours/24 computation.
	now := time.Date(2026, 7, 11, 22, 0, 0, 0, time.UTC)
	exp := time.Date(2026, 7, 11, 3, 0, 0, 0, time.UTC)
	if got := dte(exp, now); got != 0 {
		t.Errorf("dte() = %d, want 0", got)
	}
}

func TestDTE_AcrossMonthBoundary(t *testing.T) {
	now := time.Date(2026, 7, 30, 23, 0, 0, 0, time.UTC)
	exp := time.Date(2026, 8, 1, 1, 0, 0, 0, time.UTC)
	if got := dte(exp, now); got != 2 {
		t.Errorf("dte() = %d, want 2", got)
	}
}

func TestDTE_AcrossTimeZones(t *testing.T) {
	// now is just before midnight UTC on day 1, exp is just after midnight
	// in a different zone that is still calendar day 2 in its own
	// location. A naive Sub().Hours()/24 truncation of a ~2h gap would
	// yield 0; normalizing each to its own y/m/d must yield 1.
	loc := time.FixedZone("TEST", -5*3600)
	now := time.Date(2026, 7, 11, 23, 30, 0, 0, time.UTC)
	exp := time.Date(2026, 7, 12, 1, 0, 0, 0, loc)
	if got := dte(exp, now); got != 1 {
		t.Errorf("dte() = %d, want 1", got)
	}
}

// TestDTE_RealExpirationStamp exercises dte with an expiration timestamp
// matching the live wire convention: the options/expirations endpoint
// stamps expirations at 16:00 Eastern on the expiration day (probed live
// 2026-07-11: 1784318400 = 2026-07-17 16:00 ET / 20:00 UTC, t%86400 =
// 72000). This test exists because the SDK's own unit-test fixtures use
// UTC-midnight stamps, which do not match the wire convention and would
// mislead about dte's behavior on real data: a UTC-midnight stamp
// converted to Eastern lands on the *previous* calendar day, whereas a
// real 16:00-ET stamp lands on the correct one.
func TestDTE_RealExpirationStamp(t *testing.T) {
	eastern, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("LoadLocation() error = %v", err)
	}
	// 2026-07-17 16:00 ET, as the SDK's ToEastern conversion presents it.
	exp := time.Unix(1784318400, 0).In(eastern)
	// A trading morning six calendar days before expiration.
	now := time.Date(2026, 7, 11, 9, 30, 0, 0, eastern)
	if got := dte(exp, now); got != 6 {
		t.Errorf("dte() = %d, want 6", got)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{252 * time.Second, "4m12s"},
		{42 * time.Second, "42s"},
		{3852 * time.Second, "1h4m12s"},
		{0, "0s"},
		{59500 * time.Millisecond, "1m0s"}, // rounds up to nearest second
	}
	for _, tt := range tests {
		if got := formatDuration(tt.d); got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestClassify_RateLimited(t *testing.T) {
	now := testNow
	rle := &marketdata.RateLimitError{
		ResetAt: now.Add(4*time.Minute + 12*time.Second),
	}
	got := classify(rle, now)
	want := "rate limited — resets in 4m12s"
	if got != want {
		t.Errorf("classify() = %q, want %q", got, want)
	}
}

func TestClassify_RateLimited_ResetAtInPastFloorsAtZero(t *testing.T) {
	now := testNow
	rle := &marketdata.RateLimitError{
		ResetAt: now.Add(-time.Minute), // already past reset
	}
	got := classify(rle, now)
	want := "rate limited — resets in 0s"
	if got != want {
		t.Errorf("classify() = %q, want %q", got, want)
	}
}

func TestClassify_Authentication(t *testing.T) {
	ae := &marketdata.AuthenticationError{}
	got := classify(ae, time.Now())
	want := "auth failed — check MARKETDATA_TOKEN"
	if got != want {
		t.Errorf("classify() = %q, want %q", got, want)
	}
}

func TestClassify_NetworkTimeout(t *testing.T) {
	ne := &marketdata.NetworkError{Timeout: true}
	got := classify(ne, time.Now())
	want := "network timeout — retrying on next tick"
	if got != want {
		t.Errorf("classify() = %q, want %q", got, want)
	}
}

func TestClassify_NetworkNonTimeoutFallsThrough(t *testing.T) {
	// Timeout=false is not one of the three special-cased branches, so
	// classify falls through to err.Error() just like any other error.
	ne := &marketdata.NetworkError{Timeout: false}
	got := classify(ne, time.Now())
	want := ne.Error()
	if got != want {
		t.Errorf("classify() = %q, want %q", got, want)
	}
}

func TestClassify_Fallback(t *testing.T) {
	err := errors.New("boom")
	got := classify(err, time.Now())
	if got != "boom" {
		t.Errorf("classify() = %q, want %q", got, "boom")
	}
}
