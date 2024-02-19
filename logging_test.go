package client

import (
	"fmt"
	"log"
	"testing"
)

func TestLogging(t *testing.T) {
	// Initialize the MarketData client
	client, err := GetClient()
	client.Debug(true)
	if err != nil {
		log.Fatalf("Failed to get market data client: %v", err)
	}
	client.Debug(false)

	sc, err := StockCandles().Resolution("D").Symbol("AAPL").Date("2023-01-03").Raw()
	if err != nil {
		fmt.Print(err)
		fmt.Print(client)
		t.FailNow()
	}

	scBodyString := sc.String()
	lastLogResponse := GetLogs().GetLastLogResponse()

	if scBodyString != lastLogResponse {
		t.Errorf("Expected last log response to be %v, got %v instead", scBodyString, lastLogResponse)
	}
}
