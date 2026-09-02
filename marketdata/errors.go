package marketdata

import (
	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

// Sentinel errors for quick classification with errors.Is. Each sentinel
// matches the corresponding typed error, so
// errors.Is(err, marketdata.ErrRateLimited) is true whenever err wraps a
// [RateLimitError]. Use errors.As with the typed errors instead when the
// error's fields (such as reset times or support context) are needed.
var (
	// ErrAuthentication matches [AuthenticationError]: the API token is
	// invalid or missing (HTTP 401).
	ErrAuthentication = sdkerrors.ErrAuthentication

	// ErrPaymentRequired matches [PaymentRequiredError]: the request needs
	// a higher plan (HTTP 402).
	ErrPaymentRequired = sdkerrors.ErrPaymentRequired

	// ErrForbidden matches [ForbiddenError]: access was denied due to an IP
	// policy violation (HTTP 403).
	ErrForbidden = sdkerrors.ErrForbidden

	// ErrBadRequest matches [BadRequestError]: the request parameters were
	// invalid (HTTP 400).
	ErrBadRequest = sdkerrors.ErrBadRequest

	// ErrNotFound matches [NotFoundError]. In practice this SDK never
	// returns it: every HTTP 404 is treated as "no data" and reported
	// through the NoData field of the returned [Response], never as an
	// error. See [NotFoundError] for why the type exists anyway.
	ErrNotFound = sdkerrors.ErrNotFound

	// ErrPayloadTooLarge matches [PayloadTooLargeError]: the request spans
	// too much data (HTTP 413).
	ErrPayloadTooLarge = sdkerrors.ErrPayloadTooLarge

	// ErrRateLimited matches [RateLimitError]: the rate limit has been
	// exceeded (HTTP 429).
	ErrRateLimited = sdkerrors.ErrRateLimited

	// ErrResponseTooLarge matches [ResponseTooLargeError]: the response body
	// exceeded the SDK's safety cap and was refused before being buffered.
	ErrResponseTooLarge = sdkerrors.ErrResponseTooLarge

	// ErrInsecureToken matches [InsecureTokenError]: the SDK refused to send
	// the API token over a connection that is neither HTTPS nor loopback.
	ErrInsecureToken = sdkerrors.ErrInsecureToken

	// ErrInternal matches [InternalError]: an internal server error
	// (HTTP 500). Not retried by the SDK.
	ErrInternal = sdkerrors.ErrInternal

	// ErrServer matches [ServerError]: a temporary server error
	// (HTTP 501-599). Retried automatically by the SDK.
	ErrServer = sdkerrors.ErrServer

	// ErrInvalidRequest matches [ValidationError]: the SDK rejected the
	// input client-side before any request was made.
	ErrInvalidRequest = sdkerrors.ErrInvalidRequest
)

// Error is the interface implemented by all SDK errors. Beyond the
// standard error and Unwrap methods, it reports whether the failed
// operation is safe to retry and exposes SupportInfo, which formats the
// request details for a Market Data support ticket.
type Error = sdkerrors.Error

// SupportContext carries the request details embedded in every API error:
// the request ID (the cf-ray header), request URL, HTTP status code,
// timestamp, message, and exception type. Its SupportInfo method formats
// these fields as a ready-to-paste block for Market Data support tickets,
// so any API error can produce one directly:
//
//	var apiErr *marketdata.AuthenticationError
//	if errors.As(err, &apiErr) {
//		fmt.Println(apiErr.SupportInfo())
//	}
type SupportContext = sdkerrors.SupportContext

// AuthenticationError is returned for an HTTP 401 response: the API token
// is invalid, expired, or missing. It embeds [SupportContext], matches
// [ErrAuthentication] with errors.Is, and is not retryable.
type AuthenticationError = sdkerrors.AuthenticationError

// PaymentRequiredError is returned for an HTTP 402 response: the request
// was valid but the account's plan does not include the requested feature
// or data. It embeds [SupportContext], matches [ErrPaymentRequired] with
// errors.Is, and is not retryable.
type PaymentRequiredError = sdkerrors.PaymentRequiredError

// ForbiddenError is returned for an HTTP 403 response, which typically
// occurs when the account's IP address changes and access is temporarily
// blocked. The AuthorizedIP and BlockedIP fields identify the addresses
// involved and TroubleshootingGuide links to the relevant documentation.
// It embeds [SupportContext], matches [ErrForbidden] with errors.Is, and
// is not retryable.
type ForbiddenError = sdkerrors.ForbiddenError

// BadRequestError is returned for an HTTP 400 response: the request
// parameters were invalid. It embeds [SupportContext], matches
// [ErrBadRequest] with errors.Is, and is not retryable.
type BadRequestError = sdkerrors.BadRequestError

// NotFoundError represents an HTTP 404 response in the cross-SDK error
// taxonomy (SDK requirements §6.1), which every Market Data SDK defines.
// In practice this SDK never returns it: the API answers 404 for "no data
// matched the request" — including unknown symbols — and the SDK reports
// that through the NoData field of the returned [Response] with a nil
// error, never as an error value. Branching on [ErrNotFound] therefore
// never fires; check Response.NoData (or a nil result from the Get*
// convenience methods) instead. NotFoundError embeds [SupportContext],
// matches [ErrNotFound] with errors.Is, and is not retryable.
type NotFoundError = sdkerrors.NotFoundError

// PayloadTooLargeError is returned for an HTTP 413 response: the request
// spans too much data, typically an intraday candle request covering more
// than one year. It embeds [SupportContext], matches [ErrPayloadTooLarge]
// with errors.Is, and is not retryable.
type PayloadTooLargeError = sdkerrors.PayloadTooLargeError

// RateLimitError is returned for an HTTP 429 response: the account's rate
// limit has been exceeded. The Limit, Remaining, and ResetAt fields
// describe the current window, and WaitDuration reports how long to wait
// before trying again. It embeds [SupportContext], matches [ErrRateLimited]
// with errors.Is, and is not retried automatically by the SDK.
type RateLimitError = sdkerrors.RateLimitError

// InternalError is returned for an HTTP 500 response: a permanent server
// failure that the SDK does not retry. Include the request ID (via
// SupportInfo) when opening a support ticket. It embeds [SupportContext]
// and matches [ErrInternal] with errors.Is.
type InternalError = sdkerrors.InternalError

// ServerError is returned for an HTTP 501-599 response: a temporary server
// failure. These are the only status codes the SDK retries automatically
// with exponential backoff, so a ServerError surfaces only after the retry
// budget is exhausted. It embeds [SupportContext] and matches [ErrServer]
// with errors.Is.
type ServerError = sdkerrors.ServerError

// NetworkError represents a connection failure or timeout. It covers both
// failures before an HTTP response was received — where StatusCode is 0 —
// and a body that failed or timed out mid-read, where StatusCode is taken
// from the interrupted response. The Timeout and Temporary fields classify
// the failure, and Retryable reports true because network errors are
// transient; the SDK retries them automatically.
type NetworkError = sdkerrors.NetworkError

// ParseError is returned when an API response is received but its body
// cannot be decoded into the expected type. It embeds [SupportContext] and
// is not retryable.
type ParseError = sdkerrors.ParseError

// ResponseTooLargeError is returned when an API response body exceeds the
// SDK's size cap. The body is refused rather than buffered, so a hostile or
// malfunctioning server cannot exhaust the caller's memory. It embeds
// [SupportContext] and is not retryable.
type ResponseTooLargeError = sdkerrors.ResponseTooLargeError

// InsecureTokenError is returned when the SDK refuses to transmit the API
// token over a connection that is neither HTTPS nor a loopback host. It is
// raised before the request is sent, so the token never leaves the process.
type InsecureTokenError = sdkerrors.InsecureTokenError

// APIError represents an unexpected API response where the HTTP status
// reported success but the response body indicated an error (for example,
// a status field other than "ok").
type APIError = sdkerrors.APIError

// ValidationError is returned when the SDK rejects input client-side, such
// as an empty symbol or a malformed base URL, before any request is made.
// The Field and Message fields identify the offending parameter. It
// matches [ErrInvalidRequest] with errors.Is, is not retryable, and, since
// no request occurred, its SupportInfo returns an empty string.
type ValidationError = sdkerrors.ValidationError
