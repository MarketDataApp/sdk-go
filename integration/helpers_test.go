//go:build integration

// Package integration contains integration tests for the MarketData SDK.
// These tests run against the live API and require a valid API key.
//
// Run with: go test ./integration/... -tags=integration -v
package integration

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
)

// Test constants - well-known symbols for testing
const (
	TestStockSymbol  = "AAPL"
	TestStockSymbol2 = "MSFT"
	TestStockSymbol3 = "META"
	TestFundSymbol   = "VFINX"
	TestIndexSymbol  = "VIX"
)

// setupClient creates a new MarketData client for testing. It skips the
// test if no token is available. The token is resolved through the SDK's
// own configuration cascade (process env, then a .env file) via a bare
// NewClient() call, rather than reading os.Getenv directly: a token that
// lives only in .env (as it does in local development) satisfies the SDK
// but would not satisfy a direct os.Getenv("MARKETDATA_TOKEN") check,
// since NewClient never writes .env values into the process environment.
func setupClient(t *testing.T) *marketdata.Client {
	t.Helper()

	client, err := marketdata.NewClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	if client.DemoMode() {
		t.Skip("no API token available (checked the process environment and .env), skipping integration test")
	}

	return client
}

// testContext creates a context with timeout for integration tests.
func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// Historical date helpers - use dates that are known to have data
func historicalDate() time.Time {
	// Use a date in the past that we know had trading activity.
	// (2024-01-15 was Martin Luther King Jr. Day — markets closed.)
	return time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
}

func historicalDateRange() (time.Time, time.Time) {
	// January 2024 - a month with known trading days
	from := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	return from, to
}

// Assertion helpers

// assertNoError fails the test if err is not nil.
func assertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// assertNotNil fails the test if value is nil.
func assertNotNil(t *testing.T, value interface{}, msg string) {
	t.Helper()
	if value == nil {
		t.Fatalf("%s: expected non-nil value", msg)
	}
	// A nil *T boxed in an interface is not == nil — the interface holds a
	// type — so the check above passes for exactly the value that would
	// panic on the next dereference, which is the only value worth
	// asserting about. Reflection sees through the box.
	switch v := reflect.ValueOf(value); v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface, reflect.Func, reflect.Chan:
		if v.IsNil() {
			t.Fatalf("%s: expected non-nil value, got a nil %s", msg, v.Type())
		}
	}
}

// assertNotEmpty fails the test if slice is empty.
func assertNotEmpty[T any](t *testing.T, slice []T, msg string) {
	t.Helper()
	if len(slice) == 0 {
		t.Fatalf("%s: expected non-empty slice", msg)
	}
}

// assertPositive fails the test if value is not positive.
func assertPositive(t *testing.T, value float64, msg string) {
	t.Helper()
	if value <= 0 {
		t.Fatalf("%s: expected positive value, got %f", msg, value)
	}
}

// assertEqual fails the test if the values are not equal.
func assertEqual[T comparable](t *testing.T, got, want T, msg string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: got %v, want %v", msg, got, want)
	}
}

// assertTrue fails the test if condition is false.
func assertTrue(t *testing.T, condition bool, msg string) {
	t.Helper()
	if !condition {
		t.Fatalf("%s: expected true", msg)
	}
}

// assertValidOHLC validates that OHLC data is consistent.
func assertValidOHLC(t *testing.T, open, high, low, close float64) {
	t.Helper()

	if high < low {
		t.Fatalf("Invalid OHLC: high (%f) < low (%f)", high, low)
	}
	if high < open || high < close {
		t.Fatalf("Invalid OHLC: high (%f) is not the highest (open=%f, close=%f)", high, open, close)
	}
	if low > open || low > close {
		t.Fatalf("Invalid OHLC: low (%f) is not the lowest (open=%f, close=%f)", low, open, close)
	}
}

// assertTimeInPast validates that the time is in the past.
func assertTimeInPast(t *testing.T, tm time.Time, msg string) {
	t.Helper()
	if tm.After(time.Now()) {
		t.Fatalf("%s: time %v is in the future", msg, tm)
	}
}

// assertTimeInFuture validates that the time is in the future.
func assertTimeInFuture(t *testing.T, tm time.Time, msg string) {
	t.Helper()
	if tm.Before(time.Now()) {
		t.Fatalf("%s: time %v is in the past", msg, tm)
	}
}

// assertTimeSorted validates that times are sorted in ascending order.
func assertTimeSorted(t *testing.T, times []time.Time, msg string) {
	t.Helper()
	for i := 1; i < len(times); i++ {
		if times[i].Before(times[i-1]) {
			t.Fatalf("%s: not sorted at index %d: %v < %v", msg, i, times[i], times[i-1])
		}
	}
}
