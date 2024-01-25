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

// Packed sends the StockQuoteRequest and returns the StockQuotesResponse.
// It returns an error if the request fails.
func (sqr *StockQuoteRequest) Packed() (*models.StockQuotesResponse, error) {
	if sqr == nil {
		return nil, fmt.Errorf("StockQuoteRequest is nil")
	}
	var sqrResp models.StockQuotesResponse
	_, err := sqr.baseRequest.client.GetFromRequest(sqr.baseRequest, &sqrResp)
	if err != nil {
		return nil, err
	}

	return &sqrResp, nil
}

// Get sends the StockQuoteRequest, unpacks the StockQuotesResponse and returns the data.
// It returns an error if the request or unpacking fails.
func (sqr *StockQuoteRequest) Get() ([]models.StockQuote, error) {
	if sqr == nil {
		return nil, fmt.Errorf("StockQuoteRequest is nil")
	}
	
	// Use the Packed method to make the request
	sqrResp, err := sqr.Packed()
	if err != nil {
		return nil, err
	}

	// Unpack the data using the Unpack method in the response
	data, err := sqrResp.Unpack()
	if err != nil {
		return nil, err
	}

	return data, nil
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
