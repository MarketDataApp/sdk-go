package models

import (
	"fmt"
	"strings"
	"time"
)

/* Example Response:
{"s":"ok","optionSymbol":["SPXW240202P04845000"],"underlying":["SPXW"],"expiration":[1706907600],"side":["put"],"strike":[4845.0],"firstTraded":[1703601000],"dte":[1],"updated":[1706811106],"bid":[7.2],"bidSize":[219],"mid":[7.3],"ask":[7.4],"askSize":[154],"last":[7.35],"openInterest":[752],"volume":[2012],"inTheMoney":[false],"intrinsicValue":[0.0],"extrinsicValue":[7.3],"underlyingPrice":[4894.75],"iv":[0.192],"delta":[-0.309],"gamma":[0.007],"theta":[-8.494],"vega":[0.988],"rho":[0.107]}
*/

type OptionQuotesResponse struct {
	OptionSymbol    []string  `json:"optionSymbol"`
	Underlying      []string  `json:"underlying"`
	Expiration      []int64   `json:"expiration"`
	Side            []string  `json:"side"`
	Strike          []float64 `json:"strike"`
	FirstTraded     []int64   `json:"firstTraded"`
	DTE             []int     `json:"dte"`
	Ask             []float64 `json:"ask"`
	AskSize         []int64   `json:"askSize"`
	Bid             []float64 `json:"bid"`
	BidSize         []int64   `json:"bidSize"`
	Mid             []float64 `json:"mid"`
	Last            []float64 `json:"last"`
	Volume          []int64   `json:"volume"`
	OpenInterest    []int64   `json:"openInterest"`
	UnderlyingPrice []float64 `json:"underlyingPrice"`
	InTheMoney      []bool    `json:"inTheMoney"`
	Updated         []int64   `json:"updated"`
	IV              []float64 `json:"iv"`
	Delta           []float64 `json:"delta"`
	Gamma           []float64 `json:"gamma"`
	Theta           []float64 `json:"theta"`
	Vega            []float64 `json:"vega"`
	Rho             []float64 `json:"rho"`
	IntrinsicValue  []float64 `json:"intrinsicValue"`
	ExtrinsicValue  []float64 `json:"extrinsicValue"`
}

type OptionQuote struct {
	OptionSymbol    string
	Underlying      string
	Expiration      time.Time
	Side            string
	Strike          float64
	FirstTraded     time.Time
	DTE             int
	Ask             float64
	AskSize         int64
	Bid             float64
	BidSize         int64
	Mid             float64
	Last            float64
	Volume          int64
	OpenInterest    int64
	UnderlyingPrice float64
	InTheMoney      bool
	Updated         time.Time
	IV              float64
	Delta           float64
	Gamma           float64
	Theta           float64
	Vega            float64
	Rho             float64
	IntrinsicValue  float64
	ExtrinsicValue  float64
}

func (oq OptionQuote) String() string {
	loc, _ := time.LoadLocation("America/New_York")
	return fmt.Sprintf("Option Symbol: %s, Underlying: %s, Expiration: %v, Side: %s, Strike: %v, First Traded: %v, DTE: %v, Ask: %v, Ask Size: %v, Bid: %v, Bid Size: %v, Mid: %v, Last: %v, Volume: %v, Open Interest: %v, Underlying Price: %v, In The Money: %v, Updated: %s, IV: %v, Delta: %v, Gamma: %v, Theta: %v, Vega: %v, Rho: %v, Intrinsic Value: %v, Extrinsic Value: %v",
		oq.OptionSymbol, oq.Underlying, oq.Expiration, oq.Side, oq.Strike, oq.FirstTraded, oq.DTE, oq.Ask, oq.AskSize, oq.Bid, oq.BidSize, oq.Mid, oq.Last, oq.Volume, oq.OpenInterest, oq.UnderlyingPrice, oq.InTheMoney, oq.Updated.In(loc).Format("2006-01-02 15:04:05 Z07:00"), oq.IV, oq.Delta, oq.Gamma, oq.Theta, oq.Vega, oq.Rho, oq.IntrinsicValue, oq.ExtrinsicValue)
}

func (oqr *OptionQuotesResponse) Unpack() ([]OptionQuote, error) {
	loc, err := time.LoadLocation("America/New_York") // Load the New York time zone
	if err != nil {
		return nil, fmt.Errorf("failed to load New York time zone: %v", err)
	}
	var optionQuotes []OptionQuote
	for i := range oqr.OptionSymbol {
		optionQuote := OptionQuote{
			OptionSymbol:    oqr.OptionSymbol[i],
			Underlying:      oqr.Underlying[i],
			Expiration:      time.Unix(oqr.Expiration[i], 0).In(loc), // Convert to time.Time in America/New_York
			Side:            oqr.Side[i],
			Strike:          oqr.Strike[i],
			FirstTraded:     time.Unix(oqr.FirstTraded[i], 0).In(loc), // Convert to time.Time in America/New_York
			DTE:             oqr.DTE[i],
			Ask:             oqr.Ask[i],
			AskSize:         oqr.AskSize[i],
			Bid:             oqr.Bid[i],
			BidSize:         oqr.BidSize[i],
			Mid:             oqr.Mid[i],
			Last:            oqr.Last[i],
			Volume:          oqr.Volume[i],
			OpenInterest:    oqr.OpenInterest[i],
			UnderlyingPrice: oqr.UnderlyingPrice[i],
			InTheMoney:      oqr.InTheMoney[i],
			Updated:         time.Unix(oqr.Updated[i], 0).In(loc), // Convert to time.Time in America/New_York
			IV:              oqr.IV[i],
			Delta:           oqr.Delta[i],
			Gamma:           oqr.Gamma[i],
			Theta:           oqr.Theta[i],
			Vega:            oqr.Vega[i],
			Rho:             oqr.Rho[i],
			IntrinsicValue:  oqr.IntrinsicValue[i],
			ExtrinsicValue:  oqr.ExtrinsicValue[i],
		}
		optionQuotes = append(optionQuotes, optionQuote)
	}
	return optionQuotes, nil
}

func (oqr *OptionQuotesResponse) String() string {
	optionQuotes, err := oqr.Unpack()
	if err != nil {
		return fmt.Sprintf("Error unpacking OptionQuotesResponse: %v", err)
	}

	var quotesStrings []string
	for _, quote := range optionQuotes {
		quotesStrings = append(quotesStrings, quote.String())
	}

	return strings.Join(quotesStrings, "\n")
}
