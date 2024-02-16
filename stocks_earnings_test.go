package client

import (
	"fmt"
	"regexp"
	"time"
)

func ExampleStockEarningsRequest_raw() {
	ser, err := StockEarnings().Symbol("AAPL").From("2022-01-01").To("2022-01-31").Raw()
	if err != nil {
		fmt.Print(err)
		return
	}

	// Convert the response to a string if it's not already
	serString := ser.String()

	// Use regex to remove the "updated" key and its value so that the output is consistent between runs.
	re := regexp.MustCompile(`,"updated":\[\d+\]`)
	cleanedSerString := re.ReplaceAllString(serString, "")

	fmt.Println(cleanedSerString)
	// Output: {"s":"ok","symbol":["AAPL"],"fiscalYear":[2022],"fiscalQuarter":[1],"date":[1640926800],"reportDate":[1643259600],"reportTime":["after close"],"currency":["USD"],"reportedEPS":[2.1],"estimatedEPS":[1.89],"surpriseEPS":[0.21],"surpriseEPSpct":[0.1111]}
}

func ExampleStockEarningsRequest_packed() {
	ser, err := StockEarnings().Symbol("AAPL").From("2022-01-01").To("2022-01-31").Packed()
	if err != nil {
		fmt.Print(err)
		return
	}

	ser.Updated = []int64{} // Delete the updated field so the string output does not change between runs.

	fmt.Println(ser)
	// Output: StockEarningsResponse{Symbol: [AAPL], FiscalYear: 2022, FiscalQuarter: 1, Date: [1640926800], ReportDate: [1643259600], ReportTime: [after close], Currency: [USD], ReportedEPS: [2.100000 ], EstimatedEPS: [1.890000 ], SurpriseEPS: [0.210000 ], SurpriseEPSpct: [0.111100 ], Updated: []}
}

func ExampleStockEarningsRequest_get() {
	ser, err := StockEarnings().Symbol("AAPL").From("2022-01-01").To("2022-01-31").Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	for _, report := range ser {
		report.Updated = time.Time{}
		fmt.Println(report)
	}
	// Output: StockEarningsReport{Symbol: "AAPL", FiscalYear: 2022, FiscalQuarter: 1, Date: "2021-12-31", ReportDate: "2022-01-27", ReportTime: "after close", Currency: "USD", ReportedEPS: 2.100000, EstimatedEPS: 1.890000, SurpriseEPS: 0.210000, SurpriseEPSPct: 0.111100, Updated: "nil"}
}
