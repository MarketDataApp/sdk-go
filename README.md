<div align="center">

# Go SDK for Market Data v1.0
### Access Financial Data with Ease

> This is the official Go SDK for [Market Data](https://www.marketdata.app/). It provides developers with a powerful, easy-to-use interface to obtain real-time and historical financial data. Ideal for building financial applications, trading bots, and investment strategies.

[![GoDoc](https://godoc.org/github.com/MarketDataApp/sdk-go?status.svg)](https://godoc.org/github.com/MarketDataApp/sdk-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/MarketDataApp/sdk-go)](https://goreportcard.com/report/github.com/MarketDataApp/sdk-go)
![Tests and linters](https://github.com/MarketDataApp/sdk-go/actions/workflows/go.yml/badge.svg)
[![Coverage](https://codecov.io/gh/MarketDataApp/sdk-go/branch/main/graph/badge.svg)](https://codecov.io/gh/MarketDataApp/sdk-go)
[![License](https://img.shields.io/github/license/MarketDataApp/sdk-go.svg)](https://github.com/MarketDataApp/sdk-go/blob/master/LICENSE)
![SDK Version](https://img.shields.io/badge/version-1.0.0-blue.svg)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/MarketDataApp/sdk-go)
![Lines of Code](https://img.shields.io/badge/lines_of_code-8342-blue)

#### Connect With The Market Data Community

[![Website](https://img.shields.io/badge/Website-marketdata.app-blue)](https://www.marketdata.app/)
[![Discord](https://img.shields.io/badge/Discord-join%20chat-7389D8.svg?logo=discord&logoColor=ffffff)](https://discord.com/invite/GmdeAVRtnT)
[![Twitter](https://img.shields.io/twitter/follow/MarketDataApp?style=social)](https://twitter.com/MarketDataApp)
[![Helpdesk](https://img.shields.io/badge/Support-Ticketing-ff69b4.svg?logo=TicketTailor&logoColor=white)](https://www.marketdata.app/dashboard/)

</div>

# Installation

To install the SDK, you can use the following command:

```bash
go get github.com/MarketDataApp/sdk-go
```

After installation, you can import it in your project like this:

```go
import api "github.com/MarketDataApp/sdk-go"
```

# Documentation

For advanced usage, review the module's complete documentation:

- [Go SDK Documentation at GoDoc](https://godoc.org/github.com/MarketDataApp/sdk-go)
- [Go SDK Documentation at Market Data](https://www.marketdata.app/docs/sdk/go)

# Get Started Quickly

### 1. Sign-Up For A Free Market Data Account

Signing up for a Market Data account is straightforward and grants you immediate access to our wealth of market data. We offer a free account tier that allows you to explore the capabilities of our API without any cost. Additionally, all our paid plans come with a free 30-day trial, giving you the opportunity to test out the advanced features and decide which plan best suits your needs.

- To sign up, simply visit our website at [Market Data](https://www.marketdata.app/). The process is quick and only requires your basic information. Once signed up, you can start using the API right away with the free tier or take advantage of the 30-day trial to explore our premium offerings.

Remember, no credit card is required for the free account tier, and you can upgrade, downgrade, or cancel your subscription at any time during or after the trial period.

### 2. Set-up Authentication

To authenticate with the Market Data API, you need to set your token, which should have been e-mailed to you when you first signed up for an account. If you do not have a token, request a new one from the [Market Data Dashboard](https://www.marketdata.app/dashboard/). Set the token in the environment variable MARKETDATA_TOKEN. Alternatively, you can hardcode it in your code, but please be aware that this is not secure and could pose a risk if your code is shared.

```bash
export MARKETDATA_TOKEN="<your_api_token>"   # mac/linux
setx MARKETDATA_TOKEN "<your_api_token>"     # windows
```

### 3. Make Your First Request

Check the examples folder for working examples for each request. 

Get a stock quote:
```go
import (
  "fmt"
  "log"

  api "github.com/MarketDataApp/sdk-go"
)

func main() {
  // Initialize a new request, set the symbol parameter, make the request.
	quotes, err := api.StockQuote().Symbol("AAPL").Get() 
	if err != nil {
		log.Fatalf("Failed to get stock quotes: %v", err)
	}

  // Loop over the quotes and print them out.
  for _, quote := range quotes {
		fmt.Println(quote)
	}
```

# SDK Usage

- All requests are initialized using the name of the endpoint and parameters are set using a builder pattern. The quickest way to get started is by viewing the examples.
- All requests have 3 methods to get results:
  1.  Use the `.Get()` method on any request **to get a slice of objects**. _This is what you will use in most cases._
  2.  Use the `.Packed()` method on any request to **get a struct that models the Market Data JSON response**. _If you want to model the objects differently_, this method could be useful for you.
  3.  Use the `.Raw()` method on any request to **get the raw *resty.Response object**. This allows you to _access the raw JSON or the raw *http.Response_ via any of Resty's methods. You can use any of the Resty methods on the response.
- We have already implemented `.String()` methods for all `.Get()` and `.Packed()` responses and sorting methods for all candle objects.

> Note: Since all our structs are pre-defined based on the Market Data API's standard response format, our API's optional parameters such as `human`, `columns`, `dateformat`, `format` are not supported in this SDK because they modify the API's standard JSON output. If you wish to make use of these parameters, _you will need to model your own structs that can unmarshal the modified API response_.

# Logging

The SDK systematically logs API responses to facilitate troubleshooting and analysis, adhering to the following rules:
- **Client Errors:** Responses with status codes in the range 400-499 are logged to `client_error.log`.
- **Server Errors:** Responses with status codes in the range 500-599 are logged to `server_error.log`.
- **Successful Requests:** If debug mode is activated, responses with status codes in the range 200-299 are logged to `success.log`.

All log files are formatted in JSON and stored within the `/logs` subdirectory. Each entry captures comprehensive details including the request URL, request headers, CF Ray ID (a unique identifier for the request), response status code, response headers, and the response body, providing a full context for each logged event.

Example of a log entry:
```json
{
  "level":"info",
  "ts":"2024-01-25T15:34:10.642-0300",
  "msg":"Successful Request",
  "cf_ray":"84b29bd46f468d96-MIA",
  "request_url":"https://api.marketdata.app/v1/stocks/quotes/AAPL/",
  "ratelimit_consumed":0,
  "response_code":200,
  "delay_ms":254,
  "request_headers":{
    "Authorization": ["Bearer **********************************************************HMD0"],
    "User-Agent": ["sdk-go/0.0.4"]
  },
  "response_headers": {
    "Allow": ["GET, HEAD, OPTIONS"],
    "Alt-Svc": ["h3=\":443\"; ma=86400"],
    "Cf-Cache-Status": ["DYNAMIC"],
    "Cf-Ray": ["84b29bd46f468d96-MIA"],
    "Content-Type": ["application/json"],
    "Cross-Origin-Opener-Policy": ["same-origin"],
    "Date": ["Thu, 25 Jan 2024 18:34:10 GMT"],
    "Nel": ["{\"success_fraction\":0,\"report_to\":\"cf-nel\",\"max_age\":604800}"],
    "Referrer-Policy": ["same-origin"],
    "Report-To": ["{\"endpoints\":[{\"url\":\"https:\\/\\/a.nel.cloudflare.com\\/report\\/v3?s=9vEr7PiX6zgR6cdNLegGNMCOzC6yy9KHd0IIzN3yPl14KDMBB9kkMV19xVP79jOdqPWBS9Ena%2B43XHWh%2B7cKqAQc7GrRCm2ZWpX4xqhXidyQeRgNoPcWsSsyv5xSD8v9ywFQdNc%3D\"}],\"group\":\"cf-nel\",\"max_age\":604800}"],
    "Server": ["cloudflare"],
    "Vary": ["Accept, Origin"],
    "X-Api-Ratelimit-Consumed": ["0"],
    "X-Api-Ratelimit-Limit": ["100000"],
    "X-Api-Ratelimit-Remaining": ["100000"],
    "X-Api-Ratelimit-Reset": ["1706279400"],
    "X-Api-Response-Log-Id": ["77524556"],
    "X-Content-Type-Options": ["nosniff"],
    "X-Frame-Options": ["DENY"]
  },
  "response_body": {
    "ask": [194.39],
    "askSize": [1],
    "bid": [194.38],
    "bidSize": [4],
    "change": [-0.11],
    "changepct": [-0.0006],
    "last": [194.39],
    "mid": [194.38],
    "s": "ok",
    "symbol": ["AAPL"],
    "updated": [1706207650],
    "volume": [29497567]
  }
}
```

# Troubleshooting: Debug Mode

The SDK provides a debug mode that can be enabled to help you understand how the SDK is working and troubleshoot any issues you might encounter. When debug mode is enabled, the SDK will print the log to the console. This includes the full URL of the request, all request and response headers, and more.

To enable debug mode, you need to call the `.Debug()` method on the `MarketDataClient` instance and pass `true` as the argument. 

#### Debug Code Example

```go
package main

import (
  "log"
  "fmt"

	api "github.com/MarketDataApp/sdk-go"
)

func main() {
	client, err := api.GetClient()
	if err != nil {
		log.Fatalf("Failed to get client: %v", err)
	}

	client.Debug(true) // 👈 Here is where debug mode is turned on.

	quotes, err := client.StockQuotes().Symbol("AAPL").Get()
	if err != nil {
		log.Fatalf("Failed to get stock quotes: %v", err)
	}

  for _, quote := range quotes {
		fmt.Println(quote)
	}
}
```

Please note that the information printed in debug mode can be quite verbose. It is recommended to use this mode only when you are facing issues and need to understand what's happening under the hood. When debug mode is activated all requests are logged, not just requests that fail.

Debug mode can be particularly useful when you are first getting started with the SDK. It can help you understand how the SDK constructs requests, how it handles responses, and how it manages errors. By examining the debug output, you can gain a deeper understanding of the SDK's inner workings and be better prepared to handle any issues that might arise.

# Important Information for SDK Users

### Endpoint Coverage:

Market Data's Go SDK covers the vast majority of v1 endpoints. See our complete list of endpoints in the [Market Data API Documentation](https://www.marketdata.app/docs/api).

 | **Category** | **Endpoint** | **v1 Status** | **v2 Status** |
 |-------------------|--------------|-----------|-----------|
 | Markets           | Status       | ✅        |           |
 | Stocks            | Candles      | ✅        |     ✅    |
 | Stocks            | Bulk Candles | ✅        |           |
 | Stocks            | Quotes       | ✅        |           |
 | Stocks            | Bulk Quotes  | ✅        |           |
 | Stocks            | Earnings     | ✅        |           |
 | Stocks            | Tickers      | ❌        |     ✅    |
 | Stocks            | News         | ✅        |     ❌    |
 | Options           | Expirations  | ✅        |           |
 | Options           | Lookup       | ✅        |           |
 | Options           | Strikes      | ✅        |           |
 | Options           | Option Chain | ✅        |           |
 | Options           | Quotes       | ✅        |           |
 | Indices           | Candles      | ✅        |           |
 | Indices           | Quotes       | ✅        |           |

> Note on v2: Even though some v2 endpoints are available for use in this SDK, Market Data has not yet released v2 of its API for clients and v2 usage is restricted to admins only. Clients should only use v1 endpoints at this time. Even after v2 is released, we do not plan on deprecating v1 endpoints, so please build your applications with confidence using v1 endpoints.

 | **Category**         | **Endpoint**     | **v1 Status** | **v2 Status** |
 |----------------------|------------------|:-------------:|:-------------:|
 |                      |                  |               |               |
 | **[MARKETS](https://www.marketdata.app/docs/api/markets)**         |                  |               |               |
 |                      | [Status](https://www.marketdata.app/docs/api/markets/status)           |       ✅      |       ❌       |
 |                      |                  |               |               |
 | **[STOCKS](https://www.marketdata.app/docs/api/stocks)**           |                  |               |               |
 |                      | [Candles](https://www.marketdata.app/docs/api/stocks/candles)          |       ✅      |       ✅      |
 |                      | [Bulk Candles](https://www.marketdata.app/docs/api/stocks/bulkcandles)     |       ✅      |       ❌        |
 |                      | [Quotes](https://www.marketdata.app/docs/api/stocks/quotes)           |       ✅      |       ❌        |
 |                      | [Bulk Quotes](https://www.marketdata.app/docs/api/stocks/bulkquotes)      |       ✅      |       ❌        |
 |                      | [Earnings](https://www.marketdata.app/docs/api/stocks/earnings)         |       ✅      |       ❌        |
 |                      | [Tickers](https://www.marketdata.app/docs/api/stocks/tickers)          |       ❌      |       ✅      |
 |                      | [News](https://www.marketdata.app/docs/api/stocks/news)             |       ✅      |       ❌      |
 |                      |                  |               |               |
 | **[OPTIONS](https://www.marketdata.app/docs/api/options)**          |                  |               |               |
 |                      | [Expirations](https://www.marketdata.app/docs/api/options/expirations)      |       ✅      |       ❌        |
 |                      | [Lookup](https://www.marketdata.app/docs/api/options/lookup)           |       ✅      |       ❌       |
 |                      | [Strikes](https://www.marketdata.app/docs/api/options/strikes)          |       ✅      |       ❌        |
 |                      | [Option Chain](https://www.marketdata.app/docs/api/options/chain)     |       ✅      |       ❌        |
 |                      | [Quotes](https://www.marketdata.app/docs/api/options/quotes)           |       ✅      |       ❌        |
 |                      |                  |               |               |
 | **[INDICES](https://www.marketdata.app/docs/api/indices)**          |                  |               |               |
 |                      | [Candles](https://www.marketdata.app/docs/api/indices/candles)          |       ✅      |       ❌         |
 |                      | [Quotes](https://www.marketdata.app/docs/api/indices/quotes)           |       ✅      |       ❌        |

### SDK License & Data Usage Terms

The license for the Go SDK for Market Data is specifically for the SDK software and does not cover the data provided by the Market Data API. It's crucial to understand that accessing data through the SDK means you're also subject to the Market Data Terms of Service. You can review these terms at [https://www.marketdata.app/terms/](https://www.marketdata.app/terms/) to fully comprehend your rights and responsibilities concerning data usage.

### Contributing to the SDK

Your contributions to the Go SDK are highly valued. Whether you're looking to add new features, rectify bugs, or enhance the documentation, we encourage you to get involved. Please submit your contributions via pull requests or issues on our GitHub repository. Your efforts play a significant role in improving the SDK for the benefit of all users.

### Contact and Support

Should you need any assistance, the most effective way to reach us is through our [helpdesk](https://www.marketdata.app/dashboard/). We kindly ask that you use GitHub issues solely for bug reports and not for support inquiries.
