// Example: Basic usage of the MarketData Go SDK v2
//
// This example demonstrates both the simple convenience API and the
// full context-aware API for fetching market data.
//
// To run:
//
//	export MARKETDATA_TOKEN="your-api-key"
//	go run ./examples/basic
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

func main() {
	// Create client (reads MARKETDATA_TOKEN from environment or .env file)
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// ============= SIMPLE HAPPY PATH (2-3 lines) =============
	// The Get* convenience methods use context.Background() and return (data, error).

	fmt.Println("=== QUICK START ===")

	// Single stock quote
	quote, err := client.Stocks.GetQuote("AAPL")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(quote)

	// Multiple stock quotes
	quotes, err := client.Stocks.GetQuotes("AAPL", "MSFT", "GOOG")
	if err != nil {
		log.Fatal(err)
	}
	for _, q := range quotes {
		fmt.Println(q)
	}

	// Market status
	status, err := client.Markets.GetStatus()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(status)

	// ============= FULL API (with context and response metadata) =============
	// Use the full methods when you need context control or rate limit metadata.

	fmt.Println("\n=== FULL API ===")

	ctx := context.Background()

	// Single quote with 52-week data
	q, resp, err := client.Stocks.Quote(ctx, "AAPL", stocks.WithFiftyTwoWeek(true))
	if err != nil {
		log.Fatal(err)
	}
	if q == nil {
		fmt.Println("AAPL: no quote data available")
	} else {
		fmt.Printf("AAPL: $%.2f (52w high: $%.2f, low: $%.2f)\n", q.Last, q.FiftyTwoWeekHigh, q.FiftyTwoWeekLow)
	}
	fmt.Printf("  Credits remaining: %d\n", resp.RateLimit.Remaining)

	// Daily candles
	candles, _, err := client.Stocks.Candles(ctx, "AAPL",
		stocks.WithResolution(stocks.ResolutionDaily),
		stocks.WithCandleWindow(stocks.LastN(3)),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range candles {
		fmt.Println(c)
	}

	// Rate limits
	// client.RateLimits() is a snapshot of the LAST completed response, not a
	// running total: Consumed is what that single request cost. A session
	// total means summing resp.RateLimit.Consumed across your own calls — see
	// the response-formats example.
	limits := client.RateLimits()
	fmt.Printf("\nCredits remaining: %d/%d (last request cost %d)\n",
		limits.Remaining, limits.Limit, limits.Consumed)
}
