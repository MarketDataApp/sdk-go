package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/iancoleman/orderedmap"
)

// IndicesCandlesResponse represents the response structure for indices candles data.
// It includes slices for time, open, high, low, and close values of the indices.
type IndicesCandlesResponse struct {
	Date  []int64   `json:"t" human:"Date"`
	Open  []float64 `json:"o" human:"Open"`
	High  []float64 `json:"h" human:"High"`
	Low   []float64 `json:"l" human:"Low"`
	Close []float64 `json:"c" human:"Close"`
}

// String returns a string representation of the IndicesCandlesResponse.
//
// Returns:
//   - A formatted string containing the time, open, high, low, and close values.
func (icr *IndicesCandlesResponse) String() string {
	return fmt.Sprintf("IndicesCandlesResponse{Time: %v, Open: %v, High: %v, Low: %v, Close: %v}",
		icr.Date, icr.Open, icr.High, icr.Low, icr.Close)
}

// checkTimeInAscendingOrder checks if the times in the IndicesCandlesResponse are in ascending order.
//
// Returns:
//   - An error if the times are not in ascending order, nil otherwise.
func (icr *IndicesCandlesResponse) checkTimeInAscendingOrder() error {
	for i := 1; i < len(icr.Date); i++ {
		if icr.Date[i] < icr.Date[i-1] {
			return fmt.Errorf("time is not in ascending order")
		}
	}
	return nil
}

// checkForEqualSlices checks if all slices in the IndicesCandlesResponse struct have the same length.
// It returns an error if the lengths are not equal.
//
// Returns:
//   - An error if the lengths are not equal.
func (icr *IndicesCandlesResponse) checkForEqualSlices() error {
	// Create a slice of the lengths of the Time, Open, High, Low, and Close slices
	lengths := []int{
		len(icr.Date),
		len(icr.Open),
		len(icr.High),
		len(icr.Low),
		len(icr.Close),
	}

	// Check if all lengths in the lengths slice are equal
	// If they are not, return an error
	for i := 1; i < len(lengths); i++ {
		if lengths[i] != lengths[0] {
			return fmt.Errorf("arrays are not of the same length")
		}
	}

	// If all lengths are equal, return nil
	return nil
}

// checkForEmptySlices checks if any of the slices in the IndicesCandlesResponse are empty.
//
// Returns:
//   - An error if one or more slices are empty, nil otherwise.
func (icr *IndicesCandlesResponse) checkForEmptySlices() error {
	// Check if any of the slices are empty
	if len(icr.Date) == 0 || len(icr.Open) == 0 || len(icr.High) == 0 || len(icr.Low) == 0 || len(icr.Close) == 0 {
		return fmt.Errorf("one or more slices are empty")
	}

	// If none of the slices are empty, return nil
	return nil
}

// Unpack converts the IndicesCandlesResponse into a slice of IndexCandle.
//
// Returns:
//   - A slice of IndexCandle, error if there's an inconsistency in the data slices.
func (icr *IndicesCandlesResponse) Unpack() ([]Candle, error) {
	if err := icr.checkForEqualSlices(); err != nil {
		return nil, err
	}

	var indexCandles []Candle
	for i := range icr.Date {
		indexCandle := Candle{
			Date:  time.Unix(icr.Date[i], 0),
			Open:  icr.Open[i],
			High:  icr.High[i],
			Low:   icr.Low[i],
			Close: icr.Close[i],
		}
		indexCandles = append(indexCandles, indexCandle)
	}
	return indexCandles, nil
}

// MarshalJSON is a method on the IndicesCandlesResponse struct.
// It marshals the struct into a JSON object.
// The JSON object is an ordered map with keys "s", "t", "o", "h", "l", and "c".
// The "s" key is always set to "ok".
// The "t", "o", "h", "l", and "c" keys correspond to the Time, Open, High, Low, and Close slices in the struct.
//
// Returns:
//   - A byte slice of the JSON object, error if marshaling fails.
func (icr *IndicesCandlesResponse) MarshalJSON() ([]byte, error) {
	// Create a new ordered map
	o := orderedmap.New()

	// Set the "s" key to "ok"
	o.Set("s", "ok")

	// Set the "t", "o", "h", "l", and "c" keys to the corresponding slices in the struct
	o.Set("t", icr.Date)
	o.Set("o", icr.Open)
	o.Set("h", icr.High)
	o.Set("l", icr.Low)
	o.Set("c", icr.Close)

	// Marshal the ordered map into a JSON object and return the result
	return json.Marshal(o)
}

// UnmarshalJSON custom unmarshals a JSON object into the IndicesCandlesResponse.
//
// Parameters:
//   - data: A byte slice of the JSON object to be unmarshaled.
//
// Returns:
//   - An error if unmarshaling or validation fails.
func (icr *IndicesCandlesResponse) UnmarshalJSON(data []byte) error {
	// Define a secondary type to prevent infinite recursion
	type Alias IndicesCandlesResponse
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(icr),
	}

	// Unmarshal the data into our auxiliary struct
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Call the Validate method
	if err := icr.Validate(); err != nil {
		// Print the contents of the auxiliary struct only if validation fails
		fmt.Println(icr.String())
		return err
	}

	// Return nil if everything went well
	return nil
}

// Validate runs multiple checks on the IndicesCandlesResponse: time in ascending order, equal slice lengths, and no empty slices.
//
// Returns:
//   - An error if any of the checks fail, nil otherwise.
func (icr *IndicesCandlesResponse) Validate() error {
	// Create a channel to handle errors
	errChan := make(chan error, 3)

	// Run each validation function concurrently
	go func() { errChan <- icr.checkTimeInAscendingOrder() }()
	go func() { errChan <- icr.checkForEqualSlices() }()
	go func() { errChan <- icr.checkForEmptySlices() }()

	// Wait for all validation functions to finish
	for i := 0; i < 3; i++ {
		if err := <-errChan; err != nil {
			return err
		}
	}

	// Return nil if there were no errors
	return nil
}

// IsValid checks if the IndicesCandlesResponse passes all validation checks.
//
// Returns:
//   - A boolean indicating if the response is valid.
func (icr *IndicesCandlesResponse) IsValid() bool {
	if err := icr.Validate(); err != nil {
		return false
	}
	return true
}
