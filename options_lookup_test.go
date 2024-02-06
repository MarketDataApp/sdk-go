package client

import "fmt"

func ExampleOptionLookupRequest_get() {
	resp, err := OptionLookup().UserInput("AAPL 7/28/2023 200 Call").Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(resp)
	// Output: AAPL230728C00200000
}

func ExampleOptionLookupRequest_packed() {
	resp, err := OptionLookup().UserInput("AAPL 7/28/2023 200 Call").Packed()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(resp)
	// Output: OptionLookupResponse{OptionSymbol: "AAPL230728C00200000"}
}
