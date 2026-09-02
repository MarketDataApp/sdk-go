# ADR-009: Logging and Observability

## Status
Accepted

## Context
SDKs need observability for:
- Debugging during development
- Production monitoring
- Support ticket diagnosis

However, logging is contentious in Go libraries because there's no standard logger interface.

## Decision

### Log Format

All SDK log output uses a consistent format:

```
{timestamp} - marketdata.{component} - {level} - {message}
```

Example: `2025-02-21 12:00:00 - marketdata.client - INFO - Making request to /v1/stocks/quotes/`

### Log Levels

Configurable via `MARKETDATA_LOGGING_LEVEL` environment variable (default: `INFO`):

| Level   | What to Log                                         |
|---------|-----------------------------------------------------|
| DEBUG   | Token (redacted), request details, response headers |
| INFO    | Client initialization, base URL, API version        |
| WARNING | Demo mode, deprecated features                      |
| ERROR   | Request failures, rate limit errors                  |

### Required Log Points

- Client initialization (INFO)
- Token value — **redacted** (DEBUG): show only last 4 chars (`****YKT0`)
- Request URL (DEBUG)
- Response status and timing (DEBUG/INFO based on status)
- Errors with support context (ERROR)

### Token Redaction

Never log full tokens. Redact to show only last 4 characters:

```
Token: ************************************YKT0
```

### Debug Mode

The SDK provides a debug mode that sets log level to DEBUG and logs to stderr:

```go
client, _ := marketdata.NewClient(
    marketdata.WithDebug(true),
)
```

### Custom Logger Interface

For production use, clients can provide a custom logger:

```go
type Logger interface {
    Debug(msg string, keysAndValues ...interface{})
    Info(msg string, keysAndValues ...interface{})
    Warn(msg string, keysAndValues ...interface{})
    Error(msg string, keysAndValues ...interface{})
}

client, _ := marketdata.NewClient(
    marketdata.WithLogger(myLogger),
)
```

This interface is compatible with popular loggers (zap, zerolog, slog) via thin adapters.

### Default Behavior

By default, the SDK logs at INFO level to stderr using the standard format. The level is controlled by `MARKETDATA_LOGGING_LEVEL`. Users can suppress all logging by setting the level to a value above ERROR, or provide a custom logger for production integration.

### Observability Hooks (Future)

The design allows for future additions:
- OpenTelemetry tracing integration
- Metrics export (request counts, latencies)
- Custom middleware for request/response inspection

## Consequences

### Positive
- Consistent log format across all SDKs
- Configurable via environment variable
- Debug mode for development
- Custom logger for production integration
- Token redaction prevents credential leakage

### Negative
- Logging at INFO by default may surprise Go library purists
- Logger interface adds abstraction layer
- Default output goes to stderr (not configurable without custom logger)

## Implementation Notes
- Default log level is INFO, controlled by `MARKETDATA_LOGGING_LEVEL` env var
- Debug mode (`WithDebug(true)`) overrides to DEBUG level
- Logger interface uses `...interface{}` for compatibility with most logging libraries
- Auth tokens are always redacted to last 4 characters in all log output
