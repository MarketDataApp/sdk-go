//go:build integration

package integration

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/response"
)

// TestWireShape_FormattedNoData is the live witness for the formatted
// no-data body shapes. The chunk-merge filter (noDataChunk) recognizes the
// two shapes response.NoDataBody documents, and it deliberately fails open:
// if production ever changes the shape, the filter silently stops matching
// and the merge corruption it exists to prevent returns with no signal.
// This test closes that hole the same way wire_shape_test.go does for the
// JSON shapes: re-fetch the exact case the shapes were captured from and
// fail on any drift. The unit fixtures consume the same constants, so
// live == constant here plus constant == recognized there covers the whole
// chain without exporting the filter.
func TestWireShape_FormattedNoData(t *testing.T) {
	const preListing = "/v1/stocks/candles/D/RIVN/?from=2015-01-01&to=2015-12-31&format=csv"

	for _, tc := range []struct {
		name string
		path string
		want string
	}{
		{"headers on", preListing, response.NoDataBody},
		{"headers off", preListing + "&headers=false", response.NoDataBodyNoHeaders},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, status := fetchRawBody(t, tc.path)
			if status != 200 {
				t.Fatalf("status %d, want 200 — the formatted no-data case is a plain 200", status)
			}
			if string(body) != tc.want {
				t.Errorf("no-data body changed: got %q, want %q — noDataChunk fails open, so without this failure the chunk-merge corruption would return silently", body, tc.want)
			}
		})
	}
}

// fetchRawBody fetches a path's raw response body, bypassing the SDK.
func fetchRawBody(t *testing.T, path string) ([]byte, int) {
	t.Helper()
	token := wireToken(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wireBaseURL()+path, nil)
	if err != nil {
		t.Fatalf("build request for %s: %v", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request %s: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		t.Fatalf("read body for %s: %v", path, err)
	}
	return body, resp.StatusCode
}
