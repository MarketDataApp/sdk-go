package parameters

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/types"
	"github.com/go-resty/resty/v2"
)

// CountryParams represents the parameters for identifying a country in requests.
// It encapsulates a country code as a string.
type CountryParams struct {
	Country string `query:"country"` // Country code used in various API requests.
}

// SetCountry sets the country code in CountryParams after validating it.
// The country code must be exactly 2 characters long and consist only of alphabetic characters.
//
// Parameters:
//   - q: A string representing the country code to be set.
//
// Returns:
//   - An error if the country code is invalid (either not 2 characters long or contains non-alphabetic characters).
func (cp *CountryParams) SetCountry(q string) error {
	if len(q) != 2 || !types.IsAlpha(q) {
		return fmt.Errorf("invalid country code: must be 2 alphabetic characters")
	}
	cp.Country = q
	return nil
}

// SetParams sets the country parameter for a given request.
// It utilizes the ParseAndSetParams function to apply the parameters to the request.
//
// Parameters:
//   - request: A pointer to a resty.Request object where the country parameter will be set.
//
// Returns:
//   - An error if parsing and setting the parameters fail.
func (cp *CountryParams) SetParams(request *resty.Request) error {
	return ParseAndSetParams(cp, request)
}
