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
 | Stocks            | Candles      | ❌        |           |
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