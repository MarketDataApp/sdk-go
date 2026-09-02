// Strike-window math, days-to-expiration, duration formatting, and error
// classification for the status line. Every function here is pure: no
// I/O, no client access, deterministic given its arguments (including the
// injected clock, where relevant), so Task 3.4's model/Update logic can
// depend on them without any test having to fake time.Now or the network.
package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
)

// windowStep is the fixed additive amount by which the strike window
// widens or narrows on each user adjustment (Update does
// clampWindow(w ± windowStep)).
const windowStep = 0.05

// minWindow and maxWindow bound the strike window enforced by
// clampWindow.
const (
	minWindow = 0.02
	maxWindow = 0.50
)

// strikeWindow computes the [lo, hi] strike bounds around underlyingPx for
// a fractional window (e.g. 0.10 means +/-10%): lo = px*(1-window), hi =
// px*(1+window). A non-positive underlyingPx means the underlying price
// isn't known yet, so strikeWindow returns (0, 0) — the sentinel fetchChain
// (see fetch.go) reads as "no filter": it omits WithStrikeRange whenever
// lo and hi are not both greater than zero.
func strikeWindow(underlyingPx, window float64) (lo, hi float64) {
	if underlyingPx <= 0 {
		return 0, 0
	}
	return underlyingPx * (1 - window), underlyingPx * (1 + window)
}

// clampWindow bounds w to [minWindow, maxWindow].
func clampWindow(w float64) float64 {
	if w < minWindow {
		return minWindow
	}
	if w > maxWindow {
		return maxWindow
	}
	return w
}

// dte returns the whole number of calendar days between now and exp. Both
// times are normalized to their own year/month/day before differencing,
// so the result reflects calendar days, not a truncated hours/24: a gap
// that crosses one midnight is 1 day even if it spans less than 24 hours,
// and two times on the same calendar day (in their respective locations)
// are 0 days apart even if many hours apart.
func dte(exp time.Time, now time.Time) int {
	y1, m1, d1 := now.Date()
	y2, m2, d2 := exp.Date()
	n := time.Date(y1, m1, d1, 0, 0, 0, 0, time.UTC)
	e := time.Date(y2, m2, d2, 0, 0, 0, 0, time.UTC)
	return int(e.Sub(n).Hours() / 24)
}

// formatDuration renders d in a compact "1h4m12s" style, rounded to the
// nearest second: the hour component is included only when d is at least
// an hour, the minute component is included whenever d is at least a
// minute (including alongside a nonzero hour component), and the second
// component is always shown.
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	sec := d / time.Second

	out := ""
	if h > 0 {
		out += fmt.Sprintf("%dh", h)
	}
	if h > 0 || m > 0 {
		out += fmt.Sprintf("%dm", m)
	}
	out += fmt.Sprintf("%ds", sec)
	return out
}

// classify renders err for the status line using the SDK error taxonomy.
// It never string-matches; each branch uses errors.As/Is. The rate-limit
// branch computes its wait duration from now (the app's injectable clock)
// rather than calling [marketdata.RateLimitError.WaitDuration] directly:
// that method measures against the real wall clock, which would make this
// function's output nondeterministic and hard to test. The formula
// mirrors WaitDuration exactly — ResetAt minus the reference time,
// floored at zero — so WaitDuration remains the SDK's own helper for
// callers that don't need an injectable clock.
func classify(err error, now time.Time) string {
	var rle *marketdata.RateLimitError
	if errors.As(err, &rle) {
		d := rle.ResetAt.Sub(now)
		if d < 0 {
			d = 0
		}
		return "rate limited — resets in " + formatDuration(d)
	}
	var ae *marketdata.AuthenticationError
	if errors.As(err, &ae) {
		return "auth failed — check MARKETDATA_TOKEN"
	}
	var ne *marketdata.NetworkError
	if errors.As(err, &ne) && ne.Timeout {
		return "network timeout — retrying on next tick"
	}
	return err.Error()
}
