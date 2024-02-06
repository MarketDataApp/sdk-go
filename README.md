# Market Data's Official Go SDK

> :warning: **Pre-Alpha Version**: This SDK is currently in development and is not stable. Not currently suitable for client use.

[![GoDoc](https://godoc.org/github.com/MarketDataApp/sdk-go?status.svg)](https://godoc.org/github.com/MarketDataApp/sdk-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/MarketDataApp/sdk-go)](https://goreportcard.com/report/github.com/MarketDataApp/sdk-go)
![Tests and linters](https://github.com/MarketDataApp/sdk-go/actions/workflows/go.yml/badge.svg)
[![Coverage](https://codecov.io/gh/MarketDataApp/sdk-go/branch/main/graph/badge.svg)](https://codecov.io/gh/MarketDataApp/sdk-go)
[![License](https://img.shields.io/github/license/MarketDataApp/sdk-go.svg)](https://github.com/MarketDataApp/sdk-go/blob/master/LICENSE)
![SDK Version](https://img.shields.io/badge/version-0.0.5-blue.svg)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/MarketDataApp/sdk-go)

#### Connect With The Market Data Community

[![Discord](https://img.shields.io/badge/Discord-join%20chat-7389D8.svg?logo=discord&logoColor=ffffff)](https://discord.com/invite/GmdeAVRtnT)
[![Twitter](https://img.shields.io/twitter/follow/MarketDataApp?style=social)](https://twitter.com/MarketDataApp)
[![Helpdesk](https://img.shields.io/badge/Support-Ticketing-ff69b4.svg?logo=TicketTailor&logoColor=white)](https://www.marketdata.app/dashboard/)

# Installation

To install the SDK, you can use the following command:

```bash
go get github.com/MarketDataApp/sdk-go
```

After installation, you can import it in your project like this:

```go
import api "github.com/MarketDataApp/sdk-go"
```

# Endpoints Status:

 | Endpoint Category | Endpoint     | v1 Status | v2 Status |
 |-------------------|--------------|-----------|-----------|
 | Markets           | Status       | ✅        |           |
 | Stocks            | Candles      | ✅        |     ✅    |
 | Stocks            | Quotes       | ✅        |           |
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

Note on v2: Even though some v2 endpoints are available for use in this SDK, Market Data has not yet released v2 of its API for clients and v2 usage is restricted to admins only. Clients should onlly use v1 endpoints at this time. Even after v2 is released, we do not plan on deprecating v1 endpoints, so please build your applications with confidence using v1 endpoints.

# Example Usage

See the examples files for each data type for examples of how to use each endpoint.

### Authentication

To authenticate with the Market Data API, you need to set your token, which should have been e-mailed to you when you first signed up for an account. If you do not have a token, request a new one from the [Market Data Dashboard](https://www.marketdata.app/dashboard/). Set the token in the environment variable MARKETDATA_TOKEN. Alternatively, you can hardcode it in your code, but please be aware that this is not secure and could pose a risk if your code is shared.

```bash
export MARKETDATA_TOKEN="<your_api_token>"   # mac/linux
setx MARKETDATA_TOKEN "<your_api_token>"     # windows
```

### Logging

The SDK systematically logs API responses to facilitate troubleshooting and analysis, adhering to the following rules:
- **Client Errors:** Responses with status codes in the range 400-499 are logged to `client_error.log`.
- **Server Errors:** Responses with status codes in the range 500-599 are logged to `server_error.log`.
- **Successful Requests:** If debug mode is activated, responses with status codes in the range 200-299 are logged to `success.log`.

All log files are formatted in JSON and stored within the `/logs` directory. Each entry captures comprehensive details including the request URL, request headers, CF Ray ID (a unique identifier for the request), response status code, response headers, and the response body, providing a full context for each logged event.

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

### Troubleshooting: Debug Mode

The SDK provides a debug mode that can be enabled to help you understand how the SDK is working and troubleshoot any issues you might encounter. When debug mode is enabled, the SDK will print the log to the console. This includes the full URL of the request, all request and response headers, and more.

To enable debug mode, you need to call the `Debug` method on the `MarketDataClient` instance and pass `true` as the argument. 

#### Debug Code Example

```go
package main

import (
	api "github.com/MarketDataApp/sdk-go"
	"log"
)

func main() {
	client, err := api.GetClient()
	if err != nil {
		log.Fatalf("Failed to get client: %v", err)
	}

	client.Debug(true)

	quote, err := client.StockQuotes().Symbol("AAPL").Get()
	if err != nil {
		log.Fatalf("Failed to get stock quotes: %v", err)
	}

	log.Printf("Quote: %+v\n", quote)
}
```

Please note that the information printed in debug mode can be quite verbose. It is recommended to use this mode only when you are facing issues and need to understand what's happening under the hood. When debug mode is activated all requests are logged, not just requests that fail.

Debug mode can be particularly useful when you are first getting started with the SDK. It can help you understand how the SDK constructs requests, how it handles responses, and how it manages errors. By examining the debug output, you can gain a deeper understanding of the SDK's inner workings and be better prepared to handle any issues that might arise.