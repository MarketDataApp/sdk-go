package models

import (
	"fmt"
	"strings"
	"time"
)

/* Example Response:
{"s":"ok","optionSymbol":["SPXW240202P04845000"],"underlying":["SPXW"],"expiration":[1706907600],"side":["put"],"strike":[4845.0],"firstTraded":[1703601000],"dte":[1],"updated":[1706811106],"bid":[7.2],"bidSize":[219],"mid":[7.3],"ask":[7.4],"askSize":[154],"last":[7.35],"openInterest":[752],"volume":[2012],"inTheMoney":[false],"intrinsicValue":[0.0],"extrinsicValue":[7.3],"underlyingPrice":[4894.75],"iv":[0.192],"delta":[-0.309],"gamma":[0.007],"theta":[-8.494],"vega":[0.988],"rho":[0.107]}
*/

// OptionQuotesResponse represents the JSON structure of the response received for option quotes.
// It includes slices for various option attributes such as symbols, underlying assets, expiration times, and pricing information.
type OptionQuotesResponse struct {
	OptionSymbol    []string  `json:"optionSymbol"`    // OptionSymbol holds the symbols of the options.
	Underlying      []string  `json:"underlying"`      // Underlying contains the symbols of the underlying assets.
	Expiration      []int64   `json:"expiration"`      // Expiration stores UNIX timestamps for when the options expire.
	Side            []string  `json:"side"`            // Side indicates whether the option is a call or a put.
	Strike          []float64 `json:"strike"`          // Strike represents the strike prices of the options.
	FirstTraded     []int64   `json:"firstTraded"`     // FirstTraded stores UNIX timestamps for when the options were first traded.
	DTE             []int     `json:"dte"`             // DTE (Days to Expiration) indicates the number of days until the options expire.
	Ask             []float64 `json:"ask"`             // Ask contains the ask prices of the options.
	AskSize         []int64   `json:"askSize"`         // AskSize holds the sizes of the ask orders.
	Bid             []float64 `json:"bid"`             // Bid contains the bid prices of the options.
	BidSize         []int64   `json:"bidSize"`         // BidSize holds the sizes of the bid orders.
	Mid             []float64 `json:"mid"`             // Mid represents the mid prices calculated between the bid and ask prices.
	Last            []float64 `json:"last"`            // Last contains the last traded prices of the options.
	Volume          []int64   `json:"volume"`          // Volume indicates the trading volumes of the options.
	OpenInterest    []int64   `json:"openInterest"`    // OpenInterest represents the total number of outstanding contracts.
	UnderlyingPrice []float64 `json:"underlyingPrice"` // UnderlyingPrice contains the prices of the underlying assets.
	InTheMoney      []bool    `json:"inTheMoney"`      // InTheMoney indicates whether the options are in the money.
	Updated         []int64   `json:"updated"`         // Updated stores UNIX timestamps for when the option data was last updated.
	IV              []float64 `json:"iv"`              // IV (Implied Volatility) represents the implied volatilities of the options.
	Delta           []float64 `json:"delta"`           // Delta measures the rate of change of the option's price with respect to the underlying asset's price.
	Gamma           []float64 `json:"gamma"`           // Gamma measures the rate of change in delta over the underlying asset's price movement.
	Theta           []float64 `json:"theta"`           // Theta represents the rate of decline in the option's value with time.
	Vega            []float64 `json:"vega"`            // Vega measures sensitivity to volatility.
	Rho             []float64 `json:"rho"`             // Rho measures sensitivity to the interest rate.
	IntrinsicValue  []float64 `json:"intrinsicValue"`  // IntrinsicValue represents the value of the option if it were exercised now.
	ExtrinsicValue  []float64 `json:"extrinsicValue"`  // ExtrinsicValue represents the value of the option above its intrinsic value.
}

// OptionQuote represents a single option quote with detailed information such as the symbol, underlying asset, expiration time, and pricing information.
type OptionQuote struct {
	OptionSymbol    string    // OptionSymbol is the symbol of the option.
	Underlying      string    // Underlying is the symbol of the underlying asset.
	Expiration      time.Time // Expiration is the time when the option expires.
	Side            string    // Side indicates whether the option is a call or a put.
	Strike          float64   // Strike is the strike price of the option.
	FirstTraded     time.Time // FirstTraded is the time when the option was first traded.
	DTE             int       // DTE (Days to Expiration) is the number of days until the option expires.
	Ask             float64   // Ask is the ask price of the option.
	AskSize         int64     // AskSize is the size of the ask order.
	Bid             float64   // Bid is the bid price of the option.
	BidSize         int64     // BidSize is the size of the bid order.
	Mid             float64   // Mid is the mid price calculated between the bid and ask prices.
	Last            float64   // Last is the last traded price of the option.
	Volume          int64     // Volume is the trading volume of the option.
	OpenInterest    int64     // OpenInterest is the total number of outstanding contracts.
	UnderlyingPrice float64   // UnderlyingPrice is the price of the underlying asset.
	InTheMoney      bool      // InTheMoney indicates whether the option is in the money.
	Updated         time.Time // Updated is the time when the option data was last updated.
	IV              float64   // IV (Implied Volatility) is the implied volatility of the option.
	Delta           float64   // Delta measures the rate of change of the option's price with respect to the underlying asset's price.
	Gamma           float64   // Gamma measures the rate of change in delta over the underlying asset's price movement.
	Theta           float64   // Theta represents the rate of decline in the option's value with time.
	Vega            float64   // Vega measures sensitivity to volatility.
	Rho             float64   // Rho measures sensitivity to the interest rate.
	IntrinsicValue  float64   // IntrinsicValue is the value of the option if it were exercised now.
	ExtrinsicValue  float64   // ExtrinsicValue is the value of the option above its intrinsic value.
}

// String returns a formatted string representation of an OptionQuote.
//
// Parameters:
//   - oq: The OptionQuote instance.
//
// Returns:
//   - A string that represents the OptionQuote in a human-readable format.
func (oq OptionQuote) String() string {
	loc, _ := time.LoadLocation("America/New_York")
	return fmt.Sprintf("Option Symbol: %s, Underlying: %s, Expiration: %v, Side: %s, Strike: %v, First Traded: %v, DTE: %v, Ask: %v, Ask Size: %v, Bid: %v, Bid Size: %v, Mid: %v, Last: %v, Volume: %v, Open Interest: %v, Underlying Price: %v, In The Money: %v, Updated: %s, IV: %v, Delta: %v, Gamma: %v, Theta: %v, Vega: %v, Rho: %v, Intrinsic Value: %v, Extrinsic Value: %v",
		oq.OptionSymbol, oq.Underlying, oq.Expiration, oq.Side, oq.Strike, oq.FirstTraded, oq.DTE, oq.Ask, oq.AskSize, oq.Bid, oq.BidSize, oq.Mid, oq.Last, oq.Volume, oq.OpenInterest, oq.UnderlyingPrice, oq.InTheMoney, oq.Updated.In(loc).Format("2006-01-02 15:04:05 Z07:00"), oq.IV, oq.Delta, oq.Gamma, oq.Theta, oq.Vega, oq.Rho, oq.IntrinsicValue, oq.ExtrinsicValue)
}

// Unpack converts the OptionQuotesResponse into a slice of OptionQuote structs.
//
// Parameters:
//   - oqr: A pointer to the OptionQuotesResponse instance to be unpacked.
//
// Returns:
//   - A slice of OptionQuote structs that represent the unpacked data.
//   - An error if the time zone loading fails.
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

// String returns a formatted string representation of all OptionQuotes contained in the OptionQuotesResponse.
//
// Parameters:
//   - oqr: A pointer to the OptionQuotesResponse instance.
//
// Returns:
//   - A string that represents all OptionQuotes in a human-readable format.
//   - If an error occurs during unpacking, it returns a string describing the error.
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
