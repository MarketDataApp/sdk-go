package ratelimit

import (
	"net/http"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/timezone"
)

func TestParseHeaders(t *testing.T) {
	t.Run("nil header set", func(t *testing.T) {
		got := ParseHeaders(nil)
		if got.HasLimit || got.HasRemaining || got.HasConsumed || got.HasReset {
			t.Errorf("ParseHeaders(nil) = %+v, want every Has flag false", got)
		}
	})

	t.Run("full set", func(t *testing.T) {
		const epoch = int64(1787166240)
		got := ParseHeaders(http.Header{
			"X-Api-Ratelimit-Limit":     []string{"10000"},
			"X-Api-Ratelimit-Remaining": []string{"9998"},
			"X-Api-Ratelimit-Consumed":  []string{"2"},
			"X-Api-Ratelimit-Reset":     []string{"1787166240"},
		})
		if got.Limit != 10000 || got.Remaining != 9998 || got.Consumed != 2 {
			t.Errorf("counts = %d/%d/%d, want 10000/9998/2", got.Limit, got.Remaining, got.Consumed)
		}
		if !got.HasLimit || !got.HasRemaining || !got.HasConsumed || !got.HasReset {
			t.Errorf("Has flags = %+v, want all true", got)
		}
		if !got.ResetAt.Equal(time.Unix(epoch, 0)) {
			t.Errorf("ResetAt = %v, want the same instant as the epoch", got.ResetAt)
		}
		// ADR-005: every time-bearing field is US/Eastern. This one used to
		// be built with a bare time.Unix, so it carried the host's zone.
		if got.ResetAt.Location() != timezone.Eastern {
			t.Errorf("ResetAt location = %v, want %v", got.ResetAt.Location(), timezone.Eastern)
		}
	})

	t.Run("absent is distinguishable from zero", func(t *testing.T) {
		got := ParseHeaders(http.Header{"X-Api-Ratelimit-Remaining": []string{"0"}})
		if !got.HasRemaining || got.Remaining != 0 {
			t.Errorf("remaining = %d (has=%v), want 0 present", got.Remaining, got.HasRemaining)
		}
		if got.HasLimit {
			t.Error("HasLimit = true for an absent header")
		}
	})

	t.Run("reset of zero is treated as absent", func(t *testing.T) {
		// The API sends reset=0 on anonymous responses. Reading it as an
		// instant is what rendered 1969-12-31 on two public surfaces.
		for _, v := range []string{"0", "-1"} {
			got := ParseHeaders(http.Header{"X-Api-Ratelimit-Reset": []string{v}})
			if got.HasReset || !got.ResetAt.IsZero() {
				t.Errorf("reset %q -> %v (has=%v), want absent", v, got.ResetAt, got.HasReset)
			}
		}
	})

	t.Run("malformed values are rejected, not truncated", func(t *testing.T) {
		// A Sscanf-based parser read "100abc" as 100; strconv rejects it.
		got := ParseHeaders(http.Header{
			"X-Api-Ratelimit-Limit": []string{"100abc"},
			"X-Api-Ratelimit-Reset": []string{"not-an-epoch"},
		})
		if got.HasLimit || got.Limit != 0 {
			t.Errorf("limit = %d (has=%v), want rejected", got.Limit, got.HasLimit)
		}
		if got.HasReset {
			t.Error("HasReset = true for an unparseable epoch")
		}
	})
}
