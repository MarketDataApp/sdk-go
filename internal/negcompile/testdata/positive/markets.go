// Legal markets calls: WithCountry works on both methods; WithDate only on
// Status; WithHistoryWindow only on StatusHistory. Builds.
package main

import (
	"context"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/markets"
)

func main() {
	var s *markets.Service
	_, _, _ = s.Status(context.Background(), markets.WithDate(time.Now()), markets.WithCountry("US"))
	_, _, _ = s.StatusHistory(context.Background(), markets.WithHistoryWindow(markets.LastNUntil(5, time.Now())), markets.WithCountry("US"))
}
