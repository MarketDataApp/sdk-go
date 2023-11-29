package main

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/MarketDataApp/sdk-go/client"
	"github.com/MarketDataApp/sdk-go/endpoints/markets"
	"github.com/MarketDataApp/sdk-go/endpoints/stocks"
)

func marketstatusExample() {

	status, err := markets.New()
	if err != nil {
		log.Fatalf("Failed to create new status: %v", err)
	}

	result, _ := status.Country("US").From("2022-01-01").To("2022-12-31").GetMarketStatus()
	fmt.Println(result)

}

func stocksTickersExample() {
	client.SetEnvironment("dev")
	tickers, err := stocks.New()
	if err != nil {
		log.Fatalf("Failed to create new tickers: %v", err)
	}

	result, _ := tickers.Date("2023-01-05").GetTickers()
	fmt.Println(result)
	result, _ = tickers.Date("2023-01-06").GetTickers()
	fmt.Println(result)

}

func SaveTickersToCSV(startDate, endDate string, filename string) error {
	// Initialize the markets client
	client.SetEnvironment("dev")

	marketStatus, err := markets.New()
	if err != nil {
		log.Fatalf("Failed to create new market status: %v", err)
	}

	marketStatus.From(startDate).To(endDate)
	// Get all open dates between start and end date
	marketStatusResp, err := marketStatus.GetMarketStatus()
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
	tickers, err := stocks.New()
	if err != nil {
		log.Fatalf("Failed to create new tickers: %v", err)
	}

	// Get TickersResponse for each date and combine them into a map
	tickerMap := make(map[string]stocks.TickerInfo)
	for _, date := range openDates {
		// Convert date to string in the format "YYYY-MM-DD"
		dateStr := date.Format("2006-01-02")

		// Get the TickersResponse for the date
		response, err := tickers.Date(dateStr).GetTickers()
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
	sortedTickerMap := make(map[string]stocks.TickerInfo)
	for _, key := range keys {
		sortedTickerMap[key] = tickerMap[key]
	}

	// Save the sorted map to a CSV file
	err = stocks.SaveToCSV(sortedTickerMap, filename)
	if err != nil {
		return err
	}

	return nil
}

func SaveSingleDayTickersToCSV(date time.Time, filename string) error {
	// Initialize the markets client
	client.SetEnvironment("dev")

	// Initialize the stocks client
	tickers, err := stocks.New()
	if err != nil {
		log.Fatalf("Failed to create new tickers: %v", err)
	}

	// Convert date to string in the format "YYYY-MM-DD"
	dateStr := date.Format("2006-01-02")

	// Get the TickersResponse for the date
	response, err := tickers.Date(dateStr).GetTickers()
	if err != nil {
		return err
	}

	// Convert the response to a map
	responseMap, err := response.ToMap()
	if err != nil {
		return err
	}

	// Save the map to a CSV file
	err = stocks.SaveToCSV(responseMap, filename)
	if err != nil {
		return err
	}

	return nil
}