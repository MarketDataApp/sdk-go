// Package status provides a cache for the MarketData API status endpoint,
// used to avoid retrying requests when the service is known to be offline.
package status

import (
	"context"
	"sync"
	"time"
)

const (
	// refreshInterval is the age at which a non-blocking background refresh is triggered.
	refreshInterval = 270 * time.Second

	// cacheValidity is the age at which the cached status is considered stale.
	cacheValidity = 300 * time.Second
)

// Fetcher is a function that fetches the current API status.
// It returns true if the API is online.
type Fetcher func(ctx context.Context) (online bool, err error)

// Cache tracks the API's online/offline status for use in retry decisions.
// It is safe for concurrent use.
type Cache struct {
	mu        sync.RWMutex
	online    bool
	fetchedAt time.Time
	fetcher   Fetcher

	// refreshing prevents duplicate background refreshes.
	refreshing sync.Mutex
}

// New creates a new status cache with the given fetcher function.
func New(fetcher Fetcher) *Cache {
	return &Cache{
		fetcher: fetcher,
		online:  true, // optimistic default
	}
}

// IsOnline returns whether the API is currently considered online.
// It applies the caching strategy with 270s/300s thresholds:
//   - Cached offline: always re-probe immediately (see below), return false
//     until a probe succeeds.
//   - Cache age < 270s: return cached status.
//   - Cache age 270s-300s: return cached status, trigger non-blocking refresh.
//   - Cache age >= 300s or empty: return unknown (true), trigger non-blocking refresh.
func (c *Cache) IsOnline() bool {
	c.mu.RLock()
	age := time.Since(c.fetchedAt)
	online := c.online
	empty := c.fetchedAt.IsZero()
	c.mu.RUnlock()

	if empty || age >= cacheValidity {
		// Stale or empty: treat as unknown (online), trigger refresh
		c.triggerRefresh()
		return true
	}

	if !online {
		// Cached offline: re-probe on every check instead of waiting out
		// the normal refreshInterval cadence. A single failed probe (e.g. a
		// transient /status/ timeout during network stress — precisely
		// when a false offline reading is most likely) would otherwise
		// stamp fetchedAt=now and leave the whole client without retries
		// for up to refreshInterval (270s), since the record reads as
		// "fresh" even though it reflects a probe failure, not a
		// confirmed outage. triggerRefresh's own lock still caps this to
		// one in-flight probe at a time.
		c.triggerRefresh()
		return false
	}

	if age >= refreshInterval {
		// Approaching stale: use cached value, trigger refresh
		c.triggerRefresh()
	}

	return online
}

// triggerRefresh starts a non-blocking background refresh of the status cache.
func (c *Cache) triggerRefresh() {
	// Try to acquire the refreshing lock; if already refreshing, skip.
	if !c.refreshing.TryLock() {
		return
	}

	go func() {
		defer c.refreshing.Unlock()

		c.mu.RLock()
		fetcher := c.fetcher
		c.mu.RUnlock()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		online, err := fetcher(ctx)
		if err != nil {
			// A failed probe counts as offline: if /status/ itself is
			// unreachable the API almost certainly is too, and this is the
			// only signal a real outage produces (a downed host never
			// returns a non-200). The cost of a false negative is small —
			// the gate only suppresses retries, never first attempts, and
			// the state is re-probed on the normal refresh cadence.
			online = false
		}

		c.mu.Lock()
		c.online = online
		c.fetchedAt = time.Now()
		c.mu.Unlock()
	}()
}

// Update directly sets the cache status. Used when a status response
// is received as part of normal operations (e.g., during startup validation).
func (c *Cache) Update(online bool) {
	c.mu.Lock()
	c.online = online
	c.fetchedAt = time.Now()
	c.mu.Unlock()
}
