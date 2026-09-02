package main

import (
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
)

func TestParseLookup_Call(t *testing.T) {
	now := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	underlying, exp, strike, typ, err := parseLookup("AAPL 2027-01-15 230 call", now)
	if err != nil {
		t.Fatalf("parseLookup() error = %v, want nil", err)
	}
	if underlying != "AAPL" {
		t.Errorf("underlying = %q, want AAPL", underlying)
	}
	wantExp := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)
	if !exp.Equal(wantExp) {
		t.Errorf("exp = %v, want %v", exp, wantExp)
	}
	if strike != 230 {
		t.Errorf("strike = %v, want 230", strike)
	}
	if typ != options.Call {
		t.Errorf("typ = %v, want Call", typ)
	}
}

func TestParseLookup_PutCaseInsensitiveAndUppercasing(t *testing.T) {
	now := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	underlying, _, strike, typ, err := parseLookup("aapl 2027-01-15 230.5 PUT", now)
	if err != nil {
		t.Fatalf("parseLookup() error = %v, want nil", err)
	}
	if underlying != "AAPL" {
		t.Errorf("underlying = %q, want AAPL (uppercased)", underlying)
	}
	if strike != 230.5 {
		t.Errorf("strike = %v, want 230.5", strike)
	}
	if typ != options.Put {
		t.Errorf("typ = %v, want Put", typ)
	}
}

func TestParseLookup_MixedCaseSide(t *testing.T) {
	now := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	_, _, _, typ, err := parseLookup("MSFT 2027-06-18 400 Call", now)
	if err != nil {
		t.Fatalf("parseLookup() error = %v, want nil", err)
	}
	if typ != options.Call {
		t.Errorf("typ = %v, want Call", typ)
	}
}

func TestParseLookup_WrongFieldCount(t *testing.T) {
	now := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	for _, s := range []string{
		"AAPL 2027-01-15 230",
		"AAPL 2027-01-15 230 call extra",
		"AAPL",
		"",
	} {
		_, _, _, _, err := parseLookup(s, now)
		if err == nil {
			t.Errorf("parseLookup(%q) error = nil, want error for wrong field count", s)
		}
	}
}

func TestParseLookup_BadDate(t *testing.T) {
	now := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	_, _, _, _, err := parseLookup("AAPL 2027-13-40 230 call", now)
	if err == nil {
		t.Fatal("parseLookup() error = nil, want error for bad date")
	}
}

func TestParseLookup_BadStrike(t *testing.T) {
	now := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	for _, s := range []string{
		"AAPL 2027-01-15 abc call",
		"AAPL 2027-01-15 0 call",
		"AAPL 2027-01-15 -5 call",
		// strconv.ParseFloat accepts these without error, and NaN
		// escapes a plain <= 0 check (every comparison with NaN is
		// false), so they need explicit rejection.
		"AAPL 2027-01-15 NaN call",
		"AAPL 2027-01-15 Inf call",
	} {
		_, _, _, _, err := parseLookup(s, now)
		if err == nil {
			t.Errorf("parseLookup(%q) error = nil, want error for bad strike", s)
		}
	}
}

func TestParseLookup_BadSide(t *testing.T) {
	now := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	_, _, _, _, err := parseLookup("AAPL 2027-01-15 230 straddle", now)
	if err == nil {
		t.Fatal("parseLookup() error = nil, want error for bad side")
	}
}

// parseLookup does not reject expiration dates before now: the SDK's
// Lookup endpoint is the authority on whether a contract exists, and a
// query for an expired-but-previously-valid contract (e.g. inspecting
// historical detail) should reach it rather than being rejected
// client-side. now is accepted for future-proofing (e.g. a UI warning
// about querying a stale date), not for date validation today.
func TestParseLookup_PastDateAccepted(t *testing.T) {
	now := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	_, _, _, _, err := parseLookup("AAPL 2020-01-15 230 call", now)
	if err != nil {
		t.Errorf("parseLookup() error = %v, want nil for past date", err)
	}
}

func TestParseLookup_ExtraWhitespaceTolerated(t *testing.T) {
	now := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	underlying, _, strike, typ, err := parseLookup("  AAPL   2027-01-15\t230   call  ", now)
	if err != nil {
		t.Fatalf("parseLookup() error = %v, want nil", err)
	}
	if underlying != "AAPL" || strike != 230 || typ != options.Call {
		t.Errorf("got (%q, %v, %v), want (AAPL, 230, Call)", underlying, strike, typ)
	}
}
