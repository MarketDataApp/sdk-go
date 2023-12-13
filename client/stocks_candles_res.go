package client

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/MarketDataApp/sdk-go/helpers/dates"
	"github.com/iancoleman/orderedmap"
)

// Example JSON:
//
//	{
//	  "s": "ok",
//	  "t": [1699462680,1699462740,1699462800,1699462860,1699462920,1699462980,1699463040,1699463100,1699463160,1699463220],
//	  "o": [182.09,182.04,182.02,181.94,181.9698,181.95,181.9197,181.94,181.9499,181.9703],
//	  "h": [182.095,182.07,182.07,181.9799,182.04,181.97,182.02,182.0,182.07,181.9999],
//	  "l": [182.01,182.0,181.89,181.855,181.915,181.87,181.91,181.9101,181.89,181.68],
//	  "c": [182.03,182.01,181.947,181.9699,181.9592,181.9203,181.93,181.94,181.98,181.695],
//	  "v": [81954,96569,236174,133964,103286,62645,71547,48792,109215,195926]}

type StockCandlesResponse struct {
	Time   []int64    `json:"t" human:"Date"`
	Open   []float64  `json:"o" human:"Open"`
	High   []float64  `json:"h" human:"High"`
	Low    []float64  `json:"l" human:"Low"`
	Close  []float64  `json:"c" human:"Close"`
	Volume []int64    `json:"v" human:"Volume"`
	VWAP   *[]float64 `json:"vwap,omitempty" human:"VWAP,omitempty"`         // Optional, for V2 candles
	N      *[]int64   `json:"n,omitempty" human:"No. of Trades,omitempty"` // Optional, for V2 candles
}

type StockCandle struct {
	Time time.Time
	Open float64
	High   float64 
	Low    float64  
	Close  float64  
	Volume int64   
	VWAP   float64 
	N      int64 

}

func (scr *StockCandlesResponse) Unpack() ([]StockCandle, error) {
	if err := scr.checkForEqualSlices(); err != nil {
		return nil, err
	}

	var stockCandles []StockCandle
	for i := range scr.Time {
		stockCandle := StockCandle{
			Time:   time.Unix(scr.Time[i], 0),
			Open:   scr.Open[i],
			High:   scr.High[i],
			Low:    scr.Low[i],
			Close:  scr.Close[i],
			Volume: scr.Volume[i],
		}
		if scr.VWAP != nil {
			stockCandle.VWAP = (*scr.VWAP)[i]
		}
		if scr.N != nil {
			stockCandle.N = (*scr.N)[i]
		}
		stockCandles = append(stockCandles, stockCandle)
	}
	return stockCandles, nil
}


func (s *StockCandlesResponse) String() string {
	// Determine the version of the struct
	version, _ := s.getVersion()

	if version == 1 {
		return fmt.Sprintf("Time: %v, Open: %v, High: %v, Low: %v, Close: %v, Volume: %v",
			 s.Time, s.Open, s.High, s.Low, s.Close, s.Volume)
	} else {
		vwap := "nil"
		n := "nil"
		if s.VWAP != nil {
			vwap = fmt.Sprint(*s.VWAP)
		}
		if s.N != nil {
			n = fmt.Sprint(*s.N)
		}
		return fmt.Sprintf("Time: %v, Open: %v, High: %v, Low: %v, Close: %v, Volume: %v, VWAP: %v, N: %v",
			s.Time, s.Open, s.High, s.Low, s.Close, s.Volume, vwap, n)
	}
}


func (s *StockCandlesResponse) checkTimeInAscendingOrder() error {
	for i := 1; i < len(s.Time); i++ {
		if s.Time[i] < s.Time[i-1] {
			return fmt.Errorf("time is not in ascending order")
		}
	}
	return nil
}

func (s *StockCandlesResponse) Validate() error {
	// Check if the time is in ascending order
	if err := s.checkTimeInAscendingOrder(); err != nil {
		return err
	}

	// Validate the JSON after unmarshaling
	if err := s.checkForEqualSlices(); err != nil {
		return err
	}

	// Check for empty slices
	if err := s.checkForEmptySlices(); err != nil {
		return err
	}

	// Check the version for errors:
	_, err := s.getVersion()
	if err != nil {
		return err
	}

	return nil

}

// checkForEqualSlices checks if all slices in the StockCandles struct have the same length.
// It returns an error if the lengths are not equal.
// This is important to ensure that each element in a slice corresponds to the same element in the other slices.
// For example, the first element in the Time slice should correspond to the first element in the Open, High, Low, Close, and Volume slices.
// If the Version is 2, it also checks the VWAP and N slices.
func (s *StockCandlesResponse) checkForEqualSlices() error {
	// Create a slice of the lengths of the Time, Open, High, Low, Close, and Volume slices
	lengths := []int{
		len(s.Time),
		len(s.Open),
		len(s.High),
		len(s.Low),
		len(s.Close),
		len(s.Volume),
	}

	// If the Version is 2, add the lengths of the VWAP and N slices to the lengths slice
	version, err := s.getVersion()
	if err != nil {
		return err
	}
	if version == 2 {
		lengths = append(lengths, len(*s.VWAP), len(*s.N))
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

func (s *StockCandlesResponse) checkForEmptySlices() error {
	// Check if any of the slices are empty
	if len(s.Time) == 0 || len(s.Open) == 0 || len(s.High) == 0 || len(s.Low) == 0 || len(s.Close) == 0 || len(s.Volume) == 0 {
		return fmt.Errorf("one or more slices are empty")
	}

	// Use the getVersion method to check if the Version is 2, also check the VWAP and N slices
	version, err := s.getVersion()
	if err != nil {
		return err
	}
	if version == 2 {
		if s.VWAP != nil && len(*s.VWAP) == 0 {
			return fmt.Errorf("slice VWAP is empty")
		}
		if s.N != nil && len(*s.N) == 0 {
			return fmt.Errorf("slice N is empty")
		}
	}

	// If none of the slices are empty, return nil
	return nil
}

// getVersion returns the version of the StockCandles.
// If the version is not 1 or 2, it returns an error.
func (s *StockCandlesResponse) getVersion() (int, error) {
	if s.Time != nil && s.Open != nil && s.High != nil && s.Low != nil && s.Close != nil && s.Volume != nil && s.VWAP == nil && s.N == nil {
		return 1, nil
	} else if s.Time != nil && s.Open != nil && s.High != nil && s.Low != nil && s.Close != nil && s.Volume != nil && s.VWAP != nil && s.N != nil && len(*s.VWAP) > 0 && len(*s.N) > 0 {
		return 2, nil
	} else {
		return 0, fmt.Errorf("invalid StockCandle version")
	}
}

// MarshalJSON is a method on the StockCandles struct.
// It marshals the struct into a JSON object.
// The JSON object is an ordered map with keys "s", "t", "o", "h", "l", "c", "v", "vw", and "n".
// The "s" key is always set to "ok".
// The "t", "o", "h", "l", "c", and "v" keys correspond to the Time, Open, High, Low, Close, and Volume slices in the struct.
// If the Version of the struct is 2, the "vw" and "n" keys are also set, corresponding to the VWAP and N slices in the struct.
// The method returns the JSON object as a byte slice and any error encountered during the marshaling process.
func (s *StockCandlesResponse) MarshalJSON() ([]byte, error) {
	// Create a new ordered map
	o := orderedmap.New()

	// Set the "s" key to "ok"
	o.Set("s", "ok")

	// Set the "t", "o", "h", "l", "c", and "v" keys to the corresponding slices in the struct
	o.Set("t", s.Time)
	o.Set("o", s.Open)
	o.Set("h", s.High)
	o.Set("l", s.Low)
	o.Set("c", s.Close)
	o.Set("v", s.Volume)

	// If the Version of the struct is 2, set the "vw" and "n" keys to the corresponding slices in the struct
	version, err := s.getVersion()
	if err != nil {
		return nil, err
	}
	if version == 2 {
		o.Set("vwap", s.VWAP)
		o.Set("n", s.N)
	}

	// Marshal the ordered map into a JSON object and return the result
	return json.Marshal(o)
}

func (s *StockCandlesResponse) UnmarshalJSON(data []byte) error {
	// Define a secondary type to prevent infinite recursion
	type Alias StockCandlesResponse
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	// Unmarshal the data into our auxiliary struct
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Call the Validate method
	if err := s.Validate(); err != nil {
		// Print the contents of the auxiliary struct only if validation fails
		fmt.Println(s.String())
		return err
	}

	// Return nil if everything went well
	return nil
}

func (s *StockCandlesResponse) GetDateRange() (dates.DateRange, error) {
	// Pass the slice of timestamps directly to Earliest and Latest
	min, err1 := dates.Earliest(s.Time)
	max, err2 := dates.Latest(s.Time)
	if err1 != nil || err2 != nil {
		return dates.DateRange{}, fmt.Errorf("error calculating date ranges: %v, %v", err1, err2)
	}

	// Use NewDateRange to create a new DateRange instance
	dr, err := dates.NewDateRange(min, max)
	if err != nil {
		return dates.DateRange{}, err
	}

	return *dr, nil
}

// pruneIndices is a method on the StockCandles struct.
// It removes the data points at the given indices from the StockCandles instance.
// It modifies the Time, Open, High, Low, Close, and Volume slices to exclude the data points at the given indices.
// If the VWAP and N fields are not nil, it also prunes these slices.
// The indices are sorted in reverse order before pruning to avoid index out of range errors.
// If an index is out of range, it is ignored.
func (s *StockCandlesResponse) pruneIndices(indices ...int) {
	sort.Sort(sort.Reverse(sort.IntSlice(indices)))
	for _, index := range indices {
		if index < 0 || index >= len(s.Time) {
			continue
		}
		s.Time = append(s.Time[:index], s.Time[index+1:]...)
		s.Open = append(s.Open[:index], s.Open[index+1:]...)
		s.High = append(s.High[:index], s.High[index+1:]...)
		s.Low = append(s.Low[:index], s.Low[index+1:]...)
		s.Close = append(s.Close[:index], s.Close[index+1:]...)
		s.Volume = append(s.Volume[:index], s.Volume[index+1:]...)

		if s.VWAP != nil {
			*s.VWAP = append((*s.VWAP)[:index], (*s.VWAP)[index+1:]...)
		}

		if s.N != nil {
			*s.N = append((*s.N)[:index], (*s.N)[index+1:]...)
		}
	}
}

// pruneBeforeIndex removes the data point at the given index and all data points before it from the StockCandles instance.
// It modifies the Time, Open, High, Low, Close, and Volume slices to only include data from the given index onwards.
// If the VWAP and N fields are not nil, it also prunes these slices.
func (s *StockCandlesResponse) pruneBeforeIndex(index int) {
	if index+1 < len(s.Time) {
		s.Time = s.Time[index+1:]
		s.Open = s.Open[index+1:]
		s.High = s.High[index+1:]
		s.Low = s.Low[index+1:]
		s.Close = s.Close[index+1:]
		s.Volume = s.Volume[index+1:]

		if s.VWAP != nil {
			*s.VWAP = (*s.VWAP)[index+1:]
		}

		if s.N != nil {
			*s.N = (*s.N)[index+1:]
		}
	}
}

// pruneAfterIndex removes the data point at the given index and all data points after it from the StockCandles instance.
// It modifies the Time, Open, High, Low, Close, and Volume slices to only include data up to, but not including, the given index.
// If the VWAP and N fields are not nil, it also prunes these slices.
// If the index is out of range, it returns an error.
func (s *StockCandlesResponse) pruneAfterIndex(index int) error {
	// Check if the index is within the range of the slices
	if index < 0 || index >= len(s.Time) {
		return fmt.Errorf("index %d out of range (0-%d)", index, len(s.Time)-1)
	}

	// Prune the Time, Open, High, Low, Close, and Volume slices
	s.Time = s.Time[:index]
	s.Open = s.Open[:index]
	s.High = s.High[:index]
	s.Low = s.Low[:index]
	s.Close = s.Close[:index]
	s.Volume = s.Volume[:index]

	// If the VWAP field is not nil, prune the VWAP slice
	if s.VWAP != nil {
		*s.VWAP = (*s.VWAP)[:index]
	}

	// If the N field is not nil, prune the N slice
	if s.N != nil {
		*s.N = (*s.N)[:index]
	}

	return nil
}

func (s *StockCandlesResponse) PruneOutsideDateRange(dr dates.DateRange) error {
	// Validate all timestamps
	validTimestamps, invalidTimestamps := dr.ValidateTimestamps(s.Time...)
	fmt.Println("Valid timestamps: ", validTimestamps)
	fmt.Println("Invalid timestamps: ", invalidTimestamps)

	// Loop through invalid timestamps, get index and prune
	for _, invalidTimestamp := range invalidTimestamps {
		for {
			index := s.getIndex(invalidTimestamp)
			if index >= len(s.Time) || s.Time[index] != invalidTimestamp {
				break
			}
			s.pruneIndex(index)
		}
	}

	return nil
}

// getIndex is a method on the StockCandles struct.
// It iterates over the Time slice and returns the index of the first occurrence of the provided timestamp.
// If the timestamp is not found in the Time slice, it returns the length of the Time slice.
func (s *StockCandlesResponse) getIndex(t int64) int {
	for i, timestamp := range s.Time {
		if timestamp == t {
			return i
		}
	}
	return len(s.Time)
}

// pruneIndex is a method on the StockCandles struct.
// It removes the element at the specified index from all slices in the struct.
// If the index is out of range, it returns an error.
// If the VWAP or N slices are not nil, it also removes the element at the index from these slices.
func (s *StockCandlesResponse) pruneIndex(index int) error {
	if index < 0 || index >= len(s.Time) {
		return fmt.Errorf("index %d out of range (0-%d)", index, len(s.Time)-1)
	}

	// Remove the element at the index from the Time, Open, High, Low, Close, and Volume slices
	s.Time = append(s.Time[:index], s.Time[index+1:]...)
	s.Open = append(s.Open[:index], s.Open[index+1:]...)
	s.High = append(s.High[:index], s.High[index+1:]...)
	s.Low = append(s.Low[:index], s.Low[index+1:]...)
	s.Close = append(s.Close[:index], s.Close[index+1:]...)
	s.Volume = append(s.Volume[:index], s.Volume[index+1:]...)

	// If the VWAP slice is not nil, remove the element at the index from the VWAP slice
	if s.VWAP != nil {
		*s.VWAP = append((*s.VWAP)[:index], (*s.VWAP)[index+1:]...)
	}

	// If the N slice is not nil, remove the element at the index from the N slice
	if s.N != nil {
		*s.N = append((*s.N)[:index], (*s.N)[index+1:]...)
	}

	return nil
}

func CombineStockCandles(s1, s2 *StockCandlesResponse) (*StockCandlesResponse, error) {
	// Check if versions are the same
	version1, err1 := s1.getVersion()
	if err1 != nil {
		return nil, fmt.Errorf("error getting version from s1: %v", err1)
	}
	version2, err2 := s2.getVersion()
	if err2 != nil {
		return nil, fmt.Errorf("error getting version from s2: %v", err2)
	}
	if version1 != version2 {
		return nil, fmt.Errorf("versions do not match")
	}

	// Check for time overlap using the DoesNotContain method
	s1DateRange, err1 := s1.GetDateRange()
	if err1 != nil {
		return nil, fmt.Errorf("error getting date range from s1: %v", err1)
	}
	s2DateRange, err2 := s2.GetDateRange()
	if err2 != nil {
		return nil, fmt.Errorf("error getting date range from s2: %v", err2)
	}
	if !s1DateRange.DoesNotContain(s2DateRange) && !s2DateRange.DoesNotContain(s1DateRange) {
		return nil, fmt.Errorf("time ranges overlap: s1 range %s, s2 range %s", s1DateRange.String(), s2DateRange.String())
	}

	// Combine the structs
	combined := &StockCandlesResponse{
		Time:   append(s1.Time, s2.Time...),
		Open:   append(s1.Open, s2.Open...),
		High:   append(s1.High, s2.High...),
		Low:    append(s1.Low, s2.Low...),
		Close:  append(s1.Close, s2.Close...),
		Volume: append(s1.Volume, s2.Volume...),
	}

	if version1 == 2 && version2 == 2 {
		combinedVWAP := append(*s1.VWAP, *s2.VWAP...)
		combinedN := append(*s1.N, *s2.N...)
		combined.VWAP = &combinedVWAP
		combined.N = &combinedN
	}

	// Validate the combined struct
	if err := combined.Validate(); err != nil {
		return nil, fmt.Errorf("combineStockCandles validation failed: %v", err)
	}

	// Dereference the old structs to free memory
	s1 = nil
	s2 = nil

	return combined, nil
}

