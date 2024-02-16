package client

import (
	"fmt"
	"log"
	"regexp"
	"testing"
	"time"
)

func ExampleStockQuoteRequest_get() {
	// This example demonstrates how to create a StockQuoteRequest, set its parameters,
	// and perform an actual request to fetch stock quotes for the "AAPL" symbol.

	// Initialize a new StockQuoteRequest and fetch a stock quote.
	sqr, err := StockQuote().Symbol("AAPL").Get()
	if err != nil {
		log.Fatalf("Failed to get stock quotes: %v", err)
	}

	// Check if the response contains the "AAPL" symbol.
	for _, quote := range sqr {
		fmt.Printf("Symbol: %s\n", quote.Symbol)
	}
	// Output: Symbol: AAPL
}

func ExampleStockQuoteRequest_packed() {
	// This example demonstrates how to create a StockQuoteRequest, set its parameters,
	// and perform an actual request to fetch stock quotes for the "AAPL" symbol.

	// Initialize a new StockQuoteRequest and fetch a stock quote.
	sqr, err := StockQuote().Symbol("AAPL").Packed()
	if err != nil {
		log.Fatalf("Failed to get stock quotes: %v", err)
	}

	// Iterate and print all the symbols in the slice.
	for _, symbol := range sqr.Symbol {
		fmt.Printf("Symbol: %s\n", symbol)
	}
	// Output: Symbol: AAPL
}

func ExampleStockQuoteRequest_raw() {
	// This example demonstrates how to create a StockQuoteRequest, set its parameters,
	// and perform an actual request to fetch stock quotes for the "AAPL" symbol.
	// The response is converted to a raw string and we print out the string at the end of the test.

	// Initialize a new StockQuoteRequest and fetch a stock quote.
	sqr, err := StockQuote().Symbol("AAPL").Raw()
	if err != nil {
		log.Fatalf("Failed to get stock quotes: %v", err)
	}

	// Convert the response to a string.
	sqrStr := sqr.String()

	// Use regex to find the symbol in the response string.
	re := regexp.MustCompile(`"symbol":\["(.*?)"\]`)
	matches := re.FindStringSubmatch(sqrStr)

	if len(matches) < 2 {
		log.Fatalf("Failed to extract symbol from response")
	}

	// Print the extracted symbol.
	fmt.Printf("Symbol: %s\n", matches[1])
	// Output: Symbol: AAPL
}

func TestStockQuoteRequest(t *testing.T) {
	sqr, err := StockQuote().Symbol("AAPL").FiftyTwoWeek(true).Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	for _, quote := range sqr {
		if quote.Symbol != "AAPL" {
			t.Errorf("Expected symbol 'AAPL', got %s", quote.Symbol)
		}

		if quote.High52 != nil && *quote.High52 <= 0 {
			t.Errorf("Expected High52 to be a positive number, got %f", *quote.High52)
		}

		if quote.Low52 != nil && *quote.Low52 <= 0 {
			t.Errorf("Expected Low52 to be a positive number, got %f", *quote.Low52)
		}

		sevenDaysAgo := time.Now().AddDate(0, 0, -7)
		if quote.Updated.Before(sevenDaysAgo) {
			t.Errorf("Expected Updated to be within the last 7 days, got %s", quote.Updated)
		}
	}
}
