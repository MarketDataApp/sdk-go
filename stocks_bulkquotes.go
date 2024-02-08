package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// BulkStockQuotesRequest represents a request to the /stocks/quote endpoint.
// It encapsulates parameters for symbol and fifty-two-week data to be used in the request.
// This struct provides methods such as Symbol() and FiftyTwoWeek() to set these parameters respectively.
//
// Public Methods:
//   - Symbol(q string) *BulkStockQuotesRequest: Sets the symbol parameter for the request.
//   - FiftyTwoWeek(q bool) *BulkStockQuotesRequest: Sets the fifty-two-week data parameter for the request.
type BulkStockQuotesRequest struct {
	*baseRequest
	bulkStockParams   *parameters.BulkStockParams
}

// Symbols sets the symbols parameter for the BulkStockCandlesRequest.
// This method is used to specify multiple stock symbols for which candle data is requested.
//
// Parameters:
//   - q: A slice of strings representing the stock symbols to be set.
//
// Returns:
//   - *BulkStockCandlesRequest: This method returns a pointer to the BulkStockCandlesRequest instance it was called on. This allows for method chaining.
func (bs *BulkStockQuotesRequest) Symbols(q []string) *BulkStockQuotesRequest {
	if bs == nil {
		return nil
	}
	err := bs.bulkStockParams.SetSymbols(q)
	if err != nil {
		bs.Error = err
	}
	return bs
}

// Snapshot sets the snapshot parameter for the BulkStockQuotesRequest.
// This method is used to enable or disable the snapshot feature in the request.
//
// Parameters:
//   - q: A boolean value representing whether to enable or disable the snapshot feature.
//
// Returns:
//   - *BulkStockQuotesRequest: This method returns a pointer to the BulkStockQuotesRequest instance it was called on. This allows for method chaining.
func (bs *BulkStockQuotesRequest) Snapshot(q bool) *BulkStockQuotesRequest {
	if bs == nil {
		return nil
	}
	bs.bulkStockParams.SetSnapshot(q)
	return bs
}

// getParams packs the BulkStockQuotesRequest struct into a slice of interface{} and returns it.
func (bs *BulkStockQuotesRequest) getParams() ([]parameters.MarketDataParam, error) {
	if bs == nil {
		return nil, fmt.Errorf("BulkStockQuotesRequest is nil")
	}
	params := []parameters.MarketDataParam{bs.bulkStockParams}
	return params, nil
}

// Packed sends the BulkStockQuotesRequest and returns the StockQuotesResponse.
// An optional MarketDataClient can be passed to replace the client used in the request.
// Parameters:
//   - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// Returns:
//   - *models.StockQuotesResponse: A pointer to the StockQuotesResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (bs *BulkStockQuotesRequest) Packed(optionalClients ...*MarketDataClient) (*models.StockQuotesResponse, error) {
	if bs == nil {
		return nil, fmt.Errorf("BulkStockQuotesRequest is nil")
	}

	// Replace the client if an optional client is provided
	if len(optionalClients) > 0 && optionalClients[0] != nil {
		bs.baseRequest.client = optionalClients[0]
	}

	var bsResp models.StockQuotesResponse
	_, err := bs.baseRequest.client.GetFromRequest(bs.baseRequest, &bsResp)
	if err != nil {
		return nil, err
	}

	return &bsResp, nil
}

// Get sends the BulkStockQuotesRequest, unpacks the StockQuotesResponse, and returns a slice of StockQuote.
// It returns an error if the request or unpacking fails.
// An optional MarketDataClient can be passed to replace the client used in the request.
// Parameters:
//   - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// Returns:
//   - []models.StockQuote: A slice of StockQuote containing the unpacked quote data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (bs *BulkStockQuotesRequest) Get(optionalClients ...*MarketDataClient) ([]models.StockQuote, error) {
	if bs == nil {
		return nil, fmt.Errorf("BulkStockQuotesRequest is nil")
	}

	// Use the Packed method to make the request, passing along any optional client
	bsResp, err := bs.Packed(optionalClients...)
	if err != nil {
		return nil, err
	}

	// Unpack the data using the Unpack method in the response
	data, err := bsResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// StockQuote creates a new BulkStockQuotesRequest and associates it with the provided client.
// If no client is provided, it uses the default client. This function initializes the request
// with default parameters for symbol and fifty-two-week data, and sets the request path based on
// the predefined endpoints for stock quotes.
//
// Parameters:
//   - client: A variadic parameter that can accept zero or one MarketDataClient pointer. If no client is provided,
//     the default client is used.
//
// Returns:
//   - *BulkStockQuotesRequest: A pointer to the newly created BulkStockQuotesRequest with default parameters and associated client.
func BulkStockQuotes(client ...*MarketDataClient) *BulkStockQuotesRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[1]["stocks"]["bulkquotes"]

	bs := &BulkStockQuotesRequest{
		baseRequest:        baseReq,
		bulkStockParams:       &parameters.BulkStockParams{},
	}

	baseReq.child = bs

	return bs
}
