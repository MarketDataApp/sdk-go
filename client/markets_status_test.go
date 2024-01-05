package client

import (
	"testing"
)

func TestMarketStatusRequestSetters(t *testing.T) {
	msr := MarketStatus()

	// Test Country setter
	msr.Country("UK")
	if msr.countryParams.Country != "UK" || msr.Error != nil {
		t.Errorf("Country setter failed, got: %s, want: %s.", msr.countryParams.Country, "UK")
	}

	// Test invalid Country setter
	msr.Country("UKK")
	if msr.Error == nil {
		t.Errorf("Country setter failed to catch invalid input.")
	}

	// Test Date setter
	msr.Date("2022-01-01")
	if msr.dateParams.Date != "2022-01-01" || msr.dateParams.From != "" || msr.dateParams.To != "" || msr.dateParams.Countback != nil {
		t.Errorf("Date setter failed, got: %s, want: %s.", msr.dateParams.Date, "2022-01-01")
	}

	// Test From setter
	msr.From("2022-01-01")
	if msr.dateParams.From != "2022-01-01" || msr.dateParams.Date != "" || msr.dateParams.Countback != nil {
		t.Errorf("From setter failed, got: %s, want: %s.", msr.dateParams.From, "2022-01-01")
	}

	// Test To setter
	msr.To("2022-12-31")
	if msr.dateParams.To != "2022-12-31" || msr.dateParams.Date != "" {
		t.Errorf("To setter failed, got: %s, want: %s.", msr.dateParams.To, "2022-12-31")
	}

	// Test Countback setter
	countback := 5
	msr.Countback(countback)
	if *msr.dateParams.Countback != countback || msr.dateParams.Date != "" || msr.dateParams.From != "" {
		t.Errorf("Countback setter failed, got: %d, want: %d.", *msr.dateParams.Countback, countback)
	}
}
