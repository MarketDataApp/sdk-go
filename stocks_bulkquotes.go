// Package client includes types and methods to access the Bulk Stock Quotes endpoint.
// Retrieve live quotes for multiple stocks symbols in a single request or even get a full market snapshot with a quote for all symbols.
//
// # Making Requests
//
// Utilize [BulkStockQuotesRequest] to make requests to the endpoint through one of the three supported execution methods:
//
//	| Method     | Execution     | Return Type                 | Description                                                                                                |
//	|------------|---------------|-----------------------------|------------------------------------------------------------------------------------------------------------|
//	| **Get**    | Direct        | `[]StockQuote`              | Directly returns a slice of `[]StockQuote`, facilitating individual access to each stock quote.            |
//	| **Packed** | Intermediate  | `*StockQuotesResponse`      | Returns a packed `*StockQuotesResponse` object. Must be unpacked to access the `[]StockQuote` slice.       |
//	| **Raw**    | Low-level     | `*resty.Response`           | Offers the raw `*resty.Response` for utmost flexibility. Direct access to raw JSON or `*http.Response`.    |
package client

import (
	"context"
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
	"github.com/go-resty/resty/v2"
)

// BulkStockQuotesRequest represents a request to the [v1/stocks/bulkquotes/] endpoint.
// It encapsulates parameters for symbol and fifty-two-week data to be used in the request.
// This struct provides methods such as Symbol() and FiftyTwoWeek() to set these parameters respectively.
//
// # Generated By
//
//   - BulkStockQuotes() *BulkStockQuotesRequest: BulkStockQuotes creates a new *BulkStockQuotesRequest and returns a pointer to the request allowing for method chaining.
//
// # Setter Methods
//
//   - Symbol(string) *BulkStockQuotesRequest: Sets the symbol parameter for the request.
//   - FiftyTwoWeek(bool) *BulkStockQuotesRequest: Sets the fifty-two-week data parameter for the request.
//
// # Execution Methods
//
// These methods are used to send the request in different formats or retrieve the data.
// They handle the actual communication with the API endpoint.
//
//   - Get() ([]Candle, error): Sends the request, unpacks the response, and returns the data in a user-friendly format.
//   - Packed() (*StockQuotesResponse, error): Returns a struct that contains equal-length slices of primitives. This packed response mirrors Market Data's JSON response.
//   - Raw() (*resty.Response, error): Sends the request as is and returns the raw HTTP response.
//
// [v1/stocks/bulkquotes/]: https://www.marketdata.app/docs/api/stocks/bulkquotes
type BulkStockQuotesRequest struct {
	*baseRequest
	bulkStockParams *parameters.BulkStockParams
}

// Symbols sets the symbols parameter for the BulkStockQuotesRequest.
// This method is used to specify multiple stock symbols for which candle data is requested.
//
// # Parameters
//
//   - []string: A slice of []string representing the stock symbols to be set.
//
// # Returns
//
//   - *BulkStockQuotesRequest: This method returns a pointer to the BulkStockQuotesRequest instance it was called on. This allows for method chaining.
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

// Snapshot sets the snapshot parameter for the BulkStockQuotesRequest. This method is used to enable
// or disable the snapshot feature in the request and will result in all available tickers being returned in the response.
//
// # Parameters
//
//   - bool: A boolean value representing whether to enable or disable the snapshot feature.
//
// # Returns
//
//   - *BulkStockQuotesRequest: This method returns a pointer to the *BulkStockQuotesRequest instance it was called on. This allows for method chaining.
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

// Raw executes the request for BulkStockQuotesRequest with the provided context and returns the raw *resty.Response.
// This method does not allow for an optional MarketDataClient to be passed.
//
// # Parameters
//
//   - ctx context.Context: The context to use for the request execution.
//
// # Returns
//
//   - *resty.Response: The raw response from the executed BulkStockQuotesRequest.
//   - error: An error object if the BulkStockQuotesRequest is nil or if an error occurs during the request execution.
func (bsqr *BulkStockQuotesRequest) Raw(ctx context.Context) (*resty.Response, error) {
	return bsqr.baseRequest.Raw(ctx)
}

// Packed sends the BulkStockQuotesRequest with the provided context and returns the StockQuotesResponse.
//
// # Parameters
//
//   - ctx context.Context: The context to use for the request execution.
//
// # Returns
//
//   - *models.StockQuotesResponse: A pointer to the StockQuotesResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (bs *BulkStockQuotesRequest) Packed(ctx context.Context) (*models.StockQuotesResponse, error) {
	if bs == nil {
		return nil, fmt.Errorf("BulkStockQuotesRequest is nil")
	}

	var bsResp models.StockQuotesResponse
	_, err := bs.baseRequest.client.getFromRequest(ctx, bs.baseRequest, &bsResp)
	if err != nil {
		return nil, err
	}

	return &bsResp, nil
}

// Get sends the BulkStockQuotesRequest with the provided context, unpacks the StockQuotesResponse, and returns a slice of StockQuote.
// It returns an error if the request or unpacking fails.
//
// # Parameters
//
//   - ctx context.Context: The context to use for the request execution.
//
// # Returns
//
//   - []models.StockQuote: A slice of StockQuote containing the unpacked quote data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (bs *BulkStockQuotesRequest) Get(ctx context.Context) ([]models.StockQuote, error) {
	if bs == nil {
		return nil, fmt.Errorf("BulkStockQuotesRequest is nil")
	}

	bsResp, err := bs.Packed(ctx)
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

// BulkStockQuotes creates a new BulkStockQuotesRequest and associates it with the default client. This function initializes the request
// with default parameters for symbol and fifty-two-week data, and sets the request path based on
// the predefined endpoints for stock quotes.
//
// # Returns
//
//   - *BulkStockQuotesRequest: A pointer to the newly created BulkStockQuotesRequest with default parameters and associated client.
func BulkStockQuotes() *BulkStockQuotesRequest {
	baseReq := newBaseRequest()
	baseReq.path = endpoints[1]["stocks"]["bulkquotes"]

	bs := &BulkStockQuotesRequest{
		baseRequest:     baseReq,
		bulkStockParams: &parameters.BulkStockParams{},
	}

	baseReq.child = bs

	return bs
}
