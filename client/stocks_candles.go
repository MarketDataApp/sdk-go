package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// StockCandlesRequest represents a request to the /v1/stocks/candles endpoint.
// It encapsulates parameters for resolution, symbol, date, and additional stock-specific parameters to be used in the request.
// This struct provides methods such as Resolution(), Symbol(), Date(), From(), To(), Countback(), AdjustSplits(), AdjustDividends(), Extended(), and Exchange() to set these parameters respectively.
//
// Public Methods:
// - Resolution(q string) *StockCandlesRequest: Sets the resolution parameter for the request.
// - Symbol(q string) *StockCandlesRequest: Sets the symbol parameter for the request.
// - Date(q interface{}) *StockCandlesRequest: Sets the date parameter for the request.
// - From(q interface{}) *StockCandlesRequest: Sets the 'from' date parameter for the request.
// - To(q interface{}) *StockCandlesRequest: Sets the 'to' date parameter for the request.
// - Countback(q int) *StockCandlesRequest: Sets the countback parameter for the request.
// - AdjustSplits(q bool) *StockCandlesRequest: Sets the adjust splits parameter for the request.
// - AdjustDividends(q bool) *StockCandlesRequest: Sets the adjust dividends parameter for the request.
// - Extended(q bool) *StockCandlesRequest: Sets the extended hours data parameter for the request.
// - Exchange(q string) *StockCandlesRequest: Sets the exchange parameter for the request.
// - Packed() (*models.StockCandlesResponse, error): Sends the StockCandlesRequest and returns the StockCandlesResponse.
// - Get() ([]models.StockCandle, error): Sends the StockCandlesRequest, unpacks the StockCandlesResponse, and returns a slice of StockCandle.
type StockCandlesRequest struct {
	*baseRequest
	stockCandleParams *parameters.StockCandleParams
	resolutionParams  *parameters.ResolutionParams
	symbolParams      *parameters.SymbolParams
	dateParams        *parameters.DateParams
}

// Resolution sets the resolution parameter for the StockCandlesRequest.
// This method is used to specify the granularity of the candle data to be retrieved.
//
// Parameters:
// - q: A string representing the resolution to be set.
//
// Returns:
// - *StockCandlesRequest: This method returns a pointer to the StockCandlesRequest instance it was called on. This allows for method chaining.
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

// Symbol sets the symbol parameter for the StockCandlesRequest.
// This method is used to specify the stock symbol for which candle data is requested.
//
// Parameters:
// - q: A string representing the stock symbol to be set.
//
// Returns:
// - *StockCandlesRequest: This method returns a pointer to the StockCandlesRequest instance it was called on. This allows for method chaining.
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

// Date sets the date parameter for the StockCandlesRequest.
// This method is used to specify the date for which the stock candle data is requested.
//
// Parameters:
// - q: An interface{} representing the date to be set.
//
// Returns:
// - *StockCandlesRequest: This method returns a pointer to the StockCandlesRequest instance it was called on. This allows for method chaining.
func (scr *StockCandlesRequest) Date(q interface{}) *StockCandlesRequest {
	err := scr.dateParams.SetDate(q)
	if err != nil {
		scr.baseRequest.Error = err
	}
	return scr
}

// From sets the 'from' date parameter for the StockCandlesRequest.
// This method is used to specify the starting point of the date range for which the stock candle data is requested.
//
// Parameters:
// - q: An interface{} representing the starting date.
//
// Returns:
// - *StockCandlesRequest: This method returns a pointer to the StockCandlesRequest instance it was called on. This allows for method chaining.
func (scr *StockCandlesRequest) From(q interface{}) *StockCandlesRequest {
	err := scr.dateParams.SetFrom(q)
	if err != nil {
		scr.baseRequest.Error = err
	}
	return scr
}

// To sets the 'to' date parameter for the StockCandlesRequest.
// This method is used to specify the ending point of the date range for which the stock candle data is requested.
//
// Parameters:
// - q: An interface{} representing the ending date.
//
// Returns:
// - *StockCandlesRequest: This method returns a pointer to the StockCandlesRequest instance it was called on
func (scr *StockCandlesRequest) To(q interface{}) *StockCandlesRequest {
	err := scr.dateParams.SetTo(q)
	if err != nil {
		scr.baseRequest.Error = err
	}
	return scr
}

// Countback sets the countback parameter for the StockCandlesRequest.
// This method specifies the number of candles to return, counting backwards from the 'to' date.
//
// Parameters:
// - q: An int representing the number of candles to return.
//
// Returns:
// - *StockCandlesRequest: This method returns a pointer to the StockCandlesRequest instance it was called on. This allows for method chaining.
func (scr *StockCandlesRequest) Countback(q int) *StockCandlesRequest {
	err := scr.dateParams.SetCountback(q)
	if err != nil {
		scr.baseRequest.Error = err
	}
	return scr
}

// AdjustSplits sets the adjust splits parameter for the StockCandlesRequest.
// This method indicates whether the returned data should be adjusted for stock splits.
//
// Parameters:
// - q: A bool indicating whether to adjust for splits.
//
// Returns:
// - *StockCandlesRequest: This method returns a pointer to the StockCandlesRequest instance it was called on. This allows for method chaining.
func (scr *StockCandlesRequest) AdjustSplits(q bool) *StockCandlesRequest {
	if scr == nil {
		return nil
	}
	scr.stockCandleParams.SetAdjustSplits(q)
	return scr
}

// AdjustDividends sets the adjust dividends parameter for the StockCandlesRequest.
// This method indicates whether the returned data should be adjusted for dividends.
//
// Parameters:
// - q: A bool indicating whether to adjust for dividends.
//
// Returns:
// - *StockCandlesRequest: This method returns a pointer to the StockCandlesRequest instance it was called on. This allows for method chaining.
func (scr *StockCandlesRequest) AdjustDividends(q bool) *StockCandlesRequest {
	if scr == nil {
		return nil
	}
	scr.stockCandleParams.SetAdjustDividends(q)
	return scr
}

// Extended sets the extended hours data parameter for the StockCandlesRequest.
// This method indicates whether the returned data should include extended hours trading data.
//
// Parameters:
// - q: A bool indicating whether to include extended hours data.
//
// Returns:
// - *StockCandlesRequest: This method returns a pointer to the StockCandlesRequest instance it was called on. This allows for method chaining.
func (scr *StockCandlesRequest) Extended(q bool) *StockCandlesRequest {
	if scr == nil {
		return nil
	}
	scr.stockCandleParams.SetExtended(q)
	return scr
}

// Exchange sets the exchange parameter for the StockCandlesRequest.
// This method is used to specify the exchange from which the stock candle data is requested.
//
// Parameters:
// - q: A string representing the exchange to be set.
//
// Returns:
// - *StockCandlesRequest: This method returns a pointer to the StockCandlesRequest instance it was called on. This allows for method chaining.
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

// getParams packs the CandlesRequest struct into a slice of interface{} and returns it.
func (scr *StockCandlesRequest) getParams() ([]parameters.MarketDataParam, error) {
	if scr == nil {
		return nil, fmt.Errorf("StockCandlesRequest is nil")
	}
	params := []parameters.MarketDataParam{scr.dateParams, scr.symbolParams, scr.resolutionParams, scr.stockCandleParams}
	return params, nil
}

// Packed sends the StockCandlesRequest and returns the StockCandlesResponse.
// This method checks if the StockCandlesRequest receiver is nil, returning an error if true.
// Otherwise, it proceeds to send the request and returns the StockCandlesResponse along with any error encountered during the request.
//
// Returns:
// - *models.StockCandlesResponse: A pointer to the StockCandlesResponse obtained from the request.
// - error: An error object that indicates a failure in sending the request.
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

// Get sends the StockCandlesRequest, unpacks the StockCandlesResponse, and returns a slice of StockCandle.
// It returns an error if the request or unpacking fails. This method is crucial for obtaining the actual stock candle data
// from the stock candles request. The method first checks if the StockCandlesRequest receiver is nil, which would
// result in an error as the request cannot be sent. It then proceeds to send the request using the Packed method.
// Upon receiving the response, it unpacks the data into a slice of StockCandle using the Unpack method from the response.
//
// Returns:
// - []models.StockCandle: A slice of StockCandle containing the unpacked candle data from the response.
// - error: An error object that indicates a failure in sending the request or unpacking the response.
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
