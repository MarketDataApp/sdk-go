package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
	"github.com/go-resty/resty/v2"
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

// Get sends the IndicesCandlesRequest and returns the CandlesResponse along with the MarketDataResponse.
// It returns an error if the request fails.
func (icr *IndicesCandlesRequest) Get() (*models.IndicesCandlesResponse, *resty.Response, error) {
	if icr == nil {
		return nil, nil, fmt.Errorf("IndicesCandlesRequest is nil")
	}
	var icrResp models.IndicesCandlesResponse
	mdr, err := icr.baseRequest.client.GetFromRequest(icr.baseRequest, &icrResp)
	if err != nil {
		return nil, nil, err
	}

	return &icrResp, mdr, nil
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
