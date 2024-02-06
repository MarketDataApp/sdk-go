package examples

import (
	"fmt"
	"log"
	"sort"
	"time"

	api "github.com/MarketDataApp/sdk-go"
	md "github.com/MarketDataApp/sdk-go/models"
)

func StockCandlesV2Example() {

	sce, err := api.StockCandlesV2().Resolution("1").Symbol("AAPL").DateKey("2023-01").Packed()
	if err != nil {
		fmt.Print(err)
		return
	}

	candles, err := sce.Unpack()
	if err != nil {
		fmt.Print(err)
		return
	}

	for _, candle := range candles {
		fmt.Println(candle)
	}
}

func StocksTickersV2Example() {
	tickers, err := api.StockTickers().DateKey("2023-01-05").Packed()
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(tickers)
}

func SaveTickersToCSV(startDate, endDate string, filename string) error {
	// Initialize the markets client

	marketStatusResp, err := api.MarketStatus().From(startDate).To(endDate).Packed()
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
	tickerMap := make(map[string]md.Ticker)
	for _, date := range openDates {
		// Convert date to string in the format "YYYY-MM-DD"
		dateStr := date.Format("2006-01-02")

		// Get the TickersResponse for the date
		response, err := tickers.DateKey(dateStr).Packed()
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
	sortedTickerMap := make(map[string]md.Ticker)
	for _, key := range keys {
		sortedTickerMap[key] = tickerMap[key]
	}

	// Save the sorted map to a CSV file
	err = md.SaveToCSV(sortedTickerMap, filename)
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
	response, err := tickers.DateKey(dateStr).Packed()
	if err != nil {
		return err
	}

	// Convert the response to a map
	responseMap, err := response.ToMap()
	if err != nil {
		return err
	}

	// Save the map to a CSV file
	err = md.SaveToCSV(responseMap, filename)
	if err != nil {
		return err
	}

	return nil
}
