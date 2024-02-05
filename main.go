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

	client.Debug(false)

}

/*

	fmt.Println("Staring log example...")
	logExample()

	//fmt.Println("Starting market status request...")
	//marketstatusExample()

	fmt.Println("Staring rawResponse example...")
	rawHttpResponseExample()

*/

func optionsExamples() {
	fmt.Println("Staring Options/Chain example...")
	optionsChainExample()

	fmt.Println("Staring Options/Strikes example...")
	optionsStrikesExample()

	fmt.Println("Staring Options/Quotes example...")
	optionsQuotesExample()

	fmt.Println("Staring Options/Lookup example...")
	optionsLookupExample()

	fmt.Println("Staring Options/Expirations example...")
	optionsExpirationsExample()

}

func stocksExamples() {
	fmt.Println("Staring Stocks/News example...")
	stockNewsExample()

	fmt.Println("Starting stock earnings request...")
	stockEarningsExample()

	fmt.Println("Starting stock quote request...")
	stockQuoteExample()

	fmt.Println("Starting stock candles request...")
	stockCandlesExample()

}

func indexExamples() {
	fmt.Println("Starting index quote request...")
	indexQuoteExample()

	fmt.Println("Starting index candles request...")
	indexCandlesExample()

}
