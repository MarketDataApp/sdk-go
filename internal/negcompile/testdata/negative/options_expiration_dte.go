// want: undefined: options\.(WithExpiration|WithDTE)
//
// Chain expiry modes (expiration vs dte) are mutually exclusive; both old
// options were removed in favor of a single ExpiryFilter via WithExpiry.
package main

import (
	"context"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
)

func main() {
	var s *options.Service
	_, _, _ = s.Chain(context.Background(), "AAPL",
		options.WithExpiration(time.Now()),
		options.WithDTE(30),
	)
}
