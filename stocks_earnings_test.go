package client

import (
	"fmt"
	"time"
)

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
