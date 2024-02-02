package parameters

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

// CandleParams represents the parameters for a stock candle request.
type StockCandleParams struct {
	AdjustSplits    bool   `query:"adjustsplits" validate:"optional"`
	AdjustDividends bool   `query:"adjustdividends" validate:"optional"` // Not yet implemented in the API
	Extended        bool   `query:"extended" validate:"optional"`
	Exchange        string `query:"exchange" validate:"optional"` // Not needed until non-US exchanges are added
}

// SetAdjustSplits sets the AdjustSplits parameter for the StockCandleParams.
func (scp *StockCandleParams) SetAdjustSplits(adjustSplits bool) {
	scp.AdjustSplits = adjustSplits
}

// SetAdjustDividends sets the AdjustDividends parameter for the StockCandleParams.
func (scp *StockCandleParams) SetAdjustDividends(adjustDividends bool) {
	scp.AdjustDividends = adjustDividends
}

// SetExtended sets the Extended parameter for the StockCandleParams.
func (scp *StockCandleParams) SetExtended(extended bool) {
	scp.Extended = extended
}

// SetExchange sets the Exchange parameter for the StockCandleParams.
// It validates that the exchange is not an empty string.
// If the validation fails, it returns an error.
func (scp *StockCandleParams) SetExchange(exchange string) error {
	if exchange == "" {
		return fmt.Errorf("nil value set for exchange")
	}
	scp.Exchange = exchange
	return nil
}

// SetParams sets the parameters for the StockCandleParams.
// It uses the parseAndSetParams function to parse and set the parameters.
func (scp *StockCandleParams) SetParams(request *resty.Request) error {
	return ParseAndSetParams(scp, request)
}
