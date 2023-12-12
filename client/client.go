package client

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	_ "github.com/joho/godotenv/autoload"
)

const Version = "1.0.0"

const (
	ProdEnv = "prod"
	TestEnv = "test"
	DevEnv  = "dev"

	ProdHost = "api.marketdata.app"
	TestHost = "tst.marketdata.app"
	DevHost  = "localhost"

	ProdProtocol = "https"
	TestProtocol = "https"
	DevProtocol  = "http"
)

type MarketDataClient struct {
	*resty.Client
	RateLimitLimit     int
	RateLimitRemaining int
	RateLimitReset     time.Time
	mu                 sync.Mutex
	Error              error
}

// MarketDataResponse represents the response from the Market Data API.
// It embeds the resty.Response and includes additional fields for RayID and RateLimitConsumed.
type MarketDataResponse struct {
	*resty.Response          // The response from the resty client
	RayID             string // The Ray ID from the HTTP response
	RateLimitConsumed int    // The number of requests consumed from the rate limit
	Delay             int64  // The time (in miliseconds) between the request and the server's response.
}

func (mdr *MarketDataResponse) setLatency() {
	trace := mdr.Request.TraceInfo()
	mdr.Delay = trace.ServerTime.Milliseconds()
}

// GetRateLimitConsumed retrieves the number of requests consumed from the HTTP response.
// It sets the number of requests consumed to the struct and returns an error if any.
func (mdr *MarketDataResponse) setRateLimitConsumed() error {
	rateLimitConsumedStr := mdr.Response.Header().Get("X-Api-RateLimit-Consumed")
	if rateLimitConsumedStr == "" {
		return errors.New("x-Api-RateLimit-Consumed header not found")
	}
	rateLimitConsumed, err := strconv.Atoi(rateLimitConsumedStr)
	if err != nil {
		return err
	}
	mdr.RateLimitConsumed = rateLimitConsumed
	return nil
}

// GetRayID retrieves the Cf-Ray ID from the HTTP response.
// It sets the Cf-Ray ID to the struct and returns an error if any.
func (mdr *MarketDataResponse) setRayID() error {
	rayID := mdr.Response.Header().Get("Cf-Ray")
	if rayID == "" {
		return errors.New("Cf-Ray header not found")
	}
	mdr.RayID = rayID
	return nil
}

func (c *MarketDataClient) GetEnv() string {
	u, err := url.Parse(c.Client.HostURL)
	if err != nil {
		log.Printf("Error parsing host URL: %v", err)
		return "Unknown"
	}
	switch u.Hostname() {
	case ProdHost:
		return ProdEnv
	case TestHost:
		return TestEnv
	case DevHost:
		return DevEnv
	default:
		return "Unknown"
	}
}

func (c *MarketDataClient) String() string {
	clientType := c.GetEnv()
	return fmt.Sprintf("ClientType: %s, RateLimitLimit: %d, RateLimitRemaining: %d, RateLimitReset: %v", clientType, c.RateLimitLimit, c.RateLimitRemaining, c.RateLimitReset)
}

var marketDataClient *MarketDataClient

func (c *MarketDataClient) setDefaultResetTime() {
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

	c.RateLimitReset = defaultReset
}

func NewClient() *MarketDataClient {
	client := &MarketDataClient{
		Client: resty.New(),
	}

	client.setDefaultResetTime()

	// Set default environment to prod using the built-in method
	client.Env(ProdEnv)

	// Set the "User-Agent" header
	client.Client.SetHeader("User-Agent", "sdk-go/"+Version)

	// Enable client trace
	client.Client.EnableTrace()

	// Set the OnAfterResponse hook
	client.Client.OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
		log.Printf("Sent request URL: %s", resp.Request.URL) // Debug print
		client.updateRateLimit(resp)
		return nil
	})

	return client
}

func (c *MarketDataClient) updateRateLimit(resp *resty.Response) {
	// Lock the mutex before updating the shared rate limit fields
	c.mu.Lock()
	defer c.mu.Unlock()

	limitHeader := resp.Header().Get("X-Api-Ratelimit-Limit")
	remainingHeader := resp.Header().Get("X-Api-Ratelimit-Remaining")
	resetHeader := resp.Header().Get("X-Api-Ratelimit-Reset")

	if limitHeader == "" || remainingHeader == "" || resetHeader == "" {
		log.Printf("URL of the request made: %v", resp.Request.URL)
		log.Println("Error: missing rate limit headers")
		return
	}

	limitVal, err := strconv.Atoi(limitHeader)
	if err != nil {
		log.Printf("Error converting limit header to int: %v", err)
		return
	}

	remainingVal, err := strconv.Atoi(remainingHeader)
	if err != nil {
		log.Printf("Error converting remaining header to int: %v", err)
		return
	}

	resetVal, err := strconv.ParseInt(resetHeader, 10, 64)
	if err != nil {
		log.Printf("Error converting reset header to int64: %v", err)
		return
	}

	c.RateLimitLimit = limitVal
	c.RateLimitRemaining = remainingVal
	c.RateLimitReset = time.Unix(resetVal, 0)
}

func (c *MarketDataClient) GetFromRequest(br *baseRequest, result interface{}) (*MarketDataResponse, error) {
	if c.RateLimitRemaining < 0 {
		return nil, errors.New("rate limit exceeded")
	}
	req := br.getResty().SetResult(result)

	// Parse the parameters from the request
	paramsSlice, err := br.getParams()
	if err != nil {
		return nil, err
	}
	for _, param := range paramsSlice {
		err := param.SetParams(req)
		if err != nil {
			return nil, err
		}
	}

	path, err := br.getPath()
	if err != nil {
		return nil, err
	}

	if err := br.getError(); err != nil {
		return nil, err
	}

	resp, err := c.WrapResponse(req, path) // Must run GET after setting all params.
	if err != nil {
		log.Printf("Error sending request: %v", err) // Debug print
		return resp, err
	}

	if !resp.IsSuccess() {
		log.Printf("Received non-OK status: %s", resp.Status()) // Debug print
		return resp, fmt.Errorf("received non-OK status: %s", resp.Status())
	}

	return resp, nil
}

func (c *MarketDataClient) Get(path string) (*MarketDataResponse, error) {
	req := c.Client.R()
	return c.WrapResponse(req, path)
}

func (c *MarketDataClient) WrapResponse(req *resty.Request, path string) (*MarketDataResponse, error) {
	resp, err := req.Get(path) // Must run GET after setting all params.
	if err != nil {
		return nil, err
	}

	mdr := &MarketDataResponse{Response: resp}
	err = mdr.setRateLimitConsumed()
	if err != nil {
		return nil, err
	}

	err = mdr.setRayID()
	if err != nil {
		return nil, err
	}

	mdr.setLatency()

	return mdr, nil
}

func GetClient(token ...string) (*MarketDataClient, error) {
	if len(token) == 0 {
		if marketDataClient != nil {
			if marketDataClient.Error != nil {
				return nil, marketDataClient.Error
			}
			return marketDataClient, nil
		}
		token = append(token, os.Getenv("MARKETDATA_TOKEN"))
	}

	if token[0] == "" {
		return nil, errors.New("no token provided")
	}

	// Always create a new client when a token is provided
	client := NewClient()
	if client.Error != nil {
		return nil, client.Error
	}

	client.Token(token[0])
	if client.Error != nil {
		return nil, client.Error
	}

	// Save the new client to the global variable if no errors are present
	marketDataClient = client

	return client, nil
}

func (c *MarketDataClient) Env(env string) *MarketDataClient {
	var baseURL string
	switch env {
	case ProdEnv:
		baseURL = ProdProtocol + "://" + ProdHost
	case TestEnv:
		baseURL = TestProtocol + "://" + TestHost
	case DevEnv:
		baseURL = DevProtocol + "://" + DevHost
	default:
		c.Error = fmt.Errorf("invalid environment: %s", env)
		return c
	}

	c.Client.SetBaseURL(baseURL)

	return c
}

func init() {
	token := os.Getenv("MARKETDATA_TOKEN")
	env := os.Getenv("DEFAULT_ENV")

	if token != "" && env != "" {
		marketDataClient = NewClient().Env(env).Token(token)
	}
}

func (c *MarketDataClient) Token(bearerToken string) *MarketDataClient {
	// Set the authentication scheme to "Bearer"
	c.Client.SetAuthScheme("Bearer")

	// Set the authentication token
	c.Client.SetAuthToken(bearerToken)

	// Make an initial request to authorize the token and load the rate limit information
	resp, err := c.Client.R().Get("https://api.marketdata.app/v1/stocks/candles/D/SPY/?from=2020-01-01&to=2020-01-31")
	if err != nil {
		c.Error = err
		return c
	}
	if !resp.IsSuccess() {
		err = fmt.Errorf("received non-OK status: %s", resp.Status())
		c.Error = err
		return c
	}

	return c
}
