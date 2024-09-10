// Package client includes types and methods to access the Stock News endpoint.
// Retrieve news articles for any supported stock symbol.
//
// # Making Requests
//
// Use [StockNewsRequest] to make requests to the endpoint using any of the three supported execution methods:
//
//	| Method     | Execution     | Return Type                 | Description                                                                                                |
//	|------------|---------------|-----------------------------|------------------------------------------------------------------------------------------------------------|
//	| **Get**    | Direct        | `[]StockNews  `             | Directly returns a slice of `[]StockNews`, facilitating individual access to each news article.            |
//	| **Packed** | Intermediate  | `*StockNewsResponse`        | Returns a packed `*StockNewsResponse` object. Must be unpacked to access the `[]StockNews` slice.          |
//	| **Raw**    | Low-level     | `*resty.Response`           | Provides the raw `*resty.Response` for maximum flexibility. Direct access to raw JSON or `*http.Response`. |
package client

import (
	"context"
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
	"github.com/go-resty/resty/v2"
)

// StockNewsRequest represents a request to the [/v1/stocks/news/] endpoint.
// It encapsulates parameters for symbol, date, and additional news-specific parameters to be used in the request.
// This struct provides methods such as Symbol(), Date(), From(), To(), and Countback() to set these parameters respectively.
//
// Setter Methods
//
//   - Symbol(string) *StockNewsRequest: Sets the symbol parameter for the request.
//   - Date(interface{}) *StockNewsRequest: Sets the date parameter for the request.
//   - From(interface{}) *StockNewsRequest: Sets the 'from' date parameter for the request.
//   - To(interface{}) *StockNewsRequest: Sets the 'to' date parameter for the request.
//   - Countback(int) *StockNewsRequest: Sets the countback parameter for the request.
//
// # Execution Methods
//
// These methods are used to send the request in different formats or retrieve the data.
// They handle the actual communication with the API endpoint.
//
//   - Get() ([]StockNews, error): Initiates the request, processes the response, and provides an slice of `StockNews` objects for straightforward access to news articles.
//   - Packed() (*StockNewsResponse, error): Delivers a packed `StockNewsResponse` object containing slices of data that directly correspond to the JSON structure returned by the Market Data API.
//   - Raw() (*resty.Response, error): Executes the request in its raw form and retrieves the raw HTTP response for maximum flexibility.
//
// [/v1/stocks/news/]: https://www.marketdata.app/docs/api/stocks/news
type StockNewsRequest struct {
	*baseRequest
	symbolParams *parameters.SymbolParams
	dateParams   *parameters.DateParams
}

// Symbol sets the symbol parameter for the StockNewsRequest.
// This method is used to specify the stock symbol for which news data is requested.
//
// # Parameters
//
//   - string: A string representing the stock symbol to be set.
//
// # Returns
//
//   - *StockNewsRequest: This method returns a pointer to the StockNewsRequest instance it was called on. This allows for method chaining.
func (snr *StockNewsRequest) Symbol(q string) *StockNewsRequest {
	if snr == nil {
		return nil
	}
	err := snr.symbolParams.SetSymbol(q)
	if err != nil {
		snr.Error = err
	}
	return snr
}

// Date sets the date parameter for the StockNewsRequest.
// This method is used to specify the date for which the stock news data is requested.
//
// # Parameters
//
//   - interface{}: An interface{} representing the date to be set. It can be a string, a time.Time object, a Unix int, or any other type that the underlying dates package method can process.
//
// # Returns
//
//   - *StockNewsRequest: This method returns a pointer to the StockNewsRequest instance it was called on. This allows for method chaining.
func (snr *StockNewsRequest) Date(q interface{}) *StockNewsRequest {
	if snr == nil {
		return nil
	}
	err := snr.dateParams.SetDate(q)
	if err != nil {
		snr.Error = err
	}
	return snr
}

// From sets the 'from' date parameter for the StockNewsRequest.
// This method is used to specify the starting point of the date range for which the stock news data is requested.
//
// # Parameters
//
//   - interface{}: An interface{} representing the date to be set. It can be a string, a time.Time object, a Unix int, or any other type that the underlying dates package method can process.
//
// # Returns
//
//   - *StockNewsRequest: This method returns a pointer to the StockNewsRequest instance it was called on. This allows for method chaining.
func (snr *StockNewsRequest) From(q interface{}) *StockNewsRequest {
	if snr == nil {
		return nil
	}
	err := snr.dateParams.SetFrom(q)
	if err != nil {
		snr.Error = err
	}
	return snr
}

// To sets the 'to' date parameter for the StockNewsRequest.
// This method is used to specify the ending point of the date range for which the stock news data is requested.
//
// # Parameters
//
//   - interface{}: An interface{} representing the date to be set. It can be a string, a time.Time object, a Unix int, or any other type that the underlying dates package method can process.
//
// # Returns
//
//   - *StockNewsRequest: This method returns a pointer to the StockNewsRequest instance it was called on. This allows for method chaining.
func (snr *StockNewsRequest) To(q interface{}) *StockNewsRequest {
	if snr == nil {
		return nil
	}
	err := snr.dateParams.SetTo(q)
	if err != nil {
		snr.Error = err
	}
	return snr
}

// Countback sets the countback parameter for the StockNewsRequest.
// This method specifies the number of news items to return, counting backwards from the 'to' date.
//
// # Parameters
//
//   - int: The number of news items to return.
//
// # Returns
//
//   - *StockNewsRequest: This method returns a pointer to the *StockNewsRequest instance it was called on. This allows for method chaining.
func (snr *StockNewsRequest) Countback(q int) *StockNewsRequest {
	if snr == nil {
		return nil
	}
	err := snr.dateParams.SetCountback(q)
	if err != nil {
		snr.Error = err
	}
	return snr
}

// getParams packs the StockNewsRequest struct into a slice of interface{} and returns it.
func (snr *StockNewsRequest) getParams() ([]parameters.MarketDataParam, error) {
	if snr == nil {
		return nil, fmt.Errorf("StockNewsRequest is nil")
	}
	params := []parameters.MarketDataParam{snr.symbolParams, snr.dateParams}
	return params, nil
}

// Raw executes the StockNewsRequest with the provided context and returns the raw *resty.Response.
// This method retrieves the raw HTTP response for further processing.
//
// # Parameters
//
//   - ctx context.Context: The context to use for the request execution.
//
// # Returns
//
//   - *resty.Response: The raw HTTP response from the executed request.
//   - error: An error object if the request fails due to being nil or other execution errors.
func (snr *StockNewsRequest) Raw(ctx context.Context) (*resty.Response, error) {
	return snr.baseRequest.Raw(ctx)
}

// Packed sends the StockNewsRequest with the provided context and returns the StockNewsResponse.
//
// # Parameters
//
//   - ctx context.Context: The context to use for the request execution.
//
// # Returns
//
//   - *models.StockNewsResponse: A pointer to the StockNewsResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (snr *StockNewsRequest) Packed(ctx context.Context) (*models.StockNewsResponse, error) {
	if snr == nil {
		return nil, fmt.Errorf("StockNewsRequest is nil")
	}

	var snrResp models.StockNewsResponse
	_, err := snr.baseRequest.client.getFromRequest(ctx, snr.baseRequest, &snrResp)
	if err != nil {
		return nil, err
	}

	return &snrResp, nil
}

// Get sends the StockNewsRequest with the provided context, unpacks the StockNewsResponse, and returns a slice of StockNews.
// It returns an error if the request or unpacking fails.
//
// # Parameters
//
//   - ctx context.Context: The context to use for the request execution.
//
// # Returns
//
//   - []models.StockNews: A slice of StockNews containing the unpacked news data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (snr *StockNewsRequest) Get(ctx context.Context) ([]models.StockNews, error) {
	if snr == nil {
		return nil, fmt.Errorf("StockNewsRequest is nil")
	}

	// Use the Packed method to make the request
	snrResp, err := snr.Packed(ctx)
	if err != nil {
		return nil, err
	}

	// Unpack the data using the Unpack method in the response
	data, err := snrResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// StockNews creates a new StockNewsRequest and associates it with the default client. This function initializes the request
// with default parameters for symbol, date, and additional news-specific parameters, and sets the request path based on
// the predefined endpoints for stock news.
//
// # Returns
//
//   - *StockNewsRequest: A pointer to the newly created *StockNewsRequest with default parameters and associated client.
func StockNews() *StockNewsRequest {
	baseReq := newBaseRequest()
	baseReq.path = endpoints[1]["stocks"]["news"]

	snr := &StockNewsRequest{
		baseRequest:  baseReq,
		symbolParams: &parameters.SymbolParams{},
		dateParams:   &parameters.DateParams{},
	}

	baseReq.child = snr

	return snr
}
