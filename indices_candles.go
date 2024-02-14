package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// IndicesCandlesRequest represents a request to the [/v1/indices/candles/] endpoint.
// It encapsulates parameters for resolution, symbol, and dates to be used in the request.
//
// # Generated By
//
//   - IndexCandles(client ...*MarketDataClient) *IndicesCandlesRequest: IndexCandles creates a new *IndicesCandlesRequest and returns a pointer to the request allowing for method chaining.
//
// # Setter Methods
// 
// These methods are used to set the parameters of the request. They allow for method chaining
// by returning a pointer to the *IndicesCandlesRequest instance they modify.
//
//   - Resolution(string) *IndicesCandlesRequest: Sets the resolution parameter for the request.
//   - Symbol(string) *IndicesCandlesRequest: Sets the symbol parameter for the request.
//   - Date(interface{}) *IndicesCandlesRequest: Sets the date parameter for the request.
//   - From(interface{}) *IndicesCandlesRequest: Sets the 'from' date parameter for the request.
//
// # Execution Methods
// 
// These methods are used to send the request in different formats or retrieve the data.
// They handle the actual communication with the API endpoint.
//
//   - Raw(...*MarketDataClient) (*resty.Response, error): Sends the request as is and returns the raw HTTP response.
//   - Packed(...*MarketDataClient) (*IndicesCandlesResponse, error): Packs the request parameters and sends the request, returning a structured response.
//   - Get(...*MarketDataClient) ([]Candle, error): Sends the request, unpacks the response, and returns the data in a user-friendly format.
//
// [/v1/indices/candles/]: https://www.marketdata.app/docs/api/indices/candles
type IndicesCandlesRequest struct {
	*baseRequest
	resolutionParams *parameters.ResolutionParams // Holds the resolution parameter of the request.
	symbolParams     *parameters.SymbolParams     // Holds the symbol parameter of the request.
	dateParams       *parameters.DateParams       // Holds the date parameters of the request.
}

// Resolution sets the resolution parameter for the [IndicesCandlesRequest].
// This method is used to specify the granularity of the candle data to be retrieved.
// It modifies the resolutionParams field of the IndicesCandlesRequest instance to store the resolution value.
//
// # Parameters
//
//   - string: A string representing the resolution to be set. Valid resolutions may include values like "D", "5", "1h", etc. See the API's supported resolutions.
//
// # Returns
//
//   - *IndicesCandlesRequest: This method returns a pointer to the *IndicesCandlesRequest instance it was called on. This allows for method chaining, where multiple setter methods can be called in a single statement. If the receiver (*IndicesCandlesRequest) is nil, it returns nil to prevent a panic.
//
// # Notes
//
//   - If an error occurs while setting the resolution (e.g., if the resolution value is not supported), the Error field of the *IndicesCandlesRequest is set with the encountered error, but the method still returns the IndicesCandlesRequest instance to allow for further method calls by the caller.
func (icr *IndicesCandlesRequest) Resolution(q string) *IndicesCandlesRequest {
	if icr == nil {
		return nil
	}
	err := icr.resolutionParams.SetResolution(q)
	if err != nil {
		icr.Error = err
	}
	return icr
}

// Symbol sets the symbol parameter for the [IndicesCandlesRequest].
// This method is used to specify the index symbol for which the candle data is requested.
//
// # Parameters
//
//   - string: A string representing the index symbol to be set.
//
// # Returns
//
//   - *IndicesCandlesRequest: This method returns a pointer to the *IndicesCandlesRequest instance it was called on. This allows for method chaining, where multiple setter methods can be called in a single statement. If the receiver (*IndicesCandlesRequest) is nil, it returns nil to prevent a panic.
//
// # Notes
//
//   - If an error occurs while setting the symbol (e.g., if the symbol value is not supported), the Error field of the IndicesCandlesRequest is set with the encountered error, but the method still returns the IndicesCandlesRequest instance to allow for further method calls or error handling by the caller.
func (icr *IndicesCandlesRequest) Symbol(q string) *IndicesCandlesRequest {
	if icr == nil {
		return nil
	}
	err := icr.symbolParams.SetSymbol(q)
	if err != nil {
		icr.Error = err
	}
	return icr
}

// Date sets the date parameter for the IndicesCandlesRequest.
// This method is used to specify the date for which the candle data is requested.
// It modifies the 'date' field of the IndicesCandlesRequest instance to store the date value.
//
// # Parameters
//
//   - interface{}: An interface{} representing the date to be set. It can be a string, a time.Time object, a Unix int, or any other type that the underlying dates package method can process.
//
// # Returns
//
//   - *IndicesCandlesRequest: This method returns a pointer to the *IndicesCandlesRequest instance it was called on. This allows for method chaining, where multiple setter methods can be called in a single statement.
//
// # Notes
//
//   - If an error occurs while setting the date (e.g., if the date value is not supported), the Error field of the request is set with the encountered error, but the method still returns the *IndicesCandlesRequest instance to allow for further method calls.
func (icr *IndicesCandlesRequest) Date(q interface{}) *IndicesCandlesRequest {
	err := icr.dateParams.SetDate(q)
	if err != nil {
		icr.baseRequest.Error = err
	}
	return icr
}

// From sets the 'from' date parameter for the IndicesCandlesRequest. It configures the starting point of the date range for which the candle data is requested.
//
// # Parameters 
//
//   - interface{}: An interface{} that represents the starting date. It can be a string, a time.Time object, a Unix timestamp or any other type that the underlying dates package can process.
//
// # Returns
//
//   - *IndicesCandlesRequest: A pointer to the *IndicesCandlesRequest instance to allow for method chaining.
func (icr *IndicesCandlesRequest) From(q interface{}) *IndicesCandlesRequest {
	err := icr.dateParams.SetFrom(q)
	if err != nil {
		icr.baseRequest.Error = err
	}
	return icr
}

// To sets the 'to' date parameter for the IndicesCandlesRequest. It configures the ending point of the date range for which the candle data is requested.
//
// # Parameters
//
//   - interface{}: An interface{} that represents the ending date. It can be a string, a time.Time object, or any other type that the underlying SetTo method can process.
//
// # Returns
//
//   - *IndicesCandlesRequest: A pointer to the *IndicesCandlesRequest instance to allow for method chaining.
func (icr *IndicesCandlesRequest) To(q interface{}) *IndicesCandlesRequest {
	err := icr.dateParams.SetTo(q)
	if err != nil {
		icr.baseRequest.Error = err
	}
	return icr
}

// Countback sets the countback parameter for the IndicesCandlesRequest. It specifies the number of candles to return, counting backwards from the 'to' date.
//
// # Parameters
//
//   - int: An int representing the number of candles to return.
//
// # Returns
//
//   - *IndicesCandlesRequest: A pointer to the *IndicesCandlesRequest instance to allow for method chaining.
func (icr *IndicesCandlesRequest) Countback(q int) *IndicesCandlesRequest {
	err := icr.dateParams.SetCountback(q)
	if err != nil {
		icr.baseRequest.Error = err
	}
	return icr
}

// getParams packs the IndicesCandlesRequest struct into a slice of interface{} and returns it.
// This method is used to gather all the parameters set in the IndicesCandlesRequest into a single slice
// for easier manipulation and usage in subsequent requests.
//
// # Returns
//
//   - []parameters.MarketDataParam: A slice containing all the parameters set in the IndicesCandlesRequest.
//   - error: An error object indicating failure to pack the parameters, nil if successful.
func (icr *IndicesCandlesRequest) getParams() ([]parameters.MarketDataParam, error) {
	if icr == nil {
		return nil, fmt.Errorf("IndicesCandlesRequest is nil")
	}
	params := []parameters.MarketDataParam{icr.dateParams, icr.symbolParams, icr.resolutionParams}
	return params, nil
}

// Packed sends the IndicesCandlesRequest and returns the IndicesCandlesResponse.
// This method checks if the IndicesCandlesRequest receiver is nil, returning an error if true.
// An optional MarketDataClient can be passed to replace the client used in the request.
// Otherwise, it proceeds to send the request and returns the IndicesCandlesResponse along with any error encountered during the request.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one *MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// # Returns
//
//   - *models.IndicesCandlesResponse: A pointer to the *IndicesCandlesResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (icr *IndicesCandlesRequest) Packed(optionalClients ...*MarketDataClient) (*models.IndicesCandlesResponse, error) {
	if icr == nil {
		return nil, fmt.Errorf("IndicesCandlesRequest is nil")
	}

	// Replace the client if an optional client is provided
	if len(optionalClients) > 0 && optionalClients[0] != nil {
		icr.baseRequest.client = optionalClients[0]
	}

	var icrResp models.IndicesCandlesResponse
	_, err := icr.baseRequest.client.GetFromRequest(icr.baseRequest, &icrResp)
	if err != nil {
		return nil, err
	}

	return &icrResp, nil
}

// Get sends the [IndicesCandlesRequest], unpacks the [IndicesCandlesResponse], and returns a slice of [IndexCandle].
// It returns an error if the request or unpacking fails. This method is crucial for obtaining the actual candle data
// from the indices candles request. The method first checks if the IndicesCandlesRequest receiver is nil, which would
// result in an error as the request cannot be sent. It then proceeds to send the request using the Packed method.
// Upon receiving the response, it unpacks the data into a slice of IndexCandle using the Unpack method from the response.
// An optional MarketDataClient can be passed to replace the client used in the request.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one *MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// # Returns
//
//   - []models.IndexCandle: A slice of [IndexCandle] containing the unpacked candle data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (icr *IndicesCandlesRequest) Get(optionalClients ...*MarketDataClient) ([]models.Candle, error) {
	if icr == nil {
		return nil, fmt.Errorf("IndicesCandlesRequest is nil")
	}

	// Use the Packed method to make the request, passing along any optional client
	icrResp, err := icr.Packed(optionalClients...)
	if err != nil {
		return nil, err
	}

	// Unpack the data using the Unpack method in the response
	data, err := icrResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// IndexCandles creates a new [IndicesCandlesRequest] and associates it with the provided client.
// If no client is provided, it uses the default client. This function initializes the request
// with default parameters for date, resolution, and symbol, and sets the request path based on
// the predefined endpoints for indices candles.
//
// # Parameters
//
//   - ...*MarketDataClient: A variadic parameter that can accept zero or one *MarketDataClient pointer. If no client is provided,
//     the default client is used.
//
// # Returns
//
//   - *IndicesCandlesRequest: A pointer to the newly created *IndicesCandlesRequest with default parameters and associated client.
func IndexCandles(client ...*MarketDataClient) *IndicesCandlesRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[1]["indices"]["candles"]

	icr := &IndicesCandlesRequest{
		baseRequest:      baseReq,
		dateParams:       &parameters.DateParams{},
		resolutionParams: &parameters.ResolutionParams{},
		symbolParams:     &parameters.SymbolParams{},
	}

	baseReq.child = icr

	return icr
}
