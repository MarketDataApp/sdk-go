# ADR-011: Authentication and Security

## Status
Accepted

## Context

The SDK must handle authentication securely, including token validation, demo mode for unauthenticated access, and TLS enforcement. These security requirements (req 5, req 16) ensure the SDK fails fast on bad credentials, supports serverless/CLI environments where startup validation is costly, and never weakens transport security.

## Decision

### Token Resolution

Token resolution follows the configuration cascade (see ADR-012):

1. `.env` file in working directory (lowest priority)
2. `MARKETDATA_TOKEN` environment variable
3. `WithToken()` client constructor option (highest priority)

All requests include the `Authorization: Bearer {token}` header.

### Startup Token Validation

By default, `NewClient()` validates the token by calling `GET /user/` during initialization. This fails fast on invalid or expired tokens:

```go
// Default: validates token on startup
client, err := marketdata.NewClient(
    marketdata.WithToken("your-token"),
)
// err is non-nil if token is invalid
```

For short-lived runtimes (serverless functions, CLI tools) where the `/user/` call adds unacceptable latency, startup validation can be disabled:

```go
// Skip startup validation — errors surface on first request
client, err := marketdata.NewClient(
    marketdata.WithToken("your-token"),
    marketdata.WithoutStartupValidation(),
)
```

When startup validation is disabled, authentication errors surface on the first authenticated request as `*AuthenticationError`.

### Demo Mode

When no token is provided (no `.env`, no environment variable, no constructor option):

1. Omit the `Authorization` header from all requests
2. Log a warning at `WARN` level: `"No API token provided — running in demo mode with limited access"`
3. Skip rate limit initialization (no `/user/` call)
4. Requests proceed without authentication, subject to API demo-mode restrictions

```go
// Demo mode — no token
client, err := marketdata.NewClient()
// err is nil, but client operates in demo mode
```

### Token Redaction

Tokens must never appear in logs, error messages, or debug output. All logging redacts tokens to show only the last 4 characters (see ADR-009):

```
Token: ************************************YKT0
```

Error messages containing URLs must strip any token query parameters before surfacing to users.

### TLS Security

The SDK enforces strict TLS validation on all connections:

- **TLS certificate validation is always enabled** — the SDK never sets `InsecureSkipVerify: true`
- The `WithHTTPClient()` option accepts a custom `*http.Client`, but the SDK does not provide any option or mechanism to disable TLS verification
- If a user provides a custom `http.Client` with `InsecureSkipVerify: true`, that is their responsibility — the SDK itself never configures this
- All default transports use the system certificate pool

### Error Messages

Error messages must not contain sensitive data:

- No tokens in error strings
- No tokens in `SupportInfo()` output
- Request URLs in errors must have tokens redacted

### Hardening against untrusted input and hostile responses (2026-07-11)

The threat model is a published SDK embedded in third-party Go apps that feed it
untrusted symbols and parameters. The following defenses were added and are
proven by tests in `internal/http` (`security_test.go`, `untrusted_input_test.go`,
`path_test.go`):

- **Token never shipped in cleartext.** The bearer token is attached only when
  the request scheme is HTTPS, or the host is loopback (local development). A base
  URL pointed at a plain-HTTP, non-loopback host returns an `*InsecureTokenError`
  before any request is sent, so the token cannot be exfiltrated to an unintended
  origin. (`tokenSafeForURL`.)
- **Token cannot follow a cross-host redirect.** A secure `CheckRedirect` caps the
  redirect chain and refuses any redirect that changes the host, making explicit
  (and testable) the guarantee that the token never reaches another origin.
- **Response-side DoS bound.** Response bodies are read through an
  `io.LimitReader` capped at 100 MiB by default (configurable); an oversized or
  hostile body yields a `*ResponseTooLargeError` instead of exhausting memory. The
  fixed 99s request timeout bounds slow responses.
- **Untrusted symbols are safely encoded.** Caller-supplied symbols are
  percent-encoded as single path segments (`PathSegment`), with dot-segments
  neutralized. End-to-end tests drive CRLF/header-injection, path-traversal,
  query-smuggling, NUL, Unicode/RTL, pre-encoded, and oversized inputs and assert
  no control characters, no smuggled query, no added path segments, and no injected
  headers reach the wire. Server-supplied error strings are control-char-sanitized
  and length-capped before entering logs/errors.
- **Token-leak scanning.** A test drives success and error paths with debug
  logging on, then scans all captured output, error strings, and `SupportInfo()`
  for the token value — zero hits. The token lives only in the `Authorization`
  header, never in URLs, query strings, logs, or errors.
- **Vulnerability scanning.** `govulncheck ./...` is clean (the module pins a
  patched Go toolchain; SDK code has zero reachable vulnerabilities and zero
  third-party runtime dependencies). `staticcheck ./...` is clean. Both run in CI.

## Consequences

### Positive
- Untrusted input cannot inject headers, escape the path, smuggle a query, hang,
  OOM, or leak the token.
- Fail-fast by default catches invalid tokens immediately
- Serverless/CLI environments can opt out of startup cost
- Demo mode enables exploration without credentials
- TLS always enforced — no accidental insecure connections
- Token redaction prevents credential leakage in logs and errors

### Negative
- Default startup validation adds one HTTP round-trip to client creation
- Demo mode may confuse users who forget to set their token
- No way to programmatically disable TLS (intentional)

### Mitigations
- `WithoutStartupValidation()` for latency-sensitive environments
- Demo mode logs a clear warning
- Documentation emphasizes token setup in quick-start guide

## References

- Requirements: Section 5 (Authentication), Section 16 (Security)
- Related: ADR-009 (token redaction in logging), ADR-012 (configuration cascade)
