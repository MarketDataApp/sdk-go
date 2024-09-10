// Package client provides a Go SDK for interacting with the Market Data API.
// It includes functionality for making API requests, handling responses,
// managing rate limits, and logging. The SDK supports various data types
// including stocks, options, and market status information.
//
// # Usage
//
// To use the SDK, you first need to create an instance of MarketDataClient
// using the GetClient function. This client will then be used to make
// API requests to the Market Data API.
//
// # Example
//
//	client := GetClient()
//	client.Debug(true) // Enable debug mode to log detailed request and response information
//	quote, err := client.StockQuotes().Symbol("AAPL").Get()
//
// # Authentication
//
// The SDK uses an API token for authentication. The token can be set as an
// environment variable (MARKETDATA_TOKEN) or directly in your code. However,
// storing tokens in your code is not recommended for security reasons.
//
// # Rate Limiting
//
// The MarketDataClient automatically tracks and manages the API's rate limits.
// You can check if the rate limit has been exceeded with the RateLimitExceeded method.
//
// # Logging
//
// The SDK logs all unsuccessful (400-499 and 500-599) responses to specific log files
// based on the response status code. Successful responses (200-299) are logged when
// debug mode is enabled. Logs include detailed information such as request and response
// headers, response status code, and the response body.
//
// # Debug Mode
//
// Debug mode can be enabled by calling the Debug method on the MarketDataClient instance.
// When enabled, the SDK will log detailed information about each request and response,
// which is useful for troubleshooting.
package client

import (
	"context"
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/go-resty/resty/v2"
)

type MarketDataPacked interface {
	IsValid() bool
	Unpack() any
}

// baseRequest is a struct that represents a basic request in the Market Data Client package.
// It contains a request of type *resty.Request, a path of type string, a client of type *MarketDataClient,
// a child of type any, and an Error of type error.
type baseRequest struct {
	request *resty.Request
	path    string
	client  *MarketDataClient
	child   any
	Error   error
}

// getParams calls the getParams method of the appropriate MarketDataRequest.
func (br *baseRequest) getParams() ([]parameters.MarketDataParam, error) {
	if br == nil || br.child == nil {
		return []parameters.MarketDataParam{}, nil
	}

	// Check if child is of type *baseRequest
	if _, ok := br.child.(*baseRequest); ok {
		return nil, fmt.Errorf("child is of type *baseRequest, stopping recursion")
	}

	if msr, ok := br.child.(*MarketStatusRequest); ok {
		params, err := msr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if tr, ok := br.child.(*StockTickersRequestV2); ok {
		params, err := tr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if scr, ok := br.child.(*StockCandlesRequest); ok {
		params, err := scr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if sbcr, ok := br.child.(*BulkStockCandlesRequest); ok {
		params, err := sbcr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if scr, ok := br.child.(*StockCandlesRequestV2); ok {
		params, err := scr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if sqr, ok := br.child.(*StockQuoteRequest); ok {
		params, err := sqr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if bsqr, ok := br.child.(*BulkStockQuotesRequest); ok {
		params, err := bsqr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if snr, ok := br.child.(*StockNewsRequest); ok {
		params, err := snr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if iqr, ok := br.child.(*IndexQuoteRequest); ok {
		params, err := iqr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if ser, ok := br.child.(*StockEarningsRequest); ok {
		params, err := ser.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if icr, ok := br.child.(*IndicesCandlesRequest); ok {
		params, err := icr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if oer, ok := br.child.(*OptionsExpirationsRequest); ok {
		params, err := oer.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if olr, ok := br.child.(*OptionLookupRequest); ok {
		params, err := olr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if osr, ok := br.child.(*OptionStrikesRequest); ok {
		params, err := osr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if oqr, ok := br.child.(*OptionQuoteRequest); ok {
		params, err := oqr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if ocr, ok := br.child.(*OptionChainRequest); ok {
		params, err := ocr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if fcr, ok := br.child.(*FundCandlesRequest); ok {
		params, err := fcr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	return []parameters.MarketDataParam{}, nil
}

// getPath returns the path of the BaseRequest.
// It returns an error if the BaseRequest is nil.
func (br *baseRequest) getPath() (string, error) {
	if br == nil {
		return "", fmt.Errorf("path is nil")
	}

	if br.path == "" {
		return "", fmt.Errorf("path is empty")
	}

	return br.path, nil
}

// getResty returns the resty.Request for the BaseRequest.
func (br *baseRequest) getResty() *resty.Request {
	return br.request
}

func newBaseRequest(clients ...*MarketDataClient) *baseRequest {
	var mdClient *MarketDataClient
	var err error

	if len(clients) > 0 {
		mdClient = clients[0]
	} else {
		mdClient, err = GetClient()
		if mdClient == nil {
			return nil
		}
	}

	baseReq := &baseRequest{
		request: mdClient.R(),
		client:  mdClient,
		Error:   err,
	}

	return baseReq
}

// getError returns the error of the BaseRequest.
func (br *baseRequest) getError() error {
	return br.Error
}

// Raw executes the request and returns the raw resty.Response.
//
// # Returns
//
//   - *resty.Response: The raw response from the executed request.
//   - error: An error object if the baseRequest is nil, or if an error occurs during the request execution.
func (request *baseRequest) Raw(ctx context.Context) (*resty.Response, error) {
	if request == nil {
		return nil, fmt.Errorf("baseRequest is nil")
	}

	if request.client == nil {
		return nil, fmt.Errorf("MarketDataClient is nil")
	}

	response, err := request.client.getRawResponse(ctx, request)
	return response, err
}
