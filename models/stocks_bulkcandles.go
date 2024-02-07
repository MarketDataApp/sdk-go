package models

import (
	"fmt"
	"time"
)

// StockCandlesResponse represents the JSON response structure for stock candles data.
// It includes slices for time, open, high, low, close prices, and volume for each candle.
// Optional fields VWAP and N are available for V2 candles.
type BulkStockCandlesResponse struct {
	Symbol []string  `json:"symbol" human:"Symbol"` // Symbol holds the stock ticker of the candle.
	Date   []int64   `json:"t" human:"Date"`        // Date holds UNIX timestamps for each candle.
	Open   []float64 `json:"o" human:"Open"`        // Open holds the opening prices for each candle.
	High   []float64 `json:"h" human:"High"`        // High holds the highest prices reached in each candle.
	Low    []float64 `json:"l" human:"Low"`         // Low holds the lowest prices reached in each candle.
	Close  []float64 `json:"c" human:"Close"`       // Close holds the closing prices for each candle.
	Volume []int64   `json:"v" human:"Volume"`      // Volume represents the trading volume in each candle.
}

// Unpack converts a BulkStockCandlesResponse into a slice of Candle.
//
// This method iterates through the slices of stock candle data contained within the BulkStockCandlesResponse struct.
// It constructs a Candle struct for each set of corresponding data across the slices (symbol, date, open, high, low, close, volume).
// The method ensures that each Candle struct is populated with the appropriate data, converting the UNIX timestamp to a time.Time object for the Date field.
// It is crucial that the slices within BulkStockCandlesResponse are of equal length to maintain data integrity and avoid index out-of-range errors.
//
// Parameters:
//   - bscr *BulkStockCandlesResponse: A pointer to the BulkStockCandlesResponse instance containing the bulk stock candles data to be unpacked.
//
// Returns:
//   - []Candle: A slice of Candle structs, each representing a single stock candle with data unpacked from the BulkStockCandlesResponse.
//   - error: An error object that will be non-nil if the slices within BulkStockCandlesResponse are not of equal length, indicating a failure in the unpacking process.
func (bscr *BulkStockCandlesResponse) Unpack() ([]Candle, error) {
	var bulkStockCandles []Candle
	for i := range bscr.Date {
		bulkStockCandle := Candle{
			Symbol: bscr.Symbol[i],
			Date:   time.Unix(bscr.Date[i], 0),
			Open:   bscr.Open[i],
			High:   bscr.High[i],
			Low:    bscr.Low[i],
			Close:  bscr.Close[i],
			Volume: bscr.Volume[i],
		}
		bulkStockCandles = append(bulkStockCandles, bulkStockCandle)
	}
	return bulkStockCandles, nil
}

// String returns a string representation of the BulkStockCandlesResponse struct.
//
// This method formats the BulkStockCandlesResponse's fields (Symbol, Date, Open, High, Low, Close, Volume)
// into a single string. This is particularly useful for logging or debugging purposes, where a quick textual
// representation of the BulkStockCandlesResponse is needed. The Date field is represented as UNIX timestamps.
//
// Parameters:
//   - There are no parameters for this method as it operates on the BulkStockCandlesResponse instance it is called on.
//
// Returns:
//   - string: A formatted string that contains the Symbol, Date (as UNIX timestamps), Open, High, Low, Close, and Volume
//             fields of the BulkStockCandlesResponse.
func (bscr BulkStockCandlesResponse) String() string {
	return fmt.Sprintf("BulkStockCandlesResponse{Symbol: %v, Date: %v, Open: %v, High: %v, Low: %v, Close: %v, Volume: %v}", bscr.Symbol, bscr.Date, bscr.Open, bscr.High, bscr.Low, bscr.Close, bscr.Volume)
}
