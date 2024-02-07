package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Candle represents a single candle in a stock candlestick chart.
// It includes the time, open, high, low, close prices, volume, and optionally symbol, VWAP and number of trades.
type Candle struct {
	Symbol string    `json:"symbol,omitempty"` // The symbol of the candle.
	Date   time.Time `json:"t"`                // Date represents the date and time of the candle.
	Open   float64   `json:"o"`                // Open is the opening price of the candle.
	High   float64   `json:"h"`                // High is the highest price reached during the candle's time.
	Low    float64   `json:"l"`                // Low is the lowest price reached during the candle's time.
	Close  float64   `json:"c"`                // Close is the closing price of the candle.
	Volume int64     `json:"v,omitempty"`      // Volume represents the trading volume during the candle's time.
	VWAP   float64   `json:"vwap,omitempty"`   // VWAP is the Volume Weighted Average Price, optional.
	N      int64     `json:"n,omitempty"`      // N is the number of trades that occurred, optional.
}

// String returns a string representation of a Candle.
//
// Returns:
//   - A string representation of the Candle.
func (c Candle) String() string {
	loc, _ := time.LoadLocation("America/New_York")
	var parts []string
	if c.Symbol != "" {
		parts = append(parts, fmt.Sprintf("Symbol: %s", c.Symbol))
	}
	timePart := c.Date.In(loc).Format("2006-01-02 15:04:05 Z07:00")
	parts = append(parts, fmt.Sprintf("Time: %s", timePart))
	parts = append(parts, fmt.Sprintf("Open: %v", c.Open))
	parts = append(parts, fmt.Sprintf("High: %v", c.High))
	parts = append(parts, fmt.Sprintf("Low: %v", c.Low))
	parts = append(parts, fmt.Sprintf("Close: %v", c.Close))
	if c.Volume != 0 {
		parts = append(parts, fmt.Sprintf("Volume: %v", c.Volume))
	}
	if c.VWAP != 0 {
		parts = append(parts, fmt.Sprintf("VWAP: %v", c.VWAP))
	}
	if c.N != 0 {
		parts = append(parts, fmt.Sprintf("N: %v", c.N))
	}
	return "Candle{" + strings.Join(parts, ", ") + "}"
}

func (c Candle) Equals(other Candle) bool {
	return c.Symbol == other.Symbol &&
		c.Date.Equal(other.Date) &&
		c.Open == other.Open &&
		c.High == other.High &&
		c.Low == other.Low &&
		c.Close == other.Close &&
		c.Volume == other.Volume &&
		c.VWAP == other.VWAP &&
		c.N == other.N
}

// MarshalJSON customizes the JSON output of Candle.
func (c Candle) MarshalJSON() ([]byte, error) {
	// Define a struct that matches Candle but with Date as an integer (Unix timestamp).
	type Alias Candle
	return json.Marshal(&struct {
		Date int64 `json:"t"` // Convert Date to Unix timestamp.
		*Alias
	}{
		Date:  c.Date.Unix(), // Convert time.Time to Unix timestamp.
		Alias: (*Alias)(&c),
	})
}

// UnmarshalJSON customizes the JSON input processing of Candle.
func (c *Candle) UnmarshalJSON(data []byte) error {
	// Define a struct that matches Candle but with Date as an integer (Unix timestamp).
	type Alias Candle
	aux := &struct {
		Date int64 `json:"t"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	c.Date = time.Unix(aux.Date, 0) // Convert Unix timestamp back to time.Time.
	return nil
}

// ByDate implements sort.Interface for []Candle based on the Date field.
type ByDate []Candle

func (a ByDate) Len() int           { return len(a) }
func (a ByDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByDate) Less(i, j int) bool { return a[i].Date.Before(a[j].Date) }

// ByVolume implements sort.Interface for []Candle based on the Volume field.
type ByVolume []Candle

func (a ByVolume) Len() int           { return len(a) }
func (a ByVolume) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByVolume) Less(i, j int) bool { return a[i].Volume < a[j].Volume }

// ByOpen implements sort.Interface for []Candle based on the Open field.
type ByOpen []Candle

func (a ByOpen) Len() int           { return len(a) }
func (a ByOpen) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByOpen) Less(i, j int) bool { return a[i].Open < a[j].Open }

// ByHigh implements sort.Interface for []Candle based on the High field.
type ByHigh []Candle

func (a ByHigh) Len() int           { return len(a) }
func (a ByHigh) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByHigh) Less(i, j int) bool { return a[i].High < a[j].High }

// ByLow implements sort.Interface for []Candle based on the Low field.
type ByLow []Candle

func (a ByLow) Len() int           { return len(a) }
func (a ByLow) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLow) Less(i, j int) bool { return a[i].Low < a[j].Low }

// ByClose implements sort.Interface for []Candle based on the Close field.
type ByClose []Candle

func (a ByClose) Len() int           { return len(a) }
func (a ByClose) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByClose) Less(i, j int) bool { return a[i].Close < a[j].Close }

// ByVWAP implements sort.Interface for []Candle based on the VWAP field.
type ByVWAP []Candle

func (a ByVWAP) Len() int           { return len(a) }
func (a ByVWAP) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByVWAP) Less(i, j int) bool { return a[i].VWAP < a[j].VWAP }

// ByN implements sort.Interface for []Candle based on the N field.
type ByN []Candle

func (a ByN) Len() int           { return len(a) }
func (a ByN) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByN) Less(i, j int) bool { return a[i].N < a[j].N }

// BySymbol implements sort.Interface for []Candle based on the Symbol field.
// Candles are sorted in ascending order.
type BySymbol []Candle

func (a BySymbol) Len() int           { return len(a) }
func (a BySymbol) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySymbol) Less(i, j int) bool { return a[i].Symbol < a[j].Symbol }

func (c Candle) Clone() Candle {
	return Candle{
		Symbol: c.Symbol,
		Date:   c.Date,
		Open:   c.Open,
		High:   c.High,
		Low:    c.Low,
		Close:  c.Close,
		Volume: c.Volume,
		VWAP:   c.VWAP,
		N:      c.N,
	}
}

// IsBefore checks if the current Candle instance occurred before the specified Candle.
//
// Parameters:
//   - other: A Candle instance to compare with the current Candle instance.
//
// Returns:
//   - bool: Returns true if the current Candle's date is before the 'other' Candle's date; otherwise, false.
func (c Candle) IsBefore(other Candle) bool {
	return c.Date.Before(other.Date)
}

// IsAfter checks if the current Candle instance occurred after the specified Candle.
//
// Parameters:
//   - other: A Candle instance to compare with the current Candle instance.
//
// Returns:
//   - bool: Returns true if the current Candle's date is after the 'other' Candle's date; otherwise, false.
func (c Candle) IsAfter(other Candle) bool {
	return c.Date.After(other.Date)
}

// IsValid checks the validity of the Candle based on its financial data.
//
// A Candle is considered valid if:
//   - The high price is greater than or equal to the low price.
//   - The open price is greater than or equal to the low price.
//   - The close price is greater than or equal to the low price.
//   - The high price is greater than or equal to the open price.
//   - The high price is greater than or equal to the close price.
//   - The volume is non-negative.
//
// Returns:
//   - bool: Returns true if the Candle meets all the validity criteria; otherwise, false.
func (c Candle) IsValid() bool {
	return c.High >= c.Low && c.Open >= c.Low && c.Close >= c.Low && c.High >= c.Open && c.High >= c.Close && c.Volume >= 0
}
