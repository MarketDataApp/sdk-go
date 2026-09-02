// want: (cannot use|does not implement|QuotesOption)
//
// 52-week high/low data is a single-quote capability: the bulk quotes API
// endpoint ignores the 52week parameter, so the SDK only accepts
// WithFiftyTwoWeek on Quote. Passing it to Quotes (bulk) — which would
// silently return zero values — must not compile.
package main

import (
	"context"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

func main() {
	var s *stocks.Service
	_, _, _ = s.Quotes(context.Background(), []string{"AAPL", "MSFT"},
		stocks.WithFiftyTwoWeek(true),
	)
}
