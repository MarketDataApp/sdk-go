package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// IndicesCandlesRequest represents a request to the /v1/indices/candles endpoint.
type IndicesCandlesRequest struct {
	*baseRequest
	resolutionParams *parameters.ResolutionParams
	symbolParams     *parameters.SymbolParams
	dateParams       *parameters.DateParams
}

// Resolution sets the resolution parameter for the IndicesCandlesRequest.
func (icr *IndicesCandlesRequest) Resolution(q string) *IndicesCandlesRequest {
	if icr == nil {
		return nil
	}
	err := icr.resolutionParams.SetResolution(q)
	if err != nil {
		icr.Error = err
	}
	return icr
}

// Symbol sets the symbol parameter for the IndicesCandlesRequest.
func (icr *IndicesCandlesRequest) Symbol(q string) *IndicesCandlesRequest {
	if icr == nil {
		return nil
	}
	err := icr.symbolParams.SetSymbol(q)
	if err != nil {
		icr.Error = err
	}
	return icr
}

// Date sets the date parameter of the IndicesCandlesRequest.
func (icr *IndicesCandlesRequest) Date(q interface{}) *IndicesCandlesRequest {
	err := icr.dateParams.SetDate(q)
	if err != nil {
		icr.baseRequest.Error = err
	}
	return icr
}

// From sets the from parameter of the IndicesCandlesRequest.
func (icr *IndicesCandlesRequest) From(q interface{}) *IndicesCandlesRequest {
	err := icr.dateParams.SetFrom(q)
	if err != nil {
		icr.baseRequest.Error = err
	}
	return icr
}

// To sets the to parameter of the IndicesCandlesRequest.
func (icr *IndicesCandlesRequest) To(q interface{}) *IndicesCandlesRequest {
	err := icr.dateParams.SetTo(q)
	if err != nil {
		icr.baseRequest.Error = err
	}
	return icr
}

// Countback sets the countback parameter of the IndicesCandlesRequest.
func (icr *IndicesCandlesRequest) Countback(q int) *IndicesCandlesRequest {
	err := icr.dateParams.SetCountback(q)
	if err != nil {
		icr.baseRequest.Error = err
	}
	return icr
}

// GetParams packs the IndicesCandlesRequest struct into a slice of interface{} and returns it.
func (icr *IndicesCandlesRequest) getParams() ([]parameters.MarketDataParam, error) {
	if icr == nil {
		return nil, fmt.Errorf("IndicesCandlesRequest is nil")
	}
	params := []parameters.MarketDataParam{icr.dateParams, icr.symbolParams, icr.resolutionParams}
	return params, nil
}

// Get sends the IndicesCandlesRequest and returns the CandlesResponse.
// It returns an error if the request fails.
func (icr *IndicesCandlesRequest) Packed() (*models.IndicesCandlesResponse, error) {
	if icr == nil {
		return nil, fmt.Errorf("IndicesCandlesRequest is nil")
	}
	var icrResp models.IndicesCandlesResponse
	_, err := icr.baseRequest.client.GetFromRequest(icr.baseRequest, &icrResp)
	if err != nil {
		return nil, err
	}

	return &icrResp, nil
}

// Get sends the IndicesCandlesRequest, unpacks the IndicesCandlesResponse and returns a slice of IndexCandle.
// It returns an error if the request or unpacking fails.
func (icr *IndicesCandlesRequest) Get() ([]models.IndexCandle, error) {
	if icr == nil {
		return nil, fmt.Errorf("IndicesCandlesRequest is nil")
	}
	
	// Use the Packed method to make the request
	icrResp, err := icr.Packed()
	if err != nil {
		return nil, err
	}

	// Unpack the data using the Unpack method in the response
	data, err := icrResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// IndexCandles creates a new CandlesRequest and associates it with the provided client.
// If no client is provided, it uses the default client.
func IndexCandles(client ...*MarketDataClient) *IndicesCandlesRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[1]["indices"]["candles"]

	icr := &IndicesCandlesRequest{
		baseRequest:      baseReq,
		dateParams:       &parameters.DateParams{},
		resolutionParams: &parameters.ResolutionParams{},
		symbolParams:     &parameters.SymbolParams{},
	}

	baseReq.child = icr

	return icr
}
