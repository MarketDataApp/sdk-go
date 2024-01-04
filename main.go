package main

import (
	"fmt"
	"log"

	api "github.com/MarketDataApp/sdk-go/client"
)

func main() {

	//fmt.Println("Starting stocks tickers request...")
	//stocksTickersExample()
	client, err := api.GetClient()
	if err != nil {
		log.Fatalf("Failed to get client: %v", err)
	}

	client.Debug(true)

	fmt.Println("Starting index candles request...")
	indicesCandlesExample()

	//fmt.Println("Starting market status request...")
	//marketstatusExample()
}
