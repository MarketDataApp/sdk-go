package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// BulkStockCandlesRequest represents a request to the /v1/stocks/candles endpoint.
// It encapsulates parameters for resolution, symbol, date, and additional stock-specific parameters to be used in the request.
// This struct provides methods such as Resolution(), Symbol(), Date(), From(), To(), Countback(), AdjustSplits(), AdjustDividends(), Extended(), and Exchange() to set these parameters respectively.
//
// Public Methods:
//   - Resolution(q string) *BulkStockCandlesRequest: Sets the resolution parameter for the request.
//   - Symbol(q string) *BulkStockCandlesRequest: Sets the symbol parameter for the request.
//   - Date(q interface{}) *BulkStockCandlesRequest: Sets the date parameter for the request.
//   - From(q interface{}) *BulkStockCandlesRequest: Sets the 'from' date parameter for the request.
//   - To(q interface{}) *BulkStockCandlesRequest: Sets the 'to' date parameter for the request.
//   - Countback(q int) *BulkStockCandlesRequest: Sets the countback parameter for the request.
//   - AdjustSplits(q bool) *BulkStockCandlesRequest: Sets the adjust splits parameter for the request.
//   - AdjustDividends(q bool) *BulkStockCandlesRequest: Sets the adjust dividends parameter for the request.
//   - Extended(q bool) *BulkStockCandlesRequest: Sets the extended hours data parameter for the request.
//   - Exchange(q string) *BulkStockCandlesRequest: Sets the exchange parameter for the request.
//   - Packed() (*models.StockCandlesResponse, error): Sends the BulkStockCandlesRequest and returns the StockCandlesResponse.
//   - Get() ([]models.StockCandle, error): Sends the BulkStockCandlesRequest, unpacks the StockCandlesResponse, and returns a slice of StockCandle.
type BulkStockCandlesRequest struct {
	*baseRequest
	stockCandleParams *parameters.StockCandleParams
	bulkStockParams   *parameters.BulkStockParams
	resolutionParams  *parameters.ResolutionParams
	dateParams        *parameters.DateParams
}

// Resolution sets the resolution parameter for the BulkStockCandlesRequest.
// This method is used to specify the granularity of the candle data to be retrieved.
//
// Parameters:
//   - q: A string representing the resolution to be set.
//
// Returns:
//   - *BulkStockCandlesRequest: This method returns a pointer to the BulkStockCandlesRequest instance it was called on. This allows for method chaining.
func (cr *BulkStockCandlesRequest) Resolution(q string) *BulkStockCandlesRequest {
	if cr == nil {
		return nil
	}
	err := cr.resolutionParams.SetResolution(q)
	if err != nil {
		cr.Error = err
	}
	return cr
}

// Symbols sets the symbols parameter for the BulkStockCandlesRequest.
// This method is used to specify multiple stock symbols for which candle data is requested.
//
// Parameters:
//   - q: A slice of strings representing the stock symbols to be set.
//
// Returns:
//   - *BulkStockCandlesRequest: This method returns a pointer to the BulkStockCandlesRequest instance it was called on. This allows for method chaining.
func (cr *BulkStockCandlesRequest) Symbols(q []string) *BulkStockCandlesRequest {
	if cr == nil {
		return nil
	}
	err := cr.bulkStockParams.SetSymbols(q)
	if err != nil {
		cr.Error = err
	}
	return cr
}

// Date sets the date parameter for the BulkStockCandlesRequest.
// This method is used to specify the date for which the stock candle data is requested.
//
// Parameters:
//   - q: An interface{} representing the date to be set.
//
// Returns:
//   - *BulkStockCandlesRequest: This method returns a pointer to the BulkStockCandlesRequest instance it was called on. This allows for method chaining.
func (bscr *BulkStockCandlesRequest) Date(q interface{}) *BulkStockCandlesRequest {
	err := bscr.dateParams.SetDate(q)
	if err != nil {
		bscr.baseRequest.Error = err
	}
	return bscr
}

// AdjustSplits sets the adjust splits parameter for the BulkStockCandlesRequest.
// This method indicates whether the returned data should be adjusted for stock splits.
//
// Parameters:
//   - q: A bool indicating whether to adjust for splits.
//
// Returns:
//   - *BulkStockCandlesRequest: This method returns a pointer to the BulkStockCandlesRequest instance it was called on. This allows for method chaining.
func (bscr *BulkStockCandlesRequest) AdjustSplits(q bool) *BulkStockCandlesRequest {
	if bscr == nil {
		return nil
	}
	bscr.stockCandleParams.SetAdjustSplits(q)
	return bscr
}

// Snapshot sets the snapshot parameter for the BulkStockCandlesRequest.
// This method is used to enable or disable the snapshot feature in the request.
//
// Parameters:
//   - q: A boolean value representing whether to enable or disable the snapshot feature.
//
// Returns:
//   - *BulkStockCandlesRequest: This method returns a pointer to the BulkStockCandlesRequest instance it was called on. This allows for method chaining.
func (bscr *BulkStockCandlesRequest) Snapshot(q bool) *BulkStockCandlesRequest {
	if bscr == nil {
		return nil
	}
	bscr.bulkStockParams.SetSnapshot(q)
	return bscr
}

// getParams packs the CandlesRequest struct into a slice of interface{} and returns it.
func (bscr *BulkStockCandlesRequest) getParams() ([]parameters.MarketDataParam, error) {
	if bscr == nil {
		return nil, fmt.Errorf("BulkStockCandlesRequest is nil")
	}
	params := []parameters.MarketDataParam{bscr.dateParams, bscr.bulkStockParams, bscr.resolutionParams, bscr.stockCandleParams}
	return params, nil
}

// Packed sends the BulkStockCandlesRequest and returns the StockCandlesResponse.
// An optional MarketDataClient can be passed to replace the client used in the request.
// Parameters:
//   - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// Returns:
//   - *models.StockCandlesResponse: A pointer to the StockCandlesResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (bscr *BulkStockCandlesRequest) Packed(optionalClients ...*MarketDataClient) (*models.BulkStockCandlesResponse, error) {
	if bscr == nil {
		return nil, fmt.Errorf("BulkStockCandlesRequest is nil")
	}

	// Replace the client if an optional client is provided
	if len(optionalClients) > 0 && optionalClients[0] != nil {
		bscr.baseRequest.client = optionalClients[0]
	}

	var scrResp models.BulkStockCandlesResponse
	_, err := bscr.baseRequest.client.GetFromRequest(bscr.baseRequest, &scrResp)
	if err != nil {
		return nil, err
	}

	return &scrResp, nil
}


// BulkStockCandles initializes a new BulkStockCandlesRequest with default parameters.
// This function prepares a request to fetch bulk stock candles data. It sets up all necessary parameters
// and configurations to make the request ready to be sent. The function accepts a variadic parameter
// that allows passing an optional MarketDataClient. If a client is provided, it will be used for the request;
// otherwise, a default client is initialized and used.
//
// Parameters:
//   - client: A variadic parameter that can accept zero or one MarketDataClient pointer. This allows
//             for the optional customization of the client used for the request. If no client is provided,
//             a default client is initialized and used.
//
// Returns:
//   - *BulkStockCandlesRequest: A pointer to the newly created BulkStockCandlesRequest instance. This instance
//                                contains all the necessary parameters set to their default values and is ready
//                                to have additional parameters set or to be sent.
func BulkStockCandles(client ...*MarketDataClient) *BulkStockCandlesRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[1]["stocks"]["bulkcandles"]

	bscr := &BulkStockCandlesRequest{
		baseRequest:       baseReq,
		dateParams:        &parameters.DateParams{},
		resolutionParams:  &parameters.ResolutionParams{},
		bulkStockParams:   &parameters.BulkStockParams{},
		stockCandleParams: &parameters.StockCandleParams{},
	}

	// Set the date to the current time
	baseReq.child = bscr

	return bscr
}

// Get sends the BulkStockCandlesRequest, unpacks the StockCandlesResponse, and returns a slice of StockCandle.
// It returns an error if the request or unpacking fails.
// An optional MarketDataClient can be passed to replace the client used in the request.
// Parameters:
//   - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// Returns:
//   - []models.StockCandle: A slice of StockCandle containing the unpacked candle data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (bscr *BulkStockCandlesRequest) Get(optionalClients ...*MarketDataClient) ([]models.Candle, error) {
	if bscr == nil {
		return nil, fmt.Errorf("BulkStockCandlesRequest is nil")
	}

	// Use the Packed method to make the request, passing along any optional client
	scrResp, err := bscr.Packed(optionalClients...)
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
