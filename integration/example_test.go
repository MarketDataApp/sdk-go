//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/funds"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/markets"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

// chdirToModuleRoot switches the process's working directory to the module
// root (the directory containing go.mod). NewClient's .env resolution is
// relative to the working directory, but `go test` always runs a
// package's tests with that package's own source directory as the
// working directory — so without this, the repository's root-level .env
// (where a locally-developed token normally lives) is invisible to every
// package's tests, integration included. This makes `go test
// ./integration/...` behave the same as if it were invoked from the
// repository root.
func chdirToModuleRoot() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return os.Chdir(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return fmt.Errorf("go.mod not found in any parent directory")
		}
		dir = parent
	}
}

// TestMain gates the whole package on a token. Unlike regular tests,
// Example functions cannot call t.Skip, so without a token we exit before
// any example — or, since TestMain runs once for the entire package
// before any individual test, any Test* function either — runs against
// the live API.
//
// Locally that gate exits 0, so a contributor without a token is not
// blocked. In CI it exits 1 instead: there, a missing token is a broken
// pipeline, and reporting the whole live suite as passing while it ran
// nothing is worse than failing. That is not hypothetical — it is how
// this package stayed green while TestCatalog_ParamsAcceptedLive was
// failing on five catalogued parameters.
//
// The gate is resolved through the SDK's own configuration cascade
// (process env, then .env) via NewClient, not a direct os.Getenv check: a
// token that lives only in .env (as it does in local development)
// satisfies the SDK but not a bare os.Getenv("MARKETDATA_TOKEN"), since
// NewClient never writes .env values into the process environment.
// WithoutStartupValidation skips the live /user/ call here — this only
// probes whether a token was found, not whether it's valid; each test's
// own setupClient (or Example's own NewClient call) still validates it.
func TestMain(m *testing.M) {
	if err := chdirToModuleRoot(); err != nil {
		fmt.Fprintf(os.Stderr, "could not locate module root: %v\n", err)
		os.Exit(1)
	}

	client, err := marketdata.NewClient(marketdata.WithoutStartupValidation())
	if err != nil || client.DemoMode() {
		const missing = "no API token available (checked the process environment and .env)"
		if os.Getenv("CI") != "" {
			fmt.Fprintf(os.Stderr, "%s; refusing to report success in CI — set MARKETDATA_TOKEN, or unset CI to skip locally\n", missing)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, missing+", skipping integration tests and examples")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// The examples below run against the live API (like the v1 SDK's testable
// examples) and assert on stable fields only, so their output stays
// deterministic across market conditions.
//
// They print errors and return rather than calling log.Fatal, which is what an
// example in godoc would idiomatically do. These are not godoc examples: the
// whole package is behind the integration build tag, so pkg.go.dev never
// renders them — the user-facing ones live in each service package. Here
// log.Fatal only costs, because it calls os.Exit: the test binary dies before
// the framework can print a "--- FAIL:" line naming the example, and every
// test that had not run yet is skipped in silence. That is not hypothetical —
// a transient HTTP 500 on /status/ killed the whole suite on 2026-09-01
// (PR #35) and the log named no test at all. Printing the error instead turns
// the same blip into an ordinary output mismatch: attributable, and the rest
// of the suite still runs.

func Example_stocksQuote() {
	client, err := marketdata.NewClient()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = client.Close() }()

	quote, _, err := client.Stocks.Quote(context.Background(), "AAPL")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(quote.Symbol)
	// Output: AAPL
}

func Example_stocksCandles() {
	client, err := marketdata.NewClient()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = client.Close() }()

	from := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	candles, _, err := client.Stocks.Candles(context.Background(), "AAPL",
		stocks.WithResolution(stocks.ResolutionDaily),
		stocks.WithCandleWindow(stocks.Between(from, to)),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(len(candles) > 0)
	// Output: true
}

func Example_optionsChain() {
	client, err := marketdata.NewClient()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = client.Close() }()

	chain, _, err := client.Options.Chain(context.Background(), "AAPL")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(chain.Underlying)
	// Output: AAPL
}

func Example_optionsExpirations() {
	client, err := marketdata.NewClient()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = client.Close() }()

	expirations, _, err := client.Options.Expirations(context.Background(), "AAPL")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(len(expirations.Dates) > 0)
	// Output: true
}

func Example_fundsCandles() {
	client, err := marketdata.NewClient()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = client.Close() }()

	from := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	candles, _, err := client.Funds.Candles(context.Background(), "VFINX",
		funds.WithCandleWindow(funds.Between(from, to)),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(len(candles) > 0)
	// Output: true
}

func Example_marketsStatus() {
	client, err := marketdata.NewClient()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = client.Close() }()

	// 2024-01-16 was a regular trading day
	status, _, err := client.Markets.Status(context.Background(),
		markets.WithDate(time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(status.IsOpen())
	// Output: true
}

func Example_utilitiesStatus() {
	client, err := marketdata.NewClient()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = client.Close() }()

	status, _, err := client.Utilities.Status(context.Background())
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(status.IsOnline())
	// Output: true
}

func Example_utilitiesUser() {
	client, err := marketdata.NewClient()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = client.Close() }()

	user, _, err := client.Utilities.User(context.Background())
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(user != nil)
	// Output: true
}
