package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// IndicesCandlesRequest represents a request to the /v1/indices/candles endpoint.
// It encapsulates parameters for resolution, symbol, and date to be used in the request.
// This struct provides methods such as Resolution(), Symbol(), Date(), and From() to set these parameters respectively.
//
// Public Methods:
// - Resolution(q string) *IndicesCandlesRequest: Sets the resolution parameter for the request.
// - Symbol(q string) *IndicesCandlesRequest: Sets the symbol parameter for the request.
// - Date(q interface{}) *IndicesCandlesRequest: Sets the date parameter for the request.
// - From(q interface{}) *IndicesCandlesRequest: Sets the 'from' date parameter for the request.
type IndicesCandlesRequest struct {
	*baseRequest
	resolutionParams *parameters.ResolutionParams // Holds the resolution parameter of the request.
	symbolParams     *parameters.SymbolParams     // Holds the symbol parameter of the request.
	dateParams       *parameters.DateParams       // Holds the date parameters of the request.
}

// Resolution sets the resolution parameter for the IndicesCandlesRequest.
// This method is used to specify the granularity of the candle data to be retrieved.
// It modifies the resolutionParams field of the IndicesCandlesRequest instance to store the resolution value.
//
// Parameters:
// - q: A string representing the resolution to be set. Valid resolutions may include values like "1min", "5min", "1h", etc., depending on the API's supported resolutions.
//
// Returns:
// - *IndicesCandlesRequest: This method returns a pointer to the IndicesCandlesRequest instance it was called on. This allows for method chaining, where multiple setter methods can be called in a single statement. If the receiver (*IndicesCandlesRequest) is nil, it returns nil to prevent a panic.
//
// Note:
// If an error occurs while setting the resolution (e.g., if the resolution value is not supported), the Error field of the IndicesCandlesRequest is set with the encountered error, but the method still returns the IndicesCandlesRequest instance to allow for further method calls or error handling by the caller.
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
// it may return nil or set the Error field of the IndicesCandlesRequest respectively.
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

// Date sets the date parameter of the IndicesCandlesRequest.
// It accepts an interface{} parameter q which represents the date to be set.
// It returns a pointer to the IndicesCandlesRequest instance to allow for method chaining.
// If an error occurs while setting the date, the Error field of the baseRequest is set.
func (icr *IndicesCandlesRequest) Date(q interface{}) *IndicesCandlesRequest {
	err := icr.dateParams.SetDate(q)
	if err != nil {
		icr.baseRequest.Error = err
	}
	return icr
}

// From sets the 'from' date parameter for the IndicesCandlesRequest. It configures the starting point of the date range for which the candle data is requested.
// Parameters:
// - q: An interface{} that represents the starting date. It can be a string, a time.Time object, or any other type that the underlying SetFrom method can process.
// Returns:
// - *IndicesCandlesRequest: A pointer to the IndicesCandlesRequest instance to allow for method chaining.
func (icr *IndicesCandlesRequest) From(q interface{}) *IndicesCandlesRequest {
	err := icr.dateParams.SetFrom(q)
	if err != nil {
		icr.baseRequest.Error = err
	}
	return icr
}

// To sets the 'to' date parameter for the IndicesCandlesRequest. It configures the ending point of the date range for which the candle data is requested.
// Parameters:
// - q: An interface{} that represents the ending date. It can be a string, a time.Time object, or any other type that the underlying SetTo method can process.
// Returns:
// - *IndicesCandlesRequest: A pointer to the IndicesCandlesRequest instance to allow for method chaining.
func (icr *IndicesCandlesRequest) To(q interface{}) *IndicesCandlesRequest {
	err := icr.dateParams.SetTo(q)
	if err != nil {
		icr.baseRequest.Error = err
	}
	return icr
}

// Countback sets the countback parameter for the IndicesCandlesRequest. It specifies the number of candles to return, counting backwards from the 'to' date.
// Parameters:
// - q: An int representing the number of candles to return.
// Returns:
// - *IndicesCandlesRequest: A pointer to the IndicesCandlesRequest instance to allow for method chaining.
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
// Parameters:
// - None
//
// Returns:
// - []parameters.MarketDataParam: A slice containing all the parameters set in the IndicesCandlesRequest.
// - error: An error object indicating failure to pack the parameters, nil if successful.
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
// Parameters:
// - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//   it replaces the current client for this request.
// Returns:
// - *models.IndicesCandlesResponse: A pointer to the IndicesCandlesResponse obtained from the request.
// - error: An error object that indicates a failure in sending the request.
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

// Get sends the IndicesCandlesRequest, unpacks the IndicesCandlesResponse, and returns a slice of IndexCandle.
// It returns an error if the request or unpacking fails. This method is crucial for obtaining the actual candle data
// from the indices candles request. The method first checks if the IndicesCandlesRequest receiver is nil, which would
// result in an error as the request cannot be sent. It then proceeds to send the request using the Packed method.
// Upon receiving the response, it unpacks the data into a slice of IndexCandle using the Unpack method from the response.
// An optional MarketDataClient can be passed to replace the client used in the request.
// Parameters:
// - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//   it replaces the current client for this request.
// Returns:
// - []models.IndexCandle: A slice of IndexCandle containing the unpacked candle data from the response.
// - error: An error object that indicates a failure in sending the request or unpacking the response.
func (icr *IndicesCandlesRequest) Get(optionalClients ...*MarketDataClient) ([]models.IndexCandle, error) {
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

// IndexCandles creates a new IndicesCandlesRequest and associates it with the provided client.
// If no client is provided, it uses the default client. This function initializes the request
// with default parameters for date, resolution, and symbol, and sets the request path based on
// the predefined endpoints for indices candles.
// Parameters:
// - client: A variadic parameter that can accept zero or one MarketDataClient pointer. If no client is provided,
//   the default client is used.
// Returns:
// - *IndicesCandlesRequest: A pointer to the newly created IndicesCandlesRequest with default parameters and associated client.
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
