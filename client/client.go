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

	"github.com/MarketDataApp/sdk-go/helpers/logging"
	"github.com/go-resty/resty/v2"

	_ "github.com/joho/godotenv/autoload"
)

const (
	Version = "0.0.4"

	prodEnv = "prod"
	testEnv = "test"
	devEnv  = "dev"

	prodHost = "api.marketdata.app"
	testHost = "tst.marketdata.app"
	devHost  = "localhost"

	prodProtocol = "https"
	testProtocol = "https"
	devProtocol  = "http"
)

type MarketDataClient struct {
	*resty.Client
	RateLimitLimit     int
	RateLimitRemaining int
	RateLimitReset     time.Time
	mu                 sync.Mutex
	Error              error
	debug              bool
}

func (c *MarketDataClient) addLogFromRequestResponse(req *resty.Request, resp *resty.Response) {
	redactedHeaders := redactAuthorizationHeader(req.Header)
	resHeaders := resp.Header()
	rateLimitConsumed, err := getRateLimitConsumed(resp)
	if err != nil {
		c.Error = err
		return
	}
	rayID, err := getRayIDFromResponse(resp)
	if err != nil {
		c.Error = err
		return
	}
	delay := getLatencyFromRequest(req)
	status := resp.StatusCode()
	body := string(resp.Body())

	logEntry := logging.AddToLog(GetLogs(), time.Now(), rayID, req.URL, rateLimitConsumed, delay, status, body, redactedHeaders, resHeaders)
	if c.debug && logEntry != nil {
		logEntry.PrettyPrint()
	}
	if logEntry != nil {
		go logEntry.WriteToLog(c.debug)
	}
}

func (c *MarketDataClient) getEnvironment() string {
	u, err := url.Parse(c.Client.HostURL)
	if err != nil {
		log.Printf("Error parsing host URL: %v", err)
		return "Unknown"
	}
	switch u.Hostname() {
	case prodHost:
		return prodEnv
	case testHost:
		return testEnv
	case devHost:
		return devEnv
	default:
		return "Unknown"
	}
}

func (c *MarketDataClient) String() string {
	clientType := c.getEnvironment()
	return fmt.Sprintf("Client Type: %s, RateLimitLimit: %d, RateLimitRemaining: %d, RateLimitReset: %v", clientType, c.RateLimitLimit, c.RateLimitRemaining, c.RateLimitReset)
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
		debug:  false,
	}

	client.setDefaultResetTime()

	// Set default environment to prod using the built-in method
	client.Environment(prodEnv)

	// Set the "User-Agent" header
	client.Client.SetHeader("User-Agent", "sdk-go/"+Version)

	// Enable client trace
	client.Client.EnableTrace()

	// Set the OnBeforeRequest hook
	client.Client.OnBeforeRequest(func(c *resty.Client, req *resty.Request) error {
		//log.Printf("Sending request to URL: %s", req.URL) // Debug print
		return nil
	})

	// Set the OnAfterResponse hook
	client.Client.OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
		//log.Printf("Sent request URL: %s", resp.Request.URL) // Debug print
		client.updateRateLimit(resp)
		client.addLogFromRequestResponse(resp.Request, resp)
		//client.logRequestResponse(resp.Request, resp)
		//client.logToConsole(resp.Request, resp)
		//client.printLatest()

		return nil
	})

	return client
}

// Debug is a method that enables or disables the debug mode of the client.
// Debug mode will result in the request and response headers being printed to
// the terminal with each request.
func (c *MarketDataClient) Debug(enable bool) *MarketDataClient {
	c.debug = enable
	return c
}

func (c *MarketDataClient) updateRateLimit(resp *resty.Response) {
	// Lock the mutex before updating the shared rate limit fields
	c.mu.Lock()
	defer c.mu.Unlock()

	limitHeader := resp.Header().Get("X-Api-Ratelimit-Limit")
	remainingHeader := resp.Header().Get("X-Api-Ratelimit-Remaining")
	resetHeader := resp.Header().Get("X-Api-Ratelimit-Reset")

	if limitHeader == "" {
		log.Println("Error: missing 'X-Api-Ratelimit-Limit' header")
		return
	}
	if remainingHeader == "" {
		log.Println("Error: missing 'X-Api-Ratelimit-Remaining' header")
		return
	}
	if resetHeader == "" {
		log.Println("Error: missing 'X-Api-Ratelimit-Reset' header")
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

func (c *MarketDataClient) GetFromRequest(br *baseRequest, result interface{}) (*resty.Response, error) {
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

	resp, err := req.Get(path)
	if err != nil {
		return resp, err
	}

	if !resp.IsSuccess() {
		return resp, fmt.Errorf("received non-OK status: %s", resp.Status())
	}

	return resp, nil
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

func (c *MarketDataClient) Environment(env string) *MarketDataClient {
	var baseURL string
	switch env {
	case prodEnv:
		baseURL = prodProtocol + "://" + prodHost
	case testEnv:
		baseURL = testProtocol + "://" + testHost
	case devEnv:
		baseURL = devProtocol + "://" + devHost
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
		marketDataClient = NewClient().Environment(env).Token(token)
	}
}

func (c *MarketDataClient) Token(bearerToken string) *MarketDataClient {
	// Set the authentication scheme to "Bearer"
	c.Client.SetAuthScheme("Bearer")

	// Set the authentication token
	c.Client.SetAuthToken(bearerToken)

	// Make an initial request to authorize the token and load the rate limit information
	resp, err := c.Client.R().Get("https://api.marketdata.app/user/")
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

// GetLogs method returns the Logs variable.
func GetLogs() *logging.HttpRequestLogs {
	return logging.Logs
}
