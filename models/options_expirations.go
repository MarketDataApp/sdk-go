package models

import (
	"fmt"
	"time"

	"github.com/MarketDataApp/sdk-go/helpers/dates"
)

// OptionsExpirationsResponse represents the response structure for options expirations.
// It includes a slice of expiration dates as strings and a timestamp for when the data was last updated.
type OptionsExpirationsResponse struct {
	Expirations []string // Expirations is a slice of strings representing the expiration dates of options.
	Updated     int64    // Updated is a UNIX timestamp indicating when the data was last updated.
}

// IsValid checks the validity of the options expirations response.
//
// It verifies that the Expirations slice is not empty and that each expiration date string can be parsed into a time.Time object.
// This parsing is done according to the "America/New_York" timezone.
//
// Returns:
//   - A boolean indicating whether the OptionsExpirationsResponse is valid.
func (oer *OptionsExpirationsResponse) IsValid() bool {
	loc, _ := time.LoadLocation("America/New_York")
	if len(oer.Expirations) == 0 {
		return false
	}
	for _, exp := range oer.Expirations {
		_, err := dates.ToTime(exp, loc)
		if err != nil {
			return false
		}
	}
	return true
}


// String provides a string representation of the OptionsExpirationsResponse.
//
// It formats the expirations and the updated timestamp into a readable string.
//
// Returns:
//   - A string that represents the OptionsExpirationsResponse object.
func (oer *OptionsExpirationsResponse) String() string {
	return fmt.Sprintf("Expirations: %v, Updated: %v", oer.Expirations, oer.Updated)
}

// Unpack converts the expiration date strings in the OptionsExpirationsResponse to a slice of time.Time objects.
//
// It parses each date string in the Expirations slice into a time.Time object, adjusting the time to 4:00 PM Eastern Time,
// which is the typical expiration time for options contracts.
//
// Returns:
//   - A slice of time.Time objects representing the expiration dates and times.
//   - An error if any date string cannot be parsed or if the "America/New_York" timezone cannot be loaded.
func (oer *OptionsExpirationsResponse) Unpack() ([]time.Time, error) {
	expirations := make([]time.Time, len(oer.Expirations))
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return nil, err
	}
	for i, exp := range oer.Expirations {
		t, err := dates.ToTime(exp, loc)
		if err != nil {
			return nil, err
		}
		t = t.Add(time.Duration(16) * time.Hour) // Adding 16 hours to the time after parsing. Options expire 4:00 PM Eastern Time.
		expirations[i] = t
	}
	return expirations, nil
}
