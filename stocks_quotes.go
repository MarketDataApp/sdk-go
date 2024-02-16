// Package client includes types and methods to access the Stock Quotes endpoint.
// Retrieve real-time quotes for any supported stock symbol.
//
// # Making Requests
//
// Utilize [StockQuoteRequest] to make requests to the endpoint through one of the three supported execution methods:
//
//	| Method     | Execution     | Return Type                | Description                                                                                               |
//	|------------|---------------|----------------------------|-----------------------------------------------------------------------------------------------------------|
//	| **Get**    | Direct        | `[]StockQuote`             | Directly returns a slice of `[]StockQuote`, facilitating individual access to each stock quote.           |
//	| **Packed** | Intermediate  | `StockQuotesResponse`      | Returns a packed `StockQuotesResponse` object. Must be unpacked to access the `[]StockQuote` slice.       |
//	| **Raw**    | Low-level     | `resty.Response`           | Offers the raw `resty.Response` for utmost flexibility. Direct access to raw JSON or `*http.Response`.    |
package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
	"github.com/go-resty/resty/v2"
)

// StockQuoteRequest represents a request to the [/v1/stocks/quotes/] endpoint.
// It encapsulates parameters for symbol and fifty-two-week data to be used in the request.
// This struct provides methods such as Symbol() and FiftyTwoWeek() to set these parameters respectively.
//
// # Setter Methods
//
//   - Symbol(string) *StockQuoteRequest: Sets the symbol parameter for the request.
//   - FiftyTwoWeek(bool) *StockQuoteRequest: Sets the fifty-two-week data parameter for the request.
//
// # Execution Methods
//
// These methods are used to send the request in different formats or retrieve the data.
// They handle the actual communication with the API endpoint.
//
//   - Get(...*MarketDataClient) ([]Candle, error): Sends the request, unpacks the response, and returns the data in a user-friendly format.
//   - Packed(...*MarketDataClient) (*IndicesCandlesResponse, error): Returns a struct that contains equal-length slices of primitives. This packed response mirrors Market Data's JSON response.
//   - Raw(...*MarketDataClient) (*resty.Response, error): Sends the request as is and returns the raw HTTP response.
//
// [/v1/stocks/quotes/]: https://www.marketdata.app/docs/api/stocks/quotes
type StockQuoteRequest struct {
	*baseRequest
	symbolParams       *parameters.SymbolParams
	fiftyTwoWeekParams *parameters.FiftyTwoWeekParams
}

// Symbol sets the symbol parameter for the StockQuoteRequest.
// This method is used to specify the stock symbol for which quote data is requested.
//
// # Parameters
//
//   - string: A string representing the stock symbol to be set.
//
// # Returns
//
//   - *StockQuoteRequest: This method returns a pointer to the *StockQuoteRequest instance it was called on. This allows for method chaining.
func (sqr *StockQuoteRequest) Symbol(q string) *StockQuoteRequest {
	if sqr == nil {
		return nil
	}
	err := sqr.symbolParams.SetSymbol(q)
	if err != nil {
		sqr.Error = err
	}
	return sqr
}

// FiftyTwoWeek sets the fifty-two-week data parameter for the StockQuoteRequest.
// This method indicates whether to include fifty-two-week high and low data in the quote.
//
// # Parameters
//
//   - bool: A bool indicating whether to include fifty-two-week data.
//
// # Returns
//
//   - *StockQuoteRequest: This method returns a pointer to the *StockQuoteRequest instance it was called on. This allows for method chaining.
func (sqr *StockQuoteRequest) FiftyTwoWeek(q bool) *StockQuoteRequest {
	if sqr == nil {
		return nil
	}
	sqr.fiftyTwoWeekParams.SetFiftyTwoWeek(q)
	return sqr
}

// getParams packs the StockQuoteRequest struct into a slice of interface{} and returns it.
func (sqr *StockQuoteRequest) getParams() ([]parameters.MarketDataParam, error) {
	if sqr == nil {
		return nil, fmt.Errorf("StockQuoteRequest is nil")
	}
	params := []parameters.MarketDataParam{sqr.symbolParams, sqr.fiftyTwoWeekParams}
	return params, nil
}

// Raw executes the StockQuoteRequest and returns the raw *resty.Response.
// This method optionally accepts a *MarketDataClient to use for the request, replacing the default client if provided.
// The *resty.Response allows access to the raw JSON or *http.Response for further processing.
//
// # Parameters
//
//   - ...*MarketDataClient: An optional variadic parameter that can accept a *MarketDataClient pointer. If provided, this client is used for the request instead of the default.
//
// # Returns
//
//   - *resty.Response: The raw HTTP response from the executed request.
//   - error: An error object if the request fails due to being nil, the MarketDataClient being nil, or other execution errors.
func (sqr *StockQuoteRequest) Raw(optionalClients ...*MarketDataClient) (*resty.Response, error) {
	return sqr.baseRequest.Raw(optionalClients...)
}

// Packed sends the StockQuoteRequest and returns the StockQuotesResponse.
// An optional MarketDataClient can be passed to replace the client used in the request.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided, it replaces the current client for this request.
//
// # Returns
//
//   - *models.StockQuotesResponse: A pointer to the StockQuotesResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (sqr *StockQuoteRequest) Packed(optionalClients ...*MarketDataClient) (*models.StockQuotesResponse, error) {
	if sqr == nil {
		return nil, fmt.Errorf("StockQuoteRequest is nil")
	}

	// Replace the client if an optional client is provided
	if len(optionalClients) > 0 && optionalClients[0] != nil {
		sqr.baseRequest.client = optionalClients[0]
	}

	var sqrResp models.StockQuotesResponse
	_, err := sqr.baseRequest.client.GetFromRequest(sqr.baseRequest, &sqrResp)
	if err != nil {
		return nil, err
	}

	return &sqrResp, nil
}

// Get sends the StockQuoteRequest, unpacks the StockQuotesResponse, and returns a slice of StockQuote.
// It returns an error if the request or unpacking fails.
// An optional MarketDataClient can be passed to replace the client used in the request.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided, it replaces the current client for this request.
//
// # Returns
//
//   - []models.StockQuote: A slice of StockQuote containing the unpacked quote data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (sqr *StockQuoteRequest) Get(optionalClients ...*MarketDataClient) ([]models.StockQuote, error) {
	if sqr == nil {
		return nil, fmt.Errorf("StockQuoteRequest is nil")
	}

	// Use the Packed method to make the request, passing along any optional client
	sqrResp, err := sqr.Packed(optionalClients...)
	if err != nil {
		return nil, err
	}

	// Unpack the data using the Unpack method in the response
	data, err := sqrResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// StockQuote creates a new StockQuoteRequest and associates it with the provided client.
// If no client is provided, it uses the default client. This function initializes the request
// with default parameters for symbol and fifty-two-week data, and sets the request path based on
// the predefined endpoints for stock quotes.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one MarketDataClient pointer. If no client is provided, the default client is used.
//
// # Returns
//
//   - *StockQuoteRequest: A pointer to the newly created StockQuoteRequest with default parameters and associated client.
func StockQuote(client ...*MarketDataClient) *StockQuoteRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[1]["stocks"]["quotes"]

	sqr := &StockQuoteRequest{
		baseRequest:        baseReq,
		symbolParams:       &parameters.SymbolParams{},
		fiftyTwoWeekParams: &parameters.FiftyTwoWeekParams{},
	}

	baseReq.child = sqr

	return sqr
}
