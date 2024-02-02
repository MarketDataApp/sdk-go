package main

import (
	"fmt"

	api "github.com/MarketDataApp/sdk-go/client"
)

func rawHttpResponseExample() {
	resp, err := api.StockQuote().Symbol("AAPL").Raw()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(resp)
}

func logExample() {
	_, err := api.IndexQuotes().Symbol("VIX").FiftyTwoWeek(true).Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(api.GetLogs())

}

func marketStatusExample() {

	msr, err := api.MarketStatus().From("2022-01-01").To("2022-01-10").Packed()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(msr)

}
