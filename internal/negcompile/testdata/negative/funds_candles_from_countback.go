// want: undefined: funds\.(WithFrom|WithCountback)
//
// Funds candles date modes are mutually exclusive; WithFrom/WithCountback were
// removed in favor of a single DateWindow via WithCandleWindow.
package main

import (
	"context"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/funds"
)

func main() {
	var s *funds.Service
	_, _, _ = s.Candles(context.Background(), "VFINX",
		funds.WithFrom(time.Now()),
		funds.WithCountback(10),
	)
}
