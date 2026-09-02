//go:build integration

package integration

import (
	"os"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

func TestStocks_Quote(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	quote, _, err := client.Stocks.Quote(ctx, TestStockSymbol)
	assertNoError(t, err, "Quote")
	assertNotNil(t, quote, "quote should not be nil")

	// Validate symbol matches
	assertEqual(t, quote.Symbol, TestStockSymbol, "symbol should match")

	// Validate prices are positive
	assertPositive(t, quote.Last, "last price")
	assertPositive(t, quote.Bid, "bid price")
	assertPositive(t, quote.Ask, "ask price")

	// Validate bid <= ask (spread must be non-negative)
	if quote.Bid > quote.Ask {
		t.Errorf("Bid (%f) should be <= Ask (%f)", quote.Bid, quote.Ask)
	}

	// Validate mid is between bid and ask
	if quote.Mid < quote.Bid || quote.Mid > quote.Ask {
		t.Errorf("Mid (%f) should be between Bid (%f) and Ask (%f)", quote.Mid, quote.Bid, quote.Ask)
	}

	// Updated time should be recent (within last 24 hours for market hours)
	assertTimeInPast(t, quote.Updated, "quote update time")
}

func TestStocks_Quote_With52Week(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	quote, _, err := client.Stocks.Quote(ctx, TestStockSymbol, stocks.WithFiftyTwoWeek(true))
	assertNoError(t, err, "Quote with 52week")
	assertNotNil(t, quote, "quote should not be nil")

	// 52-week data may or may not be returned depending on API plan/availability
	// If both values are present, validate the relationship
	if quote.FiftyTwoWeekHigh > 0 && quote.FiftyTwoWeekLow > 0 {
		// Low should be <= High
		if quote.FiftyTwoWeekLow > quote.FiftyTwoWeekHigh {
			t.Errorf("52-week low (%f) should be <= high (%f)",
				quote.FiftyTwoWeekLow, quote.FiftyTwoWeekHigh)
		}
	} else {
		t.Log("52-week data not returned by API (may require specific API plan)")
	}
}

func TestStocks_Quotes(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	symbols := []string{TestStockSymbol, TestStockSymbol2}
	quotes, _, err := client.Stocks.Quotes(ctx, symbols)
	assertNoError(t, err, "Quotes")
	assertNotEmpty(t, quotes, "quotes should not be empty")

	// Should return quotes for all requested symbols
	if len(quotes) != len(symbols) {
		t.Errorf("Expected %d quotes, got %d", len(symbols), len(quotes))
	}

	// Verify all symbols are present
	foundSymbols := make(map[string]bool)
	for _, q := range quotes {
		foundSymbols[q.Symbol] = true
		assertPositive(t, q.Last, "last price for "+q.Symbol)
	}

	for _, sym := range symbols {
		if !foundSymbols[sym] {
			t.Errorf("Missing quote for symbol %s", sym)
		}
	}
}

func TestStocks_Candles_Daily(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	// Request last 10 daily candles
	from, to := historicalDateRange()
	candles, _, err := client.Stocks.Candles(ctx, TestStockSymbol,
		stocks.WithResolution(stocks.ResolutionDaily),
		stocks.WithCandleWindow(stocks.Between(from, to)),
	)
	assertNoError(t, err, "Candles")
	assertNotEmpty(t, candles, "candles should not be empty")

	for i, c := range candles {
		// Validate OHLC relationships
		assertValidOHLC(t, c.Open, c.High, c.Low, c.Close)

		// Validate prices are positive
		assertPositive(t, c.Open, "open price")
		assertPositive(t, c.High, "high price")
		assertPositive(t, c.Low, "low price")
		assertPositive(t, c.Close, "close price")

		// Volume should be non-negative
		if c.Volume < 0 {
			t.Errorf("candle %d: volume should be non-negative, got %d", i, c.Volume)
		}

		// Timestamp should be in the past
		assertTimeInPast(t, c.Time, "candle timestamp")
	}
}

func TestStocks_Candles_DateRange(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	from, to := historicalDateRange()
	candles, _, err := client.Stocks.Candles(ctx, TestStockSymbol,
		stocks.WithResolution(stocks.ResolutionDaily),
		stocks.WithCandleWindow(stocks.Between(from, to)),
	)
	assertNoError(t, err, "Candles with date range")
	assertNotEmpty(t, candles, "candles should not be empty")

	// All candles should be within the requested date range
	// Allow one day tolerance since candle times might be end-of-day
	fromWithTolerance := from.Add(-24 * time.Hour)
	toWithTolerance := to.Add(24 * time.Hour)

	for i, c := range candles {
		if c.Time.Before(fromWithTolerance) || c.Time.After(toWithTolerance) {
			t.Errorf("candle %d: timestamp %v outside range [%v, %v]",
				i, c.Time, from, to)
		}
	}
}

func TestStocks_Candles_IntradayLongRange_NoDuplicates(t *testing.T) {
	// Regression for the chunk-boundary duplication bug: an intraday range
	// spanning more than a year is auto-split into concurrent chunks, and the
	// API's inclusive date-only from/to used to hand the boundary day to two
	// chunks, silently duplicating its candles in the merged result.
	client := setupClient(t)
	ctx := testContext(t)

	// Anchor the range so the year mark after `from` lands on a Wednesday: a
	// weekend/holiday boundary has no candles to duplicate and would let the
	// old shared-boundary chunking pass by luck.
	boundary := time.Now().AddDate(0, 0, -7)
	for boundary.Weekday() != time.Wednesday {
		boundary = boundary.AddDate(0, 0, -1)
	}
	from := boundary.AddDate(-1, 0, 0)
	to := time.Now()
	candles, _, err := client.Stocks.Candles(ctx, TestStockSymbol,
		stocks.WithResolution(stocks.Resolution1Hour),
		stocks.WithCandleWindow(stocks.Between(from, to)),
	)
	assertNoError(t, err, "Candles intraday long range")
	assertNotEmpty(t, candles, "candles should not be empty")

	for i := 1; i < len(candles); i++ {
		if !candles[i].Time.After(candles[i-1].Time) {
			t.Errorf("candle %d: timestamp %v is not strictly after candle %d (%v) — duplicate or out of order",
				i, candles[i].Time, i-1, candles[i-1].Time)
		}
	}
}

func TestStocks_Prices(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	prices, _, err := client.Stocks.Prices(ctx, []string{TestStockSymbol})
	assertNoError(t, err, "Prices")
	assertNotEmpty(t, prices, "prices should not be empty")

	price := prices[0]
	assertEqual(t, price.Symbol, TestStockSymbol, "symbol should match")
	assertPositive(t, price.Mid, "mid price")
}

func TestStocks_Prices_Multiple(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	symbols := []string{TestStockSymbol, TestStockSymbol2}
	prices, _, err := client.Stocks.Prices(ctx, symbols)
	assertNoError(t, err, "Prices for multiple symbols")
	assertNotEmpty(t, prices, "prices should not be empty")

	// Should return prices for all requested symbols
	if len(prices) != len(symbols) {
		t.Errorf("Expected %d prices, got %d", len(symbols), len(prices))
	}

	foundSymbols := make(map[string]bool)
	for _, p := range prices {
		foundSymbols[p.Symbol] = true
		assertPositive(t, p.Mid, "mid price for "+p.Symbol)
	}

	for _, sym := range symbols {
		if !foundSymbols[sym] {
			t.Errorf("Missing price for symbol %s", sym)
		}
	}
}

func TestStocks_Earnings(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	// The n most recent reports via LastN, which the SDK anchors with an
	// explicit to= of today (Eastern): the API silently ignores countback
	// unless `to` is present (its bare default is the undocumented
	// upcoming-only window), so an unanchored LastN(4) used to return a
	// single future, unreported row — and this test's old "non-empty"
	// assertion never noticed. Assert what "the last 4 reports" actually
	// means: exactly 4 historical rows with real data.
	earnings, _, err := client.Stocks.Earnings(ctx, TestStockSymbol,
		stocks.WithEarningsWindow(stocks.LastN(4)),
	)
	assertNoError(t, err, "Earnings")
	if len(earnings) != 4 {
		t.Fatalf("len(earnings) = %d, want exactly 4 (countback honored)", len(earnings))
	}

	reported := 0
	for i, e := range earnings {
		assertEqual(t, e.Symbol, TestStockSymbol, "symbol should match")

		// Every returned report must be historical (report date not in the
		// future) — the buggy upcoming-window behavior returned future rows.
		assertTimeInPast(t, e.ReportDate, "earnings report date")

		// Fiscal quarter should be 1-4
		if e.FiscalQuarter < 1 || e.FiscalQuarter > 4 {
			t.Errorf("Fiscal quarter %d should be 1-4", e.FiscalQuarter)
		}

		// Fiscal year should be reasonable (2000-2030)
		if e.FiscalYear < 2000 || e.FiscalYear > 2030 {
			t.Errorf("Fiscal year %d seems unreasonable", e.FiscalYear)
		}

		// Estimates exist for every listed quarter, past or pending.
		if e.EstimatedEPS == nil {
			t.Errorf("earnings[%d]: EstimatedEPS is nil, want a value", i)
		}
		if e.ReportedEPS != nil {
			reported++
		}
	}

	// Reported EPS must be present for at least 3 of the last 4 reports:
	// only the most recent one may still be inside the providers' data lag
	// (a report from the last few days can briefly carry a null EPS).
	if reported < 3 {
		t.Errorf("only %d of 4 reports carry ReportedEPS; want at least 3 (historical reports must have real EPS)", reported)
	}
}

func TestStocks_News(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	// Get recent news
	news, _, err := client.Stocks.News(ctx, TestStockSymbol,
		stocks.WithNewsWindow(stocks.LastN(5)),
	)
	assertNoError(t, err, "News")

	// News may be empty if no recent news exists - this is OK
	// If there is news, validate it
	for _, article := range news {
		assertEqual(t, article.Symbol, TestStockSymbol, "symbol should match")

		// Headlines should not be empty
		if article.Headline == "" {
			t.Error("News headline should not be empty")
		}
	}
}

func TestStocks_BulkCandles(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	symbols := []string{TestStockSymbol, TestStockSymbol2}
	candles, _, err := client.Stocks.BulkCandles(ctx, symbols)
	assertNoError(t, err, "BulkCandles")
	assertNotEmpty(t, candles, "bulk candles should not be empty")

	// Should have candles for all symbols
	foundSymbols := make(map[string]bool)
	for _, c := range candles {
		foundSymbols[c.Symbol] = true
		assertValidOHLC(t, c.Open, c.High, c.Low, c.Close)
		assertPositive(t, c.Open, "open price for "+c.Symbol)
	}

	for _, sym := range symbols {
		if !foundSymbols[sym] {
			t.Errorf("Missing bulk candle for symbol %s", sym)
		}
	}
}

// TestStocks_BulkCandles_MarketWideSnapshot exercises the empty-symbol snapshot
// form against the live API. It is opt-in because it is by far the most
// expensive request the SDK can make: the API answers with a candle for every
// symbol it covers, and a single call was enough to exhaust a test account's
// daily credit allotment while this feature was being investigated. Run it
// deliberately:
//
//	MARKETDATA_RUN_MARKET_SNAPSHOT=1 go test ./integration/... -tags=integration \
//	    -run TestStocks_BulkCandles_MarketWideSnapshot -v
func TestStocks_BulkCandles_MarketWideSnapshot(t *testing.T) {
	if os.Getenv("MARKETDATA_RUN_MARKET_SNAPSHOT") == "" {
		t.Skip("set MARKETDATA_RUN_MARKET_SNAPSHOT=1 to run: this request returns the whole market and costs credits accordingly")
	}
	client := setupClient(t)
	ctx := testContext(t)

	candles, _, err := client.Stocks.BulkCandles(ctx, nil, stocks.WithSnapshot(true))
	assertNoError(t, err, "BulkCandles market-wide snapshot")
	assertNotEmpty(t, candles, "market-wide snapshot should not be empty")

	// The whole point is breadth: a market snapshot covers far more symbols
	// than any explicit list a caller would have passed.
	symbols := make(map[string]bool, len(candles))
	for _, c := range candles {
		symbols[c.Symbol] = true
	}
	if len(symbols) < 100 {
		t.Errorf("snapshot covered %d symbols, want many more — this does not look market-wide", len(symbols))
	}
}

func TestStocks_BulkCandles_WithDate(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	date := historicalDate()
	symbols := []string{TestStockSymbol, TestStockSymbol2}

	candles, _, err := client.Stocks.BulkCandles(ctx, symbols,
		stocks.WithBulkDate(date),
	)
	// BulkCandles with historical date may not be available on all API plans
	if err != nil {
		t.Skipf("BulkCandles with date not available: %v", err)
	}
	assertNotEmpty(t, candles, "bulk candles should not be empty")

	// All candles should be for the requested date (within tolerance)
	dateStart := date.Truncate(24 * time.Hour)
	dateEnd := dateStart.Add(48 * time.Hour) // Allow some tolerance

	for _, c := range candles {
		if c.Time.Before(dateStart) || c.Time.After(dateEnd) {
			t.Errorf("Candle for %s at %v outside expected date %v",
				c.Symbol, c.Time, date)
		}
	}
}
