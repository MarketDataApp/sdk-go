package client

import "fmt"

func ExampleStockCandlesRequest_raw() {
	scr, err := StockCandles().Resolution("4H").Symbol("AAPL").From("2023-01-01").To("2023-01-04").Raw()
	if err != nil {
		fmt.Print(err)
		return
	}
	fmt.Println(scr)

	// Output: {"s":"ok","t":[1672756200,1672770600,1672842600,1672857000],"o":[130.28,124.6699,126.89,127.265],"h":[130.9,125.42,128.6557,127.87],"l":[124.19,124.17,125.08,125.28],"c":[124.6499,125.05,127.2601,126.38],"v":[64411753,30727802,49976607,28870878]}
}

func ExampleStockCandlesRequest_packed() {
	scr, err := StockCandles().Resolution("4H").Symbol("AAPL").From("2023-01-01").To("2023-01-04").Packed()
	if err != nil {
		fmt.Print(err)
		return
	}
	fmt.Println(scr)

	// Output: StockCandlesResponse{Date: [1672756200 1672770600 1672842600 1672857000], Open: [130.28 124.6699 126.89 127.265], High: [130.9 125.42 128.6557 127.87], Low: [124.19 124.17 125.08 125.28], Close: [124.6499 125.05 127.2601 126.38], Volume: [64411753 30727802 49976607 28870878]}
}

func ExampleStockCandlesRequest_get() {
	scr, err := StockCandles().Resolution("4H").Symbol("AAPL").From("2023-01-01").To("2023-01-04").Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	for _, candle := range scr {
		fmt.Println(candle)
	}
	// Output: Candle{Time: 2023-01-03 09:30:00 -05:00, Open: 130.28, High: 130.9, Low: 124.19, Close: 124.6499, Volume: 64411753}
	// Candle{Time: 2023-01-03 13:30:00 -05:00, Open: 124.6699, High: 125.42, Low: 124.17, Close: 125.05, Volume: 30727802}
	// Candle{Time: 2023-01-04 09:30:00 -05:00, Open: 126.89, High: 128.6557, Low: 125.08, Close: 127.2601, Volume: 49976607}
	// Candle{Time: 2023-01-04 13:30:00 -05:00, Open: 127.265, High: 127.87, Low: 125.28, Close: 126.38, Volume: 28870878}

}