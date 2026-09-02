// Legal funds candles calls with a single DateWindow value. Must build.
package main

import (
	"context"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/funds"
)

func main() {
	var s *funds.Service
	_, _, _ = s.Candles(context.Background(), "VFINX",
		funds.WithResolution(funds.ResolutionDaily),
		funds.WithCandleWindow(funds.Between(time.Now().AddDate(0, -1, 0), time.Now())),
	)
	_, _, _ = s.Candles(context.Background(), "VFINX",
		funds.WithCandleWindow(funds.OnDate(time.Now())),
	)
}
