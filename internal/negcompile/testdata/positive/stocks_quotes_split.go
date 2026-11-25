// Legal quote calls after the Quote/Quotes option split: 52-week data and
// extended-hours on both methods, each through its own option type. This
// must build.
package main

import (
	"context"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

func main() {
	var s *stocks.Service
	_, _, _ = s.Quote(context.Background(), "AAPL",
		stocks.WithFiftyTwoWeek(true),
		stocks.WithExtended(true),
	)
	_, _, _ = s.Quotes(context.Background(), []string{"AAPL", "MSFT"},
		stocks.WithQuotesFiftyTwoWeek(true),
		stocks.WithQuotesExtended(true),
	)
}
