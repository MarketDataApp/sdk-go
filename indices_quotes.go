// Package client includes types and methods to access the Index Quotes endpoint. Retrieve real-time quotes for any supported index.
//
// # Making Requests
//
// Utilize [IndexQuoteRequest] to make requests to the endpoint through one of the three supported execution methods:
//
//	| Method     | Execution     | Return Type                 | Description                                                                                                |
//	|------------|---------------|-----------------------------|------------------------------------------------------------------------------------------------------------|
//	| **Get**    | Direct        | `[]IndexQuote`              | Directly returns a slice of `[]IndexQuote`, facilitating individual access to each quote.                  |
//	| **Packed** | Intermediate  | `*IndexQuotesResponse`      | Returns a packed `*IndexQuotesResponse` object. Must be unpacked to access the `[]IndexQuote` slice.       |
//	| **Raw**    | Low-level     | `*resty.Response`           | Offers the raw `*resty.Response` for utmost flexibility. Direct access to raw JSON or `*http.Response`.    |
package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
	"github.com/go-resty/resty/v2"
)

// IndexQuoteRequest represents a request to the [/v1/indices/quotes/] endpoint.
// It encapsulates parameters for symbol and fifty-two-week data to be used in the request.
// This struct provides methods such as Symbol() and FiftyTwoWeek() to set these parameters respectively.
//
// # Generated By
//
//   - IndexQuotes(): IndexQuotes creates a new *IndexQuoteRequest and returns a pointer to the request allowing for method chaining.
//
// # Setter Methods
//
// These methods are used to set the parameters of the request. They allow for method chaining
// by returning a pointer to the *IndexQuoteRequest instance they modify.
//
//   - Symbol(string) *IndexQuoteRequest: Sets the symbol parameter for the request.
//   - FiftyTwoWeek(bool) *IndexQuoteRequest: Sets the fifty-two-week parameter for the request.
//
// # Execution Methods
//
// These methods are used to send the request in different formats or retrieve the data.
// They handle the actual communication with the API endpoint.
//
//   - Get) ([]IndexQuote, error): Sends the request, unpacks the response, and returns the data in a user-friendly format.
//   - Packed) (*IndexQuotesResponse, error): Returns a struct that contains equal-length slices of primitives. This packed response mirrors Market Data's JSON response.
//   - Raw) (*resty.Response, error): Sends the request as is and returns the raw HTTP response.
//
// [/v1/indices/quotes/]: https://www.marketdata.app/docs/api/indices/quotes
type IndexQuoteRequest struct {
	*baseRequest
	symbolParams       *parameters.SymbolParams
	fiftyTwoWeekParams *parameters.FiftyTwoWeekParams
}

// Symbol sets the symbol parameter for the IndexQuoteRequest.
// This method is used to specify the market symbol for which the quote data is requested.
// It modifies the symbolParams field of the IndexQuoteRequest instance to store the symbol value.
//
// # Parameters
//
//   - string: A string representing the market symbol to be set.
//
// # Returns
//
//   - *IndexQuoteRequest: This method returns a pointer to the IndexQuoteRequest instance it was called on. This allows for method chaining, where multiple setter methods can be called in a single statement. If the receiver (*IndexQuoteRequest) is nil, it returns nil to prevent a panic.
//
// # Note:
//
//   - If an error occurs while setting the symbol (e.g., if the symbol value is not supported), the Error field of the IndexQuoteRequest is set with the encountered error, but the method still returns the IndexQuoteRequest instance to allow for further method calls or error handling by the caller.
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
// # Parameters
//
//   - bool: A boolean indicating whether to include fifty-two-week data.
//
// # Returns
//
//   - *IndexQuoteRequest: This method returns a pointer to the IndexQuoteRequest instance it was called on. This allows for method chaining. If the receiver (*IndexQuoteRequest) is nil, it returns nil to prevent a panic.
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
// # Returns
//
//   - []parameters.MarketDataParam: A slice containing all the parameters set in the IndexQuoteRequest.
//   - error: An error object indicating failure to pack the parameters, nil if successful.
func (iqr *IndexQuoteRequest) getParams() ([]parameters.MarketDataParam, error) {
	if iqr == nil {
		return nil, fmt.Errorf("IndexQuoteRequest is nil")
	}
	params := []parameters.MarketDataParam{iqr.symbolParams, iqr.fiftyTwoWeekParams}
	return params, nil
}

// Packed sends the IndexQuoteRequest and returns the IndexQuotesResponse.
// This method checks if the IndexQuoteRequest receiver is nil, returning an error if true.
// It proceeds to send the request using the default client and returns the IndexQuotesResponse along with any error encountered during the request.
//
// # Returns
//
//   - *models.IndexQuotesResponse: A pointer to the IndexQuotesResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (iqr *IndexQuoteRequest) Packed() (*models.IndexQuotesResponse, error) {
	if iqr == nil {
		return nil, fmt.Errorf("IndexQuoteRequest is nil")
	}

	var iqrResp models.IndexQuotesResponse
	_, err := iqr.baseRequest.client.getFromRequest(iqr.baseRequest, &iqrResp)
	if err != nil {
		return nil, err
	}

	return &iqrResp, nil
}

// Raw executes the IndexQuoteRequest and returns the raw *resty.Response.
// The *resty.Response can be directly used to access the raw JSON or *http.Response for further processing.
//
// # Returns
//
//   - *resty.Response: The raw HTTP response from the executed IndexQuoteRequest.
//   - error: An error object if the IndexQuoteRequest is nil or if an error occurs during the request execution.
func (iqr *IndexQuoteRequest) Raw() (*resty.Response, error) {
	return iqr.baseRequest.Raw()
}

// Get sends the IndexQuoteRequest, unpacks the IndexQuotesResponse, and returns a slice of IndexQuote.
// It returns an error if the request or unpacking fails. This method is crucial for obtaining the actual quote data
// from the index quote request. The method first checks if the IndexQuoteRequest receiver is nil, which would
// result in an error as the request cannot be sent. It then proceeds to send the request using the default client with the Packed method.
// Upon receiving the response, it unpacks the data into a slice of IndexQuote using the Unpack method from the response.
//
// # Returns
//
//   - []models.IndexQuote: A slice of IndexQuote containing the unpacked quote data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (iqr *IndexQuoteRequest) Get() ([]models.IndexQuote, error) {
	if iqr == nil {
		return nil, fmt.Errorf("IndexQuoteRequest is nil")
	}

	iqrResp, err := iqr.Packed()
	if err != nil {
		return nil, err
	}

	data, err := iqrResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// IndexQuotes creates a new IndexQuoteRequest and associates it with the default client.
// This function initializes the request with default parameters for symbol and fifty-two-week data, and sets the request path based on
// the predefined endpoints for index quotes.
//
// # Returns
//
//   - *IndexQuoteRequest: A pointer to the newly created IndexQuoteRequest with default parameters and associated with the default client.
func IndexQuotes() *IndexQuoteRequest {
	baseReq := newBaseRequest()
	baseReq.path = endpoints[1]["indices"]["quotes"]

	iqr := &IndexQuoteRequest{
		baseRequest:        baseReq,
		symbolParams:       &parameters.SymbolParams{},
		fiftyTwoWeekParams: &parameters.FiftyTwoWeekParams{},
	}

	baseReq.child = iqr

	return iqr
}
