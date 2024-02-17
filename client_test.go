package client

import (
	"os"
	"testing"

	"github.com/go-resty/resty/v2"
	_ "github.com/joho/godotenv/autoload"
)

func TestGetClient(t *testing.T) {
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

// TestGetEnvironment tests the getEnvironment method for various host URLs.
func TestGetEnvironment(t *testing.T) {
	// Define test cases
	tests := []struct {
		name     string
		hostURL  string
		expected string
	}{
		{
			name:     "Production Environment",
			hostURL:  "https://api.marketdata.app",
			expected: prodEnv,
		},
		{
			name:     "Testing Environment",
			hostURL:  "https://tst.marketdata.app",
			expected: testEnv,
		},
		{
			name:     "Development Environment",
			hostURL:  "http://localhost",
			expected: devEnv,
		},
		{
			name:     "Unknown Environment",
			hostURL:  "https://unknown.environment",
			expected: "Unknown",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new MarketDataClient instance
			client := &MarketDataClient{
				Client: resty.New(),
			}

			// Set the HostURL to the test case's host URL
			client.Client.SetHostURL(tc.hostURL)

			// Call getEnvironment and check the result
			result := client.getEnvironment()
			if result != tc.expected {
				t.Errorf("Expected environment %s, got %s", tc.expected, result)
			}
		})
	}
}

// TestEnvironmentMethod tests the Environment method for setting the client's environment.
func TestEnvironmentMethod(t *testing.T) {
	// Define test cases
	tests := []struct {
		name        string
		environment string
		expectedURL string
		expectError bool
	}{
		{
			name:        "Set Production Environment",
			environment: prodEnv,
			expectedURL: prodProtocol + "://" + prodHost,
			expectError: false,
		},
		{
			name:        "Set Testing Environment",
			environment: testEnv,
			expectedURL: testProtocol + "://" + testHost,
			expectError: false,
		},
		{
			name:        "Set Development Environment",
			environment: devEnv,
			expectedURL: devProtocol + "://" + devHost,
			expectError: false,
		},
		{
			name:        "Set Invalid Environment",
			environment: "invalidEnv",
			expectedURL: "",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new MarketDataClient instance
			client := new()

			// Set the environment using the Environment method
			client = client.Environment(tc.environment)

			// Check if an error was expected
			if tc.expectError {
				if client.Error == nil {
					t.Errorf("Expected an error for environment %s, but got none", tc.environment)
				}
			} else {
				if client.Error != nil {
					t.Errorf("Did not expect an error for environment %s, but got: %v", tc.environment, client.Error)
				}

				// Verify that the baseURL was set correctly
				if client.Client.HostURL != tc.expectedURL {
					t.Errorf("Expected baseURL %s, got %s", tc.expectedURL, client.Client.HostURL)
				}

				// Additionally, verify that getEnvironment returns the correct environment
				result := client.getEnvironment()
				if result != tc.environment {
					t.Errorf("Expected environment %s, got %s", tc.environment, result)
				}
			}
		})
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
