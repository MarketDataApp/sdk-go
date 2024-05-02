package client

import "fmt"

func ExampleFundCandlesRequest_raw() {
	fcr, err := FundCandles().Resolution("D").Symbol("VFINX").From("2023-01-01").To("2023-01-06").Raw()
	if err != nil {
		fmt.Print(err)
		return
	}
	fmt.Println(fcr)

	// Output: {"s":"ok","t":[1672722000,1672808400,1672894800,1672981200],"o":[352.76,355.43,351.35,359.38],"h":[352.76,355.43,351.35,359.38],"l":[352.76,355.43,351.35,359.38],"c":[352.76,355.43,351.35,359.38]}
}

func ExampleFundCandlesRequest_packed() {
	fcr, err := FundCandles().Resolution("D").Symbol("VFINX").From("2023-01-01").To("2023-01-06").Packed()
	if err != nil {
		fmt.Print(err)
		return
	}
	fmt.Println(fcr)

	// Output: FundCandlesResponse{Date: [1672722000 1672808400 1672894800 1672981200], Open: [352.76 355.43 351.35 359.38], High: [352.76 355.43 351.35 359.38], Low: [352.76 355.43 351.35 359.38], Close: [352.76 355.43 351.35 359.38]}
}

func ExampleFundCandlesRequest_get() {
	fcr, err := FundCandles().Resolution("D").Symbol("VFINX").From("2023-01-01").To("2023-01-06").Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	for _, candle := range fcr {
		fmt.Println(candle)
	}
	// Output: Candle{Date: 2023-01-03, Open: 352.76, High: 352.76, Low: 352.76, Close: 352.76}
	// Candle{Date: 2023-01-04, Open: 355.43, High: 355.43, Low: 355.43, Close: 355.43}
	// Candle{Date: 2023-01-05, Open: 351.35, High: 351.35, Low: 351.35, Close: 351.35}
	// Candle{Date: 2023-01-06, Open: 359.38, High: 359.38, Low: 359.38, Close: 359.38}
}
