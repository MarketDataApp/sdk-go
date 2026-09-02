package timezone

import (
	"errors"
	"testing"
	"time"
)

func TestEastern_NotNil(t *testing.T) {
	if Eastern == nil {
		t.Fatal("Eastern timezone should not be nil")
	}
}

func TestEastern_IsNewYork(t *testing.T) {
	// Verify we got America/New_York (or at least a reasonable timezone)
	name, _ := time.Now().In(Eastern).Zone()
	if name != "EST" && name != "EDT" {
		t.Errorf("Eastern timezone name = %q, want EST or EDT", name)
	}
}

func TestToEastern(t *testing.T) {
	// 2024-01-01 00:00:00 UTC = 2023-12-31 19:00:00 EST
	unix := int64(1704067200) // 2024-01-01 00:00:00 UTC
	result := ToEastern(unix)

	if result.Location() != Eastern {
		t.Error("ToEastern should return time in Eastern timezone")
	}

	// Verify conversion: UTC midnight should be 7pm EST previous day
	if result.Hour() != 19 {
		t.Errorf("Hour = %d, want 19 (7pm EST)", result.Hour())
	}
	if result.Day() != 31 {
		t.Errorf("Day = %d, want 31", result.Day())
	}
}

func TestToEastern_Summer(t *testing.T) {
	// 2024-07-01 00:00:00 UTC = 2024-06-30 20:00:00 EDT
	unix := int64(1719792000) // 2024-07-01 00:00:00 UTC
	result := ToEastern(unix)

	if result.Location() != Eastern {
		t.Error("ToEastern should return time in Eastern timezone")
	}

	// During EDT, offset is -4 hours from UTC
	if result.Hour() != 20 {
		t.Errorf("Hour = %d, want 20 (8pm EDT)", result.Hour())
	}
}

func TestLoadEastern_Success(t *testing.T) {
	loc := loadEastern(time.LoadLocation)
	if loc == nil || loc.String() != "America/New_York" {
		t.Errorf("loadEastern() = %v, want America/New_York", loc)
	}
}

func TestLoadEastern_FallbackOnMissingDatabase(t *testing.T) {
	loc := loadEastern(func(string) (*time.Location, error) {
		return nil, errors.New("tzdata unavailable")
	})
	if loc == nil {
		t.Fatal("loadEastern() = nil, want fixed-offset fallback")
	}
	// The fallback is a fixed UTC-5 zone with no DST.
	name, offset := time.Date(2026, 7, 1, 12, 0, 0, 0, loc).Zone()
	if name != "EST" || offset != -5*60*60 {
		t.Errorf("fallback zone = %s (%d), want EST (-18000)", name, offset)
	}
}

func TestToEastern_NullTimestampIsZeroTime(t *testing.T) {
	// A JSON null (or absent array cell) decodes to 0; it must map to the
	// zero time so IsZero works, never to the 1969-12-31 epoch rendering.
	if got := ToEastern(0); !got.IsZero() {
		t.Errorf("ToEastern(0) = %v, want the zero time", got)
	}
	if got := ToEastern(-1); !got.IsZero() {
		t.Errorf("ToEastern(-1) = %v, want the zero time", got)
	}
	if got := ToEastern(1704067200); got.IsZero() {
		t.Error("ToEastern(positive) must not be the zero time")
	}
}
