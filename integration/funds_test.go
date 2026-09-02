//go:build integration

package integration

import (
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/funds"
)

func TestFunds_Candles(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	candles, _, err := client.Funds.Candles(ctx, TestFundSymbol,
		funds.WithResolution(funds.ResolutionDaily),
		funds.WithCandleWindow(funds.LastN(10)),
	)
	assertNoError(t, err, "Funds Candles")
	assertNotEmpty(t, candles, "candles should not be empty")

	for i, c := range candles {
		// NAV (price) should be positive
		assertPositive(t, c.Open, "fund open/NAV")
		assertPositive(t, c.High, "fund high")
		assertPositive(t, c.Low, "fund low")
		assertPositive(t, c.Close, "fund close/NAV")

		// Validate OHLC relationships
		if c.High < c.Low {
			t.Errorf("candle %d: high (%f) < low (%f)", i, c.High, c.Low)
		}

		// Timestamp should be in the past
		assertTimeInPast(t, c.Time, "fund candle timestamp")
	}
}

func TestFunds_Candles_DateRange(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	from, to := historicalDateRange()

	candles, _, err := client.Funds.Candles(ctx, TestFundSymbol,
		funds.WithResolution(funds.ResolutionDaily),
		funds.WithCandleWindow(funds.Between(from, to)),
	)
	assertNoError(t, err, "Funds Candles with date range")
	assertNotEmpty(t, candles, "candles should not be empty")

	// All candles should be within the requested date range
	// Allow tolerance for timezone differences
	fromWithTolerance := from.Add(-24 * time.Hour)
	toWithTolerance := to.Add(24 * time.Hour)

	for i, c := range candles {
		if c.Time.Before(fromWithTolerance) || c.Time.After(toWithTolerance) {
			t.Errorf("candle %d: timestamp %v outside range [%v, %v]",
				i, c.Time, from, to)
		}
	}
}

func TestFunds_Candles_Weekly(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	candles, _, err := client.Funds.Candles(ctx, TestFundSymbol,
		funds.WithResolution(funds.ResolutionWeekly),
		funds.WithCandleWindow(funds.LastN(5)),
	)
	assertNoError(t, err, "Funds Weekly Candles")
	assertNotEmpty(t, candles, "weekly candles should not be empty")

	for _, c := range candles {
		assertPositive(t, c.Close, "weekly close/NAV")
		if c.High < c.Low {
			t.Errorf("weekly candle: high (%f) < low (%f)", c.High, c.Low)
		}
	}
}

func TestFunds_Candles_Monthly(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	candles, _, err := client.Funds.Candles(ctx, TestFundSymbol,
		funds.WithResolution(funds.ResolutionMonthly),
		funds.WithCandleWindow(funds.LastN(3)),
	)
	assertNoError(t, err, "Funds Monthly Candles")
	assertNotEmpty(t, candles, "monthly candles should not be empty")

	for _, c := range candles {
		assertPositive(t, c.Close, "monthly close/NAV")
		if c.High < c.Low {
			t.Errorf("monthly candle: high (%f) < low (%f)", c.High, c.Low)
		}
	}
}
