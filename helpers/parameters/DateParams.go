package parameters

import (
	"github.com/MarketDataApp/sdk-go/helpers/dates"
	"github.com/go-resty/resty/v2"
)

// DateParams represents the parameters for a date request.
// It includes a date, from, to, and countback, which are used in various requests.
type DateParams struct {
	Date      string `query:"date"`
	From      string `query:"from"`
	To        string `query:"to"`
	Countback *int   `query:"countback"`
}

// SetDate sets the date parameter of the DateParams.
// It validates the date using the ToDayString function from the dates package.
// If the validation fails, it returns an error.
func (dp *DateParams) SetDate(q interface{}) error {
	date, err := dates.ToDayString(q)
	if err != nil {
		return err
	}
	dp.Date = date
	if dp.Date != "" {
		dp.From = ""
		dp.To = ""
		if dp.Countback != nil {
			dp.Countback = nil
		}
	}
	return nil
}

// SetFrom sets the from parameter of the DateParams.
// It validates the from date using the ToDayString function from the dates package.
// If the validation fails, it returns an error.
func (dp *DateParams) SetFrom(q interface{}) error {
	date, err := dates.ToDayString(q)
	if err != nil {
		return err
	}
	dp.From = date
	if dp.From != "" {
		if dp.Date != "" {
			dp.Date = ""
		}
		if dp.Countback != nil {
			dp.Countback = nil
		}
	}
	return nil
}

// SetTo sets the to parameter of the DateParams.
// It validates the to date using the ToDayString function from the dates package.
// If the validation fails, it returns an error.
func (dp *DateParams) SetTo(q interface{}) error {
	date, err := dates.ToDayString(q)
	if err != nil {
		return err
	}
	dp.To = date
	if dp.To != "" {
		if dp.Date != "" {
			dp.Date = ""
		}
		if dp.Countback != nil {
			dp.Countback = nil
		}
	}
	return nil
}

// SetCountback sets the countback parameter of the DateParams.
// If countback is not nil, it clears the date and from parameters.
func (dp *DateParams) SetCountback(q int) error {
	dp.Countback = &q
	if dp.Countback != nil {
		if dp.Date != "" {
			dp.Date = ""
		}
		if dp.From != "" {
			dp.From = ""
		}
	}
	return nil
}

// SetParams sets the parameters for the DateParams.
// It uses the parseAndSetParams function to parse and set the parameters.
func (dp *DateParams) SetParams(request *resty.Request) error {
	return ParseAndSetParams(dp, request)
}

