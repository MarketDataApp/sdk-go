# Go SDK

Welcome to the Market Data **Go SDK v2**. This is a complete, idiomatic rewrite of the SDK that gives you type-safe access to real-time and historical stock, option, fund, and market data with nothing but the Go standard library at runtime.

## Quick Start

Create a client, make a call, and you are done. The client reads your token from the `MARKETDATA_TOKEN` environment variable (or a `.env` file) automatically, so no token needs to appear in your code:

```go
package main

import (
	"fmt"
	"log"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
)

func main() {
	// Reads MARKETDATA_TOKEN from the environment or a .env file.
	client, err := marketdata.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	quote, err := client.Stocks.GetQuote("AAPL")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("AAPL  last: $%.2f  bid: $%.2f  ask: $%.2f\n",
		quote.Last, quote.Bid, quote.Ask)
}
```

```
AAPL  last: $234.07  bid: $234.05  ask: $234.09
```

Install it with `go get` (the import path includes the `/v2` suffix):

```bash
go get github.com/MarketDataApp/sdk-go/v2
```

The SDK requires **Go 1.22 or later** and has **zero third-party runtime dependencies**.

## Open Source

The SDK is open source under the MIT license and is available on GitHub. Feel free to contribute to the project, report bugs, or request new features.

- [Go SDK GitHub](https://github.com/MarketDataApp/sdk-go/)
- [Go SDK on pkg.go.dev](https://pkg.go.dev/github.com/MarketDataApp/sdk-go/v2)

## Documentation

The best source for documentation on the SDK is right here. This documentation is the most up-to-date and accurate source of information on the SDK.

## Using the SDK

This SDK is designed to help you get up and running with Market Data's APIs as quickly as possible, providing you with all the tools you need to access real-time stock and options prices, historical data, and much more.

### Getting Started

1. [Install the SDK](./installation.md) into your Go module.
2. Set up your [authentication token](./authentication.md) to access the API.
3. Learn about the [client](./client.md) and how to make your first API requests.
4. Configure [parameters](./parameters.md) to customize mode, columns, pagination, and other universal parameters.
5. Handle failures with the SDK's typed [errors](./errors.md), and set up [logging](./logging.md) for debugging.
6. Explore the available endpoints for [stocks](./stocks/README.md), [options](./options/README.md), [funds](./funds/README.md), [markets](./markets/README.md), and [utilities](./utilities/README.md).

### Support

- If you have any questions or need further assistance, please don't hesitate to open a ticket at our [helpdesk](https://www.marketdata.app/dashboard/).
- If you find a bug you may also [open an issue in our GitHub repository](https://github.com/MarketDataApp/sdk-go/issues). Please only open issues if you find a bug. Use our helpdesk for general questions or implementation assistance.
