// Package client provides a Go SDK for interacting with the Market Data API.
// The [Market Data Go Client] includes functionality for making API requests, handling responses,
// managing rate limits, and logging. The SDK supports various data types
// including stocks, options, indices, and market status information.
//
// # Get Started Quickly with the MarketDataClient
//
//  1. Use [GetClient] to fetch the [MarketDataClient] instance and set the API token.
//  2. Turn on Debug mode to log detailed request and response information to disk as you learn how to use the SDK.
//  3. Make a test request.
//  4. Check the rate limit in the client to keep track of your requests.
//  5. Check the in-memory logs to see the raw request and response details.
//
// [Market Data Go Client]: https://www.marketdata.app/docs/sdk/go/client
package client

import (
	"encoding/json"
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

// Environment represents the type for different environments in which the MarketDataClient can operate.
// Customers do not need to set the environment. The [MarketDataClient] will automatically be initialized with a Production
// environment if no environment is set.
//
// Market Data's Go Client supports three environments:
//
//  1. Production
//  2. Test
//  3. Development.
//
// It is used to configure the client to point to the appropriate base URL depending on the environment.
// This is used for development or testing by Market Data employees.
type Environment string

const (
	// Production specifies the production environment. It is used when the client is interacting with the live Market Data API.
	Production Environment = "prod"

	// Test specifies the testing environment. It is used for testing purposes, allowing interaction with a sandbox version of the Market Data API.
	Test Environment = "test"

	// Development specifies the development environment. It is typically used during the development phase, pointing to a local or staged version of the Market Data API.
	Development Environment = "dev"
)

// MarketDataClient struct defines the structure for the MarketData client instance.
// It embeds the resty.Client to inherit the HTTP client functionalities.
// Additionally, it includes fields for managing rate limits and synchronization,
// as well as an error field for capturing any errors that occur during API calls.
// The debug field is used to control logging verbosity.
//
// # Setter Methods
//
//   - Debug(bool): Enables or disables debug mode for logging detailed request and response information.
//   - Environment(Environment): Sets the environment for the MarketDataClient.
//   - Timeout(int): Sets the request timeout for the MarketDataClient.
//   - Token(string) error: Sets the authentication token for the MarketDataClient.
//
// # Methods
//
//   - RateLimitExceeded() bool: Checks if the rate limit for API requests has been exceeded.
//   - String() string: Generates a formatted string that represents the MarketDataClient instance.
type MarketDataClient struct {
	*resty.Client                 // Embedding resty.Client to utilize its HTTP client functionalities.
	RateLimitLimit     int        // RateLimitLimit represents the maximum number of requests that can be made in a rate limit window.
	RateLimitRemaining int        // RateLimitRemaining tracks the number of requests that can still be made before hitting the rate limit.
	RateLimitReset     time.Time  // RateLimitReset indicates the time when the rate limit will be reset.
	mu                 sync.Mutex // mu is used to ensure thread-safe access to the client's fields.
	debug              bool       // Debug indicates whether debug mode is enabled, controlling the verbosity of logs.
}

// Debug enables or disables debug mode for the MarketDataClient. When debug mode is enabled, the client logs detailed request and response information,
// which can be useful for development and troubleshooting.
//
// # Parameters
//
//   - bool: A boolean value indicating whether to enable or disable debug mode.
func (c *MarketDataClient) Debug(enable bool) {
	c.debug = enable
}

// RateLimitExceeded checks if the rate limit for API requests has been exceeded.
// It returns true if the number of remaining requests is less than or equal to zero
// and the current time is before the rate limit reset time, indicating that the client
// must wait before making further requests. Otherwise, it returns false, indicating
// that the client can continue making requests.
//
// # Returns
//
//   - bool: A boolean value indicating whether the rate limit has been exceeded.
func (c *MarketDataClient) RateLimitExceeded() bool {
	// If there are remaining requests, return false immediately.
	if c.RateLimitRemaining > 0 {
		return false
	}
	// If no remaining requests and the current time is after the rate limit reset,
	// it means the rate limit has been reset, and we can return false.
	if c.RateLimitRemaining <= 0 && time.Now().After(c.RateLimitReset) {
		return false
	}
	// If no remaining requests and the current time is before the rate limit reset,
	// it means the rate limit is exceeded, and we must wait, returning true.
	if c.RateLimitRemaining <= 0 && time.Now().Before(c.RateLimitReset) {
		return true
	}
	// Default case should not be reached, but return false as a safeguard.
	return false
}

// addLogFromRequestResponse adds a log entry based on the request and response information.
// It redacts sensitive information from headers, extracts rate limit and ray ID from the response,
// calculates request latency, and constructs a log entry with these details.
// If debug mode is enabled, the log entry is printed in a human-readable format.
// Regardless of debug mode, the log entry is written to the log.
func (c *MarketDataClient) addLogFromRequestResponse(req *resty.Request, resp *resty.Response) error {
	// Redact sensitive information from request headers.
	redactedHeaders := redactAuthorizationHeader(req.Header)
	// Extract response headers.
	resHeaders := resp.Header()
	// Attempt to extract rate limit consumed information from the response.
	rateLimitConsumed, err := getRateLimitConsumed(resp)
	if err != nil {
		return err
	}
	// Attempt to extract the ray ID from the response.
	rayID, err := getRayIDFromResponse(resp)
	if err != nil {
		return err
	}
	// Calculate the latency of the request.
	delay := getLatencyFromRequest(req)
	// Extract the status code from the response.
	status := resp.StatusCode()
	// Convert the response body to a string.
	body := string(resp.Body())

	// Create a new log entry with the gathered information.
	logEntry := addToLog(GetLogs(), time.Now(), rayID, req.URL, rateLimitConsumed, delay, status, body, redactedHeaders, resHeaders)
	// If debug mode is enabled and the log entry is not nil, pretty print the log entry.
	if c.debug && logEntry != nil {
		logEntry.PrettyPrint()
	}
	// If the log entry is not nil, write it to the log.
	if logEntry != nil {
		logEntry.writeToLog(c.debug)
	}
	return nil
}

// getEnvironment determines the environment the client is operating in based on the host URL.
// It parses the host URL to extract the hostname and matches it against predefined hostnames
// for production, testing, and development environments. If a match is found, it returns the
// corresponding environment name; otherwise, it defaults to "Unknown".
func (c *MarketDataClient) getEnvironment() Environment {
	if c == nil || c.Client == nil {
		return "Unknown"
	}

	u, err := url.Parse(c.Client.HostURL) // Parse the host URL to extract the hostname.
	if err != nil {
		log.Printf("Error parsing host URL: %v", err) // Log any error encountered during URL parsing.
		return "Unknown"                              // Default to "Unknown" if there's an error in parsing the URL.
	}
	switch u.Hostname() { // Match the extracted hostname against predefined hostnames.
	case prodHost:
		return Production // Return the production environment name if matched.
	case testHost:
		return Test // Return the testing environment name if matched.
	case devHost:
		return Development // Return the development environment name if matched.
	default:
		return "Unknown" // Default to "Unknown" if no matches are found.
	}
}

// String generates a formatted string that represents the MarketDataClient instance, including its environment, rate limit information, and the rate limit reset time. This method is useful for quickly obtaining a textual summary of the client's current state, particularly for logging or debugging purposes.
//
// # Returns
//
//   - string: A formatted string containing the client's environment, rate limit information, and rate limit reset time.
func (c *MarketDataClient) String() string {
	// Check if the MarketDataClient instance is nil
	if c == nil {
		return "MarketDataClient instance is nil"
	}

	clientType := c.getEnvironment() // Determine the client's environment.
	// Format and return the string representation.
	return fmt.Sprintf("Client Type: %s, RateLimitLimit: %d, RateLimitRemaining: %d, RateLimitReset: %v", clientType, c.RateLimitLimit, c.RateLimitRemaining, c.RateLimitReset)
}

// setDefaultResetTime sets the default rate limit reset time for the MarketDataClient.
// It calculates the reset time based on the current time in the Eastern Time Zone,
// defaulting to 9:30 AM on the current or next day depending on the current time.
func (c *MarketDataClient) setDefaultResetTime() {
	// Load the Eastern Time Zone location
	location, _ := time.LoadLocation("America/New_York")
	// Get the current time in the Eastern Time Zone
	now := time.Now().In(location)

	// Initialize defaultReset to 9:30 AM Eastern Time on the current day
	defaultReset := time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, location)

	// If the current time is after 9:30 AM, adjust defaultReset to 9:30 AM on the next day
	if now.After(defaultReset) {
		nextDay := now.AddDate(0, 0, 1)                                                                 // Calculate the next day
		defaultReset = time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 9, 30, 0, 0, location) // Set defaultReset to 9:30 AM on the next day
	}

	// Update the MarketDataClient's RateLimitReset to the calculated default reset time
	c.RateLimitReset = defaultReset
}

// New creates and configures a new MarketDataClient instance with default settings. This method is primarily used to initialize a client with predefined configurations such as the default rate limit reset time, production environment setup, and common HTTP headers and hooks. It's the starting point for interacting with the MarketDataClient functionalities.
//
// # Returns
//
//   - *MarketDataClient: A pointer to the newly created MarketDataClient instance with default configurations applied.
func NewClient(token string) error {
	client := newClient()

	// Set the client's token.
	err := client.Token(token)
	if err != nil {
		return err
	}

	// Set the global client if there are no errors
	if err == nil {
		marketDataClient = client
		return nil
	}

	return errors.New("error setting token")
}

func newClient() *MarketDataClient {
	// Initialize a new MarketDataClient with default resty client and debug mode disabled.
	client := &MarketDataClient{
		Client: resty.New(),
		debug:  false,
	}

	// Set the default rate limit reset time.
	client.setDefaultResetTime()

	// Set the client environment to production.
	client.Environment(Production)

	// Set the "User-Agent" header to include the SDK version.
	client.Client.SetHeader("User-Agent", "sdk-go/"+Version)

	// Enable tracing for the client to facilitate debugging.
	client.Client.EnableTrace()

	// Set a default timeout of 95 seconds for all requests.
	client.Client.SetTimeout(95 * time.Second)

	// Set the OnBeforeRequest hook to perform actions before sending a request.
	// Currently, this hook does not perform any actions but can be used for logging or modifying requests.
	client.Client.OnBeforeRequest(func(c *resty.Client, req *resty.Request) error {
		// This is a placeholder for pre-request actions such as logging the request URL.
		return nil
	})

	// Set the OnAfterResponse hook to perform actions after receiving a response.
	// This hook updates the rate limit information and logs the request and response details.
	client.Client.OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
		// Update the client's rate limit information based on the response headers.
		client.updateRateLimit(resp)
		// Add logs from the request and response for debugging purposes.
		client.addLogFromRequestResponse(resp.Request, resp)
		// Placeholder for additional post-response actions such as logging or processing the response.
		return nil
	})

	return client
}

// Timeout sets the request timeout for the MarketDataClient.
//
// This method allows users to specify a custom timeout duration for all HTTP requests
// made by the client. The timeout duration is specified in seconds. Setting a timeout
// helps in preventing indefinitely hanging requests in case of network issues or slow
// server responses.
//
// By default the client has a timeout of 95 seconds. The timeout may be lowered, but it cannot be increased.
// Valid timeouts are between 1 and 95 seconds. Setting the timeout to any invalid integer will cause the
// client to use the default timeout of 95 seconds.
//
// # Parameters
//
//   - int: The timeout duration in seconds. A duration of 0 means no timeout.
func (c *MarketDataClient) Timeout(seconds int) {
	if seconds > 95 || seconds < 0 || seconds == 0 {
		seconds = 95
	}

	c.Client.SetTimeout(time.Duration(seconds) * time.Second)
}

// updateRateLimit updates the client's rate limit information based on the response headers.
// It extracts the rate limit values from the response headers and updates the client's rate limit fields.
func (c *MarketDataClient) updateRateLimit(resp *resty.Response) {
	// Lock the mutex before updating the shared rate limit fields to ensure thread safety.
	c.mu.Lock()
	defer c.mu.Unlock() // Ensure the mutex is unlocked after updating.

	// Extract rate limit headers from the response.
	limitHeader := resp.Header().Get("X-Api-Ratelimit-Limit")
	remainingHeader := resp.Header().Get("X-Api-Ratelimit-Remaining")
	resetHeader := resp.Header().Get("X-Api-Ratelimit-Reset")

	// Log errors if any of the required rate limit headers are missing.
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

	// Convert the rate limit values from strings to appropriate numeric types.
	limitVal, err := strconv.Atoi(limitHeader) // Convert the limit value to an integer.
	if err != nil {
		log.Printf("Error converting limit header to int: %v", err)
		return
	}

	remainingVal, err := strconv.Atoi(remainingHeader) // Convert the remaining value to an integer.
	if err != nil {
		log.Printf("Error converting remaining header to int: %v", err)
		return
	}

	resetVal, err := strconv.ParseInt(resetHeader, 10, 64) // Convert the reset timestamp to an int64.
	if err != nil {
		log.Printf("Error converting reset header to int64: %v", err)
		return
	}

	// Update the client's rate limit fields with the new values.
	c.RateLimitLimit = limitVal
	c.RateLimitRemaining = remainingVal
	c.RateLimitReset = time.Unix(resetVal, 0) // Convert the reset timestamp to a time.Time value.
}

// prepareAndExecuteRequest prepares the request based on the provided baseRequest and executes it.
// It returns the response from the server or an error if the request preparation or execution fails.
func (c *MarketDataClient) prepareAndExecuteRequest(br *baseRequest, result interface{}) (*resty.Response, error) {

	// Check for any errors in the base request.
	if err := br.getError(); err != nil {
		return nil, err
	}

	// Check if the client's rate limit has been exceeded before proceeding with the request.
	if c.RateLimitExceeded() {
		return nil, errors.New("rate limit exceeded")
	}

	// Initialize the Resty request and set the result type if provided.
	req := br.getResty()
	if result != nil {
		req = req.SetResult(result)
	}

	// Retrieve and parse the parameters from the base request.
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

	// Get the path for the request.
	path, err := br.getPath()
	if err != nil {
		return nil, err
	}

	// Execute the GET request to the specified path.
	resp, err := req.Get(path)
	if err != nil {
		return resp, err
	}

	// Check if the response status is not successful and handle errors accordingly.
	if !resp.IsSuccess() {
		var result map[string]interface{}
		_ = json.Unmarshal(resp.Body(), &result) // Attempt to unmarshal the response body into a map.
		if errMsg, ok := result["errmsg"]; ok {
			// Return an error with the non-OK status and the error message from the response.
			return resp, fmt.Errorf("received non-OK status: %s, error message: %v, URL: %s", resp.Status(), errMsg, resp.Request.URL)
		}
		// Return an error with the non-OK status if no specific error message is found in the response.
		return resp, fmt.Errorf("received non-OK status: %s for URL: %s", resp.Status(), resp.Request.URL)
	}

	return resp, nil
}

// getFromRequest executes a prepared request and returns the response.
// It handles any errors that occur during the request execution and checks for errors in the response.
// If an error is found in the response, it is returned as part of the response object.
//
// # Parameters
//
//   - br: A pointer to a baseRequest object containing the request details.
//   - result: An interface where the result of the request will be stored if successful.
//
// # Returns
//
//   - A pointer to a resty.Response object containing the response from the server.
//   - An error object if an error occurred during the request execution or if the response contains an error.
func (c *MarketDataClient) getFromRequest(br *baseRequest, result interface{}) (*resty.Response, error) {
	// Execute the prepared request and capture the response and any error.
	resp, err := c.prepareAndExecuteRequest(br, result)
	if err != nil {
		// Return the response and the error if an error occurred during request execution.
		return resp, err
	}

	// Check if the response contains an error and return it if present.
	if resp.Error() != nil {
		// Handle unmarshalling error by returning the error contained in the response.
		return resp, resp.Error().(error)
	}

	// Return the response and nil indicating no error occurred.
	return resp, nil
}

// getRawResponse executes a prepared request without processing the response.
// This function is useful when the caller needs the raw response for custom processing.
//
// # Parameters
//
//   - br: A pointer to a baseRequest object containing the request details.
//
// # Returns
//
//   - A pointer to a resty.Response object containing the raw response from the server.
//   - An error object if an error occurred during the request execution.
func (c *MarketDataClient) getRawResponse(br *baseRequest) (*resty.Response, error) {
	return c.prepareAndExecuteRequest(br, nil)
}

// GetClient checks for an existing instance of MarketDataClient and returns it.
// If the client is not already initialized, it attempts to initialize it.
//
// # Returns
//
//   - *MarketDataClient: A pointer to the existing or newly initialized MarketDataClient instance.
//   - error: An error object if the client cannot be initialized.
func GetClient() (*MarketDataClient, error) {
	// Check if the global client exists
	if marketDataClient == nil {
		// Attempt to initialize the client if it's not already
		err := tryNewClient()
		if err != nil {
			return nil, err // Return the error if client initialization fails
		}
	}

	// Return the global client if it is initialized
	return marketDataClient, nil
}

// Environment configures the base URL of the MarketDataClient based on the provided environment string.
// This method allows the client to switch between different environments such as production, testing, and development.
//
// # Parameters
//
//   - string: A string representing the environment to configure. Accepted values are "prod", "test", and "dev".
//
// # Returns
//
//   - *MarketDataClient: A pointer to the *MarketDataClient instance with the configured environment. This allows for method chaining.
//
// If an invalid environment is provided, the client's Error field is set, and the same instance is returned.
func (c *MarketDataClient) Environment(env Environment) error {
	if c == nil || c.Client == nil {
		return errors.New("MarketDataClient is nil")
	}

	var baseURL string
	switch env {
	case Production:
		baseURL = prodProtocol + "://" + prodHost // Set baseURL for production environment
	case Test:
		baseURL = testProtocol + "://" + testHost // Set baseURL for testing environment
	case Development:
		baseURL = devProtocol + "://" + devHost // Set baseURL for development environment
	default:
		return fmt.Errorf("invalid environment: %s", env) // Set error for invalid environment
	}

	c.Client.SetBaseURL(baseURL) // Configure the client with the determined baseURL

	return nil
}
func tryNewClient() error {
	// Default to Production if MARKETDATA_ENV is empty, doesn't exist, or is not a valid option
	token := os.Getenv("MARKETDATA_TOKEN") // Retrieve the market data token from environment variables

	if token != "" {
		err := NewClient(token)

		if err != nil {
			return err
		}
		return nil

	}
	return errors.New("env variable MARKETDATA_TOKEN not set")
}

// It also attempts to retrieve the "MARKETDATA_ENV" variable. If "MARKETDATA_ENV" is empty, doesn't exist, or doesn't use a valid option, it defaults to prodEnv.
// A new MarketDataClient instance is created and configured with the environment and token, then assigned to the global marketDataClient variable.
func init() {
	envValue := os.Getenv("MARKETDATA_ENV")
	env := Environment(envValue) // Convert the string value to Environment type

	// Default to Production if MARKETDATA_ENV is empty, doesn't exist, or is not a valid option
	if env != Production && env != Test && env != Development {
		env = Production
	}

	err := tryNewClient()
	if err != nil {
		fmt.Println("Error initializing MarketDataClient:", err)
		return
	}

	// Assign the environment to the global marketDataClient after successful initialization
	if marketDataClient != nil {
		marketDataClient.Environment(env)
	}
}

// Token configures the authentication token for the MarketDataClient.
// This method sets the authentication scheme to "Bearer" and assigns the provided bearerToken for subsequent requests.
// It also makes an initial request to the MarketData API to authorize the token and fetch rate limit information.
//
// # Parameters
//
//   - string: A string representing the bearer token to be used for API requests.
//
// # Returns
//
//   - error: An error if the authorization was not successful or nil if it was.
//
// # Notes
//
// If an error occurs during the initial authorization request or if the response indicates a failure, the client remains unmodified. The token will only be set if authorization is successful.
func (c *MarketDataClient) Token(bearerToken string) error {
	if c == nil || c.Client == nil {
		return fmt.Errorf("MarketDataClient is nil")
	}

	// Create a temporary client to make the initial request without modifying the original client
	tempClient := resty.New().SetAuthScheme("Bearer").SetAuthToken(bearerToken)

	// Make an initial request to authorize the token
	resp, err := tempClient.R().Get(user_endpoint)
	if err != nil {
		return err // Return error if there's an issue with the request
	}
	if !resp.IsSuccess() {
		return fmt.Errorf("invalid token. received non-OK status: %s", resp.Status()) // Return error for non-successful response
	}

	// If the token is valid, set the authentication scheme and token on the original client
	c.Client.SetAuthScheme("Bearer")
	c.Client.SetAuthToken(bearerToken)

	// Make a second request to load the rate limit information
	resp, err = c.Client.R().Get(user_endpoint)
	if err != nil {
		return err
	}

	if resp.IsSuccess() {
		return nil
	}

	return fmt.Errorf("invalid token. received non-OK status: %s", resp.Status()) // Return error for non-successful response
}
