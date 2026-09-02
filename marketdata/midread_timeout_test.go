package marketdata_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
)

// TestMidReadTimeoutSetsTimeoutFlag pins that a timeout firing while the
// body is read produces a NetworkError whose Timeout field says so, with
// the StatusCode carried over from the interrupted response. The read
// branch used to hardcode Timeout=false on the premise that a timeout
// cannot reach the body read; a context deadline demonstrably interrupts
// io.ReadAll (this test), and so does http.Client.Timeout, which is how the
// SDK applies its own fixed timeout. The other two classification cases —
// a net.Error timeout without context.DeadlineExceeded, and a body that
// merely ended early — are covered by
// internal/http.TestMidReadFailureClassifiesTimeout.
func TestMidReadTimeoutSetsTimeoutFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(200)
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		flusher.Flush()
		_, _ = w.Write([]byte(`{"s":"ok"`)) // partial body
		flusher.Flush()
		time.Sleep(2 * time.Second) // stall past the caller's deadline
	}))
	defer srv.Close()

	client, err := marketdata.NewClient(
		marketdata.WithToken("test-token"),
		marketdata.WithBaseURL(srv.URL),
		marketdata.WithMaxRetries(0),
		marketdata.WithoutStartupValidation(),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	_, _, err = client.Stocks.Quote(ctx, "AAPL")
	var netErr *marketdata.NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("want NetworkError, got %T: %v", err, err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("test setup: failure was not deadline-caused: %v", err)
	}
	if !netErr.Timeout {
		t.Errorf("a deadline-caused mid-read failure carries Timeout=false; the documented classifier misreports the failure")
	}
}
