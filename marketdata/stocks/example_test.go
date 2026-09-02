package stocks_test

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
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

// The examples in this file are compile-only: they have no output
// declarations, so they are built by go test but never executed against
// the live API. The client reads its token from the MARKETDATA_TOKEN
// environment variable.

func ExampleService_Quote() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	quote, _, err := client.Stocks.Quote(context.Background(), "AAPL")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(quote)
}

func ExampleService_Quotes() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	quotes, _, err := client.Stocks.Quotes(context.Background(), []string{"AAPL", "MSFT", "GOOG"})
	if err != nil {
		log.Fatal(err)
	}
	for _, q := range quotes {
		fmt.Println(q)
	}
}

func ExampleService_Candles() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	candles, _, err := client.Stocks.Candles(context.Background(), "AAPL",
		stocks.WithResolution(stocks.ResolutionDaily),
		stocks.WithCandleWindow(stocks.Between(time.Now().AddDate(0, -1, 0), time.Now())),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range candles {
		fmt.Println(c)
	}
}

func ExampleService_BulkCandles() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	candles, _, err := client.Stocks.BulkCandles(context.Background(), []string{"AAPL", "MSFT", "GOOG"})
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range candles {
		fmt.Println(c)
	}
}

func ExampleService_Prices() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	prices, _, err := client.Stocks.Prices(context.Background(), []string{"AAPL", "MSFT"})
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range prices {
		fmt.Println(p)
	}
}

func ExampleService_Earnings() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	earnings, _, err := client.Stocks.Earnings(context.Background(), "AAPL",
		stocks.WithEarningsWindow(stocks.LastN(4)),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range earnings {
		fmt.Println(e)
	}
}

func ExampleService_News() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	articles, _, err := client.Stocks.News(context.Background(), "AAPL",
		stocks.WithNewsWindow(stocks.LastN(5)),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, a := range articles {
		fmt.Println(a)
	}
}

func ExampleService_GetQuote() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	quote, err := client.Stocks.GetQuote("AAPL")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(quote)
}

func ExampleService_AsCSV() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	csv, err := client.Stocks.AsCSV().Quote(context.Background(), "AAPL")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(csv.CSV())
}
