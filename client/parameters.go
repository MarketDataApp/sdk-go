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

// Date sets the date parameter for the TickersRequest.
func (dk *DateKeyParam) SetDateKey(q interface{}) error {
	dateString, err := dates.ToDayString(q)
	if err != nil {
		return err
	} else {
		dk.DateKey = dateString
	}
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
	date, err := DecodeDate(q)
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
	date, err := DecodeDate(q)
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
	date, err := DecodeDate(q)
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
