// A legal candles call: the date range is a single DateWindow value. This must
// build.
package main

import (
	"context"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

func main() {
	var s *stocks.Service
	_, _, _ = s.Candles(context.Background(), "AAPL",
		stocks.WithResolution(stocks.ResolutionDaily),
		stocks.WithCandleWindow(stocks.Between(time.Now().AddDate(0, -1, 0), time.Now())),
	)
	_, _, _ = s.Candles(context.Background(), "AAPL",
		stocks.WithCandleWindow(stocks.LastN(30)),
	)
	_, _, _ = s.Candles(context.Background(), "AAPL",
		stocks.WithCandleWindow(stocks.OnDate(time.Now())),
	)
}
