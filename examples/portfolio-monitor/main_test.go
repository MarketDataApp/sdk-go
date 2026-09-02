package main

import (
	"strings"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/markets"
)

// TestMarketStatusNote_NilStatusDoesNotPanic is a regression test: an
// earlier "fix" for a nil-pointer panic on NoData added a `status != nil`
// guard to an `else if`, which routed the nil case straight into a final
// `else` that still dereferenced status.Status — moving the panic instead
// of removing it. marketStatusNote must handle nil (the API's NoData
// outcome) without ever dereferencing it.
func TestMarketStatusNote_NilStatusDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("marketStatusNote(nil) panicked: %v", r)
		}
	}()
	if got := marketStatusNote(nil); got != "" {
		t.Errorf("marketStatusNote(nil) = %q, want empty string", got)
	}
}

func TestMarketStatusNote_Closed(t *testing.T) {
	status := &markets.MarketStatus{Status: "closed", Date: time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)}
	got := marketStatusNote(status)
	if !strings.Contains(got, "closed") || !strings.Contains(got, "2026-08-04") {
		t.Errorf("marketStatusNote(closed) = %q, want it to mention the status and date", got)
	}
}

func TestMarketStatusNote_Open(t *testing.T) {
	status := &markets.MarketStatus{Status: "open"}
	got := marketStatusNote(status)
	if !strings.Contains(got, "open") {
		t.Errorf("marketStatusNote(open) = %q, want it to mention the status", got)
	}
}
