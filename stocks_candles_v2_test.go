package client

import "fmt"

func ExampleStockCandlesRequestV2_get() {

	scr2, err := StockCandlesV2().Resolution("D").Symbol("AAPL").DateKey("2023-01").Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	for _, candle := range scr2 {
		fmt.Println(candle)
	}
	// Output: Candle{Date: 2023-01-03, Open: 130.28, High: 130.9, Low: 124.17, Close: 125.07, Volume: 112117471, VWAP: 125.725, N: 1021065}
	// Candle{Date: 2023-01-04, Open: 126.89, High: 128.6557, Low: 125.08, Close: 126.36, Volume: 89100633, VWAP: 126.6464, N: 770042}
	// Candle{Date: 2023-01-05, Open: 127.13, High: 127.77, Low: 124.76, Close: 125.02, Volume: 80716808, VWAP: 126.0883, N: 665458}
	// Candle{Date: 2023-01-06, Open: 126.01, High: 130.29, Low: 124.89, Close: 129.62, Volume: 87754715, VWAP: 128.1982, N: 711520}
	// Candle{Date: 2023-01-09, Open: 130.465, High: 133.41, Low: 129.89, Close: 130.15, Volume: 70790813, VWAP: 131.6292, N: 645365}
	// Candle{Date: 2023-01-10, Open: 130.26, High: 131.2636, Low: 128.12, Close: 130.73, Volume: 63896155, VWAP: 129.822, N: 554940}
	// Candle{Date: 2023-01-11, Open: 131.25, High: 133.51, Low: 130.46, Close: 133.49, Volume: 69458949, VWAP: 132.3081, N: 561278}
	// Candle{Date: 2023-01-12, Open: 133.88, High: 134.26, Low: 131.44, Close: 133.41, Volume: 71379648, VWAP: 133.171, N: 635331}
	// Candle{Date: 2023-01-13, Open: 132.03, High: 134.92, Low: 131.66, Close: 134.76, Volume: 57809719, VWAP: 133.6773, N: 537385}
	// Candle{Date: 2023-01-17, Open: 134.83, High: 137.29, Low: 134.13, Close: 135.94, Volume: 63612627, VWAP: 135.7587, N: 595831}
	// Candle{Date: 2023-01-18, Open: 136.815, High: 138.61, Low: 135.03, Close: 135.21, Volume: 69672800, VWAP: 136.3316, N: 578304}
	// Candle{Date: 2023-01-19, Open: 134.08, High: 136.25, Low: 133.77, Close: 135.27, Volume: 58280413, VWAP: 134.9653, N: 491674}
	// Candle{Date: 2023-01-20, Open: 135.28, High: 138.02, Low: 134.22, Close: 137.87, Volume: 80200655, VWAP: 136.3762, N: 552230}
	// Candle{Date: 2023-01-23, Open: 138.12, High: 143.315, Low: 137.9, Close: 141.11, Volume: 81760313, VWAP: 141.2116, N: 719288}
	// Candle{Date: 2023-01-24, Open: 140.305, High: 143.16, Low: 140.3, Close: 142.53, Volume: 66435142, VWAP: 142.0507, N: 498679}
	// Candle{Date: 2023-01-25, Open: 140.89, High: 142.43, Low: 138.81, Close: 141.86, Volume: 65799349, VWAP: 140.7526, N: 536505}
	// Candle{Date: 2023-01-26, Open: 143.17, High: 144.25, Low: 141.9, Close: 143.96, Volume: 54105068, VWAP: 143.3429, N: 472135}
	// Candle{Date: 2023-01-27, Open: 143.155, High: 147.23, Low: 143.08, Close: 145.93, Volume: 70547743, VWAP: 145.8365, N: 560022}
	// Candle{Date: 2023-01-30, Open: 144.955, High: 145.55, Low: 142.85, Close: 143, Volume: 64015274, VWAP: 143.6524, N: 551111}
	// Candle{Date: 2023-01-31, Open: 142.7, High: 144.34, Low: 142.28, Close: 144.29, Volume: 65874459, VWAP: 143.6473, N: 468170}
}

func ExampleStockCandlesRequestV2_packed() {

	scr2, err := StockCandlesV2().Resolution("D").Symbol("AAPL").DateKey("2023-01").Packed()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(scr2)
	// Output: StockCandlesResponse{Date: [1672722000 1672808400 1672894800 1672981200 1673240400 1673326800 1673413200 1673499600 1673586000 1673931600 1674018000 1674104400 1674190800 1674450000 1674536400 1674622800 1674709200 1674795600 1675054800 1675141200], Open: [130.28 126.89 127.13 126.01 130.465 130.26 131.25 133.88 132.03 134.83 136.815 134.08 135.28 138.12 140.305 140.89 143.17 143.155 144.955 142.7], High: [130.9 128.6557 127.77 130.29 133.41 131.2636 133.51 134.26 134.92 137.29 138.61 136.25 138.02 143.315 143.16 142.43 144.25 147.23 145.55 144.34], Low: [124.17 125.08 124.76 124.89 129.89 128.12 130.46 131.44 131.66 134.13 135.03 133.77 134.22 137.9 140.3 138.81 141.9 143.08 142.85 142.28], Close: [125.07 126.36 125.02 129.62 130.15 130.73 133.49 133.41 134.76 135.94 135.21 135.27 137.87 141.11 142.53 141.86 143.96 145.93 143 144.29], Volume: [112117471 89100633 80716808 87754715 70790813 63896155 69458949 71379648 57809719 63612627 69672800 58280413 80200655 81760313 66435142 65799349 54105068 70547743 64015274 65874459], VWAP: [125.725 126.6464 126.0883 128.1982 131.6292 129.822 132.3081 133.171 133.6773 135.7587 136.3316 134.9653 136.3762 141.2116 142.0507 140.7526 143.3429 145.8365 143.6524 143.6473], N: [1021065 770042 665458 711520 645365 554940 561278 635331 537385 595831 578304 491674 552230 719288 498679 536505 472135 560022 551111 468170]}
}
