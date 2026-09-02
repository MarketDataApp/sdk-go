package sdkerrors

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestSupportContext_SupportInfo(t *testing.T) {
	ctx := SupportContext{
		RequestID:     "req-123",
		RequestURL:    "https://api.example.com/v1/test",
		StatusCode:    400,
		Timestamp:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Message:       "test error",
		ExceptionType: "BadRequestError",
	}

	info := ctx.SupportInfo()
	if info == "" {
		t.Error("SupportInfo() returned empty string")
	}
	for _, want := range []string{"req-123", "https://api.example.com/v1/test", "400", "test error", "BadRequestError"} {
		if !containsStr(info, want) {
			t.Errorf("SupportInfo() should contain %q", want)
		}
	}
}

// AuthenticationError tests

func TestAuthenticationError_Error_WithRequestID(t *testing.T) {
	err := &AuthenticationError{
		SupportContext: SupportContext{RequestID: "req-123", Message: "invalid token"},
	}
	errStr := err.Error()
	if !containsStr(errStr, "req-123") {
		t.Errorf("Error() = %q, should contain request ID", errStr)
	}
	if !containsStr(errStr, "authentication failed") {
		t.Errorf("Error() = %q, should contain 'authentication failed'", errStr)
	}
}

func TestAuthenticationError_Error_WithoutRequestID(t *testing.T) {
	err := &AuthenticationError{
		SupportContext: SupportContext{Message: "invalid token"},
	}
	errStr := err.Error()
	if !containsStr(errStr, "authentication failed") {
		t.Errorf("Error() = %q, should contain 'authentication failed'", errStr)
	}
}

func TestAuthenticationError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := &AuthenticationError{Cause: cause}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestAuthenticationError_Retryable(t *testing.T) {
	err := &AuthenticationError{}
	if err.Retryable() {
		t.Error("AuthenticationError should not be retryable")
	}
}

func TestAuthenticationError_Is(t *testing.T) {
	err := &AuthenticationError{}
	if !err.Is(ErrAuthentication) {
		t.Error("AuthenticationError.Is(ErrAuthentication) should be true")
	}
	if err.Is(ErrBadRequest) {
		t.Error("AuthenticationError.Is(ErrBadRequest) should be false")
	}
}

// BadRequestError tests

func TestBadRequestError_Error_WithRequestID(t *testing.T) {
	err := &BadRequestError{
		SupportContext: SupportContext{RequestID: "req-123", Message: "bad request"},
	}
	if !containsStr(err.Error(), "req-123") {
		t.Errorf("Error() should contain request ID")
	}
}

func TestBadRequestError_Error_WithoutRequestID(t *testing.T) {
	err := &BadRequestError{
		SupportContext: SupportContext{Message: "bad request"},
	}
	if !containsStr(err.Error(), "bad request") {
		t.Errorf("Error() should contain message")
	}
}

func TestBadRequestError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root")
	err := &BadRequestError{Cause: cause}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestBadRequestError_Retryable(t *testing.T) {
	err := &BadRequestError{}
	if err.Retryable() {
		t.Error("BadRequestError should not be retryable")
	}
}

func TestBadRequestError_Is(t *testing.T) {
	err := &BadRequestError{}
	if !err.Is(ErrBadRequest) {
		t.Error("BadRequestError.Is(ErrBadRequest) should be true")
	}
}

// NotFoundError tests

func TestNotFoundError_Error_WithRequestID(t *testing.T) {
	err := &NotFoundError{
		SupportContext: SupportContext{RequestID: "req-123", Message: "not found"},
	}
	if !containsStr(err.Error(), "req-123") {
		t.Errorf("Error() should contain request ID")
	}
}

func TestNotFoundError_Error_WithoutRequestID(t *testing.T) {
	err := &NotFoundError{
		SupportContext: SupportContext{Message: "not found"},
	}
	if !containsStr(err.Error(), "not found") {
		t.Errorf("Error() should contain message")
	}
}

func TestNotFoundError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root")
	err := &NotFoundError{Cause: cause}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestNotFoundError_Retryable(t *testing.T) {
	err := &NotFoundError{}
	if err.Retryable() {
		t.Error("NotFoundError should not be retryable")
	}
}

func TestNotFoundError_Is(t *testing.T) {
	err := &NotFoundError{}
	if !err.Is(ErrNotFound) {
		t.Error("NotFoundError.Is(ErrNotFound) should be true")
	}
}

// RateLimitError tests

func TestRateLimitError_Error(t *testing.T) {
	err := &RateLimitError{
		Limit:     10000,
		Remaining: 0,
		ResetAt:   time.Now().Add(1 * time.Hour),
	}
	errStr := err.Error()
	if !containsStr(errStr, "rate limit exceeded") {
		t.Errorf("Error() = %q, should contain 'rate limit exceeded'", errStr)
	}
}

func TestRateLimitError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root")
	err := &RateLimitError{Cause: cause}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestRateLimitError_Retryable(t *testing.T) {
	err := &RateLimitError{}
	if err.Retryable() {
		t.Error("RateLimitError should not be retryable")
	}
}

func TestRateLimitError_Is(t *testing.T) {
	err := &RateLimitError{}
	if !err.Is(ErrRateLimited) {
		t.Error("RateLimitError.Is(ErrRateLimited) should be true")
	}
}

func TestRateLimitError_WaitDuration_Future(t *testing.T) {
	err := &RateLimitError{
		ResetAt: time.Now().Add(5 * time.Minute),
	}
	dur := err.WaitDuration()
	if dur <= 0 {
		t.Error("WaitDuration() should be positive for future reset time")
	}
	if dur > 6*time.Minute {
		t.Errorf("WaitDuration() = %v, should be close to 5 minutes", dur)
	}
}

func TestRateLimitError_WaitDuration_Past(t *testing.T) {
	err := &RateLimitError{
		ResetAt: time.Now().Add(-5 * time.Minute),
	}
	dur := err.WaitDuration()
	if dur != 0 {
		t.Errorf("WaitDuration() = %v, should be 0 for past reset time", dur)
	}
}

// ServerError tests

func TestServerError_Error_WithRequestID(t *testing.T) {
	err := &ServerError{
		SupportContext: SupportContext{RequestID: "req-123", StatusCode: 503, Message: "unavailable"},
	}
	if !containsStr(err.Error(), "req-123") {
		t.Errorf("Error() should contain request ID")
	}
	if !containsStr(err.Error(), "server error") {
		t.Errorf("Error() should contain 'server error'")
	}
}

func TestServerError_Error_WithoutRequestID(t *testing.T) {
	err := &ServerError{
		SupportContext: SupportContext{StatusCode: 500, Message: "internal"},
	}
	errStr := err.Error()
	if !containsStr(errStr, "server error") {
		t.Errorf("Error() = %q, should contain 'server error'", errStr)
	}
}

func TestServerError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root")
	err := &ServerError{Cause: cause}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestServerError_Retryable_503(t *testing.T) {
	err := &ServerError{SupportContext: SupportContext{StatusCode: 503}}
	if !err.Retryable() {
		t.Error("503 should be retryable")
	}
}

func TestServerError_Is(t *testing.T) {
	err := &ServerError{}
	if !err.Is(ErrServer) {
		t.Error("ServerError.Is(ErrServer) should be true")
	}
}

// NetworkError tests

func TestNetworkError_Error_Timeout(t *testing.T) {
	err := &NetworkError{
		SupportContext: SupportContext{Message: "connection timed out"},
		Timeout:        true,
	}
	errStr := err.Error()
	if !containsStr(errStr, "network timeout") {
		t.Errorf("Error() = %q, should contain 'network timeout'", errStr)
	}
}

func TestNetworkError_Error_General(t *testing.T) {
	err := &NetworkError{
		SupportContext: SupportContext{Message: "connection refused"},
	}
	errStr := err.Error()
	if !containsStr(errStr, "network error") {
		t.Errorf("Error() = %q, should contain 'network error'", errStr)
	}
}

func TestNetworkError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root")
	err := &NetworkError{Cause: cause}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestNetworkError_Retryable(t *testing.T) {
	err := &NetworkError{}
	if !err.Retryable() {
		t.Error("NetworkError should always be retryable")
	}
}

// ParseError tests

func TestParseError_Error_WithRequestID(t *testing.T) {
	err := &ParseError{
		SupportContext: SupportContext{RequestID: "req-123", Message: "invalid json"},
	}
	if !containsStr(err.Error(), "req-123") {
		t.Errorf("Error() should contain request ID")
	}
	if !containsStr(err.Error(), "parse error") {
		t.Errorf("Error() should contain 'parse error'")
	}
}

func TestParseError_Error_WithoutRequestID(t *testing.T) {
	err := &ParseError{
		SupportContext: SupportContext{Message: "invalid json"},
	}
	if !containsStr(err.Error(), "parse error") {
		t.Errorf("Error() should contain 'parse error'")
	}
}

func TestParseError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root")
	err := &ParseError{Cause: cause}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestParseError_Retryable(t *testing.T) {
	err := &ParseError{}
	if err.Retryable() {
		t.Error("ParseError should not be retryable")
	}
}

// APIError tests

// var _ Error is a compile-time check that *APIError implements the Error
// interface — the regression this pins: APIError used to have only a
// Message field and an Error() method, so it satisfied error but not Error,
// and this line would not compile.
var _ Error = (*APIError)(nil)

func TestAPIError_Error(t *testing.T) {
	err := &APIError{SupportContext: SupportContext{Message: "something went wrong", RequestID: "req-123"}}
	if !containsStr(err.Error(), "API error") {
		t.Errorf("Error() should contain 'API error'")
	}
	if !containsStr(err.Error(), "something went wrong") {
		t.Errorf("Error() should contain the message")
	}
	if !containsStr(err.Error(), "req-123") {
		t.Errorf("Error() should contain the request ID")
	}
}

func TestAPIError_NoRequestID(t *testing.T) {
	err := &APIError{SupportContext: SupportContext{Message: "something went wrong"}}
	if containsStr(err.Error(), "request_id") {
		t.Error("Error() without RequestID should not contain request_id")
	}
}

func TestAPIError_Retryable(t *testing.T) {
	err := &APIError{}
	if err.Retryable() {
		t.Error("APIError should not be retryable")
	}
}

func TestAPIError_Unwrap(t *testing.T) {
	err := &APIError{}
	if err.Unwrap() != nil {
		t.Error("APIError.Unwrap() should return nil")
	}
}

func TestAPIError_SupportInfo(t *testing.T) {
	err := &APIError{SupportContext: SupportContext{Message: "boom", StatusCode: 200, ExceptionType: "APIError"}}
	if !containsStr(err.SupportInfo(), "boom") {
		t.Error("SupportInfo() should contain the message")
	}
}

// ValidationError tests

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{Field: "symbol", Message: "is required"}
	errStr := err.Error()
	if !containsStr(errStr, "validation error") {
		t.Errorf("Error() should contain 'validation error'")
	}
	if !containsStr(errStr, "symbol") {
		t.Errorf("Error() should contain field name")
	}
}

func TestValidationError_Unwrap(t *testing.T) {
	err := &ValidationError{}
	if err.Unwrap() != nil {
		t.Error("ValidationError.Unwrap() should return nil")
	}
}

func TestValidationError_Retryable(t *testing.T) {
	err := &ValidationError{}
	if err.Retryable() {
		t.Error("ValidationError should not be retryable")
	}
}

func TestValidationError_SupportInfo(t *testing.T) {
	err := &ValidationError{}
	if err.SupportInfo() != "" {
		t.Error("ValidationError.SupportInfo() should return empty string")
	}
}

func TestValidationError_Is(t *testing.T) {
	err := &ValidationError{}
	if !err.Is(ErrInvalidRequest) {
		t.Error("ValidationError.Is(ErrInvalidRequest) should be true")
	}
}

// PaymentRequiredError tests

func TestPaymentRequiredError_Error_WithRequestID(t *testing.T) {
	err := &PaymentRequiredError{
		SupportContext: SupportContext{RequestID: "req-123", Message: "upgrade required"},
	}
	errStr := err.Error()
	if !containsStr(errStr, "req-123") {
		t.Errorf("Error() = %q, should contain request ID", errStr)
	}
	if !containsStr(errStr, "payment required") {
		t.Errorf("Error() = %q, should contain 'payment required'", errStr)
	}
}

func TestPaymentRequiredError_Error_WithoutRequestID(t *testing.T) {
	err := &PaymentRequiredError{
		SupportContext: SupportContext{Message: "upgrade required"},
	}
	if !containsStr(err.Error(), "payment required") {
		t.Errorf("Error() should contain 'payment required'")
	}
}

func TestPaymentRequiredError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root")
	err := &PaymentRequiredError{Cause: cause}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestPaymentRequiredError_Retryable(t *testing.T) {
	err := &PaymentRequiredError{}
	if err.Retryable() {
		t.Error("PaymentRequiredError should not be retryable")
	}
}

func TestPaymentRequiredError_Is(t *testing.T) {
	err := &PaymentRequiredError{}
	if !err.Is(ErrPaymentRequired) {
		t.Error("PaymentRequiredError.Is(ErrPaymentRequired) should be true")
	}
}

// ForbiddenError tests

func TestForbiddenError_Error_WithRequestID(t *testing.T) {
	err := &ForbiddenError{
		SupportContext: SupportContext{RequestID: "req-123", Message: "access denied"},
	}
	errStr := err.Error()
	if !containsStr(errStr, "req-123") {
		t.Errorf("Error() = %q, should contain request ID", errStr)
	}
	if !containsStr(errStr, "forbidden") {
		t.Errorf("Error() = %q, should contain 'forbidden'", errStr)
	}
}

func TestForbiddenError_Error_WithoutRequestID(t *testing.T) {
	err := &ForbiddenError{
		SupportContext: SupportContext{Message: "access denied"},
	}
	if !containsStr(err.Error(), "forbidden") {
		t.Errorf("Error() should contain 'forbidden'")
	}
}

func TestForbiddenError_IPFields(t *testing.T) {
	err := &ForbiddenError{
		SupportContext:       SupportContext{Message: "access denied"},
		AuthorizedIP:         "107.178.202.2",
		BlockedIP:            "44.116.21.32",
		TroubleshootingGuide: "https://www.marketdata.app/docs/api/troubleshooting/multiple-ip-addresses",
	}
	if err.AuthorizedIP != "107.178.202.2" {
		t.Errorf("AuthorizedIP = %q, want %q", err.AuthorizedIP, "107.178.202.2")
	}
	if err.BlockedIP != "44.116.21.32" {
		t.Errorf("BlockedIP = %q, want %q", err.BlockedIP, "44.116.21.32")
	}
	if err.TroubleshootingGuide == "" {
		t.Error("TroubleshootingGuide should not be empty")
	}
}

func TestForbiddenError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root")
	err := &ForbiddenError{Cause: cause}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestForbiddenError_Retryable(t *testing.T) {
	err := &ForbiddenError{}
	if err.Retryable() {
		t.Error("ForbiddenError should not be retryable")
	}
}

func TestForbiddenError_Is(t *testing.T) {
	err := &ForbiddenError{}
	if !err.Is(ErrForbidden) {
		t.Error("ForbiddenError.Is(ErrForbidden) should be true")
	}
}

// PayloadTooLargeError tests

func TestPayloadTooLargeError_Error_WithRequestID(t *testing.T) {
	err := &PayloadTooLargeError{
		SupportContext: SupportContext{RequestID: "req-123", Message: "too large"},
	}
	errStr := err.Error()
	if !containsStr(errStr, "req-123") {
		t.Errorf("Error() = %q, should contain request ID", errStr)
	}
	if !containsStr(errStr, "payload too large") {
		t.Errorf("Error() = %q, should contain 'payload too large'", errStr)
	}
}

func TestPayloadTooLargeError_Error_WithoutRequestID(t *testing.T) {
	err := &PayloadTooLargeError{
		SupportContext: SupportContext{Message: "too large"},
	}
	if !containsStr(err.Error(), "payload too large") {
		t.Errorf("Error() should contain 'payload too large'")
	}
}

func TestPayloadTooLargeError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root")
	err := &PayloadTooLargeError{Cause: cause}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestPayloadTooLargeError_Retryable(t *testing.T) {
	err := &PayloadTooLargeError{}
	if err.Retryable() {
		t.Error("PayloadTooLargeError should not be retryable")
	}
}

func TestPayloadTooLargeError_Is(t *testing.T) {
	err := &PayloadTooLargeError{}
	if !err.Is(ErrPayloadTooLarge) {
		t.Error("PayloadTooLargeError.Is(ErrPayloadTooLarge) should be true")
	}
}

// InternalError tests

func TestInternalError_Error_WithRequestID(t *testing.T) {
	err := &InternalError{
		SupportContext: SupportContext{RequestID: "req-123", Message: "unknown failure"},
	}
	errStr := err.Error()
	if !containsStr(errStr, "req-123") {
		t.Errorf("Error() = %q, should contain request ID", errStr)
	}
	if !containsStr(errStr, "internal server error") {
		t.Errorf("Error() = %q, should contain 'internal server error'", errStr)
	}
}

func TestInternalError_Error_WithoutRequestID(t *testing.T) {
	err := &InternalError{
		SupportContext: SupportContext{Message: "unknown failure"},
	}
	if !containsStr(err.Error(), "internal server error") {
		t.Errorf("Error() should contain 'internal server error'")
	}
}

func TestInternalError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root")
	err := &InternalError{Cause: cause}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestInternalError_Retryable(t *testing.T) {
	err := &InternalError{}
	if err.Retryable() {
		t.Error("InternalError should not be retryable")
	}
}

func TestInternalError_Is(t *testing.T) {
	err := &InternalError{}
	if !err.Is(ErrInternal) {
		t.Error("InternalError.Is(ErrInternal) should be true")
	}
	if err.Is(ErrServer) {
		t.Error("InternalError.Is(ErrServer) should be false")
	}
}

// ServerError retryable — now always true (501-599 only)

func TestServerError_Retryable_AlwaysTrue(t *testing.T) {
	for _, code := range []int{501, 502, 503, 504, 509, 524, 529, 530, 540, 598, 599} {
		err := &ServerError{SupportContext: SupportContext{StatusCode: code}}
		if !err.Retryable() {
			t.Errorf("ServerError with status %d should be retryable", code)
		}
	}
}

// Sentinel errors

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"ErrAuthentication", ErrAuthentication, "authentication"},
		{"ErrPaymentRequired", ErrPaymentRequired, "payment required"},
		{"ErrForbidden", ErrForbidden, "forbidden"},
		{"ErrBadRequest", ErrBadRequest, "bad request"},
		{"ErrNotFound", ErrNotFound, "not found"},
		{"ErrPayloadTooLarge", ErrPayloadTooLarge, "payload too large"},
		{"ErrRateLimited", ErrRateLimited, "rate limit"},
		{"ErrInternal", ErrInternal, "internal server error"},
		{"ErrServer", ErrServer, "server error"},
		{"ErrInvalidRequest", ErrInvalidRequest, "invalid request"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !containsStr(tt.err.Error(), tt.want) {
				t.Errorf("%s.Error() = %q, should contain %q", tt.name, tt.err.Error(), tt.want)
			}
		})
	}
}

// Test errors.Is integration with wrapped errors

func TestErrorsIs_AuthenticationError(t *testing.T) {
	err := &AuthenticationError{Cause: ErrAuthentication}
	if !errors.Is(err, ErrAuthentication) {
		t.Error("errors.Is should match ErrAuthentication")
	}
}

func TestErrorsIs_PaymentRequiredError(t *testing.T) {
	err := &PaymentRequiredError{Cause: ErrPaymentRequired}
	if !errors.Is(err, ErrPaymentRequired) {
		t.Error("errors.Is should match ErrPaymentRequired")
	}
}

func TestErrorsIs_ForbiddenError(t *testing.T) {
	err := &ForbiddenError{Cause: ErrForbidden}
	if !errors.Is(err, ErrForbidden) {
		t.Error("errors.Is should match ErrForbidden")
	}
}

func TestErrorsIs_BadRequestError(t *testing.T) {
	err := &BadRequestError{Cause: ErrBadRequest}
	if !errors.Is(err, ErrBadRequest) {
		t.Error("errors.Is should match ErrBadRequest")
	}
}

func TestErrorsIs_NotFoundError(t *testing.T) {
	err := &NotFoundError{Cause: ErrNotFound}
	if !errors.Is(err, ErrNotFound) {
		t.Error("errors.Is should match ErrNotFound")
	}
}

func TestErrorsIs_PayloadTooLargeError(t *testing.T) {
	err := &PayloadTooLargeError{Cause: ErrPayloadTooLarge}
	if !errors.Is(err, ErrPayloadTooLarge) {
		t.Error("errors.Is should match ErrPayloadTooLarge")
	}
}

func TestErrorsIs_RateLimitError(t *testing.T) {
	err := &RateLimitError{Cause: ErrRateLimited}
	if !errors.Is(err, ErrRateLimited) {
		t.Error("errors.Is should match ErrRateLimited")
	}
}

func TestErrorsIs_InternalError(t *testing.T) {
	err := &InternalError{Cause: ErrInternal}
	if !errors.Is(err, ErrInternal) {
		t.Error("errors.Is should match ErrInternal")
	}
}

func TestErrorsIs_ServerError(t *testing.T) {
	err := &ServerError{Cause: ErrServer}
	if !errors.Is(err, ErrServer) {
		t.Error("errors.Is should match ErrServer")
	}
}

func TestErrorsIs_ValidationError(t *testing.T) {
	err := &ValidationError{Field: "test", Message: "test"}
	if !errors.Is(err, ErrInvalidRequest) {
		t.Error("errors.Is should match ErrInvalidRequest")
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
