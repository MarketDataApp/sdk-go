package stocks

import (
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/params"
)

// DateWindow selects the date range for time-series stock endpoints
// ([Service.Candles], [Service.Earnings], and [Service.News]). It is a sealed
// union: build it with exactly one of the mode constructors below. Because a
// DateWindow is a single value, the API's mutually-exclusive date parameters
// (date, from, to, countback) can never be combined by mistake — an illegal
// pairing such as "from plus countback" is not expressible.
//
// The constructors mirror the ways the API accepts dates:
//
//	OnDate(d)          // a single calendar day        -> date=d
//	Between(from, to)  // an explicit closed range     -> from=from&to=to
//	Since(from)        // everything since a day        -> from=from
//	Until(to)          // up to and including a day     -> to=to
//	LastN(n)           // the n most recent periods     -> countback=n
//	LastNUntil(n, to)  // the n periods ending at a day -> countback=n&to=to
//
// Only the calendar date of each time.Time is used; the time-of-day and zone
// are ignored.
type DateWindow interface {
	window() params.Window
}

// dateWindow is the single, unexported implementation of [DateWindow]; it
// wraps the canonical internal window so callers cannot construct their own.
type dateWindow struct{ w params.Window }

func (d dateWindow) window() params.Window { return d.w }

// OnDate selects a single calendar day.
func OnDate(d time.Time) DateWindow { return dateWindow{params.OnDate(d)} }

// Between selects an explicit closed date range from..to (inclusive).
func Between(from, to time.Time) DateWindow { return dateWindow{params.Between(from, to)} }

// Since selects everything from the given day onward, letting the API default
// the end of the range to the most recent period.
func Since(from time.Time) DateWindow { return dateWindow{params.Since(from)} }

// Until selects data up to and including the given day.
func Until(to time.Time) DateWindow { return dateWindow{params.Until(to)} }

// LastN selects the n most recent periods (the API's countback parameter).
func LastN(n int) DateWindow { return dateWindow{params.LastN(n)} }

// LastNUntil selects the n periods ending at the given day.
func LastNUntil(n int, to time.Time) DateWindow { return dateWindow{params.LastNUntil(n, to)} }

// QuoteOption is a functional option for [Service.Quote] (single-symbol
// quotes). Create options with [WithFiftyTwoWeek] and [WithExtended].
//
// It is deliberately distinct from [QuotesOption]: the bulk quotes endpoint
// ignores the 52week parameter, so requesting 52-week data is only
// expressible on [Service.Quote], where it works.
type QuoteOption interface {
	apply(*quoteOptions)
}

type quoteOptionFunc func(*quoteOptions)

func (f quoteOptionFunc) apply(o *quoteOptions) { f(o) }

type quoteOptions struct {
	fiftyTwoWeek bool
	extended     *bool
	candle       *bool
}

// WithFiftyTwoWeek requests 52-week high/low data, populating the
// FiftyTwoWeekHigh and FiftyTwoWeekLow fields of the returned [Quote].
// Passing false is equivalent to omitting the option.
//
// Only [Service.Quote] supports this: the bulk quotes API endpoint ignores
// the parameter, which is why it cannot be passed to [Service.Quotes].
func WithFiftyTwoWeek(enabled bool) QuoteOption {
	return quoteOptionFunc(func(o *quoteOptions) { o.fiftyTwoWeek = enabled })
}

// WithExtended controls whether the single-symbol quote includes
// extended-hours data. When the option is omitted the parameter is not sent
// and the API default applies; passing an explicit true or false overrides
// that default. For [Service.Quotes], use [WithQuotesExtended].
func WithExtended(extended bool) QuoteOption {
	return quoteOptionFunc(func(o *quoteOptions) { o.extended = &extended })
}

// WithCandle requests the current session's OHLC alongside the quote,
// populating the Open, High, Low, and Close fields of the returned [Quote].
// The API omits those fields entirely unless this option is used, in which case
// they are zero. Passing false is equivalent to omitting the option. For
// [Service.Quotes], use [WithQuotesCandle].
func WithCandle(candle bool) QuoteOption {
	return quoteOptionFunc(func(o *quoteOptions) { o.candle = &candle })
}

// QuotesOption is a functional option for [Service.Quotes] (bulk quotes).
// Create options with [WithQuotesExtended].
//
// There is no bulk equivalent of [WithFiftyTwoWeek]: the bulk quotes API
// endpoint ignores the 52week parameter, so the option is only accepted by
// [Service.Quote], where the API honors it.
type QuotesOption interface {
	apply(*quotesOptions)
}

type quotesOptionFunc func(*quotesOptions)

func (f quotesOptionFunc) apply(o *quotesOptions) { f(o) }

type quotesOptions struct {
	extended *bool
	candle   *bool
}

// WithQuotesExtended controls whether bulk quotes include extended-hours
// data. When the option is omitted the parameter is not sent and the API
// default applies; passing an explicit true or false overrides that default.
func WithQuotesExtended(extended bool) QuotesOption {
	return quotesOptionFunc(func(o *quotesOptions) { o.extended = &extended })
}

// WithQuotesCandle requests the current session's OHLC alongside each bulk
// quote, populating the Open, High, Low, and Close fields of every returned
// [Quote]. Unlike [WithFiftyTwoWeek], the bulk endpoint does honor this
// parameter. Passing false is equivalent to omitting the option.
func WithQuotesCandle(candle bool) QuotesOption {
	return quotesOptionFunc(func(o *quotesOptions) { o.candle = &candle })
}

// CandleOption is a functional option for [Service.Candles]. Create options
// with [WithResolution], [WithCandleWindow], [WithCandleExtended], and
// [WithCandleAdjustSplits]. The date range is a single [DateWindow] value, so
// the previously combinable from/to/countback footguns cannot be written.
type CandleOption interface {
	apply(*candleOptions)
}

type candleOptionFunc func(*candleOptions)

func (f candleOptionFunc) apply(o *candleOptions) { f(o) }

type candleOptions struct {
	resolution      Resolution
	window          params.Window
	extended        *bool
	adjustSplits    *bool
	adjustDividends *bool
}

func defaultCandleOptions() *candleOptions {
	return &candleOptions{resolution: ResolutionDaily}
}

// WithResolution sets the candle resolution/timeframe, such as [Resolution5Min]
// or [ResolutionWeekly]. When omitted, [Service.Candles] defaults to
// [ResolutionDaily].
func WithResolution(r Resolution) CandleOption {
	return candleOptionFunc(func(o *candleOptions) { o.resolution = r })
}

// WithCandleWindow sets the date range for candle data as a single
// [DateWindow] value (for example stocks.Between(from, to) or stocks.LastN(30)).
// Omitting it lets the API return its default recent window.
func WithCandleWindow(w DateWindow) CandleOption {
	return candleOptionFunc(func(o *candleOptions) { o.window = w.window() })
}

// WithCandleExtended controls whether candles include extended-hours data.
// When omitted the parameter is not sent and the API default applies.
func WithCandleExtended(extended bool) CandleOption {
	return candleOptionFunc(func(o *candleOptions) { o.extended = &extended })
}

// WithCandleAdjustSplits controls whether candle data is adjusted for stock
// splits. When omitted the parameter is not sent and the API default applies.
func WithCandleAdjustSplits(adjust bool) CandleOption {
	return candleOptionFunc(func(o *candleOptions) { o.adjustSplits = &adjust })
}

// WithCandleAdjustDividends controls whether candle data is adjusted for
// dividends. The API adjusts for dividends by default; pass false to receive
// raw, unadjusted prices. When omitted the parameter is not sent and the API
// default applies.
func WithCandleAdjustDividends(adjust bool) CandleOption {
	return candleOptionFunc(func(o *candleOptions) { o.adjustDividends = &adjust })
}

// EarningsOption is a functional option for [Service.Earnings]. The reporting
// window is set with [WithEarningsWindow] using a single [DateWindow] value.
type EarningsOption interface {
	apply(*earningsOptions)
}

type earningsOptionFunc func(*earningsOptions)

func (f earningsOptionFunc) apply(o *earningsOptions) { f(o) }

type earningsOptions struct {
	report string
	window params.Window
}

// WithEarningsWindow sets the date range for earnings reports as a single
// [DateWindow] value (for example stocks.OnDate(d) or stocks.LastN(4)).
func WithEarningsWindow(w DateWindow) EarningsOption {
	return earningsOptionFunc(func(o *earningsOptions) { o.window = w.window() })
}

// WithEarningsReport limits the earnings request to a single fiscal report,
// identified the way the API names them (for example "2024-Q1"). An empty
// string leaves the parameter unset.
//
// The API declares this parameter but does not currently act on it — a live
// probe with report=2024-Q1 returned the endpoint's default row unchanged
// (verified 2026-08-11). It is exposed for parity with sdk-java, which carries
// the same field, so a caller's request survives unchanged once the API
// honors it.
func WithEarningsReport(report string) EarningsOption {
	return earningsOptionFunc(func(o *earningsOptions) { o.report = report })
}

// NewsOption is a functional option for [Service.News]. The date window is set
// with [WithNewsWindow] using a single [DateWindow] value.
type NewsOption interface {
	apply(*newsOptions)
}

type newsOptionFunc func(*newsOptions)

func (f newsOptionFunc) apply(o *newsOptions) { f(o) }

type newsOptions struct {
	window params.Window
}

// WithNewsWindow sets the date range for news articles as a single
// [DateWindow] value (for example stocks.OnDate(d) or stocks.Between(a, b)).
func WithNewsWindow(w DateWindow) NewsOption {
	return newsOptionFunc(func(o *newsOptions) { o.window = w.window() })
}

// PriceOption is a functional option for [Service.Prices]. Create options with
// [WithPriceExtended].
type PriceOption interface {
	apply(*priceOptions)
}

type priceOptionFunc func(*priceOptions)

func (f priceOptionFunc) apply(o *priceOptions) { f(o) }

type priceOptions struct {
	extended *bool
}

// WithPriceExtended controls whether prices include extended-hours data. When
// omitted the parameter is not sent and the API default applies.
func WithPriceExtended(extended bool) PriceOption {
	return priceOptionFunc(func(o *priceOptions) { o.extended = &extended })
}

// BulkCandleOption is a functional option for [Service.BulkCandles]. Create
// options with [WithBulkDate], [WithBulkResolution], [WithAdjustSplits], and
// [WithSnapshot].
type BulkCandleOption interface {
	apply(*bulkCandleOptions)
}

type bulkCandleOptionFunc func(*bulkCandleOptions)

func (f bulkCandleOptionFunc) apply(o *bulkCandleOptions) { f(o) }

type bulkCandleOptions struct {
	resolution      Resolution
	date            time.Time
	adjustSplits    *bool
	adjustDividends *bool
	snapshot        *bool
}

// WithBulkDate requests bulk candles for a specific historical date instead of
// the most recent trading day. Only the calendar date is used. The bulk
// candles endpoint accepts a single date only; it has no range or countback
// mode, so this is a plain option rather than a [DateWindow].
func WithBulkDate(t time.Time) BulkCandleOption {
	return bulkCandleOptionFunc(func(o *bulkCandleOptions) { o.date = t })
}

// WithBulkResolution sets the resolution for bulk candles. The endpoint only
// supports [ResolutionDaily], which is also the default.
func WithBulkResolution(r Resolution) BulkCandleOption {
	return bulkCandleOptionFunc(func(o *bulkCandleOptions) { o.resolution = r })
}

// WithAdjustSplits controls whether bulk candle data is adjusted for stock
// splits. When omitted the parameter is not sent and the API default applies.
func WithAdjustSplits(adjust bool) BulkCandleOption {
	return bulkCandleOptionFunc(func(o *bulkCandleOptions) { o.adjustSplits = &adjust })
}

// WithAdjustDividends controls whether bulk candle data is adjusted for
// dividends. The API adjusts for dividends by default; pass false to receive
// raw, unadjusted prices. When omitted the parameter is not sent and the API
// default applies.
func WithAdjustDividends(adjust bool) BulkCandleOption {
	return bulkCandleOptionFunc(func(o *bulkCandleOptions) { o.adjustDividends = &adjust })
}

// WithSnapshot controls whether the API returns a snapshot of the latest
// candle. When omitted the parameter is not sent and the API default applies.
func WithSnapshot(snapshot bool) BulkCandleOption {
	return bulkCandleOptionFunc(func(o *bulkCandleOptions) { o.snapshot = &snapshot })
}
