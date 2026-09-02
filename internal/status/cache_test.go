package status

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestCache_DefaultOnline(t *testing.T) {
	c := New(func(ctx context.Context) (bool, error) {
		return true, nil
	})
	if !c.IsOnline() {
		t.Error("new cache should default to online")
	}
}

func TestCache_Update(t *testing.T) {
	c := New(func(ctx context.Context) (bool, error) {
		return true, nil
	})

	c.Update(false)
	if c.IsOnline() {
		t.Error("cache should be offline after Update(false)")
	}

	c.Update(true)
	if !c.IsOnline() {
		t.Error("cache should be online after Update(true)")
	}
}

func TestCache_FreshCacheNoRefresh(t *testing.T) {
	fetched := make(chan struct{}, 8)
	c := New(func(ctx context.Context) (bool, error) {
		fetched <- struct{}{}
		return true, nil
	})

	// Set cache as fresh
	c.Update(true)

	// Should not trigger a refresh. A negative needs a bounded observation
	// window; 25ms is enough for a wrongly-spawned goroutine to signal.
	_ = c.IsOnline()
	select {
	case <-fetched:
		t.Error("fresh cache must not trigger a refresh")
	case <-time.After(25 * time.Millisecond):
	}
}

// waitFetched fails the test if the fetcher does not signal within the
// deadline. Channel-based instead of a fixed sleep: fast when it works,
// unflaky when the runner is slow.
func waitFetched(t *testing.T, fetched <-chan struct{}) {
	t.Helper()
	select {
	case <-fetched:
	case <-time.After(2 * time.Second):
		t.Fatal("background refresh never ran")
	}
}

func TestCache_StaleTriggersRefresh(t *testing.T) {
	fetched := make(chan struct{}, 8)
	c := New(func(ctx context.Context) (bool, error) {
		fetched <- struct{}{}
		return true, nil
	})

	// Manually set cache to be stale (> 300s old)
	c.mu.Lock()
	c.online = false
	c.fetchedAt = time.Now().Add(-301 * time.Second)
	c.mu.Unlock()

	// Should return true (unknown) and trigger refresh
	if !c.IsOnline() {
		t.Error("stale cache should return true (unknown/optimistic)")
	}
	waitFetched(t, fetched)
}

func TestCache_ApproachingStaleTriggersRefresh(t *testing.T) {
	fetched := make(chan struct{}, 8)
	c := New(func(ctx context.Context) (bool, error) {
		fetched <- struct{}{}
		return true, nil
	})

	// A healthy (online) cache 275s old (between 270s and 300s): the
	// offline case is covered separately (offline always re-probes
	// immediately, regardless of age — see TestCache_OfflineStateRetriesImmediately),
	// so this exercises the cadence-refresh path for a still-online cache.
	c.mu.Lock()
	c.online = true
	c.fetchedAt = time.Now().Add(-275 * time.Second)
	c.mu.Unlock()

	// Should return cached value (true) and trigger refresh
	if !c.IsOnline() {
		t.Error("cache between 270-300s should return cached value (true)")
	}
	waitFetched(t, fetched)
}

func TestCache_EmptyCacheTriggersRefresh(t *testing.T) {
	fetched := make(chan struct{}, 8)
	c := New(func(ctx context.Context) (bool, error) {
		fetched <- struct{}{}
		return false, nil
	})

	// Empty cache (never fetched) should return true (unknown)
	if !c.IsOnline() {
		t.Error("empty cache should return true (unknown/optimistic)")
	}
	waitFetched(t, fetched)
}

func TestCache_DuplicateRefreshPrevented(t *testing.T) {
	var fetchCount atomic.Int32
	started := make(chan struct{}, 8)
	release := make(chan struct{})
	c := New(func(ctx context.Context) (bool, error) {
		fetchCount.Add(1)
		started <- struct{}{}
		<-release // hold the refresh open while the duplicates are attempted
		return true, nil
	})

	// While the first refresh is in flight, further triggers must be no-ops.
	c.triggerRefresh()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first refresh never started")
	}
	c.triggerRefresh()
	c.triggerRefresh()
	if got := fetchCount.Load(); got != 1 {
		t.Errorf("fetches while one is in flight = %d, want 1", got)
	}
	close(release)
}

// waitOffline polls until the cache reads offline or the deadline passes.
// Bounded polling instead of a fixed sleep keeps the test fast and unflaky.
func waitOffline(t *testing.T, c *Cache) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c.mu.RLock()
		online, populated := c.online, !c.fetchedAt.IsZero()
		c.mu.RUnlock()
		if populated && !online {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("cache never turned offline after failing probes")
}

func TestCache_RefreshError_CountsAsOffline(t *testing.T) {
	// A failed probe is the only signal a real outage produces (a downed
	// host never answers with a non-200), so it must flip the gate to
	// offline instead of preserving the stale last-known state.
	c := New(func(ctx context.Context) (bool, error) {
		return false, fmt.Errorf("dial tcp: connection refused")
	})

	c.Update(true)
	c.mu.Lock()
	c.fetchedAt = time.Now().Add(-301 * time.Second)
	c.mu.Unlock()

	_ = c.IsOnline() // stale → triggers the failing refresh
	waitOffline(t, c)

	if c.IsOnline() {
		t.Error("IsOnline() = true right after a failed probe, want false")
	}
}

// TestCache_OfflineStateRetriesImmediately is a regression test: a failed
// probe used to stamp fetchedAt=now, and since the record then read as
// "fresh" (age < refreshInterval), IsOnline() would not trigger another
// probe for up to 270s — a single transient /status/ timeout (most likely
// exactly when the network is under stress) left the whole client without
// retries for 4.5 minutes. IsOnline() must re-probe on every call while the
// cached state is offline, not wait out the normal cadence.
func TestCache_OfflineStateRetriesImmediately(t *testing.T) {
	var calls atomic.Int32
	secondProbe := make(chan struct{}, 1)
	c := New(func(ctx context.Context) (bool, error) {
		switch calls.Add(1) {
		case 1:
			return false, fmt.Errorf("transient timeout")
		case 2:
			select {
			case secondProbe <- struct{}{}:
			default:
			}
			return true, nil
		default:
			return true, nil
		}
	})

	// First probe: triggered by staleness, fails, cache goes offline.
	c.mu.Lock()
	c.fetchedAt = time.Now().Add(-301 * time.Second)
	c.mu.Unlock()
	_ = c.IsOnline()
	waitOffline(t, c)

	// Immediately after — fetchedAt is now "fresh" by the normal cadence —
	// IsOnline() must still trigger a new probe rather than trusting the
	// offline reading for the next 270s.
	if c.IsOnline() {
		t.Fatal("IsOnline() = true right after the failed probe, want false")
	}
	select {
	case <-secondProbe:
	case <-time.After(2 * time.Second):
		t.Fatal("a second probe was never triggered — an offline reading was not re-probed immediately")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if c.IsOnline() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("cache never recovered to online after the successful re-probe")
}

func TestCache_OutageFromColdStart_TurnsOffline(t *testing.T) {
	// Cold cache + unreachable API: the first call reads optimistic true
	// (unknown) and triggers the probe; once the probe fails the gate must
	// read offline — previously it stayed true forever because errors
	// never populated the cache.
	c := New(func(ctx context.Context) (bool, error) {
		return false, fmt.Errorf("dial tcp: i/o timeout")
	})

	if !c.IsOnline() {
		t.Fatal("empty cache should read online (unknown) while the first probe runs")
	}
	waitOffline(t, c)

	// Recovery path: a successful status observation flips it back.
	c.Update(true)
	if !c.IsOnline() {
		t.Error("IsOnline() = false after Update(true), want true")
	}
}

func TestCache_FreshCacheReturnsCachedValue(t *testing.T) {
	c := New(func(ctx context.Context) (bool, error) {
		return true, nil
	})

	// Set to offline with fresh timestamp
	c.Update(false)

	// Should return false (the cached value) since cache is fresh
	if c.IsOnline() {
		t.Error("fresh cache should return cached value (false)")
	}
}
