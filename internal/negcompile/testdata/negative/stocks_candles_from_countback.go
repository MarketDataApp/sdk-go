// want: undefined: stocks\.(WithFrom|WithCountback)
//
// Candles date modes are mutually exclusive: from/to and countback cannot be
// combined. In the redesigned surface neither WithFrom nor WithCountback
// exists — the date range is a single DateWindow value — so the old footgun
// literally cannot be written.
package main

import (
	"context"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

func main() {
	var s *stocks.Service
	_, _, _ = s.Candles(context.Background(), "AAPL",
		stocks.WithFrom(time.Now()),
		stocks.WithCountback(5),
	)
}
