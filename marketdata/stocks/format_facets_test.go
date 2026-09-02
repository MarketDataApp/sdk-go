package stocks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	internalhttp "github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/retry"
	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

func TestQuotePath_EmptySymbol(t *testing.T) {
	_, _, err := quotePath("", nil)
	if ve, ok := err.(*sdkerrors.ValidationError); !ok || ve.Field != "symbol" {
		t.Errorf("err = %v (%T), want *sdkerrors.ValidationError{Field: \"symbol\"}", err, err)
	}
}

func TestQuotePath_Params(t *testing.T) {
	path, params, err := quotePath("AAPL", []QuoteOption{WithFiftyTwoWeek(true), WithExtended(false), WithCandle(true)})
	if err != nil {
		t.Fatalf("quotePath() error = %v", err)
	}
	if path != "stocks/bulkquotes/" {
		t.Errorf("path = %q, want stocks/bulkquotes/", path)
	}
	if params.Get("symbols") != "AAPL" || params.Get("52week") != "true" || params.Get("extended") != "false" {
		t.Errorf("params = %v, want symbols=AAPL&52week=true&extended=false", params)
	}
	// candle used to be dropped here while Service.Quote sent it, so the CSV
	// facet silently returned quotes without the session OHLC fields.
	if params.Get("candle") != "true" {
		t.Errorf("params = %v, want candle=true", params)
	}
}

func TestQuotesPath_EmptySymbols(t *testing.T) {
	if _, _, err := quotesPath(nil, nil); err == nil {
		t.Fatal("quotesPath(nil) should return an error")
	}
}

func TestQuotesPath_Params(t *testing.T) {
	path, params, err := quotesPath([]string{"AAPL", "MSFT"}, []QuotesOption{WithQuotesExtended(true), WithQuotesCandle(false)})
	if err != nil {
		t.Fatalf("quotesPath() error = %v", err)
	}
	if path != "stocks/bulkquotes/" || params.Get("symbols") != "AAPL,MSFT" || params.Get("extended") != "true" {
		t.Errorf("path=%q params=%v, want bulkquotes with joined symbols and extended=true", path, params)
	}
	if params.Get("candle") != "false" {
		t.Errorf("params = %v, want candle=false", params)
	}
}

func TestPricesPath_EmptySymbols(t *testing.T) {
	if _, _, err := pricesPath(nil, nil); err == nil {
		t.Fatal("pricesPath(nil) should return an error")
	}
}

func TestPricesPath_SingleVsMulti(t *testing.T) {
	path, params, err := pricesPath([]string{"AAPL"}, nil)
	if err != nil {
		t.Fatalf("pricesPath() error = %v", err)
	}
	if path != "stocks/prices/AAPL/" || params.Get("symbols") != "" {
		t.Errorf("single-symbol path = %q params = %v, want path-embedded symbol and no symbols param", path, params)
	}

	path, params, err = pricesPath([]string{"AAPL", "MSFT"}, []PriceOption{WithPriceExtended(true)})
	if err != nil {
		t.Fatalf("pricesPath() error = %v", err)
	}
	if path != "stocks/prices/" || params.Get("symbols") != "AAPL,MSFT" || params.Get("extended") != "true" {
		t.Errorf("multi-symbol path=%q params=%v, want stocks/prices/ with joined symbols param", path, params)
	}
}

func TestNewsPath_EmptySymbol(t *testing.T) {
	if _, _, err := newsPath("", nil); err == nil {
		t.Fatal("newsPath(\"\") should return an error")
	}
}

func TestNewsPath_InvalidWindow(t *testing.T) {
	_, _, err := newsPath("AAPL", []NewsOption{WithNewsWindow(LastN(-1))})
	if err == nil {
		t.Fatal("newsPath() with an invalid window should return an error")
	}
}

func TestEarningsPath_EmptySymbol(t *testing.T) {
	if _, _, err := earningsPath("", nil); err == nil {
		t.Fatal("earningsPath(\"\") should return an error")
	}
}

func TestEarningsPath_AnchorsCountback(t *testing.T) {
	path, params, err := earningsPath("AAPL", []EarningsOption{WithEarningsWindow(LastN(4))})
	if err != nil {
		t.Fatalf("earningsPath() error = %v", err)
	}
	if path != "stocks/earnings/AAPL/" {
		t.Errorf("path = %q, want stocks/earnings/AAPL/", path)
	}
	if params.Get("countback") != "4" || params.Get("to") == "" {
		t.Errorf("params = %v, want countback=4 anchored with an explicit to=", params)
	}
}

func TestEarningsPath_InvalidWindow(t *testing.T) {
	_, _, err := earningsPath("AAPL", []EarningsOption{WithEarningsWindow(LastN(-1))})
	if err == nil {
		t.Fatal("earningsPath() with an invalid window should return an error")
	}
}

func TestCandleParams_AllOptionsSet(t *testing.T) {
	options := defaultCandleOptions()
	WithCandleExtended(true).apply(options)
	WithCandleAdjustSplits(false).apply(options)
	WithCandleAdjustDividends(true).apply(options)

	p := candleParams(options, options.window)
	if p.Get("extended") != "true" || p.Get("adjustsplits") != "false" || p.Get("adjustdividends") != "true" {
		t.Errorf("params = %v, want extended=true&adjustsplits=false&adjustdividends=true", p)
	}
	// The resolution rides in the path, never the query: the API ignores the
	// query parameter (verified live).
	if _, ok := p["resolution"]; ok {
		t.Errorf("params = %v, want no resolution key", p)
	}
}

func TestEarningsPath_CarriesReport(t *testing.T) {
	// report used to be dropped here while Service.Earnings sent it.
	_, params, err := earningsPath("AAPL", []EarningsOption{WithEarningsReport("2024-Q1")})
	if err != nil {
		t.Fatalf("earningsPath() error = %v", err)
	}
	if params.Get("report") != "2024-Q1" {
		t.Errorf("params = %v, want report=2024-Q1", params)
	}
}

func TestCandlesPath_EmptySymbol(t *testing.T) {
	if _, _, err := candlesPath("", nil); err == nil {
		t.Fatal("candlesPath(\"\") should return an error")
	}
}

func TestCandlesPath_SingleChunk(t *testing.T) {
	path, chunkParams, err := candlesPath("AAPL", []CandleOption{
		WithResolution(ResolutionDaily),
		WithCandleWindow(Between(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC))),
	})
	if err != nil {
		t.Fatalf("candlesPath() error = %v", err)
	}
	if path != "stocks/candles/D/AAPL/" {
		t.Errorf("path = %q, want stocks/candles/D/AAPL/", path)
	}
	if len(chunkParams) != 1 {
		t.Fatalf("len(chunkParams) = %d, want 1 (daily resolution never splits)", len(chunkParams))
	}
}

func TestCandlesPath_MultiChunk(t *testing.T) {
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	path, chunkParams, err := candlesPath("AAPL", []CandleOption{
		WithResolution(Resolution1Min),
		WithCandleWindow(Between(from, to)),
	})
	if err != nil {
		t.Fatalf("candlesPath() error = %v", err)
	}
	if path != "stocks/candles/1/AAPL/" {
		t.Errorf("path = %q, want stocks/candles/1/AAPL/", path)
	}
	// 2020-01-01..2023-01-01 disjointly chunks into three full inclusive
	// years (2020, 2021, 2022) plus a fourth one-day leftover chunk for
	// 2023-01-01 itself.
	if len(chunkParams) != 4 {
		t.Fatalf("len(chunkParams) = %d, want 4 (>1yr intraday range split into disjoint year chunks)", len(chunkParams))
	}
	if chunkParams[0].Get("from") != "2020-01-01" || chunkParams[len(chunkParams)-1].Get("to") != "2023-01-01" {
		t.Errorf("chunk params = %v, want the first chunk to start at from and the last to end at to", chunkParams)
	}
}

func TestCandlesPath_InvalidWindow(t *testing.T) {
	_, _, err := candlesPath("AAPL", []CandleOption{WithCandleWindow(LastN(-1))})
	if err == nil {
		t.Fatal("candlesPath() with an invalid window should return an error")
	}
}

// --- htmlService: package-private, exercised only from within the package
// (see ADR-018 — the facet is built but not exported until the API
// supports format=html).

func testHTMLService(t *testing.T, handler http.HandlerFunc) *htmlService {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	httpClient := internalhttp.New(internalhttp.Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "test-key",
		RetryCfg: retry.DefaultConfig(),
	})
	return NewService(httpClient).asHTML()
}

func TestHTMLService_Quote(t *testing.T) {
	svc := testHTMLService(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("format"); got != "html" {
			t.Errorf("format param = %q, want html", got)
		}
		if got := r.URL.Path; got != "/v1/stocks/bulkquotes/" {
			t.Errorf("path = %q, want /v1/stocks/bulkquotes/", got)
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>AAPL</html>"))
	})

	got, err := svc.Quote(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if got.HTML() != "<html>AAPL</html>" {
		t.Errorf("HTML() = %q, want <html>AAPL</html>", got.HTML())
	}
}

func TestHTMLService_Quote_EmptySymbol(t *testing.T) {
	svc := &htmlService{}
	if _, err := svc.Quote(context.Background(), ""); err == nil {
		t.Fatal("Quote(\"\") should return a validation error")
	}
}

func TestHTMLService_Quotes(t *testing.T) {
	svc := testHTMLService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>quotes</html>"))
	})
	got, err := svc.Quotes(context.Background(), []string{"AAPL", "MSFT"})
	if err != nil {
		t.Fatalf("Quotes() error = %v", err)
	}
	if got.HTML() != "<html>quotes</html>" {
		t.Errorf("HTML() = %q", got.HTML())
	}
}

func TestHTMLService_Quotes_Empty(t *testing.T) {
	svc := &htmlService{}
	if _, err := svc.Quotes(context.Background(), nil); err == nil {
		t.Fatal("Quotes(nil) should return a validation error")
	}
}

func TestHTMLService_Prices(t *testing.T) {
	svc := testHTMLService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>prices</html>"))
	})
	got, err := svc.Prices(context.Background(), []string{"AAPL"})
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}
	if got.HTML() != "<html>prices</html>" {
		t.Errorf("HTML() = %q", got.HTML())
	}
}

func TestHTMLService_Prices_Empty(t *testing.T) {
	svc := &htmlService{}
	if _, err := svc.Prices(context.Background(), nil); err == nil {
		t.Fatal("Prices(nil) should return a validation error")
	}
}

func TestHTMLService_News(t *testing.T) {
	svc := testHTMLService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>news</html>"))
	})
	got, err := svc.News(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("News() error = %v", err)
	}
	if got.HTML() != "<html>news</html>" {
		t.Errorf("HTML() = %q", got.HTML())
	}
}

func TestHTMLService_News_EmptySymbol(t *testing.T) {
	svc := &htmlService{}
	if _, err := svc.News(context.Background(), ""); err == nil {
		t.Fatal("News(\"\") should return a validation error")
	}
}

func TestHTMLService_Earnings(t *testing.T) {
	svc := testHTMLService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>earnings</html>"))
	})
	got, err := svc.Earnings(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Earnings() error = %v", err)
	}
	if got.HTML() != "<html>earnings</html>" {
		t.Errorf("HTML() = %q", got.HTML())
	}
}

func TestHTMLService_Earnings_EmptySymbol(t *testing.T) {
	svc := &htmlService{}
	if _, err := svc.Earnings(context.Background(), ""); err == nil {
		t.Fatal("Earnings(\"\") should return a validation error")
	}
}

func TestHTMLService_Candles_SingleChunk(t *testing.T) {
	svc := testHTMLService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>candles</html>"))
	})
	got, err := svc.Candles(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
	if got.HTML() != "<html>candles</html>" {
		t.Errorf("HTML() = %q", got.HTML())
	}
}

func TestHTMLService_Candles_MultiChunk_Merges(t *testing.T) {
	svc := testHTMLService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		from := r.URL.Query().Get("from")
		_, _ = w.Write([]byte("<row>" + from + "</row>\n"))
	})
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	got, err := svc.Candles(context.Background(), "AAPL",
		WithResolution(Resolution1Min),
		WithCandleWindow(Between(from, to)),
	)
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
	if got.HTML() == "" {
		t.Error("HTML() should not be empty for a merged multi-chunk response")
	}
}

func TestHTMLService_Candles_EmptySymbol(t *testing.T) {
	svc := &htmlService{}
	if _, err := svc.Candles(context.Background(), ""); err == nil {
		t.Fatal("Candles(\"\") should return a validation error")
	}
}

func TestAsHTML_SharesHTTPClient(t *testing.T) {
	httpClient := internalhttp.New(internalhttp.Config{BaseURL: "http://example.test", APIVersion: "v1", Token: "k", RetryCfg: retry.DefaultConfig()})
	svc := NewService(httpClient)
	if svc.asHTML().http != httpClient {
		t.Error("asHTML() should carry the same http client as Service")
	}
}
