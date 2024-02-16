package client

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"
)

func ExampleBulkStockQuotesRequest_get() {
	// This example demonstrates how to create a BulkStockQuotesRequest, set its parameters,
	// and perform an actual request to fetch stock quotes for multiple symbols.

	// Initialize a new BulkStockQuotesRequest and fetch stock quotes for "AAPL", "META", and "MSFT".
	symbols := []string{"AAPL", "META", "MSFT"}
	bsqr, err := BulkStockQuotes().Symbols(symbols).Get()
	if err != nil {
		log.Fatalf("Failed to get bulk stock quotes: %v", err)
	}

	// Check if the response contains the symbols.
	for _, quote := range bsqr {
		fmt.Printf("Symbol: %s\n", quote.Symbol)
	}
	// Output: Symbol: AAPL
	// Symbol: META
	// Symbol: MSFT
}

func ExampleBulkStockQuotesRequest_packed() {
	// This example demonstrates how to create a BulkStockQuotesRequest, set its parameters,
	// and perform an actual request to fetch stock quotes for multiple symbols: "AAPL", "META", and "MSFT".

	// Initialize a new BulkStockQuotesRequest and fetch stock quotes for the specified symbols.
	symbols := []string{"AAPL", "META", "MSFT"}
	bsqr, err := BulkStockQuotes().Symbols(symbols).Packed()
	if err != nil {
		log.Fatalf("Failed to get bulk stock quotes: %v", err)
	}

	// Iterate and print all the symbols in the response.
	for _, symbol := range bsqr.Symbol {
		fmt.Printf("Symbol: %s\n", symbol)
	}
	// Output: Symbol: AAPL
	// Symbol: META
	// Symbol: MSFT
}

func ExampleBulkStockQuotesRequest_raw() {
	// This example demonstrates how to create a BulkStockQuotesRequest, set its parameters,
	// and perform an actual request to fetch stock quotes for multiple symbols: "AAPL", "META", and "MSFT".
	// The response is converted to a raw string and we print out the string at the end of the test.

	// Initialize a new BulkStockQuotesRequest and fetch stock quotes for "AAPL", "META", and "MSFT".
	symbols := []string{"AAPL", "META", "MSFT"}
	bsqr, err := BulkStockQuotes().Symbols(symbols).Raw()
	if err != nil {
		log.Fatalf("Failed to get bulk stock quotes: %v", err)
	}

	// Convert the response to a string.
	jsonStr := bsqr.String()

	// Define a struct to match the JSON structure
	type Response struct {
		Symbol []string `json:"symbol"`
	}

	// Unmarshal the JSON into the struct
	var resp Response
	err = json.Unmarshal([]byte(jsonStr), &resp)
	if err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	// Print the symbols
	for _, symbol := range resp.Symbol {
		fmt.Printf("Symbol: %s\n", symbol)
	}
	// Output: Symbol: AAPL
	// Symbol: META
	// Symbol: MSFT
}

func TestBulkStockQuotesRequest_packed(t *testing.T) {
	//client, _ := GetClient()
	//client.Debug(true)

	symbols := []string{"AAPL", "META", "MSFT"}
	bsqr, err := BulkStockQuotes().Symbols(symbols).Get()
	if err != nil {
		fmt.Print(err)
		return
	}

	currentDate := time.Now()
	for _, quote := range bsqr {
		// Check if the symbol is one of the expected symbols
		if quote.Symbol != "AAPL" && quote.Symbol != "META" && quote.Symbol != "MSFT" {
			t.Errorf("Unexpected symbol %s found in response", quote.Symbol)
		}

		// Check if the UpdatedAt date is within the last 7 days
		if currentDate.Sub(quote.Updated).Hours() > 168 { // 168 hours in 7 days
			t.Errorf("Quote for symbol %s was not updated within the last 7 days", quote.Symbol)
		}
	}
}
