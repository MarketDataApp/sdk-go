// Package funds provides types and methods for mutual fund data from
// the Market Data API's /v1/funds/candles/ endpoint. It exposes
// historical net asset value (NAV) candles for mutual funds through
// [Service.Candles], with the requested timeframe controlled by
// [Resolution] and the date range selected with [WithCandleWindow] using
// a single [DateWindow] value (for example [OnDate], [Between], [Since],
// [Until], [LastN], or [LastNUntil]).
//
// See https://www.marketdata.app/docs/api/funds/candles for the
// endpoint's API documentation.
//
// # Zero Values and Null Data
//
// Numeric fields use Go value types rather than pointers. When the API
// returns null for a field, it is unmarshaled as the zero value. A zero
// value may represent either an actual zero or the absence of data.
// Timestamp fields are the exception: a null or absent timestamp decodes
// to the zero time.Time, so IsZero reliably detects missing data.
package funds

import (
	"fmt"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/timezone"
)

// Resolution represents the candle resolution/timeframe.
type Resolution string

const (
	// ResolutionDaily is daily candles
	ResolutionDaily Resolution = "D"

	// ResolutionWeekly is weekly candles
	ResolutionWeekly Resolution = "W"

	// ResolutionMonthly is monthly candles
	ResolutionMonthly Resolution = "M"

	// ResolutionYearly is yearly candles. The funds candles endpoint supports
	// daily, weekly, monthly, and yearly only (quarterly is not accepted).
	ResolutionYearly Resolution = "Y"
)

// String returns the string representation of the resolution.
func (r Resolution) String() string {
	return string(r)
}

// Candle represents an OHLCV candle for a fund.
type Candle struct {
	// Time is the candle timestamp
	Time time.Time `json:"t"`

	// Open is the opening price (NAV)
	Open float64 `json:"o"`

	// High is the highest price
	High float64 `json:"h"`

	// Low is the lowest price
	Low float64 `json:"l"`

	// Close is the closing price (NAV)
	Close float64 `json:"c"`
}

// String returns a concise summary of the candle.
func (c Candle) String() string {
	return fmt.Sprintf("%s O: %.2f H: %.2f L: %.2f C: %.2f", c.Time.Format("2006-01-02"), c.Open, c.High, c.Low, c.Close)
}

// Range returns the high-low range.
func (c *Candle) Range() float64 {
	return c.High - c.Low
}

// candlesResponse is the API response for fund candles.
type candlesResponse struct {
	Status string    `json:"s"`
	Time   []int64   `json:"t"` // Unix timestamps
	Open   []float64 `json:"o"`
	High   []float64 `json:"h"`
	Low    []float64 `json:"l"`
	Close  []float64 `json:"c"`
}

// RequiredColumns names the timestamp array toCandles takes its row count
// from. See http.ColumnRequirer.
func (r *candlesResponse) RequiredColumns() []string { return []string{"t"} }

func (r *candlesResponse) toCandles() []Candle {
	if r == nil || len(r.Time) == 0 {
		return nil
	}

	candles := make([]Candle, len(r.Time))
	for i := range r.Time {
		candles[i] = Candle{
			Time:  timezone.ToEastern(r.Time[i]),
			Open:  safeIndex(r.Open, i),
			High:  safeIndex(r.High, i),
			Low:   safeIndex(r.Low, i),
			Close: safeIndex(r.Close, i),
		}
	}
	return candles
}

func safeIndex(s []float64, i int) float64 {
	if i < len(s) {
		return s[i]
	}
	return 0
}
