package markets

import (
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/params"
)

// StatusOption is a functional option for [Service.Status] and
// [Service.GetStatus], which report the market status for a single day.
// Options are applied in the order given, so a later option overrides an
// earlier one that sets the same parameter. Create options with [WithDate]
// (single calendar day) and [WithCountry]. StatusOption is a sealed
// interface: only this package can implement it, and its date parameter is a
// single day, so the range parameters used by [Service.StatusHistory] cannot
// be passed here by mistake.
type StatusOption interface {
	applyStatus(*statusOptions)
}

type statusOptionFunc func(*statusOptions)

func (f statusOptionFunc) applyStatus(o *statusOptions) { f(o) }

type statusOptions struct {
	date    time.Time
	country string
}

// WithDate sets the specific day whose market status is requested by
// [Service.Status]. The date is sent to the API in YYYY-MM-DD form, so any
// time-of-day component of d is ignored; a zero time leaves the parameter
// unset and the API reports today's status. WithDate is a Status-only
// option: it cannot be passed to [Service.StatusHistory], whose date range
// is selected with [WithHistoryWindow].
func WithDate(d time.Time) StatusOption {
	return statusOptionFunc(func(o *statusOptions) {
		o.date = d
	})
}

// HistoryWindow selects the date range for [Service.StatusHistory]. It is a
// sealed union: build it with exactly one of the mode constructors below.
// Because a HistoryWindow is a single value, the API's mutually-exclusive
// range parameters (from, to, countback) can never be combined by mistake —
// an illegal pairing such as "from plus countback" is not expressible.
//
// The constructors mirror the ways the status-history endpoint accepts a
// range:
//
//	Between(from, to)  // an explicit closed range     -> from=from&to=to
//	Since(from)        // everything since a day        -> from=from
//	Until(to)          // up to and including a day     -> to=to
//	LastN(n)           // the n most recent days         -> countback=n
//	LastNUntil(n, to)  // the n days ending at a day     -> countback=n&to=to
//
// There is deliberately no single-date mode here: a single calendar day is a
// [Service.Status] concept, selected with [WithDate]. Only the calendar date
// of each time.Time is used; the time-of-day and zone are ignored.
type HistoryWindow interface {
	window() params.Window
}

// historyWindow is the single, unexported implementation of [HistoryWindow];
// it wraps the canonical internal window so callers cannot construct their own.
type historyWindow struct{ w params.Window }

func (h historyWindow) window() params.Window { return h.w }

// Between selects an explicit closed date range from..to (inclusive).
func Between(from, to time.Time) HistoryWindow { return historyWindow{params.Between(from, to)} }

// Since selects everything from the given day onward, letting the API default
// the end of the range to the most recent day.
func Since(from time.Time) HistoryWindow { return historyWindow{params.Since(from)} }

// Until selects data up to and including the given day.
func Until(to time.Time) HistoryWindow { return historyWindow{params.Until(to)} }

// LastN selects the n most recent days (the API's countback parameter).
//
// This endpoint returns n+1 rows, consistently at every n — verified live
// 2026-08-20 with countback 1, 2, 3, 5 and 10 returning 2, 3, 4, 6 and 11,
// with and without a to anchor. Every other endpoint's countback is exact.
// The SDK does not compensate: it already adjusts for two sibling countback
// defects, but in both of those the API IGNORES the parameter, whereas here
// it honors it with an off-by-one — so subtracting one would break silently
// the day the API is corrected. Tracked in
// integration/discrepancy_test.go; slice the result if you need exactly n.
func LastN(n int) HistoryWindow { return historyWindow{params.LastN(n)} }

// LastNUntil selects the n days ending at the given day.
func LastNUntil(n int, to time.Time) HistoryWindow { return historyWindow{params.LastNUntil(n, to)} }

// HistoryOption is a functional option for [Service.StatusHistory], which
// reports the market status for each day in a range. Options are applied in
// the order given, so a later option overrides an earlier one that sets the
// same parameter. Create options with [WithHistoryWindow] (the date range)
// and [WithCountry]. HistoryOption is a sealed interface: only this package
// can implement it, and its range is a single [HistoryWindow] value, so the
// single-date parameter used by [Service.Status] cannot be passed here by
// mistake.
type HistoryOption interface {
	applyHistory(*historyOptions)
}

type historyOptionFunc func(*historyOptions)

func (f historyOptionFunc) applyHistory(o *historyOptions) { f(o) }

type historyOptions struct {
	window  params.Window
	country string
}

// WithHistoryWindow sets the date range for [Service.StatusHistory] as a
// single [HistoryWindow] value (for example markets.Between(from, to) or
// markets.LastN(5)). Omitting it lets the API return its default recent
// range. WithHistoryWindow is a StatusHistory-only option: it cannot be
// passed to [Service.Status], whose single day is selected with [WithDate].
func WithHistoryWindow(w HistoryWindow) HistoryOption {
	return historyOptionFunc(func(o *historyOptions) {
		o.window = w.window()
	})
}

// CountryOption is the option returned by [WithCountry]. It satisfies both
// [StatusOption] and [HistoryOption], which is why a single WithCountry call
// can be passed to either [Service.Status] or [Service.StatusHistory]. It is
// exported only so that its dual role has a name; construct it with
// [WithCountry] rather than as a literal.
type CountryOption struct {
	country string
}

func (c CountryOption) applyStatus(o *statusOptions)   { o.country = c.country }
func (c CountryOption) applyHistory(o *historyOptions) { o.country = c.country }

// WithCountry selects which country's market to query, using a two-letter ISO
// 3166-1 alpha-2 code such as "US". An empty string leaves the parameter
// unset and the API defaults to the United States. The returned
// [CountryOption] satisfies both [StatusOption] and [HistoryOption], so
// WithCountry applies to both [Service.Status] and [Service.StatusHistory].
func WithCountry(country string) CountryOption {
	return CountryOption{country: country}
}
