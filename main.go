package main

import (
	"fmt"
	"log"

	api "github.com/MarketDataApp/sdk-go/client"
)

func main() {
	
	client, err := api.GetClient()
	if err != nil {
		log.Fatalf("Failed to get client: %v", err)
	}

	client.Debug(true)

	fmt.Println("Starting index quote request...")
	indexQuoteExample()

	/*

	fmt.Println("Starting stock earnings request...")
	stockEarningsExample()

	fmt.Println("Starting stock quote request...")
	stockQuoteExample()

	fmt.Println("Starting index candles request...")
	indexCandlesExample()

	fmt.Println("Starting stock candles request...")
	stockCandlesExample()

	//fmt.Println("Starting market status request...")
	//marketstatusExample()

	*/
}
