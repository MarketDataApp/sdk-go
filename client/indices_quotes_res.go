package client

import (
	"fmt"
	"strings"
	"time"
)

type IndexQuotesResponse struct {
	Symbol    []string   `json:"symbol"`
	Last      []float64  `json:"last"`
	Change    []*float64 `json:"change,omitempty"`
	ChangePct []*float64 `json:"changepct,omitempty"`
	High52    *[]float64 `json:"52weekHigh,omitempty"`
	Low52     *[]float64 `json:"52weekLow,omitempty"`
	Updated   []int64    `json:"updated"`
}

type IndexQuote struct {
	Symbol    string
	Last      float64
	Change    *float64
	ChangePct *float64
	High52    *float64
	Low52     *float64
	Volume    int64
	Updated   time.Time
}

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

func (iqr *IndexQuotesResponse) String() string {
	var result strings.Builder

	fmt.Fprintf(&result, "Symbol: %v, Last: %v", 
		iqr.Symbol, iqr.Last)

	if iqr.Change != nil && len(iqr.Change) > 0 {
		fmt.Fprintf(&result, ", Change: %v", iqr.Change[0])
	}
	if iqr.ChangePct != nil && len(iqr.ChangePct) > 0 {
		fmt.Fprintf(&result, ", ChangePct: %v", iqr.ChangePct[0])
	}
	if iqr.High52 != nil && len(*iqr.High52) > 0 {
		fmt.Fprintf(&result, ", High52: %v", *iqr.High52)
	}
	if iqr.Low52 != nil && len(*iqr.Low52) > 0 {
		fmt.Fprintf(&result, ", Low52: %v", *iqr.Low52)
	}

	fmt.Fprintf(&result, ", Updated: %v", iqr.Updated)

	return result.String()
}