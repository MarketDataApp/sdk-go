package parameters

import "github.com/go-resty/resty/v2"

// FiftyTwoWeekParams represents the unique parameters for a StockQuoteRequest
type FiftyTwoWeekParams struct {
	FiftyTwoWeek bool `query:"52week"`
}

// SetFiftyTwoWeek sets the FiftyTwoWeek parameter for the FiftyTwoWeekParams.
func (sqp *FiftyTwoWeekParams) SetFiftyTwoWeek(q bool) {
	sqp.FiftyTwoWeek = q
}

// SetParams sets the FiftyTwoWeek parameter for the FiftyTwoWeekParams.
// If the parsing and setting of parameters fail, it returns an error.
func (sqp *FiftyTwoWeekParams) SetParams(request *resty.Request) error {
	return ParseAndSetParams(sqp, request)
}
