// want: (does not implement|missing method window)
//
// DateWindow is a sealed union: its only method is unexported, so third-party
// code cannot implement it and cannot smuggle an arbitrary/invalid date mode
// into the SDK. Only the package's own constructors produce a DateWindow.
package main

import "github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"

type forgedWindow struct{}

func main() {
	var _ stocks.DateWindow = forgedWindow{}
}
