// Example: optionterm
//
// An interactive terminal explorer for option chains, built on the
// Market Data Go SDK v2. It resolves an underlying symbol to its
// available expirations, loads the option chain for the selected
// expiration filtered to a strike window around the underlying price,
// and refreshes on a timer. It is reference code: every SDK call the
// program makes lives in fetch.go, one function per operation.
//
// Usage:
//
//	export MARKETDATA_TOKEN="your-api-key"
//	go run . -symbol AAPL -refresh 15s
//	go run . -once -symbol AAPL
//
// Without a token the client runs in demo mode, which serves a fixed set
// of demo data and forces the underlying symbol to AAPL regardless of
// -symbol.
//
// Flags:
//
//	-symbol    underlying stock symbol (default "AAPL")
//	-refresh   chain refresh interval (default 15s)
//	-once      fetch once, print a frame, and exit; no TTY required
//	-base-url  override the API base URL (testing)
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
)

// appConfig holds the parsed command-line configuration for optionterm.
type appConfig struct {
	// symbol is the underlying stock symbol whose option chain is
	// displayed, normalized to upper case.
	symbol string

	// refresh is the interval on which the displayed chain is reloaded.
	refresh time.Duration

	// once, when true, runs every fetch synchronously, prints one frame,
	// and exits instead of starting the interactive program.
	once bool

	// baseURL, when non-empty, overrides the API base URL and disables
	// startup token validation. It exists for testing against a mock
	// server.
	baseURL string
}

// parseFlags parses args into an appConfig. It never calls os.Exit itself
// (flag.ExitOnError does, but only on genuinely malformed input), which
// keeps it easy to exercise from tests with arbitrary argument slices.
func parseFlags(args []string) appConfig {
	fs := flag.NewFlagSet("optionterm", flag.ExitOnError)

	symbol := fs.String("symbol", "AAPL", "underlying stock symbol for the option chain")
	refresh := fs.Duration("refresh", 15*time.Second, "chain refresh interval")
	once := fs.Bool("once", false, "fetch once, print a frame, and exit (no TTY)")
	baseURL := fs.String("base-url", "", "override the API base URL (testing)")

	_ = fs.Parse(args)

	return appConfig{
		symbol:  strings.ToUpper(*symbol),
		refresh: *refresh,
		once:    *once,
		baseURL: *baseURL,
	}
}

// newClient builds a Market Data client for cfg. When cfg.baseURL is set
// it points the client at that URL and skips startup token validation, so
// tests can construct a client against a mock server (or no server at
// all) without a network round trip. Otherwise it builds a plain client,
// which resolves its token from the environment or a .env file and
// falls back to demo mode when no token is found.
func newClient(cfg appConfig) (*marketdata.Client, error) {
	if cfg.baseURL != "" {
		return marketdata.NewClient(
			marketdata.WithBaseURL(cfg.baseURL),
			marketdata.WithoutStartupValidation(),
		)
	}
	return marketdata.NewClient()
}

// runOnce (once.go) implements the -once flag: it builds its own client
// from cfg, so main does not construct one on that path.

func main() {
	cfg := parseFlags(os.Args[1:])

	if cfg.once {
		os.Exit(runOnce(cfg, os.Stdout))
	}

	client, err := newClient(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "optionterm: failed to create client:", err)
		os.Exit(1)
	}

	symbol := cfg.symbol
	demoMode := client.DemoMode()
	if demoMode {
		// Demo mode serves a fixed data set keyed to AAPL; honor that
		// regardless of what the caller asked for.
		symbol = "AAPL"
	}

	m := newModel(client, symbol, cfg.refresh, demoMode)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, runErr := p.Run()
	_ = client.Close()
	if runErr != nil {
		fmt.Fprintln(os.Stderr, "optionterm:", runErr)
		os.Exit(1)
	}
}
