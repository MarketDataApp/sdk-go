package parameters

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

type ResolutionParams struct {
	Resolution string `path:"resolution" validate:"required"`
}

// SetResolution sets the resolution parameter for the ResolutionParams.
// It validates that the resolution is not an empty string.
// If the validation fails, it returns an error.
func (rp *ResolutionParams) SetResolution(resolution string) error {
	if resolution == "" {
		return fmt.Errorf("resolution is required")
	}
	rp.Resolution = resolution
	return nil
}

// SetParams sets the parameters for the CandleParams.
// It uses the parseAndSetParams function to parse and set the parameters.
func (rp *ResolutionParams) SetParams(request *resty.Request) error {
	return ParseAndSetParams(rp, request)
}
