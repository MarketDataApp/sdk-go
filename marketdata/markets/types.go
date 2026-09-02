package markets

import (
	"fmt"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/timezone"
)

// MarketStatus represents the status of the market on a given day.
type MarketStatus struct {
	// Date is the date of this status
	Date time.Time `json:"date"`

	// Open indicates if the market was/is open on this date
	Open bool `json:"open"`

	// Status is the API's status string: "open" or "closed" (early-close
	// days are reported as "open"). It is empty for days outside the
	// calendar's coverage.
	Status string `json:"status"`
}

// String returns a summary of the market status.
func (m MarketStatus) String() string {
	return fmt.Sprintf("%s %s Open: %t", m.Date.Format("2006-01-02"), m.Status, m.Open)
}

// IsOpen reports whether this is an open market day (Status == "open").
// Early-close days count as open: the API's status vocabulary is strictly
// "open"/"closed", and shortened sessions are reported as "open".
func (m *MarketStatus) IsOpen() bool {
	return m.Status == "open"
}

// IsClosed reports whether this is a closed market day (Status ==
// "closed"). A day outside the calendar's coverage carries an empty
// Status and is neither open nor closed: IsOpen and IsClosed both return
// false then.
func (m *MarketStatus) IsClosed() bool {
	return m.Status == "closed"
}

// statusResponse is the API response for market status.
type statusResponse struct {
	Status string   `json:"s"`
	Date   []int64  `json:"date"` // Unix timestamps
	Stat   []string `json:"status"`
}

// RequiredColumns names the date array both status conversions take their
// row count from. See http.ColumnRequirer.
func (r *statusResponse) RequiredColumns() []string { return []string{"date"} }

func (r *statusResponse) toMarketStatus() *MarketStatus {
	if r == nil || len(r.Date) == 0 {
		return nil
	}

	status := safeIndexStr(r.Stat, 0)
	return &MarketStatus{
		Date:   timezone.ToEastern(r.Date[0]),
		Open:   status == "open",
		Status: status,
	}
}

func (r *statusResponse) toMarketStatuses() []MarketStatus {
	if r == nil || len(r.Date) == 0 {
		return nil
	}

	statuses := make([]MarketStatus, len(r.Date))
	for i := range r.Date {
		status := safeIndexStr(r.Stat, i)
		statuses[i] = MarketStatus{
			Date:   timezone.ToEastern(r.Date[i]),
			Open:   status == "open",
			Status: status,
		}
	}
	return statuses
}

func safeIndexStr(s []string, i int) string {
	if i < len(s) {
		return s[i]
	}
	return ""
}
