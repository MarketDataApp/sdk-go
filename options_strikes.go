// Package client provides functionalities to interact with the Option Strikes endpoint.
// Retrieve a complete or filtered list of option strike prices for a given underlying symbol. Both real-time and historical requests are possible.
//
// # Making Requests
//
// Utilize [OptionStrikesRequest] for querying the endpoint through one of the three available methods:
//
//	| Method     | Execution Level | Return Type                  | Description                                                                                                                      |
//	|------------|-----------------|------------------------------|----------------------------------------------------------------------------------------------------------------------------------|
//	| **Get**    | Direct          | `[]OptionStrikes`           | Immediately fetches a slice of `[]OptionStrikes`, allowing direct access to the options strikes data.                           |
//	| **Packed** | Intermediate    | `*OptionStrikesResponse`    | Delivers a `*OptionStrikesResponse` object containing the data, which requires unpacking to access the `[]OptionStrikes` slice.|
//	| **Raw**    | Low-level       | `*resty.Response`            | Offers the unprocessed `*resty.Response` for those seeking full control and access to the raw JSON or `*http.Response`.          |
package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
	"github.com/go-resty/resty/v2"
)

// OptionStrikesRequest represents a request to the [/v1/options/strikes/] endpoint.
// It encapsulates parameters for underlying symbol, expiration, and date to be used in the request.
// This struct provides methods such as UnderlyingSymbol(), Expiration(), and Date() to set these parameters respectively.
//
// # Setter Methods
//
//   - UnderlyingSymbol(string) *OptionStrikesRequest: Sets the underlying symbol parameter for the request.
//   - Expiration(string) *OptionStrikesRequest: Sets the expiration parameter for the request.
//   - Date(interface{}) *OptionStrikesRequest: Sets the date parameter for the request.
//
// # Execution Methods
//
// These methods are used to send the request in different formats or retrieve the data.
// They handle the actual communication with the API endpoint.
//
//   - Get(...*MarketDataClient) ([]OptionStrikes, error): Sends the request, unpacks the response, and returns the data in a user-friendly format.
//   - Packed(...*MarketDataClient) (*OptionStrikesResponse, error): Returns a struct that contains equal-length slices of primitives. This packed response mirrors Market Data's JSON response.
//   - Raw(...*MarketDataClient) (*resty.Response, error): Sends the request as is and returns the raw HTTP response.
//
// [/v1/options/strikes/]: https://www.marketdata.app/docs/api/options/strikes
type OptionStrikesRequest struct {
	*baseRequest
	underlyingSymbol *parameters.SymbolParams
	expiration       *parameters.OptionParams
	date             *parameters.DateParams
}

// UnderlyingSymbol sets the underlying symbol parameter for the OptionStrikesRequest.
// This method is used to specify the symbol of the underlying asset for which options strikes data is requested.
//
// # Parameters
//
//   - underlyingSymbol: A string representing the symbol to be set.
//
// # Returns
//
//   - *OptionStrikesRequest: This method returns a pointer to the OptionStrikesRequest instance it was called on. This allows for method chaining.
func (o *OptionStrikesRequest) UnderlyingSymbol(underlyingSymbol string) *OptionStrikesRequest {
	if o.underlyingSymbol == nil {
		o.underlyingSymbol = &parameters.SymbolParams{}
	}
	if err := o.underlyingSymbol.SetSymbol(underlyingSymbol); err != nil {
		o.Error = err
	}
	return o
}

// Expiration sets the expiration parameter for the OptionStrikesRequest.
// This method is used to specify the expiration date of the options for which strikes data is requested.
//
// # Parameters
//
//   - expiration: A string representing the expiration date to be set.
//
// # Returns
//
//   - *OptionStrikesRequest: This method returns a pointer to the OptionStrikesRequest instance it was called on. This allows for method chaining.
func (o *OptionStrikesRequest) Expiration(expiration string) *OptionStrikesRequest {
	if o.expiration == nil {
		o.expiration = &parameters.OptionParams{}
	}
	if err := o.expiration.SetExpiration(expiration); err != nil {
		o.Error = err
	}
	return o
}

// Date sets the date parameter for the OptionStrikesRequest. This is used to make a historical request.
// This method is used to specify the date for which the options strikes data is requested.
//
// # Parameters
//
//   - interface{}: An interface{} representing the date to be set. It can be a string, a time.Time object, a Unix int, or any other type that the underlying dates package method can process.
//
// # Returns
//
//   - *OptionStrikesRequest: This method returns a pointer to the OptionStrikesRequest instance it was called on. This allows for method chaining.
func (o *OptionStrikesRequest) Date(date interface{}) *OptionStrikesRequest {
	if o.date == nil {
		o.date = &parameters.DateParams{}
	}
	if err := o.date.SetDate(date); err != nil {
		o.Error = err
	}
	return o
}

func (o *OptionStrikesRequest) getParams() ([]parameters.MarketDataParam, error) {
	if o == nil {
		return nil, fmt.Errorf("OptionStrikesRequest is nil")
	}
	params := []parameters.MarketDataParam{o.underlyingSymbol, o.expiration, o.date}
	return params, nil
}

// Raw executes the OptionStrikesRequest and returns the raw *resty.Response.
// This method allows for an optional MarketDataClient to be passed. If provided, this client replaces the one currently
// attached to the OptionStrikesRequest. The *resty.Response can be used to directly access the raw JSON or *http.Response.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept an optional *MarketDataClient pointer. If provided, this client is used for the request instead of the default.
//
// # Returns
//
//   - *resty.Response: The raw HTTP response from the executed OptionStrikesRequest.
//   - error: An error object if the OptionStrikesRequest is nil, the MarketDataClient is nil, or if an error occurs during the request execution.
func (osr *OptionStrikesRequest) Raw(optionalClients ...*MarketDataClient) (*resty.Response, error) {
	return osr.baseRequest.Raw(optionalClients...)
}

// Packed sends the OptionStrikesRequest and returns the OptionStrikesResponse.
// An optional MarketDataClient can be passed to replace the client used in the request.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided, it replaces the current client for this request.
//
// # Returns
//
//   - *models.OptionStrikesResponse: A pointer to the OptionStrikesResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (osr *OptionStrikesRequest) Packed(optionalClients ...*MarketDataClient) (*models.OptionStrikesResponse, error) {
	if osr == nil {
		return nil, fmt.Errorf("OptionStrikesRequest is nil")
	}

	// Replace the client if an optional client is provided
	if len(optionalClients) > 0 && optionalClients[0] != nil {
		osr.baseRequest.client = optionalClients[0]
	}

	var osrResp models.OptionStrikesResponse
	_, err := osr.baseRequest.client.GetFromRequest(osr.baseRequest, &osrResp)
	if err != nil {
		return nil, err
	}

	return &osrResp, nil
}

// Get sends the OptionStrikesRequest, unpacks the OptionStrikesResponse, and returns a slice of OptionStrikes.
// It returns an error if the request or unpacking fails.
// An optional MarketDataClient can be passed to replace the client used in the request.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided, it replaces the current client for this request.
//
// # Returns
//
//   - []models.OptionStrikes: A slice of OptionStrikes containing the unpacked options strikes data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (osr *OptionStrikesRequest) Get(optionalClients ...*MarketDataClient) ([]models.OptionStrikes, error) {
	if osr == nil {
		return nil, fmt.Errorf("OptionStrikesRequest is nil")
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

// OptionStrikes creates a new OptionStrikesRequest and associates it with the provided client.
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
//   - *OptionStrikesRequest: A pointer to the newly created OptionStrikesRequest with default parameters and associated client.
func OptionStrikes(client ...*MarketDataClient) *OptionStrikesRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[1]["options"]["strikes"]

	osr := &OptionStrikesRequest{
		baseRequest:      baseReq,
		underlyingSymbol: &parameters.SymbolParams{},
		expiration:       &parameters.OptionParams{},
		date:             &parameters.DateParams{},
	}

	baseReq.child = osr

	return osr
}
