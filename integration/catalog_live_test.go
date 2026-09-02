//go:build integration

package integration

import (
	"errors"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/apicatalog"
	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/funds"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/markets"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

// probe runs one focused live call that sets a single catalogued parameter and
// returns only whether the API accepted it.
type probe func() (*marketdata.Response, error)

// TestCatalog_ParamsAcceptedLive is the catalog-driven acceptance test. For
// every Query and Path parameter in apicatalog.All() it issues a live SDK call
// that sets that parameter with a realistic valid value and asserts the API
// ACCEPTS it — success or a NoData (404), never an HTTP 400 rejection. The
// catalog drives coverage two ways:
//
//   - completeness: a catalogued Query/Path param with no probe fails the test,
//     so the suite cannot silently skip a parameter the API supports; and
//   - acceptance: each probe's live response must not be a 400.
//
// Residual params (SDK-owned universal knobs) are documented and skipped.
func TestCatalog_ParamsAcceptedLive(t *testing.T) {
	client := setupClient(t)
	ctx := liveContext(t, 5*time.Minute)

	fx := resolveOptionsFixture(t, ctx, client)
	t.Logf("options fixture: exp=%s underlying=%.2f atm=%.2f strikes=%d occCall=%s",
		fx.exp.Format("2006-01-02"), fx.underlying, fx.atm, len(fx.strikes), fx.occCall)

	nextYear := time.Now().Year() + 1
	recentFrom := time.Now().AddDate(0, 0, -10)
	recentTo := time.Now().AddDate(0, 0, -3)

	probes := map[string]probe{
		// ---- stocks/quotes ----
		"stocks/quotes|symbols": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.Quotes(ctx, []string{TestStockSymbol, TestStockSymbol2})
			return r, e
		},
		"stocks/quotes|52week": func() (*marketdata.Response, error) {
			// 52week is single-symbol only, so the SDK routes it through
			// Quote (which also rides the bulkquotes endpoint).
			_, r, e := client.Stocks.Quote(ctx, TestStockSymbol, stocks.WithFiftyTwoWeek(true))
			return r, e
		},
		"stocks/quotes|extended": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.Quotes(ctx, []string{TestStockSymbol}, stocks.WithQuotesExtended(true))
			return r, e
		},
		"stocks/quotes|candle": func() (*marketdata.Response, error) {
			// candle is honored on both the single- and multi-symbol forms
			// (unlike 52week above), so either would do; Quote exercises the
			// QuoteOption spelling, which is the one users reach for first.
			_, r, e := client.Stocks.Quote(ctx, TestStockSymbol, stocks.WithCandle(true))
			return r, e
		},
		// ---- stocks/candles ----
		"stocks/candles|symbol": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.Candles(ctx, TestStockSymbol, stocks.WithResolution(stocks.ResolutionDaily), stocks.WithCandleWindow(stocks.OnDate(liveHistDay)))
			return r, e
		},
		"stocks/candles|resolution": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.Candles(ctx, TestStockSymbol, stocks.WithResolution(stocks.ResolutionWeekly), stocks.WithCandleWindow(stocks.LastN(4)))
			return r, e
		},
		"stocks/candles|date": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.Candles(ctx, TestStockSymbol, stocks.WithCandleWindow(stocks.OnDate(liveHistDay)))
			return r, e
		},
		"stocks/candles|from": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.Candles(ctx, TestStockSymbol, stocks.WithCandleWindow(stocks.Since(recentFrom)))
			return r, e
		},
		"stocks/candles|to": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.Candles(ctx, TestStockSymbol, stocks.WithCandleWindow(stocks.Until(liveHistTo)))
			return r, e
		},
		"stocks/candles|countback": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.Candles(ctx, TestStockSymbol, stocks.WithCandleWindow(stocks.LastN(5)))
			return r, e
		},
		"stocks/candles|extended": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.Candles(ctx, TestStockSymbol, stocks.WithCandleExtended(true), stocks.WithCandleWindow(stocks.OnDate(liveHistDay)))
			return r, e
		},
		"stocks/candles|adjustsplits": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.Candles(ctx, TestStockSymbol, stocks.WithCandleAdjustSplits(true), stocks.WithCandleWindow(stocks.OnDate(liveHistDay)))
			return r, e
		},
		"stocks/candles|adjustdividends": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.Candles(ctx, TestStockSymbol, stocks.WithCandleAdjustDividends(false), stocks.WithCandleWindow(stocks.OnDate(liveHistDay)))
			return r, e
		},
		// ---- stocks/bulkcandles ----
		"stocks/bulkcandles|symbols": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.BulkCandles(ctx, []string{TestStockSymbol, TestStockSymbol2})
			return r, e
		},
		"stocks/bulkcandles|resolution": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.BulkCandles(ctx, []string{TestStockSymbol}, stocks.WithBulkResolution(stocks.ResolutionDaily))
			return r, e
		},
		"stocks/bulkcandles|date": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.BulkCandles(ctx, []string{TestStockSymbol}, stocks.WithBulkDate(liveHistDay))
			return r, e
		},
		"stocks/bulkcandles|adjustsplits": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.BulkCandles(ctx, []string{TestStockSymbol}, stocks.WithAdjustSplits(true))
			return r, e
		},
		"stocks/bulkcandles|adjustdividends": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.BulkCandles(ctx, []string{TestStockSymbol}, stocks.WithAdjustDividends(false))
			return r, e
		},
		"stocks/bulkcandles|snapshot": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.BulkCandles(ctx, []string{TestStockSymbol}, stocks.WithSnapshot(true))
			return r, e
		},
		// ---- stocks/prices ----
		"stocks/prices|symbols": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.Prices(ctx, []string{TestStockSymbol})
			return r, e
		},
		"stocks/prices|extended": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.Prices(ctx, []string{TestStockSymbol}, stocks.WithPriceExtended(true))
			return r, e
		},
		// ---- stocks/earnings ----
		"stocks/earnings|symbol": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.Earnings(ctx, TestStockSymbol, stocks.WithEarningsWindow(stocks.LastN(1)))
			return r, e
		},
		"stocks/earnings|date": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.Earnings(ctx, TestStockSymbol, stocks.WithEarningsWindow(stocks.OnDate(liveHistDay)))
			return r, e
		},
		"stocks/earnings|from": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.Earnings(ctx, TestStockSymbol, stocks.WithEarningsWindow(stocks.Since(time.Now().AddDate(-1, 0, 0))))
			return r, e
		},
		"stocks/earnings|to": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.Earnings(ctx, TestStockSymbol, stocks.WithEarningsWindow(stocks.Until(liveHistTo)))
			return r, e
		},
		"stocks/earnings|countback": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.Earnings(ctx, TestStockSymbol, stocks.WithEarningsWindow(stocks.LastN(2)))
			return r, e
		},
		"stocks/earnings|report": func() (*marketdata.Response, error) {
			// This probe asserts the API ACCEPTS report, which is all
			// TestCatalog_ParamsAcceptedLive claims. The parameter is inert
			// server-side — the schema declares it and no handler reads it
			// — as the catalog row and openapi_test's note both record.
			_, r, e := client.Stocks.Earnings(ctx, TestStockSymbol, stocks.WithEarningsReport("2024-Q1"))
			return r, e
		},
		// ---- stocks/news ----
		"stocks/news|symbol": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.News(ctx, TestStockSymbol, stocks.WithNewsWindow(stocks.LastN(1)))
			return r, e
		},
		"stocks/news|date": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.News(ctx, TestStockSymbol, stocks.WithNewsWindow(stocks.OnDate(liveHistDay)))
			return r, e
		},
		"stocks/news|from": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.News(ctx, TestStockSymbol, stocks.WithNewsWindow(stocks.Since(time.Now().AddDate(0, 0, -5))))
			return r, e
		},
		"stocks/news|to": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.News(ctx, TestStockSymbol, stocks.WithNewsWindow(stocks.Until(liveHistTo)))
			return r, e
		},
		"stocks/news|countback": func() (*marketdata.Response, error) {
			_, r, e := client.Stocks.News(ctx, TestStockSymbol, stocks.WithNewsWindow(stocks.LastN(2)))
			return r, e
		},
		// ---- funds/candles ----
		"funds/candles|symbol": func() (*marketdata.Response, error) {
			_, r, e := client.Funds.Candles(ctx, TestFundSymbol, funds.WithResolution(funds.ResolutionDaily), funds.WithCandleWindow(funds.OnDate(liveHistDay)))
			return r, e
		},
		"funds/candles|resolution": func() (*marketdata.Response, error) {
			_, r, e := client.Funds.Candles(ctx, TestFundSymbol, funds.WithResolution(funds.ResolutionWeekly), funds.WithCandleWindow(funds.LastN(4)))
			return r, e
		},
		"funds/candles|date": func() (*marketdata.Response, error) {
			_, r, e := client.Funds.Candles(ctx, TestFundSymbol, funds.WithCandleWindow(funds.OnDate(liveHistDay)))
			return r, e
		},
		"funds/candles|from": func() (*marketdata.Response, error) {
			_, r, e := client.Funds.Candles(ctx, TestFundSymbol, funds.WithCandleWindow(funds.Since(recentFrom)))
			return r, e
		},
		"funds/candles|to": func() (*marketdata.Response, error) {
			_, r, e := client.Funds.Candles(ctx, TestFundSymbol, funds.WithCandleWindow(funds.Until(liveHistTo)))
			return r, e
		},
		"funds/candles|countback": func() (*marketdata.Response, error) {
			_, r, e := client.Funds.Candles(ctx, TestFundSymbol, funds.WithCandleWindow(funds.LastN(5)))
			return r, e
		},
		// ---- options/chain ----
		"options/chain|symbol": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)))
			return r, e
		},
		"options/chain|expiration": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)))
			return r, e
		},
		"options/chain|dte": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.InDTE(30)))
			return r, e
		},
		"options/chain|month": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.InMonth(int(fx.exp.Month()))), options.WithStrikeLimit(1))
			return r, e
		},
		"options/chain|year": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.InYear(nextYear)), options.WithStrikeLimit(1))
			return r, e
		},
		"options/chain|strike": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithStrike(options.Strike(fx.atm)))
			return r, e
		},
		"options/chain|delta": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithStrike(options.ByDelta(0.3)))
			return r, e
		},
		"options/chain|date": func() (*marketdata.Response, error) {
			// date is the historical as-of snapshot (independent of expiry).
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithChainDate(liveHistDay))
			return r, e
		},
		"options/chain|from": func() (*marketdata.Response, error) {
			// from/to filter expirations (a window around the fixture expiration).
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.ExpirationBetween(fx.exp.AddDate(0, 0, -7), fx.exp.AddDate(0, 0, 7))))
			return r, e
		},
		"options/chain|to": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.ExpirationBetween(fx.exp.AddDate(0, 0, -7), fx.exp.AddDate(0, 0, 7))))
			return r, e
		},
		"options/chain|side": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithSide(options.SideCall))
			return r, e
		},
		"options/chain|strikeLimit": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithStrikeLimit(2))
			return r, e
		},
		"options/chain|range": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithRange(options.MoneynessITM))
			return r, e
		},
		"options/chain|minBid": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithMinBid(0.01))
			return r, e
		},
		"options/chain|maxBid": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithMaxBid(100000))
			return r, e
		},
		"options/chain|minAsk": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithMinAsk(0.01))
			return r, e
		},
		"options/chain|maxAsk": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithMaxAsk(100000))
			return r, e
		},
		"options/chain|maxBidAskSpread": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithMaxBidAskSpread(100000))
			return r, e
		},
		"options/chain|maxBidAskSpreadPct": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithMaxBidAskSpreadPct(100000))
			return r, e
		},
		"options/chain|minOpenInterest": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithMinOpenInterest(1))
			return r, e
		},
		"options/chain|minVolume": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithMinVolume(1))
			return r, e
		},
		"options/chain|weekly": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithExpirationTypes(options.IncludeExpirationTypes(options.Weekly)))
			return r, e
		},
		"options/chain|monthly": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithExpirationTypes(options.IncludeExpirationTypes(options.Monthly)))
			return r, e
		},
		"options/chain|quarterly": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithExpirationTypes(options.IncludeExpirationTypes(options.Quarterly)))
			return r, e
		},
		"options/chain|nonstandard": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithNonstandard(true))
			return r, e
		},
		"options/chain|am": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithAM(true))
			return r, e
		},
		"options/chain|pm": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithPM(true))
			return r, e
		},
		// ---- options/quotes (single contract) ----
		"options/quotes|optionSymbol": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Quote(ctx, fx.occCall)
			return r, e
		},
		"options/quotes|date": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Quote(ctx, fx.occCall, options.WithOptionQuoteWindow(options.QuoteOnDate(recentTo)))
			return r, e
		},
		"options/quotes|from": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Quote(ctx, fx.occCall, options.WithOptionQuoteWindow(options.QuoteRange(recentFrom, recentTo)))
			return r, e
		},
		"options/quotes|to": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Quote(ctx, fx.occCall, options.WithOptionQuoteWindow(options.QuoteRange(recentFrom, recentTo)))
			return r, e
		},
		"options/quotes|countback": func() (*marketdata.Response, error) {
			// Anchored with QuoteLastNUntil, not bare QuoteLastN: this
			// endpoint ignores a countback that arrives without to=
			// (verified live), so the bare form would probe nothing.
			_, r, e := client.Options.Quote(ctx, fx.occCall, options.WithOptionQuoteWindow(options.QuoteLastNUntil(2, recentTo)))
			return r, e
		},
		// ---- options/expirations ----
		"options/expirations|symbol": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Expirations(ctx, TestStockSymbol)
			return r, e
		},
		"options/expirations|strike": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Expirations(ctx, TestStockSymbol, options.WithExpirationStrike(fx.atm))
			return r, e
		},
		"options/expirations|date": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Expirations(ctx, TestStockSymbol, options.WithExpirationDate(liveHistDay))
			return r, e
		},
		// ---- options/lookup ----
		"options/lookup|query": func() (*marketdata.Response, error) {
			_, r, e := client.Options.Lookup(ctx, TestStockSymbol, fx.exp, fx.atm, options.Call)
			return r, e
		},
		// ---- markets/status ----
		"markets/status|date": func() (*marketdata.Response, error) {
			_, r, e := client.Markets.Status(ctx, markets.WithDate(liveHistDay))
			return r, e
		},
		"markets/status|country": func() (*marketdata.Response, error) {
			_, r, e := client.Markets.Status(ctx, markets.WithCountry("US"))
			return r, e
		},
		// ---- markets/status-history ----
		"markets/status-history|from": func() (*marketdata.Response, error) {
			_, r, e := client.Markets.StatusHistory(ctx, markets.WithHistoryWindow(markets.Since(recentFrom)))
			return r, e
		},
		"markets/status-history|to": func() (*marketdata.Response, error) {
			_, r, e := client.Markets.StatusHistory(ctx, markets.WithHistoryWindow(markets.Until(liveHistTo)))
			return r, e
		},
		"markets/status-history|countback": func() (*marketdata.Response, error) {
			_, r, e := client.Markets.StatusHistory(ctx, markets.WithHistoryWindow(markets.LastN(3)))
			return r, e
		},
		"markets/status-history|country": func() (*marketdata.Response, error) {
			_, r, e := client.Markets.StatusHistory(ctx, markets.WithHistoryWindow(markets.Between(liveHistFrom, liveHistTo)), markets.WithCountry("US"))
			return r, e
		},
		// ---- utilities (no request params; exercise the endpoint) ----
		"utilities/status|-": func() (*marketdata.Response, error) {
			_, r, e := client.Utilities.Status(ctx)
			return r, e
		},
		"utilities/headers|-": func() (*marketdata.Response, error) {
			_, r, e := client.Utilities.Headers(ctx)
			return r, e
		},
		"utilities/user|-": func() (*marketdata.Response, error) {
			_, r, e := client.Utilities.User(ctx)
			return r, e
		},
		// ---- universal (client-level) ----
		"universal|columns": func() (*marketdata.Response, error) {
			// Deliberately does NOT name "s": that is what makes the envelope
			// repair in the HTTP client run, so this probe covers it against
			// the live API. Naming "s" here left the repair with no live
			// coverage at all, which is how it shipped as a no-op.
			cc, err := marketdata.NewClient(
				marketdata.WithToken(liveToken()),
				marketdata.WithColumns("symbol", "bid", "ask", "last", "mid", "volume", "updated"),
			)
			if err != nil {
				return nil, err
			}
			_, r, e := cc.Stocks.Quote(ctx, TestStockSymbol)
			return r, e
		},
		"universal|mode": func() (*marketdata.Response, error) {
			cc, err := marketdata.NewClient(marketdata.WithToken(liveToken()), marketdata.WithMode(marketdata.ModeDelayed))
			if err != nil {
				return nil, err
			}
			_, r, e := cc.Stocks.Quote(ctx, TestStockSymbol)
			return r, e
		},
		"universal|maxage": func() (*marketdata.Response, error) {
			// maxage pairs with mode=cached; chain supports the cached feed.
			cc, err := marketdata.NewClient(marketdata.WithToken(liveToken()), marketdata.WithMode(marketdata.ModeCached), marketdata.WithMaxAge("5min"))
			if err != nil {
				return nil, err
			}
			_, r, e := cc.Options.Chain(ctx, TestStockSymbol)
			return r, e
		},
		"universal|limit": func() (*marketdata.Response, error) {
			cc, err := marketdata.NewClient(marketdata.WithToken(liveToken()), marketdata.WithLimit(5))
			if err != nil {
				return nil, err
			}
			_, r, e := cc.Stocks.Candles(ctx, TestStockSymbol, stocks.WithCandleWindow(stocks.LastN(10)))
			return r, e
		},
		"universal|offset": func() (*marketdata.Response, error) {
			cc, err := marketdata.NewClient(marketdata.WithToken(liveToken()), marketdata.WithLimit(5), marketdata.WithOffset(2))
			if err != nil {
				return nil, err
			}
			_, r, e := cc.Stocks.Candles(ctx, TestStockSymbol, stocks.WithCandleWindow(stocks.LastN(10)))
			return r, e
		},
	}

	tested := 0
	for _, p := range apicatalog.All() {
		key := p.Endpoint + "|" + p.Name
		switch p.Kind {
		case apicatalog.Residual:
			t.Logf("residual (SDK-owned, not user-rejectable): %s (%s)", key, p.SDKPath)
			continue
		case apicatalog.Query, apicatalog.Path:
			fn, ok := probes[key]
			if !ok {
				t.Errorf("NO acceptance probe for catalogued %s param %s (SDK path: %s)", p.Kind, key, p.SDKPath)
				continue
			}
			resp, err := liveProbe(t, key, fn)
			assertAccepted(t, key, resp, err)
			tested++
		}
	}
	t.Logf("catalog acceptance: exercised %d live parameter probes", tested)
}

// assertAccepted fails only when the API rejected the parameter with an HTTP
// 400 (or the SDK rejected the probe value client-side, which would be a test
// bug). Success and NoData (404) both count as accepted; other errors
// (network/server/plan) are logged but are not parameter rejections.
func assertAccepted(t *testing.T, key string, resp *marketdata.Response, err error) {
	t.Helper()
	var valErr *marketdata.ValidationError
	if errors.As(err, &valErr) {
		t.Errorf("%s: SDK rejected the probe VALUE client-side (test bug): %v", key, err)
		return
	}
	var badReq *marketdata.BadRequestError
	if errors.As(err, &badReq) {
		t.Errorf("%s: API REJECTED the parameter with HTTP 400: %v", key, err)
		return
	}
	if err != nil {
		t.Logf("%s: accepted (non-400 error, not a param rejection): %v", key, err)
		return
	}
	if resp != nil && resp.NoData {
		t.Logf("%s: accepted (HTTP 404 NoData)", key)
	}
}
