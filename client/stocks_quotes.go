package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// StockQuoteRequest represents a request to the /stocks/quote endpoint.
// It encapsulates parameters for symbol and fifty-two-week data to be used in the request.
// This struct provides methods such as Symbol() and FiftyTwoWeek() to set these parameters respectively.
//
// Public Methods:
//   - Symbol(q string) *StockQuoteRequest: Sets the symbol parameter for the request.
//   - FiftyTwoWeek(q bool) *StockQuoteRequest: Sets the fifty-two-week data parameter for the request.
type StockQuoteRequest struct {
	*baseRequest
	symbolParams       *parameters.SymbolParams
	fiftyTwoWeekParams *parameters.FiftyTwoWeekParams
}

// Symbol sets the symbol parameter for the StockQuoteRequest.
// This method is used to specify the stock symbol for which quote data is requested.
//
// Parameters:
//   - q: A string representing the stock symbol to be set.
//
// Returns:
//   - *StockQuoteRequest: This method returns a pointer to the StockQuoteRequest instance it was called on. This allows for method chaining.
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
// Parameters:
//   - q: A bool indicating whether to include fifty-two-week data.
//
// Returns:
//   - *StockQuoteRequest: This method returns a pointer to the StockQuoteRequest instance it was called on. This allows for method chaining.
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

// Packed sends the StockQuoteRequest and returns the StockQuotesResponse.
// An optional MarketDataClient can be passed to replace the client used in the request.
// Parameters:
//   - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// Returns:
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
// Parameters:
//   - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// Returns:
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
// Parameters:
//   - client: A variadic parameter that can accept zero or one MarketDataClient pointer. If no client is provided,
//     the default client is used.
//
// Returns:
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
