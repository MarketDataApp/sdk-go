package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// StockCandlesRequest represents a request to the [/v1/stocks/candles] endpoint.
// It encapsulates parameters for resolution, symbol, date, and additional stock-specific parameters to be used in the request.
// This struct provides methods such as Resolution(), Symbol(), Date(), From(), To(), Countback(), AdjustSplits(), AdjustDividends(), Extended(), and Exchange() to set these parameters respectively.
//
// # Generated By
//
//   - StockCandles(client ...*MarketDataClient) *StockCandlesRequest: StockCandles creates a new *StockCandlesRequest and returns a pointer to the request allowing for method chaining.
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
//   - Raw() (*resty.Response, error): Sends the request as is and returns the raw HTTP response.
//   - Packed() (*IndicesCandlesResponse, error): Packs the request parameters and sends the request, returning a structured response.
//   - Get() ([]Candle, error): Sends the request, unpacks the response, and returns the data in a user-friendly format.
//[/v1/stocks/candles]: https://www.marketdata.app/docs/api/stocks/candles
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

// Packed sends the StockCandlesRequest and returns the StockCandlesResponse.
// An optional MarketDataClient can be passed to replace the client used in the request.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided, it replaces the current client for this request.
//
// # Returns
//
//   - *models.StockCandlesResponse: A pointer to the StockCandlesResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (scr *StockCandlesRequest) Packed(optionalClients ...*MarketDataClient) (*models.StockCandlesResponse, error) {
	if scr == nil {
		return nil, fmt.Errorf("StockCandlesRequest is nil")
	}

	// Replace the client if an optional client is provided
	if len(optionalClients) > 0 && optionalClients[0] != nil {
		scr.baseRequest.client = optionalClients[0]
	}

	var scrResp models.StockCandlesResponse
	_, err := scr.baseRequest.client.GetFromRequest(scr.baseRequest, &scrResp)
	if err != nil {
		return nil, err
	}

	return &scrResp, nil
}

// Get sends the StockCandlesRequest, unpacks the StockCandlesResponse, and returns a slice of StockCandle.
// It returns an error if the request or unpacking fails.
// An optional MarketDataClient can be passed to replace the client used in the request.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided, it replaces the current client for this request.
//
// # Returns
//
//   - []models.StockCandle: A slice of StockCandle containing the unpacked candle data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (scr *StockCandlesRequest) Get(optionalClients ...*MarketDataClient) ([]models.Candle, error) {
	if scr == nil {
		return nil, fmt.Errorf("StockCandlesRequest is nil")
	}

	// Use the Packed method to make the request, passing along any optional client
	scrResp, err := scr.Packed(optionalClients...)
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
// and configurations to make the request ready to be sent. The function accepts a variadic parameter
// that allows passing an optional MarketDataClient. If a client is provided, it will be used for the request;
// otherwise, a default client is initialized and used.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or more MarketDataClient pointers. This allows for the optional customization of the client used for the request. If no client is provided, a default client is initialized and used.
//
// # Returns
//
//   - *StockCandlesRequest: A pointer to the newly created StockCandlesRequest instance. This instance contains all the necessary parameters set to their default values and is ready to have additional parameters set or to be sent.
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
