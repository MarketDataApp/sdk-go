// Package stocks provides the /stocks endpoints
package client

import (
	"fmt"
	"sync"
	"time"
)

// TickersRequest represents a request to the /stocks/tickers endpoint.
type TickersRequest struct {
	*baseRequest
	dateKey *DateKeyParam
}

// Date sets the date parameter for the TickersRequest.
func (tr *TickersRequest) Date(q interface{}) *TickersRequest {
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
func (tr *TickersRequest) getParams() ([]MarketDataParam, error) {
	if tr == nil {
		return nil, fmt.Errorf("TickersRequest is nil")
	}
	params := []MarketDataParam{tr.dateKey}
	return params, nil
}

// StockTickers creates a new TickersRequest and associates it with the provided client.
// If no client is provided, it uses the default client.
func StockTickers(client ...*MarketDataClient) *TickersRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = Paths[2]["stocks"]["tickers"]

	tr := &TickersRequest{
		baseRequest: baseReq,
		dateKey: &DateKeyParam{},
	}

	// Set the date to the current time
	tr.Date(time.Now())
	baseReq.child = tr

	return tr
}

// GetTickers sends the TickersRequest and returns the TickersResponse along with the MarketDataResponse.
// It returns an error if the request fails.
func (tr *TickersRequest) Get() (*TickersResponse, *MarketDataResponse, error) {
	if tr == nil {
		return nil, nil, fmt.Errorf("TickersRequest is nil")
	}
	var trResp TickersResponse
	mdr, err := tr.baseRequest.client.GetFromRequest(tr.baseRequest, &trResp)
	if err != nil {
		return nil, nil, err
	}

	return &trResp, mdr, nil
}

// CombineTickerResponses combines multiple TickersResponses into a single map.
func CombineTickerResponses(responses []*TickersResponse) (map[string]TickerObj, error) {
	tickerMap := make(map[string]TickerObj)
	var mutex sync.Mutex

	var wg sync.WaitGroup
	errors := make(chan error)

	for _, response := range responses {
		wg.Add(1)
		go func(response *TickersResponse) {
			defer wg.Done()
			responseMap, err := response.ToMap()
			if err != nil {
				errors <- err
				return
			}
			mutex.Lock()
			for key, value := range responseMap {
				tickerMap[key] = value
			}
			mutex.Unlock()
		}(response)
	}

	go func() {
		wg.Wait()
		close(errors)
	}()

	for err := range errors {
		if err != nil {
			return nil, err
		}
	}

	return tickerMap, nil
}
