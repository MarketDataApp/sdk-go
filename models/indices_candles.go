package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/iancoleman/orderedmap"
)

type IndicesCandlesResponse struct {
	Time  []int64   `json:"t" human:"Date"`
	Open  []float64 `json:"o" human:"Open"`
	High  []float64 `json:"h" human:"High"`
	Low   []float64 `json:"l" human:"Low"`
	Close []float64 `json:"c" human:"Close"`
}

func (icr *IndicesCandlesResponse) String() string {
	return fmt.Sprintf("Time: %v, Open: %v, High: %v, Low: %v, Close: %v",
		icr.Time, icr.Open, icr.High, icr.Low, icr.Close)
}

func (icr *IndicesCandlesResponse) checkTimeInAscendingOrder() error {
	for i := 1; i < len(icr.Time); i++ {
		if icr.Time[i] < icr.Time[i-1] {
			return fmt.Errorf("time is not in ascending order")
		}
	}
	return nil
}

// checkForEqualSlices checks if all slices in the IndicesCandlesResponse struct have the same length.
// It returns an error if the lengths are not equal.
func (icr *IndicesCandlesResponse) checkForEqualSlices() error {
	// Create a slice of the lengths of the Time, Open, High, Low, and Close slices
	lengths := []int{
		len(icr.Time),
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

func (icr *IndicesCandlesResponse) checkForEmptySlices() error {
	// Check if any of the slices are empty
	if len(icr.Time) == 0 || len(icr.Open) == 0 || len(icr.High) == 0 || len(icr.Low) == 0 || len(icr.Close) == 0 {
		return fmt.Errorf("one or more slices are empty")
	}

	// If none of the slices are empty, return nil
	return nil
}

type IndexCandle struct {
	Time  time.Time
	Open  float64
	High  float64
	Low   float64
	Close float64
}

func (ic IndexCandle) String() string {
	loc, _ := time.LoadLocation("America/New_York")
	return fmt.Sprintf("Time: %s, Open: %v, High: %v, Low: %v, Close: %v",
		ic.Time.In(loc).Format("2006-01-02 15:04:05 Z07:00"), ic.Open, ic.High, ic.Low, ic.Close)
}

func (icr *IndicesCandlesResponse) Unpack() ([]IndexCandle, error) {
	if err := icr.checkForEqualSlices(); err != nil {
		return nil, err
	}

	var indexCandles []IndexCandle
	for i := range icr.Time {
		indexCandle := IndexCandle{
			Time:  time.Unix(icr.Time[i], 0),
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
// The method returns the JSON object as a byte slice and any error encountered during the marshaling process.
func (icr *IndicesCandlesResponse) MarshalJSON() ([]byte, error) {
	// Create a new ordered map
	o := orderedmap.New()

	// Set the "s" key to "ok"
	o.Set("s", "ok")

	// Set the "t", "o", "h", "l", and "c" keys to the corresponding slices in the struct
	o.Set("t", icr.Time)
	o.Set("o", icr.Open)
	o.Set("h", icr.High)
	o.Set("l", icr.Low)
	o.Set("c", icr.Close)

	// Marshal the ordered map into a JSON object and return the result
	return json.Marshal(o)
}

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

func (icr *IndicesCandlesResponse) IsValid() bool {
	if err := icr.Validate(); err != nil {
		return false
	}
	return true
}
