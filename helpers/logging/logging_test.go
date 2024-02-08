package logging

import (
	"net/http"
	"testing"
	"time"
)

func TestAddToLog(t *testing.T) {
	// Initialize HttpRequestLogs
	logs := &HttpRequestLogs{
		Logs: []HttpRequestLog{},
	}

	// Define a sample HttpRequestLog entry
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
	logEntry := AddToLog(logs, timestamp, rayID, request, rateLimitConsumed, delay, status, body, reqHeaders, resHeaders)

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