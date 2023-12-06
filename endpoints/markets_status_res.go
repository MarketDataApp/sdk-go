package endpoints

import (
	"fmt"
	"strings"
	"time"

	md "github.com/MarketDataApp/sdk-go/client"
)

// MarketStatusResponse represents the response from a market status request.
type MarketStatusResponse struct {
	*md.MarketDataResponse
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
func (ms MarketStatusObj) String() string {
	return fmt.Sprintf("MarketStatus{Date: %v, Open: %v, Closed: %v}", ms.Date, ms.Open, ms.Closed)
}

// Unpack unpacks the MarketStatusResponse into a slice of MarketStatus.
func (msr *MarketStatusResponse) Unpack() ([]MarketStatusObj, error) {
	if len(msr.Date) != len(*msr.Status) {
		return nil, fmt.Errorf("date and status slices are not of the same length")
	}

	var marketStatuses []MarketStatusObj
	for i, date := range msr.Date {
		status := strings.ToLower((*msr.Status)[i])
		marketStatus := MarketStatusObj{
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

// MarketStatusObj represents the status of a market.
type MarketStatusObj struct {
	Date   time.Time
	Open   bool
	Closed bool
}
