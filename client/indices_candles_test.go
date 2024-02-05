package client

import "fmt"

func ExampleIndicesCandlesRequest_get() {
	vix, err := IndexCandles().Symbol("VIX").Resolution("D").From("2022-01-01").To("2022-01-05").Get()
	if err != nil {
		println("Error retrieving VIX index candles:", err.Error())
		return
	}

	for _, candle := range vix {
		fmt.Println(candle)
	}
	// Output: Time: 2022-01-03 00:00:00 -05:00, Open: 17.6, High: 18.54, Low: 16.56, Close: 16.6
	// Time: 2022-01-04 00:00:00 -05:00, Open: 16.57, High: 17.81, Low: 16.34, Close: 16.91
	// Time: 2022-01-05 00:00:00 -05:00, Open: 17.07, High: 20.17, Low: 16.58, Close: 19.73
}

func ExampleIndicesCandlesRequest_packed() {
	vix, err := IndexCandles().Symbol("VIX").Resolution("D").From("2022-01-01").To("2022-01-05").Packed()
	if err != nil {
		println("Error retrieving VIX index candles:", err.Error())
		return
	}
	fmt.Println(vix)
	// Output: Time: [1641186000 1641272400 1641358800], Open: [17.6 16.57 17.07], High: [18.54 17.81 20.17], Low: [16.56 16.34 16.58], Close: [16.6 16.91 19.73]
}

func ExampleIndicesCandlesRequest_raw() {
	vix, err := IndexCandles().Symbol("VIX").Resolution("D").From("2022-01-01").To("2022-01-05").Raw()
	if err != nil {
		println("Error retrieving VIX index candles:", err.Error())
		return
	}
	fmt.Println(vix)
	// Output: {"s":"ok","t":[1641186000,1641272400,1641358800],"o":[17.6,16.57,17.07],"h":[18.54,17.81,20.17],"l":[16.56,16.34,16.58],"c":[16.6,16.91,19.73]}

}