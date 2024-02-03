package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/go-resty/resty/v2"
)

type MarketDataPacked interface {
	IsValid() bool
	Unpack() any
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
func (br *baseRequest) getParams() ([]parameters.MarketDataParam, error) {
	if br == nil || br.child == nil {
		return []parameters.MarketDataParam{}, nil
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

	if sqr, ok := br.child.(*StockQuoteRequest); ok {
		params, err := sqr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if snr, ok := br.child.(*StockNewsRequest); ok {
		params, err := snr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if iqr, ok := br.child.(*IndexQuoteRequest); ok {
		params, err := iqr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if ser, ok := br.child.(*StockEarningsRequest); ok {
		params, err := ser.getParams()
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

	if oer, ok := br.child.(*OptionsExpirationsRequest); ok {
		params, err := oer.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if olr, ok := br.child.(*OptionLookupRequest); ok {
		params, err := olr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if osr, ok := br.child.(*OptionsStrikesRequest); ok {
		params, err := osr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if oqr, ok := br.child.(*OptionQuoteRequest); ok {
		params, err := oqr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	if ocr, ok := br.child.(*OptionChainRequest); ok {
		params, err := ocr.getParams()
		if err != nil {
			return nil, err
		}
		return params, nil
	}

	return []parameters.MarketDataParam{}, nil
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

// Raw executes the request and returns the raw resty.Response.
// An optional MarketDataClient can be passed to replace the client used in the request.
func (br *baseRequest) Raw(optionalClients ...*MarketDataClient) (*resty.Response, error) {
	if br == nil {
		return nil, fmt.Errorf("baseRequest is nil")
	}

	// Replace the client if an optional client is provided
	if len(optionalClients) > 0 && optionalClients[0] != nil {
		br.client = optionalClients[0]
	}

	// Check if the client is nil after potentially replacing it
	if br.client == nil {
		return nil, fmt.Errorf("MarketDataClient is nil")
	}

	response, err := br.client.GetRawResponse(br)
	return response, err
}
