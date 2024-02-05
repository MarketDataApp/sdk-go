package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// StockNewsRequest represents a request to the /stocks/news endpoint.
// It encapsulates parameters for symbol, date, and additional news-specific parameters to be used in the request.
// This struct provides methods such as Symbol(), Date(), From(), To(), and Countback() to set these parameters respectively.
//
// Public Methods:
//   - Symbol(q string) *StockNewsRequest: Sets the symbol parameter for the request.
//   - Date(q interface{}) *StockNewsRequest: Sets the date parameter for the request.
//   - From(q interface{}) *StockNewsRequest: Sets the 'from' date parameter for the request.
//   - To(q interface{}) *StockNewsRequest: Sets the 'to' date parameter for the request.
//   - Countback(q int) *StockNewsRequest: Sets the countback parameter for the request.
type StockNewsRequest struct {
	*baseRequest
	symbolParams *parameters.SymbolParams
	dateParams   *parameters.DateParams
}

// Symbol sets the symbol parameter for the StockNewsRequest.
// This method is used to specify the stock symbol for which news data is requested.
//
// Parameters:
//   - q: A string representing the stock symbol to be set.
//
// Returns:
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
// Parameters:
//   - q: An interface{} representing the date to be set.
//
// Returns:
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
// Parameters:
//   - q: An interface{} representing the starting date.
//
// Returns:
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
// Parameters:
//   - q: An interface{} representing the ending date.
//
// Returns:
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
// Parameters:
//   - q: An int representing the number of news items to return.
//
// Returns:
//   - *StockNewsRequest: This method returns a pointer to the StockNewsRequest instance it was called on. This allows for method chaining.
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

// Packed sends the StockNewsRequest and returns the StockNewsResponse.
// An optional MarketDataClient can be passed to replace the client used in the request.
// Parameters:
//   - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// Returns:
//   - *models.StockNewsResponse: A pointer to the StockNewsResponse obtained from the request.
//   - error: An error object that indicates a failure in sending the request.
func (snr *StockNewsRequest) Packed(optionalClients ...*MarketDataClient) (*models.StockNewsResponse, error) {
	if snr == nil {
		return nil, fmt.Errorf("StockNewsRequest is nil")
	}

	// Replace the client if an optional client is provided
	if len(optionalClients) > 0 && optionalClients[0] != nil {
		snr.baseRequest.client = optionalClients[0]
	}

	var snrResp models.StockNewsResponse
	_, err := snr.baseRequest.client.GetFromRequest(snr.baseRequest, &snrResp)
	if err != nil {
		return nil, err
	}

	return &snrResp, nil
}

// Get sends the StockNewsRequest, unpacks the StockNewsResponse, and returns a slice of StockNews.
// It returns an error if the request or unpacking fails.
// An optional MarketDataClient can be passed to replace the client used in the request.
// Parameters:
//   - optionalClients: A variadic parameter that can accept zero or one MarketDataClient pointer. If a client is provided,
//     it replaces the current client for this request.
//
// Returns:
//   - []models.StockNews: A slice of StockNews containing the unpacked news data from the response.
//   - error: An error object that indicates a failure in sending the request or unpacking the response.
func (snr *StockNewsRequest) Get(optionalClients ...*MarketDataClient) ([]models.StockNews, error) {
	if snr == nil {
		return nil, fmt.Errorf("StockNewsRequest is nil")
	}

	// Use the Packed method to make the request, passing along any optional client
	snrResp, err := snr.Packed(optionalClients...)
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

// StockNews creates a new StockNewsRequest and associates it with the provided client.
// If no client is provided, it uses the default client. This function initializes the request
// with default parameters for symbol, date, and additional news-specific parameters, and sets the request path based on
// the predefined endpoints for stock news.
//
// Parameters:
//   - client: A variadic parameter that can accept zero or one MarketDataClient pointer. If no client is provided,
//     the default client is used.
//
// Returns:
//   - *StockNewsRequest: A pointer to the newly created StockNewsRequest with default parameters and associated client.
func StockNews(client ...*MarketDataClient) *StockNewsRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[1]["stocks"]["news"]

	snr := &StockNewsRequest{
		baseRequest:  baseReq,
		symbolParams: &parameters.SymbolParams{},
		dateParams:   &parameters.DateParams{},
	}

	baseReq.child = snr

	return snr
}
