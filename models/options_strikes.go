package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

/*

Example API Response JSON:
{
	"s":"ok",
  	"updated":1706791474,
	"2024-02-02":[50.0,60.0,65.0,70.0,75.0,80.0,85.0,90.0,95.0,100.0,105.0,110.0,115.0,120.0,125.0,130.0,135.0,140.0,145.0,150.0,152.5,155.0,157.5,160.0,162.5,165.0,167.5,170.0,172.5,175.0,177.5,180.0,182.5,185.0,187.5,190.0,192.5,195.0,197.5,200.0,202.5,205.0,207.5,210.0,212.5,215.0,217.5,220.0,222.5,225.0,230.0,235.0,240.0,245.0,250.0,255.0,260.0,265.0],
	"2024-02-09":[95.0,100.0,105.0,110.0,115.0,120.0,125.0,130.0,135.0,140.0,145.0,150.0,155.0,160.0,162.5,165.0,167.5,170.0,172.5,175.0,177.5,180.0,182.5,185.0,187.5,190.0,192.5,195.0,197.5,200.0,202.5,205.0,207.5,210.0,212.5,215.0,217.5,220.0,222.5,225.0,230.0,235.0,240.0,245.0,250.0,255.0,260.0,265.0],
	"2024-02-16":[50.0,55.0,60.0,65.0,70.0,75.0,80.0,85.0,90.0,95.0,100.0,105.0,110.0,115.0,120.0,125.0,130.0,135.0,140.0,145.0,150.0,155.0,160.0,162.5,165.0,167.5,170.0,172.5,175.0,177.5,180.0,182.5,185.0,187.5,190.0,192.5,195.0,197.5,200.0,202.5,205.0,207.5,210.0,212.5,215.0,217.5,220.0,222.5,225.0,230.0,235.0,240.0,245.0,250.0,255.0,260.0,265.0,270.0,275.0,280.0,285.0,290.0,295.0,300.0,305.0,310.0,315.0,320.0],
	"2024-02-23":[100.0,105.0,110.0,115.0,120.0,125.0,130.0,135.0,140.0,145.0,150.0,155.0,160.0,165.0,170.0,175.0,180.0,185.0,190.0,195.0,200.0,205.0,210.0,215.0,220.0,225.0,230.0,235.0,240.0,245.0,250.0,255.0,260.0,265.0]
}

*/

// OptionsStrikes represents the expiration date and strike prices for an option.
type OptionsStrikes struct {
	Expiration time.Time  // Expiration is the date and time when the option expires.
	Strikes    []float64  // Strikes is a slice of strike prices available for the option.
}

// OptionsStrikesResponse encapsulates the response structure for a request to retrieve option strikes.
type OptionsStrikesResponse struct {
	Updated int                  `json:"updated"` // Updated is a UNIX timestamp indicating when the data was last updated.
	Strikes map[string][]float64 `json:"-"`       // Strikes is a map where each key is a date string and the value is a slice of strike prices for that date.
}

// UnmarshalJSON custom unmarshals the JSON data into the OptionsStrikesResponse struct.
//
// Parameters:
//   - data []byte: The JSON data to be unmarshaled.
//
// Returns:
//   - error: An error if unmarshaling fails, nil otherwise.
func (osr *OptionsStrikesResponse) UnmarshalJSON(data []byte) error {
	var aux map[string]interface{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	osr.Strikes = make(map[string][]float64)
	for key, value := range aux {
		if key != "updated" && key != "s" {
			values := value.([]interface{})
			floats := make([]float64, len(values))
			for i, v := range values {
				floats[i] = v.(float64)
			}
			osr.Strikes[key] = floats
		} else if key == "updated" {
			osr.Updated = int(value.(float64))
		}
	}
	return nil
}

// String returns a string representation of the OptionsStrikes struct.
//
// Returns:
//   - string: The string representation of the OptionsStrikes.
func (os OptionsStrikes) String() string {
	loc, _ := time.LoadLocation("America/New_York")
	dateStr := os.Expiration.In(loc).Format("Jan 02, 2006 15:04 MST")

	var strikesStrBuilder strings.Builder
	strikesStrBuilder.WriteString("[")
	for i, strike := range os.Strikes {
		if i > 0 {
			strikesStrBuilder.WriteString(", ")
		}
		strikesStrBuilder.WriteString(fmt.Sprintf("%.2f", strike))
	}
	strikesStrBuilder.WriteString("]")

	return fmt.Sprintf("Expiration: %s, Strikes: %s", dateStr, strikesStrBuilder.String())
}

// IsValid checks if the OptionsStrikesResponse is valid.
//
// Returns:
//   - bool: True if the response is valid, false otherwise.
func (osr *OptionsStrikesResponse) IsValid() bool {
	if len(osr.Strikes) == 0 {
		return false
	}
	return true
}

// String returns a string representation of the OptionsStrikesResponse struct.
//
// Returns:
//   - string: The string representation of the OptionsStrikesResponse.
func (osr *OptionsStrikesResponse) String() string {
    // First, unpack the response to get a slice of OptionsStrikes
    unpackedStrikes, err := osr.Unpack()
    if err != nil {
        return fmt.Sprintf("Error unpacking strikes: %v", err)
    }

    // Initialize a builder for constructing the output string
    var sb strings.Builder

    // Loop over each OptionsStrikes in the unpacked slice
    for _, strike := range unpackedStrikes {
        // Use the String method of OptionsStrikes to append each to the builder
        sb.WriteString(strike.String() + "; ")
    }

    // Append the "Updated" information last
    sb.WriteString(fmt.Sprintf("Updated: %v", osr.Updated))

    // Return the constructed string
    return sb.String()
}

// Unpack converts the map of strikes in the response to a slice of OptionsStrikes.
//
// Returns:
//   - []OptionsStrikes: A slice of OptionsStrikes constructed from the response.
//   - error: An error if the unpacking fails, nil otherwise.
func (osr *OptionsStrikesResponse) Unpack() ([]OptionsStrikes, error) {
	if !osr.IsValid() {
		return nil, fmt.Errorf("invalid OptionsStrikesResponse")
	}

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return nil, fmt.Errorf("error loading location: %v", err)
	}

	var unpackedStrikes []OptionsStrikes
	for dateStr, strikes := range osr.Strikes {
		// Parse the date in the given location
		date, err := time.ParseInLocation("2006-01-02", dateStr, loc)
		if err != nil {
			return nil, fmt.Errorf("error parsing date %s: %v", dateStr, err)
		}
		// Set the time to 16:00:00
		date = time.Date(date.Year(), date.Month(), date.Day(), 16, 0, 0, 0, loc)

		unpackedStrikes = append(unpackedStrikes, OptionsStrikes{
			Expiration: date,
			Strikes:    strikes,
		})
	}
	return unpackedStrikes, nil
}
