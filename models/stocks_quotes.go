package models

import (
	"fmt"
	"strings"
	"time"
)

// StockQuotesResponse represents the response structure for stock quotes.
// It includes slices for symbols, ask prices, ask sizes, bid prices, bid sizes, mid prices, last trade prices,
// changes, change percentages, 52-week highs, 52-week lows, volumes, and update timestamps.
type StockQuotesResponse struct {
	Symbol    []string   `json:"symbol"`               // Symbol holds the stock symbols.
	Ask       []float64  `json:"ask"`                  // Ask holds the asking prices for the stocks.
	AskSize   []int64    `json:"askSize"`              // AskSize holds the sizes (quantities) of the asks.
	Bid       []float64  `json:"bid"`                  // Bid holds the bidding prices for the stocks.
	BidSize   []int64    `json:"bidSize"`              // BidSize holds the sizes (quantities) of the bids.
	Mid       []float64  `json:"mid"`                  // Mid holds the mid prices calculated between the ask and bid prices.
	Last      []float64  `json:"last"`                 // Last holds the last traded prices for the stocks.
	Change    []*float64 `json:"change,omitempty"`     // Change holds the price changes, can be nil if not applicable.
	ChangePct []*float64 `json:"changepct,omitempty"`  // ChangePct holds the percentage changes in prices, can be nil if not applicable.
	High52    *[]float64 `json:"52weekHigh,omitempty"` // High52 holds the 52-week high prices, can be nil if not applicable.
	Low52     *[]float64 `json:"52weekLow,omitempty"`  // Low52 holds the 52-week low prices, can be nil if not applicable.
	Volume    []int64    `json:"volume"`               // Volume holds the trading volumes for the stocks.
	Updated   []int64    `json:"updated"`              // Updated holds the UNIX timestamps for when the quotes were last updated.
}

// StockQuote represents a single stock quote.
// It includes the stock symbol, ask price, ask size, bid price, bid size, mid price, last trade price,
// change, change percentage, 52-week high, 52-week low, volume, and the time of the last update.
type StockQuote struct {
	Symbol    string    // Symbol is the stock symbol.
	Ask       float64   // Ask is the asking price for the stock.
	AskSize   int64     // AskSize is the size (quantity) of the ask.
	Bid       float64   // Bid is the bidding price for the stock.
	BidSize   int64     // BidSize is the size (quantity) of the bid.
	Mid       float64   // Mid is the mid price calculated between the ask and bid prices.
	Last      float64   // Last is the last traded price for the stock.
	Change    *float64  // Change is the price change, can be nil if not applicable.
	ChangePct *float64  // ChangePct is the percentage change in price, can be nil if not applicable.
	High52    *float64  // High52 is the 52-week high price, can be nil if not applicable.
	Low52     *float64  // Low52 is the 52-week low price, can be nil if not applicable.
	Volume    int64     // Volume is the trading volume for the stock.
	Updated   time.Time // Updated is the time when the quote was last updated.
}

// String generates a string representation of the StockQuote struct.
//
// This method formats the fields of a StockQuote into a readable string. It includes conditional formatting
// based on the presence of optional fields such as High52, Low52, Change, and ChangePct. The updated time
// is formatted in the America/New_York timezone.
//
// Returns:
//   - A string representation of the StockQuote struct.
func (sq StockQuote) String() string {
	loc, _ := time.LoadLocation("America/New_York")
	updatedFormat := sq.Updated.In(loc).Format("2006-01-02 15:04:05 Z07:00")
	high52 := "nil"
	if sq.High52 != nil {
		high52 = fmt.Sprintf("%v", *sq.High52)
	}
	low52 := "nil"
	if sq.Low52 != nil {
		low52 = fmt.Sprintf("%v", *sq.Low52)
	}
	change := "nil"
	if sq.Change != nil {
		change = fmt.Sprintf("%v", *sq.Change)
	}
	changePct := "nil"
	if sq.ChangePct != nil {
		changePct = fmt.Sprintf("%v", *sq.ChangePct)
	}
	return fmt.Sprintf("StockQuote{Symbol: %q, Ask: %v, AskSize: %v, Bid: %v, BidSize: %v, Mid: %v, Last: %v, Volume: %v, Updated: %q, High52: %s, Low52: %s, Change: %s, ChangePct: %s}",
		sq.Symbol, sq.Ask, sq.AskSize, sq.Bid, sq.BidSize, sq.Mid, sq.Last, sq.Volume, updatedFormat, high52, low52, change, changePct)
}

// Unpack transforms the StockQuotesResponse into a slice of StockQuote structs.
//
// This method iterates over the fields of a StockQuotesResponse and constructs a slice of StockQuote
// structs by assigning the corresponding values from the response to each StockQuote. It handles optional
// fields such as Change, ChangePct, High52, and Low52 by checking for their existence before assignment.
//
// Returns:
//   - A slice of StockQuote structs representing the unpacked stock quotes.
//   - An error if any issues occur during the unpacking process (currently, this implementation always returns nil for error).
func (sqr *StockQuotesResponse) Unpack() ([]StockQuote, error) {
	var stockQuotes []StockQuote
	for i := range sqr.Symbol {
		stockQuote := StockQuote{
			Symbol:  sqr.Symbol[i],
			Ask:     sqr.Ask[i],
			AskSize: sqr.AskSize[i],
			Bid:     sqr.Bid[i],
			BidSize: sqr.BidSize[i],
			Mid:     sqr.Mid[i],
			Last:    sqr.Last[i],
			Volume:  sqr.Volume[i],
			Updated: time.Unix(sqr.Updated[i], 0),
		}
		if sqr.Change != nil && len(sqr.Change) > i {
			stockQuote.Change = sqr.Change[i]
		}
		if sqr.ChangePct != nil && len(sqr.ChangePct) > i {
			stockQuote.ChangePct = sqr.ChangePct[i]
		}
		if sqr.High52 != nil && len(*sqr.High52) > i {
			val := (*sqr.High52)[i]
			stockQuote.High52 = &val
		}
		if sqr.Low52 != nil && len(*sqr.Low52) > i {
			val := (*sqr.Low52)[i]
			stockQuote.Low52 = &val
		}
		stockQuotes = append(stockQuotes, stockQuote)
	}
	return stockQuotes, nil
}

// String returns a string representation of a StockQuotesResponse.
//
// This method constructs a string that includes the symbol, ask, ask size, bid, bid size, mid, last, change (if available),
// change percentage (if available), 52-week high (if available), 52-week low (if available), volume, and updated time of the stock quotes response.
//
// Returns:
//   - A string that represents the stock quotes response.
func (sqr *StockQuotesResponse) String() string {
	var result strings.Builder

	fmt.Fprintf(&result, "StockQuotesResponse{Symbol: [%v], Ask: [%v], AskSize: [%v], Bid: [%v], BidSize: [%v], Mid: [%v], Last: [%v]",
		strings.Join(sqr.Symbol, ", "), joinFloat64Slice(sqr.Ask), joinInt64Slice(sqr.AskSize), joinFloat64Slice(sqr.Bid), joinInt64Slice(sqr.BidSize), joinFloat64Slice(sqr.Mid), joinFloat64Slice(sqr.Last))

	if sqr.Change != nil && len(sqr.Change) > 0 {
		fmt.Fprintf(&result, ", Change: [%v]", joinFloat64PointerSlice(sqr.Change))
	}
	if sqr.ChangePct != nil && len(sqr.ChangePct) > 0 {
		fmt.Fprintf(&result, ", ChangePct: [%v]", joinFloat64PointerSlice(sqr.ChangePct))
	}
	if sqr.High52 != nil && len(*sqr.High52) > 0 {
		fmt.Fprintf(&result, ", High52: [%v]", joinFloat64Slice(*sqr.High52))
	}
	if sqr.Low52 != nil && len(*sqr.Low52) > 0 {
		fmt.Fprintf(&result, ", Low52: [%v]", joinFloat64Slice(*sqr.Low52))
	}

	fmt.Fprintf(&result, ", Volume: [%v], Updated: [%v]}", joinInt64Slice(sqr.Volume), joinInt64Slice(sqr.Updated))

	return result.String()
}
