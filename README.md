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
 | Stocks            | Candles      | ❌        |     ✅    |
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

### Debug Mode

The SDK provides a debug mode that can be enabled to help you understand how the SDK is working and troubleshoot any issues you might encounter. When debug mode is enabled, the SDK will print detailed information about each request and response to the console. This includes the full URL of the request, all request and response headers, and more.

To enable debug mode, you need to call the `Debug` method on the `MarketDataClient` instance and pass `true` as the argument. Here is an example:

```go
client.Debug(true)
```

Please note that the information printed in debug mode can be quite verbose. It is recommended to use this mode only when you are facing issues and need to understand what's happening under the hood. Also, be aware that the output may contain sensitive information such as your API token, so be careful not to share the output publicly.

Debug mode can be particularly useful when you are first getting started with the SDK. It can help you understand how the SDK constructs requests, how it handles responses, and how it manages errors. By examining the debug output, you can gain a deeper understanding of the SDK's inner workings and be better prepared to handle any issues that might arise.