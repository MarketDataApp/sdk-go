package client

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
)

// marketDataClient is the default *MarketDataClient
var marketDataClient *MarketDataClient

const (
	Version = "1.1.0" // Version specifies the current version of the SDK.

	prodHost = "api.marketdata.app" // prodHost is the hostname for the production environment.
	testHost = "tst.marketdata.app" // testHost is the hostname for the testing environment.
	devHost  = "localhost"          // devHost is the hostname for the development environment.

	prodProtocol = "https" // prodProtocol specifies the protocol to use in the production environment.
	testProtocol = "https" // testProtocol specifies the protocol to use in the testing environment.
	devProtocol  = "http"  // devProtocol specifies the protocol to use in the development environment.
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

// getRateLimitConsumed extracts the rate limit consumed value from the response headers.
// It specifically looks for the "X-Api-Ratelimit-Consumed" header and attempts to convert its value to an integer.
//
// # Parameters
//
//   - resp: A pointer to a resty.Response from which the header will be extracted.
//
// # Returns
//
//   - int: The integer value of the "X-Api-Ratelimit-Consumed" header if present and successfully converted.
//   - error: An error if the header is missing or if the conversion to an integer fails.
func getRateLimitConsumed(resp *resty.Response) (int, error) {
	rateLimitConsumedStr := resp.Header().Get("X-Api-Ratelimit-Consumed")
	if rateLimitConsumedStr == "" {
		return 0, errors.New("error: missing 'x-Api-Ratelimit-Consumed' header")
	}
	rateLimitConsumed, err := strconv.Atoi(rateLimitConsumedStr)
	if err != nil {
		return 0, err
	}
	return rateLimitConsumed, nil
}

// getRayIDFromResponse extracts the "Cf-Ray" header value from the response.
//
// # Parameters
//
//   - resp: A pointer to a resty.Response from which the header will be extracted.
//
// # Returns
//
//   - string: The value of the "Cf-Ray" header if present.
//   - error: An error if the "Cf-Ray" header is missing.
func getRayIDFromResponse(resp *resty.Response) (string, error) {
	rayID := resp.Header().Get("Cf-Ray")
	if rayID == "" {
		return "", errors.New("Cf-Ray header not found")
	}
	return rayID, nil
}

// getLatencyFromRequest calculates the server processing time for a request.
//
// # Parameters
//
//   - req: A pointer to a resty.Request which has been executed and contains trace information.
//
// # Returns
//
//   - int64: The server processing time in milliseconds.
func getLatencyFromRequest(req *resty.Request) int64 {
	trace := req.TraceInfo()
	return trace.ServerTime.Milliseconds()
}

// redactAuthorizationHeader takes an http.Header object and returns a new http.Header object with the "Authorization" header value redacted.
// The redaction replaces the token with a string that has the same length but with the characters replaced by asterisks, except for the last four characters.
//
// # Parameters
//
//   - headers: The original http.Header object containing the headers.
//
// # Returns
//
//   - http.Header: A new http.Header object with the "Authorization" header value redacted if present.
func redactAuthorizationHeader(headers http.Header) http.Header {
	// Copy the headers so we don't modify the original
	copiedHeaders := make(http.Header)
	for k, v := range headers {
		copiedHeaders[k] = v
	}

	// Redact the Authorization header if it exists
	if _, ok := copiedHeaders["Authorization"]; ok {
		token := copiedHeaders.Get("Authorization")
		redactedToken := "Bearer " + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
		copiedHeaders.Set("Authorization", redactedToken)
	}

	return copiedHeaders
}
