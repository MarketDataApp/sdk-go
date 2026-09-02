// want: (does not implement|cannot use|applyHistory)
//
// WithDate (a Status-only concept) cannot be passed to StatusHistory.
package main

import (
	"context"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/markets"
)

func main() {
	var s *markets.Service
	_, _, _ = s.StatusHistory(context.Background(),
		markets.WithDate(time.Now()),
	)
}
