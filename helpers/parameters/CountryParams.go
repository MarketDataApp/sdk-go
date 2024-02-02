package parameters

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/types"
	"github.com/go-resty/resty/v2"
)

// CountryParams represents the parameters for a country.
// It includes a country code that is used in various requests.
type CountryParams struct {
	Country string `query:"country"`
}

// SetCountry sets the country parameter for the CountryParams.
// It validates the country code to be of length 2 and only contain alphabets.
// If the validation fails, it returns an error.
func (cp *CountryParams) SetCountry(q string) error {
	if len(q) != 2 || !types.IsAlpha(q) {
		err := fmt.Errorf("invalid country code")
		return err
	}
	cp.Country = q
	return nil
}

// SetParams sets the parameters for the CountryParams in the request.
// If the parsing and setting of parameters fail, it returns an error.
func (cp *CountryParams) SetParams(request *resty.Request) error {
	return ParseAndSetParams(cp, request)
}
