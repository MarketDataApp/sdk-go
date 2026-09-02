package funds

import (
	"net/url"
	"testing"
	"time"
)

// windowParams serializes a DateWindow the way [Service.Candles] would, so
// tests can assert the exact query parameters a window produces.
func windowParams(w DateWindow) url.Values {
	v := url.Values{}
	w.window().Apply(v)
	return v
}

// TestDateWindow_Serialization proves each date-window mode round-trips to the
// exact API parameters and that no mode leaks parameters from another mode.
// The OnDate case proves funds candles can now emit date=, which the old
// from/to/countback surface could not express.
func TestDateWindow_Serialization(t *testing.T) {
	from := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		w    DateWindow
		want url.Values
	}{
		{"OnDate", OnDate(from), url.Values{"date": {"2024-01-02"}}},
		{"Between", Between(from, to), url.Values{"from": {"2024-01-02"}, "to": {"2024-01-31"}}},
		{"Since", Since(from), url.Values{"from": {"2024-01-02"}}},
		{"Until", Until(to), url.Values{"to": {"2024-01-31"}}},
		{"LastN", LastN(30), url.Values{"countback": {"30"}}},
		{"LastNUntil", LastNUntil(30, to), url.Values{"countback": {"30"}, "to": {"2024-01-31"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := windowParams(tc.w)
			if got.Encode() != tc.want.Encode() {
				t.Errorf("params = %q, want %q", got.Encode(), tc.want.Encode())
			}
		})
	}
}

// TestWithCandleWindow verifies the option stores the window on candle options.
func TestWithCandleWindow(t *testing.T) {
	opts := defaultCandleOptions()
	from := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	WithCandleWindow(Between(from, to)).apply(opts)
	if !opts.window.IsRange() {
		t.Fatal("expected a range window")
	}
	if !opts.window.From().Equal(from) || !opts.window.To().Equal(to) {
		t.Errorf("window bounds = %v..%v, want %v..%v", opts.window.From(), opts.window.To(), from, to)
	}
}

// TestWithResolution verifies the option overrides the default resolution.
func TestWithResolution(t *testing.T) {
	opts := defaultCandleOptions()
	WithResolution(ResolutionWeekly).apply(opts)
	if opts.resolution != ResolutionWeekly {
		t.Errorf("resolution = %q, want W", opts.resolution)
	}
}

// TestCandleOptionInterface verifies each option satisfies the CandleOption
// interface and is non-nil.
func TestCandleOptionInterface(t *testing.T) {
	opts := []CandleOption{
		WithResolution(ResolutionDaily),
		WithCandleWindow(Between(time.Now().AddDate(0, -1, 0), time.Now())),
		WithCandleWindow(LastN(10)),
		WithCandleWindow(OnDate(time.Now())),
		WithCandleWindow(Since(time.Now().AddDate(0, -1, 0))),
		WithCandleWindow(Until(time.Now())),
		WithCandleWindow(LastNUntil(10, time.Now())),
	}
	for _, opt := range opts {
		if opt == nil {
			t.Error("Candle option should not be nil")
		}
	}
}
