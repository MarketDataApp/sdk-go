package markets

import (
	"context"
	"net/http"
	"testing"
	"time"

	internalhttp "github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/retry"
)

func TestStatusPath_NoOptions(t *testing.T) {
	path, params := statusPath(nil)
	if path != "markets/status/" {
		t.Errorf("path = %q, want markets/status/", path)
	}
	if params.Get("date") != "" || params.Get("country") != "" {
		t.Errorf("params = %v, want empty", params)
	}
}

func TestStatusPath_WithOptions(t *testing.T) {
	path, params := statusPath([]StatusOption{
		WithDate(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)),
		WithCountry("CA"),
	})
	if path != "markets/status/" {
		t.Errorf("path = %q, want markets/status/", path)
	}
	if params.Get("date") != "2024-06-01" || params.Get("country") != "CA" {
		t.Errorf("params = %v, want date=2024-06-01&country=CA", params)
	}
}

func TestStatusHistoryPath_WithOptions(t *testing.T) {
	path, params, err := statusHistoryPath([]HistoryOption{
		WithHistoryWindow(Between(
			time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 6, 7, 0, 0, 0, 0, time.UTC),
		)),
		WithCountry("CA"),
	})
	if err != nil {
		t.Fatalf("statusHistoryPath() error = %v", err)
	}
	if path != "markets/status/" {
		t.Errorf("path = %q, want markets/status/", path)
	}
	if params.Get("from") != "2024-06-01" || params.Get("to") != "2024-06-07" {
		t.Errorf("params = %v, want from=2024-06-01&to=2024-06-07", params)
	}
	if params.Get("country") != "CA" {
		t.Errorf("params = %v, want country=CA", params)
	}
}

func TestStatusHistoryPath_InvalidWindow(t *testing.T) {
	// Validation belongs to the builder, so an unusable window is rejected
	// before the wire regardless of which caller reaches it.
	if _, _, err := statusHistoryPath([]HistoryOption{WithHistoryWindow(LastN(-1))}); err == nil {
		t.Fatal("statusHistoryPath() with an invalid window should return an error")
	}
}

// TestMarketsBuildersAgreeOnCountry pins the property that motivated pulling
// StatusHistory into a builder: the two methods share one wire path and one
// country parameter, and it used to be serialized in two places. Nothing had
// drifted — but this is the shape that has produced real bugs here, where a
// parameter was sent by one serializer and silently dropped by its twin.
func TestMarketsBuildersAgreeOnCountry(t *testing.T) {
	for _, country := range []string{"US", "CA", ""} {
		_, statusParams := statusPath([]StatusOption{WithCountry(country)})
		_, historyParams, err := statusHistoryPath([]HistoryOption{
			WithHistoryWindow(LastN(3)),
			WithCountry(country),
		})
		if err != nil {
			t.Fatalf("statusHistoryPath(%q) error = %v", country, err)
		}
		if got, want := historyParams.Get("country"), statusParams.Get("country"); got != want {
			t.Errorf("country %q: history serialized %q, status serialized %q", country, got, want)
		}
		if country == "" && statusParams.Has("country") {
			t.Errorf("empty country should leave the parameter unset, got %v", statusParams)
		}
	}
}

func TestCSVService_Status(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("format"); got != "csv" {
			t.Errorf("format param = %q, want csv", got)
		}
		if got := r.Header.Get("Accept"); got != "text/csv" {
			t.Errorf("Accept header = %q, want text/csv", got)
		}
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("date,status\n2024-06-01,open\n"))
	})).AsCSV()

	got, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if got.CSV() != "date,status\n2024-06-01,open\n" {
		t.Errorf("CSV() = %q", got.CSV())
	}
}

func TestCSVService_Status_Error(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(400)
		_, _ = w.Write([]byte("s,errmsg\nerror,\"bad country code\"\n"))
	})).AsCSV()

	_, err := svc.Status(context.Background())
	if err == nil {
		t.Fatal("Status() should return an error for a 400 response")
	}
	if got := err.Error(); got != "marketdata: bad request: bad country code" {
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

func TestHTMLService_Status(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("format"); got != "html" {
			t.Errorf("format param = %q, want html", got)
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>status</html>"))
	})).asHTML()

	got, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if got.HTML() != "<html>status</html>" {
		t.Errorf("HTML() = %q", got.HTML())
	}
}

func TestAsHTML_SharesHTTPClient(t *testing.T) {
	httpClient := internalhttp.New(internalhttp.Config{BaseURL: "http://example.test", APIVersion: "v1", Token: "k", RetryCfg: retry.DefaultConfig()})
	svc := NewService(httpClient)
	if svc.asHTML().http != httpClient {
		t.Error("asHTML() should carry the same http client as Service")
	}
}
