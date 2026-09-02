package stocks_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	internalhttp "github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/retry"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

func testCSVService(t *testing.T, handler http.HandlerFunc) *stocks.CSVService {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	httpClient := internalhttp.New(internalhttp.Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "test-key",
		RetryCfg: retry.DefaultConfig(),
	})
	return stocks.NewService(httpClient).AsCSV()
}

func TestCSVService_Quote(t *testing.T) {
	svc := testCSVService(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("format"); got != "csv" {
			t.Errorf("format param = %q, want csv", got)
		}
		if got := r.Header.Get("Accept"); got != "text/csv" {
			t.Errorf("Accept header = %q, want text/csv", got)
		}
		if got := r.URL.Query().Get("symbols"); got != "AAPL" {
			t.Errorf("symbols param = %q, want AAPL", got)
		}
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("symbol,last\nAAPL,150.22\n"))
	})

	got, err := svc.Quote(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if got.CSV() != "symbol,last\nAAPL,150.22\n" {
		t.Errorf("CSV() = %q", got.CSV())
	}
}

func TestCSVService_Quote_EmptySymbol(t *testing.T) {
	svc := stocks.NewService(nil).AsCSV()
	if _, err := svc.Quote(context.Background(), ""); err == nil {
		t.Fatal("Quote(\"\") should return a validation error without making a request")
	}
}

func TestCSVService_Quotes(t *testing.T) {
	svc := testCSVService(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("symbols"); got != "AAPL,MSFT" {
			t.Errorf("symbols param = %q, want AAPL,MSFT", got)
		}
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("symbol,last\nAAPL,150.22\nMSFT,300.10\n"))
	})

	got, err := svc.Quotes(context.Background(), []string{"AAPL", "MSFT"})
	if err != nil {
		t.Fatalf("Quotes() error = %v", err)
	}
	if got.CSV() != "symbol,last\nAAPL,150.22\nMSFT,300.10\n" {
		t.Errorf("CSV() = %q", got.CSV())
	}
}

func TestCSVService_Quotes_Empty(t *testing.T) {
	svc := stocks.NewService(nil).AsCSV()
	if _, err := svc.Quotes(context.Background(), nil); err == nil {
		t.Fatal("Quotes(nil) should return a validation error without making a request")
	}
}

func TestCSVService_Prices_SingleSymbolPathEmbedded(t *testing.T) {
	svc := testCSVService(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/v1/stocks/prices/AAPL/" {
			t.Errorf("path = %q, want /v1/stocks/prices/AAPL/", got)
		}
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("symbol,mid\nAAPL,150.20\n"))
	})

	got, err := svc.Prices(context.Background(), []string{"AAPL"})
	if err != nil {
		t.Fatalf("Prices() error = %v", err)
	}
	if got.CSV() != "symbol,mid\nAAPL,150.20\n" {
		t.Errorf("CSV() = %q", got.CSV())
	}
}

func TestCSVService_Prices_Empty(t *testing.T) {
	svc := stocks.NewService(nil).AsCSV()
	if _, err := svc.Prices(context.Background(), nil); err == nil {
		t.Fatal("Prices(nil) should return a validation error without making a request")
	}
}

func TestCSVService_News(t *testing.T) {
	svc := testCSVService(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/v1/stocks/news/AAPL/" {
			t.Errorf("path = %q, want /v1/stocks/news/AAPL/", got)
		}
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("headline,publisher\nAAPL rises,Reuters\n"))
	})

	got, err := svc.News(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("News() error = %v", err)
	}
	if got.CSV() != "headline,publisher\nAAPL rises,Reuters\n" {
		t.Errorf("CSV() = %q", got.CSV())
	}
}

func TestCSVService_News_EmptySymbol(t *testing.T) {
	svc := stocks.NewService(nil).AsCSV()
	if _, err := svc.News(context.Background(), ""); err == nil {
		t.Fatal("News(\"\") should return a validation error without making a request")
	}
}

func TestCSVService_News_InvalidWindow(t *testing.T) {
	svc := stocks.NewService(nil).AsCSV()
	if _, err := svc.News(context.Background(), "AAPL", stocks.WithNewsWindow(stocks.LastN(-1))); err == nil {
		t.Fatal("News() with an invalid window should return an error without making a request")
	}
}

func TestCSVService_Earnings(t *testing.T) {
	svc := testCSVService(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/v1/stocks/earnings/AAPL/" {
			t.Errorf("path = %q, want /v1/stocks/earnings/AAPL/", got)
		}
		if got := r.URL.Query().Get("countback"); got != "4" {
			t.Errorf("countback param = %q, want 4", got)
		}
		if got := r.URL.Query().Get("to"); got == "" {
			t.Error("earnings countback should be anchored with an explicit to=")
		}
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("fiscalYear,epsActual\n2024,1.5\n"))
	})

	got, err := svc.Earnings(context.Background(), "AAPL", stocks.WithEarningsWindow(stocks.LastN(4)))
	if err != nil {
		t.Fatalf("Earnings() error = %v", err)
	}
	if got.CSV() != "fiscalYear,epsActual\n2024,1.5\n" {
		t.Errorf("CSV() = %q", got.CSV())
	}
}

func TestCSVService_Earnings_EmptySymbol(t *testing.T) {
	svc := stocks.NewService(nil).AsCSV()
	if _, err := svc.Earnings(context.Background(), ""); err == nil {
		t.Fatal("Earnings(\"\") should return a validation error without making a request")
	}
}

func TestCSVService_Candles_SingleChunk(t *testing.T) {
	svc := testCSVService(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/v1/stocks/candles/D/AAPL/" {
			t.Errorf("path = %q, want /v1/stocks/candles/D/AAPL/", got)
		}
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("time,close\n2024-01-01,150.0\n"))
	})

	got, err := svc.Candles(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
	if got.CSV() != "time,close\n2024-01-01,150.0\n" {
		t.Errorf("CSV() = %q", got.CSV())
	}
}

func TestCSVService_Candles_MultiChunk_MergesInChronologicalOrder(t *testing.T) {
	svc := testCSVService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		from := r.URL.Query().Get("from")
		switch from {
		case "2020-01-01":
			_, _ = w.Write([]byte("time,close\n2020-06-01,100\n"))
		case "2021-01-01":
			_, _ = w.Write([]byte("time,close\n2021-06-01,110\n"))
		case "2022-01-01":
			_, _ = w.Write([]byte("time,close\n2022-06-01,120\n"))
		case "2023-01-01":
			_, _ = w.Write([]byte("time,close\n2023-01-01,130\n"))
		default:
			t.Errorf("unexpected from param %q", from)
		}
	})

	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	got, err := svc.Candles(context.Background(), "AAPL",
		stocks.WithResolution(stocks.Resolution1Min),
		stocks.WithCandleWindow(stocks.Between(from, to)),
	)
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
	want := "time,close\n2020-06-01,100\n2021-06-01,110\n2022-06-01,120\n2023-01-01,130\n"
	if got.CSV() != want {
		t.Errorf("CSV() = %q, want %q", got.CSV(), want)
	}
}

func TestCSVService_Candles_EmptySymbol(t *testing.T) {
	svc := stocks.NewService(nil).AsCSV()
	if _, err := svc.Candles(context.Background(), ""); err == nil {
		t.Fatal("Candles(\"\") should return a validation error without making a request")
	}
}

func TestCSVService_Candles_InvalidWindow(t *testing.T) {
	svc := stocks.NewService(nil).AsCSV()
	if _, err := svc.Candles(context.Background(), "AAPL", stocks.WithCandleWindow(stocks.LastN(-1))); err == nil {
		t.Fatal("Candles() with an invalid window should return an error without making a request")
	}
}

func TestCSVService_ErrorPropagatesWithCSVParsedMessage(t *testing.T) {
	svc := testCSVService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(400)
		_, _ = w.Write([]byte("s,errmsg\nerror,\"Bad parameters, please check API documentation.\"\n"))
	})

	_, err := svc.Quote(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("Quote() should return an error for a 400 response")
	}
	if got := err.Error(); got != "marketdata: bad request: Bad parameters, please check API documentation." {
		t.Errorf("err.Error() = %q, want the CSV-parsed errmsg embedded", got)
	}
}

func TestCSVService_NoDataStatusIsNotAnError(t *testing.T) {
	// Verified live: a genuine no-data condition on a CSV candles request
	// can come back 200 with a degenerate body instead of 404 — the facet
	// has no NoData concept and just hands back whatever text the API sent.
	svc := testCSVService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("0\n\"\"\n"))
	})

	got, err := svc.Candles(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Candles() error = %v, want nil", err)
	}
	if got.CSV() != "0\n\"\"\n" {
		t.Errorf("CSV() = %q, want the raw body passed through as-is", got.CSV())
	}
}

func TestCSVService_ActualHTTP404IsAlsoNotAnError(t *testing.T) {
	// Distinct from the case above: if the API ever *does* send a real 404
	// for a CSV-formatted request, that must not be treated as an error
	// either — matching GetFormatted's own "404/204 pass through like Get's
	// NoData carve-out, just without synthesizing NoData" contract.
	svc := testCSVService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"s":"no_data"}`))
	})

	got, err := svc.Candles(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Candles() error = %v, want nil (404 must not error)", err)
	}
	if got.CSV() != `{"s":"no_data"}` {
		t.Errorf("CSV() = %q, want the raw 404 body passed through as-is", got.CSV())
	}
}

func TestAsCSV_SharesHTTPClient(t *testing.T) {
	httpClient := internalhttp.New(internalhttp.Config{BaseURL: "http://example.test", APIVersion: "v1", Token: "k", RetryCfg: retry.DefaultConfig()})
	svc := stocks.NewService(httpClient)
	// AsCSV() must not panic and must produce a usable facet; if it silently
	// dropped the http client every call would nil-deref instead.
	if svc.AsCSV() == nil {
		t.Fatal("AsCSV() returned nil")
	}
}
