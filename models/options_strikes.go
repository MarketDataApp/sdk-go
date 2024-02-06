package models

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/iancoleman/orderedmap"
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
	Expiration time.Time // Expiration is the date and time when the option expires.
	Strikes    []float64 // Strikes is a slice of strike prices available for the option.
}

// OptionsStrikesResponse encapsulates the response structure for a request to retrieve option strikes.
type OptionsStrikesResponse struct {
	Updated int                    `json:"updated"` // Updated is a UNIX timestamp indicating when the data was last updated.
	Strikes *orderedmap.OrderedMap `json:"-"`       // Strikes is a map where each key is a date string and the value is a slice of strike prices for that date.
}

// UnmarshalJSON custom unmarshals the JSON data into the OptionsStrikesResponse struct.
//
// Parameters:
//   - data []byte: The JSON data to be unmarshaled.
//
// Returns:
//   - error: An error if unmarshaling fails, nil otherwise.
//
// UnmarshalJSON custom unmarshals the JSON data into the OptionsStrikesResponse struct.
func (osr *OptionsStrikesResponse) UnmarshalJSON(data []byte) error {
	var aux map[string]interface{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Initialize the OrderedMap
	osr.Strikes = orderedmap.New()

	// Extract and sort the keys (excluding "updated" and "s")
	var keys []string
	for key := range aux {
		if key != "updated" && key != "s" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys) // Sorts the keys alphabetically, which works well for ISO 8601 dates

	// Iterate over the sorted keys and add them to the OrderedMap
	for _, key := range keys {
		values, ok := aux[key].([]interface{})
		if !ok {
			return fmt.Errorf("unexpected type for key %s", key)
		}
		floats := make([]float64, len(values))
		for i, v := range values {
			floatVal, ok := v.(float64)
			if !ok {
				return fmt.Errorf("unexpected type for value in key %s", key)
			}
			floats[i] = floatVal
		}
		osr.Strikes.Set(key, floats)
	}

	// Handle the "updated" key separately
	if updated, ok := aux["updated"].(float64); ok {
		osr.Updated = int(updated)
	} else {
		return fmt.Errorf("unexpected type or missing 'updated' key")
	}

	return nil
}

// String returns a string representation of the OptionsStrikes struct.
//
// Returns:
//   - string: The string representation of the OptionsStrikes.
func (os OptionsStrikes) String() string {
	loc, _ := time.LoadLocation("America/New_York")
	dateStr := os.Expiration.In(loc).Format("2006-01-02")

	// Convert each strike price to a string with two decimal places
	var strikesStrParts []string
	for _, strike := range os.Strikes {
		strikesStrParts = append(strikesStrParts, fmt.Sprintf("%.2f", strike))
	}
	// Join the formatted strike prices with a space
	strikesStr := strings.Join(strikesStrParts, " ")

	return fmt.Sprintf("OptionsStrikes{Expiration: %s, Strikes: [%s]}", dateStr, strikesStr)
}

// IsValid checks if the OptionsStrikesResponse is valid by leveraging the Validate method.
//
// Returns:
//   - bool: True if the response is valid, false otherwise.
func (osr *OptionsStrikesResponse) IsValid() bool {
	return osr.Validate() == nil
}

// Validate checks if the OptionsStrikesResponse is valid.
//
// Returns:
//   - error: An error if the response is not valid, nil otherwise.
func (osr *OptionsStrikesResponse) Validate() error {
	if len(osr.Strikes.Keys()) == 0 {
		return fmt.Errorf("invalid OptionsStrikesResponse: no strikes data")
	}
	return nil
}

// String returns a string representation of the OptionsStrikesResponse struct.
//
// Returns:
//   - string: The string representation of the OptionsStrikesResponse.
func (osr *OptionsStrikesResponse) String() string {
	var sb strings.Builder
	sb.WriteString("OptionsStrikesResponse{Strikes: [")
	first := true
	for _, key := range osr.Strikes.Keys() {
		if !first {
			sb.WriteString(", ")
		}
		first = false
		value, _ := osr.Strikes.Get(key)
		strikes, _ := value.([]float64) // Assuming the type assertion is always successful.

		// Convert strike prices to strings and join them
		var strikeStrs []string
		for _, strike := range strikes {
			strikeStrs = append(strikeStrs, fmt.Sprintf("%.2f", strike))
		}
		strikesStr := strings.Join(strikeStrs, " ")

		sb.WriteString(fmt.Sprintf("%s:[%s]", key, strikesStr))
	}
	sb.WriteString(fmt.Sprintf("], Updated: %d}", osr.Updated))
	return sb.String()
}

// Unpack converts the ordered map of strikes in the response to a slice of OptionsStrikes.
func (osr *OptionsStrikesResponse) Unpack() ([]OptionsStrikes, error) {
	if err := osr.Validate(); err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return nil, fmt.Errorf("error loading location: %v", err)
	}

	var unpackedStrikes []OptionsStrikes
	for _, key := range osr.Strikes.Keys() {
		value, _ := osr.Strikes.Get(key)
		strikes := value.([]float64)

		date, err := time.ParseInLocation("2006-01-02", key, loc)
		if err != nil {
			return nil, fmt.Errorf("error parsing date %s: %v", key, err)
		}
		date = time.Date(date.Year(), date.Month(), date.Day(), 16, 0, 0, 0, loc)

		unpackedStrikes = append(unpackedStrikes, OptionsStrikes{
			Expiration: date,
			Strikes:    strikes,
		})
	}
	return unpackedStrikes, nil
}
