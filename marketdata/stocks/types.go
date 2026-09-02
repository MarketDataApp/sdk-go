// Package stocks provides access to the Market Data stocks endpoints:
// real-time quotes and bulk quotes, historical candles, bulk candles,
// SmartMid prices, earnings, and news. All methods are exposed through
// [Service], available as the Stocks field of the marketdata client.
// Timestamps in returned data are normalized to US Eastern time, the
// timezone of the US exchanges.
//
// API documentation: https://www.marketdata.app/docs/api/stocks/index
//
// # Zero Values and Null Data
//
// Numeric fields use Go value types (float64, int64) rather than pointers.
// When the API returns null for a field, it is unmarshaled as the zero value
// (0 for integers, 0.0 for floats). This means a zero value may represent
// either an actual zero or the absence of data. For most fields (prices,
// volume, change) this distinction is rarely meaningful. Two exceptions
// keep it meaningful where it matters: the Earning EPS fields use
// *float64 to distinguish null (not yet reported) from a true $0.00, and
// timestamp fields decode a null or absent value to the zero time.Time,
// so IsZero reliably detects missing data.
package stocks

import (
	"fmt"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/timezone"
)

// Quote represents a real-time stock quote returned by [Service.Quote]
// and [Service.Quotes]. The FiftyTwoWeekHigh and FiftyTwoWeekLow fields
// are populated only when the quote was requested with [WithFiftyTwoWeek].
type Quote struct {
	// Symbol is the stock ticker symbol
	Symbol string `json:"symbol"`

	// Ask is the current ask price
	Ask float64 `json:"ask"`

	// AskSize is the size of the ask
	AskSize int `json:"askSize"`

	// Bid is the current bid price
	Bid float64 `json:"bid"`

	// BidSize is the size of the bid
	BidSize int `json:"bidSize"`

	// Mid is the midpoint between bid and ask
	Mid float64 `json:"mid"`

	// Last is the last trade price
	Last float64 `json:"last"`

	// Change is the price change from previous close
	Change float64 `json:"change"`

	// ChangePercent is the fractional change from the previous close, as
	// sent by the API: -0.0021 means -0.21%. Multiply by 100 for a
	// percentage.
	ChangePercent float64 `json:"changepct"`

	// Volume is the trading volume
	Volume int64 `json:"volume"`

	// Updated is when this quote was last updated
	Updated time.Time `json:"updated"`

	// 52 week data (optional, requested via WithFiftyTwoWeek)
	FiftyTwoWeekHigh float64 `json:"52weekHigh,omitempty"`
	FiftyTwoWeekLow  float64 `json:"52weekLow,omitempty"`

	// Session OHLC for the current trading day (optional, requested via
	// [WithCandle] or [WithQuotesCandle]). The API omits these fields unless
	// the candle parameter is sent, in which case they are all zero here.
	Open  float64 `json:"o,omitempty"`
	High  float64 `json:"h,omitempty"`
	Low   float64 `json:"l,omitempty"`
	Close float64 `json:"c,omitempty"`
}

// String returns a summary of the quote.
func (q Quote) String() string {
	s := fmt.Sprintf("%s Last: $%.2f Bid: %.2f (%d) Ask: %.2f (%d) Mid: %.2f Chg: %.2f (%.2f%%) Vol: %d Updated: %s",
		q.Symbol, q.Last, q.Bid, q.BidSize, q.Ask, q.AskSize, q.Mid, q.Change, q.ChangePercent*100, q.Volume, q.Updated.Format("2006-01-02 15:04:05"))
	if q.FiftyTwoWeekHigh != 0 || q.FiftyTwoWeekLow != 0 {
		s += fmt.Sprintf(" 52wk: %.2f-%.2f", q.FiftyTwoWeekLow, q.FiftyTwoWeekHigh)
	}
	return s
}

// Spread returns the bid-ask spread.
func (q *Quote) Spread() float64 {
	return q.Ask - q.Bid
}

// SpreadPercent returns the bid-ask spread as a percentage of the mid price.
func (q *Quote) SpreadPercent() float64 {
	if q.Mid == 0 {
		return 0
	}
	return (q.Spread() / q.Mid) * 100
}

// Resolution represents the candle resolution/timeframe used by
// [Service.Candles] and [Service.BulkCandles]. Intraday resolutions range
// from [Resolution1Min] to [Resolution4Hour]; [ResolutionDaily],
// [ResolutionWeekly], and [ResolutionMonthly] cover longer timeframes.
type Resolution string

const (
	// Resolution1Min is 1 minute candles
	Resolution1Min Resolution = "1"

	// Resolution5Min is 5 minute candles
	Resolution5Min Resolution = "5"

	// Resolution15Min is 15 minute candles
	Resolution15Min Resolution = "15"

	// Resolution30Min is 30 minute candles
	Resolution30Min Resolution = "30"

	// Resolution1Hour is 1 hour candles
	Resolution1Hour Resolution = "60"

	// Resolution3Min is 3 minute candles
	Resolution3Min Resolution = "3"

	// Resolution45Min is 45 minute candles
	Resolution45Min Resolution = "45"

	// Resolution2Hour is 2 hour candles
	Resolution2Hour Resolution = "120"

	// Resolution4Hour is 4 hour candles
	Resolution4Hour Resolution = "240"

	// ResolutionDaily is daily candles
	ResolutionDaily Resolution = "D"

	// ResolutionWeekly is weekly candles
	ResolutionWeekly Resolution = "W"

	// ResolutionMonthly is monthly candles
	ResolutionMonthly Resolution = "M"

	// ResolutionYearly is yearly candles
	ResolutionYearly Resolution = "Y"
)

// String returns the string representation of the resolution.
func (r Resolution) String() string {
	return string(r)
}

// Candle represents a single OHLCV (open, high, low, close, volume) candle
// returned by [Service.Candles]. Time marks the start of the candle period
// and is normalized to US Eastern time.
type Candle struct {
	// Time is the candle timestamp
	Time time.Time `json:"t"`

	// Open is the opening price
	Open float64 `json:"o"`

	// High is the highest price
	High float64 `json:"h"`

	// Low is the lowest price
	Low float64 `json:"l"`

	// Close is the closing price
	Close float64 `json:"c"`

	// Volume is the trading volume
	Volume int64 `json:"v"`
}

// String returns a concise summary of the candle.
func (c Candle) String() string {
	return fmt.Sprintf("%s O: %.2f H: %.2f L: %.2f C: %.2f V: %d", c.Time.Format("2006-01-02"), c.Open, c.High, c.Low, c.Close, c.Volume)
}

// Range returns the high-low range.
func (c *Candle) Range() float64 {
	return c.High - c.Low
}

// RangePercent returns the range as a percentage of the open.
func (c *Candle) RangePercent() float64 {
	if c.Open == 0 {
		return 0
	}
	return (c.Range() / c.Open) * 100
}

// IsBullish returns true if close > open.
func (c *Candle) IsBullish() bool {
	return c.Close > c.Open
}

// IsBearish returns true if close < open.
func (c *Candle) IsBearish() bool {
	return c.Close < c.Open
}

// quotesResponse is the API response for quotes.
type quotesResponse struct {
	Status           string    `json:"s"`
	Symbol           []string  `json:"symbol"`
	Ask              []float64 `json:"ask"`
	AskSize          []int     `json:"askSize"`
	Bid              []float64 `json:"bid"`
	BidSize          []int     `json:"bidSize"`
	Mid              []float64 `json:"mid"`
	Last             []float64 `json:"last"`
	Change           []float64 `json:"change"`
	ChangePct        []float64 `json:"changepct"`
	Volume           []int64   `json:"volume"`
	Updated          []int64   `json:"updated"` // Unix timestamp
	FiftyTwoWeekHigh []float64 `json:"52weekHigh"`
	FiftyTwoWeekLow  []float64 `json:"52weekLow"`
	Open             []float64 `json:"o"`
	High             []float64 `json:"h"`
	Low              []float64 `json:"l"`
	Close            []float64 `json:"c"`
}

// RequiredColumns names the symbol array toQuotes takes its row count
// from; filtered out, a present quote decodes to zero rows and Quote
// reports QuoteNotFoundError. See http.ColumnRequirer.
func (r *quotesResponse) RequiredColumns() []string { return []string{"symbol"} }

// toQuotes converts the API response to a slice of Quote.
func (r *quotesResponse) toQuotes() []Quote {
	if r == nil || len(r.Symbol) == 0 {
		return nil
	}

	quotes := make([]Quote, len(r.Symbol))
	for i := range r.Symbol {
		quotes[i] = Quote{
			Symbol:           r.Symbol[i],
			Ask:              safeIndex(r.Ask, i),
			AskSize:          safeIndexInt(r.AskSize, i),
			Bid:              safeIndex(r.Bid, i),
			BidSize:          safeIndexInt(r.BidSize, i),
			Mid:              safeIndex(r.Mid, i),
			Last:             safeIndex(r.Last, i),
			Change:           safeIndex(r.Change, i),
			ChangePercent:    safeIndex(r.ChangePct, i),
			Volume:           safeIndexInt64(r.Volume, i),
			Updated:          timezone.ToEastern(safeIndexInt64(r.Updated, i)),
			FiftyTwoWeekHigh: safeIndex(r.FiftyTwoWeekHigh, i),
			FiftyTwoWeekLow:  safeIndex(r.FiftyTwoWeekLow, i),
			Open:             safeIndex(r.Open, i),
			High:             safeIndex(r.High, i),
			Low:              safeIndex(r.Low, i),
			Close:            safeIndex(r.Close, i),
		}
	}
	return quotes
}

// candlesResponse is the API response for candles.
type candlesResponse struct {
	Status string    `json:"s"`
	Time   []int64   `json:"t"` // Unix timestamps
	Open   []float64 `json:"o"`
	High   []float64 `json:"h"`
	Low    []float64 `json:"l"`
	Close  []float64 `json:"c"`
	Volume []int64   `json:"v"`
}

// RequiredColumns names the timestamp array toCandles takes its row count
// from. See http.ColumnRequirer.
func (r *candlesResponse) RequiredColumns() []string { return []string{"t"} }

// toCandles converts the API response to a slice of Candle.
func (r *candlesResponse) toCandles() []Candle {
	if r == nil || len(r.Time) == 0 {
		return nil
	}

	candles := make([]Candle, len(r.Time))
	for i := range r.Time {
		candles[i] = Candle{
			Time:   timezone.ToEastern(r.Time[i]),
			Open:   safeIndex(r.Open, i),
			High:   safeIndex(r.High, i),
			Low:    safeIndex(r.Low, i),
			Close:  safeIndex(r.Close, i),
			Volume: safeIndexInt64(r.Volume, i),
		}
	}
	return candles
}

// Helper functions for safe slice access
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

// copyFloatPtr returns a copy of s[i] as a fresh pointer, or nil when the
// index is out of range or the wire value was null. Copying keeps the public
// struct from aliasing the decoded response's memory.
func copyFloatPtr(s []*float64, i int) *float64 {
	if i < len(s) && s[i] != nil {
		v := *s[i]
		return &v
	}
	return nil
}

// QuoteNotFoundError is returned by [Service.Quote] when the API reports a
// successful response that contains no quote for the requested symbol.
// Use errors.As to detect it.
type QuoteNotFoundError struct {
	Symbol string
}

func (e *QuoteNotFoundError) Error() string {
	return fmt.Sprintf("stocks: quote not found for symbol: %s", e.Symbol)
}

// Price represents a SmartMid midpoint price for a stock, returned by
// [Service.Prices]. It is a lightweight alternative to [Quote] when only a
// single price per symbol is needed.
type Price struct {
	// Symbol is the stock ticker symbol
	Symbol string `json:"symbol"`

	// Mid is the SmartMid midpoint price
	Mid float64 `json:"mid"`

	// Change is the price change from previous close
	Change float64 `json:"change"`

	// ChangePercent is the fractional change from the previous close, as
	// sent by the API: -0.0021 means -0.21%. Multiply by 100 for a
	// percentage.
	ChangePercent float64 `json:"changepct"`

	// Updated is when this price was last updated
	Updated time.Time `json:"updated"`
}

// String returns a summary of the price.
func (p Price) String() string {
	return fmt.Sprintf("%s Mid: $%.2f Chg: %.2f (%.2f%%) Updated: %s", p.Symbol, p.Mid, p.Change, p.ChangePercent*100, p.Updated.Format("2006-01-02 15:04:05"))
}

// pricesResponse is the API response for prices.
type pricesResponse struct {
	Status    string    `json:"s"`
	Symbol    []string  `json:"symbol"`
	Mid       []float64 `json:"mid"`
	Change    []float64 `json:"change"`
	ChangePct []float64 `json:"changepct"`
	Updated   []int64   `json:"updated"` // Unix timestamp
}

// RequiredColumns names the symbol array toPrices takes its row count from.
// See http.ColumnRequirer.
func (r *pricesResponse) RequiredColumns() []string { return []string{"symbol"} }

// toPrices converts the API response to a slice of Price.
func (r *pricesResponse) toPrices() []Price {
	if r == nil || len(r.Symbol) == 0 {
		return nil
	}

	prices := make([]Price, len(r.Symbol))
	for i := range r.Symbol {
		prices[i] = Price{
			Symbol:        r.Symbol[i],
			Mid:           safeIndex(r.Mid, i),
			Change:        safeIndex(r.Change, i),
			ChangePercent: safeIndex(r.ChangePct, i),
			Updated:       timezone.ToEastern(safeIndexInt64(r.Updated, i)),
		}
	}
	return prices
}

// Earning represents a single earnings report for a stock, one per fiscal
// quarter, returned by [Service.Earnings]. The EPS fields are pointers and
// are nil when the API reports no value, such as for earnings that have
// not yet been reported.
type Earning struct {
	// Symbol is the stock ticker symbol
	Symbol string `json:"symbol"`

	// FiscalYear is the fiscal year of the earnings report
	FiscalYear int `json:"fiscalYear"`

	// FiscalQuarter is the fiscal quarter (1-4)
	FiscalQuarter int `json:"fiscalQuarter"`

	// Date is the last calendar day of the fiscal period
	Date time.Time `json:"date"`

	// ReportDate is when the earnings were or will be reported
	ReportDate time.Time `json:"reportDate"`

	// ReportTime indicates when during the day (before/after market, during hours)
	ReportTime string `json:"reportTime"`

	// Currency is the currency of the earnings report (may be empty for future earnings)
	Currency string `json:"currency"`

	// ReportedEPS is the actual reported earnings per share (nil for future earnings)
	ReportedEPS *float64 `json:"reportedEPS"`

	// EstimatedEPS is the consensus analyst estimate
	EstimatedEPS *float64 `json:"estimatedEPS"`

	// SurpriseEPS is the difference between reported and estimated
	SurpriseEPS *float64 `json:"surpriseEPS"`

	// SurpriseEPSPercent is the surprise as a fraction of the estimate,
	// as sent by the API: 0.0256 means 2.56%. Multiply by 100 for a
	// percentage.
	SurpriseEPSPercent *float64 `json:"surpriseEPSpct"`

	// Updated is when this earnings data was last updated
	Updated time.Time `json:"updated"`
}

// String returns a summary of the earning.
func (e Earning) String() string {
	fmtEPS := func(v *float64) string {
		if v == nil {
			return "n/a"
		}
		return fmt.Sprintf("$%.2f", *v)
	}
	// SurpriseEPSPercent is a wire fraction; render it as a percentage.
	fmtPct := func(v *float64) string {
		if v == nil {
			return "n/a"
		}
		return fmt.Sprintf("%.2f%%", *v*100)
	}
	return fmt.Sprintf("%s Q%d %d Date: %s ReportDate: %s ReportTime: %s Currency: %s Reported: %s Estimated: %s Surprise: %s (%s) Updated: %s",
		e.Symbol, e.FiscalQuarter, e.FiscalYear,
		e.Date.Format("2006-01-02"), e.ReportDate.Format("2006-01-02"),
		e.ReportTime, e.Currency,
		fmtEPS(e.ReportedEPS), fmtEPS(e.EstimatedEPS),
		fmtEPS(e.SurpriseEPS), fmtPct(e.SurpriseEPSPercent),
		e.Updated.Format("2006-01-02 15:04:05"))
}

// earningsResponse is the API response for earnings.
type earningsResponse struct {
	Status        string   `json:"s"`
	Symbol        []string `json:"symbol"`
	FiscalYear    []int    `json:"fiscalYear"`
	FiscalQuarter []int    `json:"fiscalQuarter"`
	Date          []int64  `json:"date"`
	ReportDate    []int64  `json:"reportDate"`
	ReportTime    []string `json:"reportTime"`
	Currency      []string `json:"currency"`
	// The four EPS arrays use pointer elements so a wire null (not yet
	// reported) decodes to nil while a true $0.00 survives as a pointer to
	// zero — the distinction the public Earning type promises.
	ReportedEPS    []*float64 `json:"reportedEPS"`
	EstimatedEPS   []*float64 `json:"estimatedEPS"`
	SurpriseEPS    []*float64 `json:"surpriseEPS"`
	SurpriseEPSPct []*float64 `json:"surpriseEPSpct"`
	Updated        []int64    `json:"updated"`
}

// RequiredColumns names the symbol array toEarnings takes its row count
// from. See http.ColumnRequirer.
func (r *earningsResponse) RequiredColumns() []string { return []string{"symbol"} }

// toEarnings converts the API response to a slice of Earning.
func (r *earningsResponse) toEarnings() []Earning {
	if r == nil || len(r.Symbol) == 0 {
		return nil
	}

	earnings := make([]Earning, len(r.Symbol))
	for i := range r.Symbol {
		e := Earning{
			Symbol:        r.Symbol[i],
			FiscalYear:    safeIndexInt(r.FiscalYear, i),
			FiscalQuarter: safeIndexInt(r.FiscalQuarter, i),
			Date:          timezone.ToEastern(safeIndexInt64(r.Date, i)),
			ReportDate:    timezone.ToEastern(safeIndexInt64(r.ReportDate, i)),
			ReportTime:    safeIndexString(r.ReportTime, i),
			Currency:      safeIndexString(r.Currency, i),
			Updated:       timezone.ToEastern(safeIndexInt64(r.Updated, i)),
		}
		// Nullable EPS fields: nil means the API sent null (not yet
		// reported); a true $0.00 survives as a pointer to zero.
		e.ReportedEPS = copyFloatPtr(r.ReportedEPS, i)
		e.EstimatedEPS = copyFloatPtr(r.EstimatedEPS, i)
		e.SurpriseEPS = copyFloatPtr(r.SurpriseEPS, i)
		e.SurpriseEPSPercent = copyFloatPtr(r.SurpriseEPSPct, i)
		earnings[i] = e
	}
	return earnings
}

// NewsArticle represents a news article about a stock, returned by
// [Service.News].
type NewsArticle struct {
	// Symbol is the stock ticker symbol
	Symbol string `json:"symbol"`

	// Headline is the article headline
	Headline string `json:"headline"`

	// Content is the article content (may be partial)
	Content string `json:"content"`

	// Source is the URL where the article was published
	Source string `json:"source"`

	// PublicationDate is when the article was published
	PublicationDate time.Time `json:"publicationDate"`

	// Updated is the response-level timestamp the API sends alongside the
	// news list (the same value on every article in one News call — the
	// API does not report a per-article update time).
	Updated time.Time `json:"updated"`
}

// String returns a summary of the news article.
func (n NewsArticle) String() string {
	return fmt.Sprintf("%s [%s] %s Source: %s Content: %s", n.Symbol, n.PublicationDate.Format("2006-01-02"), n.Headline, n.Source, n.Content)
}

// newsResponse is the API response for news. Updated is a response-level
// scalar (verified live 2026-08-05) — one timestamp for the whole list, not
// a per-article array like every other field here — so toNewsArticles
// copies it onto every returned NewsArticle instead of indexing into it.
type newsResponse struct {
	Status          string   `json:"s"`
	Symbol          []string `json:"symbol"`
	Headline        []string `json:"headline"`
	Content         []string `json:"content"`
	Source          []string `json:"source"`
	PublicationDate []int64  `json:"publicationDate"`
	Updated         int64    `json:"updated"`
}

// RequiredColumns names the symbol array toNewsArticles takes its row count
// from. See http.ColumnRequirer.
func (r *newsResponse) RequiredColumns() []string { return []string{"symbol"} }

// toNewsArticles converts the API response to a slice of NewsArticle.
func (r *newsResponse) toNewsArticles() []NewsArticle {
	if r == nil || len(r.Symbol) == 0 {
		return nil
	}

	updated := timezone.ToEastern(r.Updated)
	articles := make([]NewsArticle, len(r.Symbol))
	for i := range r.Symbol {
		articles[i] = NewsArticle{
			Symbol:          r.Symbol[i],
			Headline:        safeIndexString(r.Headline, i),
			Content:         safeIndexString(r.Content, i),
			Source:          safeIndexString(r.Source, i),
			PublicationDate: timezone.ToEastern(safeIndexInt64(r.PublicationDate, i)),
			Updated:         updated,
		}
	}
	return articles
}

// BulkCandle represents a daily candle for a single symbol, returned by
// [Service.BulkCandles]. It carries the same OHLCV fields as [Candle] plus
// the symbol the candle belongs to, since bulk requests cover multiple
// symbols.
type BulkCandle struct {
	// Symbol is the stock ticker symbol
	Symbol string `json:"symbol"`

	// Time is the candle timestamp
	Time time.Time `json:"t"`

	// Open is the opening price
	Open float64 `json:"o"`

	// High is the highest price
	High float64 `json:"h"`

	// Low is the lowest price
	Low float64 `json:"l"`

	// Close is the closing price
	Close float64 `json:"c"`

	// Volume is the trading volume
	Volume int64 `json:"v"`
}

// String returns a concise summary of the bulk candle.
func (bc BulkCandle) String() string {
	return fmt.Sprintf("%s %s O: %.2f H: %.2f L: %.2f C: %.2f V: %d", bc.Symbol, bc.Time.Format("2006-01-02"), bc.Open, bc.High, bc.Low, bc.Close, bc.Volume)
}

// bulkCandlesResponse is the API response for bulk candles.
type bulkCandlesResponse struct {
	Status string    `json:"s"`
	Symbol []string  `json:"symbol"`
	Time   []int64   `json:"t"` // Unix timestamps
	Open   []float64 `json:"o"`
	High   []float64 `json:"h"`
	Low    []float64 `json:"l"`
	Close  []float64 `json:"c"`
	Volume []int64   `json:"v"`
}

// RequiredColumns names the timestamp array toBulkCandles takes its row
// count from — deliberately not symbol, which the API itself omits for a
// single-symbol request and which toBulkCandles already reconstructs. See
// http.ColumnRequirer.
func (r *bulkCandlesResponse) RequiredColumns() []string { return []string{"t"} }

// toBulkCandles converts the API response to a slice of BulkCandle.
//
// The row count comes from the timestamp array, not from symbol: for a
// single-symbol request without snapshot the API omits the symbol array
// entirely (verified live 2026-08-20 — the same request for two symbols,
// or for one with snapshot=true, does include it). Keying the conversion
// off symbol therefore discarded a valid, billed candle and returned an
// empty slice with a nil error and NoData false, so a caller batching
// symbols saw "no data" whenever a batch happened to hold exactly one.
//
// requested is the caller's symbol list, used only to restore the symbol
// the API left out; when the array is present it always wins.
func (r *bulkCandlesResponse) toBulkCandles(requested []string) []BulkCandle {
	if r == nil || len(r.Time) == 0 {
		return nil
	}

	candles := make([]BulkCandle, len(r.Time))
	for i := range r.Time {
		symbol := safeIndexString(r.Symbol, i)
		if symbol == "" && len(r.Time) == 1 && len(requested) == 1 {
			symbol = requested[0]
		}
		candles[i] = BulkCandle{
			Symbol: symbol,
			Time:   timezone.ToEastern(safeIndexInt64(r.Time, i)),
			Open:   safeIndex(r.Open, i),
			High:   safeIndex(r.High, i),
			Low:    safeIndex(r.Low, i),
			Close:  safeIndex(r.Close, i),
			Volume: safeIndexInt64(r.Volume, i),
		}
	}
	return candles
}

// safeIndexString safely accesses a string slice by index.
func safeIndexString(s []string, i int) string {
	if i < len(s) {
		return s[i]
	}
	return ""
}
