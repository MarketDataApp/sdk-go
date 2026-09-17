//go:build integration

package integration

import (
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/timezone"
)

// TestMarketDay pins the bug that made TestOptions_Expirations and
// TestWireShape_StocksEarnings fail every night between UTC midnight and ET
// midnight, and pass again by morning.
//
// Without this the fix is only observable during those four hours, so a
// regression would land green and stay green until someone happened to push
// after midnight UTC.
func TestMarketDay(t *testing.T) {
	// The exact instant the suite was measured failing on 2026-09-17.
	failing := time.Date(2026, 9, 17, 0, 53, 0, 0, time.UTC)
	// A time on the same market day when it always passed.
	midday := time.Date(2026, 9, 16, 16, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name string
		now  time.Time
		want string
	}{
		{"inside the window, UTC is a day ahead of the market", failing, "2026-09-16"},
		{"outside the window, the calendars agree", midday, "2026-09-16"},
		{"ET midnight starts the next market day", time.Date(2026, 9, 17, 4, 0, 0, 0, time.UTC), "2026-09-17"},
		{"one second before ET midnight is still the old day", time.Date(2026, 9, 17, 3, 59, 59, 0, time.UTC), "2026-09-16"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := marketDay(tc.now).Format("2006-01-02"); got != tc.want {
				t.Errorf("marketDay(%v) = %s, want %s", tc.now, got, tc.want)
			}
		})
	}

	// The assertion the failing test actually makes: the current session's
	// expiration, as the API returns it, must not read as a past date.
	exp := time.Date(2026, 9, 16, 0, 0, 0, 0, timezone.Eastern)
	if truncDay(exp).Before(marketDay(failing)) {
		t.Error("the current session's expiration read as a past date inside the window")
	}
	// And the same comparison against the runner's UTC clock is what broke, so
	// prove the old behaviour really was wrong rather than merely different.
	if !truncDay(exp).Before(truncDay(failing)) {
		t.Error("expected the old UTC comparison to call the expiration past; the regression this pins is gone")
	}
}
