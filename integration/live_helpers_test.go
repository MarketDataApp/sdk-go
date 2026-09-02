//go:build integration

package integration

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/dotenv"
	"github.com/MarketDataApp/sdk-go/v2/marketdata"
)

// liveToken resolves the API token the same way the SDK does — process
// environment first, then a .env file in the working directory (TestMain
// has already chdir'd to the module root).
//
// A bare os.Getenv is not enough, and getting this wrong is silent: the
// token lives only in .env in local development, so the probes that build
// their own client with WithToken(liveToken()) were handed an empty string
// and ran in DEMO MODE. They still mostly "passed", because demo requests
// come back with plan errors that the acceptance check correctly treats as
// non-rejections — so the universal-parameter probes (mode, maxage, limit,
// offset, columns, dateformat) were never exercised against a real plan at
// all, and failed intermittently as demo limits kicked in. Same reasoning
// as setupClient and TestMain, which both document why they avoid
// os.Getenv; this helper simply had not been updated to match.
func liveToken() string {
	if tok := os.Getenv("MARKETDATA_TOKEN"); tok != "" {
		return tok
	}
	env, err := dotenv.Parse(".env")
	if err != nil {
		return ""
	}
	return env["MARKETDATA_TOKEN"]
}

// liveContext returns a context with a generous timeout for tests that make
// many sequential live calls (the 30s testContext is too tight for the
// catalog-wide acceptance sweep).
func liveContext(t *testing.T, d time.Duration) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return ctx
}

// Rate-limit-aware live-call plumbing.
//
// Every live SDK call in the exhaustive suite is routed through liveCall (or
// its response-only sibling liveProbe). On an HTTP 429 the API surfaces a
// *marketdata.RateLimitError; the wrapper sleeps for the error's WaitDuration
// (capped at liveMaxWait so a daily-quota reset can never wedge the suite) and
// retries a few times before giving up. A small pacing sleep before each
// attempt keeps the suite from bursting requests at the API.
const (
	liveMaxAttempts = 4
	liveMaxWait     = 30 * time.Second
	livePace        = 15 * time.Millisecond
)

// liveCall runs fn, retrying on a rate-limit error after sleeping for the
// server-advised (capped) wait. Any non-rate-limit error is returned to the
// caller immediately — the caller decides whether it is fatal (a real failure)
// or acceptable (for example a NoData 404, which the SDK reports as a nil
// error with resp.NoData == true).
func liveCall[T any](t *testing.T, desc string, fn func() (T, *marketdata.Response, error)) (T, *marketdata.Response, error) {
	t.Helper()
	var (
		v    T
		resp *marketdata.Response
		err  error
	)
	for attempt := 1; attempt <= liveMaxAttempts; attempt++ {
		time.Sleep(livePace) // gentle pacing to avoid bursts
		v, resp, err = fn()
		if err == nil {
			return v, resp, nil
		}
		var rl *marketdata.RateLimitError
		if errors.As(err, &rl) {
			wait := rl.WaitDuration()
			if wait <= 0 || wait > liveMaxWait {
				wait = liveMaxWait
			}
			t.Logf("%s: rate limited (attempt %d/%d), sleeping %s", desc, attempt, liveMaxAttempts, wait)
			time.Sleep(wait)
			continue
		}
		return v, resp, err // non-rate-limit error: hand back to caller
	}
	return v, resp, err
}

// liveProbe is liveCall for calls whose typed payload is irrelevant — the
// catalog acceptance test only cares whether the API accepted the request, so
// it keeps just the *marketdata.Response and error.
func liveProbe(t *testing.T, desc string, fn func() (*marketdata.Response, error)) (*marketdata.Response, error) {
	t.Helper()
	_, resp, err := liveCall(t, desc, func() (struct{}, *marketdata.Response, error) {
		r, e := fn()
		return struct{}{}, r, e
	})
	return resp, err
}

// Historical fixtures known to have data (verified live against the API on
// 2026-07-11). Daily candles exist for these dates, and markets were open.
var (
	liveHistDay  = time.Date(2024, 6, 3, 0, 0, 0, 0, time.UTC)  // a Monday trading day
	liveHistFrom = time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)  // range start
	liveHistTo   = time.Date(2024, 6, 5, 0, 0, 0, 0, time.UTC)  // range end (inclusive)
	liveHistWide = time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC) // a Friday with data
)

// truncDay reduces a time to its UTC calendar date for date-equality checks.
func truncDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
