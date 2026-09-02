package http

// Regression guards for the per-attempt rate-limit reservation in Do (gap
// 90). All three pin invariants the PR #33 round-2 review found broken in
// the first version of that fix; each failed until its repair landed and
// must never fail again.

import (
	"context"
	"errors"
	"fmt"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/ratelimit"
	"github.com/MarketDataApp/sdk-go/v2/internal/retry"
	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

// guardTransport lets a test script the transport per call.
type guardTransport func(*nethttp.Request) (*nethttp.Response, error)

func (f guardTransport) RoundTrip(r *nethttp.Request) (*nethttp.Response, error) { return f(r) }

// rateHeaders builds rate-limit headers for priming a tracker.
func rateHeaders(limit, remaining int, reset time.Time) nethttp.Header {
	h := nethttp.Header{}
	h.Set("X-Api-Ratelimit-Limit", strconv.Itoa(limit))
	h.Set("X-Api-Ratelimit-Remaining", strconv.Itoa(remaining))
	h.Set("X-Api-Ratelimit-Reset", strconv.FormatInt(reset.Unix(), 10))
	return h
}

func okJSONResponse(header nethttp.Header) *nethttp.Response {
	return &nethttp.Response{
		StatusCode: 200,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(`{"s":"ok"}`)),
	}
}

// TestDo_ReservationHeldThroughUpdate pins the invariant that a request's
// reservation is held until the tracker has absorbed the response's
// remaining count. With the tracker primed at remaining=1 and the response
// reporting remaining=0, no concurrent Reserve may succeed at ANY instant
// between the request being dispatched and Do returning: while the request
// is in flight the reservation blocks it, and from Update onwards the new
// remaining does. An earlier version released before Update, opening a
// microseconds-wide window in between; the scenario is amplified to catch a
// reintroduction, and on correct code a success is impossible in every
// interleaving, so the test cannot flake.
func TestDo_ReservationHeldThroughUpdate(t *testing.T) {
	reset := time.Now().Add(time.Hour)

	inFlight := make(chan struct{}, 1)
	server := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		inFlight <- struct{}{}
		w.Header().Set("X-Api-Ratelimit-Limit", "100")
		w.Header().Set("X-Api-Ratelimit-Remaining", "0")
		w.Header().Set("X-Api-Ratelimit-Reset", strconv.FormatInt(reset.Unix(), 10))
		_, _ = fmt.Fprint(w, `{"s":"ok"}`)
	}))
	defer server.Close()

	const iterations = 500
	for i := 0; i < iterations; i++ {
		tracker := ratelimit.New()
		tracker.Update(&nethttp.Response{Header: rateHeaders(100, 1, reset)})

		client := New(Config{
			BaseURL:    server.URL,
			APIVersion: "v1",
			Token:      "test-key",
			RetryCfg:   retry.Config{MaxRetries: 0, InitialBackoff: time.Millisecond, Multiplier: 2},
			RateLimits: tracker,
		})

		done := make(chan error, 1)
		go func() {
			_, err := client.Do(context.Background(), Request{Method: nethttp.MethodGet, Path: "probe/"})
			done <- err
		}()

		<-inFlight // the request is dispatched; its reservation is held
		overshoot := false
		for {
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("iteration %d: Do failed: %v", i, err)
				}
				if overshoot {
					t.Fatalf("iteration %d: a concurrent Reserve succeeded between dispatch and completion — the reservation was released before Update recorded remaining=0", i)
				}
				goto next
			default:
				if tracker.Reserve() {
					overshoot = true
					tracker.Release()
				}
			}
		}
	next:
	}
}

// TestDo_PanicReleasesReservation pins panic-safety of the per-attempt
// reservation: a panic escaping doOnce (a caller-supplied RoundTripper is
// the realistic source) must not leak the reserved credit. An earlier
// version released with a plain call after doOnce, so a recovered panic
// left `reserved` permanently raised and, with remaining=1, wedged every
// later call into pre-flight rejection. The release is deferred now, like
// the pool-slot release always was.
func TestDo_PanicReleasesReservation(t *testing.T) {
	reset := time.Now().Add(time.Hour)
	tracker := ratelimit.New()

	call := 0
	transport := guardTransport(func(r *nethttp.Request) (*nethttp.Response, error) {
		call++
		if call == 1 {
			return okJSONResponse(rateHeaders(100, 1, reset)), nil
		}
		panic("transport exploded")
	})

	client := New(Config{
		HTTPClient: &nethttp.Client{Transport: transport},
		BaseURL:    "https://api.test",
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.Config{MaxRetries: 0, InitialBackoff: time.Millisecond, Multiplier: 2},
		RateLimits: tracker,
	})

	// Call 1 primes the tracker to remaining=1.
	if _, err := client.Do(context.Background(), Request{Method: nethttp.MethodGet, Path: "probe/"}); err != nil {
		t.Fatalf("priming call failed: %v", err)
	}

	// Call 2 panics mid-attempt; the application recovers, as servers do.
	func() {
		defer func() { _ = recover() }()
		_, _ = client.Do(context.Background(), Request{Method: nethttp.MethodGet, Path: "probe/"})
	}()

	if !tracker.Reserve() {
		t.Fatal("reservation leaked across a recovered panic: Reserve() is rejected pre-flight while a credit is actually available")
	}
	tracker.Release()
}

// reserveTimeoutErr is a retryable transport failure (net.Error, timeout).
type reserveTimeoutErr struct{}

func (reserveTimeoutErr) Error() string   { return "dial tcp: i/o timeout" }
func (reserveTimeoutErr) Timeout() bool   { return true }
func (reserveTimeoutErr) Temporary() bool { return true }

// TestDo_ReserveFailurePreservesLastErr pins the promise the
// reserve-failure branch makes: running out of credits mid-retry must not
// replace a real failure with a pre-flight rejection whose cause the caller
// never saw. An earlier version checked only lastResp — which a transport
// failure never sets — so a retryable network error followed by mid-backoff
// credit exhaustion returned a PreFlight RateLimitError with the network
// error discarded, and logged the loss at WARN as the limiter working as
// designed. The cause stays reachable via errors.As now, as the
// offline-abort branch always kept it for the identical shape.
func TestDo_ReserveFailurePreservesLastErr(t *testing.T) {
	reset := time.Now().Add(time.Hour)
	tracker := ratelimit.New()
	tracker.Update(&nethttp.Response{Header: rateHeaders(100, 1, reset)})

	transport := guardTransport(func(r *nethttp.Request) (*nethttp.Response, error) {
		// Fail the attempt, and exhaust the tracker before the retry's
		// reserve() runs — the concurrent-callers case, made deterministic.
		tracker.Update(&nethttp.Response{Header: rateHeaders(100, 0, reset)})
		return nil, reserveTimeoutErr{}
	})

	client := New(Config{
		HTTPClient: &nethttp.Client{Transport: transport},
		BaseURL:    "https://api.test",
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.Config{MaxRetries: 2, InitialBackoff: time.Millisecond, Multiplier: 2},
		RateLimits: tracker,
	})

	_, err := client.Do(context.Background(), Request{Method: nethttp.MethodGet, Path: "probe/"})
	if err == nil {
		t.Fatal("want an error")
	}
	var netErr *sdkerrors.NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("the real transport failure is unreachable from the returned error — masked by %v", err)
	}
}
