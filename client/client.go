package client

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	_ "github.com/joho/godotenv/autoload"
)

var (
	debugModeLogger = log.New(os.Stdout, "", 0) // 0 turns off all flags, including the default timestamp flag
	blue   = color.New(color.FgBlue).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	purple = color.New(color.FgMagenta).SprintFunc()
	keys   = make([]string, 0)
)

const Version = "0.0.3"

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
	debug              bool
	Logger             *zap.Logger

}

// MarketDataResponse represents the response from the Market Data API.
// It embeds the resty.Response and includes additional fields for RayID and RateLimitConsumed.
type MarketDataResponse struct {
	*resty.Response          // The response from the resty client
	RayID             string // The Ray ID from the HTTP response
	RateLimitConsumed int    // The number of requests consumed from the rate limit
	Delay             int64  // The time (in miliseconds) between the request and the server's response
}

func (mdr *MarketDataResponse) setLatency() {
	trace := mdr.Request.TraceInfo()
	mdr.Delay = trace.ServerTime.Milliseconds()
}

// GetRateLimitConsumed retrieves the number of requests consumed from the HTTP response.
// It sets the number of requests consumed to the struct and returns an error if any.
func (mdr *MarketDataResponse) setRateLimitConsumed() {
	rateLimitConsumedStr := mdr.Response.Header().Get("X-Api-RateLimit-Consumed")
	if rateLimitConsumedStr == "" {
		log.Println("Error: missing 'x-Api-RateLimit-Consumed' header")
		return
	}
	rateLimitConsumed, err := strconv.Atoi(rateLimitConsumedStr)
	if err != nil {
		log.Println(err)
		return
	}
	mdr.RateLimitConsumed = rateLimitConsumed
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
	// Open the log file. If it does not exist, create it.
	logFile, err := os.OpenFile("sdk-go.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// Create a zapcore.Core that writes to the log file
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(logFile),
		zap.InfoLevel,
	)

	// Create a zap.Logger from the Core
	logger := zap.New(core)

	client := &MarketDataClient{
		Client: resty.New(),
		debug:  false,
		Logger: logger,
	}

	client.setDefaultResetTime()

	// Set default environment to prod using the built-in method
	client.Env(ProdEnv)

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
		client.logRequestResponse(resp.Request, resp)

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
	mdr.setRateLimitConsumed()

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

func (c *MarketDataClient) logRequestResponse(req *resty.Request, resp *resty.Response) {
	// Log the request if the response was not successful or if debug mode is turned on.
	if !resp.IsSuccess() || c.debug {
		redactedHeaders := redactAuthorizationHeader(req.Header)
		c.Logger.Info("Request",
			zap.String("request_url", req.URL),
			zap.Any("request_headers", redactedHeaders),
			zap.Int("response_code", resp.StatusCode()),
			zap.String("cf_ray", resp.Header().Get("Cf-Ray")),
			zap.Any("response_headers", resp.Header()),
			zap.String("response_body", resp.String()),
		)
	}

	if c.debug {
		// Log request
		debugModeLogger.Printf("%s %s\n", blue("Request URL:"), req.URL)
		debugModeLogger.Println(blue("Request Headers:"))

		redactedHeaders := redactAuthorizationHeader(req.Header)

		// Sort the headers
		for k := range redactedHeaders {
			keys = append(keys, k)
		}

		sort.Strings(keys)
		for _, name := range keys {
			if name == "Authorization" {
				token := req.Header.Get("Authorization")
				redactedToken := "Bearer " + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
				debugModeLogger.Printf("%s: %s\n", yellow(name), redactedToken)
			} else {
				debugModeLogger.Printf("%s: %s\n", yellow(name), strings.Join(req.Header[name], ", "))
			}
		}

		// Log response
		debugModeLogger.Println(blue("Response Headers:"))

		// Sort the headers
		keys = keys[:0]
		for k := range resp.Header() {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, name := range keys {
			// If header starts with "X-Api-", print it in purple
			if strings.HasPrefix(name, "X-Api-") {
				debugModeLogger.Printf("%s: %s\n", purple(name), strings.Join(resp.Header()[name], ", "))
			} else {
				debugModeLogger.Printf("%s: %s\n", yellow(name), strings.Join(resp.Header()[name], ", "))
			}
		}
	}
}
