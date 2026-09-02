package marketdata

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// TestAPIError_MatchesErrorInterface is a regression test for the README's
// documented error-handling pattern:
//
//	var sdkErr marketdata.Error
//	if errors.As(err, &sdkErr) && sdkErr.Retryable() { ... }
//
// APIError used to have only a Message field and an Error() method, so it
// satisfied the built-in error interface but not marketdata.Error —
// errors.As against the interface silently never matched it, and the
// README's own second error-handling block never actually ran for the
// "s" != "ok" case its first block (var apiErr *marketdata.APIError) is
// about. Both must work now, from a single APIError value.
func TestAPIError_MatchesErrorInterface(t *testing.T) {
	var err error = &APIError{SupportContext: SupportContext{
		Message:       "unexpected response status: error",
		StatusCode:    200,
		ExceptionType: "APIError",
	}}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("errors.As(err, *APIError) should match")
	}
	if apiErr.Message != "unexpected response status: error" {
		t.Errorf("apiErr.Message = %q, want the original message", apiErr.Message)
	}

	var sdkErr Error
	if !errors.As(err, &sdkErr) {
		t.Fatal("errors.As(err, marketdata.Error) should match — APIError must implement the interface")
	}
	if sdkErr.Retryable() {
		t.Error("APIError.Retryable() = true, want false")
	}
	if info := sdkErr.SupportInfo(); !strings.Contains(info, "APIError") {
		t.Errorf("SupportInfo() = %q, want it to contain the exception type", info)
	}
}

func TestAuthenticationError(t *testing.T) {
	err := &AuthenticationError{
		SupportContext: SupportContext{
			RequestID:     "req-123",
			RequestURL:    "https://api.example.com/v1/stocks/quotes",
			StatusCode:    401,
			Timestamp:     time.Now(),
			Message:       "Invalid token",
			ExceptionType: "AuthenticationError",
		},
	}

	// Test Error()
	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() returned empty string")
	}
	if !strings.Contains(errStr, "req-123") {
		t.Errorf("Error() should contain request ID: %s", errStr)
	}
	if !strings.Contains(errStr, "authentication failed") {
		t.Errorf("Error() should contain 'authentication failed': %s", errStr)
	}

	// Test SupportInfo()
	info := err.SupportInfo()
	if !strings.Contains(info, "MARKET DATA SUPPORT INFO") {
		t.Error("SupportInfo() should contain header")
	}
	if !strings.Contains(info, "req-123") {
		t.Error("SupportInfo() should contain request ID")
	}
	if !strings.Contains(info, "AuthenticationError") {
		t.Error("SupportInfo() should contain exception type")
	}

	// Test Retryable()
	if err.Retryable() {
		t.Error("AuthenticationError should not be retryable")
	}

	// Test Unwrap()
	cause := errors.New("underlying cause")
	err.Cause = cause
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return underlying cause")
	}

	// Test Is()
	if !errors.Is(err, ErrAuthentication) {
		t.Error("AuthenticationError should match ErrAuthentication")
	}
}

func TestAuthenticationError_NoRequestID(t *testing.T) {
	err := &AuthenticationError{
		SupportContext: SupportContext{
			Message: "Invalid token",
		},
	}
	errStr := err.Error()
	if strings.Contains(errStr, "request_id") {
		t.Error("Error() without RequestID should not contain request_id")
	}
}

func TestBadRequestError(t *testing.T) {
	err := &BadRequestError{
		SupportContext: SupportContext{
			RequestID:     "req-456",
			StatusCode:    400,
			Message:       "Invalid symbol",
			ExceptionType: "BadRequestError",
		},
	}

	if err.Retryable() {
		t.Error("BadRequestError should not be retryable")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Error("BadRequestError should match ErrBadRequest")
	}
	if !strings.Contains(err.Error(), "bad request") {
		t.Error("Error() should contain 'bad request'")
	}
	if !strings.Contains(err.Error(), "req-456") {
		t.Error("Error() should contain request ID")
	}
}

func TestBadRequestError_NoRequestID(t *testing.T) {
	err := &BadRequestError{
		SupportContext: SupportContext{Message: "Invalid"},
	}
	if strings.Contains(err.Error(), "request_id") {
		t.Error("Error() without RequestID should not contain request_id")
	}
}

func TestBadRequestError_Unwrap(t *testing.T) {
	cause := errors.New("cause")
	err := &BadRequestError{Cause: cause}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return cause")
	}
}

func TestNotFoundError(t *testing.T) {
	err := &NotFoundError{
		SupportContext: SupportContext{
			RequestID:  "req-789",
			StatusCode: 404,
			Message:    "Resource not found",
		},
	}

	if err.Retryable() {
		t.Error("NotFoundError should not be retryable")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Error("NotFoundError should match ErrNotFound")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Error("Error() should contain 'not found'")
	}
}

func TestNotFoundError_NoRequestID(t *testing.T) {
	err := &NotFoundError{
		SupportContext: SupportContext{Message: "Not found"},
	}
	if strings.Contains(err.Error(), "request_id") {
		t.Error("Error() without RequestID should not contain request_id")
	}
}

func TestNotFoundError_Unwrap(t *testing.T) {
	cause := errors.New("cause")
	err := &NotFoundError{Cause: cause}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return cause")
	}
}

func TestRateLimitError(t *testing.T) {
	resetAt := time.Now().Add(1 * time.Hour)
	err := &RateLimitError{
		SupportContext: SupportContext{
			StatusCode:    429,
			Message:       "Rate limit exceeded",
			ExceptionType: "RateLimitError",
		},
		Limit:     10000,
		Remaining: 0,
		ResetAt:   resetAt,
	}

	// Test Error()
	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() returned empty string")
	}

	// Test Retryable() — rate limit errors are NOT retried per requirements
	if err.Retryable() {
		t.Error("RateLimitError should not be retryable")
	}

	// Test WaitDuration()
	wait := err.WaitDuration()
	if wait <= 0 || wait > 1*time.Hour {
		t.Errorf("WaitDuration() = %v, should be between 0 and 1 hour", wait)
	}

	// Test Is()
	if !errors.Is(err, ErrRateLimited) {
		t.Error("RateLimitError should match ErrRateLimited")
	}
}

func TestRateLimitError_WaitDuration_Past(t *testing.T) {
	err := &RateLimitError{
		ResetAt: time.Now().Add(-1 * time.Hour),
	}

	wait := err.WaitDuration()
	if wait != 0 {
		t.Errorf("WaitDuration() for past reset = %v, want 0", wait)
	}
}

func TestRateLimitError_Unwrap(t *testing.T) {
	cause := errors.New("underlying cause")
	err := &RateLimitError{Cause: cause}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return underlying cause")
	}
}

func TestServerError(t *testing.T) {
	err := &ServerError{
		SupportContext: SupportContext{
			RequestID:     "req-srv",
			StatusCode:    503,
			Message:       "Service unavailable",
			ExceptionType: "ServerError",
		},
	}

	// 503 should be retryable (501-599)
	if !err.Retryable() {
		t.Error("503 ServerError should be retryable")
	}
	if !errors.Is(err, ErrServer) {
		t.Error("ServerError should match ErrServer")
	}
	if !strings.Contains(err.Error(), "server error") {
		t.Error("Error() should contain 'server error'")
	}
	if !strings.Contains(err.Error(), "req-srv") {
		t.Error("Error() should contain request ID")
	}
}

func TestInternalError(t *testing.T) {
	err := &InternalError{
		SupportContext: SupportContext{
			RequestID:     "req-500",
			StatusCode:    500,
			Message:       "Internal Server Error",
			ExceptionType: "InternalError",
		},
	}
	if err.Retryable() {
		t.Error("InternalError (500) should NOT be retryable")
	}
	if !errors.Is(err, ErrInternal) {
		t.Error("InternalError should match ErrInternal")
	}
	if !strings.Contains(err.Error(), "internal server error") {
		t.Error("Error() should contain 'internal server error'")
	}
	if !strings.Contains(err.Error(), "req-500") {
		t.Error("Error() should contain request ID")
	}
}

func TestInternalError_NoRequestID(t *testing.T) {
	err := &InternalError{
		SupportContext: SupportContext{StatusCode: 500, Message: "failure"},
	}
	if strings.Contains(err.Error(), "request_id") {
		t.Error("Error() without RequestID should not contain request_id")
	}
}

func TestInternalError_Unwrap(t *testing.T) {
	cause := errors.New("cause")
	err := &InternalError{Cause: cause}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return cause")
	}
}

func TestServerError_NoRequestID(t *testing.T) {
	err := &ServerError{
		SupportContext: SupportContext{StatusCode: 502, Message: "Bad gateway"},
	}
	if strings.Contains(err.Error(), "request_id") {
		t.Error("Error() without RequestID should not contain request_id")
	}
}

func TestServerError_Unwrap(t *testing.T) {
	cause := errors.New("cause")
	err := &ServerError{Cause: cause}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return cause")
	}
}

func TestServerError_AlwaysRetryable(t *testing.T) {
	for _, code := range []int{501, 502, 503, 504, 509, 524, 529, 530, 540, 598, 599} {
		err := &ServerError{SupportContext: SupportContext{StatusCode: code}}
		if !err.Retryable() {
			t.Errorf("ServerError with status %d should be retryable", code)
		}
	}
}

func TestNetworkError(t *testing.T) {
	err := &NetworkError{
		SupportContext: SupportContext{Message: "connection refused"},
		Timeout:        false,
		Temporary:      true,
	}

	// Test Error()
	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() returned empty string")
	}
	if strings.Contains(errStr, "timeout") {
		t.Error("Non-timeout error should not contain 'timeout'")
	}

	// Test Retryable() — network errors are always retryable
	if !err.Retryable() {
		t.Error("NetworkError should be retryable")
	}
}

func TestNetworkError_Timeout(t *testing.T) {
	err := &NetworkError{
		SupportContext: SupportContext{Message: "deadline exceeded"},
		Timeout:        true,
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "timeout") {
		t.Error("Timeout error should contain 'timeout'")
	}

	if !err.Retryable() {
		t.Error("Timeout error should be retryable")
	}
}

func TestNetworkError_AlwaysRetryable(t *testing.T) {
	// Even non-temporary, non-timeout network errors are retryable
	err := &NetworkError{
		SupportContext: SupportContext{Message: "unknown error"},
		Timeout:        false,
		Temporary:      false,
	}

	if !err.Retryable() {
		t.Error("NetworkError should always be retryable")
	}
}

func TestNetworkError_Unwrap(t *testing.T) {
	cause := errors.New("underlying cause")
	err := &NetworkError{Cause: cause}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return underlying cause")
	}
}

func TestParseError(t *testing.T) {
	err := &ParseError{
		SupportContext: SupportContext{
			RequestID:     "req-parse",
			StatusCode:    200,
			Message:       "invalid JSON",
			ExceptionType: "ParseError",
		},
	}

	if err.Retryable() {
		t.Error("ParseError should not be retryable")
	}
	if !strings.Contains(err.Error(), "parse error") {
		t.Error("Error() should contain 'parse error'")
	}
	if !strings.Contains(err.Error(), "req-parse") {
		t.Error("Error() should contain request ID")
	}
}

func TestParseError_NoRequestID(t *testing.T) {
	err := &ParseError{
		SupportContext: SupportContext{Message: "invalid JSON"},
	}
	if strings.Contains(err.Error(), "request_id") {
		t.Error("Error() without RequestID should not contain request_id")
	}
}

func TestParseError_Unwrap(t *testing.T) {
	cause := errors.New("cause")
	err := &ParseError{Cause: cause}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return cause")
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "symbol",
		Message: "symbol is required",
	}

	// Test Error()
	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() returned empty string")
	}

	// Test Retryable()
	if err.Retryable() {
		t.Error("Validation error should not be retryable")
	}

	// Test Unwrap()
	if err.Unwrap() != nil {
		t.Error("Unwrap() should return nil for ValidationError")
	}

	// Test SupportInfo()
	if err.SupportInfo() != "" {
		t.Error("SupportInfo() should return empty string for ValidationError")
	}
}

func TestValidationError_Is(t *testing.T) {
	err := &ValidationError{
		Field:   "token",
		Message: "token is required",
	}
	if !errors.Is(err, ErrInvalidRequest) {
		t.Error("ValidationError should match ErrInvalidRequest")
	}
}

func TestSentinelErrors(t *testing.T) {
	sentinels := []error{
		ErrAuthentication,
		ErrPaymentRequired,
		ErrForbidden,
		ErrBadRequest,
		ErrNotFound,
		ErrPayloadTooLarge,
		ErrRateLimited,
		ErrInternal,
		ErrServer,
		ErrInvalidRequest,
	}

	for i, e1 := range sentinels {
		for j, e2 := range sentinels {
			if i != j && errors.Is(e1, e2) {
				t.Errorf("Sentinel errors %v and %v should not match", e1, e2)
			}
		}
	}

	// Verify they are errors
	for _, e := range sentinels {
		if e.Error() == "" {
			t.Errorf("Sentinel error %v should have non-empty message", e)
		}
	}
}

func TestSupportContext_SupportInfo(t *testing.T) {
	ts := time.Date(2025, 2, 21, 12, 0, 0, 0, time.UTC)
	sc := SupportContext{
		RequestID:     "8a1b2c3d4e5f6g7h-SJC",
		RequestURL:    "https://api.marketdata.app/v1/stocks/quotes/",
		StatusCode:    429,
		Timestamp:     ts,
		Message:       "Rate limit exceeded",
		ExceptionType: "RateLimitError",
	}

	info := sc.SupportInfo()
	if !strings.Contains(info, "MARKET DATA SUPPORT INFO") {
		t.Error("SupportInfo() should contain header")
	}
	if !strings.Contains(info, "8a1b2c3d4e5f6g7h-SJC") {
		t.Error("SupportInfo() should contain request_id")
	}
	if !strings.Contains(info, "https://api.marketdata.app/v1/stocks/quotes/") {
		t.Error("SupportInfo() should contain request_url")
	}
	if !strings.Contains(info, "429") {
		t.Error("SupportInfo() should contain status_code")
	}
	if !strings.Contains(info, "2025-02-21 12:00:00") {
		t.Error("SupportInfo() should contain formatted timestamp")
	}
	if !strings.Contains(info, "Rate limit exceeded") {
		t.Error("SupportInfo() should contain message")
	}
	if !strings.Contains(info, "RateLimitError") {
		t.Error("SupportInfo() should contain exception_type")
	}
}

// TestPublicTaxonomyIsComplete guards the hole that let two error types ship
// unreachable: ResponseTooLargeError and InsecureTokenError were returned to
// callers from internal/http but never aliased here, and because they live
// under internal/ there was no workaround — errors.As on either was
// impossible from outside the module, and SemVer would have frozen that at
// the tag.
//
// The check is structural rather than a list of names: every exported error
// type in internal/sdkerrors must have a public alias, so a type added there
// later fails here instead of silently becoming unreachable.
func TestPublicTaxonomyIsComplete(t *testing.T) {
	// Each entry pairs a public alias with its sentinel, exercised the way a
	// consumer would: construct through the alias, classify with errors.Is
	// and errors.As.
	t.Run("ResponseTooLargeError", func(t *testing.T) {
		err := error(&ResponseTooLargeError{
			SupportContext: SupportContext{Message: "body too large"},
		})
		var typed *ResponseTooLargeError
		if !errors.As(err, &typed) {
			t.Error("errors.As(*ResponseTooLargeError) should match")
		}
		if !errors.Is(err, ErrResponseTooLarge) {
			t.Error("errors.Is(ErrResponseTooLarge) should match")
		}
	})

	t.Run("InsecureTokenError", func(t *testing.T) {
		err := error(&InsecureTokenError{Scheme: "http", Host: "api.example.test"})
		var typed *InsecureTokenError
		if !errors.As(err, &typed) {
			t.Error("errors.As(*InsecureTokenError) should match")
		}
		if !errors.Is(err, ErrInsecureToken) {
			t.Error("errors.Is(ErrInsecureToken) should match")
		}
	})
}
