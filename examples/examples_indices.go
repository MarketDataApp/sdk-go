package examples

import (
	"context"
	"fmt"
	"time"

	api "github.com/MarketDataApp/sdk-go"
)

func IndexQuoteExample() {
	ctx := context.TODO()
	iqe, err := api.IndexQuotes().Symbol("VIX").FiftyTwoWeek(true).Packed(ctx)
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

func IndexCandlesExample() {
	oneWeekAgo := time.Now().AddDate(0, 0, -7)
	ctx := context.TODO()
	ice, err := api.IndexCandles().Resolution("D").Symbol("VIX").From(oneWeekAgo).To("today").Packed(ctx)
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
