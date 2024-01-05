package client

import (
	"fmt"
	"strings"
	"time"
)

type StockEarningsResponse struct {
	Symbol         []string   `json:"symbol"`
	FiscalYear     []int64    `json:"fiscalYear"`
	FiscalQuarter  []int64    `json:"fiscalQuarter"`
	Date           []int64   `json:"date"`
	ReportDate     []int64   `json:"reportDate"`
	ReportTime     []string   `json:"reportTime"`
	Currency       []string   `json:"currency"`
	ReportedEPS    []*float64 `json:"reportedEPS"`
	EstimatedEPS   []*float64 `json:"estimatedEPS"`
	SurpriseEPS    []*float64 `json:"surpriseEPS"`
	SurpriseEPSpct []*float64 `json:"surpriseEPSpct"`
	Updated        []int64    `json:"updated"`
}

type StockEarningsReport struct {
	Symbol         string
	FiscalYear     int64
	FiscalQuarter  int64
	Date           time.Time
	ReportDate     time.Time
	ReportTime     string
	Currency       string
	ReportedEPS    *float64
	EstimatedEPS   *float64
	SurpriseEPS    *float64
	SurpriseEPSpct *float64
	Updated        time.Time
}

func (ser StockEarningsReport) String() string {
	loc, _ := time.LoadLocation("America/New_York")

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

	return fmt.Sprintf("Symbol: %s, Fiscal Year: %v, Fiscal Quarter: %v, Date: %v, Report Date: %v, Report Time: %v, Currency: %v, Reported EPS: %v, Estimated EPS: %v, Surprise EPS: %v, Surprise EPS Pct: %v, Updated: %s",
		ser.Symbol, ser.FiscalYear, ser.FiscalQuarter, ser.Date.Format("2006-01-02"), ser.ReportDate.Format("2006-01-02"), ser.ReportTime, ser.Currency, reportedEPS, estimatedEPS, surpriseEPS, surpriseEPSpct, ser.Updated.In(loc).Format("2006-01-02 15:04:05 Z07:00"))
}

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

func (ser *StockEarningsResponse) String() string {
	var result strings.Builder

	fmt.Fprintf(&result, "Symbol: %v, Fiscal Year: %v, Fiscal Quarter: %v, Date: %v, Report Date: %v, Report Time: %v, Currency: %v, ",
		ser.Symbol, ser.FiscalYear, ser.FiscalQuarter, ser.Date, ser.ReportDate, ser.ReportTime, ser.Currency)

	fmt.Fprintf(&result, "Reported EPS: [")
	for _, v := range ser.ReportedEPS {
		if v != nil {
			fmt.Fprintf(&result, "%f ", *v)
		} else {
			fmt.Fprintf(&result, "nil ")
		}
	}
	fmt.Fprintf(&result, "], ")

	fmt.Fprintf(&result, "Estimated EPS: [")
	for _, v := range ser.EstimatedEPS {
		if v != nil {
			fmt.Fprintf(&result, "%f ", *v)
		} else {
			fmt.Fprintf(&result, "nil ")
		}
	}
	fmt.Fprintf(&result, "], ")

	fmt.Fprintf(&result, "Surprise EPS: [")
	for _, v := range ser.SurpriseEPS {
		if v != nil {
			fmt.Fprintf(&result, "%f ", *v)
		} else {
			fmt.Fprintf(&result, "nil ")
		}
	}
	fmt.Fprintf(&result, "], ")

	fmt.Fprintf(&result, "Surprise EPS Pct: [")
	for _, v := range ser.SurpriseEPSpct {
		if v != nil {
			fmt.Fprintf(&result, "%f ", *v)
		} else {
			fmt.Fprintf(&result, "nil ")
		}
	}
	fmt.Fprintf(&result, "], ")

	fmt.Fprintf(&result, "Updated: %v", ser.Updated)

	return result.String()
}