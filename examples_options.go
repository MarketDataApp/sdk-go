package main

import (
	"fmt"

	api "github.com/MarketDataApp/sdk-go/client"
)

func optionsChainExample() {
	resp, err := api.OptionChain().UnderlyingSymbol("AAPL").Side("call").DTE(60).StrikeLimit(2).Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	for _, contract := range resp {
		fmt.Println(contract)
	}

}

func optionsQuotesExample() {
	resp, err := api.OptionQuote().OptionSymbol("AAPL250117C00150000").Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(resp)

}

func optionsStrikesExample() {
	resp, err := api.OptionsStrikes().UnderlyingSymbol("AAPL").Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	for _, expiration := range resp {
		fmt.Println(expiration)
	}
}

func optionsLookupExample() {
	resp, err := api.OptionLookup().UserInput("AAPL 7/28/2023 200 Call").Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(resp)
}

func optionsExpirationsExample() {
	resp, err := api.OptionsExpirations().UnderlyingSymbol("AAPL").Strike(200).Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	for _, expirations := range resp {
		fmt.Println(expirations)
	}
}
