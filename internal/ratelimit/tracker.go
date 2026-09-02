// Package ratelimit provides rate limit tracking for the MarketData SDK.
package ratelimit

import (
	"net/http"
	"sync"
	"time"
)

// Tracker tracks API rate limit state.
// It is safe for concurrent use.
type Tracker struct {
	mu sync.RWMutex

	limit     int
	remaining int
	consumed  int
	resetAt   time.Time

	// reserved counts in-flight requests that have passed the pre-flight
	// check but not yet completed. It prevents N concurrent requests from
	// all passing a "remaining == 1" check and overshooting the limit.
	reserved int
}

// New creates a new rate limit tracker.
func New() *Tracker {
	return &Tracker{
		// Initialize with high values so we don't block before first response
		limit:     10000,
		remaining: 10000,
	}
}

// Update updates the rate limit state from response headers.
// Out-of-order updates are discarded: a response carrying an older reset
// window, or a higher remaining count within the same window, completed
// out of order and would move the snapshot backwards.
func (t *Tracker) Update(resp *http.Response) {
	h := ParseHeaders(resp.Header)
	limit, hasLimit := h.Limit, h.HasLimit
	remaining, hasRemaining := h.Remaining, h.HasRemaining
	consumed, hasConsumed := h.Consumed, h.HasConsumed
	resetAt, hasReset := h.ResetAt, h.HasReset

	t.mu.Lock()
	defer t.mu.Unlock()

	if hasReset && !t.resetAt.IsZero() {
		if resetAt.Before(t.resetAt) {
			return
		}
		if resetAt.Equal(t.resetAt) && hasRemaining && remaining > t.remaining {
			return
		}
	}

	if hasLimit {
		t.limit = limit
	}
	if hasRemaining {
		t.remaining = remaining
	}
	if hasConsumed {
		t.consumed = consumed
	}
	if hasReset {
		t.resetAt = resetAt
	}
}

// headerInt parses an integer header value, reporting whether it was present and valid.

// Reserve reserves one credit for an in-flight request. It returns false
// when credits are exhausted (accounting for other in-flight requests)
// and the reset time hasn't passed yet. Callers must call Release when
// the request completes.
//
// A limit of 0 marks unmetered access, not exhaustion: the API sends
// limit=0 headers on anonymous (demo) responses, so a zero limit must
// never block — the same reading the Java SDK applies.
func (t *Tracker) Reserve() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.limit == 0 {
		t.reserved++
		return true
	}

	if t.remaining-t.reserved <= 0 && time.Now().Before(t.resetAt) {
		return false
	}
	t.reserved++
	return true
}

// Release releases a credit reserved by Reserve.
func (t *Tracker) Release() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.reserved > 0 {
		t.reserved--
	}
}

// State returns the current rate limit state.
func (t *Tracker) State() State {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return State{
		Limit:     t.limit,
		Remaining: t.remaining,
		Consumed:  t.consumed,
		ResetAt:   t.resetAt,
	}
}

// State represents the current rate limit state.
type State struct {
	// Limit is the maximum requests allowed in the current window
	Limit int

	// Remaining is the number of requests remaining
	Remaining int

	// Consumed is the credit cost of the MOST RECENT request, not a running
	// total for the window: the API reports it per response and the tracker
	// stores the latest value. For the window total use Limit - Remaining.
	Consumed int

	// ResetAt is when the rate limit resets
	ResetAt time.Time
}
