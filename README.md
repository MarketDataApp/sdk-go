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
 | Stocks            | Quotes       | ❌        |           |
 | Stocks            | Earnings     | ❌        |           |
 | Stocks            | Tickers      | ❌        |     ✅    |
 | Options           | Expirations  | ❌        |           |
 | Options           | Lookup       | ❌        |           |
 | Options           | Strikes      | ❌        |           |
 | Options           | Option Chain | ❌        |           |
 | Options           | Quotes       | ❌        |           |
 | Indices           | Candles      | ❌        |           |
 | Indices           | Quotes       | ❌        |           |

# Usage

Usage examples coming soon.

## Authentication

To authenticate with the Market Data API, you need to set your token, which should have been e-mailed to you when you first signed up for an account. If you do not have a token, request a new one from the [Market Data Dashboard](https://www.marketdata.app/dashboard/). Set the token in the environment variable MARKETDATA_TOKEN. Alternatively, you can hardcode it in your code, but please be aware that this is not secure and could pose a risk if your code is shared.

```bash
export MARKETDATA_TOKEN="<your_api_token>"        # mac/linux
setx MARKETDATA_TOKEN "<your_api_token>"          # windows
```

### Logging

The SDK logs all unsuccessful requests to a file named `sdk-go.log` in JSON format. This feature is designed to help you understand and troubleshoot any issues that might occur while interacting with the API.

Each log entry includes details such as the request URL, request headers, CF Ray ID (unique identifier for the request), response status code, response headers, and response body. This comprehensive information can be invaluable when diagnosing problems.

Example of a log entry:

```json
{
    "level": "info",
    "ts": "2024-01-02T14:46:21.971-0300",
    "msg": "Request",
    "request_url": "https://api.marketdata.app/v1/markets/status/?country=US&from=2022-01-01&to=2022-01-10",
    "request_headers": {
        "Authorization": [
            "Bearer **********************************************************HMD0"
        ],
        "User-Agent": [
            "sdk-go/0.0.3"
        ]
    },
    "response_code": 200,
    "cf_ray": "83f4d22edf7325a0-MIA",
    "response_headers": {
        "Allow": [
            "GET, HEAD, OPTIONS"
        ],
        "Alt-Svc": [
            "h3=\":443\"; ma=86400"
        ],
        "Cf-Cache-Status": [
            "DYNAMIC"
        ],
        "Cf-Ray": [
            "83f4d22edf7325a0-MIA"
        ],
        "Content-Type": [
            "application/json"
        ],
        "Cross-Origin-Opener-Policy": [
            "same-origin"
        ],
        "Date": [
            "Tue, 02 Jan 2024 17:46:22 GMT"
        ],
        "Nel": [
            "{\"success_fraction\":0,\"report_to\":\"cf-nel\",\"max_age\":604800}"
        ],
        "Referrer-Policy": [
            "same-origin"
        ],
        "Report-To": [
            "{\"endpoints\":[{\"url\":\"https:\\/\\/a.nel.cloudflare.com\\/report\\/v3?s=Q%2B4VIX2nQeWFotmIejRWNoVQyEYAcMsR629YlytkuffJ%2B6bzgE5cd5SjKL2yqnHehxjZH%2BuEiJaHtApfhpQxDfsoTID4d2OGdkF8H8ojSFAT%2BHqC13wGbVeZSxZOX4U1RH0%2F3iU%3D\"}],\"group\":\"cf-nel\",\"max_age\":604800}"
        ],
        "Server": [
            "cloudflare"
        ],
        "Vary": [
            "Accept, Origin"
        ],
        "X-Api-Ratelimit-Consumed": [
            "1"
        ],
        "X-Api-Ratelimit-Limit": [
            "100000"
        ],
        "X-Api-Ratelimit-Remaining": [
            "99992"
        ],
        "X-Api-Ratelimit-Reset": [
            "1704292200"
        ],
        "X-Content-Type-Options": [
            "nosniff"
        ],
        "X-Frame-Options": [
            "DENY"
        ]
    },
    "response_body": "{\"s\":\"ok\",\"date\":[1641013200,1641099600,1641186000,1641272400,1641358800,1641445200,1641531600,1641618000,1641704400,1641790800],\"status\":[\"closed\",\"closed\",\"open\",\"open\",\"open\",\"open\",\"open\",\"closed\",\"closed\",\"open\"]}"
}
```

### Debug Mode

The SDK provides a debug mode that can be enabled to help you understand how the SDK is working and troubleshoot any issues you might encounter. When debug mode is enabled, the SDK will print detailed information about each request and response to the console. This includes the full URL of the request, all request and response headers, and more.

To enable debug mode, you need to call the `Debug` method on the `MarketDataClient` instance and pass `true` as the argument. Here is an example:

```go
client.Debug(true)
```

Please note that the information printed in debug mode can be quite verbose. It is recommended to use this mode only when you are facing issues and need to understand what's happening under the hood. When debug mode is activated all requests are logged, not just requests that fail.

Debug mode can be particularly useful when you are first getting started with the SDK. It can help you understand how the SDK constructs requests, how it handles responses, and how it manages errors. By examining the debug output, you can gain a deeper understanding of the SDK's inner workings and be better prepared to handle any issues that might arise.