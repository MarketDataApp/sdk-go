package client

import (
	"fmt"
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

func ExampleMarketStatus_packed() {

	msr, err := MarketStatus().From("2022-01-01").To("2022-01-10").Packed()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(msr)
	//Output: MarketStatusResponse{Date: [1641013200, 1641099600, 1641186000, 1641272400, 1641358800, 1641445200, 1641531600, 1641618000, 1641704400, 1641790800], Status: ["closed", "closed", "open", "open", "open", "open", "open", "closed", "closed", "open"]}
}

func ExampleMarketStatus_get() {

	msr, err := MarketStatus().From("2022-01-01").To("2022-01-10").Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	for _, report := range msr {
		fmt.Println(report)
	}
	// Output: MarketStatusReport{Date: 2022-01-01, Open: false, Closed: true}
	// MarketStatusReport{Date: 2022-01-02, Open: false, Closed: true}
	// MarketStatusReport{Date: 2022-01-03, Open: true, Closed: false}
	// MarketStatusReport{Date: 2022-01-04, Open: true, Closed: false}
	// MarketStatusReport{Date: 2022-01-05, Open: true, Closed: false}
	// MarketStatusReport{Date: 2022-01-06, Open: true, Closed: false}
	// MarketStatusReport{Date: 2022-01-07, Open: true, Closed: false}
	// MarketStatusReport{Date: 2022-01-08, Open: false, Closed: true}
	// MarketStatusReport{Date: 2022-01-09, Open: false, Closed: true}
	// MarketStatusReport{Date: 2022-01-10, Open: true, Closed: false}
}
