package examples

import (
	"context"
	"fmt"

	api "github.com/MarketDataApp/sdk-go"
)

func RawHttpResponseExample() {
	ctx := context.TODO()
	resp, err := api.StockQuote().Symbol("AAPL").Raw(ctx)
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(resp)
}

func LogExample() {
	ctx := context.TODO()
	_, err := api.IndexQuotes().Symbol("VIX").FiftyTwoWeek(true).Get(ctx)
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(api.GetLogs())

}

func MarketStatusExample() {
	ctx := context.TODO()
	msr, err := api.MarketStatus().From("2022-01-01").To("2022-01-10").Packed(ctx)
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(msr)

}
