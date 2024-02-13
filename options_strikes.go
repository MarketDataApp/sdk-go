package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// OptionsStrikesRequest represents a request to the [/v1/options/strikes/] endpoint.
// It encapsulates parameters for underlying symbol, expiration, and date to be used in the request.
// This struct provides methods such as UnderlyingSymbol(), Expiration(), and Date() to set these parameters respectively.
//
// # Setter Methods
//
//   - UnderlyingSymbol(string) *OptionsStrikesRequest: Sets the underlying symbol parameter for the request.
//   - Expiration(string) *OptionsStrikesRequest: Sets the expiration parameter for the request.
//   - Date(interface{}) *OptionsStrikesRequest: Sets the date parameter for the request.
//
// # Execution Methods
//
// These methods are used to send the request in different formats or retrieve the data.
// They handle the actual communication with the API endpoint.
//
//   - Raw() (*resty.Response, error): Sends the request as is and returns the raw HTTP response.
//   - Packed() (*IndicesCandlesResponse, error): Packs the request parameters and sends the request, returning a structured response.
//   - Get() ([]Candle, error): Sends the request, unpacks the response, and returns the data in a user-friendly format.
// [/v1/options/strikes/]: https://www.marketdata.app/docs/api/options/strikes
type OptionsStrikesRequest struct {
	*baseRequest
	underlyingSymbol *parameters.SymbolParams
	expiration       *parameters.OptionParams
	date             *parameters.DateParams
}

// UnderlyingSymbol sets the underlying symbol parameter for the OptionsStrikesRequest.
// This method is used to specify the symbol of the underlying asset for which options strikes data is requested.
//
// # Parameters
//
//   - underlyingSymbol: A string representing the symbol to be set.
//
// # Returns
//
//   - *OptionsStrikesRequest: This method returns a pointer to the OptionsStrikesRequest instance it was called on. This allows for method chaining.
func (o *OptionsStrikesRequest) UnderlyingSymbol(underlyingSymbol string) *OptionsStrikesRequest {
	if o.underlyingSymbol == nil {
		o.underlyingSymbol = &parameters.SymbolParams{}
	}
	if err := o.underlyingSymbol.SetSymbol(underlyingSymbol); err != nil {
		o.Error = err
	}
	return o
}

// Expiration sets the expiration parameter for the OptionsStrikesRequest.
// This method is used to specify the expiration date of the options for which strikes data is requested.
//
// # Parameters
//
//   - expiration: A string representing the expiration date to be set.
//
// # Returns
//
//   - *OptionsStrikesRequest: This method returns a pointer to the OptionsStrikesRequest instance it was called on. This allows for method chaining.
func (o *OptionsStrikesRequest) Expiration(expiration string) *OptionsStrikesRequest {
	if o.expiration == nil {
		o.expiration = &parameters.OptionParams{}
	}
	if err := o.expiration.SetExpiration(expiration); err != nil {
		o.Error = err
	}
	return o
}

// Date sets the date parameter for the OptionsStrikesRequest. This is used to make a historical request.
// This method is used to specify the date for which the options strikes data is requested.
//
// # Parameters
//
//   - interface{}: An interface{} representing the date to be set. It can be a string, a time.Time object, a Unix int, or any other type that the underlying dates package method can process.
//
// # Returns
//
//   - *OptionsStrikesRequest: This method returns a pointer to the OptionsStrikesRequest instance it was called on. This allows for method chaining.
func (o *OptionsStrikesRequest) Date(date interface{}) *OptionsStrikesRequest {
	if o.date == nil {
		o.date = &parameters.DateParams{}
	}
	if err := o.date.SetDate(date); err != nil {
		o.Error = err
	}
	return o
}

func (o *OptionsStrikesRequest) getParams() ([]parameters.MarketDataParam, error) {
	if o == nil {
		return nil, fmt.Errorf("OptionsStrikesRequest is nil")
	}
	params := []parameters.MarketDataParam{o.underlyingSymbol, o.expiration, o.date}
	return params, nil
}

// Packed sends the OptionsStrikesRequest and returns the OptionsStrikesResponse.
// An optional MarketDataClient can be passed to replace the client used in the request.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided, it replaces the current client for this request.
//
// # Returns
//
//   - *models.OptionsStrikesResponse: A pointer to the OptionsStrikesResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (osr *OptionsStrikesRequest) Packed(optionalClients ...*MarketDataClient) (*models.OptionsStrikesResponse, error) {
	if osr == nil {
		return nil, fmt.Errorf("OptionsStrikesRequest is nil")
	}

	// Replace the client if an optional client is provided
	if len(optionalClients) > 0 && optionalClients[0] != nil {
		osr.baseRequest.client = optionalClients[0]
	}

	var osrResp models.OptionsStrikesResponse
	_, err := osr.baseRequest.client.GetFromRequest(osr.baseRequest, &osrResp)
	if err != nil {
		return nil, err
	}

	return &osrResp, nil
}

// Get sends the OptionsStrikesRequest, unpacks the OptionsStrikesResponse, and returns a slice of OptionsStrikes.
// It returns an error if the request or unpacking fails.
// An optional MarketDataClient can be passed to replace the client used in the request.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided, it replaces the current client for this request.
//
// # Returns
//
//   - []models.OptionsStrikes: A slice of OptionsStrikes containing the unpacked options strikes data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (osr *OptionsStrikesRequest) Get(optionalClients ...*MarketDataClient) ([]models.OptionsStrikes, error) {
	if osr == nil {
		return nil, fmt.Errorf("OptionsStrikesRequest is nil")
	}

	// Use the Packed method to make the request, passing along any optional client
	osrResp, err := osr.Packed(optionalClients...)
	if err != nil {
		return nil, err
	}

	// Unpack the data using the Unpack method in the response
	data, err := osrResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// OptionsStrikes creates a new OptionsStrikesRequest and associates it with the provided client.
// If no client is provided, it uses the default client. This function initializes the request
// with default parameters for underlying symbol, expiration, and date, and sets the request path based on
// the predefined endpoints for options strikes.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one MarketDataClient pointer. If no client is provided, the default client is used.
//
// # Returns
//
//   - *OptionsStrikesRequest: A pointer to the newly created OptionsStrikesRequest with default parameters and associated client.
func OptionsStrikes(client ...*MarketDataClient) *OptionsStrikesRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[1]["options"]["strikes"]

	osr := &OptionsStrikesRequest{
		baseRequest:      baseReq,
		underlyingSymbol: &parameters.SymbolParams{},
		expiration:       &parameters.OptionParams{},
		date:             &parameters.DateParams{},
	}

	baseReq.child = osr

	return osr
}
