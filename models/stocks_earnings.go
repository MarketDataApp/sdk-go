package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/MarketDataApp/sdk-go/helpers/dates"
)

// StockEarningsResponse encapsulates the data received as a response to a stock earnings data request.
// It contains detailed information about stock earnings, including symbols, fiscal years, fiscal quarters,
// dates, report dates, report times, currencies, reported EPS, estimated EPS, surprise EPS, surprise EPS
// percentage, and last update times.
//
// # Generated By
//
//   - StockEarningsRequest.Packed(): Generates this struct as a response to a stock earnings request.
//
// # Methods
//
//   - String() string: Returns a string representation of the StockEarningsResponse.
//   - Unpack() ([]StockEarningsReport, error): Converts the response into a slice of StockEarningsReport structs.
//
// # Notes
//
//   - The dates in this struct are represented as UNIX timestamps.
//   - EPS fields may be nil if the data is not available.
type StockEarningsResponse struct {
	Symbol         []string   `json:"symbol"`         // Symbol represents the stock symbols.
	FiscalYear     []int64    `json:"fiscalYear"`     // FiscalYear represents the fiscal years of the earnings report.
	FiscalQuarter  []int64    `json:"fiscalQuarter"`  // FiscalQuarter represents the fiscal quarters of the earnings report.
	Date           []int64    `json:"date"`           // Date represents the earnings announcement dates in UNIX timestamp.
	ReportDate     []int64    `json:"reportDate"`     // ReportDate represents the report release dates in UNIX timestamp.
	ReportTime     []string   `json:"reportTime"`     // ReportTime represents the time of day the earnings were reported.
	Currency       []string   `json:"currency"`       // Currency represents the currency used in the earnings report.
	ReportedEPS    []*float64 `json:"reportedEPS"`    // ReportedEPS represents the actual earnings per share reported.
	EstimatedEPS   []*float64 `json:"estimatedEPS"`   // EstimatedEPS represents the consensus earnings per share estimate.
	SurpriseEPS    []*float64 `json:"surpriseEPS"`    // SurpriseEPS represents the difference between reported EPS and estimated EPS.
	SurpriseEPSpct []*float64 `json:"surpriseEPSpct"` // SurpriseEPSpct represents the percentage difference between reported and estimated EPS.
	Updated        []int64    `json:"updated"`        // Updated represents the last update time in UNIX timestamp.
}

// StockEarningsReport represents a single earnings report for a stock, encapsulating details such as the stock symbol, fiscal year, fiscal quarter, earnings date, report date and time, currency used in the report, reported EPS, estimated EPS, surprise EPS, surprise EPS percentage, and the last update time.
//
// # Generated By
//
//   - Unpack(): Converts a StockEarningsResponse into a slice of StockEarningsReport structs.
//
// # Methods
//
//   - String() string: Returns a string representation of the StockEarningsReport.
//
// # Notes
//
//   - The Date, ReportDate, and Updated fields are represented as time.Time objects, providing a standardized format for date and time.
//   - EPS fields (ReportedEPS, EstimatedEPS, SurpriseEPS, SurpriseEPSpct) are pointers to float64, allowing for nil values to represent missing data.
type StockEarningsReport struct {
	Symbol         string    // Symbol represents the stock symbol.
	FiscalYear     int64     // FiscalYear represents the fiscal year of the earnings report.
	FiscalQuarter  int64     // FiscalQuarter represents the fiscal quarter of the earnings report.
	Date           time.Time // Date represents the earnings announcement date.
	ReportDate     time.Time // ReportDate represents the report release date.
	ReportTime     string    // ReportTime represents the time of day the earnings were reported.
	Currency       string    // Currency represents the currency used in the earnings report.
	ReportedEPS    *float64  // ReportedEPS represents the actual earnings per share reported.
	EstimatedEPS   *float64  // EstimatedEPS represents the consensus earnings per share estimate.
	SurpriseEPS    *float64  // SurpriseEPS represents the difference between reported EPS and estimated EPS.
	SurpriseEPSpct *float64  // SurpriseEPSpct represents the percentage difference between reported and estimated EPS.
	Updated        time.Time // Updated represents the last update time.
}

// String returns a string representation of the StockEarningsReport struct. This method is primarily used for logging or debugging purposes, allowing developers to easily view the contents of a StockEarningsReport instance in a human-readable format. The method formats various fields of the struct, including handling nil pointers for EPS values gracefully by displaying them as "nil".
//
// # Returns
//
//   - string: A formatted string that represents the StockEarningsReport instance, including all its fields. Fields that are pointers to float64 (ReportedEPS, EstimatedEPS, SurpriseEPS, SurpriseEPSpct) are displayed as their dereferenced values if not nil, otherwise "nil".
//
// # Notes
//
//   - The Date, ReportDate, and Updated fields are formatted as "YYYY-MM-DD" for consistency and readability.
//   - This method ensures that nil pointer fields do not cause a panic by checking their existence before dereferencing.
func (ser StockEarningsReport) String() string {
	reportedEPS := "nil"
	if ser.ReportedEPS != nil {
		reportedEPS = fmt.Sprintf("%f", *ser.ReportedEPS)
	}

	estimatedEPS := "nil"
	if ser.EstimatedEPS != nil {
		estimatedEPS = fmt.Sprintf("%f", *ser.EstimatedEPS)
	}

	surpriseEPS := "nil"
	if ser.SurpriseEPS != nil {
		surpriseEPS = fmt.Sprintf("%f", *ser.SurpriseEPS)
	}

	surpriseEPSpct := "nil"
	if ser.SurpriseEPSpct != nil {
		surpriseEPSpct = fmt.Sprintf("%f", *ser.SurpriseEPSpct)
	}

	return fmt.Sprintf("StockEarningsReport{Symbol: %q, FiscalYear: %d, FiscalQuarter: %d, Date: %q, ReportDate: %q, ReportTime: %q, Currency: %q, ReportedEPS: %s, EstimatedEPS: %s, SurpriseEPS: %s, SurpriseEPSPct: %s, Updated: %q}",
		ser.Symbol, ser.FiscalYear, ser.FiscalQuarter, ser.Date.Format("2006-01-02"), dates.TimeString(ser.ReportDate), ser.ReportTime, ser.Currency, reportedEPS, estimatedEPS, surpriseEPS, surpriseEPSpct, dates.TimeString(ser.Updated))
}

// Unpack converts the StockEarningsResponse struct into a slice of StockEarningsReport structs.
//
// This method iterates over the fields of a StockEarningsResponse struct, creating a StockEarningsReport struct for each symbol present in the response. It then populates the fields of each StockEarningsReport struct with the corresponding data from the StockEarningsResponse struct. The method handles the conversion of Unix timestamps to time.Time objects for the Date, ReportDate, and Updated fields. It also directly assigns pointer fields for ReportedEPS, EstimatedEPS, SurpriseEPS, and SurpriseEPSpct to handle potential nil values gracefully.
//
// # Returns
//
//   - []StockEarningsReport: A slice of StockEarningsReport structs constructed from the StockEarningsResponse.
//   - error: An error if the unpacking process fails, nil otherwise.
func (ser *StockEarningsResponse) Unpack() ([]StockEarningsReport, error) {
	var stockEarningsReports []StockEarningsReport
	for i := range ser.Symbol {
		stockEarningsReport := StockEarningsReport{
			Symbol:         ser.Symbol[i],
			FiscalYear:     ser.FiscalYear[i],
			FiscalQuarter:  ser.FiscalQuarter[i],
			Date:           time.Unix(ser.Date[i], 0),
			ReportDate:     time.Unix(ser.ReportDate[i], 0),
			ReportTime:     ser.ReportTime[i],
			Currency:       ser.Currency[i],
			ReportedEPS:    ser.ReportedEPS[i],
			EstimatedEPS:   ser.EstimatedEPS[i],
			SurpriseEPS:    ser.SurpriseEPS[i],
			SurpriseEPSpct: ser.SurpriseEPSpct[i],
			Updated:        time.Unix(ser.Updated[i], 0),
		}
		stockEarningsReports = append(stockEarningsReports, stockEarningsReport)
	}
	return stockEarningsReports, nil
}

// String generates a string representation of the StockEarningsResponse.
//
// This method formats the StockEarningsResponse fields into a readable string, including handling nil values for EPS fields and empty values for fiscalYear and fiscalQuarter gracefully by displaying them as "nil" or "empty" respectively.
//
// # Returns
//
//   - A string representation of the StockEarningsResponse.
func (ser *StockEarningsResponse) String() string {
	var result strings.Builder

	fiscalYear := "nil"
	if len(ser.FiscalYear) > 0 {
		fiscalYearValues := make([]string, len(ser.FiscalYear))
		for i, v := range ser.FiscalYear {
			fiscalYearValues[i] = fmt.Sprintf("%d", v)
		}
		fiscalYear = strings.Join(fiscalYearValues, ", ")
	}

	fiscalQuarter := "nil"
	if len(ser.FiscalQuarter) > 0 {
		fiscalQuarterValues := make([]string, len(ser.FiscalQuarter))
		for i, v := range ser.FiscalQuarter {
			fiscalQuarterValues[i] = fmt.Sprintf("%d", v)
		}
		fiscalQuarter = strings.Join(fiscalQuarterValues, ", ")
	}

	fmt.Fprintf(&result, "StockEarningsResponse{Symbol: %v, FiscalYear: %v, FiscalQuarter: %v, Date: %v, ReportDate: %v, ReportTime: %v, Currency: %v, ",
		ser.Symbol, fiscalYear, fiscalQuarter, ser.Date, ser.ReportDate, ser.ReportTime, ser.Currency)

	fmt.Fprintf(&result, "ReportedEPS: [")
	for _, v := range ser.ReportedEPS {
		if v != nil {
			fmt.Fprintf(&result, "%f ", *v)
		} else {
			fmt.Fprintf(&result, "nil ")
		}
	}
	fmt.Fprintf(&result, "], ")

	fmt.Fprintf(&result, "EstimatedEPS: [")
	for _, v := range ser.EstimatedEPS {
		if v != nil {
			fmt.Fprintf(&result, "%f ", *v)
		} else {
			fmt.Fprintf(&result, "nil ")
		}
	}
	fmt.Fprintf(&result, "], ")

	fmt.Fprintf(&result, "SurpriseEPS: [")
	for _, v := range ser.SurpriseEPS {
		if v != nil {
			fmt.Fprintf(&result, "%f ", *v)
		} else {
			fmt.Fprintf(&result, "nil ")
		}
	}
	fmt.Fprintf(&result, "], ")

	fmt.Fprintf(&result, "SurpriseEPSpct: [")
	for _, v := range ser.SurpriseEPSpct {
		if v != nil {
			fmt.Fprintf(&result, "%f ", *v)
		} else {
			fmt.Fprintf(&result, "nil ")
		}
	}
	fmt.Fprintf(&result, "], ")

	fmt.Fprintf(&result, "Updated: %v}", ser.Updated)

	return result.String()
}
