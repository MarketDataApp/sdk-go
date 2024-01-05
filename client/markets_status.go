package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// MarketStatusRequest represents a request for market status.
type MarketStatusRequest struct {
	*baseRequest
	countryParams   *parameters.CountryParams
	universalParams *parameters.UniversalParams
	dateParams      *parameters.DateParams
}

// GetParams returns a slice of interface containing the MarketStatusRequest, UniversalParams and DateParams structs.
// It returns an error if one of these 3 structs don't exist.
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
func (msr *MarketStatusRequest) Country(q string) *MarketStatusRequest {
	err := msr.countryParams.SetCountry(q)
	if err != nil {
		msr.baseRequest.Error = err
	}
	return msr
}

// Date sets the date parameter of the MarketStatusRequest.
func (msr *MarketStatusRequest) Date(q interface{}) *MarketStatusRequest {
	err := msr.dateParams.SetDate(q)
	if err != nil {
		msr.baseRequest.Error = err
	}
	return msr
}

// From sets the from parameter of the MarketStatusRequest.
func (msr *MarketStatusRequest) From(q interface{}) *MarketStatusRequest {
	err := msr.dateParams.SetFrom(q)
	if err != nil {
		msr.baseRequest.Error = err
	}
	return msr
}

// To sets the to parameter of the MarketStatusRequest.
func (msr *MarketStatusRequest) To(q interface{}) *MarketStatusRequest {
	err := msr.dateParams.SetTo(q)
	if err != nil {
		msr.baseRequest.Error = err
	}
	return msr
}

// Countback sets the countback parameter of the MarketStatusRequest.
func (msr *MarketStatusRequest) Countback(q int) *MarketStatusRequest {
	err := msr.dateParams.SetCountback(q)
	if err != nil {
		msr.baseRequest.Error = err
	}
	return msr
}

// GetMarketStatus sends the MarketStatusRequest and returns the response.
func (msr *MarketStatusRequest) Get() (*models.MarketStatusResponse, *MarketDataResponse, error) {
	var msrResp models.MarketStatusResponse
	mdr, err := msr.baseRequest.client.GetFromRequest(msr.baseRequest, &msrResp)
	if err != nil {
		return nil, nil, err
	}

	return &msrResp, mdr, nil
}

// New creates a new MarketStatusRequest.
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
