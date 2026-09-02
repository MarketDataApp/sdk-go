//go:build integration

package integration

import (
	"math"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/markets"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

// Tracked API discrepancies.
//
// Each test below documents a live API behavior that objectively contradicts
// its documented contract, verified with a raw curl repro and filed as a
// GitHub issue. The test is skipped (referencing the issue) but carries the
// STRICT assertion that the fixed API should satisfy, so it starts passing —
// and the skip can be removed — once the API is corrected.

// TestDiscrepancy_StatusHistoryCountback tracks:
//
//	markets/status-history returns countback+1 rows (and includes the `to`
//	date), while stocks/candles and funds/candles return exactly `countback`
//	rows for the same parameter.
//
// Verified live 2026-07-11 with identical countback=3&to=2024-06-14:
//
//	curl -H "Authorization: Bearer $T" \
//	  "https://api.marketdata.app/v1/stocks/candles/D/AAPL/?countback=3&to=2024-06-14"
//	  -> 3 rows: 2024-06-11, 06-12, 06-13   (excludes `to`)
//	curl -H "Authorization: Bearer $T" \
//	  "https://api.marketdata.app/v1/markets/status/?countback=3&to=2024-06-14"
//	  -> 4 rows: 2024-06-11, 06-12, 06-13, 06-14   (includes `to`, N+1)
//
// Issue: https://github.com/MarketData-App/api/issues/349
func TestDiscrepancy_StatusHistoryCountback(t *testing.T) {
	t.Skip("API discrepancy: markets/status-history returns countback+1 rows — https://github.com/MarketData-App/api/issues/349")

	client := setupClient(t)
	ctx := liveContext(t, 30*time.Second)

	const n = 3
	to := time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC)
	hist, _, err := liveCall(t, "status-history countback strict", func() ([]markets.MarketStatus, *marketdata.Response, error) {
		return client.Markets.StatusHistory(ctx, markets.WithHistoryWindow(markets.LastNUntil(n, to)))
	})
	assertNoError(t, err, "status-history countback")
	// When the API is fixed, countback=N returns exactly N rows, consistent with
	// the stocks/funds candle endpoints.
	assertEqual(t, len(hist), n, "countback=N should return exactly N status rows")
}

// TestDiscrepancy_OptionQuotesCountback tracks:
//
//	options/quotes ignores the `to` anchor when a countback is present and
//	returns the contract's OLDEST n quotes instead of the n ending at `to`.
//	The same endpoint honors from/to correctly, and stocks/funds candles honor
//	countback correctly, so the defect is specific to countback here.
//
// Verified live 2026-08-11 on AAPL260821C00300000 (listed 2025-08-11):
//
//	?countback=3&to=2026-08-11 -> 2025-08-11, 08-12, 08-13
//	?countback=3&to=2026-08-08 -> 2025-08-11, 08-12, 08-13   (same rows)
//	?countback=3&to=2025-08-13 -> 2025-08-11, 08-12, 08-13   (same rows)
//	?from=2026-08-05&to=2026-08-11 -> 2026-08-05, 08-06, 08-07, 08-10  (correct)
//
// A bare countback (no `to`) is ignored outright and yields a single current
// quote; the SDK anchors it with to=today, which is why QuoteLastN returns n
// rows at all.
func TestDiscrepancy_OptionQuotesCountback(t *testing.T) {
	t.Skip("API discrepancy: options/quotes countback ignores `to` and returns the oldest n quotes")

	client := setupClient(t)
	ctx := liveContext(t, 30*time.Second)

	chain, _, err := client.Options.Chain(ctx, TestStockSymbol, options.WithSide(options.SideCall))
	assertNoError(t, err, "chain for countback discrepancy")
	assertNotNil(t, chain, "chain should not be nil")
	assertNotEmpty(t, chain.Options, "chain should not be empty")
	symbol := chain.Options[0].OptionSymbol

	const n = 3
	to := time.Now().AddDate(0, 0, -1)
	quotes, _, err := client.Options.QuoteHistory(ctx, symbol,
		options.WithOptionQuoteWindow(options.QuoteLastNUntil(n, to)))
	assertNoError(t, err, "option quotes countback")
	assertEqual(t, len(quotes), n, "countback=N should return exactly N quotes")

	// When the API is fixed, the rows end at `to` rather than starting at the
	// contract's first traded day.
	last := quotes[len(quotes)-1].Updated
	if last.Before(to.AddDate(0, 0, -14)) {
		t.Errorf("newest quote is %s, more than two weeks before the requested to=%s — "+
			"countback is still anchored to the start of the contract's history",
			last.Format("2006-01-02"), to.Format("2006-01-02"))
	}
}

// TestDiscrepancy_ChainStrikeZero is a RESOLVED discrepancy, kept as a
// regression guard. It tracked:
//
//	options/chain reports "strike":0 for a large share of its rows — every
//	deep-in-the-money contract — while the same request narrowed by a strike
//	filter returns the true values. The contracts are real and priced; only
//	the strike field is zeroed. The OCC symbol in the same row carries the
//	correct strike, which is how the defect is provable from one response.
//
// Verified live 2026-08-20:
//
//	curl -H "Authorization: Bearer $T" \
//	  "https://api.marketdata.app/v1/options/chain/AAPL/"
//	  -> 198 contracts, 83 of them with strike 0, e.g.
//	     optionSymbol "AAPL260821C00110000" (strike 110 per OCC)
//	     strike 0, bid 205.85
//	curl -H "Authorization: Bearer $T" \
//	  "https://api.marketdata.app/v1/options/chain/AAPL/?strikeLimit=3"
//	  -> strikes [315, 317.5, 320]   (correct when filtered)
//
// The defect propagates into the endpoint's own filters, which match against
// the same zeroed field:
//
//	?expiration=2026-10-02&strike=315  -> 404 no_data, though the contract
//	                                      AAPL261002C00315000 is listed
//	maxStrike=190                      -> returns contracts whose real
//	                                      strikes are 220, 230, 235, 240, 255
//
// RESOLVED 2026-08-20, roughly a day after it was observed: every symbol
// probed (AAPL, TSLA, SPY, with and without an expiration filter) now
// reports real strikes, and this assertion passes. The skip is removed so it
// guards the fix. If it fails again the defect was intermittent rather than
// corrected, which is worth knowing — the observation above is kept in full
// for that reason.
//
// Two claims in the original write-up did NOT survive re-verification and
// are corrected here. The strike filters were said to "match against the
// same zeroed field", evidenced by maxStrike=190 returning strikes of 220
// to 255. That evidence came from a test run while an experimental OCC
// derivation was active (057ab31, reverted in 559630c), which turned the
// API's zeros into real strikes and so broke a ceiling the API had applied
// correctly to its own data. Re-probed with the real wire syntax — the SDK
// sends strike=<=190, not a maxStrike parameter — the expression filters
// work exactly: <=190 returns 110..190, >=350 returns 350..415, 190-220
// returns 190..220. Likewise ?strike=315 returning 404 was the zeroed field,
// not a filter defect. There was one defect here, not three.
func TestDiscrepancy_ChainStrikeZero(t *testing.T) {

	client := setupClient(t)
	ctx := liveContext(t, 60*time.Second)

	chain, _, err := liveCall(t, "chain strike strict", func() (*options.OptionsChain, *marketdata.Response, error) {
		return client.Options.Chain(ctx, TestStockSymbol)
	})
	assertNoError(t, err, "chain")
	assertNotEmpty(t, chain.Options, "chain options")

	// When the API is fixed, every listed contract carries its real strike.
	for _, o := range chain.Options {
		assertPositive(t, o.Strike, "strike for "+o.OptionSymbol)
	}
}

// TestDiscrepancy_ChainDeltaDroppedOnNullDelta tracks:
//
//	options/chain drops a delta filter entirely whenever the chain it
//	fetched holds a contract with a null delta. The unfiltered chain comes
//	back with a 200 and no signal, so every contract in it looks like a
//	legitimate answer.
//
// First seen 2026-08-20 against AAPL, and read at the time as a side
// limitation:
//
//	?delta=0.30&side=call   -> 1 contract,   delta 0.338   (works)
//	?delta=-0.30&side=call  -> 1 contract,   delta 0.338   (sign ignored, as documented)
//	?delta=0.30&side=put    -> 99 contracts, deltas null/-0.008  (filter ignored)
//	?delta=-0.30&side=put   -> 99 contracts, same           (filter ignored)
//	?delta=0.30             -> 198 contracts, deltas ~0.99  (filter ignored)
//
// The "call side only" reading of that run was a misdiagnosis: it named the
// symptom, not the cause. Re-probed 2026-08-26, the same three queries
// returned 1, 1 and 2 contracts, all correctly filtered — the put side
// works. What differs is the data. Per
// https://github.com/MarketData-App/api/issues/352 the filter is skipped
// wholesale whenever ANY contract in the fetched chain has a null delta,
// and the 2026-08-20 run shows exactly that (the unfiltered put rows carry
// "deltas null/-0.008"). Null greeks come and go with liquidity and listing
// age, so the defect is intermittent per symbol and expiration rather than
// tied to a side.
//
// The skip therefore stays even though the assertions would pass today:
// the server-side early return is still in place, so an un-skipped test
// would be green on the data and red on the defect at random. It is lifted
// when issue 352 closes.
//
// Issue: https://github.com/MarketData-App/api/issues/352
func TestDiscrepancy_ChainDeltaDroppedOnNullDelta(t *testing.T) {
	t.Skip("API discrepancy: options/chain drops the delta filter — https://github.com/MarketData-App/api/issues/352")

	client := setupClient(t)
	ctx := liveContext(t, 60*time.Second)

	// When the API is fixed, the filter applies to every chain, including
	// those holding a null-delta contract, on both sides and with none.
	for _, tc := range []struct {
		name string
		opts []options.ChainOption
	}{
		{"put side", []options.ChainOption{options.WithStrike(options.ByDelta(0.30)), options.WithSide(options.SidePut)}},
		{"no side", []options.ChainOption{options.WithStrike(options.ByDelta(0.30))}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chain, _, err := liveCall(t, "chain delta "+tc.name, func() (*options.OptionsChain, *marketdata.Response, error) {
				return client.Options.Chain(ctx, TestStockSymbol, tc.opts...)
			})
			assertNoError(t, err, "chain delta")
			assertNotEmpty(t, chain.Options, "delta-filtered options")
			for _, o := range chain.Options {
				if math.Abs(math.Abs(o.Delta)-0.30) > 0.25 {
					t.Errorf("delta filter 0.30 returned %s at delta %.4f", o.OptionSymbol, o.Delta)
				}
			}
		})
	}
}

// TestDiscrepancy_LimitOffsetIgnoredOnCandles tracks:
//
//	the universal limit and offset parameters are honored by some endpoints
//	and silently ignored by the candles endpoints, with no way for a caller
//	to tell which they got.
//
// Verified live 2026-08-20:
//
//	?from=2026-08-01&to=2026-08-19&limit=3           -> 13 candles
//	?from=2026-08-01&to=2026-08-19&limit=3&offset=2  -> the same 13
//	/v1/options/expirations/AAPL/?limit=3            -> 3   (honored)
//	/v1/markets/status/?countback=5&limit=2          -> 2   (honored)
//
// offset is the dangerous half: paging over candles returns the identical
// full set on every page, so a loop either duplicates every row for as long
// as it runs or terminates on a condition that was never true. WithLimit and
// WithOffset now say so in their godoc.
//
// Issue: https://github.com/MarketData-App/api/issues/375
func TestDiscrepancy_LimitOffsetIgnoredOnCandles(t *testing.T) {
	t.Skip("API discrepancy: candles ignores limit/offset — https://github.com/MarketData-App/api/issues/375")

	ctx := liveContext(t, 60*time.Second)
	client, err := marketdata.NewClient(marketdata.WithToken(liveToken()), marketdata.WithLimit(3))
	assertNoError(t, err, "NewClient with limit")

	// When the API is fixed, a client-level limit caps candles too.
	candles, _, err := liveCall(t, "candles with limit", func() ([]stocks.Candle, *marketdata.Response, error) {
		return client.Stocks.Candles(ctx, TestStockSymbol, stocks.WithCandleWindow(stocks.LastN(13)))
	})
	assertNoError(t, err, "candles with limit")
	if len(candles) > 3 {
		t.Errorf("limit=3 returned %d candles", len(candles))
	}
}
