// Legal chain call: one value per exclusivity group plus free filters. Builds.
package main

import (
	"context"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
)

func main() {
	var s *options.Service
	_, _, _ = s.Chain(context.Background(), "AAPL",
		options.WithExpiry(options.OnExpiration(time.Now())),
		options.WithStrike(options.StrikeRange(140, 160)),
		options.WithChainDate(time.Now()),
		options.WithSide(options.SideCall),
	)
	_, _, _ = s.Chain(context.Background(), "AAPL",
		options.WithExpiry(options.InDTE(30)),
		options.WithStrike(options.ByDelta(-0.30)),
	)
	// Include-or-exclude expiration types is a single sealed value.
	_, _, _ = s.Chain(context.Background(), "AAPL",
		options.WithExpirationTypes(options.IncludeExpirationTypes(options.Weekly, options.Monthly)),
	)
	_, _, _ = s.Chain(context.Background(), "AAPL",
		options.WithExpirationTypes(options.ExcludeExpirationTypes(options.Quarterly)),
	)
}
