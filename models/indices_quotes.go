package models

import (
	"fmt"
	"strings"
	"time"
)

// IndexQuotesResponse represents the response structure for index quotes.
// It includes slices for symbols, last prices, changes, percentage changes,
// 52-week highs, 52-week lows, and update timestamps.
type IndexQuotesResponse struct {
	Symbol    []string   `json:"symbol"`             // Symbols are the stock symbols or tickers.
	Last      []float64  `json:"last"`               // Last contains the last traded prices.
	Change    []*float64 `json:"change,omitempty"`   // Change represents the change in price, can be nil if not applicable.
	ChangePct []*float64 `json:"changepct,omitempty"`// ChangePct represents the percentage change in price, can be nil if not applicable.
	High52    *[]float64 `json:"52weekHigh,omitempty"`// High52 points to a slice of 52-week high prices, can be nil if not applicable.
	Low52     *[]float64 `json:"52weekLow,omitempty"` // Low52 points to a slice of 52-week low prices, can be nil if not applicable.
	Updated   []int64    `json:"updated"`            // Updated contains timestamps of the last updates.
}

// IndexQuote represents a single quote for an index.
// It includes the symbol, last price, change, percentage change,
// 52-week high, 52-week low, volume, and update timestamp.
type IndexQuote struct {
	Symbol    string     // Symbol is the stock symbol or ticker.
	Last      float64    // Last is the last traded price.
	Change    *float64   // Change represents the change in price, can be nil if not applicable.
	ChangePct *float64   // ChangePct represents the percentage change in price, can be nil if not applicable.
	High52    *float64   // High52 is the 52-week high price, can be nil if not applicable.
	Low52     *float64   // Low52 is the 52-week low price, can be nil if not applicable.
	Volume    int64      // Volume is the number of shares traded.
	Updated   time.Time  // Updated is the timestamp of the last update.
}

// String returns a string representation of the IndexQuote.
//
// Returns:
//   - A string that represents the IndexQuote.
func (iq IndexQuote) String() string {
	loc, _ := time.LoadLocation("America/New_York")
	if iq.High52 != nil && iq.Low52 != nil && iq.Change != nil && iq.ChangePct != nil {
		return fmt.Sprintf("Symbol: %s, Last: %v, Volume: %v, Updated: %s, High52: %v, Low52: %v, Change: %v, ChangePct: %v",
			iq.Symbol, iq.Last, iq.Volume, iq.Updated.In(loc).Format("2006-01-02 15:04:05 Z07:00"), *iq.High52, *iq.Low52, *iq.Change, *iq.ChangePct)
	} else if iq.High52 != nil && iq.Low52 != nil {
		return fmt.Sprintf("Symbol: %s, Last: %v, Volume: %v, Updated: %s, High52: %v, Low52: %v",
			iq.Symbol, iq.Last, iq.Volume, iq.Updated.In(loc).Format("2006-01-02 15:04:05 Z07:00"), *iq.High52, *iq.Low52)
	} else if iq.Change != nil && iq.ChangePct != nil {
		return fmt.Sprintf("Symbol: %s, Last: %v, Volume: %v, Updated: %s, Change: %v, ChangePct: %v",
			iq.Symbol, iq.Last, iq.Volume, iq.Updated.In(loc).Format("2006-01-02 15:04:05 Z07:00"), *iq.Change, *iq.ChangePct)
	} else {
		return fmt.Sprintf("Symbol: %s, Last: %v, Volume: %v, Updated: %s",
			iq.Symbol, iq.Last, iq.Volume, iq.Updated.In(loc).Format("2006-01-02 15:04:05 Z07:00"))
	}
}

// Unpack transforms the IndexQuotesResponse into a slice of IndexQuote.
//
// Returns:
//   - A slice of IndexQuote derived from the IndexQuotesResponse.
//   - An error if any issues occur during the unpacking process.
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

// String returns a string representation of the IndexQuotesResponse.
//
// Returns:
//   - A string that represents the IndexQuotesResponse.
func (iqr *IndexQuotesResponse) String() string {
	var result strings.Builder

	fmt.Fprintf(&result, "Symbol: [%v], Last: [%v]", strings.Join(iqr.Symbol, ", "), joinFloat64Slice(iqr.Last))

	if iqr.Change != nil {
		fmt.Fprintf(&result, ", Change: [%v]", joinFloat64PointerSlice(iqr.Change))
	}
	if iqr.ChangePct != nil {
		fmt.Fprintf(&result, ", ChangePct: [%v]", joinFloat64PointerSlice(iqr.ChangePct))
	}
	if iqr.High52 != nil {
		fmt.Fprintf(&result, ", High52: [%v]", joinFloat64Slice(*iqr.High52))
	}
	if iqr.Low52 != nil {
		fmt.Fprintf(&result, ", Low52: [%v]", joinFloat64Slice(*iqr.Low52))
	}

	fmt.Fprintf(&result, ", Updated: [%v]", joinInt64Slice(iqr.Updated))

	return result.String()
}

// joinFloat64Slice joins a slice of float64 into a single string.
//
// Parameters:
//   - slice: The slice of float64 to be joined.
//
// Returns:
//   - A string representation of the joined slice.
func joinFloat64Slice(slice []float64) string {
	strs := make([]string, len(slice))
	for i, v := range slice {
		strs[i] = fmt.Sprintf("%v", v)
	}
	return strings.Join(strs, ", ")
}

// joinFloat64PointerSlice joins a slice of *float64 into a single string, handling nil pointers gracefully.
//
// Parameters:
//   - slice: The slice of *float64 to be joined.
//
// Returns:
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
// Parameters:
//   - slice: The slice of int64 to be joined.
//
// Returns:
//   - A string representation of the joined slice.
func joinInt64Slice(slice []int64) string {
	strs := make([]string, len(slice))
	for i, v := range slice {
		strs[i] = fmt.Sprintf("%v", v)
	}
	return strings.Join(strs, ", ")
}