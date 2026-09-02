package response

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	internalhttp "github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/retry"
)

func testFormattedClient(t *testing.T, handler http.HandlerFunc) *internalhttp.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return internalhttp.New(internalhttp.Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "test-key",
		RetryCfg: retry.DefaultConfig(),
	})
}

func TestFetchFormatted_Success(t *testing.T) {
	client := testFormattedClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("symbol,last\nAAPL,150.22\n"))
	})

	got, err := FetchFormatted(context.Background(), client, "stocks/quotes/AAPL/", nil, "csv", NewCSV)
	if err != nil {
		t.Fatalf("FetchFormatted() error = %v", err)
	}
	if got.CSV() != "symbol,last\nAAPL,150.22\n" {
		t.Errorf("CSV() = %q, want the raw body", got.CSV())
	}
}

func TestFetchFormatted_Error(t *testing.T) {
	client := testFormattedClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(400)
		_, _ = w.Write([]byte("s,errmsg\nerror,\"bad\"\n"))
	})

	got, err := FetchFormatted(context.Background(), client, "stocks/quotes/AAPL/", nil, "csv", NewCSV)
	if err == nil {
		t.Fatal("FetchFormatted() should return an error")
	}
	if got != nil {
		t.Errorf("FetchFormatted() result = %v, want nil on error", got)
	}
}

func TestFetchFormattedChunked_SingleChunk(t *testing.T) {
	var hits atomic.Int32
	client := testFormattedClient(t, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("symbol,last\nAAPL,150.22\n"))
	})

	got, err := FetchFormattedChunked(context.Background(), client, "stocks/candles/D/AAPL/", []url.Values{{}}, "csv", NewCSV)
	if err != nil {
		t.Fatalf("FetchFormattedChunked() error = %v", err)
	}
	if hits.Load() != 1 {
		t.Errorf("hits = %d, want exactly 1 (single chunk should not fan out)", hits.Load())
	}
	if got.CSV() != "symbol,last\nAAPL,150.22\n" {
		t.Errorf("CSV() = %q, want the raw body", got.CSV())
	}
}

func TestFetchFormattedChunked_MultiChunk_MergesAndDropsRepeatedHeader(t *testing.T) {
	client := testFormattedClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		from := r.URL.Query().Get("from")
		switch from {
		case "2020-01-01":
			_, _ = w.Write([]byte("time,close\n2020-01-01,100\n"))
		case "2021-01-01":
			_, _ = w.Write([]byte("time,close\n2021-01-01,110\n"))
		default:
			t.Errorf("unexpected from param %q", from)
		}
	})

	chunks := []url.Values{
		{"from": []string{"2020-01-01"}},
		{"from": []string{"2021-01-01"}},
	}
	got, err := FetchFormattedChunked(context.Background(), client, "stocks/candles/D/AAPL/", chunks, "csv", NewCSV)
	if err != nil {
		t.Fatalf("FetchFormattedChunked() error = %v", err)
	}
	want := "time,close\n2020-01-01,100\n2021-01-01,110\n"
	if got.CSV() != want {
		t.Errorf("CSV() = %q, want %q", got.CSV(), want)
	}
}

func TestFetchFormattedChunked_FirstErrorCancelsSiblings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("from") == "2020-01-01" {
			w.Header().Set("Content-Type", "text/csv")
			w.WriteHeader(400)
			_, _ = w.Write([]byte("s,errmsg\nerror,\"boom\"\n"))
			return
		}
		// Blocks until canceled by the batch, or a bounded fallback fires —
		// so a missing cancellation fails the elapsed-time assertion below
		// quickly instead of hanging until the package-level test timeout.
		select {
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
		}
	}))
	defer server.Close()
	client := internalhttp.New(internalhttp.Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "test-key",
		RetryCfg: retry.DefaultConfig(),
	})

	chunks := []url.Values{
		{"from": []string{"2020-01-01"}},
		{"from": []string{"2021-01-01"}},
	}
	start := time.Now()
	_, err := FetchFormattedChunked(context.Background(), client, "stocks/candles/D/AAPL/", chunks, "csv", NewCSV)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("FetchFormattedChunked() should return the first chunk's error")
	}
	if errors.Is(err, context.Canceled) {
		t.Error("error should be the root failure, not a cancellation echo")
	}
	if elapsed > 1*time.Second {
		t.Errorf("elapsed = %v, want well under the 3s fallback (cancellation should have short-circuited it)", elapsed)
	}
}

func TestFetchFormattedMap_SingleKey(t *testing.T) {
	var hits atomic.Int32
	client := testFormattedClient(t, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("symbol,last\n" + r.URL.Path))
	})

	got, err := FetchFormattedMap(context.Background(), client, []string{"AAPL"},
		func(sym string) (string, url.Values, error) { return "options/quotes/" + sym + "/", nil, nil },
		"csv", NewCSV)
	if err != nil {
		t.Fatalf("FetchFormattedMap() error = %v", err)
	}
	if hits.Load() != 1 {
		t.Errorf("hits = %d, want exactly 1 (single key should not fan out)", hits.Load())
	}
	if len(got) != 1 || got["AAPL"] == nil {
		t.Fatalf("result = %v, want a single AAPL entry", got)
	}
}

func TestFetchFormattedMap_SingleKey_Error(t *testing.T) {
	client := testFormattedClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(400)
		_, _ = w.Write([]byte("s,errmsg\nerror,\"boom\"\n"))
	})

	got, err := FetchFormattedMap(context.Background(), client, []string{"AAPL"},
		func(sym string) (string, url.Values, error) { return "options/quotes/" + sym + "/", nil, nil },
		"csv", NewCSV)
	if err == nil {
		t.Fatal("FetchFormattedMap() should return an error")
	}
	if got != nil {
		t.Errorf("result = %v, want nil on error", got)
	}
}

func TestFetchFormattedMap_MultiKey(t *testing.T) {
	client := testFormattedClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte("symbol\n" + r.URL.Query().Get("symbol")))
	})

	keys := []string{"AAPL", "MSFT", "GOOG"}
	got, err := FetchFormattedMap(context.Background(), client, keys,
		func(sym string) (string, url.Values, error) {
			return "options/quotes/" + sym + "/", url.Values{"symbol": []string{sym}}, nil
		},
		"csv", NewCSV)
	if err != nil {
		t.Fatalf("FetchFormattedMap() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(result) = %d, want 3", len(got))
	}
	for _, k := range keys {
		if got[k] == nil {
			t.Errorf("result[%q] = nil, want an entry", k)
			continue
		}
		if want := "symbol\n" + k; got[k].CSV() != want {
			t.Errorf("result[%q].CSV() = %q, want %q", k, got[k].CSV(), want)
		}
	}
}

func TestFetchFormattedMap_FirstErrorCancelsSiblings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("symbol") == "BAD" {
			w.Header().Set("Content-Type", "text/csv")
			w.WriteHeader(400)
			_, _ = w.Write([]byte("s,errmsg\nerror,\"boom\"\n"))
			return
		}
		select {
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
		}
	}))
	defer server.Close()
	client := internalhttp.New(internalhttp.Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "test-key",
		RetryCfg: retry.DefaultConfig(),
	})

	keys := []string{"BAD", "GOOD"}
	start := time.Now()
	_, err := FetchFormattedMap(context.Background(), client, keys,
		func(sym string) (string, url.Values, error) {
			return "options/quotes/" + sym + "/", url.Values{"symbol": []string{sym}}, nil
		},
		"csv", NewCSV)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("FetchFormattedMap() should return the failing key's error")
	}
	if errors.Is(err, context.Canceled) {
		t.Error("error should be the root failure, not a cancellation echo")
	}
	if elapsed > 1*time.Second {
		t.Errorf("elapsed = %v, want well under the 3s fallback (cancellation should have short-circuited it)", elapsed)
	}
}

func TestMergeTextChunks(t *testing.T) {
	cases := []struct {
		name   string
		bodies [][]byte
		want   string
	}{
		{"empty", nil, ""},
		{"single", [][]byte{[]byte("a,b\n1,2\n")}, "a,b\n1,2\n"},
		{
			"dedup repeated header",
			[][]byte{[]byte("a,b\n1,2\n"), []byte("a,b\n3,4\n")},
			"a,b\n1,2\n3,4\n",
		},
		{
			"no trailing newline on first chunk still joins cleanly",
			[][]byte{[]byte("a,b\n1,2"), []byte("a,b\n3,4\n")},
			"a,b\n1,2\n3,4\n",
		},
		{
			"chunk body empty after header strip is skipped",
			[][]byte{[]byte("a,b\n1,2\n"), []byte("a,b\n")},
			"a,b\n1,2\n",
		},
		{
			"differing header not treated as repeat",
			[][]byte{[]byte("a,b\n1,2\n"), []byte("x,y\n3,4\n")},
			"a,b\n1,2\nx,y\n3,4\n",
		},
		{
			"single chunk with no newline at all",
			[][]byte{[]byte("no data")},
			"no data",
		},
		{
			"second chunk with no newline at all is appended whole (no header to strip)",
			[][]byte{[]byte("a,b\n1,2\n"), []byte("no newline here")},
			"a,b\n1,2\nno newline here",
		},
		{
			"second chunk is exactly the repeated header with no trailing newline",
			[][]byte{[]byte("a,b\n1,2\n"), []byte("a,b")},
			"a,b\n1,2\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := string(mergeTextChunks(tc.bodies))
			if got != tc.want {
				t.Errorf("mergeTextChunks() = %q, want %q", got, tc.want)
			}
		})
	}
}

// liveNoDataCSV aliases the canonical no-data shapes (see NoDataBody in
// formatted.go, which the live wire-shape witness re-fetches). The CRLF
// line endings are the API's own and are part of what the detection has to
// survive.
const (
	liveNoDataCSV           = NoDataBody
	liveNoDataCSVNoHeaders  = NoDataBodyNoHeaders
	liveDataCSV             = "t,o,h,l,c,v\r\n1786420800,307.75,309.97,302.79,304.91,37476746\r\n"
	liveDataCSVNoHeadersRow = "1786420800,307.75,309.97,302.79,304.91,37476746\r\n"
)

// TestFetchFormattedChunked_DropsLeadingNoDataChunk is the case that
// motivated the filter: the OLDEST chunk is the one most likely to predate
// the listing, and mergeTextChunks takes its reference header from the
// first body — so before the fix the header became "0" and every later
// chunk kept its own header row inline, inside a successful 200 response
// with a nil error.
func TestFetchFormattedChunked_DropsLeadingNoDataChunk(t *testing.T) {
	client := testFormattedClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		switch r.URL.Query().Get("from") {
		case "2015-01-01":
			_, _ = w.Write([]byte(liveNoDataCSV))
		case "2020-01-01":
			_, _ = w.Write([]byte("t,o,h,l,c,v\r\n1577836800,100,105,95,102,1000\r\n"))
		case "2021-01-01":
			_, _ = w.Write([]byte("t,o,h,l,c,v\r\n1609459200,110,115,105,112,2000\r\n"))
		default:
			t.Errorf("unexpected from param %q", r.URL.Query().Get("from"))
		}
	})

	chunks := []url.Values{
		{"from": []string{"2015-01-01"}},
		{"from": []string{"2020-01-01"}},
		{"from": []string{"2021-01-01"}},
	}
	got, err := FetchFormattedChunked(context.Background(), client, "stocks/candles/60/RIVN/", chunks, "csv", NewCSV)
	if err != nil {
		t.Fatalf("FetchFormattedChunked() error = %v", err)
	}

	want := "t,o,h,l,c,v\r\n1577836800,100,105,95,102,1000\r\n1609459200,110,115,105,112,2000\r\n"
	if got.CSV() != want {
		t.Errorf("CSV() = %q, want %q", got.CSV(), want)
	}
}

// TestFetchFormattedChunked_DropsNoDataChunkWithoutHeaders covers the other
// live shape: with headers=false the no-data body is a bare `""` line, with
// no "0" placeholder before it.
func TestFetchFormattedChunked_DropsNoDataChunkWithoutHeaders(t *testing.T) {
	client := testFormattedClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		if r.URL.Query().Get("from") == "2015-01-01" {
			_, _ = w.Write([]byte(liveNoDataCSVNoHeaders))
			return
		}
		_, _ = w.Write([]byte(liveDataCSVNoHeadersRow))
	})

	chunks := []url.Values{
		{"from": []string{"2015-01-01"}},
		{"from": []string{"2026-01-01"}},
	}
	got, err := FetchFormattedChunked(context.Background(), client, "stocks/candles/60/RIVN/", chunks, "csv", NewCSV)
	if err != nil {
		t.Fatalf("FetchFormattedChunked() error = %v", err)
	}
	if got.CSV() != liveDataCSVNoHeadersRow {
		t.Errorf("CSV() = %q, want %q", got.CSV(), liveDataCSVNoHeadersRow)
	}
}

// TestFetchFormattedChunked_AllChunksNoData verifies the degenerate case
// hands back the API's own no-data body rather than an empty string: these
// formats have no NoData concept (ADR-018), so the caller has to be able to
// see what the API actually said.
func TestFetchFormattedChunked_AllChunksNoData(t *testing.T) {
	client := testFormattedClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte(liveNoDataCSV))
	})

	chunks := []url.Values{
		{"from": []string{"2015-01-01"}},
		{"from": []string{"2016-01-01"}},
	}
	got, err := FetchFormattedChunked(context.Background(), client, "stocks/candles/60/RIVN/", chunks, "csv", NewCSV)
	if err != nil {
		t.Fatalf("FetchFormattedChunked() error = %v", err)
	}
	if got.CSV() != liveNoDataCSV {
		t.Errorf("CSV() = %q, want the API's own no-data body %q", got.CSV(), liveNoDataCSV)
	}
}

func TestNoDataChunk(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{"live shape, headers on", liveNoDataCSV, true},
		{"live shape, headers off", liveNoDataCSVNoHeaders, true},
		{"no trailing newline", "0\r\n\"\"", true},
		{"unix line endings", "0\n\"\"\n", true},
		{"marker alone without placeholder", "\"\"\n", true},
		{"data with headers", liveDataCSV, false},
		{"data without headers", liveDataCSVNoHeadersRow, false},
		{"placeholder alone is not enough", "0\r\n", false},
		{"empty body", "", false},
		{"header row only", "t,o,h,l,c,v\r\n", false},
		{"a row whose first field is 0", "0,1,2\r\n", false},
		{"a row of empty fields is data, not the marker", ",,,,,\r\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := noDataChunk([]byte(tt.body)); got != tt.want {
				t.Errorf("noDataChunk(%q) = %v, want %v", tt.body, got, tt.want)
			}
		})
	}
}

// TestFetchFormattedMap_RequestForErrorSingleKey and its multi-key sibling
// cover requestFor rejecting a key. The callback can fail because the
// facets build their request through the same validating path builder the
// JSON methods use, so an unusable option must surface as an error instead
// of being sent to the API.
func TestFetchFormattedMap_RequestForErrorSingleKey(t *testing.T) {
	var hits atomic.Int32
	client := testFormattedClient(t, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("ok"))
	})

	boom := errors.New("invalid window")
	_, err := FetchFormattedMap(context.Background(), client, []string{"AAPL"},
		func(sym string) (string, url.Values, error) { return "", nil, boom },
		"csv", NewCSV)
	if !errors.Is(err, boom) {
		t.Fatalf("FetchFormattedMap() error = %v, want the builder's error", err)
	}
	if hits.Load() != 0 {
		t.Errorf("made %d requests, want 0 — a rejected key must not reach the API", hits.Load())
	}
}

func TestFetchFormattedMap_RequestForErrorMultiKey(t *testing.T) {
	var hits atomic.Int32
	client := testFormattedClient(t, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("ok"))
	})

	boom := errors.New("invalid window")
	_, err := FetchFormattedMap(context.Background(), client, []string{"AAPL", "MSFT", "GOOG"},
		func(sym string) (string, url.Values, error) { return "", nil, boom },
		"csv", NewCSV)
	if !errors.Is(err, boom) {
		t.Fatalf("FetchFormattedMap() error = %v, want the builder's error", err)
	}
	if hits.Load() != 0 {
		t.Errorf("made %d requests, want 0 — a rejected key must not reach the API", hits.Load())
	}
}
