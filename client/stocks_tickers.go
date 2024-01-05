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

// GetTickers sends the TickersRequest and returns the TickersResponse along with the MarketDataResponse.
// It returns an error if the request fails.
func (tr *TickersRequest) Get() (*models.TickersResponse, *MarketDataResponse, error) {
	if tr == nil {
		return nil, nil, fmt.Errorf("TickersRequest is nil")
	}
	var trResp models.TickersResponse
	mdr, err := tr.baseRequest.client.GetFromRequest(tr.baseRequest, &trResp)
	if err != nil {
		return nil, nil, err
	}

	return &trResp, mdr, nil
}
