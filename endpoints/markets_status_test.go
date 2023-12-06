package endpoints

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
	msr := MarketStatus()

	// Test Country setter
	msr.Country("UK")
	if msr.ParamCountry != "UK" || msr.Error != nil {
		t.Errorf("Country setter failed, got: %s, want: %s.", msr.ParamCountry, "UK")
	}

	// Test invalid Country setter
	msr.Country("UKK")
	if msr.Error == nil {
		t.Errorf("Country setter failed to catch invalid input.")
	}

	// Test Date setter
	msr.Date("2022-01-01")
	if msr.DateParams.Date != "2022-01-01" || msr.DateParams.From != "" || msr.DateParams.To != "" || msr.DateParams.Countback != nil {
		t.Errorf("Date setter failed, got: %s, want: %s.", msr.DateParams.Date, "2022-01-01")
	}

	// Test From setter
	msr.From("2022-01-01")
	if msr.DateParams.From != "2022-01-01" || msr.DateParams.Date != "" || msr.DateParams.Countback != nil {
		t.Errorf("From setter failed, got: %s, want: %s.", msr.DateParams.From, "2022-01-01")
	}

	// Test To setter
	msr.To("2022-12-31")
	if msr.DateParams.To != "2022-12-31" || msr.DateParams.Date != "" {
		t.Errorf("To setter failed, got: %s, want: %s.", msr.DateParams.To, "2022-12-31")
	}

	// Test Countback setter
	countback := 5
	msr.Countback(countback)
	if *msr.DateParams.Countback != countback || msr.DateParams.Date != "" || msr.DateParams.From != "" {
		t.Errorf("Countback setter failed, got: %d, want: %d.", *msr.DateParams.Countback, countback)
	}
}