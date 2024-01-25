package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// StockNewsRequest represents a request to the /stocks/news endpoint.
type StockNewsRequest struct {
	*baseRequest
	symbolParams *parameters.SymbolParams
	dateParams   *parameters.DateParams
}

// Symbol sets the symbol parameter for the StockNewsRequest.
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

// From sets the from parameter for the StockNewsRequest.
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

// To sets the to parameter for the StockNewsRequest.
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

// GetParams packs the StockNewsRequest struct into a slice of interface{} and returns it.
func (snr *StockNewsRequest) getParams() ([]parameters.MarketDataParam, error) {
	if snr == nil {
		return nil, fmt.Errorf("StockNewsRequest is nil")
	}
	params := []parameters.MarketDataParam{snr.symbolParams, snr.dateParams}
	return params, nil
}

// Packed sends the StockNewsRequest and returns the StockNewsResponse.
// It returns an error if the request fails.
func (snr *StockNewsRequest) Packed() (*models.StockNewsResponse, error) {
	if snr == nil {
		return nil, fmt.Errorf("StockNewsRequest is nil")
	}
	var snrResp models.StockNewsResponse
	_, err := snr.baseRequest.client.GetFromRequest(snr.baseRequest, &snrResp)
	if err != nil {
		return nil, err
	}

	return &snrResp, nil
}

// Get sends the StockNewsRequest, unpacks the StockNewsResponse and returns the data.
// It returns an error if the request or unpacking fails.
func (snr *StockNewsRequest) Get() ([]models.StockNews, error) {
	if snr == nil {
		return nil, fmt.Errorf("StockNewsRequest is nil")
	}

	// Use the Packed method to make the request
	snrResp, err := snr.Packed()
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
// If no client is provided, it uses the default client.
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
