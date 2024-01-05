package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/parameters"
	"github.com/MarketDataApp/sdk-go/models"
)

// StockQuoteRequest represents a request to the /stocks/quote endpoint.
type StockQuoteRequest struct {
	*baseRequest
	symbolParams       *parameters.SymbolParams
	fiftyTwoWeekParams *parameters.FiftyTwoWeekParams
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
	sqr.fiftyTwoWeekParams.SetFiftyTwoWeek(q)
	return sqr
}

// GetParams packs the StockQuoteRequest struct into a slice of interface{} and returns it.
func (sqr *StockQuoteRequest) getParams() ([]parameters.MarketDataParam, error) {
	if sqr == nil {
		return nil, fmt.Errorf("StockQuoteRequest is nil")
	}
	params := []parameters.MarketDataParam{sqr.symbolParams, sqr.fiftyTwoWeekParams}
	return params, nil
}

// Get sends the StockQuoteRequest and returns the StockQuoteResponse along with the MarketDataResponse.
// It returns an error if the request fails.
func (sqr *StockQuoteRequest) Get() (*models.StockQuotesResponse, *MarketDataResponse, error) {
	if sqr == nil {
		return nil, nil, fmt.Errorf("StockQuoteRequest is nil")
	}
	var sqrResp models.StockQuotesResponse
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
	baseReq.path = endpoints[1]["stocks"]["quotes"]

	sqr := &StockQuoteRequest{
		baseRequest:        baseReq,
		symbolParams:       &parameters.SymbolParams{},
		fiftyTwoWeekParams: &parameters.FiftyTwoWeekParams{},
	}

	baseReq.child = sqr

	return sqr
}
