// Package client provides functionalities to interact with the Options Lookup endpoint.
// Lookup the OCC-formatted option symbol based on user input.
//
// # Making Requests
//
// Utilize [OptionsLookupRequest] for querying the endpoint through one of the three available methods:
//
//	| Method     | Execution Level | Return Type                  | Description                                                                                                             |
//	|------------|-----------------|------------------------------|-------------------------------------------------------------------------------------------------------------------------|
//	| **Get**    | Direct          | `string`                     | Immediately fetches and `string`, allowing direct access to the option symbol.                                          |
//	| **Packed** | Intermediate    | `*OptionLookupResponse`      | Delivers a `*OptionLookupResponse` object containing the data, which requires unpacking to access the `string` data.    |
//	| **Raw**    | Low-level       | `*resty.Response`            | Offers the unprocessed `*resty.Response` for those seeking full control and access to the raw JSON or `*http.Response`. |
package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// OptionsLookupRequest represents a request to the [/v1/options/lookup/] endpoint for retrieving an OCC-formatted option symbol based on user input.
// It encapsulates parameters for user input to be used in the request.
//
// # Generated By
//
//   - OptionsLookup(client ...*MarketDataClient) *OptionsLookupRequest: OptionsLookup creates a new *OptionsLookupRequest and returns a pointer to the request allowing for method chaining.
//
// # Setter Methods
//
//   - UserInput(string) *OptionLookupRequest: Sets the user input parameter for the request.
//
// # Execution Methods
//
// These methods are used to send the request in different formats or retrieve the data.
// They handle the actual communication with the API endpoint.
//
//   - Get(...*MarketDataClient) (string, error): Sends the request, unpacks the response, and returns the data in a user-friendly format.
//   - Packed(...*MarketDataClient) (*OptionLookupResponse, error): Returns a struct that contains equal-length slices of primitives. This packed response mirrors Market Data's JSON response.
//   - Raw(...*MarketDataClient) (*resty.Response, error): Sends the request as is and returns the raw HTTP response.
//
// [/v1/options/lookup/]: https://www.marketdata.app/docs/api/options/lookup
type OptionLookupRequest struct {
	*baseRequest
	userInput *parameters.UserInputParams
}

// UserInput sets the user input parameter for the OptionsLookupRequest.
// This method is used to specify the user input for which the options data is requested.
//
// # Parameters
//
//   - string: A string representing the text to lookup with the OptionsLookupRequest endpoint.
//
// # Returns
//
//   - *OptionsLookupRequest: This method returns a pointer to the OptionsLookupRequest instance it was called on, allowing for method chaining.
func (o *OptionLookupRequest) UserInput(userInput string) *OptionLookupRequest {
	if o.userInput == nil {
		o.userInput = &parameters.UserInputParams{}
	}
	if err := o.userInput.SetUserInput(userInput); err != nil {
		o.Error = err
	}
	return o
}

// getParams packs the OptionsLookupRequest struct into a slice of interface{} and returns it.
// This method is used to gather all the parameters set in the OptionsLookupRequest into a single slice for easier manipulation and usage in subsequent requests.
//
// # Returns
//
//   - []parameters.MarketDataParam: A slice containing all the parameters set in the OptionsLookupRequest.
//   - error: An error object indicating failure to pack the parameters, nil if successful.
func (o *OptionLookupRequest) getParams() ([]parameters.MarketDataParam, error) {
	if o == nil {
		return nil, fmt.Errorf("OptionLookupRequest is nil")
	}
	params := []parameters.MarketDataParam{o.userInput}
	return params, nil
}

// Packed sends the OptionLookupRequest and returns the OptionsLookupResponse.
// An optional MarketDataClient can be passed to replace the client used in the request.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided, it replaces the current client for this request.
//
// # Returns
//
//   - *models.OptionsLookupResponse: A pointer to the OptionsLookupResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (o *OptionLookupRequest) Packed(optionalClients ...*MarketDataClient) (*models.OptionLookupResponse, error) {
	if o == nil {
		return nil, fmt.Errorf("OptionsLookupRequest is nil")
	}

	// Replace the client if an optional client is provided
	if len(optionalClients) > 0 && optionalClients[0] != nil {
		o.baseRequest.client = optionalClients[0]
	}

	var oResp models.OptionLookupResponse
	_, err := o.baseRequest.client.GetFromRequest(o.baseRequest, &oResp)
	if err != nil {
		return nil, err
	}

	return &oResp, nil
}

// Get sends the OptionLookupRequest, unpacks the OptionsLookupResponse, and returns the unpacked data as a string.
// It returns an error if the request or unpacking fails.
// An optional MarketDataClient can be passed to replace the client used in the request.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided, it replaces the current client for this request.
//
// # Returns
//
//   - string: A string containing the unpacked options data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (o *OptionLookupRequest) Get(optionalClients ...*MarketDataClient) (string, error) {
	if o == nil {
		return "", fmt.Errorf("OptionsLookupRequest is nil")
	}

	// Use the Packed method to make the request, passing along any optional client
	oResp, err := o.Packed(optionalClients...)
	if err != nil {
		return "", err
	}

	// Unpack the data using the Unpack method in the response
	data, err := oResp.Unpack()
	if err != nil {
		return "", err
	}

	return data, nil
}

// OptionLookup creates a new OptionsLookupRequest and associates it with the provided client.
// If no client is provided, it uses the default client.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one MarketDataClient pointer. If no client is provided, the default client is used.
//
// # Returns
//
//   - *OptionsLookupRequest: A pointer to the newly created OptionsLookupRequest with default parameters and associated client.
func OptionLookup(client ...*MarketDataClient) *OptionLookupRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[1]["options"]["lookup"]

	olr := &OptionLookupRequest{
		baseRequest: baseReq,
		userInput:   &parameters.UserInputParams{},
	}

	baseReq.child = olr

	return olr
}
