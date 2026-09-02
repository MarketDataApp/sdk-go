package ratelimit

import (
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tracker := New()

	state := tracker.State()
	if state.Limit != 10000 {
		t.Errorf("Limit = %d, want 10000", state.Limit)
	}
	if state.Remaining != 10000 {
		t.Errorf("Remaining = %d, want 10000", state.Remaining)
	}
	if state.Consumed != 0 {
		t.Errorf("Consumed = %d, want 0", state.Consumed)
	}
}

func TestTracker_Update(t *testing.T) {
	tracker := New()

	resp := &http.Response{
		Header: http.Header{
			"X-Api-Ratelimit-Limit":     []string{"5000"},
			"X-Api-Ratelimit-Remaining": []string{"4999"},
			"X-Api-Ratelimit-Consumed":  []string{"1"},
			"X-Api-Ratelimit-Reset":     []string{"1735689600"},
		},
	}

	tracker.Update(resp)

	state := tracker.State()
	if state.Limit != 5000 {
		t.Errorf("Limit = %d, want 5000", state.Limit)
	}
	if state.Remaining != 4999 {
		t.Errorf("Remaining = %d, want 4999", state.Remaining)
	}
	if state.Consumed != 1 {
		t.Errorf("Consumed = %d, want 1", state.Consumed)
	}
	expectedReset := time.Unix(1735689600, 0)
	if !state.ResetAt.Equal(expectedReset) {
		t.Errorf("ResetAt = %v, want %v", state.ResetAt, expectedReset)
	}
}

func TestTracker_Update_PartialHeaders(t *testing.T) {
	tracker := New()

	// Only update limit
	resp := &http.Response{
		Header: http.Header{
			"X-Api-Ratelimit-Limit": []string{"3000"},
		},
	}

	tracker.Update(resp)

	state := tracker.State()
	if state.Limit != 3000 {
		t.Errorf("Limit = %d, want 3000", state.Limit)
	}
	// Remaining should stay at initial value
	if state.Remaining != 10000 {
		t.Errorf("Remaining = %d, want 10000", state.Remaining)
	}
}

func TestTracker_Update_InvalidHeaders(t *testing.T) {
	tracker := New()

	resp := &http.Response{
		Header: http.Header{
			"X-Api-Ratelimit-Limit":     []string{"not-a-number"},
			"X-Api-Ratelimit-Remaining": []string{"invalid"},
			"X-Api-Ratelimit-Consumed":  []string{"bad"},
			"X-Api-Ratelimit-Reset":     []string{"broken"},
		},
	}

	// Should not panic and should keep original values
	tracker.Update(resp)

	state := tracker.State()
	if state.Limit != 10000 {
		t.Errorf("Limit = %d, want 10000 (unchanged)", state.Limit)
	}
}

func TestTracker_ConcurrentAccess(t *testing.T) {
	tracker := New()
	done := make(chan bool)

	// Run multiple concurrent updates
	for i := 0; i < 10; i++ {
		go func(n int) {
			resp := &http.Response{
				Header: http.Header{
					"X-Api-Ratelimit-Remaining": []string{"100"},
				},
			}
			tracker.Update(resp)
			_ = tracker.State()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func formatUnix(t time.Time) string {
	return strconv.FormatInt(t.Unix(), 10)
}

func TestTracker_Reserve_Release(t *testing.T) {
	tracker := New()
	futureReset := time.Now().Add(1 * time.Hour)

	// Set remaining to 2 with a future reset
	tracker.Update(&http.Response{
		Header: http.Header{
			"X-Api-Ratelimit-Remaining": []string{"2"},
			"X-Api-Ratelimit-Reset":     []string{formatUnix(futureReset)},
		},
	})

	// Two reservations succeed, third fails (TOCTOU guard)
	if !tracker.Reserve() {
		t.Fatal("first Reserve() = false, want true")
	}
	if !tracker.Reserve() {
		t.Fatal("second Reserve() = false, want true")
	}
	if tracker.Reserve() {
		t.Fatal("third Reserve() = true, want false (2 in flight, remaining 2)")
	}

	// Releasing frees a slot
	tracker.Release()
	if !tracker.Reserve() {
		t.Error("Reserve() after Release() = false, want true")
	}
}

func TestTracker_Reserve_PastReset(t *testing.T) {
	tracker := New()
	pastReset := time.Now().Add(-1 * time.Hour)

	tracker.Update(&http.Response{
		Header: http.Header{
			"X-Api-Ratelimit-Remaining": []string{"0"},
			"X-Api-Ratelimit-Reset":     []string{formatUnix(pastReset)},
		},
	})

	// Exhausted but window ended: allow through so the snapshot can refresh
	if !tracker.Reserve() {
		t.Error("Reserve() = false, want true (reset time passed)")
	}
}

func TestTracker_Release_NeverNegative(t *testing.T) {
	tracker := New()
	tracker.Release() // no reservation outstanding — must not underflow

	futureReset := time.Now().Add(1 * time.Hour)
	tracker.Update(&http.Response{
		Header: http.Header{
			"X-Api-Ratelimit-Remaining": []string{"1"},
			"X-Api-Ratelimit-Reset":     []string{formatUnix(futureReset)},
		},
	})

	if !tracker.Reserve() {
		t.Fatal("Reserve() = false, want true")
	}
	if tracker.Reserve() {
		t.Error("Reserve() = true, want false (a spurious Release must not add capacity)")
	}
}

func TestTracker_Reserve_Concurrent(t *testing.T) {
	tracker := New()
	futureReset := time.Now().Add(1 * time.Hour)

	tracker.Update(&http.Response{
		Header: http.Header{
			"X-Api-Ratelimit-Remaining": []string{"5"},
			"X-Api-Ratelimit-Reset":     []string{formatUnix(futureReset)},
		},
	})

	var wg sync.WaitGroup
	granted := make(chan struct{}, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tracker.Reserve() {
				granted <- struct{}{}
			}
		}()
	}
	wg.Wait()
	close(granted)

	count := 0
	for range granted {
		count++
	}
	if count != 5 {
		t.Errorf("granted %d reservations, want exactly 5", count)
	}
}

func TestTracker_Update_OutOfOrder(t *testing.T) {
	tracker := New()
	newer := time.Now().Add(1 * time.Hour)
	older := time.Now().Add(-1 * time.Hour)

	tracker.Update(&http.Response{
		Header: http.Header{
			"X-Api-Ratelimit-Remaining": []string{"100"},
			"X-Api-Ratelimit-Reset":     []string{formatUnix(newer)},
		},
	})

	// An update from an older reset window is discarded entirely
	tracker.Update(&http.Response{
		Header: http.Header{
			"X-Api-Ratelimit-Remaining": []string{"5000"},
			"X-Api-Ratelimit-Consumed":  []string{"99"},
			"X-Api-Ratelimit-Reset":     []string{formatUnix(older)},
		},
	})
	if state := tracker.State(); state.Remaining != 100 {
		t.Errorf("Remaining = %d, want 100 (older window discarded)", state.Remaining)
	}

	// A higher remaining within the same window is discarded
	tracker.Update(&http.Response{
		Header: http.Header{
			"X-Api-Ratelimit-Remaining": []string{"500"},
			"X-Api-Ratelimit-Reset":     []string{formatUnix(newer)},
		},
	})
	if state := tracker.State(); state.Remaining != 100 {
		t.Errorf("Remaining = %d, want 100 (stale same-window update discarded)", state.Remaining)
	}

	// A lower remaining within the same window applies
	tracker.Update(&http.Response{
		Header: http.Header{
			"X-Api-Ratelimit-Remaining": []string{"99"},
			"X-Api-Ratelimit-Reset":     []string{formatUnix(newer)},
		},
	})
	if state := tracker.State(); state.Remaining != 99 {
		t.Errorf("Remaining = %d, want 99 (in-order update applied)", state.Remaining)
	}

	// A newer window always applies, even with higher remaining
	newest := newer.Add(1 * time.Hour)
	tracker.Update(&http.Response{
		Header: http.Header{
			"X-Api-Ratelimit-Remaining": []string{"10000"},
			"X-Api-Ratelimit-Reset":     []string{formatUnix(newest)},
		},
	})
	if state := tracker.State(); state.Remaining != 10000 {
		t.Errorf("Remaining = %d, want 10000 (new window applied)", state.Remaining)
	}
}

// TestTracker_ZeroLimitUnmetered covers the demo-mode wedge (QA finding C-1):
// anonymous API responses carry limit=0/remaining=0 with a future reset. A
// zero limit marks unmetered access, so it must never block Reserve —
// otherwise the first demo response wedges the tracker
// into rejecting every subsequent call until the reset.
func TestTracker_ZeroLimitUnmetered(t *testing.T) {
	tracker := New()
	tracker.Update(&http.Response{
		Header: http.Header{
			"X-Api-Ratelimit-Limit":     []string{"0"},
			"X-Api-Ratelimit-Remaining": []string{"0"},
			"X-Api-Ratelimit-Consumed":  []string{"0"},
			"X-Api-Ratelimit-Reset":     []string{formatUnix(time.Now().Add(24 * time.Hour))},
		},
	})

	for i := 0; i < 5; i++ {
		if !tracker.Reserve() {
			t.Fatalf("Reserve() call %d = false, want true (limit=0 is unmetered, not exhausted)", i+1)
		}
	}
	if !tracker.Reserve() {
		t.Error("Reserve() = false, want true (limit=0 is unmetered)")
	}
	tracker.Release()
}
