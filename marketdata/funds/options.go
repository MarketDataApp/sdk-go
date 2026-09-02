package funds

import (
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/params"
)

// DateWindow selects the date range for [Service.Candles]. It is a sealed
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

// CandleOption is a functional option for [Service.Candles] and
// [Service.GetCandles]. Create options with [WithResolution] and
// [WithCandleWindow]. The date range is a single [DateWindow] value, so the
// previously combinable from/to/countback footguns cannot be written.
type CandleOption interface {
	apply(*candleOptions)
}

type candleOptionFunc func(*candleOptions)

func (f candleOptionFunc) apply(o *candleOptions) { f(o) }

type candleOptions struct {
	resolution Resolution
	window     params.Window
}

func defaultCandleOptions() *candleOptions {
	return &candleOptions{
		resolution: ResolutionDaily,
	}
}

// WithResolution sets the candle resolution, that is, the duration of
// each candle. The predefined values are [ResolutionDaily] (the
// default), [ResolutionWeekly], [ResolutionMonthly], and
// [ResolutionYearly]. The resolution becomes part of the request path,
// so an unsupported value causes the API to reject the request.
func WithResolution(r Resolution) CandleOption {
	return candleOptionFunc(func(o *candleOptions) {
		o.resolution = r
	})
}

// WithCandleWindow sets the date range for candle data as a single
// [DateWindow] value (for example funds.Between(from, to) or funds.LastN(30)).
// Omitting it lets the API return its default recent window.
func WithCandleWindow(w DateWindow) CandleOption {
	return candleOptionFunc(func(o *candleOptions) { o.window = w.window() })
}
