package markets_test

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
	"github.com/MarketDataApp/sdk-go/v2/marketdata/markets"
)

func ExampleService_Status() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	status, _, err := client.Markets.Status(context.Background(),
		markets.WithCountry("US"),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(status)
}

func ExampleService_StatusHistory() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	statuses, _, err := client.Markets.StatusHistory(context.Background(),
		markets.WithHistoryWindow(markets.Between(
			time.Date(2024, time.December, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, time.December, 31, 0, 0, 0, 0, time.UTC),
		)),
		markets.WithCountry("US"),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, status := range statuses {
		fmt.Println(status)
	}
}

func ExampleService_GetStatus() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	status, err := client.Markets.GetStatus(markets.WithDate(time.Date(2024, time.July, 4, 0, 0, 0, 0, time.UTC)))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(status)
}

func ExampleService_AsCSV() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	csv, err := client.Markets.AsCSV().Status(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(csv.CSV())
}
