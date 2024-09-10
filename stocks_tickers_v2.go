// Package stocks provides the /stocks endpoints
package client

import (
	"context"
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// StockTickersRequestV2 represents a request to the /v2/stocks/tickers/ endpoint.
// It encapsulates the date parameter to be used in the request.
// This struct provides the method DateKey() to set this parameter.
//
// # Setter Methods
//
//   - DateKey(q string) *StockTickersRequestV2: Sets the date parameter for the StockTickersRequestV2.
type StockTickersRequestV2 struct {
	*baseRequest
	dateKey *parameters.DateKeyParam
}

// DateKey sets the date parameter for the StockTickersRequestV2.
// This method is used to specify the date for which the stock tickers data is requested.
//
// # Parameters
//
//   - string: A string representing the date to be set.
//
// # Returns
//
//   - *StockTickersRequestV2: This method returns a pointer to the StockTickersRequestV2 instance it was called on. This allows for method chaining.
func (str *StockTickersRequestV2) DateKey(q string) *StockTickersRequestV2 {
	if str == nil {
		return nil
	}
	err := str.dateKey.SetDateKey(q)
	if err != nil {
		str.Error = err
	}
	return str
}

// getParams packs the StockTickersRequestV2 struct into a slice of interface{} and returns it.
func (str *StockTickersRequestV2) getParams() ([]parameters.MarketDataParam, error) {
	if str == nil {
		return nil, fmt.Errorf("StockTickersRequestV2 is nil")
	}
	params := []parameters.MarketDataParam{str.dateKey}
	return params, nil
}

// Packed sends the StockTickersRequestV2 with the provided context and returns the TickersResponse.
// This method checks if the StockTickersRequestV2 receiver is nil, returning an error if true.
// Otherwise, it proceeds to send the request and returns the TickersResponse along with any error encountered during the request.
//
// # Parameters
//
//   - ctx context.Context: The context to use for the request execution.
//
// # Returns
//
//   - *models.TickersResponse: A pointer to the TickersResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (str *StockTickersRequestV2) Packed(ctx context.Context) (*models.TickersResponse, error) {
	if str == nil {
		return nil, fmt.Errorf("StockTickersRequestV2 is nil")
	}
	var trResp models.TickersResponse
	_, err := str.baseRequest.client.getFromRequest(ctx, str.baseRequest, &trResp)
	if err != nil {
		return nil, err
	}

	return &trResp, nil
}

// Get sends the StockTickersRequestV2 with the provided context, unpacks the TickersResponse, and returns a slice of Ticker.
// It returns an error if the request or unpacking fails. This method is crucial for obtaining the actual stock tickers data
// from the stock tickers request. The method first checks if the StockTickersRequestV2 receiver is nil, which would
// result in an error as the request cannot be sent. It then proceeds to send the request using the Packed method.
// Upon receiving the response, it unpacks the data into a slice of Ticker using the Unpack method from the response.
//
// # Parameters
//
//   - ctx context.Context: The context to use for the request execution.
//
// # Returns
//
//   - []models.Ticker: A slice of Ticker containing the unpacked tickers data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (str *StockTickersRequestV2) Get(ctx context.Context) ([]models.Ticker, error) {
	if str == nil {
		return nil, fmt.Errorf("StockTickersRequestV2 is nil")
	}

	// Use the Packed method to make the request
	trResp, err := str.Packed(ctx)
	if err != nil {
		return nil, err
	}

	// Unpack the data using the Unpack method in the response
	data, err := trResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// StockTickers creates a new StockTickersRequestV2 and associates it with the default client. This function initializes the request
// with the default date parameter and sets the request path based on
// the predefined endpoints for stock tickers.
//
// # Returns
//
//   - *StockTickersRequestV2: A pointer to the newly created StockTickersRequestV2 with default parameters and associated client.
func StockTickers() *StockTickersRequestV2 {
	baseReq := newBaseRequest()
	baseReq.path = endpoints[2]["stocks"]["tickers"]

	str := &StockTickersRequestV2{
		baseRequest: baseReq,
		dateKey:     &parameters.DateKeyParam{},
	}

	// Set the date to the current time
	baseReq.child = str

	return str
}
