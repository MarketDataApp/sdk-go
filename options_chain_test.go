package client

import (
	"context"
	"fmt"
	"testing"
)

func TestOptionChainRequest(t *testing.T) {
	ctx := context.TODO()
	resp, err := OptionChain().UnderlyingSymbol("AAPL").Side("call").DTE(60).StrikeLimit(2).Range("itm").Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get option chain: %v", err)
	}

	if len(resp) < 2 {
		t.Fatalf("Expected at least 2 slices, got %d", len(resp))
	}

	for _, contract := range resp {
		if contract.Underlying != "AAPL" {
			t.Errorf("Expected underlying symbol to be AAPL, got %s", contract.Underlying)
		}
		if !contract.InTheMoney {
			t.Errorf("Expected contract to be in the money, but it was not. Contract: %+v", contract)
		}
	}
}

func ExampleOptionChainRequest_packed() {
	ctx := context.TODO()
	resp, err := OptionChain().UnderlyingSymbol("AAPL").Side("call").Date("2022-01-03").
		Month(2).Year(2022).Range("itm").Strike(150).Weekly(false).Monthly(true).Nonstandard(false).Packed(ctx)
	if err != nil {
		fmt.Println("Error fetching packed option chain:", err)
		return
	}
	fmt.Println(resp)
	// Output: OptionQuotesResponse{OptionSymbol: ["AAPL220121C00150000"], Underlying: ["AAPL"], Expiration: [1642798800], Side: ["call"], Strike: [150], FirstTraded: [1568640600], DTE: [18], Ask: [32.15], AskSize: [2], Bid: [31.8], BidSize: [359], Mid: [31.98], Last: [32], Volume: [3763], OpenInterest: [98804], UnderlyingPrice: [182.01], InTheMoney: [true], Updated: [1641243600], IV: [nil], Delta: [nil], Gamma: [nil], Theta: [nil], Vega: [nil], Rho: [nil], IntrinsicValue: [32.01], ExtrinsicValue: [0.03]}
}

func ExampleOptionChainRequest_get() {
	ctx := context.TODO()
	resp, err := OptionChain().UnderlyingSymbol("AAPL").Side("call").Date("2022-01-03").DTE(60).StrikeLimit(2).Range("itm").Get(ctx)
	if err != nil {
		fmt.Println("Error fetching option chain:", err)
		return
	}
	for _, contract := range resp {
		fmt.Println(contract)
	}
	// Output: OptionQuote{OptionSymbol: "AAPL220318C00175000", Underlying: "AAPL", Expiration: 2022-03-18 16:00:00 -04:00, Side: "call", Strike: 175, FirstTraded: 2021-07-13 09:30:00 -04:00, DTE: 74, Ask: 13.1, AskSize: 2, Bid: 12.95, BidSize: 3, Mid: 13.02, Last: 12.9, Volume: 1295, OpenInterest: 15232, UnderlyingPrice: 182.01, InTheMoney: true, Updated: "2022-01-03 16:00:00 -05:00", IV: nil, Delta: nil, Gamma: nil, Theta: nil, Vega: nil, Rho: nil, IntrinsicValue: 7.01, ExtrinsicValue: 6.02}
	// OptionQuote{OptionSymbol: "AAPL220318C00180000", Underlying: "AAPL", Expiration: 2022-03-18 16:00:00 -04:00, Side: "call", Strike: 180, FirstTraded: 2021-07-13 09:30:00 -04:00, DTE: 74, Ask: 10.2, AskSize: 12, Bid: 10, BidSize: 38, Mid: 10.1, Last: 10.1, Volume: 4609, OpenInterest: 18299, UnderlyingPrice: 182.01, InTheMoney: true, Updated: "2022-01-03 16:00:00 -05:00", IV: nil, Delta: nil, Gamma: nil, Theta: nil, Vega: nil, Rho: nil, IntrinsicValue: 2.01, ExtrinsicValue: 8.09}
}
