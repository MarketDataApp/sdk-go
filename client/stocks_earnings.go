package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// StockQuoteRequest represents a request to the /stocks/quote endpoint.
type StockEarningsRequest struct {
	*baseRequest
	symbolParams        *parameters.SymbolParams
	stockEarningsParams *parameters.StockEarningsParams
	dateParams          *parameters.DateParams
}

// Report sets the report parameter for the StockEarningsRequest.
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

// Date sets the date parameter of the StockEarningsRequest.
func (ser *StockEarningsRequest) Date(q interface{}) *StockEarningsRequest {
	err := ser.dateParams.SetDate(q)
	if err != nil {
		ser.Error = err
	}
	return ser
}

// From sets the from parameter of the StockEarningsRequest.
func (ser *StockEarningsRequest) From(q interface{}) *StockEarningsRequest {
	err := ser.dateParams.SetFrom(q)
	if err != nil {
		ser.Error = err
	}
	return ser
}

// To sets the to parameter of the StockEarningsRequest.
func (ser *StockEarningsRequest) To(q interface{}) *StockEarningsRequest {
	err := ser.dateParams.SetTo(q)
	if err != nil {
		ser.Error = err
	}
	return ser
}

// Countback sets the countback parameter of the StockEarningsRequest.
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
// It returns an error if the request fails.
func (ser *StockEarningsRequest) Packed() (*models.StockEarningsResponse, error) {
	if ser == nil {
		return nil, fmt.Errorf("StockEarningsRequest is nil")
	}
	var serResp models.StockEarningsResponse
	_, err := ser.baseRequest.client.GetFromRequest(ser.baseRequest, &serResp)
	if err != nil {
		return nil, err
	}

	return &serResp, nil
}

// Get sends the StockEarningsRequest, unpacks the StockEarningsResponse and returns the data.
// It returns an error if the request or unpacking fails.
func (ser *StockEarningsRequest) Get() ([]models.StockEarningsReport, error) {
	if ser == nil {
		return nil, fmt.Errorf("StockEarningsRequest is nil")
	}
	
	// Use the Packed method to make the request
	serResp, err := ser.Packed()
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
// If no client is provided, it uses the default client.
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
