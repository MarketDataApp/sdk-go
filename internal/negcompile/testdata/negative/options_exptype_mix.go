// want: undefined: options\.(WithWeekly|WithMonthly)
//
// The API forbids mixing inclusion and exclusion of expiration types
// (weekly=true&monthly=false is an error). The independent WithWeekly/WithMonthly
// bool options were removed in favor of a single include-or-exclude
// ExpirationTypeFilter, so the illegal mix cannot be written.
package main

import (
	"context"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
)

func main() {
	var s *options.Service
	_, _, _ = s.Chain(context.Background(), "AAPL",
		options.WithWeekly(true),
		options.WithMonthly(false),
	)
}
