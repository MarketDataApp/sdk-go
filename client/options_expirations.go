package client

import (
	"fmt"
	"time"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// OptionsExpirationsRequest represents a request for retrieving options expirations data.
// It encapsulates parameters for the underlying symbol and strike price to be used in the request.
// This struct provides methods such as UnderlyingSymbol() and Strike() to set these parameters respectively.
//
// Public Methods:
// - Strike(strike float64) *OptionsExpirationsRequest: Sets the strike price parameter for the options expirations request.
// - UnderlyingSymbol(symbol string) *OptionsExpirationsRequest: Sets the underlying symbol parameter for the options expirations request.
type OptionsExpirationsRequest struct {
	*baseRequest
	underlyingSymbol *parameters.SymbolParams
	strike           *parameters.OptionParams
}

// Strike sets the strike price parameter for the OptionsExpirationsRequest.
// This method is used to specify a particular strike price for filtering the options expirations.
// Parameters:
// - strike: A float64 representing the strike price to be set.
// Returns:
// - *OptionsExpirationsRequest: This method returns a pointer to the OptionsExpirationsRequest instance it was called on, allowing for method chaining.
func (o *OptionsExpirationsRequest) Strike(strike float64) *OptionsExpirationsRequest {
	if o.strike == nil {
		o.strike = &parameters.OptionParams{}
	}
	if err := o.strike.SetStrike(strike); err != nil {
		o.Error = err
	}
	return o
}

// UnderlyingSymbol sets the underlying symbol parameter for the OptionsExpirationsRequest.
// This method is used to specify the symbol of the underlying asset for which the options expirations are requested.
// Parameters:
// - symbol: A string representing the underlying symbol to be set.
// Returns:
// - *OptionsExpirationsRequest: This method returns a pointer to the OptionsExpirationsRequest instance it was called on, allowing for method chaining.
func (o *OptionsExpirationsRequest) UnderlyingSymbol(symbol string) *OptionsExpirationsRequest {
	if err := o.underlyingSymbol.SetSymbol(symbol); err != nil {
		o.Error = err
	}
	return o
}

// getParams packs the OptionsExpirationsRequest struct into a slice of interface{} and returns it.
// This method is used to gather all the parameters set in the OptionsExpirationsRequest into a single slice for easier manipulation and usage in subsequent requests.
// Returns:
// - []parameters.MarketDataParam: A slice containing all the parameters set in the OptionsExpirationsRequest.
// - error: An error object indicating failure to pack the parameters, nil if successful.
func (o *OptionsExpirationsRequest) getParams() ([]parameters.MarketDataParam, error) {
	if o == nil {
		return nil, fmt.Errorf("OptionsExpirationsRequest is nil")
	}
	params := []parameters.MarketDataParam{o.underlyingSymbol, o.strike}
	return params, nil
}

// Packed sends the OptionsExpirationsRequest and returns the OptionsExpirationsResponse.
// An optional MarketDataClient can be passed to replace the client used in the request.
// Parameters:
// - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//   it replaces the current client for this request.
// Returns:
// - *models.OptionsExpirationsResponse: A pointer to the OptionsExpirationsResponse obtained from the request.
// - error: An error object that indicates a failure in sending the request.
func (o *OptionsExpirationsRequest) Packed(optionalClients ...*MarketDataClient) (*models.OptionsExpirationsResponse, error) {
	if o == nil {
		return nil, fmt.Errorf("OptionsExpirationsRequest is nil")
	}

	// Replace the client if an optional client is provided
	if len(optionalClients) > 0 && optionalClients[0] != nil {
		o.baseRequest.client = optionalClients[0]
	}

	var oResp models.OptionsExpirationsResponse
	_, err := o.baseRequest.client.GetFromRequest(o.baseRequest, &oResp)
	if err != nil {
		return nil, err
	}

	return &oResp, nil
}

// Get sends the OptionsExpirationsRequest, unpacks the OptionsExpirationsResponse, and returns a slice of time.Time.
// It returns an error if the request or unpacking fails.
// An optional MarketDataClient can be passed to replace the client used in the request.
// Parameters:
// - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//   it replaces the current client for this request.
// Returns:
// - []time.Time: A slice of time.Time containing the unpacked options expirations data from the response.
// - error: An error object that indicates a failure in sending the request or unpacking the response.
func (o *OptionsExpirationsRequest) Get(optionalClients ...*MarketDataClient) ([]time.Time, error) {
	if o == nil {
		return nil, fmt.Errorf("OptionsExpirationsRequest is nil")
	}
	
	// Use the Packed method to make the request, passing along any optional client
	oResp, err := o.Packed(optionalClients...)
	if err != nil {
		return nil, err
	}

	// Unpack the data using the Unpack method in the response
	data, err := oResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// OptionsExpirations creates a new OptionsExpirationsRequest and associates it with the provided client.
// If no client is provided, it uses the default client.
// Parameters:
// - client: A variadic parameter that can accept zero or one MarketDataClient pointer. If no client is provided, the default client is used.
// Returns:
// - *OptionsExpirationsRequest: A pointer to the newly created OptionsExpirationsRequest with default parameters and associated client.
func OptionsExpirations(client ...*MarketDataClient) *OptionsExpirationsRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[1]["options"]["expirations"]

	oer := &OptionsExpirationsRequest{
		baseRequest:      baseReq,
		underlyingSymbol: &parameters.SymbolParams{},
		strike:           &parameters.OptionParams{},
	}

	baseReq.child = oer

	return oer
}
