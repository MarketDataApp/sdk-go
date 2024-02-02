package parameters

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/dates"
	"github.com/go-resty/resty/v2"
)

// DateKeyParam represents the date key parameter for a request.
// It includes a date key that is used in various requests and is required for V2 requests.
type DateKeyParam struct {
	DateKey string `path:"datekey" validate:"required"`
}

// SetDateKey sets the date key parameter for the TickersRequest.
// It validates the date key using the IsValidDateKey function from the dates package.
func (dk *DateKeyParam) SetDateKey(q string) error {
	if !dates.IsValidDateKey(q) {
		return fmt.Errorf("invalid date key format")
	}
	dk.DateKey = q
	return nil
}

// SetParams sets the parameters for the DateKeyParam.
// It uses the parseAndSetParams function to parse and set the parameters.
func (dk *DateKeyParam) SetParams(request *resty.Request) error {
	return ParseAndSetParams(dk, request)
}
