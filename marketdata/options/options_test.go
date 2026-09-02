package options

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
	"github.com/MarketDataApp/sdk-go/v2/internal/timezone"
)

// strikeVals serializes a StrikeFilter the way Chain would, so tests can
// assert the exact query parameters a filter produces.
func strikeVals(f StrikeFilter) url.Values {
	v := url.Values{}
	f.strike().apply(v)
	return v
}

// expiryVals serializes an ExpiryFilter the way Chain would.
func expiryVals(f ExpiryFilter) url.Values {
	v := url.Values{}
	f.expiry().apply(v)
	return v
}

// quoteWindowVals serializes an OptionQuoteWindow the way Quote would.
func quoteWindowVals(w OptionQuoteWindow) url.Values {
	v := url.Values{}
	w.optionQuoteWindow().apply(v)
	return v
}

// TestStrikeFilter_Serialization proves each strike mode round-trips to the
// exact API parameters and that no mode leaks parameters from another mode.
func TestStrikeFilter_Serialization(t *testing.T) {
	cases := []struct {
		name string
		f    StrikeFilter
		want url.Values
	}{
		{"Strike", Strike(150), url.Values{"strike": {"150"}}},
		{"StrikeRange", StrikeRange(150, 160), url.Values{"strike": {"150-160"}}},
		{"MinStrike", MinStrike(140), url.Values{"strike": {">=140"}}},
		{"MaxStrike", MaxStrike(160), url.Values{"strike": {"<=160"}}},
		{"StrikeExpr", StrikeExpr("140-160"), url.Values{"strike": {"140-160"}}},
		{"ByDelta", ByDelta(0.30), url.Values{"delta": {"0.3"}}},
		{"ByDeltaNegative", ByDelta(-0.30), url.Values{"delta": {"-0.3"}}},
		{"Strikes", Strikes(150, 160, 170), url.Values{"strike": {"150,160,170"}}},
		{"StrikesSingle", Strikes(150), url.Values{"strike": {"150"}}},
		{"ByDeltas", ByDeltas(0.16, 0.30), url.Values{"delta": {"0.16,0.3"}}},
		{"ByDeltasMixedSign", ByDeltas(0.30, -0.30), url.Values{"delta": {"0.3,-0.3"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := strikeVals(tc.f)
			if got.Encode() != tc.want.Encode() {
				t.Errorf("params = %q, want %q", got.Encode(), tc.want.Encode())
			}
		})
	}
}

// TestExpiryFilter_Serialization proves each expiry mode round-trips to the
// exact API parameters.
func TestExpiryFilter_Serialization(t *testing.T) {
	exp := time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		f    ExpiryFilter
		want url.Values
	}{
		{"AllExpirations", AllExpirations(), url.Values{"expiration": {"all"}}},
		{"OnExpiration", OnExpiration(exp), url.Values{"expiration": {"2026-08-28"}}},
		{"InDTE", InDTE(45), url.Values{"dte": {"45"}}},
		{"InDTEZero", InDTE(0), url.Values{"dte": {"0"}}},
		{"InMonth", InMonth(8), url.Values{"month": {"8"}}},
		{"InYear", InYear(2027), url.Values{"year": {"2027"}}},
		{"InMonthOfYear", InMonthOfYear(6, 2026), url.Values{"month": {"6"}, "year": {"2026"}}},
		{"ExpirationBetween", ExpirationBetween(
			time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC),
		), url.Values{"from": {"2026-08-01"}, "to": {"2026-10-31"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := expiryVals(tc.f)
			if got.Encode() != tc.want.Encode() {
				t.Errorf("params = %q, want %q", got.Encode(), tc.want.Encode())
			}
		})
	}
}

// TestOptionQuoteWindow_Serialization proves each quote-window mode round-trips
// to the exact API parameters.
func TestOptionQuoteWindow_Serialization(t *testing.T) {
	from := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		w    OptionQuoteWindow
		want url.Values
	}{
		{"QuoteOnDate", QuoteOnDate(from), url.Values{"date": {"2024-01-02"}}},
		{"QuoteRange", QuoteRange(from, to), url.Values{"from": {"2024-01-02"}, "to": {"2024-01-31"}}},
		{"QuoteLastNUntil", QuoteLastNUntil(5, to), url.Values{"countback": {"5"}, "to": {"2024-01-31"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := quoteWindowVals(tc.w)
			if got.Encode() != tc.want.Encode() {
				t.Errorf("params = %q, want %q", got.Encode(), tc.want.Encode())
			}
		})
	}
}

// isValidationError reports whether err is a *sdkerrors.ValidationError.
func isValidationError(err error) bool {
	var v *sdkerrors.ValidationError
	return errors.As(err, &v)
}

// TestStrikeFilter_Validate proves each invalid strike/delta value is rejected
// pre-network with a ValidationError.
func TestStrikeFilter_Validate(t *testing.T) {
	cases := []struct {
		name    string
		f       StrikeFilter
		wantErr bool
	}{
		{"StrikeZero", Strike(0), true},
		{"StrikeNegative", Strike(-1), true},
		{"StrikeValid", Strike(150), false},
		{"RangeLowZero", StrikeRange(0, 160), true},
		{"RangeHighZero", StrikeRange(150, 0), true},
		{"RangeInverted", StrikeRange(160, 150), true},
		{"RangeValid", StrikeRange(150, 160), false},
		{"MinZero", MinStrike(0), true},
		{"MinValid", MinStrike(140), false},
		{"MaxZero", MaxStrike(0), true},
		{"MaxValid", MaxStrike(160), false},
		{"ExprEmpty", StrikeExpr(""), true},
		{"ExprValid", StrikeExpr(">=140"), false},
		{"StrikesEmpty", Strikes(), true},
		{"StrikesZeroElement", Strikes(150, 0), true},
		{"StrikesNegativeElement", Strikes(150, -1), true},
		{"StrikesValid", Strikes(150, 160), false},
		{"DeltasEmpty", ByDeltas(), true},
		{"DeltasZeroElement", ByDeltas(0.30, 0), true},
		{"DeltasOutOfRange", ByDeltas(0.30, 1.5), true},
		{"DeltasValid", ByDeltas(0.16, -0.30), false},
		{"DeltaZero", ByDelta(0), true},
		{"DeltaTooHigh", ByDelta(1.5), true},
		{"DeltaTooLow", ByDelta(-1.5), true},
		{"DeltaPositive", ByDelta(0.30), false},
		{"DeltaNegative", ByDelta(-0.30), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.f.strike().validate()
			if tc.wantErr && !isValidationError(err) {
				t.Errorf("validate() = %v, want *ValidationError", err)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("validate() = %v, want nil", err)
			}
		})
	}
}

// TestExpiryFilter_Validate proves each invalid expiry value is rejected
// pre-network with a ValidationError.
func TestExpiryFilter_Validate(t *testing.T) {
	valid := time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC)
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name    string
		f       ExpiryFilter
		wantErr bool
	}{
		{"AllExpirations", AllExpirations(), false},
		{"OnExpirationZero", OnExpiration(time.Time{}), true},
		{"OnExpirationValid", OnExpiration(valid), false},
		{"DTENegative", InDTE(-1), true},
		{"DTEZero", InDTE(0), false},
		{"DTEValid", InDTE(45), false},
		{"MonthLow", InMonth(0), true},
		{"MonthHigh", InMonth(13), true},
		{"MonthValid", InMonth(6), false},
		{"YearLow", InYear(1899), true},
		{"YearHigh", InYear(10000), true},
		{"YearValid", InYear(2026), false},
		{"MonthOfYearBadMonth", InMonthOfYear(0, 2026), true},
		{"MonthOfYearBadYear", InMonthOfYear(6, 1800), true},
		{"MonthOfYearValid", InMonthOfYear(6, 2026), false},
		{"RangeMissingFrom", ExpirationBetween(time.Time{}, to), true},
		{"RangeMissingTo", ExpirationBetween(from, time.Time{}), true},
		{"RangeInverted", ExpirationBetween(to, from), true},
		{"RangeValid", ExpirationBetween(from, to), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.f.expiry().validate()
			if tc.wantErr && !isValidationError(err) {
				t.Errorf("validate() = %v, want *ValidationError", err)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("validate() = %v, want nil", err)
			}
		})
	}
}

// TestOptionQuoteWindow_Validate proves malformed quote windows are rejected
// pre-network with a ValidationError.
func TestOptionQuoteWindow_Validate(t *testing.T) {
	from := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name    string
		w       OptionQuoteWindow
		wantErr bool
	}{
		{"OnDateZero", QuoteOnDate(time.Time{}), true},
		{"OnDateValid", QuoteOnDate(from), false},
		{"RangeMissingFrom", QuoteRange(time.Time{}, to), true},
		{"RangeMissingTo", QuoteRange(from, time.Time{}), true},
		{"RangeInverted", QuoteRange(to, from), true},
		{"RangeValid", QuoteRange(from, to), false},
		{"LastNZero", QuoteLastN(0), true},
		{"LastNNegative", QuoteLastN(-1), true},
		{"LastNValid", QuoteLastN(5), false},
		{"LastNUntilZeroCount", QuoteLastNUntil(0, to), true},
		{"LastNUntilZeroDate", QuoteLastNUntil(5, time.Time{}), true},
		{"LastNUntilValid", QuoteLastNUntil(5, to), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.w.optionQuoteWindow().validate("date")
			if tc.wantErr && !isValidationError(err) {
				t.Errorf("validate() = %v, want *ValidationError", err)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("validate() = %v, want nil", err)
			}
		})
	}
}

// TestChain_RejectsInvalidStrikePreNetwork proves the Chain method rejects an
// invalid union value before any network call: the service is given a nil HTTP
// client, so a network attempt would panic.
func TestChain_RejectsInvalidStrikePreNetwork(t *testing.T) {
	svc := NewService(nil)
	_, _, err := svc.Chain(context.Background(), "AAPL", WithStrike(Strike(-1)))
	if !isValidationError(err) {
		t.Fatalf("Chain() error = %v, want *ValidationError", err)
	}
}

// TestChain_RejectsInvalidExpiryPreNetwork proves an invalid expiry is rejected
// pre-network.
func TestChain_RejectsInvalidExpiryPreNetwork(t *testing.T) {
	svc := NewService(nil)
	_, _, err := svc.Chain(context.Background(), "AAPL", WithExpiry(InMonth(13)))
	if !isValidationError(err) {
		t.Fatalf("Chain() error = %v, want *ValidationError", err)
	}
}

// TestChain_RejectsInvalidExpiryRangePreNetwork proves an inverted expiration
// range is rejected pre-network.
func TestChain_RejectsInvalidExpiryRangePreNetwork(t *testing.T) {
	svc := NewService(nil)
	from := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _, err := svc.Chain(context.Background(), "AAPL", WithExpiry(ExpirationBetween(from, to)))
	if !isValidationError(err) {
		t.Fatalf("Chain() error = %v, want *ValidationError", err)
	}
}

// TestQuote_RejectsInvalidWindowPreNetwork proves the Quote method rejects an
// invalid quote window before any network call.
func TestQuote_RejectsInvalidWindowPreNetwork(t *testing.T) {
	svc := NewService(nil)
	_, _, err := svc.Quote(context.Background(), "AAPL230120C00150000",
		WithOptionQuoteWindow(QuoteOnDate(time.Time{})))
	if !isValidationError(err) {
		t.Fatalf("Quote() error = %v, want *ValidationError", err)
	}
}

// TestQuoteLastN_AnchorsCountback pins the anchor that makes a bare countback
// work on this endpoint. Verified live 2026-08-11: countback=3 alone returns a
// single row (the current quote), while countback=3&to=... returns 3 — the
// same API defect stocks/earnings has, fixed the same way.
func TestQuoteLastN_AnchorsCountback(t *testing.T) {
	got := quoteWindowVals(QuoteLastN(5))

	if got.Get("countback") != "5" {
		t.Errorf("countback = %q, want 5", got.Get("countback"))
	}
	anchor := got.Get("to")
	if anchor == "" {
		t.Fatal("to is missing: a bare countback is ignored by this endpoint")
	}
	today := time.Now().In(timezone.Eastern).Format("2006-01-02")
	if anchor != today {
		t.Errorf("to = %q, want today in Eastern (%q)", anchor, today)
	}
	if got.Has("from") || got.Has("date") {
		t.Errorf("countback must not be combined with from/date, got %v", got)
	}
}
