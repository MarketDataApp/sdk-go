package client

import "fmt"

// IndicesCandlesRequest represents a request to the /v1/indices/candles endpoint.
type IndicesCandlesRequest struct {
	*baseRequest
	candleParams *CandleParams
	dateParams   *DateParams
}

// Resolution sets the resolution parameter for the IndicesCandlesRequest.
func (icr *IndicesCandlesRequest) Resolution(q string) *IndicesCandlesRequest {
	if icr == nil {
		return nil
	}
	err := icr.candleParams.SetResolution(q)
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
	err := icr.candleParams.SetSymbol(q)
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
func (icr *IndicesCandlesRequest) getParams() ([]MarketDataParam, error) {
	if icr == nil {
		return nil, fmt.Errorf("IndicesCandlesRequest is nil")
	}
	params := []MarketDataParam{icr.dateParams, icr.candleParams}
	return params, nil
}

// Get sends the IndicesCandlesRequest and returns the CandlesResponse along with the MarketDataResponse.
// It returns an error if the request fails.
func (icr *IndicesCandlesRequest) Get() (*IndicesCandlesResponse, *MarketDataResponse, error) {
	if icr == nil {
		return nil, nil, fmt.Errorf("IndicesCandlesRequest is nil")
	}
	var icrResp IndicesCandlesResponse
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
	baseReq.path = Paths[1]["indices"]["candles"]

	icr := &IndicesCandlesRequest{
		baseRequest:  baseReq,
		dateParams:   &DateParams{},
		candleParams: &CandleParams{},
	}

	baseReq.child = icr

	return icr
}
