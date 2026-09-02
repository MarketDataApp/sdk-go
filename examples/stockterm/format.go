// Pure text-formatting helpers used across the watchlist and detail pane:
// comma and formatDuration render numbers and durations, and classify
// turns an SDK error into a one-line status message. This file is
// deliberately style-free — colors (signColor and friends) live in
// styles.go, added in task 2.5 — so every function here returns plain
// text.
package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
)

// comma formats n with comma thousands separators, e.g. comma(41203110)
// == "41,203,110". Zero and negative values are handled; a negative
// value keeps its sign before the first digit group, e.g. comma(-1234)
// == "-1,234".
func comma(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	digits := strconv.FormatInt(n, 10)

	var groups []string
	for len(digits) > 3 {
		split := len(digits) - 3
		groups = append([]string{digits[split:]}, groups...)
		digits = digits[:split]
	}
	groups = append([]string{digits}, groups...)

	out := strings.Join(groups, ",")
	if neg {
		out = "-" + out
	}
	return out
}

// formatDuration renders d in a compact "1h4m12s" style, rounded to the
// nearest second. Units above the largest non-zero unit are omitted
// (under a minute renders as "42s", not "0h0m42s"), but units below it
// are always shown even when zero (an exact minute renders as "1m0s",
// not "1m"). Negative durations are rendered using their absolute value.
func formatDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	total := int64(d.Round(time.Second) / time.Second)

	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60

	switch {
	case h > 0:
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm%ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

// classify renders err for the status line using the SDK error taxonomy.
// It never string-matches; each branch uses errors.As so wrapped errors
// classify the same as the typed error itself. The rate-limit branch
// computes its wait duration from now (the app's injectable clock) rather
// than calling [marketdata.RateLimitError.WaitDuration] directly: that
// method measures against the real wall clock, which would make this
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
