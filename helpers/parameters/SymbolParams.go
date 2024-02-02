package parameters

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

type SymbolParams struct {
	Symbol string `path:"symbol" validate:"required"`
}

// SetSymbol sets the symbol parameter for the SymbolParams.
// It validates that the symbol is not an empty string.
// If the validation fails, it returns an error.
func (sp *SymbolParams) SetSymbol(symbol string) error {
	if symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	sp.Symbol = symbol
	return nil
}

// SetParams sets the parameters for the CandleParams.
// It uses the parseAndSetParams function to parse and set the parameters.
func (sp *SymbolParams) SetParams(request *resty.Request) error {
	return ParseAndSetParams(sp, request)
}
