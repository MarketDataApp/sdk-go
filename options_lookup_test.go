package client

import (
	"context"
	"fmt"
)

func ExampleOptionLookupRequest_get() {
	ctx := context.TODO()
	resp, err := OptionLookup().UserInput("AAPL 7/28/2023 200 Call").Get(ctx)
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(resp)
	// Output: AAPL230728C00200000
}

func ExampleOptionLookupRequest_packed() {
	ctx := context.TODO()
	resp, err := OptionLookup().UserInput("AAPL 7/28/2023 200 Call").Packed(ctx)
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(resp)
	// Output: OptionLookupResponse{OptionSymbol: "AAPL230728C00200000"}
}
