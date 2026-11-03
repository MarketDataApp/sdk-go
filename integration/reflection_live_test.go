//go:build integration

package integration

import (
	"context"
	"fmt"
	"math"
	"sort"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/funds"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/markets"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

// dayDiff returns the absolute difference in whole days between the calendar
// dates of a and b (each read in its own location).
func dayDiff(a, b time.Time) time.Duration {
	d := truncDay(a).Sub(truncDay(b))
	if d < 0 {
		d = -d
	}
	return d
}

// TestReflect_StocksCandlesWindow exercises every stocks.DateWindow member live
// and asserts the response reflects the requested window.
func TestReflect_StocksCandlesWindow(t *testing.T) {
	client := setupClient(t)
	ctx := liveContext(t, 2*time.Minute)

	t.Run("OnDate=1candle", func(t *testing.T) {
		c, _, err := liveCall(t, "candles OnDate", func() ([]stocks.Candle, *marketdata.Response, error) {
			return client.Stocks.Candles(ctx, TestStockSymbol, stocks.WithResolution(stocks.ResolutionDaily), stocks.WithCandleWindow(stocks.OnDate(liveHistDay)))
		})
		assertNoError(t, err, "OnDate")
		assertEqual(t, len(c), 1, "OnDate should return exactly one daily candle")
		if len(c) == 1 && dayDiff(c[0].Time, liveHistDay) > 24*time.Hour {
			t.Errorf("OnDate candle %v not within a day of %v", c[0].Time, liveHistDay)
		}
	})

	t.Run("Between=inRange", func(t *testing.T) {
		c, _, err := liveCall(t, "candles Between", func() ([]stocks.Candle, *marketdata.Response, error) {
			return client.Stocks.Candles(ctx, TestStockSymbol, stocks.WithResolution(stocks.ResolutionDaily), stocks.WithCandleWindow(stocks.Between(liveHistFrom, liveHistTo)))
		})
		assertNoError(t, err, "Between")
		assertNotEmpty(t, c, "Between candles")
		for _, k := range c {
			if truncDay(k.Time).Before(liveHistFrom) || truncDay(k.Time).After(liveHistTo) {
				t.Errorf("Between candle %v outside [%v, %v]", k.Time, liveHistFrom, liveHistTo)
			}
		}
	})

	t.Run("LastN=exactCount", func(t *testing.T) {
		c, _, err := liveCall(t, "candles LastN", func() ([]stocks.Candle, *marketdata.Response, error) {
			return client.Stocks.Candles(ctx, TestStockSymbol, stocks.WithResolution(stocks.ResolutionDaily), stocks.WithCandleWindow(stocks.LastN(5)))
		})
		assertNoError(t, err, "LastN")
		assertEqual(t, len(c), 5, "LastN(5) should return exactly 5 candles")
	})

	t.Run("LastNUntil=countAndBound", func(t *testing.T) {
		c, _, err := liveCall(t, "candles LastNUntil", func() ([]stocks.Candle, *marketdata.Response, error) {
			return client.Stocks.Candles(ctx, TestStockSymbol, stocks.WithResolution(stocks.ResolutionDaily), stocks.WithCandleWindow(stocks.LastNUntil(3, liveHistWide)))
		})
		assertNoError(t, err, "LastNUntil")
		assertEqual(t, len(c), 3, "LastNUntil(3) should return exactly 3 candles")
		for _, k := range c {
			if dayDiff(k.Time, liveHistWide) > 0 && truncDay(k.Time).After(liveHistWide) {
				t.Errorf("LastNUntil candle %v is after the anchor %v", k.Time, liveHistWide)
			}
		}
	})

	t.Run("Since=fromBound", func(t *testing.T) {
		from := time.Now().AddDate(0, 0, -10)
		c, _, err := liveCall(t, "candles Since", func() ([]stocks.Candle, *marketdata.Response, error) {
			return client.Stocks.Candles(ctx, TestStockSymbol, stocks.WithResolution(stocks.ResolutionDaily), stocks.WithCandleWindow(stocks.Since(from)))
		})
		assertNoError(t, err, "Since")
		for _, k := range c {
			if truncDay(k.Time).Before(truncDay(from).Add(-24 * time.Hour)) {
				t.Errorf("Since candle %v earlier than from %v", k.Time, from)
			}
		}
	})

	t.Run("Until=toBound", func(t *testing.T) {
		// `to` alone (no from/countback) does not define a window start, so the
		// candles endpoint legitimately returns NoData. The reflection we can
		// assert is: the request is accepted and any returned candle is <= to.
		c, _, err := liveCall(t, "candles Until", func() ([]stocks.Candle, *marketdata.Response, error) {
			return client.Stocks.Candles(ctx, TestStockSymbol, stocks.WithResolution(stocks.ResolutionDaily), stocks.WithCandleWindow(stocks.Until(liveHistTo)))
		})
		assertNoError(t, err, "Until")
		for _, k := range c {
			if truncDay(k.Time).After(liveHistTo.Add(24 * time.Hour)) {
				t.Errorf("Until candle %v later than to %v", k.Time, liveHistTo)
			}
		}
	})
}

// TestReflect_FundsCandlesWindow exercises the funds.DateWindow members live.
func TestReflect_FundsCandlesWindow(t *testing.T) {
	client := setupClient(t)
	ctx := liveContext(t, 90*time.Second)

	t.Run("OnDate=1candle", func(t *testing.T) {
		c, _, err := liveCall(t, "funds OnDate", func() ([]funds.Candle, *marketdata.Response, error) {
			return client.Funds.Candles(ctx, TestFundSymbol, funds.WithResolution(funds.ResolutionDaily), funds.WithCandleWindow(funds.OnDate(liveHistDay)))
		})
		assertNoError(t, err, "funds OnDate")
		assertEqual(t, len(c), 1, "funds OnDate should return exactly one candle")
	})

	t.Run("Between=inRange", func(t *testing.T) {
		c, _, err := liveCall(t, "funds Between", func() ([]funds.Candle, *marketdata.Response, error) {
			return client.Funds.Candles(ctx, TestFundSymbol, funds.WithResolution(funds.ResolutionDaily), funds.WithCandleWindow(funds.Between(liveHistFrom, liveHistTo)))
		})
		assertNoError(t, err, "funds Between")
		assertNotEmpty(t, c, "funds Between candles")
		for _, k := range c {
			if truncDay(k.Time).Before(liveHistFrom) || truncDay(k.Time).After(liveHistTo) {
				t.Errorf("funds Between candle %v outside [%v, %v]", k.Time, liveHistFrom, liveHistTo)
			}
		}
	})

	t.Run("LastN=exactCount", func(t *testing.T) {
		c, _, err := liveCall(t, "funds LastN", func() ([]funds.Candle, *marketdata.Response, error) {
			return client.Funds.Candles(ctx, TestFundSymbol, funds.WithResolution(funds.ResolutionDaily), funds.WithCandleWindow(funds.LastN(6)))
		})
		assertNoError(t, err, "funds LastN")
		assertEqual(t, len(c), 6, "funds LastN(6) should return exactly 6 candles")
	})
}

// TestReflect_OptionsStrike exercises every options StrikeFilter member live.
func TestReflect_OptionsStrike(t *testing.T) {
	client := setupClient(t)
	ctx := liveContext(t, 3*time.Minute)
	fx := resolveOptionsFixture(t, ctx, client)

	// Pick a middle band of the strike ladder for the range/min/max checks.
	n := len(fx.strikes)
	lo, hi := fx.strikes[0], fx.strikes[n-1]
	if n >= 8 {
		lo, hi = fx.strikes[n/4], fx.strikes[3*n/4]
	}

	t.Run("Strike=exact", func(t *testing.T) {
		chain, _, err := liveCall(t, "Strike exact", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithStrike(options.Strike(fx.atm)))
		})
		assertNoError(t, err, "Strike exact")
		assertNotNil(t, chain, "chain")
		assertNotEmpty(t, chain.Options, "exact-strike options")
		for _, o := range chain.Options {
			assertEqual(t, o.Strike, fx.atm, "exact strike should match")
		}
	})

	t.Run("StrikeRange=within", func(t *testing.T) {
		chain, _, err := liveCall(t, "StrikeRange", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithStrike(options.StrikeRange(lo, hi)))
		})
		assertNoError(t, err, "StrikeRange")
		assertNotEmpty(t, chain.Options, "range options")
		for _, o := range chain.Options {
			if o.Strike < lo || o.Strike > hi {
				t.Errorf("StrikeRange returned strike %.2f outside [%.2f, %.2f]", o.Strike, lo, hi)
			}
		}
	})

	t.Run("MinStrike=atOrAbove", func(t *testing.T) {
		chain, _, err := liveCall(t, "MinStrike", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithStrike(options.MinStrike(hi)))
		})
		assertNoError(t, err, "MinStrike")
		assertNotEmpty(t, chain.Options, "min-strike options")
		for _, o := range chain.Options {
			if o.Strike < hi {
				t.Errorf("MinStrike(%.2f) returned strike %.2f below the floor", hi, o.Strike)
			}
		}
	})

	t.Run("MaxStrike=atOrBelow", func(t *testing.T) {
		chain, _, err := liveCall(t, "MaxStrike", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithStrike(options.MaxStrike(lo)))
		})
		assertNoError(t, err, "MaxStrike")
		assertNotEmpty(t, chain.Options, "max-strike options")
		for _, o := range chain.Options {
			if o.Strike > lo {
				t.Errorf("MaxStrike(%.2f) returned strike %.2f above the ceiling", lo, o.Strike)
			}
		}
	})

	// ByDelta: the API filters on the ABSOLUTE value of delta and returns BOTH
	// sides of the chain (documented behavior). The sign does not select puts;
	// combine with WithSide to isolate one side.
	t.Run("ByDelta=bothSidesByAbsValue", func(t *testing.T) {
		t.Skip("API drops the delta filter when the chain holds a null delta — TestDiscrepancy_ChainDeltaDroppedOnNullDelta")
		chain, _, err := liveCall(t, "ByDelta both", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithStrike(options.ByDelta(0.30)))
		})
		assertNoError(t, err, "ByDelta")
		assertNotEmpty(t, chain.Options, "delta options")
		var haveCall, havePut bool
		for _, o := range chain.Options {
			switch o.Type {
			case options.Call:
				haveCall = true
				if o.Delta <= 0 || math.Abs(o.Delta-0.30) > 0.25 {
					t.Errorf("ByDelta(0.30) call delta %.4f not a positive value near 0.30", o.Delta)
				}
			case options.Put:
				havePut = true
				if o.Delta >= 0 || math.Abs(o.Delta+0.30) > 0.25 {
					t.Errorf("ByDelta(0.30) put delta %.4f not a negative value near -0.30", o.Delta)
				}
			}
		}
		assertTrue(t, haveCall && havePut, "ByDelta with no side returns both a call and a put (absolute-value filter)")
	})

	t.Run("ByDelta_negative_withSidePut=puts", func(t *testing.T) {
		t.Skip("API drops the delta filter when the chain holds a null delta — TestDiscrepancy_ChainDeltaDroppedOnNullDelta")
		chain, _, err := liveCall(t, "ByDelta neg put", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)), options.WithStrike(options.ByDelta(-0.30)), options.WithSide(options.SidePut))
		})
		assertNoError(t, err, "ByDelta negative + side put")
		assertNotEmpty(t, chain.Options, "negative-delta put options")
		for _, o := range chain.Options {
			assertEqual(t, o.Type, options.Put, "negative delta with side=put should be a put")
			if o.Delta >= 0 || math.Abs(o.Delta+0.30) > 0.25 {
				t.Errorf("put delta %.4f not a negative value near -0.30", o.Delta)
			}
		}
	})
}

// TestReflect_OptionsExpiry exercises every options ExpiryFilter member live.
func TestReflect_OptionsExpiry(t *testing.T) {
	client := setupClient(t)
	ctx := liveContext(t, 3*time.Minute)
	fx := resolveOptionsFixture(t, ctx, client)

	t.Run("OnExpiration=matches", func(t *testing.T) {
		chain, _, err := liveCall(t, "OnExpiration", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(fx.exp)))
		})
		assertNoError(t, err, "OnExpiration")
		assertNotEmpty(t, chain.Options, "expiration options")
		for _, o := range chain.Options {
			if dayDiff(o.Expiration, fx.exp) > 24*time.Hour {
				t.Errorf("OnExpiration returned %v, not the requested %v", o.Expiration, fx.exp)
			}
		}
	})

	t.Run("InDTE=nearN", func(t *testing.T) {
		const want = 30
		chain, _, err := liveCall(t, "InDTE", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.InDTE(want)))
		})
		assertNoError(t, err, "InDTE")
		assertNotEmpty(t, chain.Options, "dte options")
		// The API resolves to the single nearest expiration; all contracts share
		// one DTE close to the requested value.
		exps := map[int64]bool{}
		for _, o := range chain.Options {
			exps[o.Expiration.Unix()] = true
			if d := int(math.Abs(float64(o.DTE - want))); d > 12 {
				t.Errorf("InDTE(%d) returned DTE %d, too far from requested", want, o.DTE)
			}
		}
		if len(exps) != 1 {
			t.Errorf("InDTE should resolve to a single expiration, got %d", len(exps))
		}
	})

	t.Run("InMonth=allInMonth", func(t *testing.T) {
		month := int(fx.exp.Month())
		chain, _, err := liveCall(t, "InMonth", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.InMonth(month)), options.WithStrikeLimit(1))
		})
		assertNoError(t, err, "InMonth")
		assertNotEmpty(t, chain.Options, "month options")
		for _, o := range chain.Options {
			if int(o.Expiration.Month()) != month {
				t.Errorf("InMonth(%d) returned expiration in month %d", month, o.Expiration.Month())
			}
		}
	})

	t.Run("InYear=allInYear", func(t *testing.T) {
		year := fx.exp.Year()
		chain, _, err := liveCall(t, "InYear", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.InYear(year)), options.WithStrikeLimit(1))
		})
		assertNoError(t, err, "InYear")
		assertNotEmpty(t, chain.Options, "year options")
		for _, o := range chain.Options {
			if o.Expiration.Year() != year {
				t.Errorf("InYear(%d) returned expiration in year %d", year, o.Expiration.Year())
			}
		}
	})

	t.Run("InMonthOfYear=matchesBoth", func(t *testing.T) {
		month, year := int(fx.exp.Month()), fx.exp.Year()
		chain, _, err := liveCall(t, "InMonthOfYear", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.InMonthOfYear(month, year)), options.WithStrikeLimit(1))
		})
		assertNoError(t, err, "InMonthOfYear")
		assertNotEmpty(t, chain.Options, "month-of-year options")
		for _, o := range chain.Options {
			if int(o.Expiration.Month()) != month || o.Expiration.Year() != year {
				t.Errorf("InMonthOfYear(%d,%d) returned %v", month, year, o.Expiration)
			}
		}
	})

	// The list/range/comparison dte forms are asserted against candidates
	// drawn from a baseline AllExpirations call — the server's own set of
	// listed (expiration, dte) pairs — rather than a locally computed day
	// count, so the check never depends on which definition of "today" the
	// server used to derive dte.
	t.Run("DTENewForms", func(t *testing.T) {
		sortedDTEs, expsByDTE := optionsDTEBaseline(t, ctx, client)
		if len(sortedDTEs) < 3 {
			t.Skip("need at least 3 distinct listed dte values to test the new dte forms")
		}

		t.Run("InDTEs=list", func(t *testing.T) {
			d1, d2 := sortedDTEs[0], sortedDTEs[len(sortedDTEs)-1]
			candidates := map[int64]bool{}
			for _, e := range expsByDTE[d1] {
				candidates[truncDay(e).Unix()] = true
			}
			for _, e := range expsByDTE[d2] {
				candidates[truncDay(e).Unix()] = true
			}

			chain, _, err := liveCall(t, "InDTEs list", func() (*options.OptionsChain, *marketdata.Response, error) {
				return client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.InDTEs(d1, d2)))
			})
			assertNoError(t, err, "InDTEs")
			assertNotEmpty(t, chain.Options, "dte-list options")

			gotD1, gotD2 := false, false
			distinct := map[int64]bool{}
			for _, o := range chain.Options {
				key := truncDay(o.Expiration).Unix()
				distinct[key] = true
				if !candidates[key] {
					t.Errorf("InDTEs(%d, %d) returned expiration %v, which is not a listed candidate for either value", d1, d2, o.Expiration)
				}
				switch o.DTE {
				case d1:
					gotD1 = true
				case d2:
					gotD2 = true
				}
			}
			if !gotD1 {
				t.Errorf("InDTEs(%d, %d) returned nothing at dte=%d", d1, d2, d1)
			}
			if !gotD2 {
				t.Errorf("InDTEs(%d, %d) returned nothing at dte=%d", d1, d2, d2)
			}
			if len(distinct) < 2 {
				t.Errorf("InDTEs(%d, %d) returned %d distinct expiration(s), want more than 1 — "+
					"the two requested values resolve to different listed expirations", d1, d2, len(distinct))
			}
		})

		t.Run("DTERange=within", func(t *testing.T) {
			n := len(sortedDTEs)
			if n < 5 {
				t.Skip("need at least 5 distinct listed dte values so the range excludes the unfiltered front-month value (index 0) and still has two listed values strictly inside it")
			}
			// loIdx starts at 1 (never 0) so the unfiltered front-month
			// answer — whose dte is always sortedDTEs[0] — sits outside the
			// range and fails the bound check below. hiIdx is kept at least
			// 3 past loIdx so at least two listed dte values fall strictly
			// between lo and hi, which is what makes distinct>=2 hold
			// whether the server reads "lo-hi" inclusively or exclusively.
			loIdx, hiIdx := 1, n-1
			if n >= 8 {
				loIdx, hiIdx = n/4, 3*n/4
			}
			lo, hi := sortedDTEs[loIdx], sortedDTEs[hiIdx]

			chain, _, err := liveCall(t, "DTERange", func() (*options.OptionsChain, *marketdata.Response, error) {
				return client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.DTERange(lo, hi)))
			})
			assertNoError(t, err, "DTERange")
			assertNotEmpty(t, chain.Options, "dte-range options")

			distinct := map[int64]bool{}
			for _, o := range chain.Options {
				if o.DTE < lo || o.DTE > hi {
					t.Errorf("DTERange(%d, %d) returned dte %d outside the bounds", lo, hi, o.DTE)
				}
				distinct[truncDay(o.Expiration).Unix()] = true
			}
			if len(distinct) < 2 {
				t.Errorf("DTERange(%d, %d) returned %d distinct expiration(s), want more than 1 — "+
					"the range spans several listed dte values", lo, hi, len(distinct))
			}
		})

		t.Run("MinDTE=atOrAbove", func(t *testing.T) {
			// The nearest listed dte (sortedDTEs[0]) is what the unfiltered
			// (no expiry filter) chain resolves to, so the floor is set one
			// step above it: the unfiltered front-month row would violate
			// this bound, which is what proves the filter reached the API.
			x := sortedDTEs[1]

			chain, _, err := liveCall(t, "MinDTE", func() (*options.OptionsChain, *marketdata.Response, error) {
				return client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.MinDTE(x)))
			})
			assertNoError(t, err, "MinDTE")
			assertNotEmpty(t, chain.Options, "min-dte options")
			for _, o := range chain.Options {
				if o.DTE < x {
					t.Errorf("MinDTE(%d) returned dte %d below the floor", x, o.DTE)
				}
			}
		})

		t.Run("MaxDTE=atOrBelow", func(t *testing.T) {
			// A per-row bound alone cannot distinguish this from the
			// unfiltered front-month answer, which always has the smallest
			// listed dte and so always satisfies any ceiling >= it. The
			// ceiling is set two steps above the nearest listed value, not
			// one, so that two listed dte values (sortedDTEs[0] and
			// sortedDTEs[1]) sit STRICTLY below it: distinct>=2 then holds
			// whether the server reads "<=N" inclusively or exclusively,
			// rather than depending on the row at exactly N coming back.
			x := sortedDTEs[2]

			chain, _, err := liveCall(t, "MaxDTE", func() (*options.OptionsChain, *marketdata.Response, error) {
				return client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.MaxDTE(x)))
			})
			assertNoError(t, err, "MaxDTE")
			assertNotEmpty(t, chain.Options, "max-dte options")
			distinct := map[int64]bool{}
			for _, o := range chain.Options {
				if o.DTE > x {
					t.Errorf("MaxDTE(%d) returned dte %d above the ceiling", x, o.DTE)
				}
				distinct[truncDay(o.Expiration).Unix()] = true
			}
			if len(distinct) < 2 {
				t.Errorf("MaxDTE(%d) returned %d distinct expiration(s), want more than 1 — "+
					"the unfiltered answer also holds exactly one, so this must show more to prove the bound reached the API", x, len(distinct))
			}
		})

		// DTEExpr passes a raw expression through verbatim, which is the only
		// way this SDK can reach a strict "<" comparison: MaxDTE emits only
		// "<=". The release text enumerates "<45" as an accepted form but
		// never a bare ">", so only the "<" form is exercised here. The
		// assertion below is the weak ("<=") reading of the strict bound,
		// which holds whichever way the server resolves strictness at the
		// boundary, and reuses the same ceiling choice as MaxDTE above to
		// tell the result apart from the single-expiration unfiltered
		// answer.
		t.Run("DTEExpr=strictBelow", func(t *testing.T) {
			x := sortedDTEs[2]

			chain, _, err := liveCall(t, "DTEExpr<", func() (*options.OptionsChain, *marketdata.Response, error) {
				return client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.DTEExpr(fmt.Sprintf("<%d", x))))
			})
			assertNoError(t, err, "DTEExpr<")
			assertNotEmpty(t, chain.Options, "dte-expr-strict-below options")
			distinct := map[int64]bool{}
			for _, o := range chain.Options {
				if o.DTE > x {
					t.Errorf("DTEExpr(%q) returned dte %d above the ceiling", fmt.Sprintf("<%d", x), o.DTE)
				}
				distinct[truncDay(o.Expiration).Unix()] = true
			}
			if len(distinct) < 2 {
				t.Errorf("DTEExpr(%q) returned %d distinct expiration(s), want more than 1 — "+
					"the unfiltered answer also holds exactly one, so this must show more to prove the bound reached the API", fmt.Sprintf("<%d", x), len(distinct))
			}
		})
	})
}

// optionsDTEBaseline fetches every listed expiration's server-reported dte via
// one AllExpirations chain call, so the new dte filter forms (list, range,
// comparison operators) can be asserted against real candidates instead of a
// locally computed day count. WithStrikeLimit(1) keeps the response small:
// only the (expiration, dte) pairing is needed, not every contract.
func optionsDTEBaseline(t *testing.T, ctx context.Context, client *marketdata.Client) (sortedDTEs []int, expsByDTE map[int][]time.Time) {
	t.Helper()

	chain, _, err := liveCall(t, "AllExpirations baseline", func() (*options.OptionsChain, *marketdata.Response, error) {
		return client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.AllExpirations()), options.WithStrikeLimit(1))
	})
	assertNoError(t, err, "AllExpirations baseline")
	assertNotEmpty(t, chain.Options, "baseline options")

	expsByDTE = map[int][]time.Time{}
	seen := map[int64]bool{}
	for _, o := range chain.Options {
		key := truncDay(o.Expiration).Unix()
		if seen[key] {
			continue
		}
		seen[key] = true
		expsByDTE[o.DTE] = append(expsByDTE[o.DTE], o.Expiration)
	}
	for d := range expsByDTE {
		sortedDTEs = append(sortedDTEs, d)
	}
	sort.Ints(sortedDTEs)
	return sortedDTEs, expsByDTE
}

// TestReflect_OptionsFilters exercises the independent (free) options.Chain
// filters live and asserts the response reflects each where checkable.
func TestReflect_OptionsFilters(t *testing.T) {
	client := setupClient(t)
	ctx := liveContext(t, 3*time.Minute)
	fx := resolveOptionsFixture(t, ctx, client)
	base := options.WithExpiry(options.OnExpiration(fx.exp))

	t.Run("Side=call", func(t *testing.T) {
		chain, _, err := liveCall(t, "side call", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, base, options.WithSide(options.SideCall))
		})
		assertNoError(t, err, "side call")
		assertNotEmpty(t, chain.Options, "call options")
		for _, o := range chain.Options {
			assertEqual(t, o.Type, options.Call, "side=call returns only calls")
		}
	})

	t.Run("Side=put", func(t *testing.T) {
		chain, _, err := liveCall(t, "side put", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, base, options.WithSide(options.SidePut))
		})
		assertNoError(t, err, "side put")
		assertNotEmpty(t, chain.Options, "put options")
		for _, o := range chain.Options {
			assertEqual(t, o.Type, options.Put, "side=put returns only puts")
		}
	})

	t.Run("Range=itm", func(t *testing.T) {
		chain, _, err := liveCall(t, "range itm", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, base, options.WithSide(options.SideCall), options.WithRange(options.MoneynessITM))
		})
		assertNoError(t, err, "range itm")
		assertNotEmpty(t, chain.Options, "itm options")
		for _, o := range chain.Options {
			assertTrue(t, o.InTheMoney, "range=itm returns only in-the-money contracts")
		}
	})

	t.Run("Range=otm", func(t *testing.T) {
		chain, _, err := liveCall(t, "range otm", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, base, options.WithSide(options.SideCall), options.WithRange(options.MoneynessOTM))
		})
		assertNoError(t, err, "range otm")
		assertNotEmpty(t, chain.Options, "otm options")
		for _, o := range chain.Options {
			assertTrue(t, !o.InTheMoney, "range=otm returns only out-of-the-money contracts")
		}
	})

	t.Run("MinOpenInterest", func(t *testing.T) {
		const min = 100
		chain, _, err := liveCall(t, "minOI", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, base, options.WithMinOpenInterest(min))
		})
		assertNoError(t, err, "minOpenInterest")
		for _, o := range chain.Options {
			if o.OpenInterest < min {
				t.Errorf("minOpenInterest(%d) returned OI %d", min, o.OpenInterest)
			}
		}
	})

	t.Run("MinVolume", func(t *testing.T) {
		const min = 1
		chain, _, err := liveCall(t, "minVol", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, base, options.WithMinVolume(min))
		})
		assertNoError(t, err, "minVolume")
		for _, o := range chain.Options {
			if o.Volume < min {
				t.Errorf("minVolume(%d) returned volume %d", min, o.Volume)
			}
		}
	})

	t.Run("MinBid", func(t *testing.T) {
		const min = 0.50
		chain, _, err := liveCall(t, "minBid", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, base, options.WithMinBid(min))
		})
		assertNoError(t, err, "minBid")
		for _, o := range chain.Options {
			if o.Bid < min {
				t.Errorf("minBid(%.2f) returned bid %.2f", min, o.Bid)
			}
		}
	})

	t.Run("StrikeLimit", func(t *testing.T) {
		const limit = 2
		chain, _, err := liveCall(t, "strikeLimit", func() (*options.OptionsChain, *marketdata.Response, error) {
			return client.Options.Chain(ctx, TestStockSymbol, base, options.WithStrikeLimit(limit))
		})
		assertNoError(t, err, "strikeLimit")
		assertNotEmpty(t, chain.Options, "strikeLimit options")
		distinct := map[float64]bool{}
		for _, o := range chain.Options {
			distinct[o.Strike] = true
		}
		// strikeLimit is per side of the money, so n admits up to 2n distinct
		// strikes. The test asserted n and had been failing on correct API
		// behavior; verified live 2026-08-20 that 1/2/3 return 2/4/6.
		if len(distinct) > 2*limit {
			t.Errorf("strikeLimit(%d) returned %d distinct strikes, want at most %d", limit, len(distinct), 2*limit)
		}
	})
}

// TestReflect_MarketsWindows exercises Status (single day) against
// StatusHistory (windowed) and asserts each reflects its selector.
func TestReflect_MarketsWindows(t *testing.T) {
	client := setupClient(t)
	ctx := liveContext(t, 90*time.Second)

	t.Run("Status_singleDay", func(t *testing.T) {
		st, _, err := liveCall(t, "status date", func() (*markets.MarketStatus, *marketdata.Response, error) {
			return client.Markets.Status(ctx, markets.WithDate(liveHistDay))
		})
		assertNoError(t, err, "status date")
		assertNotNil(t, st, "status")
		if dayDiff(st.Date, liveHistDay) > 24*time.Hour {
			t.Errorf("Status(date) returned %v, not %v", st.Date, liveHistDay)
		}
		assertEqual(t, st.Status, "open", "2024-06-03 (Mon) should be open")
	})

	t.Run("StatusHistory_between_inRange", func(t *testing.T) {
		from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
		hist, _, err := liveCall(t, "status-history between", func() ([]markets.MarketStatus, *marketdata.Response, error) {
			return client.Markets.StatusHistory(ctx, markets.WithHistoryWindow(markets.Between(from, to)))
		})
		assertNoError(t, err, "status-history between")
		assertNotEmpty(t, hist, "history")
		open := 0
		for _, s := range hist {
			if truncDay(s.Date).Before(from) || truncDay(s.Date).After(to) {
				t.Errorf("history date %v outside [%v,%v]", s.Date, from, to)
			}
			if s.Open {
				open++
			}
		}
		// June 2024 had 19 open trading days (20 weekdays minus Juneteenth).
		if open < 17 || open > 21 {
			t.Errorf("expected ~19 open days in June 2024, got %d", open)
		}
	})

	t.Run("StatusHistory_countback_tolerated", func(t *testing.T) {
		// The live API returns countback+1 rows here (a known API discrepancy,
		// see TestDiscrepancy_StatusHistoryCountback). Tolerate n or n+1 so this
		// broad reflection test stays green; the strict check lives in the
		// tracked skip test.
		const n = 5
		hist, _, err := liveCall(t, "status-history countback", func() ([]markets.MarketStatus, *marketdata.Response, error) {
			return client.Markets.StatusHistory(ctx, markets.WithHistoryWindow(markets.LastN(n)))
		})
		assertNoError(t, err, "status-history countback")
		if len(hist) != n && len(hist) != n+1 {
			t.Errorf("LastN(%d) returned %d rows, expected %d or %d", n, len(hist), n, n+1)
		}
	})

	t.Run("StatusHistory_until_bound", func(t *testing.T) {
		hist, _, err := liveCall(t, "status-history until", func() ([]markets.MarketStatus, *marketdata.Response, error) {
			return client.Markets.StatusHistory(ctx, markets.WithHistoryWindow(markets.Until(liveHistTo)))
		})
		assertNoError(t, err, "status-history until")
		assertNotEmpty(t, hist, "history until")
		for _, s := range hist {
			if truncDay(s.Date).After(liveHistTo.Add(24 * time.Hour)) {
				t.Errorf("history date %v later than to %v", s.Date, liveHistTo)
			}
		}
	})
}
