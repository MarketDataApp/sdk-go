package parameters

import (
	"github.com/go-resty/resty/v2"
)

// UniversalParams represents the universal parameters for a request.
// It includes limit, date format, offset, format, headers, columns, human, error, and feed.
type UniversalParams struct {
	Limit      int    `query:"limit"`
	DateFormat string `query:"dateformat"`
	Offset     int    `query:"offset"`
	Format     string `query:"format"`
	Headers    bool   `query:"headers"`
	Columns    string `query:"columns"`
	Human      bool   `query:"human"`
	Feed       string `query:"feed"`
	Error      error
}

// SetParams sets the parameters for the UniversalParams.
// It uses the parseAndSetParams function to parse and set the parameters.
func (up *UniversalParams) SetParams(request *resty.Request) error {
	return ParseAndSetParams(up, request)
}
