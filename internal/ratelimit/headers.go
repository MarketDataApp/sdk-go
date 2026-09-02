package ratelimit

import (
	"net/http"
	"strconv"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/timezone"
)

// Headers is the X-Api-Ratelimit-* set parsed from one response.
//
// It exists because the same four headers were parsed in four places, by
// three different rules, and they disagreed. On a single response carrying
// "X-Api-Ratelimit-Reset: 0" the SDK reported the reset time as the zero
// time through Response.RateLimit, and as 1969-12-31 through both
// Client.RateLimits() and UserInfo — the 1969 rendering being a bug this
// release had already fixed, in one of the four parsers. The copies also
// diverged on malformed input: a Sscanf-based reader accepted "100abc" as
// 100 where a strconv-based one read 0.
//
// One parser, one set of rules:
//
//   - strconv, so a value with trailing garbage is rejected rather than
//     silently truncated.
//   - A missing or unparseable header leaves its Has* flag false, so a
//     caller can tell "absent" from "present and zero" — which matters for
//     the reset time, where zero is not a plausible instant.
//   - ResetAt is normalized to US/Eastern like every other time-bearing
//     field in the SDK (ADR-005).
type Headers struct {
	Limit     int
	Remaining int
	Consumed  int
	ResetAt   time.Time

	HasLimit     bool
	HasRemaining bool
	HasConsumed  bool
	HasReset     bool
}

// ParseHeaders reads the rate-limit headers from h. A nil header set yields
// the zero value, with every Has* flag false.
func ParseHeaders(h http.Header) Headers {
	var out Headers
	if h == nil {
		return out
	}

	out.Limit, out.HasLimit = headerInt(h, "X-Api-Ratelimit-Limit")
	out.Remaining, out.HasRemaining = headerInt(h, "X-Api-Ratelimit-Remaining")
	out.Consumed, out.HasConsumed = headerInt(h, "X-Api-Ratelimit-Consumed")

	if v := h.Get("X-Api-Ratelimit-Reset"); v != "" {
		// A reset epoch of 0 or below is not a plausible instant; the API
		// sends it on anonymous responses alongside limit=0. Treating it as
		// absent keeps 1969-12-31 out of every public surface.
		if epoch, err := strconv.ParseInt(v, 10, 64); err == nil && epoch > 0 {
			out.ResetAt = timezone.ToEastern(epoch)
			out.HasReset = true
		}
	}

	return out
}

func headerInt(h http.Header, key string) (int, bool) {
	v := h.Get(key)
	if v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}
