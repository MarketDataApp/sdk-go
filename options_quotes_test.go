package client

import (
	"context"
	"fmt"
)

func ExampleOptionQuoteRequest_get() {
	ctx := context.TODO()
	resp, err := OptionQuote().OptionSymbol("AAPL250117C00150000").Date("2024-02-05").Get(ctx)
	if err != nil {
		fmt.Print(err)
		return
	}

	for _, quote := range resp {
		fmt.Println(quote)
	}
	// Output: OptionQuote{OptionSymbol: "AAPL250117C00150000", Underlying: "AAPL", Expiration: 2025-01-17 16:00:00 -05:00, Side: "call", Strike: 150, FirstTraded: 2022-09-12 09:30:00 -04:00, DTE: 347, Ask: 47.7, AskSize: 17, Bid: 47.2, BidSize: 36, Mid: 47.45, Last: 48.65, Volume: 202, OpenInterest: 10768, UnderlyingPrice: 187.68, InTheMoney: true, Updated: "2024-02-05 16:00:00 -05:00", IV: nil, Delta: nil, Gamma: nil, Theta: nil, Vega: nil, Rho: nil, IntrinsicValue: 37.68, ExtrinsicValue: 9.77}
}

func ExampleOptionQuoteRequest_packed() {
	ctx := context.TODO()
	resp, err := OptionQuote().OptionSymbol("AAPL250117P00150000").Date("2024-02-05").Packed(ctx)
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(resp)
	// Output: OptionQuotesResponse{OptionSymbol: ["AAPL250117P00150000"], Underlying: ["AAPL"], Expiration: [1737147600], Side: ["put"], Strike: [150], FirstTraded: [1662989400], DTE: [347], Ask: [3.65], AskSize: [292], Bid: [3.5], BidSize: [634], Mid: [3.58], Last: [3.55], Volume: [44], OpenInterest: [18027], UnderlyingPrice: [187.68], InTheMoney: [false], Updated: [1707166800], IV: [nil], Delta: [nil], Gamma: [nil], Theta: [nil], Vega: [nil], Rho: [nil], IntrinsicValue: [0], ExtrinsicValue: [3.58]}
}

func ExampleOptionQuoteRequest_raw() {
	ctx := context.TODO()
	resp, err := OptionQuote().OptionSymbol("AAPL250117P00150000").Date("2024-02-05").Raw(ctx)
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(resp)
	// Output: {"s":"ok","optionSymbol":["AAPL250117P00150000"],"underlying":["AAPL"],"expiration":[1737147600],"side":["put"],"strike":[150.0],"firstTraded":[1662989400],"dte":[347],"updated":[1707166800],"bid":[3.5],"bidSize":[634],"mid":[3.58],"ask":[3.65],"askSize":[292],"last":[3.55],"openInterest":[18027],"volume":[44],"inTheMoney":[false],"intrinsicValue":[0.0],"extrinsicValue":[3.58],"underlyingPrice":[187.68],"iv":[null],"delta":[null],"gamma":[null],"theta":[null],"vega":[null],"rho":[null]}
}
