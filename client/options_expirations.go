package client

import (
	"fmt"
	"time"

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

// Packed sends the OptionsExpirationsRequest and returns the OptionsExpirationsResponse.
// It returns an error if the request fails.
func (o *OptionsExpirationsRequest) Packed() (*models.OptionsExpirationsResponse, error) {
	if o == nil {
		return nil, fmt.Errorf("OptionsExpirationsRequest is nil")
	}
	var oResp models.OptionsExpirationsResponse
	_, err := o.baseRequest.client.GetFromRequest(o.baseRequest, &oResp)
	if err != nil {
		return nil, err
	}

	return &oResp, nil
}

// Get sends the OptionsExpirationsRequest, unpacks the OptionsExpirationsResponse and returns a slice of time.Time.
// It returns an error if the request or unpacking fails.
func (o *OptionsExpirationsRequest) Get() ([]time.Time, error) {
	if o == nil {
		return nil, fmt.Errorf("OptionsExpirationsRequest is nil")
	}

	// Use the Packed method to make the request
	oResp, err := o.Packed()
	if err != nil {
		return nil, err
	}

	// Unpack the data using the Unpack method in the response
	data, err := oResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
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
