package utilities_test

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

	"github.com/MarketDataApp/sdk-go/v2/marketdata"

	// Imported so the examples are associated with the utilities package.
	_ "github.com/MarketDataApp/sdk-go/v2/marketdata/utilities"
)

func ExampleService_Status() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	status, _, err := client.Utilities.Status(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(status)
}

func ExampleService_Headers() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	headers, _, err := client.Utilities.Headers(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	if headers != nil {
		for name, value := range headers.Headers {
			fmt.Println(name, value)
		}
	}
}

func ExampleService_User() {
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	user, _, err := client.Utilities.User(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(user)
}
