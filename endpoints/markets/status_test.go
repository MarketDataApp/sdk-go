package markets

import (
	"testing"
	"time"
)

func TestGetOpenDates(t *testing.T) {
	// Create a mock MarketStatusResponse
	msr := &MarketStatusResponse{
		Date:   []int64{time.Now().Unix(), time.Now().AddDate(0, 0, -1).Unix()},
		Status: &[]string{"open", "closed"},
	}

	openDates, err := msr.GetOpenDates()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(openDates) != 1 {
		t.Fatalf("Expected 1 open date, got %d", len(openDates))
	}

	if openDates[0].Day() != time.Now().Day() {
		t.Fatalf("Expected today's date, got %v", openDates[0])
	}
}

func TestGetClosedDates(t *testing.T) {
	// Create a mock MarketStatusResponse
	msr := &MarketStatusResponse{
		Date:   []int64{time.Now().Unix(), time.Now().AddDate(0, 0, -1).Unix()},
		Status: &[]string{"open", "closed"},
	}

	closedDates, err := msr.GetClosedDates()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(closedDates) != 1 {
		t.Fatalf("Expected 1 closed date, got %d", len(closedDates))
	}

	if closedDates[0].Day() != time.Now().AddDate(0, 0, -1).Day() {
		t.Fatalf("Expected yesterday's date, got %v", closedDates[0])
	}
}

func TestMarketStatusRequestSetters(t *testing.T) {
	msr, _ := New()

	// Test Country setter
	msr.Country("UK")
	if msr.Params.Country != "UK" || msr.Params.Err != nil {
		t.Errorf("Country setter failed, got: %s, want: %s.", msr.Params.Country, "UK")
	}

	// Test invalid Country setter
	msr.Country("UKK")
	if msr.Params.Err == nil {
		t.Errorf("Country setter failed to catch invalid input.")
	}

	// Test Date setter
	msr.Date("2022-01-01")
	if msr.Params.Date != "2022-01-01" || msr.Params.From != "" || msr.Params.To != "" || msr.Params.Countback != nil {
		t.Errorf("Date setter failed, got: %s, want: %s.", msr.Params.Date, "2022-01-01")
	}

	// Test From setter
	msr.From("2022-01-01")
	if msr.Params.From != "2022-01-01" || msr.Params.Date != "" || msr.Params.Countback != nil {
		t.Errorf("From setter failed, got: %s, want: %s.", msr.Params.From, "2022-01-01")
	}

	// Test To setter
	msr.To("2022-12-31")
	if msr.Params.To != "2022-12-31" || msr.Params.Date != "" {
		t.Errorf("To setter failed, got: %s, want: %s.", msr.Params.To, "2022-12-31")
	}

	// Test Countback setter
	countback := 5
	msr.Countback(countback)
	if *msr.Params.Countback != countback || msr.Params.Date != "" || msr.Params.From != "" {
		t.Errorf("Countback setter failed, got: %d, want: %d.", *msr.Params.Countback, countback)
	}
}