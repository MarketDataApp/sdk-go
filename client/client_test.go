package client

import (
	"os"
	"testing"

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
/*

func TestRateLimit(t *testing.T) {
	client, err := GetClient(os.Getenv("MARKETDATA_TOKEN"))
	if err != nil {
		t.Fatal(err)
	}

	// Check initial rate limit limit and reset time
	initialLimit := client.RateLimitLimit
	if initialLimit <= 0 {
		t.Errorf("Expected positive rate limit limit, but got %d", initialLimit)
	}
	resetTime := client.RateLimitReset
	if time.Now().After(resetTime) {
		t.Errorf("Expected reset time in the future, but got %v", resetTime)
	}

	// Request to https://api.marketdata.app/stocks/quotes/AAPL/
	resp, err := client.Get("https://api.marketdata.app/v1/stocks/quotes/AAPL/")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Headers after AAPL request:", resp.Header())
	body := resp.Body()
	fmt.Println("Status code after AAPL request:", resp.StatusCode())
	fmt.Println("Body after AAPL request:", string(body))
	initialRemaining := client.RateLimitRemaining
	//fmt.Println("Rate limit consumed after AAPL request:", resp.RateLimitConsumed)

	// Request to https://api.marketdata.app/stocks/quotes/SPY/
	resp, err = client.Get("https://api.marketdata.app/v1/stocks/quotes/SPY/")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Headers after SPY request:", resp.Header())
	body = resp.Body()
	fmt.Println("Status code after SPY request:", resp.StatusCode())
	fmt.Println("Body after SPY request:", string(body))
	afterRequestRemaining := client.RateLimitRemaining
	//fmt.Println("Rate limit consumed after SPY request:", resp.RateLimitConsumed)

	if afterRequestRemaining != initialRemaining-1 {
		t.Errorf("Expected remaining rate limit to decrease by 1, but got %d", initialRemaining-afterRequestRemaining)
	}
}

*/