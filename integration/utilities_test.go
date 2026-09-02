//go:build integration

package integration

import (
	"testing"
)

func TestUtilities_Status(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	status, _, err := client.Utilities.Status(ctx)
	assertNoError(t, err, "Utilities Status")
	assertNotNil(t, status, "status should not be nil")

	// API should be online (otherwise we couldn't make this request)
	assertTrue(t, status.IsOnline(), "API should be online")

	// Uptime should be reasonable (>90%)
	if status.Uptime30d < 90.0 {
		t.Errorf("30-day uptime seems too low: %.2f%%", status.Uptime30d)
	}
	if status.Uptime90d < 90.0 {
		t.Errorf("90-day uptime seems too low: %.2f%%", status.Uptime90d)
	}

	// Uptime should be <= 100%
	if status.Uptime30d > 100.0 {
		t.Errorf("30-day uptime > 100%%: %.2f%%", status.Uptime30d)
	}
	if status.Uptime90d > 100.0 {
		t.Errorf("90-day uptime > 100%%: %.2f%%", status.Uptime90d)
	}

	// Updated time should be in the past
	assertTimeInPast(t, status.Updated, "status update time")
}

func TestUtilities_Headers(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	headers, _, err := client.Utilities.Headers(ctx)
	// Headers endpoint may not be available or may return different format
	if err != nil {
		t.Skipf("Headers endpoint not available: %v", err)
	}
	assertNotNil(t, headers, "headers should not be nil")

	// Headers map may be empty depending on the endpoint format
	// Just log what we got
	t.Logf("Got %d headers", len(headers.Headers))
}

func TestUtilities_User(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	user, _, err := client.Utilities.User(ctx)
	assertNoError(t, err, "Utilities User")
	assertNotNil(t, user, "user should not be nil")

	// CreditLimit should be positive
	if user.CreditLimit <= 0 {
		t.Errorf("CreditLimit should be positive, got %d", user.CreditLimit)
	}

	// CreditsRemaining should be non-negative and within the limit
	if user.CreditsRemaining < 0 || user.CreditsRemaining > user.CreditLimit {
		t.Errorf("CreditsRemaining = %d, want within [0, %d]", user.CreditsRemaining, user.CreditLimit)
	}

	// The credit window should reset in the future
	assertTimeInFuture(t, user.ResetAt, "credit window reset")

	t.Logf("User: %s", user)
}

func TestRateLimits_Tracking(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	// Make a request to populate rate limits
	_, _, err := client.Stocks.Quote(ctx, TestStockSymbol)
	assertNoError(t, err, "Quote for rate limit test")

	// Check rate limits
	limits := client.RateLimits()

	// Limit should be positive (we have a quota)
	if limits.Limit <= 0 {
		t.Errorf("Expected positive rate limit, got %d", limits.Limit)
	}

	// Remaining should be non-negative
	if limits.Remaining < 0 {
		t.Errorf("Expected non-negative remaining, got %d", limits.Remaining)
	}

	// Consumed may or may not be returned by the API
	// Some API plans may not include this header
	if limits.Consumed == 0 {
		t.Log("Consumed header not returned by API (may be plan-specific)")
	}

	// Remaining should be <= Limit
	if limits.Remaining > limits.Limit {
		t.Errorf("Remaining (%d) should be <= Limit (%d)", limits.Remaining, limits.Limit)
	}
}

func TestRateLimits_DecrementAfterRequest(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	// Make a request to populate rate limits
	_, _, err := client.Stocks.Quote(ctx, TestStockSymbol)
	assertNoError(t, err, "First quote")

	// Get initial state
	initial := client.RateLimits()

	// Make another request
	_, _, err = client.Stocks.Quote(ctx, TestStockSymbol2)
	assertNoError(t, err, "Second quote")

	// Get new state
	after := client.RateLimits()

	// Consumed should have increased
	if after.Consumed <= initial.Consumed {
		t.Errorf("Consumed should increase: before=%d, after=%d",
			initial.Consumed, after.Consumed)
	}

	// Remaining should have decreased (or stayed same if limit reset)
	// We can't strictly assert this because limits might reset between requests
}
