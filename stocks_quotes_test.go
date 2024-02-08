package client

import (
	"fmt"
	"testing"
	"time"
)

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
