// want: undefined: stocks\.(WithEarningsFrom|WithEarningsCountback)
//
// Earnings date modes are mutually exclusive; the old WithEarningsFrom /
// WithEarningsCountback combination no longer exists — earnings takes a single
// DateWindow via WithEarningsWindow.
package main

import (
	"context"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

func main() {
	var s *stocks.Service
	_, _, _ = s.Earnings(context.Background(), "AAPL",
		stocks.WithEarningsFrom(time.Now()),
		stocks.WithEarningsCountback(4),
	)
}
