package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// StockEarningsRequest represents a request to the /stocks/earnings endpoint.
// It encapsulates parameters for symbol, report type, and date to be used in the request.
// This struct provides methods such as Report(), Symbol(), Date(), From(), To(), and Countback() to set these parameters respectively.
//
// Public Methods:
//   - Report(q string) *StockEarningsRequest: Sets the report type parameter for the request.
//   - Symbol(q string) *StockEarningsRequest: Sets the symbol parameter for the request.
//   - Date(q interface{}) *StockEarningsRequest: Sets the date parameter for the request.
//   - From(q interface{}) *StockEarningsRequest: Sets the 'from' date parameter for the request.
//   - To(q interface{}) *StockEarningsRequest: Sets the 'to' date parameter for the request.
//   - Countback(q int) *StockEarningsRequest: Sets the countback parameter for the request.
//   - Packed() (*models.StockEarningsResponse, error): Sends the StockEarningsRequest and returns the StockEarningsResponse.
//   - Get() ([]models.StockEarningsReport, error): Sends the StockEarningsRequest, unpacks the StockEarningsResponse, and returns a slice of StockEarningsReport.
type StockEarningsRequest struct {
	*baseRequest
	symbolParams        *parameters.SymbolParams
	stockEarningsParams *parameters.StockEarningsParams
	dateParams          *parameters.DateParams
}

// Report sets the report type parameter for the StockEarningsRequest.
// This method is used to specify which earnings report to be retrieved.
//
// Parameters:
//   - q: A string representing which report to be returned.
//
// Returns:
//   - *StockEarningsRequest: This method returns a pointer to the StockEarningsRequest instance it was called on. This allows for method chaining.
func (ser *StockEarningsRequest) Report(q string) *StockEarningsRequest {
	if ser == nil {
		return nil
	}
	err := ser.stockEarningsParams.SetReport(q)
	if err != nil {
		ser.Error = err
	}
	return ser
}

// Symbol sets the symbol parameter for the StockEarningsRequest.
// This method is used to specify the stock symbol for which earnings data is requested.
//
// Parameters:
//   - q: A string representing the stock symbol to be set.
//
// Returns:
//   - *StockEarningsRequest: This method returns a pointer to the StockEarningsRequest instance it was called on. This allows for method chaining.
func (ser *StockEarningsRequest) Symbol(q string) *StockEarningsRequest {
	if ser == nil {
		return nil
	}
	err := ser.symbolParams.SetSymbol(q)
	if err != nil {
		ser.Error = err
	}
	return ser
}

// Date sets the date parameter for the StockEarningsRequest.
// This method is used to specify the date for which the stock earnings data is requested.
//
// Parameters:
//   - q: An interface{} representing the date to be set.
//
// Returns:
//   - *StockEarningsRequest: This method returns a pointer to the StockEarningsRequest instance it was called on. This allows for method chaining.
func (ser *StockEarningsRequest) Date(q interface{}) *StockEarningsRequest {
	err := ser.dateParams.SetDate(q)
	if err != nil {
		ser.Error = err
	}
	return ser
}

// From sets the 'from' date parameter for the StockEarningsRequest.
// This method is used to specify the starting point of the date range for which the stock earnings data is requested.
//
// Parameters:
//   - q: An interface{} representing the starting date.
//
// Returns:
//   - *StockEarningsRequest: This method returns a pointer to the StockEarningsRequest instance it was called on. This allows for method chaining.
func (ser *StockEarningsRequest) From(q interface{}) *StockEarningsRequest {
	err := ser.dateParams.SetFrom(q)
	if err != nil {
		ser.Error = err
	}
	return ser
}

// To sets the 'to' date parameter for the StockEarningsRequest.
// This method is used to specify the ending point of the date range for which the stock earnings data is requested.
//
// Parameters:
//   - q: An interface{} representing the ending date.
//
// Returns:
//   - *StockEarningsRequest: This method returns a pointer to the StockEarningsRequest instance it was called on. This allows for method chaining.
func (ser *StockEarningsRequest) To(q interface{}) *StockEarningsRequest {
	err := ser.dateParams.SetTo(q)
	if err != nil {
		ser.Error = err
	}
	return ser
}

// Countback sets the countback parameter for the StockEarningsRequest.
// This method specifies the number of periods to return, counting backwards from the 'to' date.
//
// Parameters:
//   - q: An int representing the number of periods to return.
//
// Returns:
//   - *StockEarningsRequest: This method returns a pointer to the StockEarningsRequest instance it was called on. This allows for method chaining.
func (ser *StockEarningsRequest) Countback(q int) *StockEarningsRequest {
	err := ser.dateParams.SetCountback(q)
	if err != nil {
		ser.Error = err
	}
	return ser
}

// GetParams packs the StockEarningsRequest struct into a slice of interface{} and returns it.
func (ser *StockEarningsRequest) getParams() ([]parameters.MarketDataParam, error) {
	if ser == nil {
		return nil, fmt.Errorf("StockEarningsRequest is nil")
	}
	params := []parameters.MarketDataParam{ser.symbolParams, ser.dateParams, ser.stockEarningsParams}
	return params, nil
}

// Packed sends the StockEarningsRequest and returns the StockEarningsResponse.
// An optional MarketDataClient can be passed to replace the client used in the request.
// Parameters:
//   - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// Returns:
//   - *models.StockEarningsResponse: A pointer to the StockEarningsResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (ser *StockEarningsRequest) Packed(optionalClients ...*MarketDataClient) (*models.StockEarningsResponse, error) {
	if ser == nil {
		return nil, fmt.Errorf("StockEarningsRequest is nil")
	}

	// Replace the client if an optional client is provided
	if len(optionalClients) > 0 && optionalClients[0] != nil {
		ser.baseRequest.client = optionalClients[0]
	}

	var serResp models.StockEarningsResponse
	_, err := ser.baseRequest.client.GetFromRequest(ser.baseRequest, &serResp)
	if err != nil {
		return nil, err
	}

	return &serResp, nil
}

// Get sends the StockEarningsRequest, unpacks the StockEarningsResponse, and returns a slice of StockEarningsReport.
// It returns an error if the request or unpacking fails.
// An optional MarketDataClient can be passed to replace the client used in the request.
// Parameters:
//   - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// Returns:
//   - []models.StockEarningsReport: A slice of StockEarningsReport containing the unpacked earnings data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (ser *StockEarningsRequest) Get(optionalClients ...*MarketDataClient) ([]models.StockEarningsReport, error) {
	if ser == nil {
		return nil, fmt.Errorf("StockEarningsRequest is nil")
	}

	// Use the Packed method to make the request, passing along any optional client
	serResp, err := ser.Packed(optionalClients...)
	if err != nil {
		return nil, err
	}

	// Unpack the data using the Unpack method in the response
	data, err := serResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// StockEarnings creates a new StockEarningsRequest and associates it with the provided client.
// If no client is provided, it uses the default client. This function initializes the request
// with default parameters for symbol, report type, and date, and sets the request path based on
// the predefined endpoints for stock earnings.
//
// Parameters:
//   - client: A variadic parameter that can accept zero or one MarketDataClient pointer. If no client is provided,
//     the default client is used.
//
// Returns:
//   - *StockEarningsRequest: A pointer to the newly created StockEarningsRequest with default parameters and associated client.
func StockEarnings(client ...*MarketDataClient) *StockEarningsRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[1]["stocks"]["earnings"]

	ser := &StockEarningsRequest{
		baseRequest:         baseReq,
		dateParams:          &parameters.DateParams{},
		symbolParams:        &parameters.SymbolParams{},
		stockEarningsParams: &parameters.StockEarningsParams{},
	}

	baseReq.child = ser

	return ser
}
