# ADR-002: Client Design

## Status
Accepted

## Context

The Go SDK v1 used a singleton pattern with global state and auto-initialization via `init()`. This led to:
- Hard to test code
- Surprising behavior (network calls during import)
- Thread-safety concerns with global state
- Inability to use multiple clients with different configurations

We need a client design that is:
- Explicitly constructed (no magic initialization)
- Thread-safe by default
- Testable and mockable
- Configurable via idiomatic Go patterns

## Decision Drivers

- **No Surprises**: No side effects on import
- **Explicit Construction**: Users create clients intentionally
- **Testability**: Easy to mock for unit tests
- **Thread Safety**: Safe for concurrent use
- **Configuration Flexibility**: Support multiple configuration sources

## Considered Options

### Option 1: Global Singleton (v1 approach)
```go
// Auto-initializes from environment
func init() {
    client = newClient()
}

func GetClient() *Client { return client }
```

**Pros**: Convenient, no explicit setup
**Cons**: Untestable, global state, v1 failure mode

### Option 2: Constructor with Options Struct
```go
type Config struct {
    APIKey  string
    Timeout time.Duration
}

func NewClient(cfg Config) (*Client, error)
```

**Pros**: Explicit, clear required/optional fields
**Cons**: Verbose for simple cases, all-or-nothing configuration

### Option 3: Functional Options Pattern
```go
func NewClient(opts ...Option) (*Client, error)

// Options
func WithAPIKey(key string) Option
func WithTimeout(d time.Duration) Option
func WithHTTPClient(c *http.Client) Option
```

**Pros**: Clean API, extensible, optional params without nil/zero confusion
**Cons**: Slightly more complex implementation

## Decision

We adopt **Option 3: Functional Options Pattern** for client construction:

```go
// Client creation
client, err := marketdata.NewClient(
    marketdata.WithToken("your-token"),
    marketdata.WithEnvironment(marketdata.Production),
)

// Or with environment variable fallback
client, err := marketdata.NewClient() // Uses MARKETDATA_TOKEN env var
```

### Client Structure

```go
type Client struct {
    // Configuration (immutable after creation)
    config *Config

    // HTTP client (thread-safe)
    http *http.Client

    // Resources (accessed via client)
    Stocks  *stocks.Service
    Options *options.Service
    Funds   *funds.Service
    Markets *markets.Service
    Utilities *utilities.Service

    // Rate limiting state (thread-safe)
    rateLimit *ratelimit.Tracker
}

// NewClient creates a new MarketData client with the given options
func NewClient(opts ...Option) (*Client, error) {
    cfg := defaultConfig()
    for _, opt := range opts {
        opt.apply(cfg)
    }

    if err := cfg.validate(); err != nil {
        return nil, err
    }

    // Initialize client and resources
    c := &Client{
        config: cfg,
        http:   cfg.httpClient,
    }

    // Initialize resources with reference to client
    c.Stocks = stocks.NewService(c)
    c.Options = options.NewService(c)
    // ... etc

    return c, nil
}
```

### Configuration Options

```go
// Core options
WithToken(token string)          // Set API token directly
WithEnvironment(env Environment) // Production, Test, Development
WithHTTPClient(c *http.Client)   // Custom HTTP client
WithBaseURL(url string)          // Custom base URL (overrides environment)

// Debug options
WithDebug(enabled bool)          // Enable debug logging
WithLogger(logger *slog.Logger)  // Custom structured logger
```

**Note:** Timeout is **not configurable**. The SDK enforces a fixed 99-second request timeout on all HTTP requests, as required by the API specification. If the HTTP client supports a separate connection timeout, it is set to a fixed 2 seconds. Users cannot override these values.

### User-Agent Header

The SDK sets a `User-Agent` header on all outgoing HTTP requests to identify itself to the API:

```
User-Agent: marketdata-sdk-go/{version}
```

The version is automatically detected from Go module metadata at runtime (see ADR-015). This header is always set and cannot be disabled.

### Resource Cleanup

The client provides an explicit `Close()` method for releasing resources (HTTP connections, background goroutines):

```go
// Close releases resources held by the client.
// After Close is called, the client must not be reused.
func (c *Client) Close() error {
    c.http.CloseIdleConnections()
    return nil
}
```

Users should call `Close()` when the client is no longer needed, typically via `defer`:

```go
client, err := marketdata.NewClient()
if err != nil { log.Fatal(err) }
defer client.Close()
```

`Close()` is safe to call multiple times and is a no-op after the first call.

### Environment Variable Fallbacks

```go
// Default configuration loads from environment
func defaultConfig() *Config {
    return &Config{
        token:       getTokenFromEnv(), // Checks MARKETDATA_TOKEN
        environment: Production,
        baseURL:     baseURLs[Production],
        apiVersion:  "v1",
        logger:      slog.Default(),
    }
}
```

## Consequences

### Positive
- No global state or singleton pattern
- Explicit, testable client construction
- Flexible configuration with sensible defaults
- Environment variable support without magic
- Multiple clients can coexist with different configs
- Thread-safe by design

### Negative
- Slightly more verbose than singleton
- Users must manage client lifecycle
- Options pattern requires more initial code

### Mitigations
- Provide good defaults (env var fallback)
- Clear documentation and examples
- Helper methods for common configurations

## References

- [Functional Options Pattern - Dave Cheney](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis)
- [AWS SDK for Go v2 Config](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/config)
- [Stripe Go SDK Client](https://github.com/stripe/stripe-go)
