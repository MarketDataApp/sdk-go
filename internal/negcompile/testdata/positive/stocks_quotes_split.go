// Legal quote calls after the Quote/Quotes option split: 52-week data on the
// single-quote endpoint (where the API honors it) and extended-hours on both
// endpoints via their respective option types. This must build.
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
		stocks.WithQuotesExtended(true),
	)
}
