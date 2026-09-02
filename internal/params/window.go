// Package params holds canonical, internal representations of Market Data API
// request parameters that are shared across resource packages. Its central
// type, [Window], models the API's mutually-exclusive date-selection
// parameters (date / from / to / countback). Public resource packages wrap
// Window in their own sealed union types so that a caller can only ever
// express one valid date mode, making illegal combinations impossible to
// write. Window's fields are unexported and it can only be built through the
// constructors here, so the exclusivity rules cannot be bypassed.
package params

import (
	"net/url"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

// dateLayout is the calendar-date form the API accepts for date parameters.
const dateLayout = "2006-01-02"

// WindowKind identifies which mutually-exclusive date mode a [Window] carries.
type WindowKind uint8

const (
	// KindNone is the zero Window: no date parameters are sent.
	KindNone WindowKind = iota
	// KindDate is a single calendar date (date=).
	KindDate
	// KindFromTo is an explicit closed range (from= & to=).
	KindFromTo
	// KindFrom is an open-ended range starting at from (from=).
	KindFrom
	// KindTo is a range ending at to (to=).
	KindTo
	// KindCountback is a fixed number of the most recent periods (countback=).
	KindCountback
	// KindCountbackTo is a fixed number of periods ending at to (countback= & to=).
	KindCountbackTo
)

// Window is the canonical, validated representation of the API's date-window
// parameters. Construct it only with the mode constructors ([OnDate],
// [Between], [Since], [Until], [LastN], [LastNUntil]); the zero value is a
// valid "no date parameters" window.
type Window struct {
	kind      WindowKind
	date      time.Time
	from      time.Time
	to        time.Time
	countback int
}

// OnDate selects a single calendar date (date=). Time-of-day is ignored.
func OnDate(d time.Time) Window { return Window{kind: KindDate, date: d} }

// Between selects an explicit closed date range (from= & to=).
func Between(from, to time.Time) Window {
	return Window{kind: KindFromTo, from: from, to: to}
}

// Since selects an open-ended range starting at from (from=), letting the API
// default the end to the most recent period.
func Since(from time.Time) Window { return Window{kind: KindFrom, from: from} }

// Until selects a range ending at to (to=), letting the API default the count.
func Until(to time.Time) Window { return Window{kind: KindTo, to: to} }

// LastN selects the n most recent periods (countback=).
func LastN(n int) Window { return Window{kind: KindCountback, countback: n} }

// LastNUntil selects the n periods ending at to (countback= & to=).
func LastNUntil(n int, to time.Time) Window {
	return Window{kind: KindCountbackTo, countback: n, to: to}
}

// Kind reports which date mode the window carries.
func (w Window) Kind() WindowKind { return w.kind }

// IsZero reports whether the window sends no date parameters.
func (w Window) IsZero() bool { return w.kind == KindNone }

// Validate checks the window's values before any request is built. It enforces
// the value-range rules that cannot be encoded in the type system: countback
// must be positive, dates present for their mode must be non-zero, and an
// explicit from must not be after to. It returns a [sdkerrors.ValidationError]
// on failure so callers can reject the request pre-network.
func (w Window) Validate() error {
	switch w.kind {
	case KindNone:
		return nil
	case KindDate:
		if w.date.IsZero() {
			return windowErr("date", "date must not be zero")
		}
	case KindFromTo:
		if w.from.IsZero() || w.to.IsZero() {
			return windowErr("window", "both from and to must be set")
		}
		if truncateDay(w.from).After(truncateDay(w.to)) {
			return windowErr("window", "from must not be after to")
		}
	case KindFrom:
		if w.from.IsZero() {
			return windowErr("from", "from must not be zero")
		}
	case KindTo:
		if w.to.IsZero() {
			return windowErr("to", "to must not be zero")
		}
	case KindCountback:
		if w.countback <= 0 {
			return windowErr("countback", "countback must be greater than zero")
		}
	case KindCountbackTo:
		if w.countback <= 0 {
			return windowErr("countback", "countback must be greater than zero")
		}
		if w.to.IsZero() {
			return windowErr("to", "to must not be zero")
		}
	}
	return nil
}

// Apply writes the window's parameters into v using the API's calendar-date
// format. It assumes the window has already passed [Window.Validate].
func (w Window) Apply(v url.Values) {
	switch w.kind {
	case KindDate:
		v.Set("date", w.date.Format(dateLayout))
	case KindFromTo:
		v.Set("from", w.from.Format(dateLayout))
		v.Set("to", w.to.Format(dateLayout))
	case KindFrom:
		v.Set("from", w.from.Format(dateLayout))
	case KindTo:
		v.Set("to", w.to.Format(dateLayout))
	case KindCountback:
		v.Set("countback", intToStr(w.countback))
	case KindCountbackTo:
		v.Set("countback", intToStr(w.countback))
		v.Set("to", w.to.Format(dateLayout))
	}
}

// From, To, Date, and Countback expose the underlying values for callers that
// need them (for example, the candle range-splitting logic in stocks).
func (w Window) From() time.Time { return w.from }
func (w Window) To() time.Time   { return w.to }
func (w Window) Date() time.Time { return w.date }
func (w Window) Countback() int  { return w.countback }

// IsRange reports whether the window is an explicit closed from/to range, the
// only mode eligible for the stocks candle range-splitting optimization.
func (w Window) IsRange() bool { return w.kind == KindFromTo }

// Chunk derives a from/to sub-window bounded by the given dates. It lets a
// caller split a large range into smaller concurrent requests without needing
// to name this package.
func (w Window) Chunk(from, to time.Time) Window {
	return Window{kind: KindFromTo, from: from, to: to}
}

// AnchorCountback returns the window unchanged unless it is a bare countback
// (KindCountback), in which case it returns the equivalent
// countback-ending-at-to window. It lets an endpoint pin the "n most recent
// periods" semantics with an explicit to= anchor on endpoints where the API
// silently ignores a countback that arrives without one.
func (w Window) AnchorCountback(to time.Time) Window {
	if w.kind != KindCountback {
		return w
	}
	return LastNUntil(w.countback, to)
}

func windowErr(field, msg string) error {
	return &sdkerrors.ValidationError{Field: field, Message: msg}
}

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func intToStr(n int) string {
	// Small, allocation-light integer formatting without importing strconv at
	// every call site; n is always a small count.
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
