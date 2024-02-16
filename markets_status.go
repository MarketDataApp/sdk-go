// Package client includes types and methods to access the Market Status endpoint. Retrieve current, future, and historical open / closed status information.
//
// # Making Requests
//
// Utilize [MarketStatusRequest] to make requests to the endpoint through one of the three supported execution methods:
//
//	| Method     | Execution     | Return Type                 | Description                                                                                                     |
//	|------------|---------------|-----------------------------|-----------------------------------------------------------------------------------------------------------------|
//	| **Get**    | Direct        | `[]MarketStatusReport`      | Directly returns a slice of `[]MarketStatusReport`, facilitating individual access to each market status entry. |
//	| **Packed** | Intermediate  | `*MarketStatusResponse`     | Returns a packed `*MarketStatusResponse` object. Must be unpacked to access the `[]MarketStatusReport` slice.   |
//	| **Raw**    | Low-level     | `*resty.Response`           | Offers the raw `*resty.Response` for utmost flexibility. Direct access to raw JSON or `*http.Response`.         |
package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// MarketStatusRequest represents a request to the [/v1/markets/status/] endpoint for market status information.
// It encapsulates parameters for country, and date to be used in the request.
// This struct provides methods such as Country(), Date(), From(), To(), and Countback() to set these parameters respectively.
//
// # Generated By
//
//   - MarketStatus(client ...*MarketDataClient) *MarketStatusRequest: MarketStatus creates a new *MarketStatusRequest and returns a pointer to the request allowing for method chaining.
//
// # Setter Methods
//
//   - Country(string) *MarketStatusRequest: Sets the country parameter for the request.
//   - Date(interface{}) *MarketStatusRequest: Sets the date parameter for the request.
//   - From(interface{}) *MarketStatusRequest: Sets the 'from' date parameter for the request.
//   - To(interface{}) *MarketStatusRequest: Sets the 'to' date parameter for the request.
//   - Countback(int) *MarketStatusRequest: Sets the countback parameter for the request.
//
// # Execution Methods
//
// These methods are used to send the request in different formats or retrieve the data.
// They handle the actual communication with the API endpoint.
//
//   - Get(...*MarketDataClient) ([]MarketStatusReport, error): Sends the request, unpacks the response, and returns the data in a user-friendly format.
//   - Packed(...*MarketDataClient) (*MarketStatusResponse, error): Returns a struct that contains equal-length slices of primitives. This packed response mirrors Market Data's JSON response.
//   - Raw(...*MarketDataClient) (*resty.Response, error): Sends the request as is and returns the raw HTTP response.
//
// [/v1/markets/status/]: https://www.marketdata.app/docs/api/markets/status
type MarketStatusRequest struct {
	*baseRequest
	countryParams   *parameters.CountryParams
	universalParams *parameters.UniversalParams
	dateParams      *parameters.DateParams
}

// getParams packs the MarketStatusRequest struct into a slice of interface{} and returns it.
// This method is used to gather all the parameters set in the MarketStatusRequest into a single slice
// for easier manipulation and usage in subsequent requests.
//
// # Returns
//
//   - []parameters.MarketDataParam: A slice containing all the parameters set in the MarketStatusRequest.
//   - error: An error object indicating failure to pack the parameters, nil if successful.
func (msr *MarketStatusRequest) getParams() ([]parameters.MarketDataParam, error) {
	if msr.countryParams == nil {
		return nil, fmt.Errorf("required struct CountryParams doesn't exist")
	}
	if msr.universalParams == nil {
		return nil, fmt.Errorf("required struct UniversalParams doesn't exist")
	}
	if msr.dateParams == nil {
		return nil, fmt.Errorf("required struct DateParams doesn't exist")
	}
	params := []parameters.MarketDataParam{msr.countryParams, msr.universalParams, msr.dateParams}
	return params, nil
}

// Country sets the country parameter of the MarketStatusRequest.
// This method is used to specify the country for which the market status is requested.
// It modifies the countryParams field of the MarketStatusRequest instance to store the country value.
//
// # Parameters
//
//   - q: A string representing the country to be set.
//
// # Returns
//
//   - *MarketStatusRequest: This method returns a pointer to the MarketStatusRequest instance it was called on. This allows for method chaining, where multiple setter methods can be called in a single statement. If the receiver (*MarketStatusRequest) is nil, it returns nil to prevent a panic.
func (msr *MarketStatusRequest) Country(q string) *MarketStatusRequest {
	err := msr.countryParams.SetCountry(q)
	if err != nil {
		msr.baseRequest.Error = err
	}
	return msr
}

// Date sets the date parameter of the MarketStatusRequest.
// This method is used to specify the date for which the market status is requested.
// It modifies the dateParams field of the MarketStatusRequest instance to store the date value.
//
// # Parameters
//
//   - interface{}: An interface{} that represents the starting date. It can be a string, a time.Time object, a Unix timestamp or any other type that the underlying dates package can process.
//
// # Returns
//
//   - *MarketStatusRequest: This method returns a pointer to the MarketStatusRequest instance it was called on. This allows for method chaining. If the receiver (*MarketStatusRequest) is nil, it returns nil to prevent a panic.
func (msr *MarketStatusRequest) Date(q interface{}) *MarketStatusRequest {
	err := msr.dateParams.SetDate(q)
	if err != nil {
		msr.baseRequest.Error = err
	}
	return msr
}

// From sets the 'from' date parameter of the MarketStatusRequest.
// This method is used to specify the starting date of the period for which the market status is requested.
// It modifies the dateParams field of the MarketStatusRequest instance to store the 'from' date value.
//
// # Parameters
//
//   - interface{}: The 'from' date to be set.
//
// # Returns
//
//   - *MarketStatusRequest: This method returns a pointer to the MarketStatusRequest instance it was called on. This allows for method chaining. If the receiver (*MarketStatusRequest) is nil, it returns nil to prevent a panic.
func (msr *MarketStatusRequest) From(q interface{}) *MarketStatusRequest {
	err := msr.dateParams.SetFrom(q)
	if err != nil {
		msr.baseRequest.Error = err
	}
	return msr
}

// To sets the 'to' date parameter of the MarketStatusRequest.
// This method is used to specify the ending date of the period for which the market status is requested.
// It modifies the dateParams field of the MarketStatusRequest instance to store the 'to' date value.
//
// # Parameters
//
//   - interface{}: The 'to' date to be set.
//
// # Returns
//
//   - *MarketStatusRequest: This method returns a pointer to the MarketStatusRequest instance it was called on. This allows for method chaining. If the receiver (*MarketStatusRequest) is nil, it returns nil to prevent a panic.
func (msr *MarketStatusRequest) To(q interface{}) *MarketStatusRequest {
	err := msr.dateParams.SetTo(q)
	if err != nil {
		msr.baseRequest.Error = err
	}
	return msr
}

// Countback sets the countback parameter for the MarketStatusRequest. It specifies the number of days to return, counting backwards from the 'to' date.
//
// # Parameters
//
//   - int: The number of days to return before `to`.
//
// # Returns
//
//   - *MarketStatusRequest: A pointer to the MarketStatusRequest instance to allow for method chaining.
func (msr *MarketStatusRequest) Countback(q int) *MarketStatusRequest {
	err := msr.dateParams.SetCountback(q)
	if err != nil {
		msr.baseRequest.Error = err
	}
	return msr
}

// Packed sends the MarketStatusRequest and returns the MarketStatusResponse.
// This method checks if the MarketStatusRequest receiver is nil, returning an error if true.
// An optional MarketDataClient can be passed to replace the client used in the request.
// Otherwise, it proceeds to send the request and returns the MarketStatusResponse along with any error encountered during the request.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// # Returns
//
//   - *models.MarketStatusResponse: A pointer to the *MarketStatusResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (msr *MarketStatusRequest) Packed(optionalClients ...*MarketDataClient) (*models.MarketStatusResponse, error) {
	if msr == nil {
		return nil, fmt.Errorf("MarketStatusRequest is nil")
	}

	// Replace the client if an optional client is provided
	if len(optionalClients) > 0 && optionalClients[0] != nil {
		msr.baseRequest.client = optionalClients[0]
	}

	var msrResp models.MarketStatusResponse
	_, err := msr.baseRequest.client.GetFromRequest(msr.baseRequest, &msrResp)
	if err != nil {
		return nil, err
	}

	return &msrResp, nil
}

// Get sends the MarketStatusRequest, unpacks the MarketStatusResponse, and returns a slice of MarketStatusReport.
// It returns an error if the request or unpacking fails. This method is crucial for obtaining the actual market status data
// from the market status request. The method first checks if the MarketStatusRequest receiver is nil, which would
// result in an error as the request cannot be sent. It then proceeds to send the request using the Packed method.
// Upon receiving the response, it unpacks the data into a slice of MarketStatusReport using the Unpack method from the response.
// An optional MarketDataClient can be passed to replace the client used in the request.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one *MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// # Returns
//
//   - []models.MarketStatusReport: A slice of MarketStatusReport containing the unpacked market status data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (msr *MarketStatusRequest) Get(optionalClients ...*MarketDataClient) ([]models.MarketStatusReport, error) {
	if msr == nil {
		return nil, fmt.Errorf("MarketStatusRequest is nil")
	}

	// Use the Packed method to make the request, passing along any optional client
	msrResp, err := msr.Packed(optionalClients...)
	if err != nil {
		return nil, err
	}

	// Unpack the data using the Unpack method in the response
	data, err := msrResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// MarketStatus creates a new MarketStatusRequest and associates it with the provided client.
// If no client is provided, it uses the default client. This function initializes the request
// with default parameters for country, universal, and date, and sets the request path based on
// the predefined endpoints for market status.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one *MarketDataClient pointer. If no client is provided,
//     the default client is used.
//
// # Returns
//
//   - *MarketStatusRequest: A pointer to the newly created *MarketStatusRequest with default parameters and associated client.
func MarketStatus(clients ...*MarketDataClient) *MarketStatusRequest {
	baseReq := newBaseRequest(clients...)

	msr := &MarketStatusRequest{
		baseRequest:     baseReq,
		countryParams:   &parameters.CountryParams{},
		universalParams: &parameters.UniversalParams{},
		dateParams:      &parameters.DateParams{},
	}

	baseReq.child = msr

	msr.Country("US") // Set default country value to "US"

	path, ok := endpoints[1]["markets"]["status"]
	if !ok {
		msr.baseRequest.Error = fmt.Errorf("path not found for market status")
		return msr
	}
	msr.baseRequest.path = path

	return msr
}
