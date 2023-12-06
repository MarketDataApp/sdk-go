package main

import (
	"fmt"
	"log"
	"sort"
	"time"

	api "github.com/MarketDataApp/sdk-go/endpoints"
)

func marketstatusExample() {

	msr, _ := api.MarketStatus().Country("US").From("2022-01-01").To("2022-12-31").Get()
	fmt.Println(msr)

}

func stocksTickersExample() {
	tickers, _ := api.StockTickers().Date("2023-01-05").Get()
	fmt.Println(tickers)
}

func SaveTickersToCSV(startDate, endDate string, filename string) error {
	// Initialize the markets client

	marketStatusResp, err := api.MarketStatus().From(startDate).To(endDate).Get()
	if err != nil {
		log.Fatalf("Failed to get market status: %v", err)
	}
	// Print out marketStatusResp for test run visibility
	fmt.Printf("Market Status Response: %v\n", marketStatusResp)

	openDates, err := marketStatusResp.GetOpenDates()
	if err != nil {
		log.Fatalf("Failed to get open dates: %v", err)
	}
	// Print out openDates for test run visibility
	fmt.Println("Open dates:")
	for _, date := range openDates {
		fmt.Printf("%v\n", date)
	}

	// Sort dates in ascending order
	sort.Slice(openDates, func(i, j int) bool {
		return openDates[i].Before(openDates[j])
	})

	// Initialize the stocks client
	tickers := api.StockTickers()

	// Get TickersResponse for each date and combine them into a map
	tickerMap := make(map[string]api.TickerObj)
	for _, date := range openDates {
		// Convert date to string in the format "YYYY-MM-DD"
		dateStr := date.Format("2006-01-02")

		// Get the TickersResponse for the date
		response, err := tickers.Date(dateStr).Get()
		if err != nil {
			return err
		}

		// Convert the response to a map
		responseMap, err := response.ToMap()
		if err != nil {
			return err
		}

		// Merge the map into the combined map
		for key, value := range responseMap {
			tickerMap[key] = value
		}
	}

	// Get the keys of the tickerMap
	keys := make([]string, 0, len(tickerMap))
	for key := range tickerMap {
		keys = append(keys, key)
	}

	// Sort the keys in alphabetical order
	sort.Strings(keys)

	// Create a new map with sorted keys
	sortedTickerMap := make(map[string]api.TickerObj)
	for _, key := range keys {
		sortedTickerMap[key] = tickerMap[key]
	}

	// Save the sorted map to a CSV file
	err = api.SaveToCSV(sortedTickerMap, filename)
	if err != nil {
		return err
	}

	return nil
}

func SaveSingleDayTickersToCSV(date time.Time, filename string) error {
	// Initialize the markets client

	// Initialize the stocks client
	tickers := api.StockTickers()

	// Convert date to string in the format "YYYY-MM-DD"
	dateStr := date.Format("2006-01-02")

	// Get the TickersResponse for the date
	response, err := tickers.Date(dateStr).Get()
	if err != nil {
		return err
	}

	// Convert the response to a map
	responseMap, err := response.ToMap()
	if err != nil {
		return err
	}

	// Save the map to a CSV file
	err = api.SaveToCSV(responseMap, filename)
	if err != nil {
		return err
	}

	return nil
}
