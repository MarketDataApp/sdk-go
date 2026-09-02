package stocks_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

// bodyServer answers every request with one fixed status and body.
func bodyServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(s.Close)
	return s
}

// A typo'd or delisted ticker is a caller error the API discriminates with
// errmsg. It used to come back as a nil quote and a nil error, i.e. as a
// successful "this symbol has no data" — a silently wrong answer.
func TestQuote_NonexistentSymbolIsNotFoundError(t *testing.T) {
	// The API's real body for stocks/quotes/ZZZZQQ/, verified live 2026-09-01.
	server := bodyServer(t, http.StatusNotFound, `{"s":"no_data","errmsg":"Symbol not found."}`)
	svc := newTestService(server.URL)

	quote, _, err := svc.Quote(context.Background(), "ZZZZQQ")
	if err == nil {
		t.Fatalf("Quote() error = nil, want NotFoundError (got quote %+v)", quote)
	}
	var nf *sdkerrors.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("Quote() error = %T (%v), want *sdkerrors.NotFoundError", err, err)
	}
	if nf.Message != "Symbol not found." {
		t.Errorf("Message = %q, want the API's own errmsg", nf.Message)
	}
}

// stocks/candles answers a nonexistent symbol with a markerless 404
// (verified live 2026-09-01) — the API gives nothing to tell it apart from
// an empty window, so it stays no-data rather than the SDK inventing a
// distinction the wire does not make.
func TestCandles_MarkerlessNotFoundStaysNoData(t *testing.T) {
	server := bodyServer(t, http.StatusNotFound, `{"s":"no_data","prevTime":null,"nextTime":null}`)
	svc := newTestService(server.URL)

	candles, resp, err := svc.Candles(context.Background(), "ZZZZQQ")
	if err != nil {
		t.Fatalf("Candles() error = %v, want nil for a markerless 404", err)
	}
	if len(candles) != 0 {
		t.Errorf("Candles() = %d candles, want 0", len(candles))
	}
	if resp == nil || !resp.NoData {
		t.Errorf("response = %v, want NoData true", resp)
	}
}

// Earnings and News do carry the marker (verified live), so they fail loudly.
func TestEarningsAndNews_NonexistentSymbolIsNotFoundError(t *testing.T) {
	server := bodyServer(t, http.StatusNotFound, `{"s":"no_data","errmsg":"Symbol not found."}`)
	svc := newTestService(server.URL)

	if _, _, err := svc.Earnings(context.Background(), "ZZZZQQ"); !errors.Is(err, sdkerrors.ErrNotFound) {
		t.Errorf("Earnings() error = %v, want ErrNotFound", err)
	}
	if _, _, err := svc.News(context.Background(), "ZZZZQQ"); !errors.Is(err, sdkerrors.ErrNotFound) {
		t.Errorf("News() error = %v, want ErrNotFound", err)
	}
}
