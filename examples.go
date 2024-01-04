package main

import (
	"fmt"
	"time"

	api "github.com/MarketDataApp/sdk-go/client"
)

func indicesCandlesExample() {

	oneWeekAgo := time.Now().AddDate(0, 0, -7)
	ice, _, err := api.IndexCandles().Resolution("1").Symbol("VIX").From(oneWeekAgo).To("today").Get()
	if err != nil {
		fmt.Print(err)
		return
	}
	
	candles, err := ice.Unpack()
	if err != nil {
		fmt.Print(err)
		return
	}

	for _, candle := range candles {
		fmt.Println(candle)
	}
	
}

func stockCandlesExample() {

	sce, _, err := api.StockCandles().Resolution("1").Symbol("AAPL").From("2023-01-01").To("2023-01-04").Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	candles, err := sce.Unpack()
	if err != nil {
		fmt.Print(err)
		return
	}

	for _, candle := range candles {
		fmt.Println(candle)
	}

}

func marketStatusExample() {

	msr, _, err := api.MarketStatus().From("2022-01-01").To("2022-01-10").Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(msr)

}
