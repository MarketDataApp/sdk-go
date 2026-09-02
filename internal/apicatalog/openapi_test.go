package apicatalog

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

// The catalog in catalog.go is hand-authored, and every other guarantee in this
// package is built on it: the reachability proof shows that each catalogued
// parameter has an SDK accessor, and integration/catalog_live_test.go shows the
// API accepts each one. Neither can say anything about a parameter nobody wrote
// down. That blind spot was not theoretical: `expiration=all`, the strike and
// delta lists, and four more parameters all lived in it, and all three suites
// stayed green throughout.
//
// This test closes the loop from the other side: it compares the catalog
// against the API's own published OpenAPI schema, captured in
// testdata/openapi_v1.json, and fails when either side knows about a parameter
// the other does not. Divergences that are genuine and understood are listed in
// the two allowlists below, each with the evidence for why it is there — so the
// file doubles as the written record of every place the SDK and the API's
// declared surface disagree on purpose.
//
// The fixture is checked in rather than fetched, so this runs offline in normal
// CI. integration/openapi_drift_test.go re-fetches the live schema and fails
// when the fixture has gone stale, which is what makes the allowlists below
// trustworthy over time.

// openapiFixture is the shape of testdata/openapi_v1.json.
type openapiFixture struct {
	Source  string              `json:"_source"`
	Fetched string              `json:"_fetched"`
	Note    string              `json:"_note"`
	Paths   map[string][]string `json:"paths"`
}

// universalParams are sent by the client for every request rather than being
// declared per method, so the schema lists them on every path while the catalog
// records them once under the "universal" pseudo-endpoint. They are subtracted
// from both sides before comparing.
var universalWireParams = map[string]bool{
	"columns": true, "dateformat": true, "format": true,
	"headers": true, "human": true, "limit": true, "offset": true,
}

// schemaPathToEndpoint maps each OpenAPI path the SDK implements to its catalog
// endpoint name. Paths absent from this map are out of the SDK's scope and are
// listed in outOfScopePaths with the reason.
var schemaPathToEndpoint = map[string]string{
	"/v1/options/chain/{underlying}/":           "options/chain",
	"/v1/options/expirations/{underlying}/":     "options/expirations",
	"/v1/options/lookup/{userInput}/":           "options/lookup",
	"/v1/options/quotes/{optionSymbol}/":        "options/quotes",
	"/v1/stocks/bulkcandles/{resolution}/":      "stocks/bulkcandles",
	"/v1/stocks/bulkquotes/":                    "stocks/quotes",
	"/v1/stocks/candles/{resolution}/{symbol}/": "stocks/candles",
	"/v1/stocks/earnings/{symbol}/":             "stocks/earnings",
	"/v1/stocks/news/{symbol}/":                 "stocks/news",
	"/v1/stocks/prices/":                        "stocks/prices",
}

// marketsPath is compared against the union of two catalog endpoints: the API
// serves the current status and the historical range from one path, while the
// SDK splits them into Markets.Status and Markets.StatusHistory so a single
// date and a range cannot be combined (ADR-017). Comparing per-endpoint would
// report each half's parameters as missing from the other.
const marketsPath = "/v1/markets/status/"

var marketsEndpoints = []string{"markets/status", "markets/status-history"}

// outOfScopePaths are schema paths the SDK deliberately does not implement.
// Listing them here rather than ignoring unknown paths means a NEW endpoint
// appearing in the API fails this test instead of passing unnoticed.
var outOfScopePaths = map[string]string{
	"/v1/options/strikes/{underlying}/": "requirements §2.2 does not list a strikes capability; Chain with strike filters covers the use case (see docs/COMPARISON.md)",
	"/v1/stocks/prices/{symbol}/":       "single-symbol prices; the SDK serves Prices from the bulk /v1/stocks/prices/ path for one symbol or many",
	"/v1/stocks/quotes/{symbol}/":       "single-symbol quotes; the SDK serves both Quote and Quotes from /v1/stocks/bulkquotes/",
	"/v1/stocks/quotes/":                "bulk quotes without the 52week/candle support the SDK needs; it uses /v1/stocks/bulkquotes/ instead",
}

// schemaOnly lists parameters the schema declares that the SDK deliberately
// does not expose. Each entry records why, with the live evidence.
var schemaOnly = map[string]map[string]string{
	"stocks/candles": {
		"exchange": "declared but inert: the endpoint accepts any value and ignores it — probed live 2026-08-19, RY with exchange=XTSE and with no exchange returned byte-identical bodies, and AAPL with exchange=BASURATOTAL returned the normal US candles at HTTP 203. WithCandleExchange was removed rather than kept as an inert option",
		"country":  "declared but inert: see exchange. RY with country=CA byte-identical to the default; AAPL with country=BOGUS returned the normal US candles",
	},
	"stocks/bulkcandles": {
		// The schema reuses the candles parameter list for bulkcandles, but
		// the endpoint returns one candle per symbol for a single day.
		"countback": "declared but inert: live probe with countback=3&to=... returned one candle per symbol dated today (2026-08-11)",
		"from":      "declared but inert: see countback",
		"to":        "declared but inert: see countback",
		"extended":  "declared but inert: bulk candles are daily only",
		"exchange":  "declared but inert: symbol resolution is per-request on the single-symbol candles path",
		"country":   "declared but inert: see exchange",
	},
	"options/expirations": {
		"nonstandard": "REAL GAP, tracked: the handler honors it (verified live 2026-08-11) and the SDK cannot send it. Pending alongside the weekly/monthly/quarterly filters the schema does not declare either",
	},
}

// catalogOnly lists parameters the SDK sends that the schema does not declare.
// The schema is generated from drf-spectacular decorators, which have drifted
// behind the handlers; every entry here was verified against the live API.
var catalogOnly = map[string]map[string]string{
	"stocks/quotes": {
		"52week": "honored on the single-symbol form only (verified live)",
		"candle": "adds session OHLC on both forms (verified live 2026-08-11)",
	},
	"stocks/prices": {
		"extended": "read by stock_price_handlers.py; undeclared in the schema",
	},
	"stocks/bulkcandles": {
		"symbols":  "required by the endpoint (or snapshot=true); undeclared in the schema",
		"snapshot": "market-wide snapshot mode, verified live",
	},
}

func loadOpenAPIFixture(t *testing.T) openapiFixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/openapi_v1.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fx openapiFixture
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if len(fx.Paths) == 0 {
		t.Fatal("fixture declares no paths")
	}
	return fx
}

// catalogQueryParams returns the query parameters the catalog records for an
// endpoint, minus the universal ones.
func catalogQueryParams(endpoint string) map[string]bool {
	out := map[string]bool{}
	for _, p := range All() {
		if p.Endpoint != endpoint || p.Kind != Query || universalWireParams[p.Name] {
			continue
		}
		out[p.Name] = true
	}
	return out
}

// schemaQueryParams returns the schema's query parameters for a path, minus the
// universal ones.
func schemaQueryParams(fx openapiFixture, path string) map[string]bool {
	out := map[string]bool{}
	for _, n := range fx.Paths[path] {
		if !universalWireParams[n] {
			out[n] = true
		}
	}
	return out
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestOpenAPI_CatalogMatchesSchema is the audit: every parameter the API
// declares must be reachable through the SDK, and every parameter the SDK sends
// must be one the API declares — unless the divergence is allowlisted above
// with its evidence.
func TestOpenAPI_CatalogMatchesSchema(t *testing.T) {
	fx := loadOpenAPIFixture(t)

	compare := func(t *testing.T, label string, schema, catalog map[string]bool, allowSchema, allowCatalog map[string]string) {
		t.Helper()
		for _, name := range sortedKeys(schema) {
			if catalog[name] {
				continue
			}
			if reason, ok := allowSchema[name]; ok {
				t.Logf("%s: %q declared by the API but not exposed — %s", label, name, reason)
				continue
			}
			t.Errorf("%s: the API declares %q and the SDK has no way to send it.\n"+
				"Either add it to the catalog with an accessor, or add it to schemaOnly with the reason.",
				label, name)
		}
		for _, name := range sortedKeys(catalog) {
			if schema[name] {
				continue
			}
			if reason, ok := allowCatalog[name]; ok {
				t.Logf("%s: %q sent by the SDK but undeclared by the API — %s", label, name, reason)
				continue
			}
			t.Errorf("%s: the SDK sends %q but the API does not declare it.\n"+
				"Either it was removed from the API, or it belongs in catalogOnly with the evidence.",
				label, name)
		}
	}

	for path, endpoint := range schemaPathToEndpoint {
		t.Run(endpoint, func(t *testing.T) {
			if _, ok := fx.Paths[path]; !ok {
				t.Fatalf("path %q is gone from the schema; the SDK still implements it as %q", path, endpoint)
			}
			compare(t, endpoint,
				schemaQueryParams(fx, path), catalogQueryParams(endpoint),
				schemaOnly[endpoint], catalogOnly[endpoint])
		})
	}

	// markets: one API path, two SDK endpoints (see marketsEndpoints).
	t.Run("markets", func(t *testing.T) {
		union := map[string]bool{}
		for _, ep := range marketsEndpoints {
			for name := range catalogQueryParams(ep) {
				union[name] = true
			}
		}
		compare(t, "markets/status(+history)",
			schemaQueryParams(fx, marketsPath), union,
			schemaOnly["markets/status"], catalogOnly["markets/status"])
	})
}

// TestOpenAPI_AllowlistsAreNotStale keeps the two allowlists honest.
//
// TestOpenAPI_CatalogMatchesSchema only consults an allowlist entry when the
// two sides actually disagree, so an entry that stopped being true is never
// read: the test stays green while its written record contradicts shipping
// code. That is exactly what happened to stocks/earnings "report" — it was
// listed as deliberately not exposed one day before WithEarningsReport
// shipped, and the audit went on describing the SDK as it no longer was.
//
// The failure mode is worse than a stale comment. The catalog is the audit's
// only view of the SDK, so a parameter parked in schemaOnly is invisible to
// every catalog-driven check: reachability, accessor coverage, and any
// future serializer-parity check all skip it silently.
//
// So an allowlist entry must be BOTH still needed (the disagreement it
// records still exists) and still real (it names an endpoint and a parameter
// that exist).
func TestOpenAPI_AllowlistsAreNotStale(t *testing.T) {
	fx := loadOpenAPIFixture(t)

	endpointToPath := map[string]string{}
	for path, endpoint := range schemaPathToEndpoint {
		endpointToPath[endpoint] = path
	}
	// markets: both allowlists are keyed by markets/status, while the
	// catalog side is the union of the two SDK endpoints the one API path
	// serves (see marketsEndpoints).
	endpointToPath["markets/status"] = marketsPath

	catalogParams := func(endpoint string) map[string]bool {
		if endpoint != "markets/status" {
			return catalogQueryParams(endpoint)
		}
		union := map[string]bool{}
		for _, ep := range marketsEndpoints {
			for name := range catalogQueryParams(ep) {
				union[name] = true
			}
		}
		return union
	}

	check := func(t *testing.T, listName string, list map[string]map[string]string, stale func(name string, schema, catalog map[string]bool) string) {
		t.Helper()
		for _, endpoint := range sortedAllowlistKeys(list) {
			path, ok := endpointToPath[endpoint]
			if !ok {
				t.Errorf("%s names endpoint %q, which no schema path maps to — a typo, or the endpoint is gone", listName, endpoint)
				continue
			}
			schema := schemaQueryParams(fx, path)
			catalog := catalogParams(endpoint)
			for _, name := range sortedAllowlistKeys(list[endpoint]) {
				if reason := stale(name, schema, catalog); reason != "" {
					t.Errorf("%s[%q][%q] is stale: %s.\nDelete the entry; leaving it masks the disagreement it no longer describes.",
						listName, endpoint, name, reason)
				}
			}
		}
	}

	// A schemaOnly entry claims the API declares a parameter the SDK does
	// not send. It goes stale when the SDK starts sending it, and is dead if
	// the API stopped declaring it.
	check(t, "schemaOnly", schemaOnly, func(name string, schema, catalog map[string]bool) string {
		switch {
		case catalog[name]:
			return "the catalog now exposes it, so the SDK does send it"
		case !schema[name]:
			return "the API no longer declares it, so there is nothing to exempt"
		}
		return ""
	})

	// A catalogOnly entry claims the SDK sends a parameter the API does not
	// declare. It goes stale when the schema catches up, and is dead if the
	// SDK stopped sending it.
	check(t, "catalogOnly", catalogOnly, func(name string, schema, catalog map[string]bool) string {
		switch {
		case schema[name]:
			return "the API now declares it, so the divergence is gone"
		case !catalog[name]:
			return "the catalog no longer sends it, so there is nothing to explain"
		}
		return ""
	})
}

func sortedAllowlistKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestOpenAPI_EveryPathIsClassified fails when the API grows a path that is
// neither implemented nor explicitly declared out of scope. Without this, a new
// endpoint would simply never be noticed.
func TestOpenAPI_EveryPathIsClassified(t *testing.T) {
	fx := loadOpenAPIFixture(t)

	for path := range fx.Paths {
		if _, ok := schemaPathToEndpoint[path]; ok {
			continue
		}
		if path == marketsPath {
			continue
		}
		if reason, ok := outOfScopePaths[path]; ok {
			t.Logf("%s: not implemented — %s", path, reason)
			continue
		}
		t.Errorf("the API declares %q, which the SDK neither implements nor lists as out of scope.\n"+
			"Add it to schemaPathToEndpoint, or to outOfScopePaths with the reason.", path)
	}

	// The reverse: an out-of-scope entry for a path the API no longer has is
	// stale bookkeeping and should be deleted.
	for path := range outOfScopePaths {
		if _, ok := fx.Paths[path]; !ok {
			t.Errorf("outOfScopePaths lists %q, which the API no longer declares; remove the entry", path)
		}
	}
}

// TestOpenAPI_CatalogEndpointsAreCoveredOrExempt records which catalog
// endpoints this audit cannot check, so the gap is visible rather than
// implicit. The API's published schema covers only its versioned /v1/ surface,
// and not all of it.
func TestOpenAPI_CatalogEndpointsAreCoveredOrExempt(t *testing.T) {
	// funds/candles is live and versioned but absent from the published
	// schema; the utilities endpoints are unversioned (/status/, /user/,
	// /headers/) and fall outside the /v1/ document entirely.
	exempt := map[string]string{
		"funds/candles":     "live and versioned, but the published OpenAPI schema omits the funds app entirely (verified 2026-08-12)",
		"utilities/status":  "served from the unversioned /status/ path, outside the /v1/ schema",
		"utilities/user":    "served from the unversioned /user/ path, outside the /v1/ schema",
		"utilities/headers": "served from the unversioned /headers/ path, outside the /v1/ schema",
		"universal":         "client-level parameters, compared by subtraction rather than per endpoint",
	}

	covered := map[string]bool{}
	for _, ep := range schemaPathToEndpoint {
		covered[ep] = true
	}
	for _, ep := range marketsEndpoints {
		covered[ep] = true
	}

	seen := map[string]bool{}
	for _, p := range All() {
		if seen[p.Endpoint] {
			continue
		}
		seen[p.Endpoint] = true
		if covered[p.Endpoint] {
			continue
		}
		if reason, ok := exempt[p.Endpoint]; ok {
			t.Logf("%s: not audited against the schema — %s", p.Endpoint, reason)
			continue
		}
		t.Errorf("catalog endpoint %q is neither audited against the schema nor exempt.\n"+
			"Map it in schemaPathToEndpoint, or record why it cannot be audited.", p.Endpoint)
	}
}

// maxFixtureAge bounds how stale the checked-in OpenAPI snapshot may get
// before the offline audit stops trusting itself. Ninety days is long enough
// not to nag on an ordinary release cadence and short enough that a schema
// change cannot sit undetected for a year in a repository without a token.
const maxFixtureAge = 90 * 24 * time.Hour

// TestOpenAPI_FixtureMetadata keeps the fixture self-describing: a bare list of
// parameters with no provenance is impossible to re-derive later.
func TestOpenAPI_FixtureMetadata(t *testing.T) {
	fx := loadOpenAPIFixture(t)

	if !strings.Contains(fx.Source, "api.marketdata.app/schema") {
		t.Errorf("_source = %q, want the published schema URL", fx.Source)
	}
	if fx.Fetched == "" {
		t.Fatal("_fetched is empty: the fixture must record when it was captured")
	}

	// Parse it, and fail once it is old. The offline audit is only as good
	// as this snapshot, and nothing offline could tell that it had gone
	// stale — the check was that the field is non-empty, which a date of
	// 1999-01-01 satisfies. The live drift test does compare against the
	// published schema, but it is token-gated, so a repository whose
	// MARKETDATA_TOKEN secret is unset had no freshness signal at all.
	fetched, err := time.Parse("2006-01-02", fx.Fetched)
	if err != nil {
		t.Fatalf("_fetched = %q, want a YYYY-MM-DD date: %v", fx.Fetched, err)
	}
	if age := time.Since(fetched); age > maxFixtureAge {
		t.Errorf("the OpenAPI fixture was captured %s (%d days ago), beyond the %d-day limit.\n"+
			"Re-capture it from %s — see _note in the fixture.",
			fx.Fetched, int(age.Hours()/24), int(maxFixtureAge.Hours()/24), fx.Source)
	}
	if fx.Note == "" {
		t.Error("_note is empty: the fixture must say how to regenerate itself")
	}
}
