package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
	"github.com/go-resty/resty/v2"
)

type OptionsExpirationsRequest struct {
	*baseRequest
	underlyingSymbol *parameters.SymbolParams
	StrikeParam      float64
}

func (o *OptionsExpirationsRequest) Strike(strike float64) *OptionsExpirationsRequest {
	if strike < 0 || strike > 99999.999 {
		o.Error = fmt.Errorf("strike must be a positive number between 0 and 99999.999")
		return o
	}
	o.StrikeParam = strike
	return o
}

func (o *OptionsExpirationsRequest) UnderlyingSymbol(symbol string) *OptionsExpirationsRequest {
	if err := o.underlyingSymbol.SetSymbol(symbol); err != nil {
		o.Error = err
	}
	return o
}

// GetParams packs the OptionsExpirationsRequest struct into a slice of interface{} and returns it.
func (o *OptionsExpirationsRequest) getParams() ([]parameters.MarketDataParam, error) {
	if o == nil {
		return nil, fmt.Errorf("OptionsExpirationsRequest is nil")
	}
	params := []parameters.MarketDataParam{o.underlyingSymbol, o}
	return params, nil
}

// SetParams sets the parameters for the OptionsExpirationsRequest.
// If the parsing and setting of parameters fail, it returns an error.
func (o *OptionsExpirationsRequest) SetParams(request *resty.Request) error {
	return parameters.ParseAndSetParams(o, request)
}

// Get sends the OptionsExpirationsRequest and returns the OptionsExpirationsResponse along with the MarketDataResponse.
// It returns an error if the request fails.
func (o *OptionsExpirationsRequest) Get() (*models.OptionsExpirationsResponse, *resty.Response, error) {
	if o == nil {
		return nil, nil, fmt.Errorf("OptionsExpirationsRequest is nil")
	}
	var oResp models.OptionsExpirationsResponse
	mdr, err := o.baseRequest.client.GetFromRequest(o.baseRequest, &oResp)
	if err != nil {
		return nil, nil, err
	}

	return &oResp, mdr, nil
}

// OptionsExpirations creates a new OptionsExpirationsRequest and associates it with the provided client.
// If no client is provided, it uses the default client.
func OptionsExpirations(client ...*MarketDataClient) *OptionsExpirationsRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[1]["options"]["expirations"]

	oer := &OptionsExpirationsRequest{
		baseRequest:      baseReq,
		underlyingSymbol: &parameters.SymbolParams{},
	}

	baseReq.child = oer

	return oer
}
