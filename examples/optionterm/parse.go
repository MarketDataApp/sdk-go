// Parsing for the lookup input: turning a human-typed query like
// "AAPL 2027-01-15 230 call" into the arguments lookupContract (fetch.go)
// needs to resolve an OCC option symbol via the options/lookup endpoint.
package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
)

// lookupDateLayout is the expiration date format accepted by parseLookup:
// the same YYYY-MM-DD layout the rest of the app uses for expirations.
const lookupDateLayout = "2006-01-02"

// parseLookup parses a whitespace-separated lookup query of the form
// "UNDERLYING DATE STRIKE call|put" (e.g. "AAPL 2027-01-15 230 call") into
// the arguments lookupContract needs. The underlying symbol is uppercased;
// the option side is matched case-insensitively against "call"/"put" and
// mapped to [options.Call]/[options.Put]; the strike must parse as a
// positive number; the date must match lookupDateLayout.
//
// now is accepted for future-proofing (e.g. a later UI warning about
// querying a stale expiration) but does not affect parsing today: the
// options/lookup endpoint is the authority on whether a contract exists
// for a given date, so parseLookup does not reject past dates itself.
//
// All returned errors are plain [fmt.Errorf] values meant to drive a
// usage hint in the UI; they are never sent to the API.
func parseLookup(s string, now time.Time) (underlying string, exp time.Time, strike float64, typ options.OptionType, err error) {
	fields := strings.Fields(s)
	if len(fields) != 4 {
		return "", time.Time{}, 0, "", fmt.Errorf(
			"optionterm: lookup query must have 4 fields (underlying date strike call|put), got %d: %q",
			len(fields), s)
	}

	underlying = strings.ToUpper(fields[0])

	exp, err = time.Parse(lookupDateLayout, fields[1])
	if err != nil {
		return "", time.Time{}, 0, "", fmt.Errorf("optionterm: invalid expiration date %q (want YYYY-MM-DD): %w", fields[1], err)
	}

	strike, err = strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return "", time.Time{}, 0, "", fmt.Errorf("optionterm: invalid strike %q: %w", fields[2], err)
	}
	// ParseFloat accepts "NaN" and "Inf" without error, and NaN escapes
	// the positivity check below (every comparison with NaN is false), so
	// both need explicit rejection.
	if math.IsNaN(strike) || math.IsInf(strike, 0) || strike <= 0 {
		return "", time.Time{}, 0, "", fmt.Errorf("optionterm: invalid strike %q: must be a positive finite number", fields[2])
	}

	switch strings.ToLower(fields[3]) {
	case "call":
		typ = options.Call
	case "put":
		typ = options.Put
	default:
		return "", time.Time{}, 0, "", fmt.Errorf("optionterm: invalid option side %q: must be call or put", fields[3])
	}

	return underlying, exp, strike, typ, nil
}
