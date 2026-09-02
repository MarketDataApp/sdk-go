//go:build integration

package integration

import (
	"context"
	"math"
	"sort"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

// optionsFixture holds live, real option-market values resolved once and reused
// by the catalog acceptance and per-group reflection tests. Everything here is
// fetched from the live API so the probes use genuine, currently-listed
// contracts rather than guessed symbols.
type optionsFixture struct {
	exp        time.Time // a real future expiration with a dense strike ladder
	underlying float64   // the underlying's price at fetch time
	strikes    []float64 // sorted distinct strikes listed for exp
	atm        float64   // the listed strike nearest the underlying price
	occCall    string    // a real OCC symbol: the ATM call for exp
	occPut     string    // a real OCC symbol: the ATM put for exp
}

// resolveOptionsFixture fetches a live expiration and chain for AAPL and
// derives realistic probe values from them. It fails the test (rather than
// skipping) if the market data needed to drive the options probes is
// unavailable, since the whole point of the suite is live coverage.
func resolveOptionsFixture(t *testing.T, ctx context.Context, client *marketdata.Client) optionsFixture {
	t.Helper()

	expResult, _, err := liveCall(t, "options.Expirations(AAPL)", func() (*options.Expirations, *marketdata.Response, error) {
		return client.Options.Expirations(ctx, TestStockSymbol)
	})
	assertNoError(t, err, "resolveOptionsFixture: Expirations")
	assertNotNil(t, expResult, "resolveOptionsFixture: expirations")
	exps := expResult.Dates
	assertNotEmpty(t, exps, "resolveOptionsFixture: expirations")

	// Pick the next MONTHLY expiration (the third Friday) among the
	// strictly-future ones. This used to target the expiration nearest ~45
	// days out on the assumption that the horizon "reliably lands on a dense
	// monthly chain"; it does not — most weeks the 45-day mark sits closer
	// to a weekly, and a far-out weekly can be listed so recently that its
	// contracts have no EOD history yet, so a token whose plan reads
	// historical EOD data gets no_data for the whole chain and every probe
	// built on the fixture dies. Monthlies are listed far in advance and
	// always carry history. A monthly whose Friday is an exchange holiday
	// settles a day early and is skipped by the third-Friday test; the next
	// month's monthly is picked instead, which stays within the fixture's
	// needs.
	now := time.Now()
	var exp time.Time
	for _, e := range exps {
		if truncDay(e).Before(truncDay(now).Add(24 * time.Hour)) {
			continue // skip today / past
		}
		if e.Weekday() == time.Friday && e.Day() >= 15 && e.Day() <= 21 {
			if exp.IsZero() || e.Before(exp) {
				exp = e
			}
		}
	}
	if exp.IsZero() {
		exp = exps[len(exps)-1]
	}

	chain, _, err := liveCall(t, "options.Chain(AAPL, exp)", func() (*options.OptionsChain, *marketdata.Response, error) {
		return client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(exp)))
	})
	assertNoError(t, err, "resolveOptionsFixture: Chain")
	assertNotNil(t, chain, "resolveOptionsFixture: chain")
	assertNotEmpty(t, chain.Options, "resolveOptionsFixture: chain options")

	fx := optionsFixture{exp: exp}
	seen := map[float64]bool{}
	for _, o := range chain.Options {
		if o.UnderlyingPrice > 0 {
			fx.underlying = o.UnderlyingPrice
		}
		if !seen[o.Strike] {
			seen[o.Strike] = true
			fx.strikes = append(fx.strikes, o.Strike)
		}
	}
	sort.Float64s(fx.strikes)
	if fx.underlying == 0 {
		// Fall back to a live quote if the chain omitted the underlying price.
		q, _, qerr := liveCall(t, "stocks.Quote(AAPL)", func() (*stocks.Quote, *marketdata.Response, error) {
			return client.Stocks.Quote(ctx, TestStockSymbol)
		})
		assertNoError(t, qerr, "resolveOptionsFixture: fallback Quote")
		fx.underlying = q.Last
	}

	// ATM strike = the listed strike nearest the underlying price.
	fx.atm = fx.strikes[0]
	bestGap := math.MaxFloat64
	for _, s := range fx.strikes {
		if g := math.Abs(s - fx.underlying); g < bestGap {
			bestGap = g
			fx.atm = s
		}
	}
	for _, o := range chain.Options {
		if o.Strike == fx.atm {
			if o.Type == options.Call && fx.occCall == "" {
				fx.occCall = o.OptionSymbol
			}
			if o.Type == options.Put && fx.occPut == "" {
				fx.occPut = o.OptionSymbol
			}
		}
	}
	if fx.occCall == "" {
		fx.occCall = chain.Options[0].OptionSymbol
	}
	return fx
}
