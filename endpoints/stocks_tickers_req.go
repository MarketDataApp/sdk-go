// Package stocks provides the /stocks endpoints
package endpoints

import (
	"fmt"
	"time"

	md "github.com/MarketDataApp/sdk-go/client"
	"github.com/MarketDataApp/sdk-go/helpers/dates"
	"github.com/go-resty/resty/v2"
)

// TickersPath is the path for the /stocks/tickers endpoint.
const (
	TickersPath = "/v2/stocks/tickers/{date}/"
)

// TickersRequest represents a request to the /stocks/tickers endpoint.
type TickersRequest struct {
	*resty.Request
	Client    *md.MarketDataClient
	Path      string
	ParamDate string `path:"date" validate:"required"`
	Error     error
}

// GetResty returns the resty.Request for the TickersRequest.
func (tr *TickersRequest) GetResty() *resty.Request {
	return tr.Request
}

// String returns a string representation of TickersRequest.
func (tr *TickersRequest) String() string {
	if tr.Error != nil {
		return fmt.Sprintf("Path: %s, Date: %s, Error: %v", tr.Path, tr.ParamDate, tr.Error)
	}
	return fmt.Sprintf("Path: %s, Date: %s", tr.Path, tr.ParamDate)
}

// GetPath returns the path for the TickersRequest.
func (tr *TickersRequest) GetPath() string {
	return tr.Path
}

// GetError returns the error for the TickersRequest.
func (tr *TickersRequest) GetError() error {
	if tr.Error != nil {
		return tr.Error
	}
	return nil
}

// Date sets the date parameter for the TickersRequest.
func (tr *TickersRequest) Date(q interface{}) *TickersRequest {
	dateString, err := dates.ToDayString(q)
	if err != nil {
		tr.Error = err
	} else {
		tr.ParamDate = dateString
	}
	return tr
}

// SetParams parses the TickersRequest and sets the parameters in the provided resty.Request.
// It returns an error if the parsing or setting of parameters fails.
func (tr *TickersRequest) SetParams(request *resty.Request) error {
	return md.ParseAndSetParams(tr, request)
}

// GetParams packs the TickersRequest struct into a slice of interface{} and returns it.
func (tr *TickersRequest) GetParams() []md.MarketDataParam {
	params := []md.MarketDataParam{tr}
	return params
}

// StockTickers creates a new TickersRequest and associates it with the provided client.
// If no client is provided, it uses the default client.
func StockTickers(client ...*md.MarketDataClient) *TickersRequest {
	var mdClient *md.MarketDataClient
	var err error

	if len(client) > 0 {
		mdClient = client[0]
	} else {
		mdClient, err = md.GetClient()
		if err != nil {
			return &TickersRequest{Error: err}
		}
	}

	tr := &TickersRequest{
		Request: mdClient.R(),
		Path:    TickersPath,
		Client:  mdClient, // Assign the client to the struct
	}

	// Set the date to the current time
	tr.Date(time.Now())

	return tr
}

// GetTickers sends the TickersRequest and returns the TickersResponse.
// It accepts an optional client variable and uses that client if it is provided.
// If not, it uses the GetClient function.
func (tr *TickersRequest) Get() (*TickersResponse, error) {
	var trResp TickersResponse
	mdr, err := tr.Client.GetFromRequest(tr, &trResp)
	if err != nil {
		return nil, err
	}
	trResp.MarketDataResponse = mdr

	return &trResp, nil
}

// CombineTickerResponses combines multiple TickersResponses into a single map.
func CombineTickerResponses(responses []*TickersResponse) (map[string]TickerObj, error) {
	tickerMap := make(map[string]TickerObj)
	for _, response := range responses {
		responseMap, err := response.ToMap()
		if err != nil {
			return nil, err
		}
		for key, value := range responseMap {
			tickerMap[key] = value
		}
	}
	return tickerMap, nil
}
