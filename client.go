// Package client provides a Go SDK for interacting with the Market Data API.
// The [Market Data Go Client] includes functionality for making API requests, handling responses,
// managing rate limits, and logging. The SDK supports various data types
// including stocks, options, indices, and market status information.
//
// # Get Started Quickly with the MarketDataClient
//
// 1. Use [GetClient] to fetch the [MarketDataClient] instance and set the API token.
// 2. Turn on Debug mode to log detailed request and response information to disk as you learn how to use the SDK.
// 3. Make a test request.
// 4. Check the rate limit in the client to keep track of your requests.
// 5. Check the in-memory logs to see the raw request and response details.
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

	"github.com/MarketDataApp/sdk-go/helpers/logging"
	"github.com/go-resty/resty/v2"
	_ "github.com/joho/godotenv/autoload"
)

// MarketDataClient struct defines the structure for the MarketData client instance.
// It embeds the resty.Client to inherit the HTTP client functionalities.
// Additionally, it includes fields for managing rate limits and synchronization,
// as well as an error field for capturing any errors that occur during API calls.
// The debug field is used to control logging verbosity.
//
// # Setter Methods
//
//   - Debug(bool) *MarketDataClient: Enables or disables debug mode for logging detailed request and response information.
//   - Environment(string) *MarketDataClient: Sets the environment for the MarketDataClient.
//   - Timeout(int) *MarketDataClient: Sets the request timeout for the MarketDataClient.
//   - Token(string) *MarketDataClient: Sets the authentication token for the MarketDataClient.
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
	Error              error      // Error captures any errors that occur during the execution of API calls.
	debug              bool       // debug indicates whether debug mode is enabled, controlling the verbosity of logs.
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
func (c *MarketDataClient) addLogFromRequestResponse(req *resty.Request, resp *resty.Response) {
	// Redact sensitive information from request headers.
	redactedHeaders := redactAuthorizationHeader(req.Header)
	// Extract response headers.
	resHeaders := resp.Header()
	// Attempt to extract rate limit consumed information from the response.
	rateLimitConsumed, err := getRateLimitConsumed(resp)
	if err != nil {
		// If an error occurs, set the client's error field and return early.
		c.Error = err
		return
	}
	// Attempt to extract the ray ID from the response.
	rayID, err := getRayIDFromResponse(resp)
	if err != nil {
		// If an error occurs, set the client's error field and return early.
		c.Error = err
		return
	}
	// Calculate the latency of the request.
	delay := getLatencyFromRequest(req)
	// Extract the status code from the response.
	status := resp.StatusCode()
	// Convert the response body to a string.
	body := string(resp.Body())

	// Create a new log entry with the gathered information.
	logEntry := logging.AddToLog(GetLogs(), time.Now(), rayID, req.URL, rateLimitConsumed, delay, status, body, redactedHeaders, resHeaders)
	// If debug mode is enabled and the log entry is not nil, pretty print the log entry.
	if c.debug && logEntry != nil {
		logEntry.PrettyPrint()
	}
	// If the log entry is not nil, write it to the log.
	if logEntry != nil {
		logEntry.WriteToLog(c.debug)
	}
}

// getEnvironment determines the environment the client is operating in based on the host URL.
// It parses the host URL to extract the hostname and matches it against predefined hostnames
// for production, testing, and development environments. If a match is found, it returns the
// corresponding environment name; otherwise, it defaults to "Unknown".
func (c *MarketDataClient) getEnvironment() string {
	u, err := url.Parse(c.Client.HostURL) // Parse the host URL to extract the hostname.
	if err != nil {
		log.Printf("Error parsing host URL: %v", err) // Log any error encountered during URL parsing.
		return "Unknown"                              // Default to "Unknown" if there's an error in parsing the URL.
	}
	switch u.Hostname() { // Match the extracted hostname against predefined hostnames.
	case prodHost:
		return prodEnv // Return the production environment name if matched.
	case testHost:
		return testEnv // Return the testing environment name if matched.
	case devHost:
		return devEnv // Return the development environment name if matched.
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
func new() *MarketDataClient {
	// Initialize a new MarketDataClient with default resty client and debug mode disabled.
	client := &MarketDataClient{
		Client: resty.New(),
		debug:  false,
	}

	// Set the default rate limit reset time.
	client.setDefaultResetTime()

	// Set the client environment to production.
	client.Environment(prodEnv)

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

	// Return the initialized MarketDataClient instance.
	return client
}

// Debug is a method that enables or disables the debug mode of the client.
// Debug mode will result in the request and response headers being printed to
// the terminal with each request.
//
// # Parameters
//
//   - enable: A boolean value indicating whether to enable or disable debug mode. By default, debug mode is disabled.
//
// # Returns
//
//   - *MarketDataClient: A pointer to the MarketDataClient instance, allowing for method chaining.
func (c *MarketDataClient) Debug(enable bool) *MarketDataClient {
	c.debug = enable
	return c
}

// Timeout sets the request timeout for the MarketDataClient.
//
// This method allows users to specify a custom timeout duration for all HTTP requests
// made by the client. The timeout duration is specified in seconds. Setting a timeout
// helps in preventing indefinitely hanging requests in case of network issues or slow
// server responses.
//
// # Parameters
//
//   - int: The timeout duration in seconds. A duration of 0 means no timeout. By default the client has a timeout of 95 seconds.
//
// # Returns
//
//   - *MarketDataClient: A pointer to the MarketDataClient instance, allowing for method chaining.
func (c *MarketDataClient) Timeout(seconds int) *MarketDataClient {
	c.Client.SetTimeout(time.Duration(seconds) * time.Second)
	return c
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

// GetClient initializes and returns a singleton instance of MarketDataClient.
// If a token is provided as an argument, it creates a new client instance with that token.
// If no token is provided, it attempts to use a token from the environment variable "MARKETDATA_TOKEN".
// This function ensures that only one instance of the client is active at any time,
// reusing the existing instance if no new token is provided and no errors are present in the current client.
//
// # Parameters
//
//   - ...string: A variadic string parameter where the first element, if provided, is used as the authentication token for the [MarketDataClient]. If not provided, the function looks for a token in the "MARKETDATA_TOKEN" environment variable.
//
// # Returns
//
//   - *MarketDataClient: A pointer to the initialized MarketDataClient instance. This client is configured with the provided or environment-sourced token.
//   - error: An error object that indicates a failure in client initialization. Possible errors include missing token (if no token is provided and none is found in the environment) and any errors encountered during the client's token configuration process.
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
	client := new()
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
func (c *MarketDataClient) Environment(env string) *MarketDataClient {
	var baseURL string
	switch env {
	case prodEnv:
		baseURL = prodProtocol + "://" + prodHost // Set baseURL for production environment
	case testEnv:
		baseURL = testProtocol + "://" + testHost // Set baseURL for testing environment
	case devEnv:
		baseURL = devProtocol + "://" + devHost // Set baseURL for development environment
	default:
		c.Error = fmt.Errorf("invalid environment: %s", env) // Set error for invalid environment
		return c
	}

	c.Client.SetBaseURL(baseURL) // Configure the client with the determined baseURL

	return c
}

// init initializes the global marketDataClient with a token and environment fetched from environment variables.
// It retrieves the "MARKETDATA_TOKEN" variable and uses it to configure the marketDataClient.
// It also attempts to retrieve the "MARKETDATA_ENV" variable. If "MARKETDATA_ENV" is empty, doesn't exist, or doesn't use a valid option, it defaults to prodEnv.
// A new MarketDataClient instance is created and configured with the environment and token, then assigned to the global marketDataClient variable.
func init() {
	token := os.Getenv("MARKETDATA_TOKEN") // Retrieve the market data token from environment variables
	env := os.Getenv("MARKETDATA_ENV")     // Attempt to retrieve the environment from environment variables

	// Default to prodEnv if MARKETDATA_ENV is empty, doesn't exist, or is not a valid option
	if env != prodEnv && env != testEnv && env != devEnv {
		env = prodEnv
	}

	// Proceed only if the token is not empty
	if token != "" {
		// Create and configure a new MarketDataClient instance with the environment and token
		client := new().Environment(env).Token(token)
		if client.Error != nil {
			marketDataClient = client
		}
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
//   - *MarketDataClient: A pointer to the MarketDataClient instance with the configured authentication token, which allows for method chaining.
//
// If an error occurs during the initial request or if the response indicates a failure, the client's Error field is set,
// and the same instance is returned.
func (c *MarketDataClient) Token(bearerToken string) *MarketDataClient {
	// Set the authentication scheme to "Bearer"
	c.Client.SetAuthScheme("Bearer")

	// Set the authentication token
	c.Client.SetAuthToken(bearerToken)

	// Make an initial request to authorize the token and load the rate limit information
	resp, err := c.Client.R().Get("https://api.marketdata.app/user/")
	if err != nil {
		c.Error = err // Set error if there's an issue with the request
		return c
	}
	if !resp.IsSuccess() {
		err = fmt.Errorf("received non-OK status: %s", resp.Status()) // Create error for non-successful response
		c.Error = err
		return c
	}

	return c
}

// GetLogs retrieves a pointer to the HttpRequestLogs instance, allowing access to the logs collected during HTTP requests.
// This method is primarily used for debugging and monitoring purposes, providing insights into the HTTP request lifecycle and any issues that may have occurred.
//
// # Returns
//
//   - *logging.HttpRequestLogs: A pointer to the HttpRequestLogs instance containing logs of HTTP requests.
func GetLogs() *logging.HttpRequestLogs {
	return logging.Logs
}
