# ADR-010: Testing Strategy

## Status
Accepted

## Context
The SDK requires multiple testing approaches:
- Unit tests for individual components
- Integration tests for HTTP handling
- Contract tests for API compatibility
- End-to-end tests against the live API

## Decision

### Unit Testing
Standard Go testing with table-driven tests:

```go
func TestQuote_Spread(t *testing.T) {
    tests := []struct {
        name string
        bid  float64
        ask  float64
        want float64
    }{
        {"normal spread", 100.00, 100.05, 0.05},
        {"zero spread", 100.00, 100.00, 0.00},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            q := &Quote{Bid: tt.bid, Ask: tt.ask}
            if got := q.Spread(); got != tt.want {
                t.Errorf("Spread() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### HTTP Mock Testing
Use `httptest.Server` for testing HTTP interactions:

```go
func TestService_Quote(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "s": "ok",
            "symbol": []string{"AAPL"},
            "last":   []float64{150.00},
        })
    }))
    defer server.Close()

    client, _ := marketdata.NewClient(
        marketdata.WithBaseURL(server.URL),
        marketdata.WithToken("test-key"),
    )

    quote, err := client.Stocks.Quote(context.Background(), "AAPL")
    // assertions...
}
```

### Interface-Based Mocking
Services depend on an HTTP client interface for testability:

```go
type HTTPClient interface {
    Get(ctx context.Context, path string, params url.Values, result interface{}) (*http.Response, error)
}
```

This allows injecting mock clients in tests without network calls.

### Test Fixtures
JSON response fixtures stored in `testdata/` directories:

```
marketdata/stocks/
├── service.go
├── service_test.go
└── testdata/
    ├── quote_single.json
    ├── quote_multiple.json
    └── candles_daily.json
```

### Integration Tests
Tests against httptest.Server validate:
- Request serialization
- Response deserialization
- Error handling
- Retry behavior
- Rate limit tracking

### End-to-End Tests (Optional)
Live API tests gated by environment variable:

```go
func TestLive_Quote(t *testing.T) {
    if os.Getenv("MARKETDATA_LIVE_TESTS") != "1" {
        t.Skip("Skipping live API test")
    }
    // Test against real API
}
```

### Coverage Goals
- **100% code coverage required**
- Lines that cannot be tested must be explicitly marked with coverage ignore comments (build tags or explicit skip) and include a comment explaining why
- Integration tests: All HTTP paths covered
- CI must produce coverage reports and fail if coverage is not 100%

### Code Style and Linting

The SDK enforces Go code style standards as part of CI:

**Required linter:** `golangci-lint` with the following enabled linters:
- `gofmt` — Standard Go formatting
- `govet` — Suspicious constructs
- `errcheck` — Unchecked errors
- `staticcheck` — Static analysis
- `unused` — Unused code
- `ineffassign` — Ineffective assignments
- `misspell` — Common misspellings

**Style guide:** Code must follow [Effective Go](https://go.dev/doc/effective_go) and [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).

**CI enforcement:**
- `golangci-lint run` must pass with zero findings on every PR
- `go fmt ./...` must produce no changes (enforced via CI check)
- `go vet ./...` must pass

**Configuration** is maintained in `.golangci.yml` at the repository root.

## Consequences

### Positive
- Fast unit tests (no network)
- Reliable integration tests (deterministic responses)
- Optional live testing for confidence
- Easy to mock for user testing

### Negative
- Test fixtures can drift from actual API
- httptest.Server tests are more complex than pure unit tests
- Live tests require API key management

## Implementation Notes
- Use `t.Parallel()` for independent tests
- Golden file testing for response parsing validation
- Benchmark tests for performance-critical paths
- **Unit tests** run on every PR and push to main, across multiple Go versions
- **Integration tests** run on every PR and in release pipelines; excluded from per-commit/push jobs and default local test runs
- Integration tests gated by environment variable (e.g., `MARKETDATA_TOKEN`), skip gracefully if not set
