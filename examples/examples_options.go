package examples

import (
	"context"
	"fmt"

	api "github.com/MarketDataApp/sdk-go"
)

func OptionsChainExample() {
	ctx := context.TODO()
	resp, err := api.OptionChain().UnderlyingSymbol("AAPL").Side("call").DTE(60).StrikeLimit(2).Get(ctx)
	if err != nil {
		fmt.Print(err)
		return
	}

	for _, contract := range resp {
		fmt.Println(contract)
	}

}

func OptionsQuotesExample() {
	ctx := context.TODO()
	resp, err := api.OptionQuote().OptionSymbol("AAPL250117C00150000").Get(ctx)
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(resp)

}

func OptionStrikesExample() {
	ctx := context.TODO()
	resp, err := api.OptionStrikes().UnderlyingSymbol("AAPL").Get(ctx)
	if err != nil {
		fmt.Print(err)
		return
	}

	for _, expiration := range resp {
		fmt.Println(expiration)
	}
}

func OptionsLookupExample() {
	ctx := context.TODO()
	resp, err := api.OptionLookup().UserInput("AAPL 7/28/2023 200 Call").Get(ctx)
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(resp)
}

func OptionsExpirationsExample() {
	ctx := context.TODO()
	resp, err := api.OptionsExpirations().UnderlyingSymbol("AAPL").Strike(200).Get(ctx)
	if err != nil {
		fmt.Print(err)
		return
	}

	for _, expirations := range resp {
		fmt.Println(expirations)
	}
}
