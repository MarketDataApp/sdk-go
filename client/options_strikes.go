package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// OptionsStrikesRequest represents a request to the options strikes endpoint.
// It encapsulates parameters for underlying symbol, expiration, and date to be used in the request.
// This struct provides methods such as UnderlyingSymbol(), Expiration(), and Date() to set these parameters respectively.
//
// Public Methods:
// - UnderlyingSymbol(symbol string) *OptionsStrikesRequest: Sets the underlying symbol parameter for the request.
// - Expiration(expiration string) *OptionsStrikesRequest: Sets the expiration parameter for the request.
// - Date(date string) *OptionsStrikesRequest: Sets the date parameter for the request.
// - Packed() (*models.OptionsStrikesResponse, error): Sends the OptionsStrikesRequest and returns the OptionsStrikesResponse.
// - Get() ([]models.OptionsStrikes, error): Sends the OptionsStrikesRequest, unpacks the OptionsStrikesResponse, and returns a slice of OptionsStrikes.
type OptionsStrikesRequest struct {
	*baseRequest
	underlyingSymbol *parameters.SymbolParams
	expiration       *parameters.OptionParams
	date             *parameters.DateParams
}

// UnderlyingSymbol sets the underlying symbol parameter for the OptionsStrikesRequest.
// This method is used to specify the symbol of the underlying asset for which options strikes data is requested.
//
// Parameters:
// - underlyingSymbol: A string representing the symbol to be set.
//
// Returns:
// - *OptionsStrikesRequest: This method returns a pointer to the OptionsStrikesRequest instance it was called on. This allows for method chaining.
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
// Parameters:
// - expiration: A string representing the expiration date to be set.
//
// Returns:
// - *OptionsStrikesRequest: This method returns a pointer to the OptionsStrikesRequest instance it was called on. This allows for method chaining.
func (o *OptionsStrikesRequest) Expiration(expiration string) *OptionsStrikesRequest {
	if o.expiration == nil {
		o.expiration = &parameters.OptionParams{}
	}
	if err := o.expiration.SetExpiration(expiration); err != nil {
		o.Error = err
	}
	return o
}

// Date sets the date parameter for the OptionsStrikesRequest.
// This method is used to specify the date for which the options strikes data is requested.
//
// Parameters:
// - date: A string representing the date to be set.
//
// Returns:
// - *OptionsStrikesRequest: This method returns a pointer to the OptionsStrikesRequest instance it was called on. This allows for method chaining.
func (o *OptionsStrikesRequest) Date(date string) *OptionsStrikesRequest {
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
// This method checks if the OptionsStrikesRequest receiver is nil, returning an error if true.
// Otherwise, it proceeds to send the request and returns the OptionsStrikesResponse along with any error encountered during the request.
//
// Returns:
// - *models.OptionsStrikesResponse: A pointer to the OptionsStrikesResponse obtained from the request.
// - error: An error object that indicates a failure in sending the request.
func (o *OptionsStrikesRequest) Packed() (*models.OptionsStrikesResponse, error) {
	if o == nil {
		return nil, fmt.Errorf("OptionsStrikesRequest is nil")
	}
	var oResp models.OptionsStrikesResponse
	_, err := o.baseRequest.client.GetFromRequest(o.baseRequest, &oResp)
	if err != nil {
		return nil, err
	}

	return &oResp, nil
}

// Get sends the OptionsStrikesRequest, unpacks the OptionsStrikesResponse, and returns a slice of OptionsStrikes.
// It returns an error if the request or unpacking fails. This method is crucial for obtaining the actual options strikes data
// from the options strikes request. The method first checks if the OptionsStrikesRequest receiver is nil, which would
// result in an error as the request cannot be sent. It then proceeds to send the request using the Packed method.
// Upon receiving the response, it unpacks the data into a slice of OptionsStrikes using the Unpack method from the response.
//
// Returns:
// - []models.OptionsStrikes: A slice of OptionsStrikes containing the unpacked data from the response.
// - error: An error object that indicates a failure in sending the request or unpacking the response.
func (o *OptionsStrikesRequest) Get() ([]models.OptionsStrikes, error) {
	if o == nil {
		return nil, fmt.Errorf("OptionsStrikesRequest is nil")
	}

	oResp, err := o.Packed()
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

// OptionsStrikes creates a new OptionsStrikesRequest and associates it with the provided client.
// If no client is provided, it uses the default client. This function initializes the request
// with default parameters for underlying symbol, expiration, and date, and sets the request path based on
// the predefined endpoints for options strikes.
//
// Parameters:
// - client: A variadic parameter that can accept zero or one MarketDataClient pointer. If no client is provided,
//   the default client is used.
//
// Returns:
// - *OptionsStrikesRequest: A pointer to the newly created OptionsStrikesRequest with default parameters and associated client.
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
