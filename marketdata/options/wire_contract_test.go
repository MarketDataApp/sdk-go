package options_test

// Wire-contract tests (ADR-010): hand-written fixtures from testdata/,
// asserted field by field with distinct values so a tag typo or a swapped
// field in the wire structs turns a test red. See the stocks package for
// the pattern's rationale.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	internalhttp "github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/ratelimit"
	"github.com/MarketDataApp/sdk-go/v2/internal/retry"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
)

func contractService(t *testing.T, fixture string) (*options.Service, func()) {
	t.Helper()
	body, err := os.ReadFile("testdata/" + fixture)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixture, err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	client := internalhttp.New(internalhttp.Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "k",
		RetryCfg: retry.DefaultConfig(), RateLimits: ratelimit.New(),
	})
	return options.NewService(client), server.Close
}

func TestWireContract_Chain(t *testing.T) {
	svc, done := contractService(t, "chain.json")
	defer done()
	chain, _, err := svc.Chain(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
	if chain == nil || chain.Underlying != "AAPL" || len(chain.Options) != 1 {
		t.Fatalf("chain = %+v, want underlying AAPL with 1 contract", chain)
	}
	q := chain.Options[0]
	checks := []struct {
		field string
		got   any
		want  any
	}{
		{"OptionSymbol", q.OptionSymbol, "AAPL250117C00150000"},
		{"Underlying", q.Underlying, "AAPL"},
		{"Expiration", q.Expiration.Unix(), int64(1737158400)},
		{"Strike", q.Strike, 150.5},
		{"Type", q.Type, options.Call},
		{"Bid", q.Bid, 5.05},
		{"BidSize", q.BidSize, 11},
		{"Ask", q.Ask, 5.25},
		{"AskSize", q.AskSize, 22},
		{"Last", q.Last, 5.15},
		{"Mid", q.Mid, 5.16},
		{"Volume", q.Volume, int64(1234)},
		{"OpenInterest", q.OpenInterest, int64(56789)},
		{"IV", q.IV, 0.3512},
		{"Delta", q.Delta, 0.5501},
		{"Gamma", q.Gamma, 0.0402},
		{"Theta", q.Theta, -0.0803},
		{"Vega", q.Vega, 0.1204},
		{"UnderlyingPrice", q.UnderlyingPrice, 151.11},
		{"IntrinsicValue", q.IntrinsicValue, 0.61},
		{"ExtrinsicValue", q.ExtrinsicValue, 4.55},
		{"FirstTraded", q.FirstTraded.Unix(), int64(1642425600)},
		{"DTE", q.DTE, 164},
		{"InTheMoney", q.InTheMoney, true},
		{"Updated", q.Updated.Unix(), int64(1704067200)},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.field, c.got, c.want)
		}
	}
}

func TestWireContract_Expirations(t *testing.T) {
	svc, done := contractService(t, "expirations.json")
	defer done()
	exps, _, err := svc.Expirations(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Expirations() error = %v", err)
	}
	if exps == nil || len(exps.Dates) != 2 {
		t.Fatalf("exps = %+v, want 2 dates", exps)
	}
	if exps.Dates[0].Unix() != 1737158400 || exps.Dates[1].Unix() != 1739577600 {
		t.Errorf("Dates = %v, do not match fixture", exps.Dates)
	}
	// The wire updated for expirations is a response-level SCALAR, not an
	// array — pinned here so it cannot silently regress to array decoding.
	if exps.Updated.Unix() != 1704067200 {
		t.Errorf("Updated = %v, want unix 1704067200", exps.Updated)
	}
}

func TestWireContract_Lookup(t *testing.T) {
	svc, done := contractService(t, "lookup.json")
	defer done()
	sym, _, err := svc.Lookup(context.Background(), "AAPL", time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC), 150, options.Call)
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if sym != "AAPL250117C00150000" {
		t.Errorf("Lookup() = %q, want the fixture's optionSymbol", sym)
	}
}

// TestPublicStructJSONTags_OptionQuote is a round-trip regression test for
// OptionQuote's json tags (T-5) — see stocks.TestPublicStructJSONTags_Quote
// for the rationale.
func TestPublicStructJSONTags_OptionQuote(t *testing.T) {
	q := options.OptionQuote{
		OptionSymbol: "AAPL250117C00150000", Underlying: "AAPL",
		Expiration: time.Unix(1737158400, 0), Strike: 150.5, Type: options.Call,
		Bid: 5.05, BidSize: 11, Ask: 5.25, AskSize: 22, Last: 5.15, Mid: 5.16,
		Volume: 1234, OpenInterest: 56789, IV: 0.3512, Delta: 0.5501,
		Gamma: 0.0402, Theta: -0.0803, Vega: 0.1204, UnderlyingPrice: 151.11,
		IntrinsicValue: 0.61, ExtrinsicValue: 4.55,
		FirstTraded: time.Unix(1642425600, 0), DTE: 164, InTheMoney: true,
		Updated: time.Unix(1704067200, 0),
	}
	body, err := json.Marshal(q)
	if err != nil {
		t.Fatalf("json.Marshal(OptionQuote) error = %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("json.Unmarshal error = %v", err)
	}
	for _, key := range []string{
		"optionSymbol", "underlying", "expiration", "strike", "side", "bid",
		"bidSize", "ask", "askSize", "last", "mid", "volume", "openInterest",
		"iv", "delta", "gamma", "theta", "vega", "underlyingPrice",
		"intrinsicValue", "extrinsicValue", "firstTraded", "dte",
		"inTheMoney", "updated",
	} {
		if _, ok := m[key]; !ok {
			t.Errorf("marshaled OptionQuote is missing key %q", key)
		}
	}
}
