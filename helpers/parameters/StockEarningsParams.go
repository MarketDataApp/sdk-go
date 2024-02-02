package parameters

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/dates"
	"github.com/go-resty/resty/v2"
)

type StockEarningsParams struct {
	Report string `query:"report"`
}

// SetReport sets the report parameter for the EarningsParams.
// It validates the report parameter using the IsValidDateKey function from the dates package.
func (ep *StockEarningsParams) SetReport(q string) error {
	if !dates.IsValidDateKey(q) {
		err := fmt.Errorf("invalid report parameter")
		return err
	}
	ep.Report = q
	return nil
}

// SetParams sets the parameters for the StockEarningsParams.
// If the parsing and setting of parameters fail, it returns an error.
func (sep *StockEarningsParams) SetParams(request *resty.Request) error {
	return ParseAndSetParams(sep, request)
}
