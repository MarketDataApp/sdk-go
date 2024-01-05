package client

import "fmt"

// IndexQuoteRequest represents a request to the /indices/quote endpoint.
type IndexQuoteRequest struct {
	*baseRequest
	symbolParams     *SymbolParams
	fiftyTwoWeekParams *FiftyTwoWeekParams
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
func (iqr *IndexQuoteRequest) getParams() ([]MarketDataParam, error) {
	if iqr == nil {
		return nil, fmt.Errorf("IndexQuoteRequest is nil")
	}
	params := []MarketDataParam{iqr.symbolParams, iqr.fiftyTwoWeekParams}
	return params, nil
}

// Get sends the IndexQuoteRequest and returns the IndexQuoteResponse along with the MarketDataResponse.
// It returns an error if the request fails.
func (iqr *IndexQuoteRequest) Get() (*IndexQuotesResponse, *MarketDataResponse, error) {
	if iqr == nil {
		return nil, nil, fmt.Errorf("IndexQuoteRequest is nil")
	}
	var iqrResp IndexQuotesResponse
	mdr, err := iqr.baseRequest.client.GetFromRequest(iqr.baseRequest, &iqrResp)
	if err != nil {
		return nil, nil, err
	}

	return &iqrResp, mdr, nil
}

// IndexQuote creates a new IndexQuoteRequest and associates it with the provided client.
// If no client is provided, it uses the default client.
func IndexQuotes(client ...*MarketDataClient) *IndexQuoteRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = Paths[1]["indices"]["quotes"]

	iqr := &IndexQuoteRequest{
		baseRequest:      baseReq,
		symbolParams:     &SymbolParams{},
		fiftyTwoWeekParams: &FiftyTwoWeekParams{},
	}

	baseReq.child = iqr

	return iqr
}
