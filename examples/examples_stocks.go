package examples

import (
	"fmt"

	api "github.com/MarketDataApp/sdk-go/client"
)

func StockQuoteExample() {
	sqe, err := api.StockQuote().Symbol("AAPL").FiftyTwoWeek(true).Packed()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println("Printing Stock Quote Response...")
	fmt.Println(sqe)

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

func StockCandlesExample() {

	sce, err := api.StockCandles().Resolution("1").Symbol("AAPL").From("2023-01-01").To("2023-01-04").Packed()
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

func StockEarningsExample() {
	see, err := api.StockEarnings().Symbol("AAPL").From("2022-01-01").To("2022-12-31").Packed()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println("Printing Earnings Response...")
	fmt.Println(see)

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

func StockNewsExample() {
	resp, err := api.StockNews().Symbol("AAPL").Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	for _, news := range resp {
		fmt.Println(news)
	}
}
