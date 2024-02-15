package models

import (
	"testing"
	"time"
)

func TestGetOpenDates(t *testing.T) {
	// Create a mock MarketStatusResponse
	msr := &MarketStatusResponse{
		Date:   []int64{time.Now().Unix(), time.Now().AddDate(0, 0, -1).Unix()},
		Status: []string{"open", "closed"},
	}

	openDates, err := msr.GetOpenDates()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(openDates) != 1 {
		t.Fatalf("Expected 1 open date, got %d", len(openDates))
	}

	if openDates[0].Day() != time.Now().Day() {
		t.Fatalf("Expected today's date, got %v", openDates[0])
	}
}

func TestGetClosedDates(t *testing.T) {
	// Create a mock MarketStatusResponse
	msr := &MarketStatusResponse{
		Date:   []int64{time.Now().Unix(), time.Now().AddDate(0, 0, -1).Unix()},
		Status: []string{"open", "closed"},
	}

	closedDates, err := msr.GetClosedDates()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(closedDates) != 1 {
		t.Fatalf("Expected 1 closed date, got %d", len(closedDates))
	}

	if closedDates[0].Day() != time.Now().AddDate(0, 0, -1).Day() {
		t.Fatalf("Expected yesterday's date, got %v", closedDates[0])
	}
}
