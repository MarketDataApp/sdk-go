package markets

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"
	"encoding/json"
	"net/http"
	"github.com/MarketDataApp/sdk-go/helpers/endpoints"

	md "github.com/MarketDataApp/sdk-go/client"
)

// MarketStatusRequest represents a request for market status.
type MarketStatusRequest struct {
	Path   string
	Params *StatusParams
}

// GetPath returns the path of the MarketStatusRequest.
func (msr *MarketStatusRequest) GetPath() (string, error) {
	if msr == nil {
		return "", fmt.Errorf("MarketStatusRequest is nil")
	}
	return msr.Path, nil
}

// GetQuery returns the query string of the MarketStatusRequest.
func (msr *MarketStatusRequest) GetQuery() (string, error) {
	if msr == nil || msr.Params == nil {
		return "", fmt.Errorf("MarketStatusRequest or Params is nil")
	}
	v := url.Values{}
	s := reflect.ValueOf(msr.Params).Elem()

	if s.Kind() != reflect.Struct {
		return "", fmt.Errorf("params is not a struct")
	}

	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		if f.CanInterface() {
			name := strings.ToLower(s.Type().Field(i).Name)
			value := f.Interface()
			if value != nil && !endpoints.IsZeroValue(value) {
				v.Add(name, fmt.Sprintf("%v", value))
			}
		}
	}

	return v.Encode(), nil
}

// Country sets the country parameter of the MarketStatusRequest.
func (msr *MarketStatusRequest) Country(q string) *MarketStatusRequest {
	if len(q) != 2 || !endpoints.IsAlpha(q) {
		msr.Params.Err = fmt.Errorf("invalid country code")
		return msr
	}
	msr.Params.Country = q
	return msr
}

// Date sets the date parameter of the MarketStatusRequest.
func (msr *MarketStatusRequest) Date(q interface{}) *MarketStatusRequest {
	date, err := md.DecodeDate(q)
	if err != nil {
		msr.Params.Err = err
		return msr
	}
	msr.Params.Date = date
	if msr.Params.Date != "" {
		msr.Params.From = ""
		msr.Params.To = ""
		if msr.Params.Countback != nil {
			msr.Params.Countback = nil
		}
	}
	return msr
}

// From sets the from parameter of the MarketStatusRequest.
func (msr *MarketStatusRequest) From(q interface{}) *MarketStatusRequest {
	date, err := md.DecodeDate(q)
	if err != nil {
		msr.Params.Err = err
		return msr
	}
	msr.Params.From = date
	if msr.Params.From != "" {
		if msr.Params.Date != "" {
			msr.Params.Date = ""
		}
		if msr.Params.Countback != nil {
			msr.Params.Countback = nil
		}
	}
	return msr
}

// To sets the to parameter of the MarketStatusRequest.
func (msr *MarketStatusRequest) To(q interface{}) *MarketStatusRequest {
	date, err := md.DecodeDate(q)
	if err != nil {
		msr.Params.Err = err
		return msr
	}
	msr.Params.To = date
	if msr.Params.To != "" {
		if msr.Params.Date != "" {
			msr.Params.Date = ""
		}
		if msr.Params.Countback != nil {
			msr.Params.Countback = nil
		}
	}
	return msr
}

// Countback sets the countback parameter of the MarketStatusRequest.
func (msr *MarketStatusRequest) Countback(q int) *MarketStatusRequest {
	msr.Params.Countback = &q
	if msr.Params.Countback != nil {
		if msr.Params.Date != "" {
			msr.Params.Date = ""
		}
		if msr.Params.From != "" {
			msr.Params.From = ""
		}
	}
	return msr
}

// MarketStatus represents the status of a market.
type MarketStatus struct {
	Date   time.Time
	Open   bool
	Closed bool
}

const (
	MarketStatusPath = "/v1/markets/status/"
)

// StatusParams represents the parameters for a status request.
type StatusParams struct {
	Country   string `query:"country"`
	Date      string `query:"date"`
	From      string `query:"from"`
	To        string `query:"to"`
	Countback *int   `query:"countback"`
	Err       error
}

// MarketStatusResponse represents the response from a market status request.
type MarketStatusResponse struct {
	Date   []int64   `json:"date"`
	Status *[]string `json:"status,omitempty"`
}

// String returns a string representation of the MarketStatusResponse.
func (msr MarketStatusResponse) String() string {
	var parts []string
	for i, date := range msr.Date {
		t := time.Unix(date, 0)
		dateStr := t.Format("2006-01-02")
		status := (*msr.Status)[i]
		part := fmt.Sprintf("Date: %s, Status: %s", dateStr, status)
		parts = append(parts, part)
	}
	return "MarketStatusResponse{\n" + strings.Join(parts, ",\n") + "\n}"
}

// String returns a string representation of the MarketStatus.
func (ms MarketStatus) String() string {
	return fmt.Sprintf("MarketStatus{Date: %v, Open: %v, Closed: %v}", ms.Date, ms.Open, ms.Closed)
}

// Unpack unpacks the MarketStatusResponse into a slice of MarketStatus.
func (msr *MarketStatusResponse) Unpack() ([]MarketStatus, error) {
	if len(msr.Date) != len(*msr.Status) {
		return nil, fmt.Errorf("date and status slices are not of the same length")
	}

	var marketStatuses []MarketStatus
	for i, date := range msr.Date {
		status := strings.ToLower((*msr.Status)[i])
		marketStatus := MarketStatus{
			Date:   time.Unix(date, 0),
			Open:   status == "open",
			Closed: status == "closed",
		}
		marketStatuses = append(marketStatuses, marketStatus)
	}

	return marketStatuses, nil
}

// GetOpenDates returns a slice of dates when the market was open.
func (msr *MarketStatusResponse) GetOpenDates() ([]time.Time, error) {
	marketStatuses, err := msr.Unpack()
	if err != nil {
		return nil, err
	}

	var openDates []time.Time
	for _, marketStatus := range marketStatuses {
		if marketStatus.Open {
			openDates = append(openDates, marketStatus.Date)
		}
	}

	return openDates, nil
}

// GetClosedDates returns a slice of dates when the market was closed.
func (msr *MarketStatusResponse) GetClosedDates() ([]time.Time, error) {
	marketStatuses, err := msr.Unpack()
	if err != nil {
		return nil, err
	}

	var closedDates []time.Time
	for _, marketStatus := range marketStatuses {
		if marketStatus.Closed {
			closedDates = append(closedDates, marketStatus.Date)
		}
	}

	return closedDates, nil
}

// New creates a new MarketStatusRequest.
func New(countries ...string) (*MarketStatusRequest, error) {
	country := "US" // Default value
	if len(countries) > 0 {
		country = countries[0]
	}

	statusParams := &StatusParams{
		Country: country,
	}

	return &MarketStatusRequest{
		Path:   MarketStatusPath,
		Params: statusParams,
	}, nil
}

// GetMarketStatus sends the MarketStatusRequest and returns the response.
func (msr *MarketStatusRequest) GetMarketStatus() (*MarketStatusResponse, error) {
	client, err := md.GetClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetFromRequest(msr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK status: %s", resp.Status)
	}

	var msrResp MarketStatusResponse
	err = json.NewDecoder(resp.Body).Decode(&msrResp)
	if err != nil {
		return nil, err
	}

	return &msrResp, nil
}