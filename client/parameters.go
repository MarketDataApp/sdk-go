package client

import (
	"fmt"

	"github.com/MarketDataApp/sdk-go/helpers/dates"
	"github.com/go-resty/resty/v2"
)

type CountryParams struct {
	Country string `query:"country"`
}

func (cp *CountryParams) SetCountry(q string) error {
	if len(q) != 2 || !IsAlpha(q) {
		err := fmt.Errorf("invalid country code")
		return err
	}
	cp.Country = q
	return nil
}

func (cp *CountryParams) SetParams(request *resty.Request) error {
	return parseAndSetParams(cp, request)
}

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

type CandleParams struct {
	Symbol     string `path:"symbol" validate:"required"`
	Resolution string `path:"resolution" validate:"required"`
}

func (cp *CandleParams) SetSymbol(symbol string) error {
	if symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	cp.Symbol = symbol
	return nil
}

func (cp *CandleParams) SetResolution(resolution string) error {
	if resolution == "" {
		return fmt.Errorf("resolution is required")
	}
	cp.Resolution = resolution
	return nil
}

type DateParams struct {
	Date      string `query:"date"`
	From      string `query:"from"`
	To        string `query:"to"`
	Countback *int   `query:"countback"`
}

// Date sets the date parameter of the DateParams.
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

// From sets the from parameter of the DateParams.
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

// To sets the to parameter of the DateParams.
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

// Countback sets the countback parameter of the DateParams.
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

func (dp *DateParams) SetParams(request *resty.Request) error {
	return parseAndSetParams(dp, request)
}

func (up *UniversalParams) SetParams(request *resty.Request) error {
	return parseAndSetParams(up, request)
}

func (dk *DateKeyParam) SetParams(request *resty.Request) error {
	return parseAndSetParams(dk, request)
}

func (cp *CandleParams) SetParams(request *resty.Request) error {
	return parseAndSetParams(cp, request)
}
