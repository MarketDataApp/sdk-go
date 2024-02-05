package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// OptionsLookupRequest represents a request for retrieving options data based on user input.
// It encapsulates parameters for user input to be used in the request.
//
// Public Methods:
//   - UserInput(userInput string) *OptionLookupRequest: Sets the user input parameter for the request.
type OptionLookupRequest struct {
	*baseRequest
	userInput *parameters.UserInputParams
}

// UserInput sets the user input parameter for the OptionsLookupRequest.
// This method is used to specify the user input for which the options data is requested.
// Parameters:
//   - userInput: A string representing the user input to be set.
//
// Returns:
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
// Returns:
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
// Parameters:
//   - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// Returns:
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
// Parameters:
//   - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// Returns:
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
// Parameters:
//   - client: A variadic parameter that can accept zero or one MarketDataClient pointer. If no client is provided, the default client is used.
//
// Returns:
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
