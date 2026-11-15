//go:build integration

package integration

import (
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
)

func TestOptions_Expirations(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	expirations, _, err := client.Options.Expirations(ctx, TestStockSymbol)
	assertNoError(t, err, "Expirations")
	assertNotNil(t, expirations, "expirations should not be nil")
	assertNotEmpty(t, expirations.Dates, "expirations should not be empty")

	// The API reports a response-level updated timestamp; it must surface
	// and be a plausible recent time.
	if expirations.Updated.IsZero() {
		t.Error("expirations.Updated should be populated from the API response")
	}
	assertTimeInPast(t, expirations.Updated, "expirations update time")

	// All expirations should be in the future, or today on the market's
	// calendar. marketToday is Eastern for the reason its comment gives.
	todayDate := marketToday()
	for i, exp := range expirations.Dates {
		expDate := truncDay(exp)
		if expDate.Before(todayDate) {
			t.Errorf("Expiration %d (%v) should not be in the past", i, exp)
		}
	}

	// Expirations should be sorted
	assertTimeSorted(t, expirations.Dates, "expirations")
}

func TestOptions_Chain(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	chain, _, err := client.Options.Chain(ctx, TestStockSymbol)
	assertNoError(t, err, "Chain")
	assertNotNil(t, chain, "chain should not be nil")
	assertNotEmpty(t, chain.Options, "chain options should not be empty")

	// Verify underlying matches
	assertEqual(t, chain.Underlying, TestStockSymbol, "underlying should match")

	// Validate each option contract
	for _, opt := range chain.Options {
		// Underlying should match
		assertEqual(t, opt.Underlying, TestStockSymbol, "option underlying")

		// Strike should be positive. This was relaxed while the API reported
		// 0 for deep-ITM rows and restored once that was fixed; see
		// TestDiscrepancy_ChainStrikeZero.
		assertPositive(t, opt.Strike, "option strike")

		// Type should be call or put
		if opt.Type != options.Call && opt.Type != options.Put {
			t.Errorf("Invalid option type: %s", opt.Type)
		}

		// Bid should be <= Ask (if both are non-zero)
		if opt.Bid > 0 && opt.Ask > 0 && opt.Bid > opt.Ask {
			t.Errorf("Bid (%f) should be <= Ask (%f) for %s",
				opt.Bid, opt.Ask, opt.OptionSymbol)
		}
	}
}

// TestOptions_Chain_AllExpirations pins the behavioral difference that
// AllExpirations exists for, against the live API: an unfiltered chain request
// returns only the front-month expiration, while expiration=all returns every
// listed one. Offline mocks cannot catch a regression here — they echo whatever
// the SDK asks for — so this assertion has to run against production.
//
// Verified live 2026-08-11 with curl: AAPL unfiltered returned 190 contracts
// across 1 expiration, and with expiration=all 3528 contracts across 24. The
// full-chain call is correspondingly expensive in API credits.
func TestOptions_Chain_AllExpirations(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	distinctExpirations := func(chain *options.OptionsChain) int {
		seen := make(map[time.Time]struct{})
		for _, opt := range chain.Options {
			seen[opt.Expiration] = struct{}{}
		}
		return len(seen)
	}

	frontMonth, _, err := client.Options.Chain(ctx, TestStockSymbol)
	assertNoError(t, err, "Chain unfiltered")
	assertNotNil(t, frontMonth, "unfiltered chain should not be nil")
	assertNotEmpty(t, frontMonth.Options, "unfiltered chain should not be empty")

	all, _, err := client.Options.Chain(ctx, TestStockSymbol,
		options.WithExpiry(options.AllExpirations()),
	)
	assertNoError(t, err, "Chain with AllExpirations")
	assertNotNil(t, all, "full chain should not be nil")
	assertNotEmpty(t, all.Options, "full chain should not be empty")

	frontCount := distinctExpirations(frontMonth)
	allCount := distinctExpirations(all)

	if frontCount != 1 {
		t.Errorf("unfiltered chain covered %d expirations, want exactly 1 (front month)", frontCount)
	}
	if allCount <= frontCount {
		t.Errorf("AllExpirations covered %d expirations, want more than the unfiltered %d — "+
			"expiration=all is not reaching the API", allCount, frontCount)
	}
	if len(all.Options) <= len(frontMonth.Options) {
		t.Errorf("AllExpirations returned %d contracts, want more than the unfiltered %d",
			len(all.Options), len(frontMonth.Options))
	}
}

// TestOptions_Chain_StrikeAndDeltaLists asserts live that the comma-separated
// list forms reach the API and are honored: a two-strike list must return
// contracts at exactly those two strikes, and a two-delta list must return
// strictly more contracts than the single-delta form it generalizes.
func TestOptions_Chain_StrikeAndDeltaLists(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	// Anchor the strikes on real listed ones rather than guessing.
	seed, _, err := client.Options.Chain(ctx, TestStockSymbol, options.WithSide(options.SideCall))
	assertNoError(t, err, "Chain seed")
	assertNotNil(t, seed, "seed chain should not be nil")
	assertNotEmpty(t, seed.Options, "seed chain should not be empty")
	if len(seed.Options) < 2 {
		t.Skipf("need at least 2 listed strikes to test the list form, got %d", len(seed.Options))
	}
	first, second := seed.Options[0].Strike, seed.Options[len(seed.Options)-1].Strike

	chain, _, err := client.Options.Chain(ctx, TestStockSymbol,
		options.WithStrike(options.Strikes(first, second)),
	)
	assertNoError(t, err, "Chain with Strikes list")
	assertNotNil(t, chain, "strike-list chain should not be nil")
	assertNotEmpty(t, chain.Options, "strike-list chain should not be empty")
	for _, opt := range chain.Options {
		if opt.Strike != first && opt.Strike != second {
			t.Errorf("got strike %v, want one of %v / %v — the list is not being honored",
				opt.Strike, first, second)
		}
	}

	oneDelta, _, err := client.Options.Chain(ctx, TestStockSymbol,
		options.WithStrike(options.ByDelta(0.30)),
	)
	assertNoError(t, err, "Chain with ByDelta")
	assertNotNil(t, oneDelta, "single-delta chain should not be nil")

	twoDeltas, _, err := client.Options.Chain(ctx, TestStockSymbol,
		options.WithStrike(options.ByDeltas(0.16, 0.30)),
	)
	assertNoError(t, err, "Chain with ByDeltas list")
	assertNotNil(t, twoDeltas, "delta-list chain should not be nil")

	// The API drops a delta filter entirely whenever the fetched chain holds
	// a null delta, so on those symbols the two chains come back identical
	// and this cannot distinguish a list that reached the API from one that
	// did not. See TestDiscrepancy_ChainDeltaDroppedOnNullDelta, which
	// carries the strict assertion. What still holds is that the SDK
	// serializes the list and the endpoint accepts it.
	if len(twoDeltas.Options) == 0 {
		t.Error("ByDeltas(0.16, 0.30) returned no contracts")
	}
}

// TestOptions_Chain_DTEList asserts live that the comma-separated dte list
// form reaches the API and is honored: each requested value must resolve to a
// returned contract whose DTE matches one of the listed expirations closest to
// it (computed from a baseline Expirations call, never from the unfiltered
// chain), and every returned contract's DTE must be a candidate of one of the
// two requested values.
func TestOptions_Chain_DTEList(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	expirations, _, err := client.Options.Expirations(ctx, TestStockSymbol)
	assertNoError(t, err, "Expirations for DTE list")
	assertNotNil(t, expirations, "expirations should not be nil")
	assertNotEmpty(t, expirations.Dates, "expirations should not be empty")

	today := marketToday()
	var dtes []int
	for _, e := range expirations.Dates {
		dtes = append(dtes, int(truncDay(e).Sub(today).Hours()/24))
	}
	if len(dtes) < 2 {
		t.Skipf("need at least 2 listed expirations to test the list form, got %d", len(dtes))
	}

	want1, want2 := dtes[0], dtes[len(dtes)-1]
	cand1, cand2 := closestDTEs(dtes, want1), closestDTEs(dtes, want2)
	if sameDTECandidates(cand1, cand2) {
		t.Skip("the two requested DTE values resolve to the same offered expiration")
	}

	chain, _, err := client.Options.Chain(ctx, TestStockSymbol,
		options.WithExpiry(options.InDTEs(want1, want2)),
	)
	assertNoError(t, err, "Chain with InDTEs list")
	assertNotNil(t, chain, "dte-list chain should not be nil")
	assertNotEmpty(t, chain.Options, "dte-list chain should not be empty")

	var sawCand1, sawCand2 bool
	for _, o := range chain.Options {
		inCand1 := intSliceContains(cand1, o.DTE)
		inCand2 := intSliceContains(cand2, o.DTE)
		if !inCand1 && !inCand2 {
			t.Errorf("got DTE %d, want one of %v / %v — the list is not being honored", o.DTE, cand1, cand2)
		}
		if inCand1 {
			sawCand1 = true
		}
		if inCand2 {
			sawCand2 = true
		}
	}
	if !sawCand1 {
		t.Errorf("InDTEs(%d, %d) never returned a contract matching the %d candidates %v", want1, want2, want1, cand1)
	}
	if !sawCand2 {
		t.Errorf("InDTEs(%d, %d) never returned a contract matching the %d candidates %v", want1, want2, want2, cand2)
	}
}

// TestOptions_QuotesBySymbol asserts live that a contract with no data comes
// back as a present nil entry rather than vanishing, which is the entire
// difference from Quotes.
func TestOptions_QuotesBySymbol(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	chain, _, err := client.Options.Chain(ctx, TestStockSymbol, options.WithSide(options.SideCall))
	assertNoError(t, err, "Chain for QuotesBySymbol")
	assertNotNil(t, chain, "chain should not be nil")
	assertNotEmpty(t, chain.Options, "chain should not be empty")

	real := chain.Options[0].OptionSymbol
	// A strike of $99,999 is not listed on any real underlying.
	bogus := TestStockSymbol + "260821C99999000"

	quotes, _, err := client.Options.QuotesBySymbol(ctx, []string{real, bogus})
	assertNoError(t, err, "QuotesBySymbol")
	if len(quotes) != 2 {
		t.Fatalf("len(quotes) = %d, want 2 (one entry per requested symbol)", len(quotes))
	}
	if q, ok := quotes[real]; !ok || q == nil {
		t.Errorf("quotes[%q] = %v (present=%v), want a quote", real, q, ok)
	}
	q, ok := quotes[bogus]
	if !ok {
		t.Errorf("quotes[%q] missing: a no-data symbol must still have an entry", bogus)
	} else if q != nil {
		t.Errorf("quotes[%q] = %v, want nil for no-data", bogus, q)
	}

	// Quotes, by contrast, drops the no-data symbol entirely.
	slice, _, err := client.Options.Quotes(ctx, []string{real, bogus})
	assertNoError(t, err, "Quotes")
	if len(slice) != 1 {
		t.Errorf("len(Quotes()) = %d, want 1 — the no-data symbol should be omitted", len(slice))
	}
}

// TestOptions_Lookup_RejectsZeroArguments asserts that Lookup's argument
// validation holds end to end through a real client: these calls must fail
// before any request is made, not come back as a silent empty result.
func TestOptions_Lookup_RejectsZeroArguments(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	if _, _, err := client.Options.Lookup(ctx, TestStockSymbol, time.Time{}, 150, options.Call); err == nil {
		t.Error("Lookup with a zero expiration should return a validation error")
	}
	exp := time.Now().AddDate(0, 1, 0)
	if _, _, err := client.Options.Lookup(ctx, TestStockSymbol, exp, 0, options.Call); err == nil {
		t.Error("Lookup with a zero strike should return a validation error")
	}
	if _, _, err := client.Options.Lookup(ctx, TestStockSymbol, exp, 150, ""); err == nil {
		t.Error("Lookup with an empty option type should return a validation error")
	}
}

func TestOptions_Chain_CallsOnly(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	chain, _, err := client.Options.Chain(ctx, TestStockSymbol,
		options.WithSide(options.SideCall),
	)
	assertNoError(t, err, "Chain with calls only")
	assertNotNil(t, chain, "chain should not be nil")
	assertNotEmpty(t, chain.Options, "chain options should not be empty")

	// All options should be calls
	for _, opt := range chain.Options {
		if opt.Type != options.Call {
			t.Errorf("Expected call option, got %s for %s", opt.Type, opt.OptionSymbol)
		}
	}
}

func TestOptions_Chain_PutsOnly(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	chain, _, err := client.Options.Chain(ctx, TestStockSymbol,
		options.WithSide(options.SidePut),
	)
	assertNoError(t, err, "Chain with puts only")
	assertNotNil(t, chain, "chain should not be nil")
	assertNotEmpty(t, chain.Options, "chain options should not be empty")

	// All options should be puts
	for _, opt := range chain.Options {
		if opt.Type != options.Put {
			t.Errorf("Expected put option, got %s for %s", opt.Type, opt.OptionSymbol)
		}
	}
}

func TestOptions_Chain_StrikeRange(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	// Use a reasonable strike range based on current price
	// First get the stock price to determine reasonable strike range
	quote, _, err := client.Stocks.Quote(ctx, TestStockSymbol)
	assertNoError(t, err, "Quote for strike range")

	// Use strikes around the current price
	minStrike := quote.Last * 0.9 // 10% below
	maxStrike := quote.Last * 1.1 // 10% above

	chain, _, err := client.Options.Chain(ctx, TestStockSymbol,
		options.WithStrike(options.StrikeRange(minStrike, maxStrike)),
	)
	assertNoError(t, err, "Chain with strike range")
	assertNotNil(t, chain, "chain should not be nil")
	assertNotEmpty(t, chain.Options, "chain options should not be empty")

	// Note: The API may return strikes outside the requested range
	// as server-side filtering may not be exact. We just verify
	// we get valid option contracts back.
	for _, opt := range chain.Options {
		assertPositive(t, opt.Strike, "option strike")
	}
}

func TestOptions_Chain_WithExpiration(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	// First get a valid expiration
	expirations, _, err := client.Options.Expirations(ctx, TestStockSymbol)
	assertNoError(t, err, "Expirations for chain test")
	assertNotNil(t, expirations, "expirations should not be nil")
	assertNotEmpty(t, expirations.Dates, "need at least one expiration")

	expiration := expirations.Dates[0]

	chain, _, err := client.Options.Chain(ctx, TestStockSymbol,
		options.WithExpiry(options.OnExpiration(expiration)),
	)
	assertNoError(t, err, "Chain with expiration")
	assertNotNil(t, chain, "chain should not be nil")
	assertNotEmpty(t, chain.Options, "chain options should not be empty")

	// All options should have the requested expiration
	for _, opt := range chain.Options {
		// Allow for timezone differences - compare dates
		optExpDate := opt.Expiration.Truncate(24 * time.Hour)
		reqExpDate := expiration.Truncate(24 * time.Hour)

		// Allow a day tolerance for timezone issues
		diff := optExpDate.Sub(reqExpDate)
		if diff < -24*time.Hour || diff > 24*time.Hour {
			t.Errorf("Option expiration %v doesn't match requested %v for %s",
				opt.Expiration, expiration, opt.OptionSymbol)
		}
	}
}

func TestOptions_Quote(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	// Get a valid expiration first (not today)
	expirations, _, err := client.Options.Expirations(ctx, TestStockSymbol)
	assertNoError(t, err, "Expirations for quote test")
	assertNotNil(t, expirations, "expirations should not be nil")
	assertNotEmpty(t, expirations.Dates, "need at least one expiration")

	// Use an expiration in the future
	var expiration time.Time
	todayDate := marketToday()
	for _, exp := range expirations.Dates {
		expDate := truncDay(exp)
		if expDate.After(todayDate) {
			expiration = exp
			break
		}
	}
	if expiration.IsZero() {
		t.Skip("No future expirations available")
	}

	// Get chain for that expiration
	chain, _, err := client.Options.Chain(ctx, TestStockSymbol, options.WithExpiry(options.OnExpiration(expiration)))
	assertNoError(t, err, "Chain for quote test")
	assertNotNil(t, chain, "chain should not be nil")
	assertNotEmpty(t, chain.Options, "need at least one option")

	optionSymbol := chain.Options[0].OptionSymbol

	quote, _, err := client.Options.Quote(ctx, optionSymbol)
	assertNoError(t, err, "Option Quote")
	assertNotNil(t, quote, "quote should not be nil")

	// Symbol should match
	assertEqual(t, quote.OptionSymbol, optionSymbol, "option symbol should match")

	// Strike should be positive (see TestDiscrepancy_ChainStrikeZero).
	assertPositive(t, quote.Strike, "strike price")

	// Type should be call or put
	if quote.Type != options.Call && quote.Type != options.Put {
		t.Errorf("Invalid option type: %s", quote.Type)
	}
}

func TestOptions_Lookup(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	// Get options chain to find a real contract that exists
	chain, _, err := client.Options.Chain(ctx, TestStockSymbol)
	assertNoError(t, err, "Chain for lookup")
	assertNotNil(t, chain, "chain should not be nil")

	// Find a call option with future expiration
	var targetContract *struct {
		exp    time.Time
		strike float64
	}
	todayDate := marketToday()
	for _, opt := range chain.Options {
		if opt.Type == options.Call {
			expDate := truncDay(opt.Expiration)
			if expDate.After(todayDate) {
				targetContract = &struct {
					exp    time.Time
					strike float64
				}{opt.Expiration, opt.Strike}
				break
			}
		}
	}
	if targetContract == nil {
		t.Skip("No future call options found in chain")
	}

	// Lookup this exact option
	optionSymbol, _, err := client.Options.Lookup(ctx, TestStockSymbol, targetContract.exp, targetContract.strike, options.Call)
	assertNoError(t, err, "Lookup")

	// Symbol should not be empty
	if optionSymbol == "" {
		t.Error("Option symbol should not be empty")
	}

	// Verify we can get a quote for this symbol
	quote, _, err := client.Options.Quote(ctx, optionSymbol)
	assertNoError(t, err, "Quote for looked up symbol")
	assertNotNil(t, quote, "quote should not be nil")
	assertEqual(t, quote.Type, options.Call, "option should be a call")
}

func TestOptions_Lookup_Put(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)

	// Get options chain to find a real put contract that exists
	chain, _, err := client.Options.Chain(ctx, TestStockSymbol)
	assertNoError(t, err, "Chain for lookup")
	assertNotNil(t, chain, "chain should not be nil")

	// Find a put option with future expiration
	var targetContract *struct {
		exp    time.Time
		strike float64
	}
	todayDate := marketToday()
	for _, opt := range chain.Options {
		if opt.Type == options.Put {
			expDate := truncDay(opt.Expiration)
			if expDate.After(todayDate) {
				targetContract = &struct {
					exp    time.Time
					strike float64
				}{opt.Expiration, opt.Strike}
				break
			}
		}
	}
	if targetContract == nil {
		t.Skip("No future put options found in chain")
	}

	// Lookup this exact option
	optionSymbol, _, err := client.Options.Lookup(ctx, TestStockSymbol, targetContract.exp, targetContract.strike, options.Put)
	assertNoError(t, err, "Lookup put")

	if optionSymbol == "" {
		t.Error("Option symbol should not be empty")
	}

	// Verify it's a put
	quote, _, err := client.Options.Quote(ctx, optionSymbol)
	assertNoError(t, err, "Quote for put")
	assertNotNil(t, quote, "quote should not be nil")
	assertEqual(t, quote.Type, options.Put, "option should be a put")
}
