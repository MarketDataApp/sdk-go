package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/MarketDataApp/sdk-go/helpers/dates"
)
// MarketStatusResponse holds the response data for a market status request.
// It includes a slice of dates and an optional slice of corresponding market statuses.
type MarketStatusResponse struct {
	Date   []int64   `json:"date"`           // Date contains UNIX timestamps for each market status entry.
	Status *[]string `json:"status,omitempty"` // Status is a pointer to a slice of market status strings, which can be omitted if empty.
}

// IsValid checks if the MarketStatusResponse contains at least one date.
//
// Returns:
//   - bool: True if there is at least one date, false otherwise.
func (msr *MarketStatusResponse) IsValid() bool {
	return len(msr.Date) > 0
}

// String returns a string representation of the MarketStatusResponse, including dates and their corresponding statuses.
//
// Returns:
//   - string: A string representation of the MarketStatusResponse.
func (msr *MarketStatusResponse) String() string {
	var parts []string
	if msr.Status != nil && len(msr.Date) == len(*msr.Status) {
		for i, date := range msr.Date {
			t := time.Unix(date, 0)
			dateStr := t.Format("2006-01-02")
			status := (*msr.Status)[i]
			part := fmt.Sprintf("Date: %s, Status: %s", dateStr, status)
			parts = append(parts, part)
		}
	}
	return "MarketStatusResponse{\n" + strings.Join(parts, ",\n") + "\n}"
}

// String returns a string representation of the MarketStatusReport.
//
// Returns:
//   - string: A string representation of the MarketStatusReport.
func (ms MarketStatusReport) String() string {
	return fmt.Sprintf("MarketStatus{Date: %v, Open: %v, Closed: %v}", ms.Date, ms.Open, ms.Closed)
}

// Unpack unpacks the MarketStatusResponse into a slice of MarketStatusReport.
//
// Returns:
//   - []MarketStatusReport: A slice of MarketStatusReport derived from the MarketStatusResponse.
//   - error: An error if the date and status slices are not of the same length or any other issue occurs.
func (msr *MarketStatusResponse) Unpack() ([]MarketStatusReport, error) {
	if msr.Status == nil || len(msr.Date) != len(*msr.Status) {
		return nil, fmt.Errorf("date and status slices are not of the same length")
	}

	var marketStatuses []MarketStatusReport
	for i, date := range msr.Date {
		status := strings.ToLower((*msr.Status)[i])
		marketStatus := MarketStatusReport{
			Date:   time.Unix(date, 0),
			Open:   status == "open",
			Closed: status == "closed",
		}
		marketStatuses = append(marketStatuses, marketStatus)
	}

	return marketStatuses, nil
}

// GetOpenDates returns a slice of dates when the market was open.
//
// Returns:
//   - []time.Time: A slice of time.Time representing the dates when the market was open.
//   - error: An error if there's an issue unpacking the MarketStatusResponse.
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
//
// Returns:
//   - []time.Time: A slice of time.Time representing the dates when the market was closed.
//   - error: An error if there's an issue unpacking the MarketStatusResponse.
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

// GetDateRange returns the date range of the MarketStatusResponse.
//
// Returns:
//   - *dates.DateRange: A pointer to a DateRange object representing the earliest and latest dates in the MarketStatusResponse.
//   - error: An error if there's an issue finding the earliest or latest date.
func (msr *MarketStatusResponse) GetDateRange() (*dates.DateRange, error) {
	// Use the Earliest and Latest functions to find the earliest and latest dates
	earliest, err := dates.Earliest(msr.Date)
	if err != nil {
		return nil, err
	}

	latest, err := dates.Latest(msr.Date)
	if err != nil {
		return nil, err
	}

	// Create a new DateRange using NewDateRange function
	dr, err := dates.NewDateRange(earliest, latest)
	if err != nil {
		return nil, err
	}

	return dr, nil
}

// MarketStatusReport represents the status of a market.
type MarketStatusReport struct {
	Date   time.Time
	Open   bool
	Closed bool
}
