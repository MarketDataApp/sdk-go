// Package options provides access to the Market Data options endpoints:
// option chains (Chain), expiration dates (Expirations), single and bulk
// contract quotes (Quote and Quotes), and OCC option symbol lookup (Lookup).
// It is used through the Options service on a marketdata client rather than
// on its own.
//
// API documentation: https://www.marketdata.app/docs/api/options/index
//
// # Zero Values and Null Data
//
// Numeric fields use Go value types (float64, int64) rather than pointers.
// When the API returns null for a field, it is unmarshaled as the zero value
// (0 for integers, 0.0 for floats). This means a zero value may represent
// either an actual zero or the absence of data. For fields like IV and the
// Greeks (Delta, Gamma, Theta, Vega), a zero value may indicate that the
// data was not calculable rather than a true zero. Timestamp fields are
// the exception: a null or absent timestamp decodes to the zero
// time.Time, so IsZero reliably detects missing data.
package options

import (
	"fmt"
	"strconv"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/timezone"
)

// OptionType identifies a contract as a call or a put. It is used as the
// Type field of [OptionQuote] and as the required contract type argument to
// [Service.Lookup]. Use the [Call] and [Put] constants.
type OptionType string

const (
	// Call is a call option contract.
	Call OptionType = "call"
	// Put is a put option contract.
	Put OptionType = "put"
)

// OptionSide is the side filter for options chain requests, passed to
// [WithSide]. Use [SideCall] for calls only, [SidePut] for puts only, or
// [SideBoth] (the default) for both sides of the chain.
type OptionSide string

const (
	// SideCall limits a chain request to call options.
	SideCall OptionSide = "call"
	// SidePut limits a chain request to put options.
	SidePut OptionSide = "put"
	// SideBoth requests both calls and puts (the default).
	SideBoth OptionSide = ""
)

// OptionsChain is the result of a [Service.Chain] request: the set of option
// contracts listed for a single underlying symbol, after any server-side
// filters have been applied. Each contract appears as an [OptionQuote] in
// the Options slice, carrying its own quote-snapshot Updated time; the API
// reports no chain-level timestamp, so none is exposed here.
type OptionsChain struct {
	// Underlying is the underlying stock symbol
	Underlying string

	// Options is the list of option quotes in the chain
	Options []OptionQuote
}

// Expirations is the result of a [Service.Expirations] request: the
// expiration dates with listed option contracts for an underlying symbol,
// plus the server's response-level update time. Both mirror the API response
// exactly.
type Expirations struct {
	// Dates is the list of expiration dates, in Eastern time.
	Dates []time.Time

	// Updated is when the server last refreshed this expirations list, as
	// reported by the API's response-level updated field. It is the zero
	// time if the API omits the field.
	Updated time.Time
}

// String returns a summary of the expirations list.
func (e Expirations) String() string {
	next := "none"
	if len(e.Dates) > 0 {
		next = e.Dates[0].Format("2006-01-02")
	}
	return fmt.Sprintf("Expirations{Count: %d, Next: %s, Updated: %s}", len(e.Dates), next, e.Updated.Format("2006-01-02 15:04:05"))
}

// OptionQuote is a quote for a single option contract, including pricing,
// volume, open interest, implied volatility, and the Greeks. It is used both
// for the entries of an [OptionsChain] returned by [Service.Chain] and for
// the single-contract responses of [Service.Quote] and [Service.Quotes].
// Timestamps are normalized to Eastern time (the exchange time zone). See
// the package documentation for how null API values map to Go zero values.
type OptionQuote struct {
	// OptionSymbol is the OCC option symbol
	OptionSymbol string `json:"optionSymbol"`

	// Underlying is the underlying stock symbol
	Underlying string `json:"underlying"`

	// Expiration is the expiration date
	Expiration time.Time `json:"expiration"`

	// Strike is the strike price
	Strike float64 `json:"strike"`

	// Type is call or put
	Type OptionType `json:"side"`

	// Bid is the bid price
	Bid float64 `json:"bid"`

	// BidSize is the bid size
	BidSize int `json:"bidSize"`

	// Ask is the ask price
	Ask float64 `json:"ask"`

	// AskSize is the ask size
	AskSize int `json:"askSize"`

	// Last is the last trade price
	Last float64 `json:"last"`

	// Volume is the trading volume
	Volume int64 `json:"volume"`

	// OpenInterest is the open interest
	OpenInterest int64 `json:"openInterest"`

	// IV is the implied volatility
	IV float64 `json:"iv"`

	// Delta is the delta greek
	Delta float64 `json:"delta"`

	// Gamma is the gamma greek
	Gamma float64 `json:"gamma"`

	// Theta is the theta greek
	Theta float64 `json:"theta"`

	// Vega is the vega greek
	Vega float64 `json:"vega"`

	// Rho is the rho greek — the contract's sensitivity to interest rates.
	// The API models rho internally but does not currently serialize it on
	// the chain or quotes endpoints (verified live 2026-08-12), so this
	// field is zero today. It is declared so that the value surfaces without
	// an SDK change once the API emits it, matching sdk-java, which models it
	// for the same reason.
	Rho float64 `json:"rho"`

	// Mid is the midpoint price from the API
	Mid float64 `json:"mid"`

	// UnderlyingPrice is the current price of the underlying
	UnderlyingPrice float64 `json:"underlyingPrice"`

	// IntrinsicValue is the intrinsic value of the option
	IntrinsicValue float64 `json:"intrinsicValue"`

	// ExtrinsicValue is the extrinsic (time) value of the option
	ExtrinsicValue float64 `json:"extrinsicValue"`

	// FirstTraded is the date the option was first traded
	FirstTraded time.Time `json:"firstTraded"`

	// DTE is the days to expiration
	DTE int `json:"dte"`

	// InTheMoney indicates if the option is ITM
	InTheMoney bool `json:"inTheMoney"`

	// Updated is when this contract was last updated
	Updated time.Time `json:"updated"`
}

// String returns a summary of the option contract.
func (c OptionQuote) String() string {
	return fmt.Sprintf("%s %s $%.2f %s Bid: %.2f (%d) Ask: %.2f (%d) Mid: %.2f Last: %.2f Vol: %d OI: %d IV: %.4f Delta: %.4f Gamma: %.4f Theta: %.4f Vega: %.4f Underlying: %s @ %.2f Intrinsic: %.2f Extrinsic: %.2f FirstTraded: %s DTE: %d ITM: %t Updated: %s",
		c.OptionSymbol, c.Expiration.Format("2006-01-02"), c.Strike, c.Type,
		c.Bid, c.BidSize, c.Ask, c.AskSize, c.Mid, c.Last,
		c.Volume, c.OpenInterest,
		c.IV, c.Delta, c.Gamma, c.Theta, c.Vega,
		c.Underlying, c.UnderlyingPrice,
		c.IntrinsicValue, c.ExtrinsicValue,
		c.FirstTraded.Format("2006-01-02"), c.DTE, c.InTheMoney,
		c.Updated.Format("2006-01-02 15:04:05"))
}

// Spread returns the bid-ask spread (Ask minus Bid).
func (c *OptionQuote) Spread() float64 {
	return c.Ask - c.Bid
}

// CalcMid calculates the bid-ask midpoint locally from the Bid and Ask
// fields, unlike the Mid field, which is the midpoint reported by the API.
func (c *OptionQuote) CalcMid() float64 {
	return (c.Bid + c.Ask) / 2
}

// String returns a summary of the options chain.
func (oc OptionsChain) String() string {
	return fmt.Sprintf("OptionsChain{Underlying: %s, Contracts: %d}", oc.Underlying, len(oc.Options))
}

// API response types

type quoteResponse struct {
	Status          string    `json:"s"`
	OptionSymbol    []string  `json:"optionSymbol"`
	Underlying      []string  `json:"underlying"`
	Expiration      []int64   `json:"expiration"`
	Strike          []float64 `json:"strike"`
	Side            []string  `json:"side"`
	Bid             []float64 `json:"bid"`
	BidSize         []int     `json:"bidSize"`
	Ask             []float64 `json:"ask"`
	AskSize         []int     `json:"askSize"`
	Last            []float64 `json:"last"`
	Mid             []float64 `json:"mid"`
	Volume          []int64   `json:"volume"`
	OpenInterest    []int64   `json:"openInterest"`
	IV              []float64 `json:"iv"`
	Delta           []float64 `json:"delta"`
	Gamma           []float64 `json:"gamma"`
	Theta           []float64 `json:"theta"`
	Vega            []float64 `json:"vega"`
	Rho             []float64 `json:"rho"`
	UnderlyingPrice []float64 `json:"underlyingPrice"`
	IntrinsicValue  []float64 `json:"intrinsicValue"`
	ExtrinsicValue  []float64 `json:"extrinsicValue"`
	FirstTraded     []int64   `json:"firstTraded"`
	DTE             []int     `json:"dte"`
	InTheMoney      []bool    `json:"inTheMoney"`
	Updated         []int64   `json:"updated"`
}

// RequiredColumns names the optionSymbol array every quoteResponse
// conversion takes its row count from. See http.ColumnRequirer.
func (r *quoteResponse) RequiredColumns() []string { return []string{"optionSymbol"} }

func (r *quoteResponse) toOptionsChain() *OptionsChain {
	if r == nil || len(r.OptionSymbol) == 0 {
		return &OptionsChain{}
	}

	chain := &OptionsChain{
		Options: make([]OptionQuote, len(r.OptionSymbol)),
	}

	if len(r.Underlying) > 0 {
		chain.Underlying = r.Underlying[0]
	}

	for i := range r.OptionSymbol {
		chain.Options[i] = r.toQuoteAt(i)
	}

	return chain
}

func (r *quoteResponse) toOptionQuote() *OptionQuote {
	if r == nil || len(r.OptionSymbol) == 0 {
		return nil
	}
	q := r.toQuoteAt(0)
	return &q
}

// toOptionQuotes converts every row of the response, preserving the API's
// order. A historical window (a range, or a countback) selects one row per
// day, all of which are returned here — unlike toOptionQuote, which keeps only
// the first because its caller's contract is a single quote.
func (r *quoteResponse) toOptionQuotes() []OptionQuote {
	if r == nil || len(r.OptionSymbol) == 0 {
		return nil
	}
	quotes := make([]OptionQuote, len(r.OptionSymbol))
	for i := range r.OptionSymbol {
		quotes[i] = r.toQuoteAt(i)
	}
	return quotes
}

func (r *quoteResponse) toQuoteAt(i int) OptionQuote {
	return OptionQuote{
		OptionSymbol:    r.OptionSymbol[i],
		Underlying:      safeIndexStr(r.Underlying, i),
		Expiration:      timezone.ToEastern(safeIndexInt64(r.Expiration, i)),
		Strike:          safeIndex(r.Strike, i),
		Type:            OptionType(safeIndexStr(r.Side, i)),
		Bid:             safeIndex(r.Bid, i),
		BidSize:         safeIndexInt(r.BidSize, i),
		Ask:             safeIndex(r.Ask, i),
		AskSize:         safeIndexInt(r.AskSize, i),
		Last:            safeIndex(r.Last, i),
		Mid:             safeIndex(r.Mid, i),
		Volume:          safeIndexInt64(r.Volume, i),
		OpenInterest:    safeIndexInt64(r.OpenInterest, i),
		IV:              safeIndex(r.IV, i),
		Delta:           safeIndex(r.Delta, i),
		Gamma:           safeIndex(r.Gamma, i),
		Theta:           safeIndex(r.Theta, i),
		Vega:            safeIndex(r.Vega, i),
		Rho:             safeIndex(r.Rho, i),
		UnderlyingPrice: safeIndex(r.UnderlyingPrice, i),
		IntrinsicValue:  safeIndex(r.IntrinsicValue, i),
		ExtrinsicValue:  safeIndex(r.ExtrinsicValue, i),
		FirstTraded:     timezone.ToEastern(safeIndexInt64(r.FirstTraded, i)),
		DTE:             safeIndexInt(r.DTE, i),
		InTheMoney:      safeIndexBool(r.InTheMoney, i),
		Updated:         timezone.ToEastern(safeIndexInt64(r.Updated, i)),
	}
}

type expirationsResponse struct {
	Status      string  `json:"s"`
	Expirations []int64 `json:"expirations"`
	Updated     int64   `json:"updated"`
}

// RequiredColumns names the expirations array, which is the whole payload
// here: filtered out it decodes to an empty list, which this endpoint
// documents as "no expirations currently listed" — a false statement rather
// than a zero value. See http.ColumnRequirer.
func (r *expirationsResponse) RequiredColumns() []string { return []string{"expirations"} }

func (r *expirationsResponse) toExpirations() *Expirations {
	if r == nil {
		return nil
	}

	// A 200 OK with an empty list is real data (no expirations currently
	// listed), not "no data" — that's signaled separately by a 404 (see
	// Service.Expirations). Collapsing it to nil here would contradict the
	// documented "nil only on 404" contract and hand callers who trust that
	// contract a nil pointer with NoData still false.
	dates := make([]time.Time, len(r.Expirations))
	for i, ts := range r.Expirations {
		dates[i] = timezone.ToEastern(ts)
	}
	// ToEastern maps a zero/absent updated to the zero time.
	return &Expirations{Dates: dates, Updated: timezone.ToEastern(r.Updated)}
}

type lookupResponse struct {
	Status       string `json:"s"`
	OptionSymbol string `json:"optionSymbol"`
}

// RequiredColumns names the single field a lookup carries; without it the
// call returns an empty symbol on a 200 OK. See http.ColumnRequirer.
func (r *lookupResponse) RequiredColumns() []string { return []string{"optionSymbol"} }

// Helper functions
func safeIndex(s []float64, i int) float64 {
	if i < len(s) {
		return s[i]
	}
	return 0
}

func safeIndexInt(s []int, i int) int {
	if i < len(s) {
		return s[i]
	}
	return 0
}

func safeIndexInt64(s []int64, i int) int64 {
	if i < len(s) {
		return s[i]
	}
	return 0
}

func safeIndexStr(s []string, i int) string {
	if i < len(s) {
		return s[i]
	}
	return ""
}

func safeIndexBool(s []bool, i int) bool {
	if i < len(s) {
		return s[i]
	}
	return false
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func formatInt(i int) string {
	return strconv.Itoa(i)
}

func formatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
