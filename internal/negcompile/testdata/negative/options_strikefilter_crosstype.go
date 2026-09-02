// want: (does not implement|cannot use)
//
// A stocks DateWindow is not an options StrikeFilter: the sealed unions are
// distinct types, so a value from one package cannot be smuggled into another.
package main

import (
	"context"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

func main() {
	var s *options.Service
	_, _, _ = s.Chain(context.Background(), "AAPL",
		options.WithStrike(stocks.LastN(5)),
	)
}
