package apicatalog_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/apicatalog"
	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/funds"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/markets"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

// capture is an http.RoundTripper that accumulates the URL of every request
// the SDK makes and returns a benign "ok" response. It accumulates (rather than
// keeping only the last URL) so that the SDK's non-blocking background status
// refresh — which fires requests through the same transport — cannot race with
// the probes and drop a captured request.
type capture struct {
	mu   sync.Mutex
	urls []*url.URL
}

func (c *capture) RoundTrip(req *http.Request) (*http.Response, error) {
	u := *req.URL
	c.mu.Lock()
	c.urls = append(c.urls, &u)
	c.mu.Unlock()
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"s":"ok"}`)),
		Request:    req,
	}, nil
}

func (c *capture) snapshot() []*url.URL {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*url.URL, len(c.urls))
	copy(out, c.urls)
	return out
}

// endpointMatches reports whether a captured request path belongs to a
// catalog endpoint. markets/status and markets/status-history share a path;
// both are checked against the shared bucket, which is fine because the probes
// drive each parameter through the correct method.
func endpointMatches(path, endpoint string) bool {
	switch endpoint {
	case "stocks/quotes":
		return strings.Contains(path, "/stocks/bulkquotes/")
	case "stocks/candles":
		return strings.Contains(path, "/stocks/candles/")
	case "stocks/bulkcandles":
		return strings.Contains(path, "/stocks/bulkcandles/")
	case "stocks/prices":
		return strings.Contains(path, "/stocks/prices/")
	case "stocks/earnings":
		return strings.Contains(path, "/stocks/earnings/")
	case "stocks/news":
		return strings.Contains(path, "/stocks/news/")
	case "funds/candles":
		return strings.Contains(path, "/funds/candles/")
	case "options/chain":
		return strings.Contains(path, "/options/chain/")
	case "options/quotes":
		return strings.Contains(path, "/options/quotes/")
	case "options/expirations":
		return strings.Contains(path, "/options/expirations/")
	case "options/lookup":
		return strings.Contains(path, "/options/lookup/")
	case "markets/status", "markets/status-history":
		return strings.Contains(path, "/markets/status/")
	case "utilities/status":
		return path == "/status/"
	case "utilities/headers":
		return path == "/headers/"
	case "utilities/user":
		return path == "/user/"
	case "universal":
		return true // client-default params ride on every request
	}
	return false
}

// TestCatalogReachability drives the real public SDK surface for every endpoint
// and mode, then asserts each catalogued Query parameter is actually emitted on
// the wire and each Path parameter's endpoint is actually reached. A catalog
// entry with no emitting SDK path fails the test — the machine-checked proof
// that nothing the API supports is unreachable through the SDK.
func TestCatalogReachability(t *testing.T) {
	cap := &capture{}
	transport := &http.Client{Transport: cap}
	client, err := marketdata.NewClient(
		marketdata.WithToken("test-token"),
		marketdata.WithBaseURL("https://api.example.test"),
		marketdata.WithHTTPClient(transport),
		marketdata.WithoutStartupValidation(),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ctx := context.Background()
	d := time.Date(2024, 6, 3, 0, 0, 0, 0, time.UTC)
	a := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	b := time.Date(2024, 6, 5, 0, 0, 0, 0, time.UTC)

	// --- stocks ---
	_, _, _ = client.Stocks.Quote(ctx, "AAPL", stocks.WithFiftyTwoWeek(true), stocks.WithExtended(true), stocks.WithCandle(true))
	_, _, _ = client.Stocks.Quotes(ctx, []string{"AAPL", "MSFT"}, stocks.WithQuotesExtended(true), stocks.WithQuotesCandle(true))
	_, _, _ = client.Stocks.Candles(ctx, "AAPL", stocks.WithResolution(stocks.ResolutionDaily),
		stocks.WithCandleWindow(stocks.OnDate(d)), stocks.WithCandleExtended(true),
		stocks.WithCandleAdjustSplits(true), stocks.WithCandleAdjustDividends(false))
	_, _, _ = client.Stocks.Candles(ctx, "AAPL", stocks.WithResolution(stocks.ResolutionDaily), stocks.WithCandleWindow(stocks.Between(a, b)))
	_, _, _ = client.Stocks.Candles(ctx, "AAPL", stocks.WithCandleWindow(stocks.LastN(5)))
	_, _, _ = client.Stocks.BulkCandles(ctx, []string{"AAPL"}, stocks.WithBulkDate(d), stocks.WithAdjustSplits(true), stocks.WithAdjustDividends(false), stocks.WithSnapshot(true))
	_, _, _ = client.Stocks.BulkCandles(ctx, nil, stocks.WithSnapshot(true))
	_, _, _ = client.Stocks.Prices(ctx, []string{"AAPL", "MSFT"}, stocks.WithPriceExtended(true))
	for _, w := range []stocks.DateWindow{stocks.OnDate(d), stocks.Between(a, b), stocks.LastN(4)} {
		_, _, _ = client.Stocks.Earnings(ctx, "AAPL", stocks.WithEarningsWindow(w), stocks.WithEarningsReport("2024-Q1"))
		_, _, _ = client.Stocks.News(ctx, "AAPL", stocks.WithNewsWindow(w))
	}

	// --- funds ---
	for _, w := range []funds.DateWindow{funds.OnDate(d), funds.Between(a, b), funds.LastN(5)} {
		_, _, _ = client.Funds.Candles(ctx, "VFINX", funds.WithResolution(funds.ResolutionDaily), funds.WithCandleWindow(w))
	}

	// --- options ---
	_, _, _ = client.Options.Chain(ctx, "AAPL",
		options.WithExpiry(options.OnExpiration(d)), options.WithStrike(options.StrikeRange(1, 2)), options.WithChainDate(d),
		options.WithSide(options.SideCall), options.WithStrikeLimit(1), options.WithRange(options.MoneynessITM),
		options.WithMinBid(1), options.WithMaxBid(2), options.WithMinAsk(1), options.WithMaxAsk(2),
		options.WithMaxBidAskSpread(1), options.WithMaxBidAskSpreadPct(1), options.WithMinOpenInterest(1), options.WithMinVolume(1),
		options.WithExpirationTypes(options.IncludeExpirationTypes(options.Weekly, options.Monthly, options.Quarterly)),
		options.WithNonstandard(true), options.WithAM(true), options.WithPM(true))
	_, _, _ = client.Options.Chain(ctx, "AAPL", options.WithExpiry(options.InDTE(30)))
	_, _, _ = client.Options.Chain(ctx, "AAPL", options.WithExpiry(options.InMonthOfYear(12, 2026)))
	_, _, _ = client.Options.Chain(ctx, "AAPL", options.WithStrike(options.ByDelta(0.3)))
	_, _, _ = client.Options.Chain(ctx, "AAPL", options.WithExpiry(options.ExpirationBetween(a, b)))
	_, _, _ = client.Options.Chain(ctx, "AAPL", options.WithExpiry(options.AllExpirations()))
	_, _, _ = client.Options.Chain(ctx, "AAPL", options.WithStrike(options.Strikes(150, 160)))
	_, _, _ = client.Options.Chain(ctx, "AAPL", options.WithStrike(options.ByDeltas(0.16, 0.3)))
	_, _, _ = client.Options.Quote(ctx, "AAPL260717C00150000", options.WithOptionQuoteWindow(options.QuoteOnDate(d)))
	_, _, _ = client.Options.Quote(ctx, "AAPL260717C00150000", options.WithOptionQuoteWindow(options.QuoteRange(a, b)))
	_, _, _ = client.Options.Quote(ctx, "AAPL260717C00150000", options.WithOptionQuoteWindow(options.QuoteLastN(5)))
	_, _, _ = client.Options.Quote(ctx, "AAPL260717C00150000", options.WithOptionQuoteWindow(options.QuoteLastNUntil(5, b)))
	_, _, _ = client.Options.Expirations(ctx, "AAPL", options.WithExpirationStrike(150), options.WithExpirationDate(d))
	_, _, _ = client.Options.Lookup(ctx, "AAPL", d, 150, options.Call)

	// --- markets ---
	_, _, _ = client.Markets.Status(ctx, markets.WithDate(d), markets.WithCountry("US"))
	_, _, _ = client.Markets.StatusHistory(ctx, markets.WithHistoryWindow(markets.Between(a, b)), markets.WithCountry("US"))
	_, _, _ = client.Markets.StatusHistory(ctx, markets.WithHistoryWindow(markets.LastN(5)))

	// --- utilities (no params) ---
	_, _, _ = client.Utilities.Status(ctx)
	_, _, _ = client.Utilities.Headers(ctx)
	_, _, _ = client.Utilities.User(ctx)

	// --- universal (client-level defaults on every request) ---
	uniClient, err := marketdata.NewClient(
		marketdata.WithToken("test-token"),
		marketdata.WithBaseURL("https://api.example.test"),
		marketdata.WithHTTPClient(transport),
		marketdata.WithoutStartupValidation(),
		marketdata.WithColumns("last", "bid"),
		marketdata.WithMode(marketdata.ModeCached),
		marketdata.WithMaxAge("5min"),
		marketdata.WithLimit(100),
		marketdata.WithOffset(20),
		marketdata.WithDateFormat("unix"),
		marketdata.WithHumanReadable(true),
		marketdata.WithAddHeaders(true),
	)
	if err != nil {
		t.Fatalf("NewClient(universal): %v", err)
	}
	_, _, _ = uniClient.Stocks.Quotes(ctx, []string{"AAPL"})

	urls := cap.snapshot()
	emitted := func(endpoint, key string) bool {
		for _, u := range urls {
			if endpointMatches(u.Path, endpoint) && u.Query().Has(key) {
				return true
			}
		}
		return false
	}
	reached := func(endpoint string) bool {
		for _, u := range urls {
			if endpointMatches(u.Path, endpoint) {
				return true
			}
		}
		return false
	}

	var residuals []string
	for _, p := range apicatalog.All() {
		switch p.Kind {
		case apicatalog.Query:
			if !emitted(p.Endpoint, p.Name) {
				t.Errorf("UNREACHABLE query param %q on %s (SDK path: %s) — no SDK call emitted it",
					p.Name, p.Endpoint, p.SDKPath)
			}
		case apicatalog.Path:
			if !reached(p.Endpoint) {
				t.Errorf("UNREACHABLE: endpoint %s (path param %q) was never reached", p.Endpoint, p.Name)
			}
		case apicatalog.Residual:
			if !emitted(p.Endpoint, p.Name) {
				t.Logf("residual %s.%s not emitted in this run (reachable via option; SDK-owned)", p.Endpoint, p.Name)
			}
			residuals = append(residuals, p.Endpoint+"."+p.Name)
		}
	}
	t.Logf("documented residuals (reachable but SDK-owned/advanced): %s", strings.Join(residuals, ", "))

	assertNoUncataloguedWireParams(t, urls)
}

// assertNoUncataloguedWireParams is the reverse of the check above: every
// query key the SDK actually put on the wire must be catalogued for that
// endpoint, or be universal.
//
// Without it the audit only ever reads the catalog, so its view of "what the
// SDK sends" is a hand-written document rather than the running system —
// the same substitution that let an allowlist describe a parameter as
// unexposed for a week after the SDK started sending it. Proven blind by
// experiment: adding a stray p.Set("adjust", "splits") to the candles
// serializer left the entire suite green, including the OpenAPI audit whose
// stated contract is "every parameter the SDK sends must be one the API
// declares".
//
// It also covers funds/candles, which is exempt from the OpenAPI comparison
// because the published schema omits the funds app, and therefore had no
// second witness at all.
func assertNoUncataloguedWireParams(t *testing.T, urls []*url.URL) {
	t.Helper()

	universal := map[string]bool{}
	byEndpoint := map[string]map[string]bool{}
	for _, p := range apicatalog.All() {
		if p.Endpoint == "universal" {
			universal[p.Name] = true
			continue
		}
		if byEndpoint[p.Endpoint] == nil {
			byEndpoint[p.Endpoint] = map[string]bool{}
		}
		byEndpoint[p.Endpoint][p.Name] = true
	}

	endpoints := make([]string, 0, len(byEndpoint))
	for endpoint := range byEndpoint {
		endpoints = append(endpoints, endpoint)
	}
	sort.Strings(endpoints)

	// Reported once per (path, key) so a parameter sent on every call does
	// not bury the output.
	seen := map[string]bool{}
	for _, u := range urls {
		// A wire path can serve more than one catalog endpoint —
		// markets/status and markets/status-history share /v1/markets/status/
		// — and the URL alone cannot say which method issued it. Check
		// against the union, exactly as the OpenAPI comparison does for the
		// same pair, so a parameter catalogued on either half counts.
		var matched []string
		allowed := map[string]bool{}
		for _, endpoint := range endpoints {
			if !endpointMatches(u.Path, endpoint) {
				continue
			}
			matched = append(matched, endpoint)
			for name := range byEndpoint[endpoint] {
				allowed[name] = true
			}
		}
		if len(matched) == 0 {
			continue
		}

		for key := range u.Query() {
			if universal[key] || allowed[key] || seen[u.Path+"|"+key] {
				continue
			}
			seen[u.Path+"|"+key] = true
			t.Errorf("UNCATALOGUED wire param %q on %s (%s) — the SDK sends it and the catalog does not know it exists.\n"+
				"Add it to catalog.go, or stop sending it.", key, strings.Join(matched, "+"), u.Path)
		}
	}
}
