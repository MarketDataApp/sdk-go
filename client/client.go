package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

const Version = "1.0.0"

const (
	ProdHost = "api.marketdata.app"
	TestHost = "tst.marketdata.app"
	DevHost  = "localhost"

	ProdProtocol = "https"
	TestProtocol = "https"
	DevProtocol  = "http"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type MarketDataClient struct {
	*http.Client
	rateLimitLimit     int
	rateLimitRemaining int
	rateLimitReset     time.Time
	host               string
	protocol           string
	mu                 sync.Mutex
}

func (c *MarketDataClient) String() string {
	clientType := "Unknown"
	switch c.host {
	case ProdHost:
		clientType = "Production"
	case TestHost:
		clientType = "Test"
	case DevHost:
		clientType = "Development"
	}
	return fmt.Sprintf("ClientType: %s, RateLimitLimit: %d, RateLimitRemaining: %d, RateLimitReset: %v", clientType, c.rateLimitLimit, c.rateLimitRemaining, c.rateLimitReset)
}

var marketDataClient *MarketDataClient

func (c *MarketDataClient) SetDefaultResetTime() {
	// Get current time in Eastern Time Zone
	location, _ := time.LoadLocation("America/New_York")
	now := time.Now().In(location)

	// Set default to 9:30 AM Eastern Time the same day
	defaultReset := time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, location)

	// If current time is after 9:30 AM, set default to 9:30 AM the next day
	if now.After(defaultReset) {
		nextDay := now.AddDate(0, 0, 1)
		defaultReset = time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 9, 30, 0, 0, location)
	}

	c.rateLimitReset = defaultReset
}

func getEnvironmentDetails(env string) (string, string, error) {
	switch env {
	case "prod":
		return ProdHost, ProdProtocol, nil
	case "test":
		return TestHost, TestProtocol, nil
	case "dev":
		return DevHost, DevProtocol, nil
	default:
		return "", "", fmt.Errorf("invalid environment: %s", env)
	}
}

func NewClient(bearerToken string) (*MarketDataClient, error) {
	baseReq, _ := http.NewRequest("GET", "", nil)

	// Set the headers
	baseReq.Header.Add("Authorization", "Bearer "+bearerToken)
	baseReq.Header.Add("User-Agent", "MarketDataGoSDK/"+Version)

	// Define client before the function
	client := &MarketDataClient{}
	client.SetDefaultResetTime()

	// Set default environment to prod
	host, protocol, err := getEnvironmentDetails("prod")
	if err != nil {
		return nil, err
	}
	client.host = host
	client.protocol = protocol

	client.Client = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req = req.WithContext(context.Background())
			req.Header = baseReq.Header.Clone()
			resp, err := http.DefaultTransport.RoundTrip(req)
			if err != nil {
				return nil, err
			}

			// Lock the mutex before updating the shared rate limit fields
			client.mu.Lock()
			limit, remaining, reset, err := extractRateLimitHeaders(resp)
			if err != nil {
				log.Printf("Error extracting rate limit headers: %v", err)
				// handle error
			} else {
				client.rateLimitLimit = *limit
				client.rateLimitRemaining = *remaining
				client.rateLimitReset = time.Unix(*reset, 0)
			}
			client.mu.Unlock()
			// Unlock the mutex and return the response

			return resp, nil
		}),
	}

	// Make an initial request to authorize the token and load the rate limit information
	req, _ := http.NewRequest("GET", "https://api.marketdata.app/v1/stocks/candles/D/SPY/?from=2020-01-01&to=2020-01-31", nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK status: %s", resp.Status)
	}
	defer resp.Body.Close()

	limit, remaining, reset, err := extractRateLimitHeaders(resp)
	if err != nil {
		log.Printf("Error extracting rate limit headers: %v", err)
		// handle error
	} else {
		client.rateLimitLimit = *limit
		client.rateLimitRemaining = *remaining
		client.rateLimitReset = time.Unix(*reset, 0)
	}

	return client, nil
}

func (c *MarketDataClient) GetRateLimitLimit() int {
	return c.rateLimitLimit
}

func (c *MarketDataClient) GetRateLimitRemaining() int {
	return c.rateLimitRemaining
}

func (c *MarketDataClient) GetRateLimitReset() time.Time {
	return c.rateLimitReset
}

func (c *MarketDataClient) GetFromRequest(mdr MarketDataRequest) (*http.Response, error) {
	if c.rateLimitRemaining < 0 {
		return nil, errors.New("rate limit exceeded")
	}

	// Validate the path using helper function
	path, err := mdr.GetPath()
	if err != nil {
		return nil, err
	}
	path, err = validatePath(path)
	if err != nil {
		return nil, err
	}

	// Validate the query using helper function
	query, err := mdr.GetQuery()
	if err != nil {
		return nil, err
	}
	query, err = validateQuery(query)
	if err != nil {
		return nil, err
	}

	// Parse and validate the path
	parsedPath, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	// Parse and validate the query
	parsedQuery, err := url.Parse(query)
	if err != nil {
		return nil, err
	}

	// If no host is provided, use the default host and protocol
	if parsedPath.Host == "" {
		parsedPath.Host = c.host
		parsedPath.Scheme = c.protocol
	}

	// Join the validated path and query
	urlStr := parsedPath.String() + parsedQuery.String()

	return c.Get(urlStr)
}

func (c *MarketDataClient) Get(urlStr string) (*http.Response, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetClient is a function that takes a variable number of string arguments (token) and returns a pointer to a MarketDataClient and an error.
// If the marketDataClient is not nil, it returns the existing marketDataClient and no error.
func GetClient(token ...string) (*MarketDataClient, error) {
	if marketDataClient != nil {
		return marketDataClient, nil
	}

	if len(token) == 0 {
		token = append(token, os.Getenv("MARKETDATA_TOKEN"))
	}

	if token[0] == "" {
		return nil, errors.New("no token provided")
	}

	client, err := NewClient(token[0])
	if err != nil {
		return nil, err
	}

	marketDataClient = client
	return marketDataClient, nil
}

func SetEnvironment(env string) error {
	client, err := GetClient()
	if err != nil {
		return err
	}

	host, protocol, err := getEnvironmentDetails(env)
	if err != nil {
		return err
	}

	client.host = host
	client.protocol = protocol

	return nil
}
