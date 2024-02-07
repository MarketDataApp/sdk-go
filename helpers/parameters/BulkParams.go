package parameters

import (
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
)

type BulkStockParams struct {
	Symbols  string `query:"symbols" validate:"required"`
	Snapshot bool   `query:"snapshot" validate:"optional"`
}

// SetSymbols sets the symbols parameter for the BulkStockParams.
// It validates that the symbols slice is not empty and then sets them as a comma-separated string.
// If the validation fails, it returns an error.
func (bsp *BulkStockParams) SetSymbols(symbols []string) error {
	if len(symbols) == 0 {
		return fmt.Errorf("symbols are required")
	}
	bsp.Symbols = strings.Join(symbols, ",")
	return nil
}

// SetSnapshot sets the snapshot parameter for the BulkStockParams.
// It allows enabling or disabling the snapshot feature in the request.
//
// Parameters:
//   - snapshot: A boolean value to enable or disable the snapshot feature.
func (bsp *BulkStockParams) SetSnapshot(snapshot bool) {
	bsp.Snapshot = snapshot
}


// SetParams sets the parameters for the BulkStockParams.
// It uses the parseAndSetParams function to parse and set the parameters.
func (bsp *BulkStockParams) SetParams(request *resty.Request) error {
	return ParseAndSetParams(bsp, request)
}

