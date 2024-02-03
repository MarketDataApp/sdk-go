package client

import (
	"fmt"
	"log"
	"testing"
)

func TestLogging(t *testing.T) {
	// Initialize the MarketData client
	client, err := GetClient()
	if err != nil {
		log.Fatalf("Failed to get market data client: %v", err)
	}
	client.Debug(true)

	sc, err := StockCandles().Resolution("D").Symbol("AAPL").Date("2023-01-03").Raw()
	if err != nil {
		fmt.Print(err)
		t.FailNow()
	}

	scBodyString := sc.String()
	lastLogResponse := Logs.GetLastLogResponse()

	if scBodyString != lastLogResponse {
		t.Errorf("Expected last log response to be %v, got %v instead", scBodyString, lastLogResponse)
	}
}
