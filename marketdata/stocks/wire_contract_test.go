package stocks_test

// Wire-contract tests (ADR-010): each test serves a hand-written fixture
// from testdata/ — never a serialized SDK struct — and asserts every public
// field decoded from it. A JSON tag typo or swapped field in the wire
// structs turns one of these red; the fixtures double as the documented
// wire format. Fixture values are distinct per field so transposed tags
// cannot cancel out.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

// serveFixture returns a server that answers every request with the named
// testdata fixture.
func serveFixture(t *testing.T, name string) *httptest.Server {
	t.Helper()
	body, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
}

func TestWireContract_Quotes(t *testing.T) {
	server := serveFixture(t, "quotes.json")
	defer server.Close()
	q, _, err := newTestService(server.URL).Quote(context.Background(), "AAPL", stocks.WithFiftyTwoWeek(true))
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}

	checks := []struct {
		field string
		got   any
		want  any
	}{
		{"Symbol", q.Symbol, "AAPL"},
		{"Ask", q.Ask, 151.25},
		{"AskSize", q.AskSize, 101},
		{"Bid", q.Bid, 150.20},
		{"BidSize", q.BidSize, 202},
		{"Mid", q.Mid, 150.72},
		{"Last", q.Last, 150.99},
		{"Change", q.Change, 1.53},
		{"ChangePercent", q.ChangePercent, 0.0104},
		{"Volume", q.Volume, int64(50000123)},
		{"Updated", q.Updated.Unix(), int64(1704067200)},
		{"FiftyTwoWeekHigh", q.FiftyTwoWeekHigh, 199.62},
		{"FiftyTwoWeekLow", q.FiftyTwoWeekLow, 124.17},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.field, c.got, c.want)
		}
	}
}

func TestWireContract_Candles(t *testing.T) {
	server := serveFixture(t, "candles.json")
	defer server.Close()
	candles, _, err := newTestService(server.URL).Candles(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
	if len(candles) != 1 {
		t.Fatalf("len = %d, want 1", len(candles))
	}
	c := candles[0]
	if c.Time.Unix() != 1704067200 || c.Open != 170.11 || c.High != 171.22 || c.Low != 169.33 || c.Close != 170.99 || c.Volume != 45000111 {
		t.Errorf("candle = %+v, does not match fixture", c)
	}
}

// TestWireContract_BulkCandles uses the shape production actually sends for
// a ONE-symbol request: no symbol array at all. The fixture used to carry
// one anyway while the test asked for a single symbol, so it certified a
// response the API has never sent — and hid the fact that the decoder,
// which keyed off that array, returned nothing.
func TestWireContract_BulkCandles(t *testing.T) {
	server := serveFixture(t, "bulkcandles.json")
	defer server.Close()
	candles, _, err := newTestService(server.URL).BulkCandles(context.Background(), []string{"AAPL"})
	if err != nil {
		t.Fatalf("BulkCandles() error = %v", err)
	}
	if len(candles) != 1 {
		t.Fatalf("len = %d, want 1", len(candles))
	}
	b := candles[0]
	// Symbol is restored from the request, since the wire has none.
	if b.Symbol != "AAPL" || b.Time.Unix() != 1704067200 || b.Open != 170.11 || b.High != 171.22 || b.Low != 169.33 || b.Close != 170.99 || b.Volume != 45000111 {
		t.Errorf("bulk candle = %+v, does not match fixture", b)
	}
}

// TestWireContract_BulkCandlesMulti covers the other live shape: two or more
// symbols (or one with snapshot=true) do carry the symbol array, and it wins.
func TestWireContract_BulkCandlesMulti(t *testing.T) {
	server := serveFixture(t, "bulkcandles_multi.json")
	defer server.Close()
	candles, _, err := newTestService(server.URL).BulkCandles(context.Background(), []string{"AAPL", "MSFT"})
	if err != nil {
		t.Fatalf("BulkCandles() error = %v", err)
	}
	if len(candles) != 2 {
		t.Fatalf("len = %d, want 2", len(candles))
	}
	if candles[0].Symbol != "AAPL" || candles[0].Close != 170.99 {
		t.Errorf("candles[0] = %+v, does not match fixture", candles[0])
	}
	if candles[1].Symbol != "MSFT" || candles[1].Close != 372.55 || candles[1].Volume != 22000333 {
		t.Errorf("candles[1] = %+v, does not match fixture", candles[1])
	}
}

func TestWireContract_Earnings(t *testing.T) {
	server := serveFixture(t, "earnings.json")
	defer server.Close()
	earnings, _, err := newTestService(server.URL).Earnings(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Earnings() error = %v", err)
	}
	if len(earnings) != 1 {
		t.Fatalf("len = %d, want 1", len(earnings))
	}
	e := earnings[0]
	fp := func(p *float64) float64 {
		if p == nil {
			t.Fatal("nil EPS pointer for a populated fixture field")
		}
		return *p
	}
	checks := []struct {
		field string
		got   any
		want  any
	}{
		{"Symbol", e.Symbol, "AAPL"},
		{"FiscalYear", e.FiscalYear, 2024},
		{"FiscalQuarter", e.FiscalQuarter, 1},
		{"Date", e.Date.Unix(), int64(1703998800)},
		{"ReportDate", e.ReportDate.Unix(), int64(1706763600)},
		{"ReportTime", e.ReportTime, "after close"},
		{"Currency", e.Currency, "USD"},
		{"ReportedEPS", fp(e.ReportedEPS), 2.18},
		{"EstimatedEPS", fp(e.EstimatedEPS), 2.10},
		{"SurpriseEPS", fp(e.SurpriseEPS), 0.08},
		{"SurpriseEPSPercent", fp(e.SurpriseEPSPercent), 0.0381},
		{"Updated", e.Updated.Unix(), int64(1706850000)},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.field, c.got, c.want)
		}
	}
}

func TestWireContract_News(t *testing.T) {
	server := serveFixture(t, "news.json")
	defer server.Close()
	news, _, err := newTestService(server.URL).News(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("News() error = %v", err)
	}
	if len(news) != 1 {
		t.Fatalf("len = %d, want 1", len(news))
	}
	n := news[0]
	if n.Symbol != "AAPL" || n.Headline != "Apple unveils something" || n.Content != "Full article body" ||
		n.Source != "https://example.com/apple" || n.PublicationDate.Unix() != 1704153600 || n.Updated.Unix() != 1704200000 {
		t.Errorf("news article = %+v, does not match fixture", n)
	}
}

func TestWireContract_Prices(t *testing.T) {
	server := serveFixture(t, "prices.json")
	defer server.Close()
	prices, _, err := newTestService(server.URL).Prices(context.Background(), []string{"AAPL"})
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}
	if len(prices) != 1 {
		t.Fatalf("len = %d, want 1", len(prices))
	}
	p := prices[0]
	if p.Symbol != "AAPL" || p.Mid != 150.72 || p.Change != 1.53 || p.ChangePercent != 0.0104 || p.Updated.Unix() != 1704067200 {
		t.Errorf("price = %+v, does not match fixture", p)
	}
}

// TestPublicStructJSONTags_Quote is a round-trip regression test for the
// public Quote struct's json tags (T-5): nothing previously verified that
// marshaling a Quote — something a caller re-serializing SDK results would
// naturally do — produces the field names the struct's tags promise. A
// typo or an accidental tag removal here would go unnoticed by the
// wire-contract tests above, which only exercise the unexported response
// structs on the decode side.
func TestPublicStructJSONTags_Quote(t *testing.T) {
	q := stocks.Quote{
		Symbol: "AAPL", Ask: 151.25, AskSize: 101, Bid: 150.20, BidSize: 202,
		Mid: 150.72, Last: 150.99, Change: 1.53, ChangePercent: 0.0104,
		Volume: 50000123, Updated: time.Unix(1704067200, 0),
		FiftyTwoWeekHigh: 199.62, FiftyTwoWeekLow: 124.17,
	}
	body, err := json.Marshal(q)
	if err != nil {
		t.Fatalf("json.Marshal(Quote) error = %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("json.Unmarshal error = %v", err)
	}
	for _, key := range []string{
		"symbol", "ask", "askSize", "bid", "bidSize", "mid", "last",
		"change", "changepct", "volume", "updated", "52weekHigh", "52weekLow",
	} {
		if _, ok := m[key]; !ok {
			t.Errorf("marshaled Quote is missing key %q (present keys: %v)", key, keysOf(m))
		}
	}
}

func keysOf(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
