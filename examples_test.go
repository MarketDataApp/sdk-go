package main

import (
	"fmt"
	"log"
	"time"
	"testing"
)

func TestSaveTickersToCSV(t *testing.T) {
	// Define the start and end dates for December of last year
	lastYear := time.Now().Year() - 1
	startDate := fmt.Sprintf("%d-%02d-%02d", lastYear, time.December, 1)
	endDate := fmt.Sprintf("%d-%02d-%02d", lastYear, time.December, 31)

	// Define the filename in the format "YYYY-MM.csv"
	filename := fmt.Sprintf("%d-%02d.csv", lastYear, time.December)

	// Call the SaveTickersToCSV function
	err := SaveTickersToCSV(startDate, endDate, filename)
	if err != nil {
		log.Fatalf("Failed to save tickers to CSV: %v", err)
	}
}

func TestSaveSingleDayTickersToCSV(t *testing.T) {
	date := time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)
	err := SaveSingleDayTickersToCSV(date, "test.csv")
	if err != nil {
		t.Errorf("SaveSingleDayTickersToCSV failed with error: %v", err)
	}
}