//go:build integration

package integration

import (
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/markets"
)

func TestMarkets_Status(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	status, _, err := client.Markets.Status(ctx)
	assertNoError(t, err, "Markets Status")
	assertNotNil(t, status, "status should not be nil")

	// Status should be one of the expected values
	validStatuses := []string{"open", "closed", "early-close"}
	found := false
	for _, valid := range validStatuses {
		if status.Status == valid {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Unexpected market status: %s", status.Status)
	}

	// Open boolean should match status string
	if status.Status == "open" && !status.Open {
		t.Error("Status is 'open' but Open is false")
	}
	if status.Status == "closed" && status.Open {
		t.Error("Status is 'closed' but Open is true")
	}
}

func TestMarkets_Status_HistoricalDate(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	// Use a known trading day
	date := historicalDate()

	status, _, err := client.Markets.Status(ctx, markets.WithDate(date))
	assertNoError(t, err, "Markets Status with historical date")
	assertNotNil(t, status, "status should not be nil")

	// Status should be one of the expected values
	validStatuses := []string{"open", "closed", "early-close"}
	found := false
	for _, valid := range validStatuses {
		if status.Status == valid {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Unexpected market status: %s", status.Status)
	}
}

func TestMarkets_Status_Weekend(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	// Find a Sunday - markets should be closed
	// Start from a known date and find the next Sunday
	date := time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC) // This is a Sunday

	status, _, err := client.Markets.Status(ctx, markets.WithDate(date))
	assertNoError(t, err, "Markets Status for weekend")
	assertNotNil(t, status, "status should not be nil")

	// Weekend should be closed
	if status.Open {
		t.Error("Market should be closed on Sunday")
	}
	assertEqual(t, status.Status, "closed", "weekend status should be closed")
}

func TestMarkets_StatusHistory(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	// Get history for January 2024
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	history, _, err := client.Markets.StatusHistory(ctx,
		markets.WithHistoryWindow(markets.Between(from, to)),
	)
	assertNoError(t, err, "Markets StatusHistory")
	assertNotEmpty(t, history, "status history should not be empty")

	// January 2024 should have ~20-21 trading days
	openDays := 0
	for _, s := range history {
		if s.Open {
			openDays++
		}

		// Each entry should have a valid status
		validStatuses := []string{"open", "closed", "early-close"}
		found := false
		for _, valid := range validStatuses {
			if s.Status == valid {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Unexpected status in history: %s", s.Status)
		}
	}

	// Should have approximately 20-21 trading days in January
	if openDays < 15 || openDays > 23 {
		t.Errorf("Expected ~20 open days in January, got %d", openDays)
	}
}

func TestMarkets_StatusHistory_DatesSorted(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	history, _, err := client.Markets.StatusHistory(ctx,
		markets.WithHistoryWindow(markets.Between(from, to)),
	)
	assertNoError(t, err, "Markets StatusHistory")
	assertNotEmpty(t, history, "status history should not be empty")

	// Dates should be sorted
	times := make([]time.Time, len(history))
	for i, s := range history {
		times[i] = s.Date
	}
	assertTimeSorted(t, times, "status history dates")
}

func TestMarkets_StatusHistory_WithCountback(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	history, _, err := client.Markets.StatusHistory(ctx,
		markets.WithHistoryWindow(markets.LastN(5)),
	)
	assertNoError(t, err, "Markets StatusHistory with countback")
	assertNotEmpty(t, history, "status history should not be empty")

	// The live API returns one row more than requested (observed:
	// countback=5 yields 6 rows, apparently including the current day).
	// The SDK passes the parameter through verbatim, so tolerate the
	// extra row here.
	if len(history) > 6 {
		t.Errorf("Expected at most countback+1 results with countback=5, got %d", len(history))
	}
}

func TestMarkets_Status_WithCountry(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	// Request US market status explicitly
	status, _, err := client.Markets.Status(ctx, markets.WithCountry("US"))
	assertNoError(t, err, "Markets Status with country")
	assertNotNil(t, status, "status should not be nil")

	// Should get a valid status
	validStatuses := []string{"open", "closed", "early-close"}
	found := false
	for _, valid := range validStatuses {
		if status.Status == valid {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Unexpected market status: %s", status.Status)
	}
}
