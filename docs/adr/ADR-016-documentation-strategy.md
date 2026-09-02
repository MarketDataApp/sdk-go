# ADR-016: Documentation Strategy

## Status
Accepted

## Context

The SDK requirements (§14) define specific documentation deliverables: a README with quick-start content, complete API documentation for all public methods, usage examples, and error handling guidance. This ADR specifies how documentation is structured and maintained for the Go SDK.

## Decision

### README Structure

The repository `README.md` must include the following sections in order:

1. **Title and badges** — Package name, Go version, CI status, coverage, Go Reference link
2. **Installation** — `go get` command
3. **Quick Start** — Minimal working example (auth + first request):
   ```go
   client, err := marketdata.NewClient()
   if err != nil { log.Fatal(err) }
   defer client.Close()

   quote, err := client.Stocks.Quotes(ctx, []string{"AAPL"})
   fmt.Println(quote)
   ```
4. **Environment Variable Configuration** — Table of supported env vars with defaults
5. **Error Handling** — Example showing typed error checks with `errors.As`:
   ```go
   var rateLimitErr *marketdata.RateLimitError
   if errors.As(err, &rateLimitErr) {
       fmt.Println(rateLimitErr.SupportInfo())
   }
   ```
6. **String Conversion** — Example showing `String()` / `fmt.Println` on response objects
7. **Resources** — Brief table of available resources and methods
8. **Link to full documentation** — Points to pkg.go.dev

### API Documentation (Go Doc)

All public types, methods, and functions use Go doc comments following [Effective Go](https://go.dev/doc/effective_go#commentary) and [Go Doc Comments](https://go.dev/doc/comment) conventions:

- Every exported symbol has a doc comment
- Doc comments start with the symbol name: `// Quote returns the latest quote for a symbol.`
- Method docs include an `Example` section using Go's testable examples:
  ```go
  func ExampleService_Quote() {
      client, _ := marketdata.NewClient()
      defer client.Close()
      quote, _ := client.Stocks.Quote(context.Background(), "AAPL")
      fmt.Println(quote)
      // Output: ...
  }
  ```
- Parameter types and constraints documented in method comments
- Error conditions documented (which error types can be returned and when)

Documentation is published automatically via [pkg.go.dev](https://pkg.go.dev) when module versions are tagged and released.

### Examples Directory

The `examples/` directory contains runnable example programs organized by use case:

```
examples/
├── basic/          # Client setup, simple quote
├── candles/        # Fetching candle data with options
├── options/        # Options chain, expirations, quotes
├── error_handling/ # Typed error handling, support info
└── concurrent/     # Batch requests, date-range splitting
```

Each example is a standalone `main.go` that can be run with:
```bash
MARKETDATA_TOKEN=your-token go run ./examples/basic/
```

### Error Type Documentation

Error types are documented in `marketdata/errors.go` with:
- When each error type is returned (which HTTP status codes trigger it)
- Which fields are populated
- How to use `errors.Is` / `errors.As` to check
- The `SupportInfo()` output format

### String Conversion Documentation

All response models that implement `String()` include doc comments showing the output format, and at least one testable example demonstrating `fmt.Println(resp)` usage.

### Documentation Maintenance

- Doc comments are reviewed as part of code review for any public API change
- Examples in `examples/` are compiled by CI (they are real Go packages) ensuring they never go stale
- Testable examples (`Example*` functions) are run as part of `go test` and verified

## Consequences

### Positive
- README provides immediate onboarding for new users
- pkg.go.dev publishes API docs automatically from source
- Testable examples guarantee docs stay in sync with code
- Examples directory provides copy-paste starting points

### Negative
- Doc comments add maintenance burden to every public API change
- Testable examples must produce deterministic output (or use `// Output:` without expected value)
- README must be kept in sync with API changes manually

### Mitigations
- CI compiles all examples, catching stale code
- Testable examples run in `go test`, catching doc drift
- README sections are minimal and point to pkg.go.dev for details

## References

- Requirements: Section 14 (Documentation Requirements)
- [Go Doc Comments](https://go.dev/doc/comment)
- [Testable Examples in Go](https://go.dev/blog/examples)
- [pkg.go.dev](https://pkg.go.dev)
