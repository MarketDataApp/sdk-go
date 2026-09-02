// want: undefined: options\.(WithStrikeRange|WithStrikeExpression)
//
// Strike selection collapses into one StrikeFilter; range and raw-expression
// can no longer be combined (both old options removed).
package main

import (
	"context"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
)

func main() {
	var s *options.Service
	_, _, _ = s.Chain(context.Background(), "AAPL",
		options.WithStrikeRange(140, 160),
		options.WithStrikeExpression(">=200"),
	)
}
