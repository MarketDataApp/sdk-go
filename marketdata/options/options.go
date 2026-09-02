package options

import (
	"net/url"
	"strings"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
	"github.com/MarketDataApp/sdk-go/v2/internal/timezone"
)

// dateLayout is the calendar-date form the options endpoints accept for
// date, from, to, and expiration parameters.
const dateLayout = "2006-01-02"

// validationErr builds a pre-network validation error for a rejected option
// value. It returns a [sdkerrors.ValidationError] so callers can reject the
// request before any HTTP call is made.
func validationErr(field, msg string) error {
	return &sdkerrors.ValidationError{Field: field, Message: msg}
}

// ---------------------------------------------------------------------------
// StrikeFilter — the contract selector (strike / delta), a sealed union.
// ---------------------------------------------------------------------------

// StrikeFilter selects which contracts a [Service.Chain] request returns by
// strike price or by delta. It is a sealed union: build it with exactly one
// of the mode constructors below and pass it to [WithStrike]. Because a
// StrikeFilter is a single value, the API's mutually-exclusive contract
// selectors can never be combined by mistake — the API silently honors strike
// over delta when both are sent, and that footgun is not expressible here.
//
// The constructors mirror the ways the API accepts a strike filter:
//
//	Strike(x)          // an exact strike            -> strike=x
//	Strikes(x, y, ...) // several exact strikes      -> strike=x,y,...
//	StrikeRange(lo, hi)// an inclusive strike range  -> strike=lo-hi
//	MinStrike(x)       // strikes at or above x      -> strike=>=x
//	MaxStrike(x)       // strikes at or below x      -> strike=<=x
//	StrikeExpr(expr)   // a raw strike expression    -> strike=expr
//	ByDelta(d)         // the strike nearest delta d -> delta=d
//	ByDeltas(c, d, ...)// the strikes nearest each   -> delta=c,d,...
//
// Strike prices must be greater than zero and a range's low must not exceed
// its high; a delta must be non-zero and within [-1, 1] (puts have negative
// delta). The list forms require at least one value and apply the same rules
// to every element. Values are checked before any request is made.
type StrikeFilter interface {
	strike() strikeParams
}

// strikeFilter is the single, unexported implementation of [StrikeFilter]; it
// wraps the validated internal representation so callers cannot construct
// their own.
type strikeFilter struct{ p strikeParams }

func (f strikeFilter) strike() strikeParams { return f.p }

// strikeKind identifies which strike mode a [strikeParams] carries.
type strikeKind uint8

const (
	strikeExact strikeKind = iota + 1
	strikeRangeMode
	strikeMinMode
	strikeMaxMode
	strikeExprMode
	strikeDeltaMode
	strikeListMode
	strikeDeltaListMode
)

// strikeParams is the validated internal representation of a [StrikeFilter].
type strikeParams struct {
	kind strikeKind
	lo   float64 // exact / min / delta / range low
	hi   float64 // range high / max
	expr string
	list []float64 // strike list / delta list
}

// Strike limits the chain to contracts at an exact strike price (strike=x).
func Strike(x float64) StrikeFilter { return strikeFilter{strikeParams{kind: strikeExact, lo: x}} }

// Strikes limits the chain to an explicit set of exact strike prices
// (strike=x,y,z), returning both sides at each one unless [WithSide] narrows
// it. Use it to price a spread or a specific set of legs in a single request
// rather than one call per strike. At least one strike is required and every
// value must be greater than zero.
func Strikes(strikes ...float64) StrikeFilter {
	return strikeFilter{strikeParams{kind: strikeListMode, list: strikes}}
}

// StrikeRange limits the chain to strikes between lo and hi inclusive
// (strike=lo-hi).
func StrikeRange(lo, hi float64) StrikeFilter {
	return strikeFilter{strikeParams{kind: strikeRangeMode, lo: lo, hi: hi}}
}

// MinStrike limits the chain to strikes at or above x (strike=>=x).
func MinStrike(x float64) StrikeFilter { return strikeFilter{strikeParams{kind: strikeMinMode, lo: x}} }

// MaxStrike limits the chain to strikes at or below x (strike=<=x).
func MaxStrike(x float64) StrikeFilter { return strikeFilter{strikeParams{kind: strikeMaxMode, hi: x}} }

// StrikeExpr sets the strike filter using the API's raw expression syntax,
// passed through verbatim (strike=expr): an exact strike ("150"), an inclusive
// range ("140-160"), or a one-sided bound (">=140", "<=160"). It is an escape
// hatch for expressions the typed constructors do not cover.
func StrikeExpr(expr string) StrikeFilter {
	return strikeFilter{strikeParams{kind: strikeExprMode, expr: expr}}
}

// ByDelta selects the strikes nearest the given delta (delta=d). The API filters
// on the ABSOLUTE value of delta and always returns both sides (calls and puts),
// so ByDelta does not choose a side — combine it with [WithSide] to keep only
// calls or puts.
//
// The filter can silently do nothing. If ANY contract in the chain the API
// fetched carries a null delta, the whole filter is skipped and the full
// chain comes back with a 200 and no signal
// (https://github.com/MarketData-App/api/issues/352). Null greeks are not
// rare — illiquid strikes, freshly listed contracts, provider metadata rot
// — so this fires on some symbols and expirations and not others.
//
// It reads as a side limitation because that is how it first presented: on
// 2026-08-20 delta=0.30&side=call returned one contract at 0.338 while
// side=put returned 99 and no side at all returned 198, which looked like
// "calls only". It is not — on 2026-08-26 the same query returned one
// contract per side and two with no side, correctly filtered. What differed
// was whether the fetched chain happened to contain a null delta. Check the
// result rather than assuming either behavior. Tracked in
// integration/discrepancy_test.go.
//
// Delta is mutually exclusive with strike (the API silently honors
// strike when both are sent), so it is a mode of [StrikeFilter] rather than a
// separate option. d must be non-zero and within [-1, 1] (negative values are
// accepted and, per the absolute-value rule, behave the same as their positive
// counterpart).
func ByDelta(d float64) StrikeFilter { return strikeFilter{strikeParams{kind: strikeDeltaMode, lo: d}} }

// ByDeltas selects the strikes nearest each of the given deltas
// (delta=c,d,...), for example the 0.16 and 0.30 deltas of a strangle in one
// request. Like [ByDelta] it filters on the ABSOLUTE value of delta and returns
// both sides — combine it with [WithSide] to keep only calls or puts. Like
// [ByDelta] the filter is silently dropped when the fetched chain contains a
// null delta; see [ByDelta]. At least one delta is required and every value must be
// non-zero and within [-1, 1].
func ByDeltas(deltas ...float64) StrikeFilter {
	return strikeFilter{strikeParams{kind: strikeDeltaListMode, list: deltas}}
}

// validate enforces the value-range rules a StrikeFilter cannot encode in its
// type, returning a [sdkerrors.ValidationError] before any request is built.
func (p strikeParams) validate() error {
	switch p.kind {
	case strikeExact:
		if p.lo <= 0 {
			return validationErr("strike", "strike must be greater than zero")
		}
	case strikeRangeMode:
		if p.lo <= 0 || p.hi <= 0 {
			return validationErr("strike", "strike range bounds must be greater than zero")
		}
		if p.lo > p.hi {
			return validationErr("strike", "strike range low must not exceed high")
		}
	case strikeMinMode:
		if p.lo <= 0 {
			return validationErr("strike", "strike must be greater than zero")
		}
	case strikeMaxMode:
		if p.hi <= 0 {
			return validationErr("strike", "strike must be greater than zero")
		}
	case strikeExprMode:
		if p.expr == "" {
			return validationErr("strike", "strike expression must not be empty")
		}
	case strikeDeltaMode:
		return validateDelta(p.lo)
	case strikeListMode:
		if len(p.list) == 0 {
			return validationErr("strike", "at least one strike is required")
		}
		for _, s := range p.list {
			if s <= 0 {
				return validationErr("strike", "every strike must be greater than zero")
			}
		}
	case strikeDeltaListMode:
		if len(p.list) == 0 {
			return validationErr("delta", "at least one delta is required")
		}
		for _, d := range p.list {
			if err := validateDelta(d); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateDelta enforces the API's delta range on a single value, shared by the
// single-value and list delta modes.
func validateDelta(d float64) error {
	if d == 0 {
		return validationErr("delta", "delta must be non-zero")
	}
	if d < -1 || d > 1 {
		return validationErr("delta", "delta must be within [-1, 1]")
	}
	return nil
}

// joinFloats renders a value list as the API's comma-separated form.
func joinFloats(vals []float64) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = formatFloat(v)
	}
	return strings.Join(parts, ",")
}

// apply writes the strike or delta parameter into v. It assumes the params
// have already passed [strikeParams.validate].
func (p strikeParams) apply(v url.Values) {
	switch p.kind {
	case strikeExact:
		v.Set("strike", formatFloat(p.lo))
	case strikeRangeMode:
		v.Set("strike", formatFloat(p.lo)+"-"+formatFloat(p.hi))
	case strikeMinMode:
		v.Set("strike", ">="+formatFloat(p.lo))
	case strikeMaxMode:
		v.Set("strike", "<="+formatFloat(p.hi))
	case strikeExprMode:
		v.Set("strike", p.expr)
	case strikeDeltaMode:
		v.Set("delta", formatFloat(p.lo))
	case strikeListMode:
		v.Set("strike", joinFloats(p.list))
	case strikeDeltaListMode:
		v.Set("delta", joinFloats(p.list))
	}
}

// ---------------------------------------------------------------------------
// ExpiryFilter — which expirations, a sealed union.
// ---------------------------------------------------------------------------

// ExpiryFilter selects which expirations a [Service.Chain] request returns. It
// is a sealed union: build it with exactly one of the mode constructors below
// and pass it to [WithExpiry]. Because an ExpiryFilter is a single value, the
// API's mutually-exclusive expiry selectors can never be combined by mistake —
// combining expiration with dte makes the API silently honor expiration, and
// combining expiration with month/year yields an empty (broken) response.
//
// The constructors mirror the ways the API accepts an expiry filter:
//
//	AllExpirations()       // every listed expiration       -> expiration=all
//	OnExpiration(t)        // a single expiration date     -> expiration=YYYY-MM-DD
//	InDTE(days)            // the expiry nearest days out   -> dte=days
//	InMonth(month)         // every year's given month      -> month=month
//	InYear(year)           // every month of a given year   -> year=year
//	InMonthOfYear(m, y)    // one specific month and year   -> month=m&year=y
//
// month must be 1 through 12, year a four-digit year (at least 1900), and dte
// zero or greater. Values are checked before any request is made.
//
// Note that omitting the filter entirely is NOT the same as
// [AllExpirations]: with no expiry filter the chain endpoint returns only the
// front-month expiration.
type ExpiryFilter interface {
	expiry() expiryParams
}

// expiryFilter is the single, unexported implementation of [ExpiryFilter].
type expiryFilter struct{ p expiryParams }

func (f expiryFilter) expiry() expiryParams { return f.p }

// expiryKind identifies which expiry mode an [expiryParams] carries.
type expiryKind uint8

const (
	expiryOnDate expiryKind = iota + 1
	expiryDTE
	expiryMonth
	expiryYear
	expiryMonthOfYear
	expiryRange
	expiryAll
)

// expiryParams is the validated internal representation of an [ExpiryFilter].
type expiryParams struct {
	kind  expiryKind
	exp   time.Time
	dte   int
	month int
	year  int
	from  time.Time
	to    time.Time
}

// AllExpirations requests the whole chain across every listed expiration
// (expiration=all).
//
// This is not the same as omitting the expiry filter: with no expiry filter
// the chain endpoint returns only the front-month (nearest) expiration, so
// AllExpirations is the only way to obtain the complete chain. For a
// liquid underlying the difference is large — an AAPL chain returns roughly
// 190 contracts across 1 expiration unfiltered, against roughly 3,500
// contracts across 24 expirations with AllExpirations — so the request is
// correspondingly more expensive in API credits. The independent filters
// (side, strike, the liquidity filters, the expiration-type filters) still
// narrow the result on top of it.
func AllExpirations() ExpiryFilter {
	return expiryFilter{expiryParams{kind: expiryAll}}
}

// OnExpiration limits the chain to contracts expiring on the given date
// (expiration=YYYY-MM-DD). Only the calendar date is used.
func OnExpiration(t time.Time) ExpiryFilter {
	return expiryFilter{expiryParams{kind: expiryOnDate, exp: t}}
}

// InDTE limits the chain to the expiration closest to the given number of days
// to expiry, counted from today (dte=days).
func InDTE(days int) ExpiryFilter { return expiryFilter{expiryParams{kind: expiryDTE, dte: days}} }

// InMonth limits the chain to contracts expiring in the given calendar month
// across all years (month=month), expressed as 1 through 12.
func InMonth(month int) ExpiryFilter {
	return expiryFilter{expiryParams{kind: expiryMonth, month: month}}
}

// InYear limits the chain to contracts expiring in the given four-digit year
// across all months (year=year).
func InYear(year int) ExpiryFilter {
	return expiryFilter{expiryParams{kind: expiryYear, year: year}}
}

// InMonthOfYear limits the chain to contracts expiring in a specific month of
// a specific year (month=month&year=year).
func InMonthOfYear(month, year int) ExpiryFilter {
	return expiryFilter{expiryParams{kind: expiryMonthOfYear, month: month, year: year}}
}

// ExpirationBetween limits the chain to contracts expiring within an explicit,
// inclusive date range (from=YYYY-MM-DD&to=YYYY-MM-DD). Only the calendar dates
// are used. It selects which expirations are returned and so is mutually
// exclusive with the other expiry selectors (the API forbids combining a
// from/to expiration range with dte).
func ExpirationBetween(from, to time.Time) ExpiryFilter {
	return expiryFilter{expiryParams{kind: expiryRange, from: from, to: to}}
}

// validateMonth reports an error if month is outside 1..12.
func validateMonth(month int) error {
	if month < 1 || month > 12 {
		return validationErr("month", "month must be between 1 and 12")
	}
	return nil
}

// validateYear reports an error if year is not a four-digit year of at least
// 1900.
func validateYear(year int) error {
	if year < 1900 || year > 9999 {
		return validationErr("year", "year must be a four-digit year of at least 1900")
	}
	return nil
}

// validate enforces the value-range rules an ExpiryFilter cannot encode in its
// type, returning a [sdkerrors.ValidationError] before any request is built.
func (p expiryParams) validate() error {
	switch p.kind {
	case expiryOnDate:
		if p.exp.IsZero() {
			return validationErr("expiration", "expiration date must not be zero")
		}
	case expiryDTE:
		if p.dte < 0 {
			return validationErr("dte", "dte must be zero or greater")
		}
	case expiryMonth:
		return validateMonth(p.month)
	case expiryYear:
		return validateYear(p.year)
	case expiryMonthOfYear:
		if err := validateMonth(p.month); err != nil {
			return err
		}
		return validateYear(p.year)
	case expiryRange:
		if p.from.IsZero() || p.to.IsZero() {
			return validationErr("expiration", "both from and to must be set")
		}
		if p.from.After(p.to) {
			return validationErr("expiration", "from must not be after to")
		}
	}
	return nil
}

// apply writes the expiry parameters into v. It assumes the params have
// already passed [expiryParams.validate].
func (p expiryParams) apply(v url.Values) {
	switch p.kind {
	case expiryAll:
		v.Set("expiration", "all")
	case expiryOnDate:
		v.Set("expiration", p.exp.Format(dateLayout))
	case expiryDTE:
		v.Set("dte", formatInt(p.dte))
	case expiryMonth:
		v.Set("month", formatInt(p.month))
	case expiryYear:
		v.Set("year", formatInt(p.year))
	case expiryMonthOfYear:
		v.Set("month", formatInt(p.month))
		v.Set("year", formatInt(p.year))
	case expiryRange:
		v.Set("from", p.from.Format(dateLayout))
		v.Set("to", p.to.Format(dateLayout))
	}
}

// ---------------------------------------------------------------------------
// dateOrRange — shared internal representation for the historical single-
// contract quote window ([OptionQuoteWindow]).
// ---------------------------------------------------------------------------

// dateRangeKind identifies which historical-date mode a [dateOrRange] carries.
type dateRangeKind uint8

const (
	dateSingle dateRangeKind = iota + 1
	dateSpan
	dateCountback
	dateCountbackUntil
)

// dateOrRange is the validated internal representation used by
// [OptionQuoteWindow]: either a single date or an explicit from/to span.
type dateOrRange struct {
	kind      dateRangeKind
	date      time.Time
	from      time.Time
	to        time.Time
	countback int
}

func newDateSingle(t time.Time) dateOrRange { return dateOrRange{kind: dateSingle, date: t} }

func newDateSpan(from, to time.Time) dateOrRange {
	return dateOrRange{kind: dateSpan, from: from, to: to}
}

func newDateCountback(n int) dateOrRange {
	return dateOrRange{kind: dateCountback, countback: n}
}

func newDateCountbackUntil(n int, to time.Time) dateOrRange {
	return dateOrRange{kind: dateCountbackUntil, countback: n, to: to}
}

// validate enforces that the required dates are present and ordered, returning
// a [sdkerrors.ValidationError] before any request is built. field names the
// single-date parameter for clearer error messages.
func (d dateOrRange) validate(field string) error {
	switch d.kind {
	case dateSingle:
		if d.date.IsZero() {
			return validationErr(field, field+" must not be zero")
		}
	case dateSpan:
		if d.from.IsZero() || d.to.IsZero() {
			return validationErr("window", "both from and to must be set")
		}
		if truncateDay(d.from).After(truncateDay(d.to)) {
			return validationErr("window", "from must not be after to")
		}
	case dateCountback:
		if d.countback <= 0 {
			return validationErr("countback", "countback must be greater than zero")
		}
	case dateCountbackUntil:
		if d.countback <= 0 {
			return validationErr("countback", "countback must be greater than zero")
		}
		if d.to.IsZero() {
			return validationErr("to", "to must not be zero")
		}
	}
	return nil
}

// apply writes the date parameters into v. It assumes the value has already
// passed [dateOrRange.validate].
func (d dateOrRange) apply(v url.Values) {
	switch d.kind {
	case dateSingle:
		v.Set("date", d.date.Format(dateLayout))
	case dateSpan:
		v.Set("from", d.from.Format(dateLayout))
		v.Set("to", d.to.Format(dateLayout))
	case dateCountback:
		// A bare countback is silently ignored by this endpoint (verified
		// live 2026-08-11: countback=3 alone returns 1 row, the current
		// quote, while countback=3&to=... returns 3). Anchoring it with an
		// explicit to= of today preserves the "n most recent" meaning and
		// stays harmless if the API ever honors a bare countback. Same
		// defect and same remedy as stocks/earnings.
		v.Set("countback", formatInt(d.countback))
		v.Set("to", time.Now().In(timezone.Eastern).Format(dateLayout))
	case dateCountbackUntil:
		v.Set("countback", formatInt(d.countback))
		v.Set("to", d.to.Format(dateLayout))
	}
}

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// The chain's historical "as of" selector is a single calendar date exposed as
// the plain [WithChainDate] option (defined with the other chain options
// below), not a union: per the API, a chain snapshot is taken as of one day.
// A from/to date range on the chain filters expirations, not history, and is
// therefore an [ExpiryFilter] mode ([ExpirationBetween]).

// ---------------------------------------------------------------------------
// OptionQuoteWindow — historical single-contract quote, a sealed union.
// ---------------------------------------------------------------------------

// OptionQuoteWindow selects the date range for a [Service.Quote] request. It
// is a sealed union: build it with exactly one of the mode constructors below
// and pass it to [WithOptionQuoteWindow]. Because an OptionQuoteWindow is a
// single value, the API's mutually-exclusive date parameters can never be
// combined by mistake — sending both date and from/to returns HTTP 400
// ("Invalid date. Use either date, or from and to.").
//
// The constructors mirror the ways the API accepts a quote window:
//
//	QuoteOnDate(t)            // a single historical date -> date=YYYY-MM-DD
//	QuoteRange(from, to)      // an explicit date range    -> from=...&to=...
//	QuoteLastN(n)             // the n most recent quotes  -> countback=n
//	QuoteLastNUntil(n, to)    // n quotes ending at to     -> countback=n&to=...
//
// Only the calendar date of each time.Time is used. A countback must be
// greater than zero. The countback modes mirror the stocks and funds date
// windows: the API pairs countback with to, never with from, and that pairing
// is the only one expressible here.
type OptionQuoteWindow interface {
	optionQuoteWindow() dateOrRange
}

// optQuoteWindow is the single, unexported implementation of
// [OptionQuoteWindow].
type optQuoteWindow struct{ d dateOrRange }

func (o optQuoteWindow) optionQuoteWindow() dateOrRange { return o.d }

// QuoteOnDate requests the contract's quote on a single historical date
// (date=YYYY-MM-DD).
func QuoteOnDate(t time.Time) OptionQuoteWindow { return optQuoteWindow{newDateSingle(t)} }

// QuoteRange requests the contract's quotes across an explicit date range
// (from=...&to=...).
func QuoteRange(from, to time.Time) OptionQuoteWindow {
	return optQuoteWindow{newDateSpan(from, to)}
}

// QuoteLastN requests n quotes for the contract (countback=n), sent with an
// explicit to= anchor of today's date in Eastern time. n must be greater than
// zero.
//
// The anchor is required: this endpoint ignores a countback that arrives
// without a to=, returning a single current quote instead of n.
//
// # Known API defect
//
// The endpoint currently returns the n OLDEST quotes of the contract's
// history rather than the n most recent, and ignores the to= anchor when
// choosing them (verified live 2026-08-11: countback=3 with to= of
// 2026-08-11, 2026-08-08, and 2025-08-13 all return the same three earliest rows).
// Until that is fixed, prefer [QuoteRange] when you need a specific period.
// The SDK sends the parameters exactly as documented and does not attempt to
// compensate; integration/discrepancy_test.go carries the strict assertion
// that will pass once the API is corrected.
func QuoteLastN(n int) OptionQuoteWindow { return optQuoteWindow{newDateCountback(n)} }

// QuoteLastNUntil requests n quotes ending on the given date
// (countback=n&to=YYYY-MM-DD). n must be greater than zero and to must not be
// zero. Only the calendar date of to is used.
//
// See [QuoteLastN] for the known API defect that currently makes this endpoint
// ignore the to= anchor and return the contract's oldest quotes.
func QuoteLastNUntil(n int, to time.Time) OptionQuoteWindow {
	return optQuoteWindow{newDateCountbackUntil(n, to)}
}

// ---------------------------------------------------------------------------
// Chain options.
// ---------------------------------------------------------------------------

// ChainOption is a functional option that filters or otherwise refines a
// [Service.Chain] request. Chain options combine: pass several to narrow the
// chain by expiration, strike, side, liquidity, and more. The two
// mutually-exclusive parameter groups are each collapsed into a single
// sealed-union option — [WithStrike] and [WithExpiry] — so the API's
// silently-conflicting combinations cannot be written.
type ChainOption interface {
	apply(*chainOptions)
}

type chainOptionFunc func(*chainOptions)

func (f chainOptionFunc) apply(o *chainOptions) {
	f(o)
}

type chainOptions struct {
	// Sealed-union selectors (nil when unset).
	strike StrikeFilter
	expiry ExpiryFilter

	// Historical "as of" date (zero when unset); independent of the selectors.
	date time.Time

	// Independent free filters.
	side               OptionSide
	strikeLimit        int
	rangeFilter        Moneyness
	expTypes           ExpirationTypeFilter
	minBid             *float64
	maxBid             *float64
	minAsk             *float64
	maxAsk             *float64
	maxBidAskSpread    *float64
	maxBidAskSpreadPct *float64
	minOpenInterest    *int
	minVolume          *int
	nonstandard        *bool
	am                 *bool
	pm                 *bool
}

func defaultChainOptions() *chainOptions {
	return &chainOptions{}
}

// WithStrike sets the chain's contract selector from a single [StrikeFilter]
// value, such as options.Strike(150), options.StrikeRange(150, 160), or
// options.ByDelta(0.30). It replaces the API's mutually-exclusive strike and
// delta parameters with one option, so they can never conflict.
func WithStrike(f StrikeFilter) ChainOption {
	return chainOptionFunc(func(o *chainOptions) {
		o.strike = f
	})
}

// WithExpiry sets the chain's expiration selector from a single [ExpiryFilter]
// value, such as options.OnExpiration(t), options.InDTE(45), or
// options.InMonthOfYear(6, 2026). It replaces the API's mutually-exclusive
// expiration, dte, month, and year parameters with one option.
func WithExpiry(f ExpiryFilter) ChainOption {
	return chainOptionFunc(func(o *chainOptions) {
		o.expiry = f
	})
}

// WithChainDate requests the option chain as it stood on a single historical
// trading day (date=YYYY-MM-DD). Only the calendar date is used; a zero time
// returns the current chain. It is independent of the expiration selectors, so
// a historical chain can still be narrowed by expiration or strike. (A from/to
// date range filters expirations, not history — use
// options.ExpirationBetween via [WithExpiry] for that.)
func WithChainDate(t time.Time) ChainOption {
	return chainOptionFunc(func(o *chainOptions) {
		o.date = t
	})
}

// WithSide limits the chain to one side of the market: [SideCall] for calls
// only or [SidePut] for puts only. Passing [SideBoth] leaves the parameter
// unset, which returns both sides (the default).
func WithSide(side OptionSide) ChainOption {
	return chainOptionFunc(func(o *chainOptions) {
		o.side = side
	})
}

// ExpirationType identifies an expiration cadence for the chain's
// expiration-type filter ([WithExpirationTypes]).
type ExpirationType string

const (
	// Weekly is a weekly expiration cadence.
	Weekly ExpirationType = "weekly"
	// Monthly is a standard monthly expiration cadence.
	Monthly ExpirationType = "monthly"
	// Quarterly is a quarterly expiration cadence.
	Quarterly ExpirationType = "quarterly"
)

// ExpirationTypeFilter includes or excludes expiration cadences (weekly,
// monthly, quarterly) from the chain. It is a sealed union built with exactly
// one of [IncludeExpirationTypes] or [ExcludeExpirationTypes]. The API forbids
// mixing inclusion and exclusion of expiration types in one request
// ("weekly=true&monthly=false" is an error); because the choice of include vs.
// exclude is a single value here, that illegal mix cannot be written.
type ExpirationTypeFilter interface {
	expirationTypes() (include bool, types []ExpirationType)
}

type expTypeFilter struct {
	include bool
	types   []ExpirationType
}

func (f expTypeFilter) expirationTypes() (bool, []ExpirationType) { return f.include, f.types }

// IncludeExpirationTypes limits the chain to only the given expiration cadences
// (for example IncludeExpirationTypes(options.Weekly, options.Monthly) sends
// weekly=true&monthly=true). Passing none is a no-op.
func IncludeExpirationTypes(types ...ExpirationType) ExpirationTypeFilter {
	return expTypeFilter{include: true, types: types}
}

// ExcludeExpirationTypes excludes the given expiration cadences from the chain
// (for example ExcludeExpirationTypes(options.Quarterly) sends quarterly=false).
// Passing none is a no-op.
func ExcludeExpirationTypes(types ...ExpirationType) ExpirationTypeFilter {
	return expTypeFilter{include: false, types: types}
}

// WithExpirationTypes filters the chain by expiration cadence, either including
// or excluding a set of types via a single [ExpirationTypeFilter] value. Because
// inclusion and exclusion are one value, the API's forbidden include/exclude mix
// cannot be expressed.
func WithExpirationTypes(f ExpirationTypeFilter) ChainOption {
	return chainOptionFunc(func(o *chainOptions) {
		o.expTypes = f
	})
}

// WithStrikeLimit limits the chain to the n strikes nearest the money on
// EACH side of it, so a request can come back with up to 2n distinct strikes,
// not n (verified live 2026-08-20: strikeLimit=1 returned 2 distinct strikes,
// 2 returned 4, 3 returned 6). Values less than or equal to zero leave the
// limit unset.
func WithStrikeLimit(limit int) ChainOption {
	return chainOptionFunc(func(o *chainOptions) {
		o.strikeLimit = limit
	})
}

// Moneyness is the chain's moneyness filter, passed to [WithRange]. It is a
// closed set of API keywords rather than a free string, so a typo cannot reach
// the wire as a silently-ignored filter.
type Moneyness string

const (
	// MoneynessITM limits the chain to in-the-money contracts.
	MoneynessITM Moneyness = "itm"
	// MoneynessOTM limits the chain to out-of-the-money contracts.
	MoneynessOTM Moneyness = "otm"
	// MoneynessAll requests every contract regardless of moneyness (the
	// API default).
	MoneynessAll Moneyness = "all"
	// MoneynessUnset leaves the filter off, equivalent to omitting the option.
	MoneynessUnset Moneyness = ""
)

// WithRange filters the chain by moneyness, using one of [MoneynessITM],
// [MoneynessOTM], or [MoneynessAll]. [MoneynessUnset] leaves the filter off.
// Pair it with [WithStrikeLimit] to ask for "the N strikes around the money".
func WithRange(r Moneyness) ChainOption {
	return chainOptionFunc(func(o *chainOptions) {
		o.rangeFilter = r
	})
}

// WithMinBid limits the chain to contracts whose bid price is greater than or
// equal to min. Useful for excluding illiquid or worthless contracts.
func WithMinBid(min float64) ChainOption {
	return chainOptionFunc(func(o *chainOptions) {
		o.minBid = &min
	})
}

// WithMaxBid limits the chain to contracts whose bid price is less than or
// equal to max.
func WithMaxBid(max float64) ChainOption {
	return chainOptionFunc(func(o *chainOptions) {
		o.maxBid = &max
	})
}

// WithMinAsk limits the chain to contracts whose ask price is greater than or
// equal to min.
func WithMinAsk(min float64) ChainOption {
	return chainOptionFunc(func(o *chainOptions) {
		o.minAsk = &min
	})
}

// WithMaxAsk limits the chain to contracts whose ask price is less than or
// equal to max.
func WithMaxAsk(max float64) ChainOption {
	return chainOptionFunc(func(o *chainOptions) {
		o.maxAsk = &max
	})
}

// WithMaxBidAskSpread limits the chain to contracts whose bid-ask spread is
// less than or equal to max, expressed in dollars. See [WithMaxBidAskSpreadPct]
// for a relative version.
func WithMaxBidAskSpread(max float64) ChainOption {
	return chainOptionFunc(func(o *chainOptions) {
		o.maxBidAskSpread = &max
	})
}

// WithMaxBidAskSpreadPct limits the chain to contracts whose bid-ask spread is
// less than or equal to max as a percentage of the UNDERLYING price — not of
// the contract's own midpoint, which this godoc claimed until 2026-08-20.
// The distinction matters: on a $316 underlying, 0.15 admits a $0.47 spread,
// while a caller reading "percentage of the midpoint" would expect it to
// admit only spreads under 15% of a mid that is often well under a dollar.
//
// Verified live by sweeping the value against one chain: 0.09 admitted only
// contracts whose spread was at most 9% of the underlying, 0.11 at most 11%,
// 0.13 at most 13% — tracking the underlying exactly at every step, and
// never the midpoint. See [WithMaxBidAskSpread] for an absolute dollar
// version, which is unambiguous.
func WithMaxBidAskSpreadPct(max float64) ChainOption {
	return chainOptionFunc(func(o *chainOptions) {
		o.maxBidAskSpreadPct = &max
	})
}

// WithMinOpenInterest limits the chain to contracts with open interest greater
// than or equal to min, filtering out thinly held contracts.
func WithMinOpenInterest(min int) ChainOption {
	return chainOptionFunc(func(o *chainOptions) {
		o.minOpenInterest = &min
	})
}

// WithMinVolume limits the chain to contracts with trading volume greater than
// or equal to min, filtering out thinly traded contracts.
func WithMinVolume(min int) ChainOption {
	return chainOptionFunc(func(o *chainOptions) {
		o.minVolume = &min
	})
}

// WithNonstandard controls whether nonstandard contracts (for example,
// adjusted contracts created by splits or mergers) are included in the chain.
// Pass true to include them or false to exclude them; when the option is not
// used, the API default applies.
func WithNonstandard(nonstandard bool) ChainOption {
	return chainOptionFunc(func(o *chainOptions) {
		o.nonstandard = &nonstandard
	})
}

// WithAM controls whether AM-settled contracts are included. Pass true to
// limit the chain to AM-settled contracts or false to exclude them; when the
// option is not used, settlement is not filtered. Settlement style is
// meaningful only for index options (for example SPX, NDX); on single-stock and
// ETF options the API tolerates the parameter but it has no effect.
func WithAM(am bool) ChainOption {
	return chainOptionFunc(func(o *chainOptions) {
		o.am = &am
	})
}

// WithPM controls whether PM-settled contracts are included. Pass true to
// limit the chain to PM-settled contracts or false to exclude them; when the
// option is not used, settlement is not filtered. Settlement style is
// meaningful only for index options (for example SPX, NDX); on single-stock and
// ETF options the API tolerates the parameter but it has no effect.
func WithPM(pm bool) ChainOption {
	return chainOptionFunc(func(o *chainOptions) {
		o.pm = &pm
	})
}

// ---------------------------------------------------------------------------
// Expiration options.
// ---------------------------------------------------------------------------

// ExpirationOption is a functional option that refines a [Service.Expirations]
// request. The two filters are independent and may be combined.
type ExpirationOption interface {
	apply(*expirationOptions)
}

type expirationOptionFunc func(*expirationOptions)

func (f expirationOptionFunc) apply(o *expirationOptions) {
	f(o)
}

type expirationOptions struct {
	strike float64
	date   time.Time
}

// WithExpirationStrike limits the expiration list to dates that have a contract
// listed at the given strike price. The value must be greater than zero to take
// effect.
func WithExpirationStrike(strike float64) ExpirationOption {
	return expirationOptionFunc(func(o *expirationOptions) {
		o.strike = strike
	})
}

// WithExpirationDate requests the expiration dates that were available on the
// given historical date rather than today's list. Only the calendar date is
// used; a zero time leaves the parameter unset.
func WithExpirationDate(d time.Time) ExpirationOption {
	return expirationOptionFunc(func(o *expirationOptions) {
		o.date = d
	})
}

// ---------------------------------------------------------------------------
// Single-contract quote options.
// ---------------------------------------------------------------------------

// QuoteOption is a functional option that refines a [Service.Quote]
// request. The historical date range is set with [WithOptionQuoteWindow] using
// a single [OptionQuoteWindow] value, so the API's mutually-exclusive date and
// from/to parameters cannot be combined.
type QuoteOption interface {
	apply(*quoteOptions)
}

type quoteOptionFunc func(*quoteOptions)

func (f quoteOptionFunc) apply(o *quoteOptions) {
	f(o)
}

type quoteOptions struct {
	window OptionQuoteWindow
}

// WithOptionQuoteWindow requests a historical quote for a single contract from
// a single [OptionQuoteWindow] value: options.QuoteOnDate(t) for a single day
// or options.QuoteRange(from, to) for a range. Omitting it returns the current
// quote.
func WithOptionQuoteWindow(w OptionQuoteWindow) QuoteOption {
	return quoteOptionFunc(func(o *quoteOptions) {
		o.window = w
	})
}
