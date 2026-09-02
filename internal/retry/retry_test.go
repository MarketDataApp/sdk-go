package retry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if cfg.InitialBackoff != 1*time.Second {
		t.Errorf("InitialBackoff = %v, want 1s", cfg.InitialBackoff)
	}
	if cfg.Multiplier != 2.0 {
		t.Errorf("Multiplier = %v, want 2.0", cfg.Multiplier)
	}
}

func TestShouldRetryStatus(t *testing.T) {
	tests := []struct {
		status int
		want   bool
	}{
		{http.StatusOK, false},                  // 200
		{http.StatusCreated, false},             // 201
		{http.StatusBadRequest, false},          // 400
		{http.StatusUnauthorized, false},        // 401
		{http.StatusForbidden, false},           // 403
		{http.StatusNotFound, false},            // 404
		{http.StatusTooManyRequests, false},     // 429 - NOT retried
		{http.StatusInternalServerError, false}, // 500 - NOT retried
		{http.StatusNotImplemented, true},       // 501
		{http.StatusBadGateway, true},           // 502
		{http.StatusServiceUnavailable, true},   // 503
		{http.StatusGatewayTimeout, true},       // 504
		{599, true},                             // 599
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			got := ShouldRetryStatus(tt.status)
			if got != tt.want {
				t.Errorf("ShouldRetryStatus(%d) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestShouldRetryError(t *testing.T) {
	// nil error should not retry
	if ShouldRetryError(nil) {
		t.Error("ShouldRetryError(nil) = true, want false")
	}

	// DNS error should retry
	dnsErr := &net.DNSError{Err: "lookup failed", Name: "example.com"}
	if !ShouldRetryError(dnsErr) {
		t.Error("ShouldRetryError(DNSError) = false, want true")
	}
}

func TestCalculateBackoff(t *testing.T) {
	cfg := Config{
		InitialBackoff: 1 * time.Second,
		Multiplier:     2.0,
	}

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 1 * time.Second},  // Initial
		{1, 2 * time.Second},  // 1 * 2
		{2, 4 * time.Second},  // 1 * 4
		{3, 8 * time.Second},  // 1 * 8
		{4, 16 * time.Second}, // 1 * 16
		{5, 32 * time.Second}, // 1 * 32, uncapped
		{6, 64 * time.Second}, // 1 * 64, uncapped
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := CalculateBackoff(cfg, tt.attempt)
			if got != tt.want {
				t.Errorf("CalculateBackoff(attempt=%d) = %v, want %v", tt.attempt, got, tt.want)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{"empty", "", 0},
		{"seconds", "30", 30 * time.Second},
		{"invalid", "not-a-number", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Header: http.Header{},
			}
			if tt.header != "" {
				resp.Header.Set("Retry-After", tt.header)
			}

			got := ParseRetryAfter(resp)
			if got != tt.want {
				t.Errorf("ParseRetryAfter() = %v, want %v", got, tt.want)
			}
		})
	}

	// Test nil response
	if got := ParseRetryAfter(nil); got != 0 {
		t.Errorf("ParseRetryAfter(nil) = %v, want 0", got)
	}
}

func TestWait(t *testing.T) {
	ctx := context.Background()

	// Test zero duration
	if err := Wait(ctx, 0); err != nil {
		t.Errorf("Wait(0) error = %v", err)
	}

	// Test negative duration
	if err := Wait(ctx, -1*time.Second); err != nil {
		t.Errorf("Wait(-1s) error = %v", err)
	}

	// Test small duration
	start := time.Now()
	if err := Wait(ctx, 10*time.Millisecond); err != nil {
		t.Errorf("Wait(10ms) error = %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 10*time.Millisecond {
		t.Errorf("Wait(10ms) elapsed = %v, want >= 10ms", elapsed)
	}
}

func TestWait_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := Wait(ctx, 10*time.Second)
	if err != context.Canceled {
		t.Errorf("Wait() error = %v, want context.Canceled", err)
	}
}

func TestShouldRetryError_Timeout(t *testing.T) {
	timeoutErr := &testTimeoutError{timeout: true}
	if !ShouldRetryError(timeoutErr) {
		t.Error("ShouldRetryError should return true for timeout error")
	}
}

func TestShouldRetryError_Temporary(t *testing.T) {
	tempErr := &testTemporaryError{temporary: true}
	if !ShouldRetryError(tempErr) {
		t.Error("ShouldRetryError should return true for temporary error")
	}
}

func TestShouldRetryError_NotRetryable(t *testing.T) {
	err := &testTemporaryError{temporary: false}
	if ShouldRetryError(err) {
		t.Error("ShouldRetryError should return false for non-temporary error")
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	futureTime := time.Now().Add(30 * time.Second)
	resp := &http.Response{
		Header: http.Header{
			"Retry-After": []string{futureTime.UTC().Format(http.TimeFormat)},
		},
	}

	duration := ParseRetryAfter(resp)
	if duration < 25*time.Second || duration > 35*time.Second {
		t.Errorf("ParseRetryAfter() = %v, want ~30s", duration)
	}
}

func TestParseRetryAfter_HTTPDatePast(t *testing.T) {
	pastTime := time.Now().Add(-30 * time.Second)
	resp := &http.Response{
		Header: http.Header{
			"Retry-After": []string{pastTime.UTC().Format(http.TimeFormat)},
		},
	}

	duration := ParseRetryAfter(resp)
	if duration != 0 {
		t.Errorf("ParseRetryAfter() = %v, want 0 for past time", duration)
	}
}

// Test helpers
type testTimeoutError struct {
	timeout bool
}

func (e *testTimeoutError) Error() string   { return "timeout" }
func (e *testTimeoutError) Timeout() bool   { return e.timeout }
func (e *testTimeoutError) Temporary() bool { return false }

type testTemporaryError struct {
	temporary bool
}

func (e *testTemporaryError) Error() string   { return "temporary" }
func (e *testTemporaryError) Timeout() bool   { return false }
func (e *testTemporaryError) Temporary() bool { return e.temporary }

type plainError struct {
	msg string
}

func (e *plainError) Error() string { return e.msg }

func TestShouldRetryError_PlainError(t *testing.T) {
	err := &plainError{msg: "some error"}
	if ShouldRetryError(err) {
		t.Error("ShouldRetryError should return false for plain error")
	}
}

func TestCalculateBackoff_OverflowClamped(t *testing.T) {
	cfg := Config{
		InitialBackoff: 1 * time.Second,
		Multiplier:     2.0,
	}

	// Extreme attempts must clamp instead of overflowing time.Duration
	for _, attempt := range []int{62, 63, 100, 1000} {
		got := CalculateBackoff(cfg, attempt)
		if got <= 0 {
			t.Errorf("CalculateBackoff(attempt=%d) = %v, overflowed to non-positive", attempt, got)
		}
		if got != time.Duration(math.MaxInt64) {
			t.Errorf("CalculateBackoff(attempt=%d) = %v, want clamped max", attempt, got)
		}
	}
}

func TestParseRetryAfter_Overflow(t *testing.T) {
	resp := &http.Response{Header: http.Header{"Retry-After": []string{"9999999999"}}}
	got := ParseRetryAfter(resp)
	if got <= 0 {
		t.Errorf("ParseRetryAfter(huge) = %v, must not overflow to non-positive", got)
	}

	neg := &http.Response{Header: http.Header{"Retry-After": []string{"-5"}}}
	if got := ParseRetryAfter(neg); got != 0 {
		t.Errorf("ParseRetryAfter(-5) = %v, want 0", got)
	}
}

// TestMaxRetryAfterIsPinned guards the anti-DoS cap on a server-supplied
// Retry-After. Its own doc explains that without a bound a hostile server
// could park the retry loop indefinitely — and a mutation to 1000 minutes
// left the whole suite green, because the value was executed everywhere and
// compared nowhere.
func TestMaxRetryAfterIsPinned(t *testing.T) {
	if MaxRetryAfter != 10*time.Minute {
		t.Errorf("MaxRetryAfter = %v, want 10m", MaxRetryAfter)
	}
}

// TestShouldRetryError_TruncatedBody covers the read-failure branch: a body
// that ended early is a dropped connection, not a bad response, and neither
// sentinel implements net.Error, so the checks above them miss both.
func TestShouldRetryError_TruncatedBody(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want bool
	}{
		{"unexpected EOF", io.ErrUnexpectedEOF, true},
		{"EOF", io.EOF, true},
		{"wrapped unexpected EOF", fmt.Errorf("failed to read response body: %w", io.ErrUnexpectedEOF), true},
		{"an unrelated error is still not retryable", errors.New("boom"), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldRetryError(tt.err); got != tt.want {
				t.Errorf("ShouldRetryError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
