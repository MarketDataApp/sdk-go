# ADR-003: Error Handling

## Status
Accepted

## Context

Go SDK v1 had inconsistent error handling - some methods returned errors, others set error fields on structs. Users couldn't distinguish between retryable and permanent errors, and error messages lacked context.

We need an error handling strategy that:
- Follows Go idioms (`return value, error`)
- Distinguishes error types (retryable vs permanent)
- Works with `errors.Is` and `errors.As`
- Provides actionable context
- Enables automatic retry decisions

## Decision Drivers

- **Idiomatic Go**: Use standard error patterns
- **Error Inspection**: Support errors.Is and errors.As
- **Actionable**: Include request ID, URL, timestamps
- **Retryable Classification**: SDK can auto-retry appropriately
- **User-Friendly**: Clear messages for debugging

## Considered Options

### Option 1: Simple Error Strings
```go
return nil, errors.New("request failed: 429 rate limited")
```

**Pros**: Simple
**Cons**: No type checking, no structured info

### Option 2: Sentinel Errors
```go
var ErrRateLimited = errors.New("rate limited")
if errors.Is(err, ErrRateLimited) { ... }
```

**Pros**: Type checking via errors.Is
**Cons**: Limited context, can't carry request-specific data

### Option 3: Custom Error Types with Wrapping
```go
type APIError struct {
    Code       string
    Message    string
    StatusCode int
    RequestID  string
    Retryable  bool
    cause      error
}

func (e *APIError) Error() string { ... }
func (e *APIError) Unwrap() error { return e.cause }
```

**Pros**: Rich context, type checking, wrapping support
**Cons**: More complex implementation

## Decision

We adopt **Option 3: Custom Error Types** with a hierarchy:

### Error Type Hierarchy

```go
// Base error interface for all SDK errors
type Error interface {
    error
    Unwrap() error

    // Retryable returns true if the error is transient
    Retryable() bool

    // SupportInfo returns a formatted string for support tickets
    SupportInfo() string
}

// SupportContext contains fields included in every API error for support troubleshooting
type SupportContext struct {
    RequestID     string    // cf-ray response header
    RequestURL    string    // Full request URL
    StatusCode    int       // HTTP status code
    Timestamp     time.Time // SDK-generated, US/Eastern
    Message       string    // Error description
    ExceptionType string    // Error type name (e.g., "RateLimitError")
}

// SupportInfo returns a formatted support string
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

// AuthenticationError represents 401 responses, invalid/missing token
type AuthenticationError struct {
    SupportContext
    cause error
}

// BadRequestError represents 400 responses, invalid parameters
type BadRequestError struct {
    SupportContext
    cause error
}

// NotFoundError represents 404 responses
type NotFoundError struct {
    SupportContext
    cause error
}

// RateLimitError represents 429 responses, rate limit exceeded
type RateLimitError struct {
    SupportContext
    Limit     int       // Request limit
    Remaining int       // Requests remaining
    ResetAt   time.Time // When limit resets
    cause     error
}

// ServerError represents 5xx responses
type ServerError struct {
    SupportContext
    cause error
}

// NetworkError represents connection failures, timeouts
type NetworkError struct {
    SupportContext
    Timeout   bool
    Temporary bool
    cause     error
}

// ParseError represents failed response parsing
type ParseError struct {
    SupportContext
    cause error
}
```

### Error Classification

| Error Type | Status Codes | Retryable | Action |
|-----------|--------------|-----------|--------|
| AuthenticationError | 401 | No, fail immediately | Fix token |
| BadRequestError | 400 | No | Fix parameters |
| NotFoundError | 404 | No | See note below |
| RateLimitError | 429 | No, expose retry-after | Wait for reset |
| ServerError | 500 | No | Report to support |
| ServerError | 501-599 | Yes | Auto-retry with backoff |
| NetworkError (timeout) | - | Yes | Auto-retry |
| NetworkError (other) | - | Yes | Auto-retry |
| ParseError | - | No | Report to support |

**404 Handling:** Most endpoints return 404 when no data exists (e.g., no quotes for a delisted symbol). SDKs should return an empty result object with `NoData: true` rather than returning an error. `NotFoundError` is reserved for cases where the endpoint itself is invalid.

### Usage Examples

```go
// Making a request
quote, err := client.Stocks.Quote(ctx, "AAPL")
if err != nil {
    // Check for specific error types
    var authErr *marketdata.AuthenticationError
    if errors.As(err, &authErr) {
        fmt.Println("Invalid token, failing immediately")
        fmt.Println(authErr.SupportInfo()) // Formatted support string
        return nil, err
    }

    var rateLimitErr *marketdata.RateLimitError
    if errors.As(err, &rateLimitErr) {
        fmt.Printf("Rate limited, resets at: %v\n", rateLimitErr.ResetAt)
        fmt.Println(rateLimitErr.SupportInfo())
    }

    // All SDK errors support SupportInfo()
    var sdkErr marketdata.Error
    if errors.As(err, &sdkErr) {
        fmt.Println(sdkErr.SupportInfo())
    }

    return nil, err
}
```

### Sentinel Errors for Common Cases

```go
// For common error checks without type assertion
var (
    ErrAuthentication = errors.New("authentication failed")
    ErrBadRequest     = errors.New("bad request")
    ErrNotFound       = errors.New("not found")
    ErrRateLimited    = errors.New("rate limit exceeded")
    ErrServer         = errors.New("server error")
)

// Each error type implements Is for sentinel comparison
// Usage
if errors.Is(err, marketdata.ErrAuthentication) {
    // Handle auth error - fail immediately, do not retry
}
if errors.Is(err, marketdata.ErrRateLimited) {
    // Handle rate limit - check ResetAt for when to retry
}
```

### Support Info Helper

Every error exposes a `SupportInfo()` method that returns a formatted string for support tickets:

```go
err := client.Stocks.Quote(ctx, "AAPL")
var sdkErr marketdata.Error
if errors.As(err, &sdkErr) {
    fmt.Println(sdkErr.SupportInfo())
    // Output:
    // --- MARKET DATA SUPPORT INFO ---
    // request_id:     8a1b2c3d4e5f6g7h-SJC
    // request_url:    https://api.marketdata.app/v1/stocks/quotes/
    // status_code:    429
    // timestamp:      2025-02-21 12:00:00
    // message:        Rate limit exceeded
    // exception_type: RateLimitError
    // --------------------------------
}
```

## Consequences

### Positive
- Clear error classification
- Rich context for debugging
- Support for errors.Is and errors.As
- Automatic retry decisions
- Consistent error handling across SDK

### Negative
- More types to maintain
- Users need to learn error hierarchy
- Error wrapping adds complexity

### Mitigations
- Comprehensive documentation
- Examples for common error handling patterns
- Helper functions for common checks

## References

- [Go Error Handling Best Practices - Datadog](https://www.datadoghq.com/blog/go-error-handling/)
- [Working with Errors in Go 1.13+](https://go.dev/blog/go1.13-errors)
- [Stripe Go SDK Errors](https://github.com/stripe/stripe-go)
- [AWS SDK Error Handling](https://aws.github.io/aws-sdk-go-v2/docs/handling-errors/)
