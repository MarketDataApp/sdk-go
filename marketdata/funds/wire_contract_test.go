package funds

// Wire-contract test (ADR-010): a hand-written fixture from testdata/,
// asserted field by field. See the stocks package for the rationale.

import (
	"context"
	"net/http"
	"os"
	"testing"
)

func TestWireContract_FundCandles(t *testing.T) {
	body, err := os.ReadFile("testdata/candles.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})
	svc := newTestService(handler)
	candles, _, err := svc.Candles(context.Background(), "VFIAX")
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
	if len(candles) != 1 {
		t.Fatalf("len = %d, want 1", len(candles))
	}
	c := candles[0]
	if c.Time.Unix() != 1704067200 || c.Open != 450.11 || c.High != 451.22 || c.Low != 449.33 || c.Close != 450.99 {
		t.Errorf("candle = %+v, does not match fixture", c)
	}
}
