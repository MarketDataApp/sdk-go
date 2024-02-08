package client

import (
	"fmt"
	"testing"
	"time"
)

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