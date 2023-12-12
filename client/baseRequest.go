package client

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

// BaseRequest is for internal use only by the Market Data Client package.
type baseRequest struct {
	request *resty.Request
	path    string
	client  *MarketDataClient
	child   any
	Error   error
}

func (br *baseRequest) getParams() ([]MarketDataParam, error) {
	if br == nil {
		return []MarketDataParam{}, nil
	}

	if msr, ok := br.child.(*MarketStatusRequest); ok {
		params, err := msr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if tr, ok := br.child.(*TickersRequest); ok {
		params, err := tr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	return []MarketDataParam{}, nil
}

// XtractPath returns the path of the BaseRequest.
// It returns an error if the BaseRequest is nil.
func (br *baseRequest) getPath() (string, error) {
	if br == nil {
		return "", fmt.Errorf("path is nil")
	}
	return br.path, nil
}

// XtractResty returns the resty.Request for the BaseRequest.
func (br *baseRequest) getResty() *resty.Request {
	return br.request
}

func newBaseRequest(clients ...*MarketDataClient) *baseRequest {
	var mdClient *MarketDataClient
	var err error

	if len(clients) > 0 {
		mdClient = clients[0]
	} else {
		mdClient, err = GetClient()
		if mdClient == nil {
			return nil
		}
	}

	baseReq := &baseRequest{
		request: mdClient.R(),
		client:  mdClient,
		Error:   err,
	}

	return baseReq
}

// XtractError returns the error of the BaseRequest.
func (br *baseRequest) getError() error {
	return br.Error
}
