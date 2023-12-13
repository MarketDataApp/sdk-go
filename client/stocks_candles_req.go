package client

import (
	"fmt"
)

// CandlesRequest represents a request to the /stocks/candles endpoint.
type StockCandlesRequestV2 struct {
	*baseRequest
	dateKey      *DateKeyParam
	candleParams *CandleParams

}

// Resolution sets the resolution parameter for the CandlesRequest.
func (cr *StockCandlesRequestV2) Resolution(q string) *StockCandlesRequestV2 {
	if cr == nil {
		return nil
	}
	err := cr.candleParams.SetResolution(q)
	if err != nil {
		cr.Error = err
	}
	return cr
}

// Symbol sets the symbol parameter for the CandlesRequest.
func (cr *StockCandlesRequestV2) Symbol(q string) *StockCandlesRequestV2 {
	if cr == nil {
		return nil
	}
	err := cr.candleParams.SetSymbol(q)
	if err != nil {
		cr.Error = err
	}
	return cr
}

// Date sets the date parameter for the CandlesRequest.
func (cr *StockCandlesRequestV2) DateKey(q string) *StockCandlesRequestV2 {
	if cr == nil {
		return nil
	}
	err := cr.dateKey.SetDateKey(q)
	if err != nil {
		cr.Error = err
	}
	return cr
}

// GetParams packs the CandlesRequest struct into a slice of interface{} and returns it.
func (cr *StockCandlesRequestV2) getParams() ([]MarketDataParam, error) {
	if cr == nil {
		return nil, fmt.Errorf("CandlesRequest is nil")
	}
	params := []MarketDataParam{cr.dateKey, cr.candleParams}
	return params, nil
}

// GetCandles sends the CandlesRequest and returns the CandlesResponse along with the MarketDataResponse.
// It returns an error if the request fails.
func (cr *StockCandlesRequestV2) Get() (*StockCandlesResponse, *MarketDataResponse, error) {
	if cr == nil {
		return nil, nil, fmt.Errorf("StockCandlesRequestV2 is nil")
	}
	var crResp StockCandlesResponse
	mdr, err := cr.baseRequest.client.GetFromRequest(cr.baseRequest, &crResp)
	if err != nil {
		return nil, nil, err
	}

	return &crResp, mdr, nil
}


// StockCandles creates a new CandlesRequest and associates it with the provided client.
// If no client is provided, it uses the default client.
func StockCandlesV2(client ...*MarketDataClient) *StockCandlesRequestV2 {
	baseReq := newBaseRequest(client...)
	baseReq.path = Paths[2]["stocks"]["candles"]

	cr := &StockCandlesRequestV2{
		baseRequest: baseReq,
		dateKey:     &DateKeyParam{},
		candleParams: &CandleParams{},
	}

	// Set the date to the current time
	baseReq.child = cr

	return cr
}