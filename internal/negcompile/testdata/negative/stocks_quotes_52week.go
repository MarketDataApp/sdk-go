// want: (cannot use|does not implement|QuotesOption)
//
// QuoteOption and QuotesOption are distinct types: the single-quote
// WithFiftyTwoWeek must not compile as an option to Quotes, which takes
// WithQuotesFiftyTwoWeek instead.
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
