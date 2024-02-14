package models

import (
	"fmt"
	"sort"
	"time"
)

func ExampleByDate() {
	// Assuming the Candle struct has at least a Date field of type time.Time
	candles := []Candle{
		{Date: time.Date(2023, 3, 10, 0, 0, 0, 0, time.UTC)},
		{Date: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
		{Date: time.Date(2023, 2, 20, 0, 0, 0, 0, time.UTC)},
	}

	// Sorting the slice of Candle instances by their Date field in ascending order
	sort.Sort(ByDate(candles))

	// Printing out the sorted dates to demonstrate the order
	for _, candle := range candles {
		fmt.Println(candle.Date.Format("2006-01-02"))
	}

	// Output:
	// 2023-01-05
	// 2023-02-20
	// 2023-03-10
}

func ExampleByVolume() {
	// Assuming the Candle struct has at least a Volume field of type int
	candles := []Candle{
		{Volume: 300},
		{Volume: 100},
		{Volume: 200},
	}

	// Sorting the slice of Candle instances by their Volume field in ascending order
	sort.Sort(ByVolume(candles))

	// Printing out the sorted volumes to demonstrate the order
	for _, candle := range candles {
		fmt.Println(candle.Volume)
	}

	// Output:
	// 100
	// 200
	// 300
}

func ExampleByOpen() {
	// Assuming the Candle struct has at least an Open field of type float64
	candles := []Candle{
		{Open: 10.5},
		{Open: 8.2},
		{Open: 9.7},
	}

	// Sorting the slice of Candle instances by their Open field in ascending order
	sort.Sort(ByOpen(candles))

	// Printing out the sorted Open values to demonstrate the order
	for _, candle := range candles {
		fmt.Printf("%.1f\n", candle.Open)
	}

	// Output:
	// 8.2
	// 9.7
	// 10.5
}

func ExampleByHigh() {
	// Assuming the Candle struct has at least a High field of type float64
	candles := []Candle{
		{High: 15.2},
		{High: 11.4},
		{High: 13.5},
	}

	// Sorting the slice of Candle instances by their High field in ascending order
	sort.Sort(ByHigh(candles))

	// Printing out the sorted High values to demonstrate the order
	for _, candle := range candles {
		fmt.Printf("%.1f\n", candle.High)
	}

	// Output:
	// 11.4
	// 13.5
	// 15.2
}

func ExampleByLow() {
	// Assuming the Candle struct has at least a Low field of type float64
	candles := []Candle{
		{Low: 5.5},
		{Low: 7.2},
		{Low: 6.3},
	}

	// Sorting the slice of Candle instances by their Low field in ascending order
	sort.Sort(ByLow(candles))

	// Printing out the sorted Low values to demonstrate the order
	for _, candle := range candles {
		fmt.Printf("%.1f\n", candle.Low)
	}

	// Output:
	// 5.5
	// 6.3
	// 7.2
}

func ExampleByClose() {
	// Create a slice of Candle instances
	candles := []Candle{
		{Symbol: "AAPL", Date: time.Now(), Open: 100, High: 105, Low: 95, Close: 102, Volume: 1000},
		{Symbol: "AAPL", Date: time.Now(), Open: 102, High: 106, Low: 98, Close: 104, Volume: 1500},
		{Symbol: "AAPL", Date: time.Now(), Open: 99, High: 103, Low: 97, Close: 100, Volume: 1200},
	}

	// Sort the candles by their Close value using sort.Sort and ByClose
	sort.Sort(ByClose(candles))

	// Print the sorted candles to demonstrate the order
	for _, candle := range candles {
		fmt.Printf("Close: %v\n", candle.Close)
	}

	// Output:
	// Close: 100
	// Close: 102
	// Close: 104
}

func ExampleByVWAP() {
	// Assuming the Candle struct has at least a VWAP (Volume Weighted Average Price) field of type float64
	candles := []Candle{
		{VWAP: 10.5},
		{VWAP: 8.2},
		{VWAP: 9.7},
	}

	// Sorting the slice of Candle instances by their VWAP field in ascending order
	sort.Sort(ByVWAP(candles))

	// Printing out the sorted VWAP values to demonstrate the order
	for _, candle := range candles {
		fmt.Printf("%.1f\n", candle.VWAP)
	}

	// Output:
	// 8.2
	// 9.7
	// 10.5
}

func ExampleByN() {
	// Assuming the Candle struct has at least an N field of type int (or any comparable type)
	candles := []Candle{
		{N: 3},
		{N: 1},
		{N: 2},
	}

	// Sorting the slice of Candle instances by their N field in ascending order
	sort.Sort(ByN(candles))

	// Printing out the sorted N values to demonstrate the order
	for _, candle := range candles {
		fmt.Println(candle.N)
	}

	// Output:
	// 1
	// 2
	// 3
}

func ExampleBySymbol() {
	// Create a slice of Candle instances with different symbols
	candles := []Candle{
		{Symbol: "MSFT", Date: time.Date(2023, 4, 10, 0, 0, 0, 0, time.UTC), Open: 250.0, High: 255.0, Low: 248.0, Close: 252.0, Volume: 3000},
		{Symbol: "AAPL", Date: time.Date(2023, 4, 10, 0, 0, 0, 0, time.UTC), Open: 150.0, High: 155.0, Low: 149.0, Close: 152.0, Volume: 2000},
		{Symbol: "GOOGL", Date: time.Date(2023, 4, 10, 0, 0, 0, 0, time.UTC), Open: 1200.0, High: 1210.0, Low: 1195.0, Close: 1205.0, Volume: 1000},
	}

	// Sort the candles by their Symbol using sort.Sort and BySymbol
	sort.Sort(BySymbol(candles))

	// Print the sorted candles to demonstrate the order
	for _, candle := range candles {
		fmt.Printf("Symbol: %s, Close: %.2f\n", candle.Symbol, candle.Close)
	}

	// Output:
	// Symbol: AAPL, Close: 152.00
	// Symbol: GOOGL, Close: 1205.00
	// Symbol: MSFT, Close: 252.00
}
