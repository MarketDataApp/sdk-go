package client

import "fmt"

func ExampleStockTickersRequestV2_get() {
	tickers, err := StockTickers().DateKey("2023-01-05").Get()
	if err != nil {
		fmt.Print(err)
		return
	}
	for _, ticker := range tickers {
		if ticker.Symbol == "AAPL" {
			fmt.Println(ticker)
			break
		}
	}
	// Output: Ticker{Symbol: AAPL, Name: Apple Inc., Type: CS, Currency: USD, Exchange: XNAS, FigiShares: BBG001S5N8V8, FigiComposite: BBG000B9XRY4, Cik: 0000320193, Updated: 2023-01-11 19:00:00 -05:00}
}
