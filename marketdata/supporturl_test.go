package marketdata_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
)

// TestStatusErrorSupportBlockCarriesQuery pins, end to end through a
// service call, that a StatusError's support block reports the URL that was
// actually sent — query included, with the client's merged universal
// defaults in it. SupportContext used to report the path alone, so the
// support block for a 200-with-error-body omitted every query parameter,
// including the symbol itself on query-addressed endpoints like bulkquotes.
// The unit half of this contract lives in internal/http
// (TestResponse_SupportContext); this covers the merge of client-level
// defaults, which no unit test sees.
func TestStatusErrorSupportBlockCarriesQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"s":"error","errmsg":"simulated body-level failure"}`)
	}))
	defer srv.Close()

	client, err := marketdata.NewClient(
		marketdata.WithToken("test-token"),
		marketdata.WithBaseURL(srv.URL),
		marketdata.WithDateFormat("spreadsheet"),
		marketdata.WithLimit(5),
		marketdata.WithoutStartupValidation(),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = client.Stocks.Quote(context.Background(), "AAPL")
	var apiErr *marketdata.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want APIError, got %v", err)
	}

	info := apiErr.SupportInfo()
	for _, want := range []string{"symbols=AAPL", "dateformat=spreadsheet", "limit=5"} {
		if !strings.Contains(info, want) {
			t.Errorf("support block omits %q — request_url is stripped of the query that was sent:\n%s", want, info)
		}
	}
}
