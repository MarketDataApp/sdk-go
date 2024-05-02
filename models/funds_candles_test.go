package models

import (
	"reflect"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/helpers/dates"
)

var (
	jsonData1 = `{"s":"ok","t":[1672808400,1672894800],"o":[355.43,351.35],"h":[355.43,351.35],"l":[355.43,351.35],"c":[355.43,351.35]}`
	jsonData2 = `{"s":"ok","t":[1704344400,1704430800],"o":[432.65,433.44],"h":[432.65,433.44],"l":[432.65,433.44],"c":[432.65,433.44]}`
)

func TestFundUnpack(t *testing.T) {
	testCases := []struct {
		name     string
		jsonData string
		expected []Candle
	}{
		{
			name:     "jsonData1",
			jsonData: jsonData1,
			expected: []Candle{
				{
					Date:  time.Unix(1672808400, 0),
					Open:  355.43,
					High:  355.43,
					Low:   355.43,
					Close: 355.43,
				},
				{
					Date:  time.Unix(1672894800, 0),
					Open:  351.35,
					High:  351.35,
					Low:   351.35,
					Close: 351.35,
				},
			},
		},
		{
			name:     "jsonData2",
			jsonData: jsonData2,
			expected: []Candle{
				{
					Date:  time.Unix(1704344400, 0),
					Open:  432.65,
					High:  432.65,
					Low:   432.65,
					Close: 432.65,
				},
				{
					Date:  time.Unix(1704430800, 0),
					Open:  433.44,
					High:  433.44,
					Low:   433.44,
					Close: 433.44,
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Unmarshal the JSON data into a FundCandlesResponse instance
			f := &FundCandlesResponse{}
			err := f.UnmarshalJSON([]byte(tc.jsonData))
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON data: %v", err)
			}

			// Call the Unpack method
			result, err := f.Unpack()
			if err != nil {
				t.Fatalf("Failed to unpack FundCandlesResponse: %v", err)
			}

			// Check if the results match the expected results
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Unpack did not return the expected result in %s. Got: %+v, want: %+v", tc.name, result, tc.expected)
			}
		})
	}
}

func TestFundPruneAtIndex(t *testing.T) {
	// Test cases
	testCases := []struct {
		jsonData string
		index    int
		expected *FundCandlesResponse
	}{
		{
			jsonData: jsonData1,
			index:    1,
			expected: &FundCandlesResponse{
				Date:  []int64{1672808400},
				Open:  []float64{355.43},
				High:  []float64{355.43},
				Low:   []float64{355.43},
				Close: []float64{355.43},
			},
		},
		{
			jsonData: jsonData2,
			index:    1,
			expected: &FundCandlesResponse{
				Date:  []int64{1704344400},
				Open:  []float64{432.65},
				High:  []float64{432.65},
				Low:   []float64{432.65},
				Close: []float64{432.65},
			},
		},
	}

	for _, tc := range testCases {
		// Unmarshal the JSON data into a StockCandlesResponse instance
		s := &FundCandlesResponse{}
		err := s.UnmarshalJSON([]byte(tc.jsonData))
		if err != nil {
			t.Fatalf("Failed to unmarshal JSON data: %v", err)
		}

		// Call the method with the specified index
		s.pruneIndices(tc.index)

		// Get the DateRange from the StockCandlesResponse instance
		dateRange, err := s.GetDateRange()
		if err != nil {
			t.Fatalf("Failed to get DateRange: %v", err)
		}

		// Get the DateRange from the expected StockCandlesResponse struct
		expectedDateRange, err := tc.expected.GetDateRange()
		if err != nil {
			t.Fatalf("Failed to get expected DateRange: %v", err)
		}

		// Check if the results match the expected results
		if !reflect.DeepEqual(dateRange, expectedDateRange) {
			t.Errorf("PruneAtIndex did not prune correctly. Got: %+v, want: %+v", dateRange, expectedDateRange)
		}
	}
}

func TestFundPruneBeforeIndex(t *testing.T) {
	// Initialize a StockCandlesResponse instance
	f := &FundCandlesResponse{
		Date:  []int64{1, 2, 3, 4, 5},
		Open:  []float64{1.1, 2.2, 3.3, 4.4, 5.5},
		High:  []float64{1.1, 2.2, 3.3, 4.4, 5.5},
		Low:   []float64{1.1, 2.2, 3.3, 4.4, 5.5},
		Close: []float64{1.1, 2.2, 3.3, 4.4, 5.5},
	}

	// Call the method with index 2
	f.pruneBeforeIndex(2)

	// Check the results
	if len(f.Date) != 2 || f.Date[0] != 4 {
		t.Errorf("Time was incorrect, got: %v, want: %v.", f.Date, []int64{4, 5})
	}
	if len(f.Open) != 2 || f.Open[0] != 4.4 {
		t.Errorf("Open was incorrect, got: %v, want: %v.", f.Open, []float64{4.4, 5.5})
	}
}

func TestFundPruneAfterIndex(t *testing.T) {
	// Initialize a StockCandlesResponse instance
	f := &FundCandlesResponse{
		Date:  []int64{1, 2, 3, 4, 5},
		Open:  []float64{1.1, 2.2, 3.3, 4.4, 5.5},
		High:  []float64{1.1, 2.2, 3.3, 4.4, 5.5},
		Low:   []float64{1.1, 2.2, 3.3, 4.4, 5.5},
		Close: []float64{1.1, 2.2, 3.3, 4.4, 5.5},
	}

	// Call the method with index 2
	f.pruneAfterIndex(2)

	// Check the results
	if len(f.Date) != 2 || f.Date[1] != 2 {
		t.Errorf("Time was incorrect, got: %v, want: %v.", f.Date, []int64{1, 2})
	}
	if len(f.Open) != 2 || f.Open[1] != 2.2 {
		t.Errorf("Open was incorrect, got: %v, want: %v.", f.Open, []float64{1.1, 2.2})
	}
	// Continue for the rest of the fields...
}

func TestFundPruneOutsideDateRange(t *testing.T) {
	// JSON data
	data := []byte(`{
		"s": "ok",
		"t": [946684800, 978307200, 1009843200, 1041379200, 1072915200],
		"o": [1.1, 2.2, 3.3, 4.4, 5.5],
		"h": [1.1, 2.2, 3.3, 4.4, 5.5],
		"l": [1.1, 2.2, 3.3, 4.4, 5.5],
		"c": [1.1, 2.2, 3.3, 4.4, 5.5]
	}`)

	// Initialize a StockCandlesResponse instance and unmarshal the JSON data into it
	s := &FundCandlesResponse{}
	err := s.UnmarshalJSON(data)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Define a date range
	dr, err := dates.NewDateRange(978307200, 1041379200)
	if err != nil {
		t.Fatalf("Failed to create DateRange: %v", err)
	}

	// Call the method with the date range
	err = s.PruneOutsideDateRange(*dr)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check the results
	if len(s.Date) != 3 || s.Date[0] != 978307200 || s.Date[2] != 1041379200 {
		t.Errorf("Time was incorrect, got: %v, want: %v.", s.Date, []int64{978307200, 1009843200, 1041379200})
	}
	if len(s.Open) != 3 || s.Open[0] != 2.2 || s.Open[2] != 4.4 {
		t.Errorf("Open was incorrect, got: %v, want: %v.", s.Open, []float64{2.2, 3.3, 4.4})
	}
	// Continue for the rest of the fields...
}

func TestFundUnmarshalJSON(t *testing.T) {
	// Define a JSON string that represents a StockCandlesResponse object
	jsonStr := `{
		"s": "ok",
		"t": [1699462680,1699462740,1699462800,1699462860,1699462920,1699462980,1699463040,1699463100,1699463160,1699463220],
		"o": [182.09,182.04,182.02,181.94,181.9698,181.95,181.9197,181.94,181.9499,181.9703],
		"h": [182.095,182.07,182.07,181.9799,182.04,181.97,182.02,182,182.07,181.9999],
		"l": [182.01,182,181.89,181.855,181.915,181.87,181.91,181.9101,181.89,181.68],
		"c": [182.03,182.01,181.947,181.9699,181.9592,181.9203,181.93,181.94,181.98,181.695]
	}`

	// Create a new StockCandlesResponse object
	sc := &FundCandlesResponse{}

	// Unmarshal the JSON into the StockCandlesResponse object
	err := sc.UnmarshalJSON([]byte(jsonStr))
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check if the DateRange is set correctly
	minTime := time.Unix(1699462680, 0)
	maxTime := time.Unix(1699463220, 0)
	expectedDateRange := dates.DateRange{StartDate: minTime, EndDate: maxTime}

	dateRange, err := sc.GetDateRange()
	if err != nil {
		t.Fatalf("Failed to get DateRange: %v", err)
	}
	if dateRange != expectedDateRange {
		t.Errorf("DateRange was incorrect, got: %v, want: %v.", dateRange, expectedDateRange)
	}
}

func TestFundMarshalJSON(t *testing.T) {
	// Initialize a StockCandlesResponse instance
	f := &FundCandlesResponse{
		Date:  []int64{1672808400, 1672894800},
		Open:  []float64{355.43, 351.35},
		High:  []float64{355.43, 351.35},
		Low:   []float64{355.43, 351.35},
		Close: []float64{355.43, 351.35},
	}
	// Call the method
	data, err := f.MarshalJSON()
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Check the results
	expected := jsonData1
	if string(data) != expected {
		t.Errorf("MarshalJSON did not return the expected result. Got: %s, want: %s", string(data), expected)
	}
}

func TestFundCombineStockCandlesResponse(t *testing.T) {
	// Test cases
	testCases := []struct {
		name      string
		jsonData1 string
		jsonData2 string
		shouldErr bool
	}{
		{
			name:      "Non-overlapping time ranges VFINX",
			jsonData1: jsonData1,
			jsonData2: jsonData2,
			shouldErr: false,
		},
	}

	for _, tc := range testCases {
		// Unmarshal the JSON data into StockCandlesResponse instances
		f1 := &FundCandlesResponse{}
		err := f1.UnmarshalJSON([]byte(tc.jsonData1))
		if err != nil {
			t.Fatalf("Failed to unmarshal JSON data: %v", err)
		}

		f2 := &FundCandlesResponse{}
		err = f2.UnmarshalJSON([]byte(tc.jsonData2))
		if err != nil {
			t.Fatalf("Failed to unmarshal JSON data: %v", err)
		}

		// Try to combine the StockCandlesResponse instances
		_, err = CombineFundCandles(f1, f2)
		if (err != nil) != tc.shouldErr {
			t.Errorf("Test case %s: CombineStockCandlesResponse() error = %v, wantErr %v", tc.name, err, tc.shouldErr)
		}
	}
}

func TestFundCheckTimeInAscendingOrder(t *testing.T) {
	// Test cases
	testCases := []struct {
		date      []int64
		shouldErr bool
	}{
		{
			date:      []int64{1, 2, 3, 4, 5},
			shouldErr: false, // Time is in ascending order
		},
		{
			date:      []int64{1, 3, 2, 4, 5},
			shouldErr: true, // Time is not in ascending order
		},
	}

	for _, tc := range testCases {
		// Create a FundCandlesResponse instance with the specified time
		f := &FundCandlesResponse{
			Date: tc.date,
		}

		// Call the method
		err := f.checkTimeInAscendingOrder()

		// Check if the result matches the expected result
		if (err != nil) != tc.shouldErr {
			t.Errorf("checkTimeInAscendingOrder() error = %v, wantErr %v", err, tc.shouldErr)
		}
	}
}
