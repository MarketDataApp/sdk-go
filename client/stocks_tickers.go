// Package stocks provides the /stocks endpoints
package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// TickersRequest represents a request to the /stocks/tickers endpoint.
type TickersRequest struct {
	*baseRequest
	dateKey *parameters.DateKeyParam
}

// Date sets the date parameter for the TickersRequest.
func (tr *TickersRequest) DateKey(q string) *TickersRequest {
	if tr == nil {
		return nil
	}
	err := tr.dateKey.SetDateKey(q)
	if err != nil {
		tr.Error = err
	}
	return tr
}

// GetParams packs the TickersRequest struct into a slice of interface{} and returns it.
func (tr *TickersRequest) getParams() ([]parameters.MarketDataParam, error) {
	if tr == nil {
		return nil, fmt.Errorf("TickersRequest is nil")
	}
	params := []parameters.MarketDataParam{tr.dateKey}
	return params, nil
}

// StockTickers creates a new TickersRequest and associates it with the provided client.
// If no client is provided, it uses the default client.
func StockTickers(client ...*MarketDataClient) *TickersRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[2]["stocks"]["tickers"]

	tr := &TickersRequest{
		baseRequest: baseReq,
		dateKey:     &parameters.DateKeyParam{},
	}

	// Set the date to the current time
	baseReq.child = tr

	return tr
}

// Packed sends the TickersRequest and returns the TickersResponse.
// It returns an error if the request fails.
func (tr *TickersRequest) Packed() (*models.TickersResponse, error) {
	if tr == nil {
		return nil, fmt.Errorf("TickersRequest is nil")
	}
	var trResp models.TickersResponse
	_, err := tr.baseRequest.client.GetFromRequest(tr.baseRequest, &trResp)
	if err != nil {
		return nil, err
	}

	return &trResp, nil
}

// Get sends the TickersRequest, unpacks the TickersResponse and returns the data.
// It returns an error if the request or unpacking fails.
func (tr *TickersRequest) Get() ([]models.Ticker, error) {
	if tr == nil {
		return nil, fmt.Errorf("TickersRequest is nil")
	}
	
	// Use the Packed method to make the request
	trResp, err := tr.Packed()
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