package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// OptionQuotesRequest represents a request for retrieving options quotes.
// It encapsulates parameters for symbol, expiration, and strike to be used in the request.
//
// Public Methods:
//   - OptionSymbol(symbol string) *OptionQuoteRequest: Sets the symbol parameter for the request.
//   - Date(date interface{}) *OptionQuoteRequest: Sets the date parameter for the request, accepting flexible date input.
//   - From(from interface{}) *OptionQuoteRequest: Sets the start date of a date range for the request, accepting flexible date input.
//   - To(from interface{}) *OptionQuoteRequest: Sets the start date of a date range for the request, accepting flexible date input.
type OptionQuoteRequest struct {
	*baseRequest
	symbolParams *parameters.SymbolParams
	dateParams   *parameters.DateParams
}

// OptionSymbol sets the symbol parameter for the OptionQuotesRequest.
// This method is used to specify the symbol of the option for which the quote is requested.
// Parameters:
//   - symbol: A string representing the symbol to be set.
//
// Returns:
//   - *OptionQuotesRequest: This method returns a pointer to the OptionQuotesRequest instance it was called on, allowing for method chaining.
func (oqr *OptionQuoteRequest) OptionSymbol(q string) *OptionQuoteRequest {
	if oqr == nil {
		return nil
	}
	err := oqr.symbolParams.SetSymbol(q)
	if err != nil {
		oqr.Error = err
	}
	return oqr
}

// Date sets the date parameter for the OptionQuotesRequest.
// This method is used to specify the date for which the option quote is requested.
// It allows for flexibility in the type of date input (e.g., string, time.Time) through the use of an interface{} parameter.
// Parameters:
//   - date: An interface{} representing the date to be set. The actual type accepted can vary (e.g., string, time.Time), depending on implementation.
//
// Returns:
//   - *OptionQuotesRequest: This method returns a pointer to the OptionQuotesRequest instance it was called on, allowing for method chaining. If the receiver (*OptionQuotesRequest) is nil, it returns nil to prevent a panic.
func (oqr *OptionQuoteRequest) Date(q interface{}) *OptionQuoteRequest {
	if oqr.dateParams == nil {
		oqr.dateParams = &parameters.DateParams{}
	}
	err := oqr.dateParams.SetDate(q)
	if err != nil {
		oqr.Error = err
	}
	return oqr
}

// From sets the from parameter for the OptionQuotesRequest.
// This method is used to specify the start date of a date range for which the option quote is requested.
// Similar to the Date method, it accepts a flexible date input through an interface{} parameter.
// Parameters:
//   - from: An interface{} representing the start date of the range to be set. The actual type accepted can vary, depending on implementation.
//
// Returns:
//   - *OptionQuotesRequest: This method returns a pointer to the OptionQuotesRequest instance it was called on, allowing for method chaining. If the receiver (*OptionQuotesRequest) is nil, it returns nil to prevent a panic.
func (oqr *OptionQuoteRequest) From(q interface{}) *OptionQuoteRequest {
	if oqr.dateParams == nil {
		oqr.dateParams = &parameters.DateParams{}
	}
	err := oqr.dateParams.SetFrom(q)
	if err != nil {
		oqr.Error = err
	}
	return oqr
}

// To sets the to parameter for the OptionQuotesRequest.
// This method is used to specify the end date of a date range for which the option quote is requested.
// It accepts a flexible date input through an interface{} parameter, similar to the From method.
// Parameters:
//   - to: An interface{} representing the end date of the range to be set. The actual type accepted can vary, depending on implementation.
//
// Returns:
//   - *OptionQuotesRequest: This method returns a pointer to the OptionQuotesRequest instance it was called on, allowing for method chaining. If the receiver (*OptionQuotesRequest) is nil, it returns nil to prevent a panic.
func (oqr *OptionQuoteRequest) To(q interface{}) *OptionQuoteRequest {
	if oqr.dateParams == nil {
		oqr.dateParams = &parameters.DateParams{}
	}
	err := oqr.dateParams.SetTo(q)
	if err != nil {
		oqr.Error = err
	}
	return oqr
}

// getParams packs the OptionQuotesRequest struct into a slice of interface{} and returns it.
// This method is used to gather all the parameters set in the OptionQuotesRequest into a single slice for easier manipulation and usage in subsequent requests.
// Returns:
//   - []parameters.MarketDataParam: A slice containing all the parameters set in the OptionQuotesRequest.
//   - error: An error object indicating failure to pack the parameters, nil if successful.
func (oqr *OptionQuoteRequest) getParams() ([]parameters.MarketDataParam, error) {
	if oqr == nil {
		return nil, fmt.Errorf("OptionQuoteRequest is nil")
	}
	params := []parameters.MarketDataParam{oqr.symbolParams}
	return params, nil
}

// Packed sends the OptionQuoteRequest and returns the OptionQuotesResponse.
// An optional MarketDataClient can be passed to replace the client used in the request.
// Parameters:
//   - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// Returns:
//   - *models.OptionQuotesResponse: A pointer to the OptionQuotesResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (oqr *OptionQuoteRequest) Packed(optionalClients ...*MarketDataClient) (*models.OptionQuotesResponse, error) {
	if oqr == nil {
		return nil, fmt.Errorf("OptionQuoteRequest is nil")
	}

	// Replace the client if an optional client is provided
	if len(optionalClients) > 0 && optionalClients[0] != nil {
		oqr.baseRequest.client = optionalClients[0]
	}

	var oqrResp models.OptionQuotesResponse
	_, err := oqr.baseRequest.client.GetFromRequest(oqr.baseRequest, &oqrResp)
	if err != nil {
		return nil, err
	}

	return &oqrResp, nil
}

// Get sends the OptionQuoteRequest, unpacks the OptionQuotesResponse, and returns a slice of OptionQuote.
// It returns an error if the request or unpacking fails.
// An optional MarketDataClient can be passed to replace the client used in the request.
// Parameters:
//   - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// Returns:
//   - []models.OptionQuote: A slice of OptionQuote containing the unpacked options quotes data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (oqr *OptionQuoteRequest) Get(optionalClients ...*MarketDataClient) ([]models.OptionQuote, error) {
	if oqr == nil {
		return nil, fmt.Errorf("OptionQuoteRequest is nil")
	}

	// Use the Packed method to make the request, passing along any optional client
	oqrResp, err := oqr.Packed(optionalClients...)
	if err != nil {
		return nil, err
	}

	// Unpack the data using the Unpack method in the response
	data, err := oqrResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// OptionQuotes creates a new OptionQuotesRequest and associates it with the provided client.
// If no client is provided, it uses the default client.
// Parameters:
//   - client: A variadic parameter that can accept zero or one MarketDataClient pointer. If no client is provided, the default client is used.
//
// Returns:
//   - *OptionQuotesRequest: A pointer to the newly created OptionQuotesRequest with default parameters and associated client.
func OptionQuote(client ...*MarketDataClient) *OptionQuoteRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[1]["options"]["quotes"]

	oqr := &OptionQuoteRequest{
		baseRequest:  baseReq,
		symbolParams: &parameters.SymbolParams{},
	}

	baseReq.child = oqr

	return oqr
}
