// Package client includes types and methods to access the Stock Candles endpoint. Retrieve historical price candles for any supported stock symbol.
//
// # Making Requests
//
// Use [StockCandlesRequest] to make requests to the endpoint using any of the three supported execution methods:
//
//	| Method     | Execution     | Return Type                 | Description                                                                                                |
//	|------------|---------------|-----------------------------|------------------------------------------------------------------------------------------------------------|
//	| **Get**    | Direct        | `[]Candle`                  | Directly returns a slice of `[]Candle`, facilitating individual access to each candle.                     |
//	| **Packed** | Intermediate  | `*StockCandlesResponse`     | Returns a packed `*StockCandlesResponse` object. Must be unpacked to access the `[]Candle` slice.          |
//	| **Raw**    | Low-level     | `*resty.Response`           | Provides the raw `*resty.Response` for maximum flexibility. Direct access to raw JSON or `*http.Response`. |
package client

import (
	"context"
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
	"github.com/go-resty/resty/v2"
)

// StockCandlesRequest represents a request to the [/v1/stocks/candles/] endpoint.
// It encapsulates parameters for resolution, symbol, date, and additional stock-specific parameters to be used in the request.
// This struct provides methods such as Resolution(), Symbol(), Date(), From(), To(), Countback(), AdjustSplits(), AdjustDividends(), Extended(), and Exchange() to set these parameters respectively.
//
// # Generated By
//
//   - StockCandles() *StockCandlesRequest: StockCandles creates a new *StockCandlesRequest and returns a pointer to the request allowing for method chaining.
//
// # Setter Methods
//
//   - Resolution(string) *StockCandlesRequest: Sets the resolution parameter for the request.
//   - Symbol(string) *StockCandlesRequest: Sets the symbol parameter for the request.
//   - Date(interface{}) *StockCandlesRequest: Sets the date parameter for the request.
//   - From(interface{}) *StockCandlesRequest: Sets the 'from' date parameter for the request.
//   - To(interface{}) *StockCandlesRequest: Sets the 'to' date parameter for the request.
//   - Countback(int) *StockCandlesRequest: Sets the countback parameter for the request.
//   - AdjustSplits(bool) *StockCandlesRequest: Sets the adjust splits parameter for the request.
//   - AdjustDividends(bool) *StockCandlesRequest: Sets the adjust dividends parameter for the request.
//   - Extended(bool) *StockCandlesRequest: Sets the extended hours data parameter for the request.
//   - Exchange(string) *StockCandlesRequest: Sets the exchange parameter for the request.
//
// # Execution Methods
//
// These methods are used to send the request in different formats or retrieve the data.
// They handle the actual communication with the API endpoint.
//
//   - Get() ([]Candle, error): Sends the request, unpacks the response, and returns the data in a user-friendly format.
//   - Packed() (*StockCandlesResponse, error): Returns a struct that contains equal-length slices of primitives. This packed response mirrors Market Data's JSON response.
//   - Raw() (*resty.Response, error): Sends the request as is and returns the raw HTTP response.
//
// [/v1/stocks/candles/]: https://www.marketdata.app/docs/api/stocks/candles
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
// # Parameters
//
//   - string: A string representing the resolution to be set.
//
// # Returns
//
//   - *StockCandlesRequest: This method returns a pointer to the *StockCandlesRequest instance it was called on. This allows for method chaining.
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
// # Parameters
//
//   - string: A string representing the stock symbol to be set.
//
// # Returns
//
//   - *StockCandlesRequest: This method returns a pointer to the *StockCandlesRequest instance it was called on. This allows for method chaining.
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
// # Parameters
//
//   - interface{}: An interface{} representing the date to be set. It can be a string, a time.Time object, a Unix timestamp as an int, or any other type that the underlying dates package method can process.
//
// # Returns
//
//   - *StockCandlesRequest: This method returns a pointer to the *StockCandlesRequest instance it was called on. This allows for method chaining.
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
// # Parameters
//
//   - interface{}: An interface{} representing the date to be set. It can be a string, a time.Time object, a Unix timestamp as an int, or any other type that the underlying dates package method can process.
//
// # Returns
//
//   - *StockCandlesRequest: This method returns a pointer to the *StockCandlesRequest instance it was called on. This allows for method chaining.
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
// # Parameters
//
//   - interface{}: An interface{} representing the date to be set. It can be a string, a time.Time object, a Unix timestamp as an int, or any other type that the underlying dates package method can process.
//
// # Returns
//
//   - *StockCandlesRequest: This method returns a pointer to the *StockCandlesRequest instance it was called on
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
// # Parameters
//
//   - int: The number of candles to return.
//
// # Returns
//
//   - *StockCandlesRequest: This method returns a pointer to the *StockCandlesRequest instance it was called on. This allows for method chaining.
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
// # Parameters
//
//   - bool: Whether to adjust for splits.
//
// # Returns
//
//   - *StockCandlesRequest: This method returns a pointer to the *StockCandlesRequest instance it was called on. This allows for method chaining.
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
// # Parameters
//
//   - bool: Whether to adjust for dividends.
//
// # Returns
//
//   - *StockCandlesRequest: This method returns a pointer to the *StockCandlesRequest instance it was called on. This allows for method chaining.
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
// # Parameters
//
//   - bool: Whether to include extended hours data.
//
// # Returns
//
//   - *StockCandlesRequest: This method returns a pointer to the *StockCandlesRequest instance it was called on. This allows for method chaining.
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
// # Parameters
//
//   - string: The exchange to be set.
//
// # Returns
//
//   - *StockCandlesRequest: This method returns a pointer to the *StockCandlesRequest instance it was called on. This allows for method chaining.
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

// Raw executes the StockCandlesRequest with the provided context and returns the raw *resty.Response.
// This method returns the raw JSON or *http.Response for further processing without accepting an alternative MarketDataClient.
//
// # Parameters
//
//   - ctx context.Context: The context to use for the request execution.
//
// # Returns
//
//   - *resty.Response: The raw HTTP response from the executed request.
//   - error: An error object if the request fails due to execution errors.
func (scr *StockCandlesRequest) Raw(ctx context.Context) (*resty.Response, error) {
	return scr.baseRequest.Raw(ctx)
}

// Packed sends the StockCandlesRequest with the provided context and returns the StockCandlesResponse.
//
// # Parameters
//
//   - ctx context.Context: The context to use for the request execution.
//
// # Returns
//
//   - *StockCandlesResponse: A pointer to the StockCandlesResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (scr *StockCandlesRequest) Packed(ctx context.Context) (*models.StockCandlesResponse, error) {
	if scr == nil {
		return nil, fmt.Errorf("StockCandlesRequest is nil")
	}

	var scrResp models.StockCandlesResponse
	_, err := scr.baseRequest.client.getFromRequest(ctx, scr.baseRequest, &scrResp)
	if err != nil {
		return nil, err
	}

	return &scrResp, nil
}

// Get sends the StockCandlesRequest with the provided context, unpacks the StockCandlesResponse, and returns a slice of StockCandle.
// It returns an error if the request or unpacking fails.
//
// # Parameters
//
//   - ctx context.Context: The context to use for the request execution.
//
// # Returns
//
//   - []Candle: A slice of []Candle containing the unpacked candle data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (scr *StockCandlesRequest) Get(ctx context.Context) ([]models.Candle, error) {
	if scr == nil {
		return nil, fmt.Errorf("StockCandlesRequest is nil")
	}

	// Use the Packed method to make the request
	scrResp, err := scr.Packed(ctx)
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

// StockCandles initializes a new StockCandlesRequest with default parameters.
// This function prepares a request to fetch stock candle data. It sets up all necessary parameters
// and configurations to make the request ready to be sent.
//
// # Returns
//
//   - *StockCandlesRequest: A pointer to the newly created StockCandlesRequest instance. This instance contains all the necessary parameters set to their default values and is ready to have additional parameters set or to be sent.
func StockCandles() *StockCandlesRequest {
	baseReq := newBaseRequest()
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
