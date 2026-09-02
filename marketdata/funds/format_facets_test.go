package funds

import (
	"context"
	"net/http"
	"testing"
	"time"

	internalhttp "github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/retry"
)

func TestCandlesPath_EmptySymbol(t *testing.T) {
	if _, _, err := candlesPath("", nil); err == nil {
		t.Fatal("candlesPath(\"\") should return an error")
	}
}

func TestCandlesPath_InvalidWindow(t *testing.T) {
	_, _, err := candlesPath("VFINX", []CandleOption{WithCandleWindow(LastN(-1))})
	if err == nil {
		t.Fatal("candlesPath() with an invalid window should return an error")
	}
}

func TestCandlesPath_Params(t *testing.T) {
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	path, params, err := candlesPath("VFINX", []CandleOption{
		WithResolution(ResolutionDaily),
		WithCandleWindow(Between(from, to)),
	})
	if err != nil {
		t.Fatalf("candlesPath() error = %v", err)
	}
	if path != "funds/candles/D/VFINX/" {
		t.Errorf("path = %q, want funds/candles/D/VFINX/", path)
	}
	if params.Get("from") != "2024-01-01" || params.Get("to") != "2024-02-01" {
		t.Errorf("params = %v, want from=2024-01-01&to=2024-02-01", params)
	}
}

func TestCSVService_Candles(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("format"); got != "csv" {
			t.Errorf("format param = %q, want csv", got)
		}
		if got := r.Header.Get("Accept"); got != "text/csv" {
			t.Errorf("Accept header = %q, want text/csv", got)
		}
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("time,close\n2024-01-01,15.2\n"))
	})).AsCSV()

	got, err := svc.Candles(context.Background(), "VFINX")
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
	if got.CSV() != "time,close\n2024-01-01,15.2\n" {
		t.Errorf("CSV() = %q", got.CSV())
	}
}

func TestCSVService_Candles_EmptySymbol(t *testing.T) {
	svc := (&Service{}).AsCSV()
	if _, err := svc.Candles(context.Background(), ""); err == nil {
		t.Fatal("Candles(\"\") should return a validation error without making a request")
	}
}

func TestCSVService_Candles_Error(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(400)
		_, _ = w.Write([]byte("s,errmsg\nerror,\"bad symbol\"\n"))
	})).AsCSV()

	_, err := svc.Candles(context.Background(), "VFINX")
	if err == nil {
		t.Fatal("Candles() should return an error for a 400 response")
	}
	if got := err.Error(); got != "marketdata: bad request: bad symbol" {
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

func TestHTMLService_Candles(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("format"); got != "html" {
			t.Errorf("format param = %q, want html", got)
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>candles</html>"))
	})).asHTML()

	got, err := svc.Candles(context.Background(), "VFINX")
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
	if got.HTML() != "<html>candles</html>" {
		t.Errorf("HTML() = %q", got.HTML())
	}
}

func TestHTMLService_Candles_EmptySymbol(t *testing.T) {
	svc := (&Service{}).asHTML()
	if _, err := svc.Candles(context.Background(), ""); err == nil {
		t.Fatal("Candles(\"\") should return a validation error without making a request")
	}
}

func TestAsHTML_SharesHTTPClient(t *testing.T) {
	httpClient := internalhttp.New(internalhttp.Config{BaseURL: "http://example.test", APIVersion: "v1", Token: "k", RetryCfg: retry.DefaultConfig()})
	svc := NewService(httpClient)
	if svc.asHTML().http != httpClient {
		t.Error("asHTML() should carry the same http client as Service")
	}
}
