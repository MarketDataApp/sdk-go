package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// IndexQuoteRequest represents a request to the /indices/quote endpoint.
// It encapsulates parameters for symbol and fifty-two-week data to be used in the request.
// This struct provides methods such as Symbol() and FiftyTwoWeek() to set these parameters respectively.
//
// Public Methods:
// - Symbol(q string) *IndexQuoteRequest: Sets the symbol parameter for the request.
// - FiftyTwoWeek(q bool) *IndexQuoteRequest: Sets the fifty-two-week parameter for the request.
type IndexQuoteRequest struct {
	*baseRequest
	symbolParams       *parameters.SymbolParams
	fiftyTwoWeekParams *parameters.FiftyTwoWeekParams
}

// Symbol sets the symbol parameter for the IndexQuoteRequest.
// This method is used to specify the market symbol for which the quote data is requested.
// It modifies the symbolParams field of the IndexQuoteRequest instance to store the symbol value.
//
// Parameters:
// - q: A string representing the market symbol to be set.
//
// Returns:
// - *IndexQuoteRequest: This method returns a pointer to the IndexQuoteRequest instance it was called on. This allows for method chaining, where multiple setter methods can be called in a single statement. If the receiver (*IndexQuoteRequest) is nil, it returns nil to prevent a panic.
//
// Note:
// If an error occurs while setting the symbol (e.g., if the symbol value is not supported), the Error field of the IndexQuoteRequest is set with the encountered error, but the method still returns the IndexQuoteRequest instance to allow for further method calls or error handling by the caller.
func (iqr *IndexQuoteRequest) Symbol(q string) *IndexQuoteRequest {
	if iqr == nil {
		return nil
	}
	err := iqr.symbolParams.SetSymbol(q)
	if err != nil {
		iqr.Error = err
	}
	return iqr
}

// FiftyTwoWeek sets the FiftyTwoWeek parameter for the IndexQuoteRequest.
// This method is used to specify whether to include fifty-two-week high and low data in the quote.
// It modifies the fiftyTwoWeekParams field of the IndexQuoteRequest instance to store the boolean value.
//
// Parameters:
// - q: A boolean indicating whether to include fifty-two-week data.
//
// Returns:
// - *IndexQuoteRequest: This method returns a pointer to the IndexQuoteRequest instance it was called on. This allows for method chaining. If the receiver (*IndexQuoteRequest) is nil, it returns nil to prevent a panic.
func (iqr *IndexQuoteRequest) FiftyTwoWeek(q bool) *IndexQuoteRequest {
	if iqr == nil {
		return nil
	}
	iqr.fiftyTwoWeekParams.SetFiftyTwoWeek(q)
	return iqr
}

// getParams packs the IndexQuoteRequest struct into a slice of interface{} and returns it.
// This method is used to gather all the parameters set in the IndexQuoteRequest into a single slice
// for easier manipulation and usage in subsequent requests.
//
// Returns:
// - []parameters.MarketDataParam: A slice containing all the parameters set in the IndexQuoteRequest.
// - error: An error object indicating failure to pack the parameters, nil if successful.
func (iqr *IndexQuoteRequest) getParams() ([]parameters.MarketDataParam, error) {
	if iqr == nil {
		return nil, fmt.Errorf("IndexQuoteRequest is nil")
	}
	params := []parameters.MarketDataParam{iqr.symbolParams, iqr.fiftyTwoWeekParams}
	return params, nil
}

// Packed sends the IndexQuoteRequest and returns the IndexQuotesResponse.
// This method checks if the IndexQuoteRequest receiver is nil, returning an error if true.
// An optional MarketDataClient can be passed to replace the client used in the request.
// Otherwise, it proceeds to send the request and returns the IndexQuotesResponse along with any error encountered during the request.
// Parameters:
// - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//   it replaces the current client for this request.
// Returns:
// - *models.IndexQuotesResponse: A pointer to the IndexQuotesResponse obtained from the request.
// - error: An error object that indicates a failure in sending the request.
func (iqr *IndexQuoteRequest) Packed(optionalClients ...*MarketDataClient) (*models.IndexQuotesResponse, error) {
	if iqr == nil {
		return nil, fmt.Errorf("IndexQuoteRequest is nil")
	}

	// Replace the client if an optional client is provided
	if len(optionalClients) > 0 && optionalClients[0] != nil {
		iqr.baseRequest.client = optionalClients[0]
	}

	var iqrResp models.IndexQuotesResponse
	_, err := iqr.baseRequest.client.GetFromRequest(iqr.baseRequest, &iqrResp)
	if err != nil {
		return nil, err
	}

	return &iqrResp, nil
}

// Get sends the IndexQuoteRequest, unpacks the IndexQuotesResponse, and returns a slice of IndexQuote.
// It returns an error if the request or unpacking fails. This method is crucial for obtaining the actual quote data
// from the index quote request. The method first checks if the IndexQuoteRequest receiver is nil, which would
// result in an error as the request cannot be sent. It then proceeds to send the request using the Packed method.
// Upon receiving the response, it unpacks the data into a slice of IndexQuote using the Unpack method from the response.
// An optional MarketDataClient can be passed to replace the client used in the request.
// Parameters:
// - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//   it replaces the current client for this request.
// Returns:
// - []models.IndexQuote: A slice of IndexQuote containing the unpacked quote data from the response.
// - error: An error object that indicates a failure in sending the request or unpacking the response.
func (iqr *IndexQuoteRequest) Get(optionalClients ...*MarketDataClient) ([]models.IndexQuote, error) {
	if iqr == nil {
		return nil, fmt.Errorf("IndexQuoteRequest is nil")
	}
	
	// Use the Packed method to make the request, passing along any optional client
	iqrResp, err := iqr.Packed(optionalClients...)
	if err != nil {
		return nil, err
	}

	// Unpack the data using the Unpack method in the response
	data, err := iqrResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// IndexQuotes creates a new IndexQuoteRequest and associates it with the provided client.
// If no client is provided, it uses the default client. This function initializes the request
// with default parameters for symbol and fifty-two-week data, and sets the request path based on
// the predefined endpoints for index quotes.
// Parameters:
// - client: A variadic parameter that can accept zero or one MarketDataClient pointer. If no client is provided,
//   the default client is used.
// Returns:
// - *IndexQuoteRequest: A pointer to the newly created IndexQuoteRequest with default parameters and associated client.
func IndexQuotes(client ...*MarketDataClient) *IndexQuoteRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[1]["indices"]["quotes"]

	iqr := &IndexQuoteRequest{
		baseRequest:        baseReq,
		symbolParams:       &parameters.SymbolParams{},
		fiftyTwoWeekParams: &parameters.FiftyTwoWeekParams{},
	}

	baseReq.child = iqr

	return iqr
}
