// Package retry provides retry logic with exponential backoff.
package retry

import (
	"context"
	"errors"
	"io"
	"math"
	"net"
	"net/http"
	"strconv"
	"time"
)

// MaxRetryAfter is the maximum server-supplied Retry-After delay that will
// be honored. A hostile or broken server could otherwise park the retry
// loop indefinitely; above this cap the calculated backoff is used instead.
const MaxRetryAfter = 10 * time.Minute

// Config holds retry configuration.
type Config struct {
	MaxRetries     int
	InitialBackoff time.Duration
	Multiplier     float64
}

// DefaultConfig returns the default retry configuration.
// MaxRetries can be overridden by the user via WithMaxRetries.
func DefaultConfig() Config {
	return Config{
		MaxRetries:     3,
		InitialBackoff: 1 * time.Second,
		Multiplier:     2.0,
	}
}

// ShouldRetryStatus returns true if the HTTP status code indicates
// the request can be retried. Only 501-599 are retryable.
// 429 (rate limit) and 500 (internal server error) are NOT retried.
func ShouldRetryStatus(statusCode int) bool {
	return statusCode > 500 && statusCode < 600
}

// ShouldRetryError returns true if the error is transient
// and the request can be retried. It unwraps errors to check
// the underlying cause.
func ShouldRetryError(err error) bool {
	if err == nil {
		return false
	}

	// Check for DNS errors first (they also implement net.Error)
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	// Check for timeout or temporary errors. net.Error.Temporary is deprecated
	// and unreliable, but it remains a useful best-effort signal for the few
	// non-timeout transient errors; timeouts are handled explicitly alongside.
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary() //nolint:staticcheck // deprecated Temporary() is an intentional best-effort transient signal
	}

	// A body that ended early is a dropped connection, not a bad response:
	// the server said how much was coming and the transport did not deliver
	// it. Retrying is the point. These do not implement net.Error, so the
	// checks above miss them.
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return true
	}

	return false
}

// CalculateBackoff returns the backoff duration for the given attempt.
// Attempt is 0-indexed (first retry is attempt 0).
// Formula: initial * 2^attempt, uncapped per SDK requirements §9.3.
// The result is clamped to MaxInt64 nanoseconds so extreme user-configured
// retry counts cannot overflow time.Duration.
func CalculateBackoff(cfg Config, attempt int) time.Duration {
	backoff := float64(cfg.InitialBackoff) * math.Pow(cfg.Multiplier, float64(attempt))
	if backoff >= float64(math.MaxInt64) {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(backoff)
}

// ParseRetryAfter parses the Retry-After header value.
// Returns the duration to wait, or 0 if not present/parseable.
func ParseRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}

	header := resp.Header.Get("Retry-After")
	if header == "" {
		return 0
	}

	// Try parsing as seconds
	if seconds, err := strconv.ParseInt(header, 10, 64); err == nil {
		if seconds <= 0 {
			return 0
		}
		// Guard duration overflow: an absurd value would wrap negative
		// and silently disable the Retry-After handling.
		if seconds > math.MaxInt64/int64(time.Second) {
			return time.Duration(math.MaxInt64)
		}
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP date
	if t, err := http.ParseTime(header); err == nil {
		wait := time.Until(t)
		if wait > 0 {
			return wait
		}
	}

	return 0
}

// Wait waits for the specified duration or until the context is canceled.
// Returns ctx.Err() if the context is canceled.
func Wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
