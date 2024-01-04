package client

import (
	"fmt"
	"strings"
	"time"
)

type StockQuotesResponse struct {
	Symbol    []string   `json:"symbol"`
	Ask       []float64  `json:"ask"`
	AskSize   []int64    `json:"askSize"`
	Bid       []float64  `json:"bid"`
	BidSize   []int64    `json:"bidSize"`
	Mid       []float64  `json:"mid"`
	Last      []float64  `json:"last"`
	Change    []*float64 `json:"change,omitempty"`
	ChangePct []*float64 `json:"changepct,omitempty"`
	High52    *[]float64 `json:"52weekHigh,omitempty"`
	Low52     *[]float64 `json:"52weekLow,omitempty"`
	Volume    []int64    `json:"volume"`
	Updated   []int64    `json:"updated"`
}

type StockQuote struct {
	Symbol    string
	Ask       float64
	AskSize   int64
	Bid       float64
	BidSize   int64
	Mid       float64
	Last      float64
	Change    *float64
	ChangePct *float64
	High52    *float64
	Low52     *float64
	Volume    int64
	Updated   time.Time
}

func (sq StockQuote) String() string {
	loc, _ := time.LoadLocation("America/New_York")
	if sq.High52 != nil && sq.Low52 != nil && sq.Change != nil && sq.ChangePct != nil {
		return fmt.Sprintf("Symbol: %s, Ask: %v, AskSize: %v, Bid: %v, BidSize: %v, Mid: %v, Last: %v, Volume: %v, Updated: %s, High52: %v, Low52: %v, Change: %v, ChangePct: %v",
			sq.Symbol, sq.Ask, sq.AskSize, sq.Bid, sq.BidSize, sq.Mid, sq.Last, sq.Volume, sq.Updated.In(loc).Format("2006-01-02 15:04:05 Z07:00"), *sq.High52, *sq.Low52, *sq.Change, *sq.ChangePct)
	} else if sq.High52 != nil && sq.Low52 != nil {
		return fmt.Sprintf("Symbol: %s, Ask: %v, AskSize: %v, Bid: %v, BidSize: %v, Mid: %v, Last: %v, Volume: %v, Updated: %s, High52: %v, Low52: %v",
			sq.Symbol, sq.Ask, sq.AskSize, sq.Bid, sq.BidSize, sq.Mid, sq.Last, sq.Volume, sq.Updated.In(loc).Format("2006-01-02 15:04:05 Z07:00"), *sq.High52, *sq.Low52)
	} else if sq.Change != nil && sq.ChangePct != nil {
		return fmt.Sprintf("Symbol: %s, Ask: %v, AskSize: %v, Bid: %v, BidSize: %v, Mid: %v, Last: %v, Volume: %v, Updated: %s, Change: %v, ChangePct: %v",
			sq.Symbol, sq.Ask, sq.AskSize, sq.Bid, sq.BidSize, sq.Mid, sq.Last, sq.Volume, sq.Updated.In(loc).Format("2006-01-02 15:04:05 Z07:00"), *sq.Change, *sq.ChangePct)
	} else {
		return fmt.Sprintf("Symbol: %s, Ask: %v, AskSize: %v, Bid: %v, BidSize: %v, Mid: %v, Last: %v, Volume: %v, Updated: %s",
			sq.Symbol, sq.Ask, sq.AskSize, sq.Bid, sq.BidSize, sq.Mid, sq.Last, sq.Volume, sq.Updated.In(loc).Format("2006-01-02 15:04:05 Z07:00"))
	}
}

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

func (sqr *StockQuotesResponse) String() string {
	var result strings.Builder

	fmt.Fprintf(&result, "s: ok, Symbol: %v, Ask: %v, AskSize: %v, Bid: %v, BidSize: %v, Mid: %v, Last: %v", 
		sqr.Symbol, sqr.Ask, sqr.AskSize, sqr.Bid, sqr.BidSize, sqr.Mid, sqr.Last)

	if sqr.Change != nil && len(sqr.Change) > 0 {
		fmt.Fprintf(&result, ", Change: %v", sqr.Change[0])
	}
	if sqr.ChangePct != nil && len(sqr.ChangePct) > 0 {
		fmt.Fprintf(&result, ", ChangePct: %v", sqr.ChangePct[0])
	}
	if sqr.High52 != nil && len(*sqr.High52) > 0 {
		fmt.Fprintf(&result, ", High52: %v", *sqr.High52)
	}
	if sqr.Low52 != nil && len(*sqr.Low52) > 0 {
		fmt.Fprintf(&result, ", Low52: %v", *sqr.Low52)
	}

	fmt.Fprintf(&result, ", Volume: %v, Updated: %v", sqr.Volume, sqr.Updated)

	return result.String()
}