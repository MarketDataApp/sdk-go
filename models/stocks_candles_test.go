package models

import (
	"reflect"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/helpers/dates"
)

var (
	jsonDataV1    = `{"s":"ok","t":[1699630200,1699630260],"o":[183.78,183.88],"h":[183.93,183.96],"l":[183.76,183.82],"c":[183.8716,183.93],"v":[147185,101982]}`
	jsonDataV2    = `{"s":"ok","t":[1699630200,1699630260],"o":[183.78,183.88],"h":[183.93,183.96],"l":[183.76,183.82],"c":[183.8716,183.93],"v":[147185,101982],"vwap":[185,182],"n":[147185,101982]}`
	jsonDataAAPL1 = `{"s":"ok","t":[1577941200,1578027600,1578286800,1578373200],"o":[74.06,74.2875,73.4475,74.96],"h":[75.15,75.145,74.99,75.225],"l":[73.7975,74.125,73.1875,74.37],"c":[75.0875,74.3575,74.95,74.5975],"v":[135647456,146535512,118578576,111510620]}`
	jsonDataAAPL2 = `{"s":"ok","t":[1698120000,1698206400,1698292800,1698379200,1698638400,1698724800,1698811200,1698897600,1698984000,1699246800,1699333200,1699419600,1699506000,1699592400,1699851600],"o":[173.05,171.88,170.37,166.91,169.02,169.35,171,175.52,174.24,176.38,179.18,182.35,182.96,183.97,185.82],"h":[173.67,173.06,171.3775,168.96,171.17,170.9,174.23,177.78,176.82,179.43,182.44,183.45,184.12,186.565,186.03],"l":[171.45,170.65,165.67,166.83,168.87,167.9,170.12,175.46,173.35,176.21,178.97,181.59,181.81,183.53,184.21],"c":[173.44,171.1,166.89,168.22,170.29,170.77,173.97,177.57,176.65,179.23,181.82,182.89,182.41,186.4,184.8],"v":[43816644,57156962,70625258,58499129,51130955,44846017,56934906,77334752,79829246,63841310,70529966,49340282,53763540,66177922,43627519]}`
	jsonDataSPY1  = `{"s":"ok","t":[1675141200,1675227600,1675314000,1675400400,1675659600,1675746000,1675832400,1675918800,1676005200,1676264400,1676350800,1676437200,1676523600,1676610000,1676955600,1677042000,1677128400,1677214800,1677474000,1677560400],"o":[401.13,405.211,414.86,411.59,409.79,408.87,413.13,414.41,405.86,408.72,411.24,410.35,408.79,406.06,403.06,399.52,401.56,395.42,399.87,397.23],"h":[406.53,413.67,418.31,416.97,411.29,416.49,414.53,414.57,408.44,412.97,415.05,414.06,412.91,407.51,404.16,401.13,402.2,397.25,401.29,399.28],"l":[400.77,402.35,412.88,411.09,408.1,407.57,409.93,405.81,405.01,408.24,408.511,409.47,408.14,404.05,398.82,397.02,396.25,393.64,396.75,396.15],"c":[406.48,410.8,416.78,412.35,409.83,415.19,410.65,407.09,408.04,412.83,412.64,413.98,408.28,407.26,399.09,398.54,400.66,396.38,397.73,396.26],"v":[86655786,101459155,101192713,94577181,60250025,90721745,76227462,78564109,70767615,64903039,87681813,61368279,76418169,89196815,82585674,83500793,96195399,108122913,80251136,96350567],"vwap":[404.1605,407.863,415.52230000000003,413.7699,409.9916,411.9916999999999,411.9252,409.7962,407.0089,411.3867,412.1239,412.2614,410.3213,406.13319999999993,401.1432,398.9103,399.6202,395.8063,398.6304,397.4209],"n":[512619,833210,724694,665739,461542,695517,529228,566607,489777,426598,627801,427499,583412,562615,559126,568095,740995,755245,592300,600687]}`
	jsonDataSPY2  = `{"s":"ok","t":[1672722000,1672808400,1672894800,1672981200,1673240400,1673326800,1673413200,1673499600,1673586000,1673931600,1674018000,1674104400,1674190800,1674450000,1674536400,1674622800,1674709200,1674795600,1675054800,1675141200],"o":[384.37,383.18,381.72,382.61,390.37,387.25,392.23,396.67,393.62,398.48,399.01,389.36,390.1,396.72,398.88,395.95,403.13,403.655,402.8,401.13],"h":[386.43,385.88,381.84,389.25,393.7,390.65,395.6,398.485,399.1,400.23,400.12,391.08,396.04,402.645,401.15,400.7,404.92,408.16,405.13,406.53],"l":[377.831,380,378.76,379.4127,387.67,386.27,391.38,392.42,393.34,397.06,391.28,387.26,388.38,395.72,397.64,393.56,400.03,403.44,400.28,400.77],"c":[380.82,383.76,379.38,388.08,387.86,390.58,395.52,396.96,398.5,397.77,391.49,388.64,395.88,400.63,400.2,400.35,404.75,405.68,400.59,406.48],"v":[74850731,85934098,76275354,104052662,73978071,65298094,68702980,90145699,63853932,62577281,99495062,86601919,91542126,84178797,59469411,84684501,72177425,68249180,74067618,86655786],"vwap":[380.9576,383.1494,380.2625,385.2463,390.3628,389.0824,393.3003,396.31199999999995,396.8377,398.1971,394.55500000000006,389.2674,392.59009999999995,399.7705,399.9358,397.8551,402.6127,405.7891,402.2276,404.1605],"n":[590240,632808,530896,687390,549428,471958,452701,665042,468376,437653,642978,548674,505912,568459,434102,583220,521281,496478,514522,512619]}`
)

func TestUnpack(t *testing.T) {
    testCases := []struct {
        name      string
        jsonData string
        expected []StockCandle
    }{
        {
            name: "jsonDataV1",
            jsonData: jsonDataV1,
            expected: []StockCandle{
                {
                    Time:   time.Unix(1699630200, 0),
                    Open:   183.78,
                    High:   183.93,
                    Low:    183.76,
                    Close:  183.8716,
                    Volume: 147185,
                },
                {
                    Time:   time.Unix(1699630260, 0),
                    Open:   183.88,
                    High:   183.96,
                    Low:    183.82,
                    Close:  183.93,
                    Volume: 101982,
                },
            },
        },
        {
            name: "jsonDataV2",
            jsonData: jsonDataV2,
            expected: []StockCandle{
                {
                    Time:   time.Unix(1699630200, 0),
                    Open:   183.78,
                    High:   183.93,
                    Low:    183.76,
                    Close:  183.8716,
                    Volume: 147185,
                    VWAP:   185,
                    N:      147185,
                },
                {
                    Time:   time.Unix(1699630260, 0),
                    Open:   183.88,
                    High:   183.96,
                    Low:    183.82,
                    Close:  183.93,
                    Volume: 101982,
                    VWAP:   182,
                    N:      101982,
                },
            },
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Unmarshal the JSON data into a StockCandlesResponse instance
            s := &StockCandlesResponse{}
            err := s.UnmarshalJSON([]byte(tc.jsonData))
            if err != nil {
                t.Fatalf("Failed to unmarshal JSON data: %v", err)
            }

            // Call the Unpack method
            result, err := s.Unpack()
            if err != nil {
                t.Fatalf("Failed to unpack StockCandlesResponse: %v", err)
            }

            // Check if the results match the expected results
            if !reflect.DeepEqual(result, tc.expected) {
                t.Errorf("Unpack did not return the expected result in %s. Got: %+v, want: %+v", tc.name, result, tc.expected)
            }
        })
    }
}

func TestPruneAtIndex(t *testing.T) {
	// Test cases
	testCases := []struct {
		jsonData string
		index    int
		expected *StockCandlesResponse
	}{
		{
			jsonData: jsonDataV1,
			index:    1,
			expected: &StockCandlesResponse{
				Time:   []int64{1699630200},
				Open:   []float64{183.78},
				High:   []float64{183.93},
				Low:    []float64{183.76},
				Close:  []float64{183.8716},
				Volume: []int64{147185},
				VWAP:   nil,
				N:      nil,
			},
		},
		{
			jsonData: jsonDataV2,
			index:    1,
			expected: &StockCandlesResponse{
				Time:   []int64{1699630200},
				Open:   []float64{183.78},
				High:   []float64{183.93},
				Low:    []float64{183.76},
				Close:  []float64{183.8716},
				Volume: []int64{147185},
				VWAP:   &[]float64{185},
				N:      &[]int64{147185},
			},
		},
	}

	for _, tc := range testCases {
		// Unmarshal the JSON data into a StockCandlesResponse instance
		s := &StockCandlesResponse{}
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

func TestPruneBeforeIndex(t *testing.T) {
	// Initialize a StockCandlesResponse instance
	s := &StockCandlesResponse{
		Time:   []int64{1, 2, 3, 4, 5},
		Open:   []float64{1.1, 2.2, 3.3, 4.4, 5.5},
		High:   []float64{1.1, 2.2, 3.3, 4.4, 5.5},
		Low:    []float64{1.1, 2.2, 3.3, 4.4, 5.5},
		Close:  []float64{1.1, 2.2, 3.3, 4.4, 5.5},
		Volume: []int64{100, 200, 300, 400, 500},
		VWAP:   &[]float64{1.1, 2.2, 3.3, 4.4, 5.5},
		N:      &[]int64{10, 20, 30, 40, 50},
	}

	// Call the method with index 2
	s.pruneBeforeIndex(2)

	// Check the results
	if len(s.Time) != 2 || s.Time[0] != 4 {
		t.Errorf("Time was incorrect, got: %v, want: %v.", s.Time, []int64{4, 5})
	}
	if len(s.Open) != 2 || s.Open[0] != 4.4 {
		t.Errorf("Open was incorrect, got: %v, want: %v.", s.Open, []float64{4.4, 5.5})
	}
}

func TestPruneAfterIndex(t *testing.T) {
	// Initialize a StockCandlesResponse instance
	s := &StockCandlesResponse{
		Time:   []int64{1, 2, 3, 4, 5},
		Open:   []float64{1.1, 2.2, 3.3, 4.4, 5.5},
		High:   []float64{1.1, 2.2, 3.3, 4.4, 5.5},
		Low:    []float64{1.1, 2.2, 3.3, 4.4, 5.5},
		Close:  []float64{1.1, 2.2, 3.3, 4.4, 5.5},
		Volume: []int64{100, 200, 300, 400, 500},
		VWAP:   &[]float64{1.1, 2.2, 3.3, 4.4, 5.5},
		N:      &[]int64{10, 20, 30, 40, 50},
	}

	// Call the method with index 2
	s.pruneAfterIndex(2)

	// Check the results
	if len(s.Time) != 2 || s.Time[1] != 2 {
		t.Errorf("Time was incorrect, got: %v, want: %v.", s.Time, []int64{1, 2})
	}
	if len(s.Open) != 2 || s.Open[1] != 2.2 {
		t.Errorf("Open was incorrect, got: %v, want: %v.", s.Open, []float64{1.1, 2.2})
	}
	// Continue for the rest of the fields...
}

func TestPruneOutsideDateRange(t *testing.T) {
	// JSON data
	data := []byte(`{
		"s": "ok",
		"t": [946684800, 978307200, 1009843200, 1041379200, 1072915200],
		"o": [1.1, 2.2, 3.3, 4.4, 5.5],
		"h": [1.1, 2.2, 3.3, 4.4, 5.5],
		"l": [1.1, 2.2, 3.3, 4.4, 5.5],
		"c": [1.1, 2.2, 3.3, 4.4, 5.5],
		"v": [100, 200, 300, 400, 500],
		"vwap": [1.1, 2.2, 3.3, 4.4, 5.5],
		"n": [10, 20, 30, 40, 50]
	}`)

	// Initialize a StockCandlesResponse instance and unmarshal the JSON data into it
	s := &StockCandlesResponse{}
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
	if len(s.Time) != 3 || s.Time[0] != 978307200 || s.Time[2] != 1041379200 {
		t.Errorf("Time was incorrect, got: %v, want: %v.", s.Time, []int64{978307200, 1009843200, 1041379200})
	}
	if len(s.Open) != 3 || s.Open[0] != 2.2 || s.Open[2] != 4.4 {
		t.Errorf("Open was incorrect, got: %v, want: %v.", s.Open, []float64{2.2, 3.3, 4.4})
	}
	// Continue for the rest of the fields...
}

func TestUnmarshalJSON(t *testing.T) {
	// Define a JSON string that represents a StockCandlesResponse object
	jsonStr := `{
		"s": "ok",
		"t": [1699462680,1699462740,1699462800,1699462860,1699462920,1699462980,1699463040,1699463100,1699463160,1699463220],
		"o": [182.09,182.04,182.02,181.94,181.9698,181.95,181.9197,181.94,181.9499,181.9703],
		"h": [182.095,182.07,182.07,181.9799,182.04,181.97,182.02,182,182.07,181.9999],
		"l": [182.01,182,181.89,181.855,181.915,181.87,181.91,181.9101,181.89,181.68],
		"c": [182.03,182.01,181.947,181.9699,181.9592,181.9203,181.93,181.94,181.98,181.695],
		"v": [81954,96569,236174,133964,103286,62645,71547,48792,109215,195926]
	}`

	// Create a new StockCandlesResponse object
	sc := &StockCandlesResponse{}

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

func TestMarshalJSON(t *testing.T) {
	// Initialize a StockCandlesResponse instance
	s := &StockCandlesResponse{
		Time:   []int64{1699630200, 1699630260},
		Open:   []float64{183.78, 183.88},
		High:   []float64{183.93, 183.96},
		Low:    []float64{183.76, 183.82},
		Close:  []float64{183.8716, 183.93},
		Volume: []int64{147185, 101982},
		VWAP:   &[]float64{185, 182},
		N:      &[]int64{147185, 101982},
	}

	// Call the method
	data, err := s.MarshalJSON()
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Check the results
	expected := jsonDataV2
	if string(data) != expected {
		t.Errorf("MarshalJSON did not return the expected result. Got: %s, want: %s", string(data), expected)
	}
}

func TestCombineStockCandlesResponse(t *testing.T) {
	// Test cases
	testCases := []struct {
		name string
		jsonData1 string
		jsonData2 string
		shouldErr bool
	}{
		{
			name: "Different versions and overlapping time ranges",
			jsonData1: jsonDataV1,
			jsonData2: jsonDataV2,
			shouldErr: true,
		},
		{
			name: "Same versions and non-overlapping time ranges AAPL",
			jsonData1: jsonDataAAPL1,
			jsonData2: jsonDataAAPL2,
			shouldErr: false, 
		},
		{
			name: "Same versions and non-overlapping time ranges SPY",
			jsonData1: jsonDataSPY1,
			jsonData2: jsonDataSPY2,
			shouldErr: true,
		},
	}

	for _, tc := range testCases {
		// Unmarshal the JSON data into StockCandlesResponse instances
		s1 := &StockCandlesResponse{}
		err := s1.UnmarshalJSON([]byte(tc.jsonData1))
		if err != nil {
			t.Fatalf("Failed to unmarshal JSON data: %v", err)
		}

		s2 := &StockCandlesResponse{}
		err = s2.UnmarshalJSON([]byte(tc.jsonData2))
		if err != nil {
			t.Fatalf("Failed to unmarshal JSON data: %v", err)
		}

		// Try to combine the StockCandlesResponse instances
		_, err = CombineStockCandles(s1, s2)
		if (err != nil) != tc.shouldErr {
			t.Errorf("Test case %s: CombineStockCandlesResponse() error = %v, wantErr %v", tc.name, err, tc.shouldErr)
		}
	}
}

func TestCheckTimeInAscendingOrder(t *testing.T) {
	// Test cases
	testCases := []struct {
		time      []int64
		shouldErr bool
	}{
		{
			time:      []int64{1, 2, 3, 4, 5},
			shouldErr: false, // Time is in ascending order
		},
		{
			time:      []int64{1, 3, 2, 4, 5},
			shouldErr: true, // Time is not in ascending order
		},
	}

	for _, tc := range testCases {
		// Create a StockCandlesResponse instance with the specified time
		s := &StockCandlesResponse{
			Time: tc.time,
		}

		// Call the method
		err := s.checkTimeInAscendingOrder()

		// Check if the result matches the expected result
		if (err != nil) != tc.shouldErr {
			t.Errorf("checkTimeInAscendingOrder() error = %v, wantErr %v", err, tc.shouldErr)
		}
	}
}
