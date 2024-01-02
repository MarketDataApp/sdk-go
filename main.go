package main

import (
	"fmt"
	api "github.com/MarketDataApp/sdk-go/client"
)

func main() {

	//fmt.Println("Starting stocks tickers request...")
	//stocksTickersExample()
	client, _ := api.GetClient()
	client.Debug(true)

	fmt.Println("Starting stocks candles request...")
	stockCandlesExample()

	//fmt.Println("Starting market status request...")
	//marketstatusExample()
}
