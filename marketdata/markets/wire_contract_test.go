package markets

// Wire-contract test (ADR-010): a hand-written fixture from testdata/,
// asserted field by field. See the stocks package for the rationale.

import (
	"context"
	"net/http"
	"os"
	"testing"
)

func TestWireContract_MarketStatus(t *testing.T) {
	body, err := os.ReadFile("testdata/status.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})

	svc := newTestService(handler)
	statuses, _, err := svc.StatusHistory(context.Background())
	if err != nil {
		t.Fatalf("StatusHistory() error = %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("len = %d, want 2", len(statuses))
	}
	first, second := statuses[0], statuses[1]
	if first.Date.Unix() != 1704085200 || first.Status != "open" || !first.Open {
		t.Errorf("first = %+v, does not match fixture", first)
	}
	if second.Date.Unix() != 1704171600 || second.Status != "closed" || second.Open {
		t.Errorf("second = %+v, does not match fixture", second)
	}
}
