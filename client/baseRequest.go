package client

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

type MarketDataPacked interface {
	IsValid() bool
	Unpack() any
}

type MarketDataParam interface {
	SetParams(*resty.Request) error
}

// baseRequest is a struct that represents a basic request in the Market Data Client package.
// It contains a request of type *resty.Request, a path of type string, a client of type *MarketDataClient,
// a child of type any, and an Error of type error.
type baseRequest struct {
	request *resty.Request
	path    string
	client  *MarketDataClient
	child   any
	Error   error
}

// getParams calls the getParams method of the appropriate MarketDataRequest.
func (br *baseRequest) getParams() ([]MarketDataParam, error) {
	if br == nil || br.child == nil {
		return []MarketDataParam{}, nil
	}

	// Check if child is of type *baseRequest
	if _, ok := br.child.(*baseRequest); ok {
		return nil, fmt.Errorf("child is of type *baseRequest, stopping recursion")
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

	if scr, ok := br.child.(*StockCandlesRequest); ok {
		params, err := scr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if scr, ok := br.child.(*StockCandlesRequestV2); ok {
		params, err := scr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if icr, ok := br.child.(*IndicesCandlesRequest); ok {
		params, err := icr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	return []MarketDataParam{}, nil
}

// getPath returns the path of the BaseRequest.
// It returns an error if the BaseRequest is nil.
func (br *baseRequest) getPath() (string, error) {
	if br == nil {
		return "", fmt.Errorf("path is nil")
	}
	return br.path, nil
}

// getResty returns the resty.Request for the BaseRequest.
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

// getError returns the error of the BaseRequest.
func (br *baseRequest) getError() error {
	return br.Error
}
