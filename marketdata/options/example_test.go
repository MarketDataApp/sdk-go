package options_test

// The Example* functions below are compile-checked documentation, not
// executed tests: none has a "// Output:" comment (T-5), which is
// deliberate, not an oversight. Each builds its client with
// marketdata.NewClient() and no token, so it runs in demo mode — the
// method calls below would still reach the live API over the network.
// Giving them Output assertions would make this package's ordinary `go
// test` (meant to be hermetic and network-free, per ADR-010) depend on
// live network access and produce non-deterministic output (prices,
// timestamps). `go test` still compiles every example here, so a method
// signature change breaks the build immediately; live, output-verified
// counterparts for the most representative method per package live in
// integration/example_test.go, gated by the `integration` build tag.

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
)

// ExampleService_Chain fetches a filtered options chain for AAPL, limiting
// the result to a single expiration and a strike range.
func ExampleService_Chain() {
	client, err := marketdata.NewClient() // token from MARKETDATA_TOKEN
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	expiration := time.Now().AddDate(0, 1, 0)
	chain, _, err := client.Options.Chain(context.Background(), "AAPL",
		options.WithExpiry(options.OnExpiration(expiration)),
		options.WithStrike(options.StrikeRange(150, 200)),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(chain)
}

// ExampleService_Expirations lists the available expiration dates for AAPL.
func ExampleService_Expirations() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	expirations, _, err := client.Options.Expirations(context.Background(), "AAPL")
	if err != nil {
		log.Fatal(err)
	}
	if expirations != nil {
		for _, exp := range expirations.Dates {
			fmt.Println(exp.Format("2006-01-02"))
		}
	}
}

// ExampleService_Quote fetches a quote for a single option contract by its
// OCC option symbol.
func ExampleService_Quote() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	quote, _, err := client.Options.Quote(context.Background(), "AAPL250117C00150000")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(quote)
}

// ExampleService_Quotes fetches quotes for several option contracts
// concurrently and prints the merged results.
func ExampleService_Quotes() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	quotes, _, err := client.Options.Quotes(context.Background(), []string{
		"AAPL250117C00150000",
		"AAPL250117P00150000",
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, q := range quotes {
		fmt.Println(q)
	}
}

// ExampleService_Lookup resolves an underlying, expiration, strike, and
// option type to the corresponding OCC option symbol.
func ExampleService_Lookup() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	expiration := time.Now().AddDate(0, 1, 0)
	symbol, _, err := client.Options.Lookup(context.Background(), "AAPL", expiration, 150, options.Call)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(symbol)
}

// ExampleService_GetChain uses the convenience wrapper, which needs no
// context and returns only the chain and an error.
func ExampleService_GetChain() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	chain, err := client.Options.GetChain("AAPL",
		options.WithExpiry(options.OnExpiration(time.Now().AddDate(0, 1, 0))),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(chain)
}

func ExampleService_AsCSV() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	csv, err := client.Options.AsCSV().Chain(context.Background(), "AAPL")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(csv.CSV())
}
