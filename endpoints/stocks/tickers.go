// Package stocks provides the /stocks endpoints
package stocks

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	md "github.com/MarketDataApp/sdk-go/client"
	"github.com/MarketDataApp/sdk-go/helpers/endpoints"
	"github.com/MarketDataApp/sdk-go/helpers/dates"

)

// TickerInfo represents the information of a ticker.
type TickerInfo struct {
	Symbol        string
	Name          string
	Type          string
	Currency      string
	Exchange      string
	FigiShares    string
	FigiComposite string
	Cik           string
	Updated       *time.Time
}

// String returns a string representation of TickerInfo.
func (ti *TickerInfo) String() string {
	updated := ""
	if ti.Updated != nil {
		updated = ti.Updated.String()
	}
	return fmt.Sprintf("TickerInfo{Symbol: %s, Name: %s, Type: %s, Currency: %s, Exchange: %s, FigiShares: %s, FigiComposite: %s, Cik: %s, Updated: %s}", ti.Symbol, ti.Name, ti.Type, ti.Currency, ti.Exchange, ti.FigiShares, ti.FigiComposite, ti.Cik, updated)
}

// TickersResponse represents the response from the /stocks/tickers endpoint.
type TickersResponse struct {
	Symbol        []string `json:"symbol"`
	Name          []string `json:"name,omitempty"`
	Type          []string `json:"type,omitempty"`
	Currency      []string `json:"currency,omitempty"`
	Exchange      []string `json:"exchange,omitempty"`
	FigiShares    []string `json:"figiShares,omitempty"`
	FigiComposite []string `json:"figiComposite,omitempty"`
	Cik           []string `json:"cik,omitempty"`
	Updated       *[]int64 `json:"updated,omitempty"`
}

// String returns a string representation of TickersResponse.
func (tr *TickersResponse) String() string {
	var str strings.Builder
	str.WriteString("TickersResponse{\n")
	for i := range tr.Symbol {
		str.WriteString(fmt.Sprintf("Symbol: %s, Name: %s, Type: %s, Currency: %s, Exchange: %s, FigiShares: %s, FigiComposite: %s, Cik: %s\n", tr.Symbol[i], tr.Name[i], tr.Type[i], tr.Currency[i], tr.Exchange[i], tr.FigiShares[i], tr.FigiComposite[i], tr.Cik[i]))
		if tr.Updated != nil && (*tr.Updated)[i] != 0 {
			str.WriteString(fmt.Sprintf("Updated: %v\n", time.Unix((*tr.Updated)[i], 0)))
		} else {
			str.WriteString("Updated: \n")
		}
	}
	str.WriteString("}")
	return str.String()
}

// ToMap converts TickersResponse to a map with the symbol as the key.
func (tr *TickersResponse) ToMap() (map[string]TickerInfo, error) {
	tickerInfos, err := tr.Unpack()
	if err != nil {
		return nil, err
	}

	tickerMap := make(map[string]TickerInfo)
	for _, tickerInfo := range tickerInfos {
		tickerMap[tickerInfo.Symbol] = tickerInfo
	}
	return tickerMap, nil
}

// Unpack converts TickersResponse to a slice of TickerInfo.
func (tr *TickersResponse) Unpack() ([]TickerInfo, error) {
	if tr == nil {
		return nil, fmt.Errorf("TickersResponse is nil")
	}
	var tickerInfos []TickerInfo
	for i := range tr.Symbol {
		if i >= len(tr.Name) || i >= len(tr.Type) || i >= len(tr.Currency) || i >= len(tr.Exchange) || i >= len(tr.FigiShares) || i >= len(tr.FigiComposite) || i >= len(tr.Cik) || (tr.Updated != nil && i >= len(*tr.Updated)) {
			return nil, fmt.Errorf("index out of range")
		}
		tickerInfo := TickerInfo{
			Symbol:        tr.Symbol[i],
			Name:          tr.Name[i],
			Type:          tr.Type[i],
			Currency:      tr.Currency[i],
			Exchange:      tr.Exchange[i],
			FigiShares:    tr.FigiShares[i],
			FigiComposite: tr.FigiComposite[i],
			Cik:           tr.Cik[i],
		}
		if tr.Updated != nil && (*tr.Updated)[i] != 0 {
			t := time.Unix((*tr.Updated)[i], 0)
			tickerInfo.Updated = &t
		}
		tickerInfos = append(tickerInfos, tickerInfo)
	}
	return tickerInfos, nil
}

// TickersParams represents the parameters for the /stocks/tickers endpoint.
type TickersParams struct {
	Date string `path:"date" validate:"required"`
	Err  error
}

// String returns a string representation of TickersParams.
func (tp *TickersParams) String() string {
	if tp.Err != nil {
		return fmt.Sprintf("Date: %s, Error: %v", tp.Date, tp.Err)
	}
	return fmt.Sprintf("Date: %s", tp.Date)
}

// TickersPath is the path for the /stocks/tickers endpoint.
const (
	TickersPath = "/v2/stocks/tickers/{date}/"
)

// TickersRequest represents a request to the /stocks/tickers endpoint.
type TickersRequest struct {
	Path   string
	Params *TickersParams
}

// String returns a string representation of TickersRequest.
func (tr *TickersRequest) String() string {
	return fmt.Sprintf("TickersRequest:\nPath: %s\nParams: %s", tr.Path, tr.Params.String())
}

// GetPath returns the path for the TickersRequest.
func (tr *TickersRequest) GetPath() (string, error) {
	path, err := endpoints.BuildPath(tr.Path, tr.Params)
	if err != nil {
		return "", err
	}
	return path, nil
}

// GetQuery returns the query string for the TickersRequest.
func (tr *TickersRequest) GetQuery() (string, error) {
	return "", nil
}

// Date sets the date parameter for the TickersRequest.
func (tr *TickersRequest) Date(q interface{}) *TickersRequest {
	dateString, err := dates.ToDayString(q)
	if err != nil {
		tr.Params.Err = err
	} else {
		tr.Params.Date = dateString
	}
	return tr
}

// New creates a new TickersRequest.
func New(dates ...string) (*TickersRequest, error) {
	tr := &TickersRequest{
		Path:   TickersPath,
		Params: &TickersParams{},
	}

	if len(dates) > 0 {
		tr.Date(dates[0])
	} else {
		tr.Date(time.Now())
	}

	if tr.Params.Err != nil {
		return nil, tr.Params.Err
	}

	return tr, nil
}

// GetTickers sends the TickersRequest and returns the TickersResponse.
func (tr *TickersRequest) GetTickers() (*TickersResponse, error) {
	client, err := md.GetClient("dev")
	if err != nil {
		return nil, err
	}
	resp, err := client.GetFromRequest(tr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK status: %s", resp.Status)
	}

	var trResp TickersResponse
	err = json.NewDecoder(resp.Body).Decode(&trResp)
	if err != nil {
		return nil, err
	}

	return &trResp, nil
}

// CombineTickerResponses combines multiple TickersResponses into a single map.
func CombineTickerResponses(responses []*TickersResponse) (map[string]TickerInfo, error) {
	tickerMap := make(map[string]TickerInfo)
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

// SaveToCSV saves the ticker map to a CSV file.
func SaveToCSV(tickerMap map[string]TickerInfo, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	err = writer.Write([]string{"Symbol", "Name", "Type", "Currency", "Exchange", "FigiShares", "FigiComposite", "Cik", "Updated"})
	if err != nil {
		return err
	}

	// Write data
	for _, tickerInfo := range tickerMap {
		updated := ""
		if tickerInfo.Updated != nil {
			updated = tickerInfo.Updated.String()
		}
		err = writer.Write([]string{tickerInfo.Symbol, tickerInfo.Name, tickerInfo.Type, tickerInfo.Currency, tickerInfo.Exchange, tickerInfo.FigiShares, tickerInfo.FigiComposite, tickerInfo.Cik, updated})
		if err != nil {
			return err
		}
	}

	return nil
}

// MapToTickersResponse converts a map of TickerInfo to a TickersResponse.
func MapToTickersResponse(tickerMap map[string]TickerInfo) *TickersResponse {
    var tr TickersResponse
    keys := make([]string, 0, len(tickerMap))
    for key := range tickerMap {
        keys = append(keys, key)
    }
    sort.Strings(keys)

    for _, key := range keys {
        tickerInfo := tickerMap[key]
        tr.Symbol = append(tr.Symbol, tickerInfo.Symbol)
        tr.Name = append(tr.Name, tickerInfo.Name)
        tr.Type = append(tr.Type, tickerInfo.Type)
        tr.Currency = append(tr.Currency, tickerInfo.Currency)
        tr.Exchange = append(tr.Exchange, tickerInfo.Exchange)
        tr.FigiShares = append(tr.FigiShares, tickerInfo.FigiShares)
        tr.FigiComposite = append(tr.FigiComposite, tickerInfo.FigiComposite)
        tr.Cik = append(tr.Cik, tickerInfo.Cik)
		if tickerInfo.Updated != nil {
			*tr.Updated = append(*tr.Updated, tickerInfo.Updated.Unix())
		} else {
			*tr.Updated = append(*tr.Updated, 0)
		}
	}
    return &tr
}
