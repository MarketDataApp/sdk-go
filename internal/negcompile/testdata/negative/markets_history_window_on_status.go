// want: (does not implement|cannot use|applyStatus)
//
// A HistoryOption (range window) cannot be passed to Status, which takes a
// single date: the two methods have distinct, sealed option types.
package main

import (
	"context"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/markets"
)

func main() {
	var s *markets.Service
	_, _, _ = s.Status(context.Background(),
		markets.WithHistoryWindow(markets.LastN(5)),
	)
}
