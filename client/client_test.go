package client

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

func TestNewClient(t *testing.T) {
	// Generate a new client with the actual token
	client, err := GetClient(os.Getenv("MARKETDATA_TOKEN"))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if client == nil {
		t.Errorf("Expected a client, got nil")
	}

	// Generate a new client with an invalid token
	client, err = GetClient("invalid_token")
	if err == nil {
		t.Errorf("Expected an error, got nil")
	}
	if client != nil {
		t.Errorf("Expected nil, got a client")
	}
}

func TestRateLimit(t *testing.T) {
	client, err := GetClient(os.Getenv("MARKETDATA_TOKEN"))
	if err != nil {
		t.Fatal(err)
	}

	// Check initial rate limit limit and reset time
	initialLimit := client.GetRateLimitLimit()
	if initialLimit <= 0 {
		t.Errorf("Expected positive rate limit limit, but got %d", initialLimit)
	}
	resetTime := client.GetRateLimitReset()
	if time.Now().After(resetTime) {
		t.Errorf("Expected reset time in the future, but got %v", resetTime)
	}

	// Request to https://api.marketdata.app/stocks/quotes/AAPL/
	resp, err := client.Get("https://api.marketdata.app/v1/stocks/quotes/AAPL/")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Headers after AAPL request:", resp.Header)
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Status code after AAPL request:", resp.StatusCode)
	fmt.Println("Body after AAPL request:", string(body))
	initialRemaining := client.GetRateLimitRemaining()
	rateLimitConsumed, err := GetRateLimitConsumed(resp)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Rate limit consumed after AAPL request:", rateLimitConsumed)
	resp.Body.Close()

	// Request to https://api.marketdata.app/stocks/quotes/SPY/
	resp, err = client.Get("/v1/stocks/quotes/SPY/")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Headers after SPY request:", resp.Header)
	body, _ = io.ReadAll(resp.Body)
	fmt.Println("Status code after SPY request:", resp.StatusCode)
	fmt.Println("Body after SPY request:", string(body))
	afterRequestRemaining := client.GetRateLimitRemaining()
	rateLimitConsumed, err = GetRateLimitConsumed(resp)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Rate limit consumed after SPY request:", rateLimitConsumed)
	resp.Body.Close()

	if afterRequestRemaining != initialRemaining-1 {
		t.Errorf("Expected remaining rate limit to decrease by 1, but got %d", initialRemaining-afterRequestRemaining)
	}
}
