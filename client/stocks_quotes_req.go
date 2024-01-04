package client

import "fmt"

// StockQuoteRequest represents a request to the /stocks/quote endpoint.
type StockQuoteRequest struct {
	*baseRequest
	symbolParams     *SymbolParams
	stockQuoteParams *StockQuoteParams
}

// Symbol sets the symbol parameter for the IndicesCandlesRequest.
func (sqr *StockQuoteRequest) Symbol(q string) *StockQuoteRequest {
	if sqr == nil {
		return nil
	}
	err := sqr.symbolParams.SetSymbol(q)
	if err != nil {
		sqr.Error = err
	}
	return sqr
}

// FiftyTwoWeek sets the FiftyTwoWeek parameter for the StockQuoteRequest.
func (sqr *StockQuoteRequest) FiftyTwoWeek(q bool) *StockQuoteRequest {
	if sqr == nil {
		return nil
	}
	sqr.stockQuoteParams.SetFiftyTwoWeek(q)
	return sqr
}

// GetParams packs the StockQuoteRequest struct into a slice of interface{} and returns it.
func (sqr *StockQuoteRequest) getParams() ([]MarketDataParam, error) {
	if sqr == nil {
		return nil, fmt.Errorf("StockQuoteRequest is nil")
	}
	params := []MarketDataParam{sqr.symbolParams, sqr.stockQuoteParams}
	return params, nil
}

// Get sends the StockQuoteRequest and returns the StockQuoteResponse along with the MarketDataResponse.
// It returns an error if the request fails.
func (sqr *StockQuoteRequest) Get() (*StockQuotesResponse, *MarketDataResponse, error) {
	if sqr == nil {
		return nil, nil, fmt.Errorf("StockQuoteRequest is nil")
	}
	var sqrResp StockQuotesResponse
	mdr, err := sqr.baseRequest.client.GetFromRequest(sqr.baseRequest, &sqrResp)
	if err != nil {
		return nil, nil, err
	}

	return &sqrResp, mdr, nil
}

// StockQuote creates a new StockQuoteRequest and associates it with the provided client.
// If no client is provided, it uses the default client.
func StockQuotes(client ...*MarketDataClient) *StockQuoteRequest {
	baseReq := newBaseRequest(client...)
	baseReq.path = Paths[1]["stocks"]["quotes"]

	sqr := &StockQuoteRequest{
		baseRequest:      baseReq,
		symbolParams:     &SymbolParams{},
		stockQuoteParams: &StockQuoteParams{},
	}

	baseReq.child = sqr

	return sqr
}
