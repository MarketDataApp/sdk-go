package main

import (
	"fmt"
	"time"

	api "github.com/MarketDataApp/sdk-go/client"
)


func stockEarningsExample() {
	see, raw, err := api.StockEarnings().Symbol("AAPL").From("2022-01-01").To("2022-12-31").Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println("Printing Earnings Response...")
	fmt.Println(see)
	fmt.Println("Printing Raw JSON Response...")
	fmt.Println(raw)

	unpacked, err := see.Unpack()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println("Priting Unpacked Earnings Reports...")
	for _, report := range unpacked {
		fmt.Println(report)
	}
}

func stockQuoteExample() {
	sqe, raw, err := api.StockQuotes().Symbol("AAPL").FiftyTwoWeek(true).Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println("Printing Stock Quote Response...")
	fmt.Println(sqe)
	fmt.Println("Printing Raw JSON Response...")
	fmt.Println(raw)

	unpacked, err := sqe.Unpack()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println("Printing Unpacked Stock Quotes...")
	for _, quote := range unpacked {
		fmt.Println(quote)
	}
}

func indexCandlesExample() {
	oneWeekAgo := time.Now().AddDate(0, 0, -7)
	ice, raw, err := api.IndexCandles().Resolution("D").Symbol("VIX").From(oneWeekAgo).To("today").Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println("Printing Index Candles Response...")
	fmt.Println(ice)
	fmt.Println("Printing Raw JSON Response...")
	fmt.Println(raw)

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
