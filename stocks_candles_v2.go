package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

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

// Packed sends the StockCandlesRequestV2 and returns the StockCandlesResponse.
// An optional MarketDataClient can be passed to replace the client used in the request.
//
// Parameters:
//   - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// Returns:
//   - *models.StockCandlesResponse: A pointer to the StockCandlesResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (scrV2 *StockCandlesRequestV2) Packed(optionalClients ...*MarketDataClient) (*models.StockCandlesResponse, error) {
	if scrV2 == nil {
		return nil, fmt.Errorf("StockCandlesRequestV2 is nil")
	}

	// Replace the client if an optional client is provided
	if len(optionalClients) > 0 && optionalClients[0] != nil {
		scrV2.baseRequest.client = optionalClients[0]
	}

	var scrResp models.StockCandlesResponse
	_, err := scrV2.baseRequest.client.GetFromRequest(scrV2.baseRequest, &scrResp)
	if err != nil {
		return nil, err
	}

	return &scrResp, nil
}

// Get sends the StockCandlesRequestV2, unpacks the StockCandlesResponse, and returns a slice of StockCandle.
// It returns an error if the request or unpacking fails.
// An optional MarketDataClient can be passed to replace the client used in the request.
//
// Parameters:
//   - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// Returns:
//   - []models.StockCandle: A slice of StockCandle containing the unpacked candle data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (scrV2 *StockCandlesRequestV2) Get(optionalClients ...*MarketDataClient) ([]models.Candle, error) {
	if scrV2 == nil {
		return nil, fmt.Errorf("StockCandlesRequestV2 is nil")
	}

	// Use the Packed method to make the request, passing along any optional client
	scrResp, err := scrV2.Packed(optionalClients...)
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
