package marketdata_test

// The Example* functions below are compile-checked documentation, not
// executed tests: none has a "// Output:" comment (T-5), which is
// deliberate, not an oversight. Each builds its client with
// marketdata.NewClient() and no token, so the calls below would still
// reach the live API over the network. Giving them Output assertions would
// make this package's ordinary `go test` (meant to be hermetic and
// network-free, per ADR-010) depend on live network access and produce
// non-deterministic output. `go test` still compiles every example here,
// so a signature change breaks the build immediately; live,
// output-verified counterparts live in integration/example_test.go, gated
// by the `integration` build tag.
//
// Compile-checking does not check what an example CLAIMS, though: the
// rate-limit example below described a per-request credit cost as a window
// total for months, and executing it would not have caught that either.
// Prose in an example is documentation and needs reading, not running.

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
)

// This example shows the shortest path to market data: with an API token in
// the MARKETDATA_TOKEN environment variable, three lines create a client
// and fetch a stock quote.
func Example() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	quote, err := client.Stocks.GetQuote("AAPL")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("AAPL last: $%.2f\n", quote.Last)
}

// This example creates a client with an explicit token instead of the
// environment variable and raises the retry budget from the default of 3.
func ExampleNewClient() {
	client, err := marketdata.NewClient(
		marketdata.WithToken("your-token"),
		marketdata.WithMaxRetries(5),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	quote, _, err := client.Stocks.Quote(context.Background(), "SPY")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("SPY last: $%.2f\n", quote.Last)
}

// This example reads the client-level rate limit snapshot. The snapshot
// reflects the most recently completed request; for exact per-request
// values, use the RateLimit field of the Response returned by each
// context-first method.
func ExampleClient_RateLimits() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	state := client.RateLimits()
	// Consumed is the cost of the most recent request, not a window total;
	// the window total is Limit - Remaining.
	fmt.Printf("last request cost %d credit(s); %d of %d used this window, resets at %s\n",
		state.Consumed, state.Limit-state.Remaining, state.Limit, state.ResetAt.Format(time.RFC3339))
	if state.Remaining < 100 {
		fmt.Println("running low on API credits")
	}
}

// This example distinguishes failures with the typed error hierarchy:
// errors.Is against a sentinel when only the category matters, and
// errors.As against a concrete type when the error's fields are needed.
// SupportInfo formats the request details for a support ticket.
func Example_errorHandling() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	_, _, err = client.Stocks.Quote(context.Background(), "AAPL")
	if err != nil {
		// Sentinel check: is this any kind of not-found error?
		if errors.Is(err, marketdata.ErrNotFound) {
			fmt.Println("symbol not found")
			return
		}

		// Typed check: pull the reset time and support details off the error.
		var rateErr *marketdata.RateLimitError
		if errors.As(err, &rateErr) {
			fmt.Printf("rate limited; safe to retry in %s\n", rateErr.WaitDuration())
			fmt.Println(rateErr.SupportInfo())
			return
		}

		log.Fatal(err)
	}
}
