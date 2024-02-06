package examples

import (
	"fmt"
	"log"

	api "github.com/MarketDataApp/sdk-go"
)

func Start() {

	client, err := api.GetClient()
	if err != nil {
		log.Fatalf("Failed to get client: %v", err)
	}

	client.Debug(false)

	fmt.Println("Staring log example...")
	LogExample()
	/*
	   //fmt.Println("Starting market status request...")
	   //marketstatusExample()

	   fmt.Println("Staring rawResponse example...")
	   rawHttpResponseExample()
	*/
}

func optionsExamples() {
	fmt.Println("Staring Options/Chain example...")
	OptionsChainExample()

	fmt.Println("Staring Options/Strikes example...")
	OptionsStrikesExample()

	fmt.Println("Staring Options/Quotes example...")
	OptionsQuotesExample()

	fmt.Println("Staring Options/Lookup example...")
	OptionsLookupExample()

	fmt.Println("Staring Options/Expirations example...")
	OptionsExpirationsExample()

}

func stocksExamples() {
	fmt.Println("Staring Stocks/News example...")
	StockNewsExample()

	fmt.Println("Starting stock earnings request...")
	StockEarningsExample()

	fmt.Println("Starting stock quote request...")
	StockQuoteExample()

	fmt.Println("Starting stock candles request...")
	StockCandlesExample()

}

func indexExamples() {
	fmt.Println("Starting index quote request...")
	IndexQuoteExample()

	fmt.Println("Starting index candles request...")
	IndexCandlesExample()

}
