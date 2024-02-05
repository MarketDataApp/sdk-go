package parameters

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

// UserInputParams represents the user input parameter for a request.
// It includes a user input that is used in various requests and is required.
type UserInputParams struct {
	UserInput string `path:"userInput" validate:"required"`
}

// SetUserInput sets the user input parameter for the OptionsLookupRequest.
// It validates that the user input is not empty.
func (u *UserInputParams) SetUserInput(q string) error {
	if q == "" {
		return fmt.Errorf("user input cannot be empty")
	}
	u.UserInput = q
	return nil
}

// SetParams sets the parameters for the UserInputParams.
// It uses the parseAndSetParams function to parse and set the parameters.
func (u *UserInputParams) SetParams(request *resty.Request) error {
	return ParseAndSetParams(u, request)
}
