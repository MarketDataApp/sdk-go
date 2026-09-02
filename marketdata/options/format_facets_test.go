package options

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	internalhttp "github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/retry"
)

// --- path builders ---

func TestChainPath_EmptySymbol(t *testing.T) {
	if _, _, err := chainPath("", nil); err == nil {
		t.Fatal("chainPath(\"\") should return an error")
	}
}

func TestChainPath_Params(t *testing.T) {
	path, params, err := chainPath("AAPL", []ChainOption{
		WithStrike(StrikeRange(150, 160)),
		WithSide(SideCall),
		WithStrikeLimit(10),
	})
	if err != nil {
		t.Fatalf("chainPath() error = %v", err)
	}
	if path != "options/chain/AAPL/" {
		t.Errorf("path = %q, want options/chain/AAPL/", path)
	}
	if params.Get("side") != "call" || params.Get("strikeLimit") != "10" {
		t.Errorf("params = %v, want side=call&strikeLimit=10", params)
	}
}

func TestChainPath_AllFilters(t *testing.T) {
	path, params, err := chainPath("AAPL", []ChainOption{
		WithExpiry(OnExpiration(time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC))),
		WithChainDate(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)),
		WithExpirationTypes(IncludeExpirationTypes("monthly")),
		WithRange(MoneynessITM),
		WithMinBid(1), WithMaxBid(2),
		WithMinAsk(1), WithMaxAsk(2),
		WithMaxBidAskSpread(0.5), WithMaxBidAskSpreadPct(10),
		WithMinOpenInterest(100), WithMinVolume(50),
		WithNonstandard(true), WithAM(true), WithPM(false),
	})
	if err != nil {
		t.Fatalf("chainPath() error = %v", err)
	}
	if path != "options/chain/AAPL/" {
		t.Errorf("path = %q, want options/chain/AAPL/", path)
	}
	want := map[string]string{
		"date": "2024-06-01", "monthly": "true", "range": "itm",
		"minBid": "1", "maxBid": "2", "minAsk": "1", "maxAsk": "2",
		"maxBidAskSpread": "0.5", "maxBidAskSpreadPct": "10",
		"minOpenInterest": "100", "minVolume": "50",
		"nonstandard": "true", "am": "true", "pm": "false",
	}
	for k, v := range want {
		if got := params.Get(k); got != v {
			t.Errorf("params[%q] = %q, want %q (full params: %v)", k, got, v, params)
		}
	}
}

func TestChainPath_InvalidExpiry(t *testing.T) {
	_, _, err := chainPath("AAPL", []ChainOption{WithExpiry(ExpirationBetween(time.Now(), time.Now().AddDate(0, 0, -1)))})
	if err == nil {
		t.Fatal("chainPath() with an invalid expiry filter should return an error")
	}
}

func TestChainPath_InvalidStrike(t *testing.T) {
	_, _, err := chainPath("AAPL", []ChainOption{WithStrike(StrikeRange(160, 150))})
	if err == nil {
		t.Fatal("chainPath() with an invalid strike range should return an error")
	}
}

func TestExpirationsPath_EmptySymbol(t *testing.T) {
	if _, _, err := expirationsPath("", nil); err == nil {
		t.Fatal("expirationsPath(\"\") should return an error")
	}
}

func TestExpirationsPath_Params(t *testing.T) {
	path, params, err := expirationsPath("AAPL", []ExpirationOption{WithExpirationStrike(150)})
	if err != nil {
		t.Fatalf("expirationsPath() error = %v", err)
	}
	if path != "options/expirations/AAPL/" {
		t.Errorf("path = %q, want options/expirations/AAPL/", path)
	}
	if params.Get("strike") != "150" {
		t.Errorf("params = %v, want strike=150", params)
	}
	// The facet must NOT force dateformat: the JSON method sets
	// dateformat=unix for its decoder, but here the wire format is the
	// user-visible output and forcing it would override the client's
	// WithDateFormat/MARKETDATA_DATE_FORMAT default, which Do() applies
	// only to absent keys.
	if _, ok := params["dateformat"]; ok {
		t.Errorf("params = %v, want no dateformat key", params)
	}
}

func TestExpirationsPath_WithDate(t *testing.T) {
	_, params, err := expirationsPath("AAPL", []ExpirationOption{WithExpirationDate(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))})
	if err != nil {
		t.Fatalf("expirationsPath() error = %v", err)
	}
	if params.Get("date") != "2024-06-01" {
		t.Errorf("params = %v, want date=2024-06-01", params)
	}
}

func TestQuotePath_EmptySymbol(t *testing.T) {
	if _, _, err := quotePath("", nil); err == nil {
		t.Fatal("quotePath(\"\") should return an error")
	}
}

func TestQuotePath_ValidWindow(t *testing.T) {
	_, params, err := quotePath("AAPL250117C00150000", []QuoteOption{
		WithOptionQuoteWindow(QuoteOnDate(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))),
	})
	if err != nil {
		t.Fatalf("quotePath() error = %v", err)
	}
	if params.Get("date") != "2024-06-01" {
		t.Errorf("params = %v, want date=2024-06-01", params)
	}
}

func TestQuotePath_InvalidWindow(t *testing.T) {
	_, _, err := quotePath("AAPL250117C00150000", []QuoteOption{WithOptionQuoteWindow(QuoteRange(time.Now(), time.Now().AddDate(0, 0, -1)))})
	if err == nil {
		t.Fatal("quotePath() with an invalid window should return an error")
	}
}

// --- CSVService ---

func TestCSVService_Chain(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("format"); got != "csv" {
			t.Errorf("format param = %q, want csv", got)
		}
		if got := r.Header.Get("Accept"); got != "text/csv" {
			t.Errorf("Accept header = %q, want text/csv", got)
		}
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("strike,bid,ask\n150,1.0,1.1\n"))
	})).AsCSV()

	got, err := svc.Chain(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
	if got.CSV() != "strike,bid,ask\n150,1.0,1.1\n" {
		t.Errorf("CSV() = %q", got.CSV())
	}
}

func TestCSVService_Chain_EmptySymbol(t *testing.T) {
	svc := (&Service{}).AsCSV()
	if _, err := svc.Chain(context.Background(), ""); err == nil {
		t.Fatal("Chain(\"\") should return a validation error without making a request")
	}
}

func TestCSVService_Expirations(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/v1/options/expirations/AAPL/" {
			t.Errorf("path = %q, want /v1/options/expirations/AAPL/", got)
		}
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("expiration\n2025-01-17\n"))
	})).AsCSV()

	got, err := svc.Expirations(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Expirations() error = %v", err)
	}
	if got.CSV() != "expiration\n2025-01-17\n" {
		t.Errorf("CSV() = %q", got.CSV())
	}
}

func TestCSVService_Expirations_EmptySymbol(t *testing.T) {
	svc := (&Service{}).AsCSV()
	if _, err := svc.Expirations(context.Background(), ""); err == nil {
		t.Fatal("Expirations(\"\") should return a validation error without making a request")
	}
}

func TestCSVService_Quote(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/v1/options/quotes/AAPL250117C00150000/" {
			t.Errorf("path = %q, want /v1/options/quotes/AAPL250117C00150000/", got)
		}
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("bid,ask\n1.0,1.1\n"))
	})).AsCSV()

	got, err := svc.Quote(context.Background(), "AAPL250117C00150000")
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if got.CSV() != "bid,ask\n1.0,1.1\n" {
		t.Errorf("CSV() = %q", got.CSV())
	}
}

func TestCSVService_Quote_EmptySymbol(t *testing.T) {
	svc := (&Service{}).AsCSV()
	if _, err := svc.Quote(context.Background(), ""); err == nil {
		t.Fatal("Quote(\"\") should return a validation error without making a request")
	}
}

func TestCSVService_Quotes_Single(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("bid,ask\n1.0,1.1\n"))
	})).AsCSV()

	got, err := svc.Quotes(context.Background(), []string{"AAPL250117C00150000"})
	if err != nil {
		t.Fatalf("Quotes() error = %v", err)
	}
	if len(got) != 1 || got["AAPL250117C00150000"] == nil {
		t.Fatalf("result = %v, want a single entry", got)
	}
}

func TestCSVService_Quotes_MultipleFanOut(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("bid,ask\n" + r.URL.Path))
	})).AsCSV()

	symbols := []string{"AAPL250117C00150000", "AAPL250117P00150000"}
	got, err := svc.Quotes(context.Background(), symbols)
	if err != nil {
		t.Fatalf("Quotes() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(got))
	}
	for _, sym := range symbols {
		if got[sym] == nil {
			t.Errorf("result[%q] = nil, want an entry", sym)
			continue
		}
		want := "bid,ask\n/v1/options/quotes/" + sym + "/"
		if got[sym].CSV() != want {
			t.Errorf("result[%q].CSV() = %q, want %q", sym, got[sym].CSV(), want)
		}
	}
}

func TestCSVService_Quotes_Empty(t *testing.T) {
	svc := (&Service{}).AsCSV()
	if _, err := svc.Quotes(context.Background(), nil); err == nil {
		t.Fatal("Quotes() with no symbols should return a validation error without making a request")
	}
}

func TestCSVService_ErrorPropagatesWithCSVParsedMessage(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(400)
		_, _ = w.Write([]byte("s,errmsg\nerror,\"Bad parameters, please check API documentation.\"\n"))
	})).AsCSV()

	_, err := svc.Chain(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("Chain() should return an error for a 400 response")
	}
	if got := err.Error(); got != "marketdata: bad request: Bad parameters, please check API documentation." {
		t.Errorf("err.Error() = %q, want the CSV-parsed errmsg embedded", got)
	}
}

func TestAsCSV_SharesHTTPClient(t *testing.T) {
	httpClient := internalhttp.New(internalhttp.Config{BaseURL: "http://example.test", APIVersion: "v1", Token: "k", RetryCfg: retry.DefaultConfig()})
	svc := NewService(httpClient)
	if svc.AsCSV().http != httpClient {
		t.Error("AsCSV() should carry the same http client as Service")
	}
}

// --- htmlService (package-private, see ADR-018) ---

func TestHTMLService_Chain(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("format"); got != "html" {
			t.Errorf("format param = %q, want html", got)
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>chain</html>"))
	})).asHTML()

	got, err := svc.Chain(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
	if got.HTML() != "<html>chain</html>" {
		t.Errorf("HTML() = %q", got.HTML())
	}
}

func TestHTMLService_Chain_EmptySymbol(t *testing.T) {
	svc := (&Service{}).asHTML()
	if _, err := svc.Chain(context.Background(), ""); err == nil {
		t.Fatal("Chain(\"\") should return a validation error without making a request")
	}
}

func TestHTMLService_Expirations(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>expirations</html>"))
	})).asHTML()

	got, err := svc.Expirations(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Expirations() error = %v", err)
	}
	if got.HTML() != "<html>expirations</html>" {
		t.Errorf("HTML() = %q", got.HTML())
	}
}

func TestHTMLService_Expirations_EmptySymbol(t *testing.T) {
	svc := (&Service{}).asHTML()
	if _, err := svc.Expirations(context.Background(), ""); err == nil {
		t.Fatal("Expirations(\"\") should return a validation error without making a request")
	}
}

func TestHTMLService_Quote(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>quote</html>"))
	})).asHTML()

	got, err := svc.Quote(context.Background(), "AAPL250117C00150000")
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if got.HTML() != "<html>quote</html>" {
		t.Errorf("HTML() = %q", got.HTML())
	}
}

func TestHTMLService_Quote_EmptySymbol(t *testing.T) {
	svc := (&Service{}).asHTML()
	if _, err := svc.Quote(context.Background(), ""); err == nil {
		t.Fatal("Quote(\"\") should return a validation error without making a request")
	}
}

func TestHTMLService_Quotes(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>quotes</html>"))
	})).asHTML()

	got, err := svc.Quotes(context.Background(), []string{"AAPL250117C00150000"})
	if err != nil {
		t.Fatalf("Quotes() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(got))
	}
}

func TestHTMLService_Quotes_Empty(t *testing.T) {
	svc := (&Service{}).asHTML()
	if _, err := svc.Quotes(context.Background(), nil); err == nil {
		t.Fatal("Quotes() with no symbols should return a validation error without making a request")
	}
}

func TestAsHTML_SharesHTTPClient(t *testing.T) {
	httpClient := internalhttp.New(internalhttp.Config{BaseURL: "http://example.test", APIVersion: "v1", Token: "k", RetryCfg: retry.DefaultConfig()})
	svc := NewService(httpClient)
	if svc.asHTML().http != httpClient {
		t.Error("asHTML() should carry the same http client as Service")
	}
}

// TestCSVService_Quotes_AppliesOptionsToEverySymbol pins the reason the
// signature changed from a variadic symbol list to a slice plus options: a
// historical window used to be expressible for one contract and not for a
// batch, so a watchlist could not be exported for last week.
func TestCSVService_Quotes_AppliesOptionsToEverySymbol(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]url.Values{}
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen[r.URL.Path] = r.URL.Query()
		mu.Unlock()
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("bid,ask\n1.0,1.1\n"))
	})).AsCSV()

	symbols := []string{"AAPL250117C00150000", "AAPL250117P00150000"}
	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 7, 0, 0, 0, 0, time.UTC)
	if _, err := svc.Quotes(context.Background(), symbols, WithOptionQuoteWindow(QuoteRange(from, to))); err != nil {
		t.Fatalf("Quotes() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 2 {
		t.Fatalf("got %d requests, want one per symbol", len(seen))
	}
	for path, params := range seen {
		if params.Get("from") != "2024-06-01" || params.Get("to") != "2024-06-07" {
			t.Errorf("%s: params = %v, want the window on every symbol", path, params)
		}
	}
}

// TestCSVService_Quotes_RejectsInvalidOptionsBeforeAnyRequest verifies the
// facet validates through the shared builder rather than hand-rolling a
// path: an unusable window must fail without spending a request.
func TestCSVService_Quotes_RejectsInvalidOptionsBeforeAnyRequest(t *testing.T) {
	var hits atomic.Int32
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("bid,ask\n1.0,1.1\n"))
	})).AsCSV()

	symbols := []string{"AAPL250117C00150000", "AAPL250117P00150000"}
	if _, err := svc.Quotes(context.Background(), symbols, WithOptionQuoteWindow(QuoteLastN(-1))); err == nil {
		t.Fatal("Quotes() with an invalid window should return a validation error")
	}
	if hits.Load() != 0 {
		t.Errorf("made %d requests, want 0 — validation must happen before the network", hits.Load())
	}
}
