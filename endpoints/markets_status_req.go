package endpoints

import (
	"fmt"

	md "github.com/MarketDataApp/sdk-go/client"
	"github.com/go-resty/resty/v2"
)

const (
	MarketStatusPath = "/v1/markets/status/"
)

// MarketStatusRequest represents a request for market status.
type MarketStatusRequest struct {
	*resty.Request
	Client          *md.MarketDataClient
	UniversalParams *UniversalParams
	DateParams      *DateParams
	Path            string
	ParamCountry    string `query:"country"`
	Error           error
}

// GetResty returns the resty.Request for the TickersRequest.
func (msr *MarketStatusRequest) GetResty() *resty.Request {
	return msr.Request
}

func (msr *MarketStatusRequest) SetParams(request *resty.Request) error {
	return md.ParseAndSetParams(msr, request)
}

// GetParams returns a slice of interface containing the MarketStatusRequest, UniversalParams and DateParams structs.
func (msr *MarketStatusRequest) GetParams() []md.MarketDataParam {
	params := []md.MarketDataParam{msr, msr.UniversalParams, msr.DateParams}
	return params
}

// GetPath returns the path of the MarketStatusRequest.
func (msr *MarketStatusRequest) GetPath() string {
	if msr == nil {
		msr.Error = fmt.Errorf("MarketStatusRequest is nil")
		return ""
	}
	return msr.Path
}

// GetError returns the error of the MarketStatusRequest.
func (msr *MarketStatusRequest) GetError() error {
	if msr == nil || msr.Error == nil {
		return nil
	}
	return msr.Error
}

// Country sets the country parameter of the MarketStatusRequest.
func (msr *MarketStatusRequest) Country(q string) *MarketStatusRequest {
	if len(q) != 2 || !IsAlpha(q) {
		msr.Error = fmt.Errorf("invalid country code")
		return msr
	}
	msr.ParamCountry = q
	return msr
}

// Date sets the date parameter of the MarketStatusRequest.
func (msr *MarketStatusRequest) Date(q interface{}) *MarketStatusRequest {
	err := msr.DateParams.SetDate(q)
	if err != nil {
		msr.Error = err
	}
	return msr
}

// From sets the from parameter of the MarketStatusRequest.
func (msr *MarketStatusRequest) From(q interface{}) *MarketStatusRequest {
	err := msr.DateParams.SetFrom(q)
	if err != nil {
		msr.Error = err
	}
	return msr
}

// To sets the to parameter of the MarketStatusRequest.
func (msr *MarketStatusRequest) To(q interface{}) *MarketStatusRequest {
	err := msr.DateParams.SetTo(q)
	if err != nil {
		msr.Error = err
	}
	return msr
}

// Countback sets the countback parameter of the MarketStatusRequest.
func (msr *MarketStatusRequest) Countback(q int) *MarketStatusRequest {
	err := msr.DateParams.SetCountback(q)
	if err != nil {
		msr.Error = err
	}
	return msr
}

// GetMarketStatus sends the MarketStatusRequest and returns the response.
func (msr *MarketStatusRequest) Get() (*MarketStatusResponse, error) {
	var msrResp MarketStatusResponse
	mdr, err := msr.Client.GetFromRequest(msr, &msrResp)
	if err != nil {
		return nil, err
	}
	msrResp.MarketDataResponse = mdr

	return &msrResp, nil
}

// New creates a new MarketStatusRequest.
func MarketStatus(clients ...*md.MarketDataClient) *MarketStatusRequest {
	var mdClient *md.MarketDataClient
	var err error

	if len(clients) > 0 {
		mdClient = clients[0]
	} else {
		mdClient, err = md.GetClient()
		if err != nil {
			return &MarketStatusRequest{Error: err}
		}
	}

	msr := &MarketStatusRequest{
		UniversalParams: &UniversalParams{},
		DateParams:      &DateParams{},
		Path:            MarketStatusPath,
		Client:          mdClient,
		Request:         mdClient.R(),
	}

	msr.Country("US") // Set default country value to "US"

	return msr
}
