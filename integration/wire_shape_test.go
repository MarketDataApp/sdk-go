//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/dotenv"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
)

// This file closes a gap the deep review's mutation testing found (T-4):
// the hand-written JSON fixtures in each package's testdata/ (added in the
// wire-contract PR) are checked against the SDK's own wire-struct tags —
// they catch a struct-tag typo or a transposed field, but they can never
// catch the API itself renaming, adding, or dropping a field, because both
// the fixture and the assertion trace back to the same source (the SDK's
// own tags). These tests close that loop: they fetch the RAW response body
// directly from the live API — bypassing the SDK's typed client entirely —
// and diff its top-level JSON keys against an expected key set transcribed
// independently from the wire struct definitions, in both directions. An
// extra live key is a field the SDK silently drops (encoding/json ignores
// unknown fields); a missing live key is one the SDK still expects but the
// API stopped sending.
//
// Coverage is the full 14-endpoint set the wire-contract fixtures cover
// (T-4, extended 2026-08-05). Extending it past the original 6-endpoint
// representative sample paid off immediately: it caught two real
// discrepancies invisible to the offline fixtures, both fixed alongside
// this file —
//
//   - Utilities.Headers() decoded assuming a {"headers": {...}} envelope,
//     but the live API echoes the request's headers as a flat object (no
//     wrapper key at all), so Headers() always returned an empty map. The
//     offline mock/fixture had encoded the SDK's own wrong struct, so they
//     could never have caught it — exactly the blind spot this file exists
//     to close.
//   - stocks/news sends a response-level "updated" timestamp the SDK
//     silently dropped. Now decoded onto every NewsArticle.Updated.

// wireToken resolves the API token through the same cascade the SDK
// itself uses (real env first, then .env) — not a bare os.Getenv, which
// would miss a token that lives only in .env, as it does in local
// development (see the A-3 fix in setupClient for the same reasoning).
// TestMain has already chdir'd the process to the module root, so a
// relative ".env" here resolves the same way it would for the SDK.
func wireToken(t *testing.T) string {
	t.Helper()
	if v := os.Getenv("MARKETDATA_TOKEN"); v != "" {
		return v
	}
	vars, _ := dotenv.Parse(".env")
	if v := vars["MARKETDATA_TOKEN"]; v != "" {
		return v
	}
	t.Skip("no API token available, skipping wire-shape test")
	return ""
}

// wireBaseURL mirrors the SDK's own MARKETDATA_BASE_URL override, if set,
// so this test hits the same host the rest of the integration suite does.
func wireBaseURL() string {
	if v := os.Getenv("MARKETDATA_BASE_URL"); v != "" {
		return v
	}
	return "https://api.marketdata.app"
}

// fetchRawKeys issues a raw GET request — bypassing the SDK's typed
// client entirely — and returns the top-level JSON keys of the response
// body, exactly as the live API sent them.
func fetchRawKeys(t *testing.T, path string) map[string]bool {
	t.Helper()
	token := wireToken(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wireBaseURL()+path, nil)
	if err != nil {
		t.Fatalf("build request for %s: %v", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request %s: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body for %s: %v", path, err)
	}
	if resp.StatusCode != 200 && resp.StatusCode != 203 {
		t.Fatalf("%s: unexpected status %d: %s", path, resp.StatusCode, body)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("%s: response is not a JSON object: %v (body: %s)", path, err, body)
	}
	keys := make(map[string]bool, len(raw))
	for k := range raw {
		keys[k] = true
	}
	return keys
}

// assertWireKeys fails with the specific keys that differ in each
// direction, so a failure immediately says whether the API added a field
// the SDK doesn't know about or stopped sending one the SDK expects.
func assertWireKeys(t *testing.T, endpoint string, live map[string]bool, expected []string) {
	t.Helper()
	want := make(map[string]bool, len(expected))
	for _, k := range expected {
		want[k] = true
	}

	var extra, missing []string
	for k := range live {
		if !want[k] {
			extra = append(extra, k)
		}
	}
	for k := range want {
		if !live[k] {
			missing = append(missing, k)
		}
	}
	sort.Strings(extra)
	sort.Strings(missing)

	if len(extra) > 0 {
		t.Errorf("%s: live API response has keys the SDK does not decode (silently dropped): %s", endpoint, strings.Join(extra, ", "))
	}
	if len(missing) > 0 {
		t.Errorf("%s: SDK expects keys the live response did not send: %s", endpoint, strings.Join(missing, ", "))
	}
}

func TestWireShape_StocksBulkQuotes(t *testing.T) {
	// Single symbol + 52week=true exercises the full field set: bulk quotes
	// only honor 52week for a single-symbol request.
	live := fetchRawKeys(t, "/v1/stocks/bulkquotes/?symbols="+TestStockSymbol+"&52week=true")
	assertWireKeys(t, "stocks/bulkquotes", live, []string{
		"s", "symbol", "ask", "askSize", "bid", "bidSize", "mid", "last",
		"change", "changepct", "volume", "updated", "52weekHigh", "52weekLow",
	})
}

func TestWireShape_StocksCandles(t *testing.T) {
	live := fetchRawKeys(t, "/v1/stocks/candles/D/"+TestStockSymbol+"/?countback=1")
	assertWireKeys(t, "stocks/candles", live, []string{"s", "t", "o", "h", "l", "c", "v"})
}

func TestWireShape_OptionsChain(t *testing.T) {
	// strikeLimit keeps the response small; the key set is the same
	// regardless of how many contracts are returned.
	live := fetchRawKeys(t, "/v1/options/chain/"+TestStockSymbol+"/?strikeLimit=1")
	assertWireKeys(t, "options/chain", live, []string{
		"s", "optionSymbol", "underlying", "expiration", "strike", "side",
		"bid", "bidSize", "ask", "askSize", "last", "mid", "volume",
		"openInterest", "iv", "delta", "gamma", "theta", "vega",
		"underlyingPrice", "intrinsicValue", "extrinsicValue", "firstTraded",
		"dte", "inTheMoney", "updated",
	})
}

func TestWireShape_OptionsExpirations(t *testing.T) {
	live := fetchRawKeys(t, "/v1/options/expirations/"+TestStockSymbol+"/")
	assertWireKeys(t, "options/expirations", live, []string{"s", "expirations", "updated"})
}

func TestWireShape_MarketsStatus(t *testing.T) {
	live := fetchRawKeys(t, "/v1/markets/status/")
	assertWireKeys(t, "markets/status", live, []string{"s", "date", "status"})
}

func TestWireShape_UtilitiesStatus(t *testing.T) {
	// Unversioned endpoint (no /v1 prefix), like the SDK's own GetUnversioned call.
	live := fetchRawKeys(t, "/status/")
	assertWireKeys(t, "utilities/status", live, []string{
		"s", "service", "status", "online", "uptimePct30d", "uptimePct90d", "updated",
	})
}

func TestWireShape_FundsCandles(t *testing.T) {
	live := fetchRawKeys(t, "/v1/funds/candles/D/"+TestFundSymbol+"/?countback=3")
	assertWireKeys(t, "funds/candles", live, []string{"s", "t", "o", "h", "l", "c"})
}

func TestWireShape_StocksBulkCandles(t *testing.T) {
	live := fetchRawKeys(t, "/v1/stocks/bulkcandles/D/?symbols="+TestStockSymbol+","+TestStockSymbol2)
	assertWireKeys(t, "stocks/bulkcandles", live, []string{"s", "symbol", "t", "o", "h", "l", "c", "v"})
}

func TestWireShape_StocksEarnings(t *testing.T) {
	// A bare countback (no to=) can degrade to the API's undocumented
	// upcoming-only default window and return {"s":"no_data"} outside an
	// earnings cycle (verified live); anchor with an explicit
	// to= of today, like the SDK's own Earnings does, so this reliably gets
	// a real response to check the shape of.
	to := time.Now().UTC().Format("2006-01-02")
	live := fetchRawKeys(t, "/v1/stocks/earnings/"+TestStockSymbol+"/?countback=4&to="+to)
	assertWireKeys(t, "stocks/earnings", live, []string{
		"s", "symbol", "fiscalYear", "fiscalQuarter", "date", "reportDate",
		"reportTime", "currency", "reportedEPS", "estimatedEPS", "surpriseEPS",
		"surpriseEPSpct", "updated",
	})
}

func TestWireShape_StocksNews(t *testing.T) {
	live := fetchRawKeys(t, "/v1/stocks/news/"+TestStockSymbol+"/?countback=5")
	assertWireKeys(t, "stocks/news", live, []string{
		"s", "symbol", "headline", "content", "source", "publicationDate", "updated",
	})
}

func TestWireShape_StocksPrices(t *testing.T) {
	// Even the single-symbol path-embedded form (stocks.pricesPath) still
	// gets "symbol" back as a one-element array, matching the bulk form —
	// verified live.
	live := fetchRawKeys(t, "/v1/stocks/prices/"+TestStockSymbol+"/")
	assertWireKeys(t, "stocks/prices", live, []string{"s", "symbol", "mid", "change", "changepct", "updated"})
}

func TestWireShape_OptionsLookup(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t)
	fx := resolveOptionsFixture(t, ctx, client)

	// Mirrors options.Service.Lookup's own query construction exactly: a
	// single human-readable path segment, "{underlying} {expiration}
	// {strike} {type}".
	query := TestStockSymbol + " " + fx.exp.Format("2006-01-02") + " " +
		strconv.FormatFloat(fx.atm, 'f', -1, 64) + " " + string(options.Call)
	live := fetchRawKeys(t, "/v1/options/lookup/"+url.PathEscape(query)+"/")
	assertWireKeys(t, "options/lookup", live, []string{"s", "optionSymbol"})
}

func TestWireShape_UtilitiesHeaders(t *testing.T) {
	// Unversioned, like /status/. Unlike every other endpoint in this file,
	// the live response has no envelope at all — the request's own headers
	// ARE the top-level object (verified live; the SDK's wrapper assumption
	// was fixed alongside this test). assertWireKeys' "extra keys" side is
	// meaningless here (the header set varies request to request), so this
	// only checks that the SDK's one always-present expectation shows up.
	live := fetchRawKeys(t, "/headers/")
	if !live["authorization"] && !live["Authorization"] {
		t.Errorf("utilities/headers: live response missing an Authorization header key (live keys: %v)", live)
	}
}

func TestWireShape_UtilitiesUser(t *testing.T) {
	// Unversioned, like /status/ and /headers/.
	live := fetchRawKeys(t, "/user/")
	assertWireKeys(t, "utilities/user", live, []string{
		"x-ratelimit-requests-remaining", "x-ratelimit-requests-limit", "x-options-data-permissions",
	})
}
