package parameters

import (
	"fmt"
	"strings"

	"github.com/MarketDataApp/sdk-go/helpers/dates"
	"github.com/go-resty/resty/v2"
)

// OptionParams represents the parameters for an options request.
type OptionParams struct {
	Expiration         string  `query:"expiration" validate:"optional"` // ISO 8601, unix, spreadsheet, or "all"
	Month              int     `query:"month" validate:"optional"`      // 1-12
	Year               int     `query:"year" validate:"optional"`
	Weekly             *bool   `query:"weekly" validate:"optional"`
	Monthly            *bool   `query:"monthly" validate:"optional"`
	Quarterly          *bool   `query:"quarterly" validate:"optional"`
	Nonstandard        *bool   `query:"nonstandard" validate:"optional"`
	DTE                *int    `query:"dte" validate:"optional"` // Days to expiry
	Delta              float64 `query:"delta" validate:"optional"`
	Side               string  `query:"side" validate:"optional"`  // "call" or "put"
	Range              string  `query:"range" validate:"optional"` // "itm", "otm", "atm", "all"
	Strike             float64 `query:"strike" validate:"optional"`
	StrikeLimit        int     `query:"strikeLimit" validate:"optional"`
	MinOpenInterest    int     `query:"minOpenInterest" validate:"optional"`
	MinVolume          int     `query:"minVolume" validate:"optional"`
	MaxBidAskSpread    float64 `query:"maxBidAskSpread" validate:"optional"`
	MaxBidAskSpreadPct float64 `query:"maxBidAskSpreadPct" validate:"optional"` // Percent relative to the underlying
}

// SetExpiration sets the Expiration parameter for the OptionParams.
// It checks if the input is the special case "all" (case insensitive) and sets it directly.
// Otherwise, it validates the expiration date using the ToDayString function from the dates package.
// If the validation fails, it returns an error.
func (op *OptionParams) SetExpiration(q interface{}) error {
	if str, ok := q.(string); ok && strings.EqualFold(str, "all") {
		op.Expiration = str
		return nil
	}

	expiration, err := dates.ToDayString(q)
	if err != nil {
		return err
	}
	op.Expiration = expiration
	return nil
}

// SetMonth sets the Month parameter for the OptionParams.
// It validates that the month is within the 1-12 range.
// If the validation fails, it returns an error.
func (op *OptionParams) SetMonth(month int) error {
	if month < 1 || month > 12 {
		return fmt.Errorf("month must be between 1 and 12")
	}
	op.Month = month
	return nil
}

// SetYear sets the Year parameter for the OptionParams.
// It validates that the year is after 2000.
// If the validation fails, it returns an error.
func (op *OptionParams) SetYear(year int) error {
	if year <= 2000 {
		return fmt.Errorf("year must be after 2000")
	}
	op.Year = year
	return nil
}

// SetWeekly sets the Weekly parameter for the OptionParams.
func (op *OptionParams) SetWeekly(weekly bool) error {
	op.Weekly = &weekly
	return nil
}

// SetMonthly sets the Monthly parameter for the OptionParams.
func (op *OptionParams) SetMonthly(monthly bool) error {
	op.Monthly = &monthly
	return nil
}

// SetQuarterly sets the Quarterly parameter for the OptionParams.
func (op *OptionParams) SetQuarterly(quarterly bool) error {
	op.Quarterly = &quarterly
	return nil
}

// SetNonstandard sets the Nonstandard parameter for the OptionParams.
func (op *OptionParams) SetNonstandard(nonstandard bool) error {
	op.Nonstandard = &nonstandard
	return nil
}

// SetDTE sets the Days to Expiry parameter for the OptionParams.
// It validates that the DTE is not negative.
func (op *OptionParams) SetDTE(dte int) error {
	if dte < 0 {
		return fmt.Errorf("DTE cannot be negative")
	}
	dtePtr := &dte  // Create a pointer to dte
	op.DTE = dtePtr // Assign the pointer to op.DTE
	return nil
}

// SetDelta sets the Delta parameter for the OptionParams.
func (op *OptionParams) SetDelta(delta float64) error {
	op.Delta = delta
	return nil
}

// SetSide sets the Side parameter for the OptionParams allowing case-insensitive input.
func (op *OptionParams) SetSide(side string) error {
	sideLower := strings.ToLower(side)
	if sideLower != "call" && sideLower != "put" && sideLower != "" {
		return fmt.Errorf("side must be 'call', 'put', or an empty string")
	}
	op.Side = sideLower
	return nil
}

// SetRange sets the Range parameter for the OptionParams allowing case-insensitive input.
func (op *OptionParams) SetRange(rangeParam string) error {
	rangeParamLower := strings.ToLower(rangeParam)
	if rangeParamLower != "itm" && rangeParamLower != "otm" && rangeParamLower != "atm" && rangeParamLower != "all" && rangeParamLower != "" {
		return fmt.Errorf("range must be 'itm', 'otm', 'atm', 'all', or an empty string")
	}
	op.Range = rangeParamLower
	return nil
}

// SetStrike sets the Strike parameter for the OptionParams.
// It validates that the strike is between >0 and <99999.99.
func (op *OptionParams) SetStrike(strike float64) error {
	if strike <= 0 || strike >= 99999.99 {
		return fmt.Errorf("strike must be between >0 and <99999.99")
	}
	op.Strike = strike
	return nil
}

// SetStrikeLimit sets the StrikeLimit parameter for the OptionParams.
// It validates that the StrikeLimit is not negative.
func (op *OptionParams) SetStrikeLimit(strikeLimit int) error {
	if strikeLimit < 0 {
		return fmt.Errorf("strikeLimit cannot be negative")
	}
	op.StrikeLimit = strikeLimit
	return nil
}

// SetMinOpenInterest sets the MinOpenInterest parameter for the OptionParams.
// It validates that the MinOpenInterest is not negative.
func (op *OptionParams) SetMinOpenInterest(minOpenInterest int) error {
	if minOpenInterest < 0 {
		return fmt.Errorf("minOpenInterest cannot be negative")
	}
	op.MinOpenInterest = minOpenInterest
	return nil
}

// SetMinVolume sets the MinVolume parameter for the OptionParams.
// It validates that the MinVolume is not negative.
func (op *OptionParams) SetMinVolume(minVolume int) error {
	if minVolume < 0 {
		return fmt.Errorf("minVolume cannot be negative")
	}
	op.MinVolume = minVolume
	return nil
}

// SetMaxBidAskSpread sets the MaxBidAskSpread parameter for the OptionParams.
// It validates that the MaxBidAskSpread is not negative.
func (op *OptionParams) SetMaxBidAskSpread(maxBidAskSpread float64) error {
	if maxBidAskSpread < 0 {
		return fmt.Errorf("maxBidAskSpread cannot be negative")
	}
	op.MaxBidAskSpread = maxBidAskSpread
	return nil
}

// SetMaxBidAskSpreadPct sets the MaxBidAskSpreadPct parameter for the OptionParams.
// It validates that the MaxBidAskSpreadPct is not negative.
func (op *OptionParams) SetMaxBidAskSpreadPct(maxBidAskSpreadPct float64) error {
	if maxBidAskSpreadPct < 0 {
		return fmt.Errorf("maxBidAskSpreadPct cannot be negative")
	}
	op.MaxBidAskSpreadPct = maxBidAskSpreadPct
	return nil
}

// SetParams sets the parameters for the OptionParams.
// If the parsing and setting of parameters fail, it returns an error.
func (op *OptionParams) SetParams(request *resty.Request) error {
	return ParseAndSetParams(op, request)
}
