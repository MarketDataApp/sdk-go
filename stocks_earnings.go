// Package client includes types and methods to access the Stock Earnings endpoint.
// Retrieve earnings data for any supported stock symbol.
//
// # Making Requests
//
// Use [StockEarningsRequest] to make requests to the endpoint using any of the three supported execution methods:
//
//	| Method     | Execution     | Return Type                 | Description                                                                                                     |
//	|------------|---------------|-----------------------------|-----------------------------------------------------------------------------------------------------------------|
//	| **Get**    | Direct        | `[]StockEarningsReport`     | Directly returns a slice of `[]StockEarningsReport`, facilitating individual access to each earnings report.    |
//	| **Packed** | Intermediate  | `*StockEarningsResponse`    | Returns a packed `*StockEarningsResponse` object. Must be unpacked to access the `[]StockEarningsReport` slice. |
//	| **Raw**    | Low-level     | `*resty.Response`           | Provides the raw `*resty.Response` for maximum flexibility. Direct access to raw JSON or `*http.Response`.      |
package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
	"github.com/go-resty/resty/v2"
)

// StockEarningsRequest represents a request to the [/v1/stocks/earnings/] endpoint.
// It encapsulates parameters for symbol, report type, and date to be used in the request.
// This struct provides methods such as Report(), Symbol(), Date(), From(), To(), and Countback() to set these parameters respectively.
//
// # Generated By
//
//   - StockEarnings() *StockEarningsRequest: StockEarnings creates a new *StockEarningsRequest and returns a pointer to the request allowing for method chaining.
//
// # Setter Methods
//
//   - Report(q string) *StockEarningsRequest: Sets the report type parameter for the request.
//   - Symbol(q string) *StockEarningsRequest: Sets the symbol parameter for the request.
//   - Date(q interface{}) *StockEarningsRequest: Sets the date parameter for the request.
//   - From(q interface{}) *StockEarningsRequest: Sets the 'from' date parameter for the request.
//   - To(q interface{}) *StockEarningsRequest: Sets the 'to' date parameter for the request.
//   - Countback(q int) *StockEarningsRequest: Sets the countback parameter for the request.
//
// # Execution Methods
//
// These methods are used to send the request in different formats or retrieve the data.
// They handle the actual communication with the API endpoint.
//
//   - Get() ([]StockEarningsReport, error): Sends the request, unpacks the response, and returns the data in a user-friendly format.
//   - Packed() (*StockEarningsResponse, error): Returns a struct that contains equal-length slices of primitives. This packed response mirrors Market Data's JSON response.
//   - Raw() (*resty.Response, error): Sends the request as is and returns the raw HTTP response.
//
// [/v1/stocks/earnings/]: https://www.marketdata.app/docs/api/stocks/earnings
type StockEarningsRequest struct {
	*baseRequest
	symbolParams        *parameters.SymbolParams
	stockEarningsParams *parameters.StockEarningsParams
	dateParams          *parameters.DateParams
}

// Report sets the report type parameter for the StockEarningsRequest.
// This method is used to specify which earnings report to be retrieved.
//
// # Parameters
//
//   - string: A string representing which report to be returned.
//
// # Returns
//
//   - *StockEarningsRequest: This method returns a pointer to the *StockEarningsRequest instance it was called on. This allows for method chaining.
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
// # Parameters
//
//   - string: A string representing the stock symbol to be set.
//
// # Returns
//
//   - *StockEarningsRequest: This method returns a pointer to the *StockEarningsRequest instance it was called on. This allows for method chaining.
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
// # Parameters
//
//   - interface{}: An interface{} representing the date to be set. It can be a string, a time.Time object, a Unix int, or any other type that the underlying dates package method can process.
//
// # Returns
//
//   - *StockEarningsRequest: This method returns a pointer to the *StockEarningsRequest instance it was called on. This allows for method chaining.
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
// # Parameters
//
//   - interface{}: An interface{} representing the date to be set. It can be a string, a time.Time object, a Unix int, or any other type that the underlying dates package method can process.
//
// # Returns
//
//   - *StockEarningsRequest: This method returns a pointer to the *StockEarningsRequest instance it was called on. This allows for method chaining.
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
// # Parameters
//
//   - interface{}: An interface{} representing the date to be set. It can be a string, a time.Time object, a Unix int, or any other type that the underlying dates package method can process.
//
// # Returns
//
//   - *StockEarningsRequest: This method returns a pointer to the *StockEarningsRequest instance it was called on. This allows for method chaining.
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
// # Parameters
//
//   - int: The number of periods to return.
//
// # Returns
//
//   - *StockEarningsRequest: This method returns a pointer to the *StockEarningsRequest instance it was called on. This allows for method chaining.
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

// Raw executes the StockEarningsRequest and returns the raw *resty.Response.
// The *resty.Response can be used to access the raw JSON or *http.Response directly.
//
// # Returns
//
//   - *resty.Response: The raw response from the executed StockEarningsRequest.
//   - error: An error object if the StockEarningsRequest is nil, the MarketDataClient is nil, or if an error occurs during the request execution.
func (ser *StockEarningsRequest) Raw() (*resty.Response, error) {
	return ser.baseRequest.Raw()
}

// Packed sends the StockEarningsRequest and returns the StockEarningsResponse.
//
// # Returns
//
//   - *models.StockEarningsResponse: A pointer to the StockEarningsResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (ser *StockEarningsRequest) Packed() (*models.StockEarningsResponse, error) {
	if ser == nil {
		return nil, fmt.Errorf("StockEarningsRequest is nil")
	}

	var serResp models.StockEarningsResponse
	_, err := ser.baseRequest.client.getFromRequest(ser.baseRequest, &serResp)
	if err != nil {
		return nil, err
	}

	return &serResp, nil
}

// Get sends the StockEarningsRequest, unpacks the StockEarningsResponse, and returns a slice of StockEarningsReport.
// It returns an error if the request or unpacking fails.
//
// # Returns
//
//   - []models.StockEarningsReport: A slice of StockEarningsReport containing the unpacked earnings data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (ser *StockEarningsRequest) Get() ([]models.StockEarningsReport, error) {
	if ser == nil {
		return nil, fmt.Errorf("StockEarningsRequest is nil")
	}

	serResp, err := ser.Packed()
	if err != nil {
		return nil, err
	}

	data, err := serResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// StockEarnings creates a new StockEarningsRequest with default parameters and associates it with the default client.
// This function initializes the request with default parameters for symbol, report type, and date, and sets the request path based on
// the predefined endpoints for stock earnings.
//
// # Returns
//
//   - *StockEarningsRequest: A pointer to the newly created StockEarningsRequest with default parameters and associated client.
func StockEarnings() *StockEarningsRequest {
	baseReq := newBaseRequest()
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
