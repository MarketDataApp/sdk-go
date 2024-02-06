package models

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/MarketDataApp/sdk-go/helpers/dates"
	"github.com/iancoleman/orderedmap"
)

// StockCandlesResponse represents the JSON response structure for stock candles data.
// It includes slices for time, open, high, low, close prices, and volume for each candle.
// Optional fields VWAP and N are available for V2 candles.
type StockCandlesResponse struct {
	Time   []int64    `json:"t" human:"Date"`                              // Time holds UNIX timestamps for each candle.
	Open   []float64  `json:"o" human:"Open"`                              // Open holds the opening prices for each candle.
	High   []float64  `json:"h" human:"High"`                              // High holds the highest prices reached in each candle.
	Low    []float64  `json:"l" human:"Low"`                               // Low holds the lowest prices reached in each candle.
	Close  []float64  `json:"c" human:"Close"`                             // Close holds the closing prices for each candle.
	Volume []int64    `json:"v" human:"Volume"`                            // Volume represents the trading volume in each candle.
	VWAP   *[]float64 `json:"vwap,omitempty" human:"VWAP,omitempty"`       // VWAP holds the Volume Weighted Average Price for each candle, optional.
	N      *[]int64   `json:"n,omitempty" human:"No. of Trades,omitempty"` // N holds the number of trades for each candle, optional.
}

// StockCandle represents a single candle in a stock candlestick chart.
// It includes the time, open, high, low, close prices, volume, and optionally VWAP and number of trades.
type StockCandle struct {
	Time   time.Time // Time represents the date and time of the candle.
	Open   float64   // Open is the opening price of the candle.
	High   float64   // High is the highest price reached during the candle's time.
	Low    float64   // Low is the lowest price reached during the candle's time.
	Close  float64   // Close is the closing price of the candle.
	Volume int64     // Volume represents the trading volume during the candle's time.
	VWAP   float64   // VWAP is the Volume Weighted Average Price, optional.
	N      int64     // N is the number of trades that occurred, optional.
}

// String returns a string representation of a StockCandle.
//
// Returns:
//   - A string representation of the StockCandle.
func (sc StockCandle) String() string {
	loc, _ := time.LoadLocation("America/New_York")
	if sc.VWAP == 0 && sc.N == 0 {
		return fmt.Sprintf("Time: %s, Open: %v, High: %v, Low: %v, Close: %v, Volume: %v",
			sc.Time.In(loc).Format("2006-01-02 15:04:05 Z07:00"), sc.Open, sc.High, sc.Low, sc.Close, sc.Volume)
	} else {
		return fmt.Sprintf("Time: %s, Open: %v, High: %v, Low: %v, Close: %v, Volume: %v, VWAP: %v, N: %v",
			sc.Time.In(loc).Format("2006-01-02 15:04:05 Z07:00"), sc.Open, sc.High, sc.Low, sc.Close, sc.Volume, sc.VWAP, sc.N)
	}
}

// Unpack converts a StockCandlesResponse into a slice of StockCandle.
//
// Returns:
//   - A slice of StockCandle.
//   - An error if the slices within StockCandlesResponse are not of equal length.
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

// String returns a string representation of a StockCandlesResponse.
//
// Returns:
//   - A string representation of the StockCandlesResponse.
func (s *StockCandlesResponse) String() string {
	// Determine the version of the struct
	version, _ := s.getVersion()

	vwap := "nil"
	n := "nil"
	if s.VWAP != nil {
		vwap = fmt.Sprint(*s.VWAP)
	}
	if s.N != nil {
		n = fmt.Sprint(*s.N)
	}

	if version == 1 {
		return fmt.Sprintf("StockCandlesResponse{Time: %v, Open: %v, High: %v, Low: %v, Close: %v, Volume: %v}",
			s.Time, s.Open, s.High, s.Low, s.Close, s.Volume)
	} else {
		return fmt.Sprintf("StockCandlesResponse{Time: %v, Open: %v, High: %v, Low: %v, Close: %v, Volume: %v, VWAP: %v, N: %v}",
			s.Time, s.Open, s.High, s.Low, s.Close, s.Volume, vwap, n)
	}
}

// checkTimeInAscendingOrder checks if the times in a StockCandlesResponse are in ascending order.
//
// Returns:
//   - An error if the times are not in ascending order.
func (s *StockCandlesResponse) checkTimeInAscendingOrder() error {
	for i := 1; i < len(s.Time); i++ {
		if s.Time[i] < s.Time[i-1] {
			return fmt.Errorf("time is not in ascending order")
		}
	}
	return nil
}

// IsValid checks if a StockCandlesResponse is valid.
//
// Returns:
//   - A boolean indicating if the StockCandlesResponse is valid.
func (s *StockCandlesResponse) IsValid() bool {
	if err := s.Validate(); err != nil {
		return false
	}
	return true
}

// Validate validates a StockCandlesResponse.
//
// Returns:
//   - An error if the StockCandlesResponse is not valid.
func (s *StockCandlesResponse) Validate() error {
	// Create a channel to handle errors
	errChan := make(chan error, 4)

	// Run each validation function concurrently
	go func() { errChan <- s.checkTimeInAscendingOrder() }()
	go func() { errChan <- s.checkForEqualSlices() }()
	go func() { errChan <- s.checkForEmptySlices() }()
	go func() { _, err := s.getVersion(); errChan <- err }()

	// Check for errors from the validation functions
	for i := 0; i < 4; i++ {
		if err := <-errChan; err != nil {
			return err
		}
	}

	return nil
}

// checkForEqualSlices checks if all slices in a StockCandlesResponse have the same length.
//
// Returns:
//   - An error if the slices have different lengths.
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

// checkForEmptySlices checks if any of the slices in a StockCandlesResponse are empty.
//
// Returns:
//   - An error if any of the slices are empty.
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

// getVersion returns the version of the StockCandlesResponse.
//
// Returns:
//   - An integer representing the version.
//   - An error if the version is invalid.
func (s *StockCandlesResponse) getVersion() (int, error) {
	if s.Time != nil && s.Open != nil && s.High != nil && s.Low != nil && s.Close != nil && s.Volume != nil && s.VWAP == nil && s.N == nil {
		return 1, nil
	} else if s.Time != nil && s.Open != nil && s.High != nil && s.Low != nil && s.Close != nil && s.Volume != nil && s.VWAP != nil && s.N != nil && len(*s.VWAP) > 0 && len(*s.N) > 0 {
		return 2, nil
	} else {
		return 0, fmt.Errorf("invalid StockCandle version")
	}
}

// MarshalJSON marshals a StockCandlesResponse into JSON.
//
// Returns:
//   - A byte slice of the JSON representation.
//   - An error if marshaling fails.
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

// UnmarshalJSON unmarshals JSON into a StockCandlesResponse.
//
// Returns:
//   - An error if unmarshaling or validation fails.
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

// GetDateRange returns the date range of a StockCandlesResponse.
//
// Returns:
//   - A DateRange object.
//   - An error if calculating the date range fails.
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

// pruneIndices removes data points at specified indices from a StockCandlesResponse.
//
// Parameters:
//   - indices: A variadic list of integers specifying the indices of data points to remove.
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

// pruneBeforeIndex removes data points before a specified index from a StockCandlesResponse.
//
// Parameters:
//   - index: The index before which all data points will be removed.
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

// pruneAfterIndex removes data points after a specified index from a StockCandlesResponse.
//
// Parameters:
//   - index: The index after which all data points will be removed.
//
// Returns:
//   - An error if the index is out of range.
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

// PruneOutsideDateRange removes data points outside a specified date range from a StockCandlesResponse.
//
// Parameters:
//   - dr: A DateRange struct specifying the start and end dates for the range within which data points should be retained.
//
// Returns:
//   - An error if pruning fails.
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

// getIndex is a method on the StockCandlesResponse struct that searches for a given timestamp within the Time slice.
//
// Parameters:
//   - t int64: The timestamp to search for within the Time slice.
//
// Returns:
//   - int: The index of the first occurrence of the provided timestamp within the Time slice.
//     If the timestamp is not found, it returns the length of the Time slice.
func (s *StockCandlesResponse) getIndex(t int64) int {
	for i, timestamp := range s.Time {
		if timestamp == t {
			return i
		}
	}
	return len(s.Time)
}

// pruneIndex removes the element at the specified index from all slices within the StockCandlesResponse struct.
//
// Parameters:
//   - index int: The index of the element to remove from each slice.
//
// Returns:
//   - error: An error if the index is out of range. Otherwise, returns nil.
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

// CombineStockCandles merges two StockCandlesResponse structs into a single one.
// It checks if the versions of the two structs are the same, ensures there is no time overlap between them,
// and then combines their data into a new StockCandlesResponse struct. If the versions are both V2,
// it also combines the VWAP and N slices. Finally, it validates the combined struct.
//
// Parameters:
//   - s1 *StockCandlesResponse: The first StockCandlesResponse struct to be combined.
//   - s2 *StockCandlesResponse: The second StockCandlesResponse struct to be combined.
//
// Returns:
//   - *StockCandlesResponse: A pointer to the newly combined StockCandlesResponse struct.
//   - error: An error if the versions do not match, there is a time overlap, or the combined struct fails validation.
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
