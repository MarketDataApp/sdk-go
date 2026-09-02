# ADR-012: Configuration Cascade

## Status
Accepted

## Context

The SDK needs a predictable configuration system where more specific settings override less specific ones. The requirements (req 4) define a 4-tier cascade: `.env` file, environment variables, client defaults, and method parameters. Go does not have a built-in `.env` loader, so we need to decide how to handle that tier.

## Decision

### Priority Order (Lowest to Highest)

| Priority    | Source                | Scope             | Go Mechanism                                       |
|-------------|-----------------------|-------------------|----------------------------------------------------|
| 1 (lowest)  | `.env` file           | Project-level     | Parsed at `NewClient()` time from working directory |
| 2           | Environment variables | System/shell      | `os.Getenv()` / `os.LookupEnv()`                  |
| 3           | Client constructor    | Per-client        | Functional options: `WithToken()`, etc.            |
| 4 (highest) | Method parameters     | Per-request       | Functional options on service methods              |

### .env File Loading

`NewClient()` automatically loads a `.env` file from the current working directory if one exists. Values from `.env` are loaded into the process environment **only if the variable is not already set**, preserving the cascade:

```go
// .env file in working directory:
// MARKETDATA_TOKEN=token-from-dotenv
// MARKETDATA_BASE_URL=https://custom.api.example.com

// Shell: export MARKETDATA_TOKEN=token-from-shell

client, err := marketdata.NewClient()
// Token = "token-from-shell" (env var overrides .env)
// Base URL = "https://custom.api.example.com" (from .env, no shell override)
```

The `.env` loader:
- Reads from the current working directory only (no recursive parent search)
- Ignores missing `.env` files silently (no error)
- Uses `KEY=VALUE` format, one per line
- Supports `#` comments and blank lines
- Does **not** overwrite existing environment variables
- Is implemented in `internal/dotenv/` with no external dependencies

`.env` loading can be disabled for environments where it is not desired:

```go
client, err := marketdata.NewClient(
    marketdata.WithoutDotEnv(),
)
```

### Client-Level Defaults

Client constructor options set defaults for all requests made through that client:

```go
client, err := marketdata.NewClient(
    marketdata.WithToken("my-token"),          // Overrides env var
    marketdata.WithBaseURL("https://..."),     // Overrides env var
    marketdata.WithDefaultDateFormat("unix"),  // Default for all requests
)
```

### Method-Level Overrides

Per-request functional options override client defaults:

```go
// Uses client default date format
quote, err := client.Stocks.Quote(ctx, "AAPL")

// Overrides to unix for this request only
quote, err := client.Stocks.Quote(ctx, "AAPL",
    stocks.WithDateFormat("unix"),
)
```

### Supported Environment Variables

| Variable                        | Purpose                    | Default                        |
|---------------------------------|----------------------------|--------------------------------|
| `MARKETDATA_TOKEN`              | API authentication token   | (none)                         |
| `MARKETDATA_BASE_URL`           | API base URL               | `https://api.marketdata.app`   |
| `MARKETDATA_API_VERSION`        | API version                | `v1`                           |
| `MARKETDATA_LOGGING_LEVEL`      | SDK log level              | `INFO`                         |
| `MARKETDATA_OUTPUT_FORMAT`      | Default output format      | (language default — JSON)      |
| `MARKETDATA_DATE_FORMAT`        | Default date format        | `timestamp`                    |
| `MARKETDATA_COLUMNS`            | Columns to include         | (all)                          |
| `MARKETDATA_ADD_HEADERS`        | Include headers in CSV     | `true`                         |
| `MARKETDATA_USE_HUMAN_READABLE` | Human-readable field names | `false`                        |
| `MARKETDATA_MODE`               | Data mode                  | `live`                         |

### Merge Behavior

When building an HTTP request, configuration is merged in cascade order:

```go
func (s *Service) buildRequest(ctx context.Context, opts ...Option) *http.Request {
    // Start with client-level defaults (already merged from .env + env vars + constructor)
    params := s.client.defaultParams.Clone()

    // Apply per-request overrides
    for _, opt := range opts {
        opt.apply(params)
    }

    // Build request with final params
    ...
}
```

## Consequences

### Positive
- Predictable override behavior — more specific always wins
- `.env` support enables per-project configuration without shell setup
- Client defaults reduce repetition across requests
- Per-request overrides enable flexibility without client reconfiguration

### Negative
- `.env` loading adds I/O during client construction
- 4-tier cascade increases configuration debugging complexity
- `.env` parsing must be maintained without external dependencies

### Mitigations
- `.env` loading is fast (single file read) and skipped silently if missing
- `WithoutDotEnv()` opt-out for controlled environments
- Debug logging shows resolved configuration at startup

## References

- Requirements: Section 4 (Configuration Cascade)
- Related: ADR-002 (client design, functional options), ADR-011 (token resolution)
