# Market Data's Official Go SDK

> :warning: **Pre-Alpha Version**: This SDK is currently in development and is not stable. Not currently suitable for client use.


[![Go Report Card](https://goreportcard.com/badge/github.com/MarketDataApp/sdk-go)](https://goreportcard.com/report/github.com/MarketDataApp/sdk-go)
![Tests and linters](https://github.com/MarketDataApp/sdk-go/actions/workflows/main.yml/badge.svg)


# Installation

To install the SDK, you can use the following command:

```bash
go get github.com/MarketDataApp/sdk-go
```

After installation, you can import it in your project like this:

```go
import md "github.com/MarketDataApp/sdk-go/client"
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
 | Options           | Expirations  | ❌        |           |
 | Options           | Lookup       | ❌        |           |
 | Options           | Strikes      | ❌        |           |
 | Options           | Option Chain | ❌        |           |
 | Options           | Quotes       | ❌        |           |
 | Indices           | Candles      | ✅        |           |
 | Indices           | Quotes       | ✅        |           |

# Example Usage

See the examples files for each data type for examples of how to use each endpoint.

## Authentication

To authenticate with the Market Data API, you need to set your token, which should have been e-mailed to you when you first signed up for an account. If you do not have a token, request a new one from the [Market Data Dashboard](https://www.marketdata.app/dashboard/). Set the token in the environment variable MARKETDATA_TOKEN. Alternatively, you can hardcode it in your code, but please be aware that this is not secure and could pose a risk if your code is shared.

```bash
export MARKETDATA_TOKEN="<your_api_token>"        # mac/linux
setx MARKETDATA_TOKEN "<your_api_token>"          # windows
```

### Logging

The SDK logs all unsuccessful 400-499 responses to a file named `client_error.log` and all unsuccessful 500-599 responses to `server_error.log` in JSON format. This feature is designed to help you understand and troubleshoot any issues that might occur while interacting with the API. If debug mode is enabled the SDK will also log successful 200-299 responses to `success.log`. The three log files are stored in the `/logs` directory. 

Each log entry includes details such as the request URL, request headers, CF Ray ID (unique identifier for the request), response status code, response headers, and response body. This comprehensive information can be invaluable when diagnosing problems.

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
    "Authorization":[
      "Bearer **********************************************************HMD0"
    ],
    "User-Agent":[
      "sdk-go/0.0.4"
    ]
  },
  "response_headers":{
    "Allow":[
      "GET, HEAD, OPTIONS"
    ],
    "Alt-Svc":[
      "h3=\":443\"; ma=86400"
    ],
    "Cf-Cache-Status":[
      "DYNAMIC"
    ],
    "Cf-Ray":[
      "84b29bd46f468d96-MIA"
    ],
    "Content-Type":[
      "application/json"
    ],
    "Cross-Origin-Opener-Policy":[
      "same-origin"
    ],
    "Date":[
      "Thu, 25 Jan 2024 18:34:10 GMT"
    ],
    "Nel":[
      "{\"success_fraction\":0,\"report_to\":\"cf-nel\",\"max_age\":604800}"
    ],
    "Referrer-Policy":[
      "same-origin"
    ],
    "Report-To":[
      "{\"endpoints\":[{\"url\":\"https:\\/\\/a.nel.cloudflare.com\\/report\\/v3?s=9vEr7PiX6zgR6cdNLegGNMCOzC6yy9KHd0IIzN3yPl14KDMBB9kkMV19xVP79jOdqPWBS9Ena%2B43XHWh%2B7cKqAQc7GrRCm2ZWpX4xqhXidyQeRgNoPcWsSsyv5xSD8v9ywFQdNc%3D\"}],\"group\":\"cf-nel\",\"max_age\":604800}"
    ],
    "Server":[
      "cloudflare"
    ],
    "Vary":[
      "Accept, Origin"
    ],
    "X-Api-Ratelimit-Consumed":[
      "0"
    ],
    "X-Api-Ratelimit-Limit":[
      "100000"
    ],
    "X-Api-Ratelimit-Remaining":[
      "100000"
    ],
    "X-Api-Ratelimit-Reset":[
      "1706279400"
    ],
    "X-Api-Response-Log-Id":[
      "77524556"
    ],
    "X-Content-Type-Options":[
      "nosniff"
    ],
    "X-Frame-Options":[
      "DENY"
    ]
  },
  "response_body":{
    "ask":[
      194.39
    ],
    "askSize":[
      1
    ],
    "bid":[
      194.38
    ],
    "bidSize":[
      4
    ],
    "change":[
      -0.11
    ],
    "changepct":[
      -0.0006
    ],
    "last":[
      194.39
    ],
    "mid":[
      194.38
    ],
    "s":"ok",
    "symbol":[
      "AAPL"
    ],
    "updated":[
      1706207650
    ],
    "volume":[
      29497567
    ]
  }
}
```

### Debug Mode

The SDK provides a debug mode that can be enabled to help you understand how the SDK is working and troubleshoot any issues you might encounter. When debug mode is enabled, the SDK will print the log to the console. This includes the full URL of the request, all request and response headers, and more.

To enable debug mode, you need to call the `Debug` method on the `MarketDataClient` instance and pass `true` as the argument. Here is an quick example:

```go
    package main

    import 	api "github.com/MarketDataApp/sdk-go/client"

    client, err := api.GetClient()
	if err != nil {
		log.Fatalf("Failed to get client: %v", err)
	}

	client.Debug(true)
    quote, err := api.StockQuotes().Symbol("AAPL").Get()
```

Please note that the information printed in debug mode can be quite verbose. It is recommended to use this mode only when you are facing issues and need to understand what's happening under the hood. When debug mode is activated all requests are logged, not just requests that fail.

Debug mode can be particularly useful when you are first getting started with the SDK. It can help you understand how the SDK constructs requests, how it handles responses, and how it manages errors. By examining the debug output, you can gain a deeper understanding of the SDK's inner workings and be better prepared to handle any issues that might arise.