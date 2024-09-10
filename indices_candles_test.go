package client

import (
	"context"
	"fmt"
)

func ExampleIndicesCandlesRequest_get() {
	ctx := context.TODO()
	vix, err := IndexCandles().Symbol("VIX").Resolution("D").From("2024-01-01").To("2024-01-05").Get(ctx)
	if err != nil {
		println("Error retrieving VIX index candles:", err.Error())
		return
	}

	for _, candle := range vix {
		fmt.Println(candle)
	}
	// Output: Candle{Date: 2024-01-02, Open: 13.21, High: 14.23, Low: 13.1, Close: 13.2}
	// Candle{Date: 2024-01-03, Open: 13.38, High: 14.22, Low: 13.36, Close: 14.04}
	// Candle{Date: 2024-01-04, Open: 13.97, High: 14.2, Low: 13.64, Close: 14.13}
	// Candle{Date: 2024-01-05, Open: 14.24, High: 14.58, Low: 13.29, Close: 13.35}
}

func ExampleIndicesCandlesRequest_packed() {
	ctx := context.TODO()
	vix, err := IndexCandles().Symbol("VIX").Resolution("D").From("2024-01-01").To("2024-01-05").Packed(ctx)
	if err != nil {
		println("Error retrieving VIX index candles:", err.Error())
		return
	}
	fmt.Println(vix)
	// Output: IndicesCandlesResponse{Time: [1704171600 1704258000 1704344400 1704430800], Open: [13.21 13.38 13.97 14.24], High: [14.23 14.22 14.2 14.58], Low: [13.1 13.36 13.64 13.29], Close: [13.2 14.04 14.13 13.35]}
}

func ExampleIndicesCandlesRequest_raw() {
	ctx := context.TODO()
	vix, err := IndexCandles().Symbol("VIX").Resolution("D").From("2024-01-01").To("2024-01-05").Raw(ctx)
	if err != nil {
		println("Error retrieving VIX index candles:", err.Error())
		return
	}
	fmt.Println(vix)
	// Output: {"s":"ok","t":[1704171600,1704258000,1704344400,1704430800],"o":[13.21,13.38,13.97,14.24],"h":[14.23,14.22,14.2,14.58],"l":[13.1,13.36,13.64,13.29],"c":[13.2,14.04,14.13,13.35]}

}
