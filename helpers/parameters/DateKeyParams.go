package parameters

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/dates"
	"github.com/go-resty/resty/v2"
)

// DateKeyParam represents the date key parameter for a request.
// It encapsulates a date key string that is utilized in various API requests, particularly mandatory for V2 requests.
type DateKeyParam struct {
	DateKey string `path:"datekey" validate:"required"` // The date key in string format.
}

// SetDateKey sets the date key parameter for a request.
// This method validates the date key to ensure it adheres to the expected format as defined in the dates package.
//
// # Parameters
//
//   - q: A string representing the date key to be set.
//
// # Returns
//
//   - An error if the date key is in an invalid format, otherwise nil.
func (dk *DateKeyParam) SetDateKey(q string) error {
	if !dates.IsValidDateKey(q) {
		return fmt.Errorf("invalid date key format")
	}
	dk.DateKey = q
	return nil
}

// SetParams sets the parameters for the DateKeyParam within a given request.
// It leverages the ParseAndSetParams function to parse the DateKeyParam and apply it to the request.
//
// # Parameters
//
//   - request: A pointer to a resty.Request object where the date key parameter will be set.
//
// # Returns
//
//   - An error if parsing and setting the parameters fail, otherwise nil.
func (dk *DateKeyParam) SetParams(request *resty.Request) error {
	return ParseAndSetParams(dk, request)
}
