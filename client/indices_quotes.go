package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// IndexQuoteRequest represents a request to the /indices/quote endpoint.
type IndexQuoteRequest struct {
	*baseRequest
	symbolParams       *parameters.SymbolParams
	fiftyTwoWeekParams *parameters.FiftyTwoWeekParams
}

// Symbol sets the symbol parameter for the IndexQuoteRequest.
func (iqr *IndexQuoteRequest) Symbol(q string) *IndexQuoteRequest {
	if iqr == nil {
		return nil
	}
	err := iqr.symbolParams.SetSymbol(q)
	if err != nil {
		iqr.Error = err
	}
	return iqr
}

// FiftyTwoWeek sets the FiftyTwoWeek parameter for the IndexQuoteRequest.
func (iqr *IndexQuoteRequest) FiftyTwoWeek(q bool) *IndexQuoteRequest {
	if iqr == nil {
		return nil
	}
	iqr.fiftyTwoWeekParams.SetFiftyTwoWeek(q)
	return iqr
}

// GetParams packs the IndexQuoteRequest struct into a slice of interface{} and returns it.
func (iqr *IndexQuoteRequest) getParams() ([]parameters.MarketDataParam, error) {
	if iqr == nil {
		return nil, fmt.Errorf("IndexQuoteRequest is nil")
	}
	params := []parameters.MarketDataParam{iqr.symbolParams, iqr.fiftyTwoWeekParams}
	return params, nil
}

// Get sends the IndexQuoteRequest and returns the IndexQuoteResponse.
// It returns an error if the request fails.
func (iqr *IndexQuoteRequest) Packed() (*models.IndexQuotesResponse, error) {
	if iqr == nil {
		return nil, fmt.Errorf("IndexQuoteRequest is nil")
	}
	var iqrResp models.IndexQuotesResponse
	_, err := iqr.baseRequest.client.GetFromRequest(iqr.baseRequest, &iqrResp)
	if err != nil {
		return nil, err
	}

	return &iqrResp, nil
}

// Get sends the IndexQuoteRequest, unpacks the IndexQuotesResponse and returns a slice of IndexQuote.
// It returns an error if the request or unpacking fails.
func (iqr *IndexQuoteRequest) Get() ([]models.IndexQuote, error) {
	if iqr == nil {
		return nil, fmt.Errorf("IndexQuoteRequest is nil")
	}
	
	// Use the Packed method to make the request
	iqrResp, err := iqr.Packed()
	if err != nil {
		return nil, err
	}

	// Unpack the data using the Unpack method in the response
	data, err := iqrResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
}

// IndexQuote creates a new IndexQuoteRequest and associates it with the provided client.
// If no client is provided, it uses the default client.
func IndexQuotes(client ...*MarketDataClient) *IndexQuoteRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = endpoints[1]["indices"]["quotes"]

	iqr := &IndexQuoteRequest{
		baseRequest:        baseReq,
		symbolParams:       &parameters.SymbolParams{},
		fiftyTwoWeekParams: &parameters.FiftyTwoWeekParams{},
	}

	baseReq.child = iqr

	return iqr
}
