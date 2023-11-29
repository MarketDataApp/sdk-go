package dates

import (
	"testing"
	"time"
)

func TestTryParseUnixTimestamp(t *testing.T) {
	// Test Case 1: Designed to succeed
	// Input is a string representing Unix timestamp in milliseconds
	successInput := "1617181723000" // This should correspond to a specific date and time
	expectedTime := time.Unix(0, 1617181723000*int64(time.Millisecond)).UTC()

	timeResult, precision, err := ParseDateInput(successInput)
	if err != nil {
		t.Errorf("TestTryParseUnixTimestamp success case failed with error: %s", err)
	}
	if timeResult.UTC() != expectedTime {
		t.Errorf("TestTryParseUnixTimestamp success case failed: expected %v, got %v", expectedTime, timeResult)
	}
	if precision != "millisecond" {
		t.Errorf("TestTryParseUnixTimestamp success case failed: expected precision %v, got %v", "millisecond", precision)
	}

	// Test Case 2: Designed to fail
	// Input is an invalid Unix timestamp
	failInput := "invalidtimestamp"
	_, _, err = ParseDateInput(failInput)
	if err == nil {
		t.Errorf("TestTryParseUnixTimestamp fail case did not result in an error as expected")
	}
}
