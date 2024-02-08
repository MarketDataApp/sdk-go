package client

import "fmt"

func ExampleBulkStockCandlesRequest_packed() {
	symbols := []string{"AAPL", "META", "MSFT"}
	bscr, err := BulkStockCandles().Resolution("D").Symbols(symbols).Date("2024-02-06").Packed()
	if err != nil {
		fmt.Print(err)
		return
	}
	fmt.Println(bscr)
	// Output: BulkStockCandlesResponse{Symbol: [AAPL META MSFT], Date: [1707195600 1707195600 1707195600], Open: [186.86 464 405.88], High: [189.31 467.12 407.97], Low: [186.7695 453 402.91], Close: [189.3 454.72 405.49], Volume: [43490759 21653114 18382624]}
}

func ExampleBulkStockCandlesRequest_get() {
	symbols := []string{"AAPL", "META", "MSFT"}
	bscr, err := BulkStockCandles().Resolution("D").Symbols(symbols).Date("2024-02-06").Get()
	if err != nil {
		fmt.Print(err)
		return
	}
	for _, candle := range bscr {
		fmt.Println(candle)
	}
	// Output: Candle{Symbol: AAPL, Date: 2024-02-06, Open: 186.86, High: 189.31, Low: 186.7695, Close: 189.3, Volume: 43490759}
	// Candle{Symbol: META, Date: 2024-02-06, Open: 464, High: 467.12, Low: 453, Close: 454.72, Volume: 21653114}
	// Candle{Symbol: MSFT, Date: 2024-02-06, Open: 405.88, High: 407.97, Low: 402.91, Close: 405.49, Volume: 18382624}

}
