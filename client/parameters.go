package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/dates"
	"github.com/go-resty/resty/v2"
)

// CountryParams represents the parameters for a country.
// It includes a country code that is used in various requests.
type CountryParams struct {
	Country string `query:"country"`
}

// SetCountry sets the country parameter for the CountryParams.
// It validates the country code to be of length 2 and only contain alphabets.
// If the validation fails, it returns an error.
func (cp *CountryParams) SetCountry(q string) error {
	if len(q) != 2 || !IsAlpha(q) {
		err := fmt.Errorf("invalid country code")
		return err
	}
	cp.Country = q
	return nil
}

// UniversalParams represents the universal parameters for a request.
// It includes limit, date format, offset, format, headers, columns, human, and error.
type UniversalParams struct {
	Limit      int    `query:"limit"`
	DateFormat string `query:"dateformat"`
	Offset     int    `query:"offset"`
	Format     string `query:"format"`
	Headers    bool   `query:"headers"`
	Columns    string `query:"columns"`
	Human      bool   `query:"human"`
	Error      error
}

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
	return parseAndSetParams(scp, request)
}

// CandleParams represents the parameters for a candle request.
// It includes a symbol and a resolution, both of which are required.
type CandleParams struct {
	Symbol     string `path:"symbol" validate:"required"`
	Resolution string `path:"resolution" validate:"required"`
}

// SetSymbol sets the symbol parameter for the CandleParams.
// It validates that the symbol is not an empty string.
// If the validation fails, it returns an error.
func (cp *CandleParams) SetSymbol(symbol string) error {
	if symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	cp.Symbol = symbol
	return nil
}

// SetResolution sets the resolution parameter for the CandleParams.
// It validates that the resolution is not an empty string.
// If the validation fails, it returns an error.
func (cp *CandleParams) SetResolution(resolution string) error {
	if resolution == "" {
		return fmt.Errorf("resolution is required")
	}
	cp.Resolution = resolution
	return nil
}

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
	return parseAndSetParams(dp, request)
}

// SetParams sets the parameters for the UniversalParams.
// It uses the parseAndSetParams function to parse and set the parameters.
func (up *UniversalParams) SetParams(request *resty.Request) error {
	return parseAndSetParams(up, request)
}

// SetParams sets the parameters for the DateKeyParam.
// It uses the parseAndSetParams function to parse and set the parameters.
func (dk *DateKeyParam) SetParams(request *resty.Request) error {
	return parseAndSetParams(dk, request)
}

// SetParams sets the parameters for the CandleParams.
// It uses the parseAndSetParams function to parse and set the parameters.
func (cp *CandleParams) SetParams(request *resty.Request) error {
	return parseAndSetParams(cp, request)
}

// SetParams sets the parameters for the CountryParams in the request.
// If the parsing and setting of parameters fail, it returns an error.
func (cp *CountryParams) SetParams(request *resty.Request) error {
	return parseAndSetParams(cp, request)
}
