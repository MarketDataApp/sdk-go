// Package sdkerrors defines the SDK error types shared across all packages.
//
// These types are re-exported as type aliases by the public marketdata package.
package sdkerrors

import (
	"errors"
	"fmt"
	"time"
)

// Common sentinel errors for quick checks with errors.Is
var (
	// ErrAuthentication indicates invalid or missing API token (401)
	ErrAuthentication = errors.New("marketdata: authentication failed")

	// ErrPaymentRequired indicates the request requires a higher plan (402)
	ErrPaymentRequired = errors.New("marketdata: payment required")

	// ErrForbidden indicates access is denied due to IP policy violation (403)
	ErrForbidden = errors.New("marketdata: forbidden")

	// ErrBadRequest indicates invalid parameters (400)
	ErrBadRequest = errors.New("marketdata: bad request")

	// ErrNotFound indicates the requested resource doesn't exist (404)
	ErrNotFound = errors.New("marketdata: not found")

	// ErrPayloadTooLarge indicates the request spans too much data (413)
	ErrPayloadTooLarge = errors.New("marketdata: payload too large")

	// ErrRateLimited indicates rate limit has been exceeded (429)
	ErrRateLimited = errors.New("marketdata: rate limit exceeded")

	// ErrResponseTooLarge indicates the API response body exceeded the SDK's
	// safety cap and was refused to protect the caller from memory exhaustion.
	ErrResponseTooLarge = errors.New("marketdata: response too large")

	// ErrInsecureToken indicates the SDK refused to transmit the API token
	// over a connection that is not HTTPS (and not a loopback host).
	ErrInsecureToken = errors.New("marketdata: refusing to send token over insecure transport")

	// ErrInternal indicates an internal server error (500) — not retryable
	ErrInternal = errors.New("marketdata: internal server error")

	// ErrServer indicates a temporary server error (501-599) — retryable
	ErrServer = errors.New("marketdata: server error")

	// ErrInvalidRequest indicates a client-side validation error
	ErrInvalidRequest = errors.New("marketdata: invalid request")
)

// Error is the interface implemented by all SDK errors.
type Error interface {
	error
	Unwrap() error

	// Retryable returns true if the operation can be retried
	Retryable() bool

	// SupportInfo returns a formatted string for support tickets
	SupportInfo() string
}

// SupportContext contains fields included in every API error for support troubleshooting.
type SupportContext struct {
	// RequestID is the cf-ray response header value
	RequestID string

	// RequestURL is the full request URL
	RequestURL string

	// StatusCode is the HTTP status code
	StatusCode int

	// Timestamp is when the error occurred (US/Eastern)
	Timestamp time.Time

	// Message is the error description
	Message string

	// ExceptionType is the error type name
	ExceptionType string
}

// SupportInfo returns a formatted support string for support tickets.
func (s SupportContext) SupportInfo() string {
	return fmt.Sprintf(
		"--- MARKET DATA SUPPORT INFO ---\n"+
			"request_id:     %s\n"+
			"request_url:    %s\n"+
			"status_code:    %d\n"+
			"timestamp:      %s\n"+
			"message:        %s\n"+
			"exception_type: %s\n"+
			"--------------------------------",
		s.RequestID, s.RequestURL, s.StatusCode,
		s.Timestamp.Format("2006-01-02 15:04:05"),
		s.Message, s.ExceptionType,
	)
}

// AuthenticationError represents a 401 response (invalid or missing token).
type AuthenticationError struct {
	SupportContext
	Cause error
}

func (e *AuthenticationError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("marketdata: authentication failed: %s (request_id=%s)", e.Message, e.RequestID)
	}
	return fmt.Sprintf("marketdata: authentication failed: %s", e.Message)
}

func (e *AuthenticationError) Unwrap() error   { return e.Cause }
func (e *AuthenticationError) Retryable() bool { return false }
func (e *AuthenticationError) Is(target error) bool {
	return errors.Is(target, ErrAuthentication)
}

// PaymentRequiredError represents a 402 response (plan limitation).
// The request is valid but the user's plan does not include the requested feature or data.
type PaymentRequiredError struct {
	SupportContext
	Cause error
}

func (e *PaymentRequiredError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("marketdata: payment required: %s (request_id=%s)", e.Message, e.RequestID)
	}
	return fmt.Sprintf("marketdata: payment required: %s", e.Message)
}

func (e *PaymentRequiredError) Unwrap() error   { return e.Cause }
func (e *PaymentRequiredError) Retryable() bool { return false }
func (e *PaymentRequiredError) Is(target error) bool {
	return errors.Is(target, ErrPaymentRequired)
}

// ForbiddenError represents a 403 response (IP policy violation).
// Occurs when the user's IP address changes and the account is temporarily blocked.
type ForbiddenError struct {
	SupportContext

	// AuthorizedIP is the IP address that is currently authorized
	AuthorizedIP string

	// BlockedIP is the IP address that was blocked
	BlockedIP string

	// TroubleshootingGuide is a URL to the relevant troubleshooting documentation
	TroubleshootingGuide string

	Cause error
}

func (e *ForbiddenError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("marketdata: forbidden: %s (request_id=%s)", e.Message, e.RequestID)
	}
	return fmt.Sprintf("marketdata: forbidden: %s", e.Message)
}

func (e *ForbiddenError) Unwrap() error   { return e.Cause }
func (e *ForbiddenError) Retryable() bool { return false }
func (e *ForbiddenError) Is(target error) bool {
	return errors.Is(target, ErrForbidden)
}

// BadRequestError represents a 400 response (invalid parameters).
type BadRequestError struct {
	SupportContext
	Cause error
}

func (e *BadRequestError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("marketdata: bad request: %s (request_id=%s)", e.Message, e.RequestID)
	}
	return fmt.Sprintf("marketdata: bad request: %s", e.Message)
}

func (e *BadRequestError) Unwrap() error   { return e.Cause }
func (e *BadRequestError) Retryable() bool { return false }
func (e *BadRequestError) Is(target error) bool {
	return errors.Is(target, ErrBadRequest)
}

// NotFoundError represents a 404 response.
type NotFoundError struct {
	SupportContext
	Cause error
}

func (e *NotFoundError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("marketdata: not found: %s (request_id=%s)", e.Message, e.RequestID)
	}
	return fmt.Sprintf("marketdata: not found: %s", e.Message)
}

func (e *NotFoundError) Unwrap() error   { return e.Cause }
func (e *NotFoundError) Retryable() bool { return false }
func (e *NotFoundError) Is(target error) bool {
	return errors.Is(target, ErrNotFound)
}

// PayloadTooLargeError represents a 413 response (request spans too much data).
// Typically occurs when an intraday candle request spans more than 1 year.
type PayloadTooLargeError struct {
	SupportContext
	Cause error
}

func (e *PayloadTooLargeError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("marketdata: payload too large: %s (request_id=%s)", e.Message, e.RequestID)
	}
	return fmt.Sprintf("marketdata: payload too large: %s", e.Message)
}

func (e *PayloadTooLargeError) Unwrap() error   { return e.Cause }
func (e *PayloadTooLargeError) Retryable() bool { return false }
func (e *PayloadTooLargeError) Is(target error) bool {
	return errors.Is(target, ErrPayloadTooLarge)
}

// ResponseTooLargeError is returned when an API response body exceeds the
// SDK's configured size cap. The SDK stops reading and refuses the response so
// a hostile or malfunctioning server cannot exhaust the caller's memory.
type ResponseTooLargeError struct {
	SupportContext

	// Limit is the maximum number of response bytes the SDK will read.
	Limit int64

	Cause error
}

func (e *ResponseTooLargeError) Error() string {
	return fmt.Sprintf("marketdata: response too large: exceeded %d-byte limit", e.Limit)
}

func (e *ResponseTooLargeError) Unwrap() error   { return e.Cause }
func (e *ResponseTooLargeError) Retryable() bool { return false }
func (e *ResponseTooLargeError) Is(target error) bool {
	return errors.Is(target, ErrResponseTooLarge)
}

// InsecureTokenError is returned when the SDK is asked to send the API token
// over a connection that is neither HTTPS nor a loopback host, which would
// expose the token in cleartext.
type InsecureTokenError struct {
	// Scheme is the URL scheme that was refused (for example "http").
	Scheme string

	// Host is the target host that was refused.
	Host string
}

func (e *InsecureTokenError) Error() string {
	return fmt.Sprintf("marketdata: refusing to send API token over insecure %s connection to %q (use https, or a loopback host for local development)", e.Scheme, e.Host)
}

func (e *InsecureTokenError) Unwrap() error   { return nil }
func (e *InsecureTokenError) Retryable() bool { return false }
func (e *InsecureTokenError) Is(target error) bool {
	return errors.Is(target, ErrInsecureToken)
}

// RateLimitError represents a 429 response (rate limit exceeded).
type RateLimitError struct {
	SupportContext

	// Limit is the maximum credits allowed in the current window
	Limit int

	// Remaining is credits remaining (usually 0)
	Remaining int

	// ResetAt is when the rate limit window resets
	ResetAt time.Time

	// TroubleshootingGuide is a URL to the relevant troubleshooting documentation
	TroubleshootingGuide string

	// PreFlight is true when the SDK itself rejected the request before
	// sending it, having predicted it would exceed the tracked limit — the
	// client-side reservation working as designed, not a server-reported
	// failure. Callers that log or alert on this error type may want to
	// treat PreFlight rejections as expected throttling rather than a
	// genuine failure.
	PreFlight bool

	Cause error
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("marketdata: rate limit exceeded (limit=%d, remaining=%d, resets=%v)",
		e.Limit, e.Remaining, e.ResetAt.Format(time.RFC3339))
}

func (e *RateLimitError) Unwrap() error   { return e.Cause }
func (e *RateLimitError) Retryable() bool { return false }
func (e *RateLimitError) Is(target error) bool {
	return errors.Is(target, ErrRateLimited)
}

// WaitDuration returns how long to wait before retrying.
func (e *RateLimitError) WaitDuration() time.Duration {
	wait := time.Until(e.ResetAt)
	if wait < 0 {
		return 0
	}
	return wait
}

// InternalError represents a 500 response (internal server error).
// This is a permanent failure — include the Ray ID when opening a support ticket.
type InternalError struct {
	SupportContext
	Cause error
}

func (e *InternalError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("marketdata: internal server error: %s (request_id=%s)", e.Message, e.RequestID)
	}
	return fmt.Sprintf("marketdata: internal server error: %s", e.Message)
}

func (e *InternalError) Unwrap() error   { return e.Cause }
func (e *InternalError) Retryable() bool { return false }
func (e *InternalError) Is(target error) bool {
	return errors.Is(target, ErrInternal)
}

// ServerError represents a temporary server error (501-599).
// These are transient failures that should resolve after a brief wait.
type ServerError struct {
	SupportContext
	Cause error
}

func (e *ServerError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("marketdata: server error %d: %s (request_id=%s)", e.StatusCode, e.Message, e.RequestID)
	}
	return fmt.Sprintf("marketdata: server error %d: %s", e.StatusCode, e.Message)
}

func (e *ServerError) Unwrap() error { return e.Cause }

// Retryable returns true — all 501-599 errors are transient.
func (e *ServerError) Retryable() bool { return true }

func (e *ServerError) Is(target error) bool {
	return errors.Is(target, ErrServer)
}

// NetworkError represents a connection failure or timeout.
type NetworkError struct {
	SupportContext

	// Timeout indicates if this was a timeout error
	Timeout bool

	// Temporary indicates if the error might be transient
	Temporary bool

	Cause error
}

func (e *NetworkError) Error() string {
	if e.Timeout {
		return fmt.Sprintf("marketdata: network timeout: %s", e.Message)
	}
	return fmt.Sprintf("marketdata: network error: %s", e.Message)
}

func (e *NetworkError) Unwrap() error { return e.Cause }

// Retryable returns true for network errors (they are transient).
func (e *NetworkError) Retryable() bool {
	return true
}

// ParseError represents a failed response parse.
type ParseError struct {
	SupportContext
	Cause error
}

func (e *ParseError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("marketdata: parse error: %s (request_id=%s)", e.Message, e.RequestID)
	}
	return fmt.Sprintf("marketdata: parse error: %s", e.Message)
}

func (e *ParseError) Unwrap() error   { return e.Cause }
func (e *ParseError) Retryable() bool { return false }

// APIError represents an unexpected API response where the HTTP status was
// successful (2xx) but the response body indicated an error (e.g. a status
// field other than "ok"). It embeds SupportContext (so Message, RequestID,
// RequestURL, StatusCode, and Timestamp are all populated), implements the
// [Error] interface, and is not retryable: the request already got a
// definitive answer from the API, so resending it would return the same
// anomalous body.
type APIError struct {
	SupportContext
}

func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("marketdata: API error: %s (request_id=%s)", e.Message, e.RequestID)
	}
	return fmt.Sprintf("marketdata: API error: %s", e.Message)
}

func (e *APIError) Unwrap() error   { return nil }
func (e *APIError) Retryable() bool { return false }

// ValidationError represents invalid input parameters.
// This is used for client-side validation before making a request.
type ValidationError struct {
	// Field is the invalid field name
	Field string

	// Message describes what's wrong
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("marketdata: validation error: %s: %s", e.Field, e.Message)
}

func (e *ValidationError) Unwrap() error       { return nil }
func (e *ValidationError) Retryable() bool     { return false }
func (e *ValidationError) SupportInfo() string { return "" }

func (e *ValidationError) Is(target error) bool {
	return errors.Is(target, ErrInvalidRequest)
}
