package client

import (
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"
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
	lastLogResponse := GetLogs().LatestString()

	if scBodyString != lastLogResponse {
		t.Errorf("Expected last log response to be %v, got %v instead", scBodyString, lastLogResponse)
	}
}

func TestAddToLog(t *testing.T) {
	// Initialize MarketDataLogs
	logs := &MarketDataLogs{
		Logs: []LogEntry{},
	}

	// Define a sample LogEntry entry
	timestamp := time.Now()
	rayID := "sampleRayID"
	request := "https://api.example.com/data"
	rateLimitConsumed := 10
	delay := int64(100)
	status := 200
	body := "{\"message\": \"success\"}"
	reqHeaders := http.Header{"Content-Type": []string{"application/json"}}
	resHeaders := http.Header{"Content-Type": []string{"application/json"}}

	// Add the log entry
	logEntry := addToLog(logs, timestamp, rayID, request, rateLimitConsumed, delay, status, body, reqHeaders, resHeaders)

	// Check if the log entry was added
	if logEntry == nil {
		t.Errorf("Log entry was not added")
	}

	// Check if the log entry matches the input
	if logEntry != nil {
		if logEntry.RayID != rayID || logEntry.Request != request || logEntry.Response != body {
			t.Errorf("Log entry does not match the input")
		}
	} else {
		t.Errorf("Log entry is nil")
	}

	// Check if the log entry is the last in the logs
	lastLogEntry := logs.Logs[len(logs.Logs)-1]
	if lastLogEntry.RayID != logEntry.RayID || lastLogEntry.Request != logEntry.Request || lastLogEntry.Response != logEntry.Response {
		t.Errorf("Log entry is not the last entry in the logs")
	}
}
