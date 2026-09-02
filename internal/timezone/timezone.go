// Package timezone provides timezone conversion helpers for the SDK.
package timezone

import (
	"time"

	// Embed the IANA timezone database so US/Eastern resolves correctly
	// (including DST) even on hosts without a system tzdata — scratch or
	// alpine containers, the typical deployment for a Go binary. Costs
	// ~450KB of binary size; the FixedZone fallback below becomes a
	// near-unreachable belt-and-suspenders.
	_ "time/tzdata"
)

// Eastern is the US/Eastern timezone location used for normalizing API timestamps.
var Eastern = loadEastern(time.LoadLocation)

// loadEastern resolves the US/Eastern location, falling back to a fixed
// UTC-5 offset if the timezone database is unavailable. The loader is
// injected so the fallback branch is testable — the real database is always
// present under `go test`.
func loadEastern(load func(name string) (*time.Location, error)) *time.Location {
	loc, err := load("America/New_York")
	if err != nil {
		return time.FixedZone("EST", -5*60*60)
	}
	return loc
}

// ToEastern converts a Unix timestamp to time.Time in US/Eastern timezone.
// A zero or negative timestamp — how a JSON null or an absent array cell
// decodes — returns the zero time.Time, so callers can test for absence
// with IsZero instead of comparing against the 1969-12-31 epoch rendering.
// No legitimate market timestamp is at or before the epoch.
func ToEastern(unix int64) time.Time {
	if unix <= 0 {
		return time.Time{}
	}
	return time.Unix(unix, 0).In(Eastern)
}
