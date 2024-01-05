package models

import (
	"fmt"
	"time"

	"github.com/MarketDataApp/sdk-go/helpers/dates"
)

type OptionsExpirationsResponse struct {
	Expirations []string
	Updated     int64
}

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


func (oer *OptionsExpirationsResponse) String() string {
	return fmt.Sprintf("Expirations: %v, Updated: %v", oer.Expirations, oer.Updated)
}

func (oer *OptionsExpirationsResponse) Unpack() ([]time.Time, error) {
	expirations := make([]time.Time, len(oer.Expirations))
	loc, _ := time.LoadLocation("America/New_York")
	for i, exp := range oer.Expirations {
		t, err := dates.ToTime(exp, loc)
		if err != nil {
			return nil, err
		}
		t = t.Add(time.Duration(16) * time.Hour) // Adding 16 hours to the time after parsing
		expirations[i] = t
	}
	return expirations, nil
}
