package stocks

import (
	"net/url"
	"testing"
	"time"
)

// windowParams serializes a DateWindow the way a service method would, so
// tests can assert the exact query parameters a window produces.
func windowParams(w DateWindow) url.Values {
	v := url.Values{}
	w.window().Apply(v)
	return v
}

// Test quote options
func TestWithFiftyTwoWeek(t *testing.T) {
	opts := &quoteOptions{}
	WithFiftyTwoWeek(true).apply(opts)
	if !opts.fiftyTwoWeek {
		t.Error("fiftyTwoWeek should be true")
	}

	WithFiftyTwoWeek(false).apply(opts)
	if opts.fiftyTwoWeek {
		t.Error("fiftyTwoWeek should be false")
	}
}

func TestWithExtended(t *testing.T) {
	opts := &quoteOptions{}
	WithExtended(true).apply(opts)
	if opts.extended == nil || !*opts.extended {
		t.Error("extended should be true")
	}
	WithExtended(false).apply(opts)
	if opts.extended == nil || *opts.extended {
		t.Error("extended should be false")
	}
}

// TestDateWindow_Serialization proves each date-window mode round-trips to the
// exact API parameters and that no mode leaks parameters from another mode.
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

func TestWithEarningsWindow(t *testing.T) {
	opts := &earningsOptions{}
	WithEarningsWindow(LastN(4)).apply(opts)
	if opts.window.Countback() != 4 {
		t.Errorf("countback = %d, want 4", opts.window.Countback())
	}
}

func TestWithNewsWindow(t *testing.T) {
	opts := &newsOptions{}
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	WithNewsWindow(OnDate(date)).apply(opts)
	if !opts.window.Date().Equal(date) {
		t.Errorf("date = %v, want %v", opts.window.Date(), date)
	}
}

// Test bulk candle options
func TestWithBulkDate(t *testing.T) {
	opts := &bulkCandleOptions{}
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	WithBulkDate(date).apply(opts)
	if !opts.date.Equal(date) {
		t.Errorf("date = %v, want %v", opts.date, date)
	}
}

func TestWithBulkResolution(t *testing.T) {
	opts := &bulkCandleOptions{}
	WithBulkResolution(ResolutionWeekly).apply(opts)
	if opts.resolution != ResolutionWeekly {
		t.Errorf("resolution = %q, want W", opts.resolution)
	}
}

func TestWithAdjustSplits(t *testing.T) {
	opts := &bulkCandleOptions{}

	WithAdjustSplits(true).apply(opts)
	if opts.adjustSplits == nil || !*opts.adjustSplits {
		t.Error("adjustSplits should be true")
	}

	WithAdjustSplits(false).apply(opts)
	if opts.adjustSplits == nil || *opts.adjustSplits {
		t.Error("adjustSplits should be false")
	}
}

func TestWithSnapshot(t *testing.T) {
	opts := &bulkCandleOptions{}
	WithSnapshot(true).apply(opts)
	if opts.snapshot == nil || !*opts.snapshot {
		t.Error("snapshot should be true")
	}
}

func TestWithCandleAdjustDividends(t *testing.T) {
	opts := defaultCandleOptions()
	WithCandleAdjustDividends(false).apply(opts)
	if opts.adjustDividends == nil || *opts.adjustDividends {
		t.Error("candle adjustDividends should be false")
	}
}

func TestWithAdjustDividends(t *testing.T) {
	opts := &bulkCandleOptions{}
	WithAdjustDividends(false).apply(opts)
	if opts.adjustDividends == nil || *opts.adjustDividends {
		t.Error("bulk adjustDividends should be false")
	}
}

// Test all resolution values
func TestResolution_AllValues(t *testing.T) {
	resolutions := []struct {
		r    Resolution
		want string
	}{
		{Resolution1Min, "1"},
		{Resolution5Min, "5"},
		{Resolution15Min, "15"},
		{Resolution30Min, "30"},
		{Resolution1Hour, "60"},
		{ResolutionDaily, "D"},
		{ResolutionWeekly, "W"},
		{ResolutionMonthly, "M"},
		{ResolutionYearly, "Y"},
	}

	for _, tt := range resolutions {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.r.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Test option interface implementations
func TestQuoteOptionInterface(t *testing.T) {
	opt := WithFiftyTwoWeek(true)
	if opt == nil {
		t.Error("WithFiftyTwoWeek should return non-nil QuoteOption")
	}
}

func TestCandleOptionInterface(t *testing.T) {
	opts := []CandleOption{
		WithResolution(ResolutionDaily),
		WithCandleWindow(Between(time.Now().AddDate(0, -1, 0), time.Now())),
		WithCandleWindow(LastN(10)),
		WithCandleExtended(true),
		WithCandleAdjustSplits(true),
		WithCandleAdjustDividends(false),
	}

	for _, opt := range opts {
		if opt == nil {
			t.Error("Candle option should not be nil")
		}
	}
}

func TestEarningsOptionInterface(t *testing.T) {
	opts := []EarningsOption{
		WithEarningsWindow(Between(time.Now().AddDate(0, -1, 0), time.Now())),
		WithEarningsWindow(LastN(5)),
		WithEarningsWindow(OnDate(time.Now())),
	}

	for _, opt := range opts {
		if opt == nil {
			t.Error("Earnings option should not be nil")
		}
	}
}

func TestNewsOptionInterface(t *testing.T) {
	opts := []NewsOption{
		WithNewsWindow(Between(time.Now().AddDate(0, -1, 0), time.Now())),
		WithNewsWindow(LastN(10)),
		WithNewsWindow(OnDate(time.Now())),
	}

	for _, opt := range opts {
		if opt == nil {
			t.Error("News option should not be nil")
		}
	}
}

func TestBulkCandleOptionInterface(t *testing.T) {
	opts := []BulkCandleOption{
		WithBulkDate(time.Now()),
		WithBulkResolution(ResolutionDaily),
		WithAdjustSplits(true),
		WithAdjustDividends(false),
		WithSnapshot(false),
	}

	for _, opt := range opts {
		if opt == nil {
			t.Error("Bulk candle option should not be nil")
		}
	}
}

func TestWithQuotesExtended(t *testing.T) {
	opts := &quotesOptions{}
	WithQuotesExtended(true).apply(opts)
	if opts.extended == nil || !*opts.extended {
		t.Error("extended should be true")
	}

	WithQuotesExtended(false).apply(opts)
	if opts.extended == nil || *opts.extended {
		t.Error("extended should be false")
	}
}
