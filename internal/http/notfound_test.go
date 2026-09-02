package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MarketDataApp/sdk-go/v2/internal/ratelimit"
	"github.com/MarketDataApp/sdk-go/v2/internal/retry"
	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

// clientFor builds a client pointed at a server that answers every request
// with the given status, content type and body.
func clientFor(t *testing.T, status int, contentType, body string) (*Client, func()) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	c := New(Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})
	return c, server.Close
}

// The API answers two different things with 404 — "your question was
// invalid" and "your valid question has an empty answer" — and separates
// them only by errmsg. These bodies are the ones production actually sends,
// captured live on 2026-09-01; see notFoundNamesAnError.
const (
	wire404SymbolNotFound = `{"s":"no_data","errmsg":"Symbol not found."}`
	wire404NoOptionFound  = `{"s":"error","errmsg":"No option found. No option was found for this strike and expiration."}`
	wire404EmptyAnswer    = `{"s":"no_data","nextTime":null,"prevTime":null}`
	wire404SymbolCSV      = "s,errmsg\r\nno_data,\"Symbol not found.\"\r\n"
	wire404EmptyCSV       = "0\r\n\"\"\r\n"
)

func TestGet_NotFoundWithErrMsgIsNotFoundError(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"symbol not found", wire404SymbolNotFound, "Symbol not found."},
		{"no option found", wire404NoOptionFound, "No option found. No option was found for this strike and expiration."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, closeFn := clientFor(t, 404, "application/json", tc.body)
			defer closeFn()

			resp, err := c.Get(context.Background(), "stocks/quotes/ZZZZQQ/", nil, nil)
			if err == nil {
				t.Fatal("Get() returned nil error for a 404 naming an error; the caller would read a typo as an empty answer")
			}
			var nf *sdkerrors.NotFoundError
			if !errors.As(err, &nf) {
				t.Fatalf("Get() error = %T (%v), want *sdkerrors.NotFoundError", err, err)
			}
			if !errors.Is(err, sdkerrors.ErrNotFound) {
				t.Error("errors.Is(err, ErrNotFound) = false, want true")
			}
			if nf.Message != tc.want {
				t.Errorf("Message = %q, want %q", nf.Message, tc.want)
			}
			if resp == nil {
				t.Fatal("Get() response = nil; the error path must still hand back the response")
			}
		})
	}
}

func TestGet_NotFoundWithoutErrMsgStaysNoData(t *testing.T) {
	c, closeFn := clientFor(t, 404, "application/json", wire404EmptyAnswer)
	defer closeFn()

	resp, err := c.Get(context.Background(), "options/chain/AAPL/", nil, nil)
	if err != nil {
		t.Fatalf("Get() error = %v, want nil: a markerless 404 is an empty answer, not a failure", err)
	}
	if resp == nil || resp.StatusCode != 404 {
		t.Fatalf("Get() response = %v, want the 404 passed through", resp)
	}
}

// A 204 is the mode=cached cache miss. Its body is empty, so it can never
// carry the marker — but the guard must not decode it into a spurious error.
func TestGet_NoContentStaysNoData(t *testing.T) {
	c, closeFn := clientFor(t, 204, "application/json", "")
	defer closeFn()

	if _, err := c.Get(context.Background(), "stocks/candles/D/AAPL/", nil, nil); err != nil {
		t.Fatalf("Get() error = %v, want nil for 204", err)
	}
}

func TestGetUnversioned_NotFoundWithErrMsgIsNotFoundError(t *testing.T) {
	c, closeFn := clientFor(t, 404, "application/json", wire404SymbolNotFound)
	defer closeFn()

	_, err := c.GetUnversioned(context.Background(), "headers/", nil, nil)
	var nf *sdkerrors.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("GetUnversioned() error = %T (%v), want *sdkerrors.NotFoundError", err, err)
	}
}

func TestGetUnversioned_NotFoundWithoutErrMsgStaysNoData(t *testing.T) {
	c, closeFn := clientFor(t, 404, "application/json", wire404EmptyAnswer)
	defer closeFn()

	if _, err := c.GetUnversioned(context.Background(), "headers/", nil, nil); err != nil {
		t.Fatalf("GetUnversioned() error = %v, want nil for a markerless 404", err)
	}
}

// The formatted facets take the same rule, and the marker reaches them as
// CSV rather than JSON — parseErrorBody reads whichever the response
// declares. On this path production does not even use 404 for the empty
// answer (an impossible filter comes back 200 with a degenerate body), but
// the markerless branch is still pinned so the two paths cannot diverge.
func TestGetFormatted_NotFoundWithErrMsgIsNotFoundError(t *testing.T) {
	c, closeFn := clientFor(t, 404, "text/csv; charset=utf-8", wire404SymbolCSV)
	defer closeFn()

	_, err := c.GetFormatted(context.Background(), "options/chain/ZZZZQQ/", nil, "csv")
	var nf *sdkerrors.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("GetFormatted() error = %T (%v), want *sdkerrors.NotFoundError", err, err)
	}
	if nf.Message != "Symbol not found." {
		t.Errorf("Message = %q, want %q", nf.Message, "Symbol not found.")
	}
}

func TestGetFormatted_NotFoundWithoutErrMsgStaysRaw(t *testing.T) {
	c, closeFn := clientFor(t, 404, "text/csv; charset=utf-8", wire404EmptyCSV)
	defer closeFn()

	resp, err := c.GetFormatted(context.Background(), "options/chain/AAPL/", nil, "csv")
	if err != nil {
		t.Fatalf("GetFormatted() error = %v, want nil for a markerless 404", err)
	}
	if string(resp.Body) != wire404EmptyCSV {
		t.Errorf("body = %q, want the raw text passed through", resp.Body)
	}
}

// A 404 whose body is neither JSON nor CSV (a proxy's HTML error page, say)
// carries no marker and must not be forced into an error: the SDK cannot
// tell it apart from an empty answer, and guessing would turn a reachable
// endpoint's empty result into a failure.
func TestGet_NotFoundWithUndecodableBodyStaysNoData(t *testing.T) {
	c, closeFn := clientFor(t, 404, "text/html", "<html>404 Not Found</html>")
	defer closeFn()

	if _, err := c.Get(context.Background(), "stocks/quotes/AAPL/", nil, nil); err != nil {
		t.Fatalf("Get() error = %v, want nil for an undecodable 404 body", err)
	}
}

func TestNotFoundNamesAnError_BlankErrMsgDoesNotCount(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want bool
	}{
		{"absent", wire404EmptyAnswer, false},
		{"empty string", `{"s":"no_data","errmsg":""}`, false},
		{"whitespace only", `{"s":"no_data","errmsg":"   "}`, false},
		{"present", wire404SymbolNotFound, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := &Response{StatusCode: 404, Body: []byte(tc.body), Headers: http.Header{}}
			if got := notFoundNamesAnError(resp); got != tc.want {
				t.Errorf("notFoundNamesAnError(%s) = %v, want %v", tc.body, got, tc.want)
			}
		})
	}
}
