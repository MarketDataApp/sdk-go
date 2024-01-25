package main

import (
	"fmt"
	"time"

	api "github.com/MarketDataApp/sdk-go/client"
)

func indexQuoteExample() {
	iqe, err := api.IndexQuotes().Symbol("VIX").FiftyTwoWeek(true).Packed()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println("Printing Index Quote Response...")
	fmt.Println(iqe)

	unpacked, err := iqe.Unpack()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println("Printing Unpacked Index Quotes...")
	for _, quote := range unpacked {
		fmt.Println(quote)
	}

}

func indexCandlesExample() {
	oneWeekAgo := time.Now().AddDate(0, 0, -7)
	ice, err := api.IndexCandles().Resolution("D").Symbol("VIX").From(oneWeekAgo).To("today").Packed()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println("Printing Index Candles Response...")
	fmt.Println(ice)

	unpacked, err := ice.Unpack()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println("Printing Unpacked Index Candles...")
	for _, candle := range unpacked {
		fmt.Println(candle)
	}
}
