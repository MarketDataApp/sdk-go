package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/MarketDataApp/sdk-go/helpers/dates"
)

// IndexQuotesResponse encapsulates the data for index quotes, including symbols, last prices, price changes, percentage changes, 52-week highs and lows, and update timestamps.
//
// # Generated By
//
//   - IndexQuoteRequest.Packed(): Fetches index quotes and returns them in an IndexQuotesResponse struct.
//
// # Methods
//
//   - Unpack() ([]IndexQuote, error): Transforms the IndexQuotesResponse into a slice of IndexQuote for individual processing.
//   - String() string: Provides a string representation of the IndexQuotesResponse for logging or debugging.
//
// # Notes
//
//   - The Change and ChangePct fields are pointers to accommodate nil values, indicating that the change information is not applicable or unavailable.
//   - The High52 and Low52 fields are pointers to slices, allowing for the entire field to be nil if 52-week high/low data is not applicable or unavailable.
type IndexQuotesResponse struct {
	Symbol    []string   `json:"symbol"`               // Symbols are the stock symbols or tickers.
	Last      []float64  `json:"last"`                 // Last contains the last traded prices.
	Change    []*float64 `json:"change,omitempty"`     // Change represents the change in price, can be nil if not applicable.
	ChangePct []*float64 `json:"changepct,omitempty"`  // ChangePct represents the percentage change in price, can be nil if not applicable.
	High52    *[]float64 `json:"52weekHigh,omitempty"` // High52 points to a slice of 52-week high prices, can be nil if not applicable.
	Low52     *[]float64 `json:"52weekLow,omitempty"`  // Low52 points to a slice of 52-week low prices, can be nil if not applicable.
	Updated   []int64    `json:"updated"`              // Updated contains timestamps of the last updates.
}

// IndexQuote represents a single quote for an index, encapsulating details such as the symbol, last traded price, price changes, 52-week high and low prices, volume, and the timestamp of the last update.
//
// # Generated By
//
//   - IndexQuotesResponse.Unpack(): Generates IndexQuote instances from an IndexQuotesResponse.
//
// # Methods
//
//   - String() string: Provides a string representation of the IndexQuote.
//
// # Notes
//
//   - The Change and ChangePct fields are pointers to accommodate nil values, indicating that the change information is not applicable or unavailable.
//   - The High52 and Low52 fields are also pointers, allowing these fields to be nil if 52-week high/low data is not applicable or unavailable.
type IndexQuote struct {
	Symbol    string    // Symbol is the stock symbol or ticker.
	Last      float64   // Last is the last traded price.
	Change    *float64  // Change represents the change in price, can be nil if not applicable.
	ChangePct *float64  // ChangePct represents the percentage change in price, can be nil if not applicable.
	High52    *float64  // High52 is the 52-week high price, can be nil if not applicable.
	Low52     *float64  // Low52 is the 52-week low price, can be nil if not applicable.
	Volume    int64     // Volume is the number of shares traded.
	Updated   time.Time // Updated is the timestamp of the last update.
}

// String generates a string representation of the IndexQuote for easy human-readable display. It is useful for logging,
// debugging, or displaying the IndexQuote details, including symbol, last traded price, volume, update timestamp,
// and optionally, 52-week highs, lows, and price changes if available.
//
// # Returns
//
//   - string: A formatted string encapsulating the IndexQuote details. It includes the symbol, last price, volume, update timestamp, and, if not nil, 52-week highs, lows, change, and percentage change.
func (iq IndexQuote) String() string {
	high52 := "nil"
	if iq.High52 != nil {
		high52 = fmt.Sprintf("%v", *iq.High52)
	}
	low52 := "nil"
	if iq.Low52 != nil {
		low52 = fmt.Sprintf("%v", *iq.Low52)
	}
	change := "nil"
	if iq.Change != nil {
		change = fmt.Sprintf("%v", *iq.Change)
	}
	changePct := "nil"
	if iq.ChangePct != nil {
		changePct = fmt.Sprintf("%v", *iq.ChangePct)
	}
	return fmt.Sprintf("IndexQuote{Symbol: %q, Last: %v, Volume: %v, Updated: %q, High52: %s, Low52: %s, Change: %s, ChangePct: %s}",
		iq.Symbol, iq.Last, iq.Volume, dates.TimeString(iq.Updated), high52, low52, change, changePct)
}

// Unpack transforms the IndexQuotesResponse into a slice of IndexQuote. This method is primarily used
// for converting a bulk response of index quotes into individual index quote objects, making them easier to work with
// in a program. It is useful when you need to process or display index quotes individually after fetching them in bulk.
//
// # Returns
//
//   - []IndexQuote: A slice of IndexQuote derived from the IndexQuotesResponse, allowing for individual access and manipulation of index quotes.
//   - error: An error if any issues occur during the unpacking process, enabling error handling in the calling function.
func (iqr *IndexQuotesResponse) Unpack() ([]IndexQuote, error) {
	var indexQuotes []IndexQuote
	for i := range iqr.Symbol {
		indexQuote := IndexQuote{
			Symbol:  iqr.Symbol[i],
			Last:    iqr.Last[i],
			Updated: time.Unix(iqr.Updated[i], 0),
		}
		if iqr.Change != nil && len(iqr.Change) > i {
			indexQuote.Change = iqr.Change[i]
		}
		if iqr.ChangePct != nil && len(iqr.ChangePct) > i {
			indexQuote.ChangePct = iqr.ChangePct[i]
		}
		if iqr.High52 != nil && len(*iqr.High52) > i {
			val := (*iqr.High52)[i]
			indexQuote.High52 = &val
		}
		if iqr.Low52 != nil && len(*iqr.Low52) > i {
			val := (*iqr.Low52)[i]
			indexQuote.Low52 = &val
		}
		indexQuotes = append(indexQuotes, indexQuote)
	}
	return indexQuotes, nil
}

// String provides a formatted string representation of the IndexQuotesResponse instance. This method is primarily used for
// logging or debugging purposes, allowing the user to easily view the contents of an IndexQuotesResponse object in a
// human-readable format. It concatenates various fields of the IndexQuotesResponse into a single string, making it
// easier to understand the response at a glance.
//
// # Returns
//
//   - string: A formatted string containing the contents of the IndexQuotesResponse.
func (iqr *IndexQuotesResponse) String() string {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("IndexQuotesResponse{Symbol: [%v], Last: [%v]", strings.Join(iqr.Symbol, ", "), joinFloat64Slice(iqr.Last)))

	if iqr.Change != nil {
		result.WriteString(fmt.Sprintf(", Change: [%v]", joinFloat64PointerSlice(iqr.Change)))
	}
	if iqr.ChangePct != nil {
		result.WriteString(fmt.Sprintf(", ChangePct: [%v]", joinFloat64PointerSlice(iqr.ChangePct)))
	}
	if iqr.High52 != nil {
		result.WriteString(fmt.Sprintf(", High52: [%v]", joinFloat64Slice(*iqr.High52)))
	}
	if iqr.Low52 != nil {
		result.WriteString(fmt.Sprintf(", Low52: [%v]", joinFloat64Slice(*iqr.Low52)))
	}

	result.WriteString(fmt.Sprintf(", Updated: [%v]}", joinInt64Slice(iqr.Updated)))

	return result.String()
}

// joinFloat64Slice converts a slice of float64 values into a single, comma-separated string.
// This method is primarily used for creating a human-readable string representation of numerical data,
// which can be useful for logging, debugging, or displaying data to end-users.

// # Parameters
//   - []float64 slice: The slice of float64 values to be joined.

// # Returns
//   - string: A comma-separated string representation of the input slice.
func joinFloat64Slice(slice []float64) string {
	strs := make([]string, len(slice))
	for i, v := range slice {
		strs[i] = fmt.Sprintf("%v", v)
	}
	return strings.Join(strs, ", ")
}

// joinFloat64PointerSlice joins a slice of *float64 into a single string, handling nil pointers gracefully.
//
// # Parameters:
//
//   - slice: The slice of *float64 to be joined.
//
// # Returns:
//
//   - A string representation of the joined slice, with "nil" for nil pointers.
func joinFloat64PointerSlice(slice []*float64) string {
	strs := make([]string, len(slice))
	for i, v := range slice {
		if v != nil {
			strs[i] = fmt.Sprintf("%v", *v)
		} else {
			strs[i] = "nil"
		}
	}
	return strings.Join(strs, ", ")
}

// joinInt64Slice joins a slice of int64 into a single string.
//
// # Parameters:
//
//   - slice: The slice of int64 to be joined.
//
// # Returns:
//
//   - A string representation of the joined slice.
func joinInt64Slice(slice []int64) string {
	strs := make([]string, len(slice))
	for i, v := range slice {
		strs[i] = fmt.Sprintf("%v", v)
	}
	return strings.Join(strs, ", ")
}
