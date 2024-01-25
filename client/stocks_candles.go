package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// StockCandlesRequest represents a request to the /v1/stocks/candles endpoint.
type StockCandlesRequest struct {
	*baseRequest
	stockCandleParams *parameters.StockCandleParams
	resolutionParams  *parameters.ResolutionParams
	symbolParams      *parameters.SymbolParams
	dateParams        *parameters.DateParams
}

// Resolution sets the resolution parameter for the CandlesRequest.
func (cr *StockCandlesRequest) Resolution(q string) *StockCandlesRequest {
	if cr == nil {
		return nil
	}
	err := cr.resolutionParams.SetResolution(q)
	if err != nil {
		cr.Error = err
	}
	return cr
}

// Symbol sets the symbol parameter for the CandlesRequest.
func (cr *StockCandlesRequest) Symbol(q string) *StockCandlesRequest {
	if cr == nil {
		return nil
	}
	err := cr.symbolParams.SetSymbol(q)
	if err != nil {
		cr.Error = err
	}
	return cr
}

// Date sets the date parameter of the StockCandlesRequest.
func (scr *StockCandlesRequest) Date(q interface{}) *StockCandlesRequest {
	err := scr.dateParams.SetDate(q)
	if err != nil {
		scr.baseRequest.Error = err
	}
	return scr
}

// From sets the from parameter of the StockCandlesRequest.
func (scr *StockCandlesRequest) From(q interface{}) *StockCandlesRequest {
	err := scr.dateParams.SetFrom(q)
	if err != nil {
		scr.baseRequest.Error = err
	}
	return scr
}

// To sets the to parameter of the StockCandlesRequest.
func (scr *StockCandlesRequest) To(q interface{}) *StockCandlesRequest {
	err := scr.dateParams.SetTo(q)
	if err != nil {
		scr.baseRequest.Error = err
	}
	return scr
}

// Countback sets the countback parameter of the StockCandlesRequest.
func (scr *StockCandlesRequest) Countback(q int) *StockCandlesRequest {
	err := scr.dateParams.SetCountback(q)
	if err != nil {
		scr.baseRequest.Error = err
	}
	return scr
}

// AdjustSplits sets the AdjustSplits parameter for the StockCandlesRequest.
func (scr *StockCandlesRequest) AdjustSplits(q bool) *StockCandlesRequest {
	if scr == nil {
		return nil
	}
	scr.stockCandleParams.SetAdjustSplits(q)
	return scr
}

// AdjustDividends sets the AdjustDividends parameter for the StockCandlesRequest.
func (scr *StockCandlesRequest) AdjustDividends(q bool) *StockCandlesRequest {
	if scr == nil {
		return nil
	}
	scr.stockCandleParams.SetAdjustDividends(q)
	return scr
}

// Extended sets the Extended parameter for the StockCandlesRequest.
func (scr *StockCandlesRequest) Extended(q bool) *StockCandlesRequest {
	if scr == nil {
		return nil
	}
	scr.stockCandleParams.SetExtended(q)
	return scr
}

// Exchange sets the Exchange parameter for the StockCandlesRequest.
func (scr *StockCandlesRequest) Exchange(q string) *StockCandlesRequest {
	if scr == nil {
		return nil
	}
	err := scr.stockCandleParams.SetExchange(q)
	if err != nil {
		scr.baseRequest.Error = err
	}
	return scr
}

// GetParams packs the CandlesRequest struct into a slice of interface{} and returns it.
func (scr *StockCandlesRequest) getParams() ([]parameters.MarketDataParam, error) {
	if scr == nil {
		return nil, fmt.Errorf("StockCandlesRequest is nil")
	}
	params := []parameters.MarketDataParam{scr.dateParams, scr.symbolParams, scr.resolutionParams, scr.stockCandleParams}
	return params, nil
}

// Packed sends the StockCandlesRequest and returns the CandlesResponse.
// It returns an error if the request fails.
func (scr *StockCandlesRequest) Packed() (*models.StockCandlesResponse, error) {
	if scr == nil {
		return nil, fmt.Errorf("StockCandlesRequest is nil")
	}
	var scrResp models.StockCandlesResponse
	_, err := scr.baseRequest.client.GetFromRequest(scr.baseRequest, &scrResp)
	if err != nil {
		return nil, err
	}

	return &scrResp, nil
}

// StockCandles creates a new CandlesRequest and associates it with the provided client.
// If no client is provided, it uses the default client.
func StockCandles(client ...*MarketDataClient) *StockCandlesRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[1]["stocks"]["candles"]

	scr := &StockCandlesRequest{
		baseRequest:       baseReq,
		dateParams:        &parameters.DateParams{},
		resolutionParams:  &parameters.ResolutionParams{},
		symbolParams:      &parameters.SymbolParams{},
		stockCandleParams: &parameters.StockCandleParams{},
	}

	// Set the date to the current time
	baseReq.child = scr

	return scr
}

// Get sends the StockCandlesRequest, unpacks the StockCandlesResponse and returns a slice of StockCandle.
// It returns an error if the request or unpacking fails.
func (scr *StockCandlesRequest) Get() ([]models.StockCandle, error) {
	if scr == nil {
		return nil, fmt.Errorf("StockCandlesRequest is nil")
	}
	
	// Use the Packed method to make the request
	scrResp, err := scr.Packed()
	if err != nil {
		return nil, err
	}

	// Unpack the data using the Unpack method in the response
	data, err := scrResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// StockCandlesRequestV2 represents a request to the /v2/stocks/candles endpoint.
type StockCandlesRequestV2 struct {
	*baseRequest
	dateKey          *parameters.DateKeyParam
	resolutionParams *parameters.ResolutionParams
	symbolParams     *parameters.SymbolParams
}

// Resolution sets the resolution parameter for the CandlesRequest.
func (cr *StockCandlesRequestV2) Resolution(q string) *StockCandlesRequestV2 {
	if cr == nil {
		return nil
	}
	err := cr.resolutionParams.SetResolution(q)
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
	err := cr.symbolParams.SetSymbol(q)
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
func (cr *StockCandlesRequestV2) getParams() ([]parameters.MarketDataParam, error) {
	if cr == nil {
		return nil, fmt.Errorf("CandlesRequest is nil")
	}
	params := []parameters.MarketDataParam{cr.dateKey, cr.resolutionParams, cr.symbolParams}
	return params, nil
}

// Packed sends the CandlesRequest and returns the CandlesResponse.
// It returns an error if the request fails.
func (cr *StockCandlesRequestV2) Packed() (*models.StockCandlesResponse, error) {
	if cr == nil {
		return nil, fmt.Errorf("StockCandlesRequestV2 is nil")
	}
	var crResp models.StockCandlesResponse
	_, err := cr.baseRequest.client.GetFromRequest(cr.baseRequest, &crResp)
	if err != nil {
		return nil, err
	}

	return &crResp, nil
}

// StockCandles creates a new CandlesRequest and associates it with the provided client.
// If no client is provided, it uses the default client.
func StockCandlesV2(client ...*MarketDataClient) *StockCandlesRequestV2 {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[2]["stocks"]["candles"]

	cr := &StockCandlesRequestV2{
		baseRequest:      baseReq,
		dateKey:          &parameters.DateKeyParam{},
		resolutionParams: &parameters.ResolutionParams{},
		symbolParams:     &parameters.SymbolParams{},
	}

	// Set the date to the current time
	baseReq.child = cr

	return cr
}
