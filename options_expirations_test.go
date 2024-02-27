package client

import "fmt"

func ExampleOptionsExpirationsResponse_get() {
	resp, err := OptionsExpirations().UnderlyingSymbol("AAPL").Date("2009-02-04").Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	for _, expirations := range resp {
		fmt.Println(expirations)
	}
	// Output: 2009-02-20 16:00:00 -0500 EST
	// 2009-03-20 16:00:00 -0400 EDT
	// 2009-04-17 16:00:00 -0400 EDT
	// 2009-07-17 16:00:00 -0400 EDT
	// 2010-01-15 16:00:00 -0500 EST
	// 2011-01-21 16:00:00 -0500 EST
}

func ExampleOptionsExpirationsResponse_packed() {
	resp, err := OptionsExpirations().UnderlyingSymbol("AAPL").Date("2009-02-04").Packed()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(resp)
	// Output: OptionsExpirationsResponse{Expirations: [2009-02-20 2009-03-20 2009-04-17 2009-07-17 2010-01-15 2011-01-21], Updated: 1233723600}
}
