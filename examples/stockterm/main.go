// Command stockterm is a terminal watchlist for the Market Data API,
// built with Bubble Tea. It refreshes a list of stock and fund symbols on
// an interval, and — in later iterations of this example — drills into a
// detail pane with candles, 52-week range, earnings, and news for the
// selected symbol.
//
// Usage:
//
//	export MARKETDATA_TOKEN="your-api-key"
//	go run . [-refresh 5s] [-funds VFINX] [-prices] [-once] [-base-url URL] [symbols...]
//
// With no arguments the watchlist defaults to AAPL, MSFT, META, SPY, and
// VFINX, with VFINX treated as a mutual fund. Positional arguments replace
// the default watchlist entirely. Running without MARKETDATA_TOKEN set
// starts the client in demo mode, which limits the watchlist to AAPL.
//
// -once runs every fetch synchronously, prints a single rendered frame to
// stdout, and exits — a headless mode used for grading and scripting
// rather than interactive use.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
)

// defaultSymbols is the watchlist used when no positional symbols are
// given on the command line.
var defaultSymbols = []string{"AAPL", "MSFT", "META", "SPY", "VFINX"}

// defaultFunds is the -funds default: the set of default symbols that are
// mutual funds rather than stocks.
const defaultFunds = "VFINX"

// defaultRefresh is the -refresh default: how often the watchlist reloads.
const defaultRefresh = 5 * time.Second

// appConfig holds the fully parsed CLI configuration for one run of
// stockterm.
type appConfig struct {
	// symbols is the watchlist, in display order.
	symbols []string

	// funds marks which symbols (upper-cased) are mutual funds, so their
	// candles come from client.Funds instead of client.Stocks.
	funds map[string]bool

	// refresh is the watchlist refresh cadence.
	refresh time.Duration

	// usePrices selects the lightweight prices endpoint instead of full
	// quotes for the watchlist refresh.
	usePrices bool

	// once runs every fetch synchronously, prints one frame, and exits
	// instead of starting the interactive Bubble Tea program.
	once bool

	// baseURL overrides the API base URL. When set, the client also skips
	// startup token validation, since the override is typically a mock or
	// test server.
	baseURL string
}

// parseFlags parses os.Args into an appConfig, applying the documented
// defaults: watchlist AAPL, MSFT, META, SPY, VFINX; funds VFINX; refresh
// 5s. Positional arguments, if any, replace the default watchlist
// entirely. Symbols in -funds are upper-cased so they match against
// upper-case ticker symbols regardless of how the user typed them.
func parseFlags() appConfig {
	fs := flag.NewFlagSet("stockterm", flag.ExitOnError)

	fundsFlag := fs.String("funds", defaultFunds, "comma-separated symbols treated as mutual funds")
	refreshFlag := fs.Duration("refresh", defaultRefresh, "watchlist refresh interval")
	pricesFlag := fs.Bool("prices", false, "use the prices endpoint instead of quotes for the watchlist")
	onceFlag := fs.Bool("once", false, "fetch once, print a frame, and exit")
	baseURLFlag := fs.String("base-url", "", "override the API base URL (implies skipping startup validation)")

	_ = fs.Parse(os.Args[1:])

	// Copy defaultSymbols rather than alias it: callers may reassign
	// cfg.symbols later (e.g. forcing demo mode down to just AAPL), and
	// parseFlags may run more than once in a test process.
	symbols := append([]string(nil), defaultSymbols...)
	if args := fs.Args(); len(args) > 0 {
		symbols = args
	}

	funds := make(map[string]bool)
	for _, s := range strings.Split(*fundsFlag, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		funds[strings.ToUpper(s)] = true
	}

	return appConfig{
		symbols:   symbols,
		funds:     funds,
		refresh:   *refreshFlag,
		usePrices: *pricesFlag,
		once:      *onceFlag,
		baseURL:   *baseURLFlag,
	}
}

// newClient builds the SDK client for cfg. When cfg.baseURL is set, it
// points the client at that URL and skips startup token validation, since
// a custom base URL is typically a mock or test server rather than the
// live API. Otherwise it builds a plain client, which resolves the token
// from the environment or a .env file.
func newClient(cfg appConfig) (*marketdata.Client, error) {
	if cfg.baseURL != "" {
		return marketdata.NewClient(
			marketdata.WithBaseURL(cfg.baseURL),
			marketdata.WithoutStartupValidation(),
		)
	}
	return marketdata.NewClient()
}

// onceFetchCmds returns, in a fixed order, every fetch command -once
// executes: everything the interactive app can reach on its own (every
// fetch.go function except validateSymbol, which needs a simulated
// keypress in the add-symbol modal and — since it also calls
// client.Stocks.Quote — adds no SDK method coverage that
// fetchDetailQuote doesn't already provide). Unlike refreshCmds/
// detailCmds, this always fetches both fetchQuotes and fetchPrices (not
// just the one -prices selects) and both fetchCandles and
// fetchFundCandles when applicable, since -once's job is exercising
// every endpoint the app owns, not reproducing one interactive session.
// The fixed order (quotes and prices first, then the rest) keeps -once
// output byte-stable across runs. Kept pure, like refreshCmds/
// detailCmds, so tests can inspect the plan without making any of the
// calls.
func onceFetchCmds(client *marketdata.Client, m model) []tea.Cmd {
	var nonFund []string
	for _, s := range m.symbols {
		if !m.funds[s] {
			nonFund = append(nonFund, s)
		}
	}

	var cmds []tea.Cmd
	if len(nonFund) > 0 {
		cmds = append(cmds, fetchQuotes(client, nonFund), fetchPrices(client, nonFund))
	}

	selected := m.symbols[m.selected]
	if !m.funds[selected] {
		cmds = append(cmds, fetchCandles(client, selected, m.rng))
	}
	for _, s := range m.symbols {
		if m.funds[s] {
			cmds = append(cmds, fetchFundCandles(client, s))
		}
	}

	cmds = append(cmds,
		fetchDetailQuote(client, selected),
		fetchEarnings(client, selected),
		fetchNews(client, selected),
		fetchMarketStatus(client),
		fetchStatusHistory(client),
	)
	if !m.demoMode {
		cmds = append(cmds, fetchUser(client))
	}
	cmds = append(cmds,
		fetchAPIStatus(client),
		fetchHeaders(client),
		fetchBulkCandles(client, m.symbols),
	)
	return cmds
}

// supportInfoOrError formats err the way -once's error report does: the
// SDK's SupportInfo() block when err implements marketdata.Error and
// that block is non-empty, or err.Error() otherwise. SupportInfo() is
// empty for *marketdata.ValidationError (a client-side error with no
// request to report on) and errors.As simply doesn't match at all for
// non-SDK errors (e.g. a wrapped context.DeadlineExceeded); err.Error()
// covers both.
func supportInfoOrError(err error) string {
	var sdkErr marketdata.Error
	if errors.As(err, &sdkErr) {
		if info := sdkErr.SupportInfo(); info != "" {
			return info
		}
	}
	return err.Error()
}

// waitForGoroutines polls runtime.NumGoroutine every 25ms until it drops
// to at most baseline+slack or timeout elapses, reporting whether it
// settled. It is runOnce's own copy of the tuitest module's identical
// Settle helper: tuitest is a test-only dependency (see its package doc
// and go.mod's replace directive), so production code — this file —
// can't import it.
func waitForGoroutines(baseline, slack int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if runtime.NumGoroutine() <= baseline+slack {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// runOnce runs the headless grading mode selected by -once: build the
// client, synchronously drive every fetch the app owns through the
// model's Update loop, print one rendered frame plus a diagnostics
// summary to out, and return a process exit code. It never starts
// tea.Program and needs no TTY, which makes it both the grader's
// instrument (see the README's "-once grading contract" section) and a
// scriptable live-API canary: a clean "-once" exit against the live API
// proves every endpoint the app depends on is still reachable and
// well-formed end to end.
func runOnce(cfg appConfig, out io.Writer) int {
	// Ascii makes every lipgloss style in styles.go render with no escape
	// codes at all (termenv.Style.Styled returns its input unmodified
	// under this profile), so View()'s output below is plain text, safe
	// to print and diff without a terminal. Set first, before anything
	// else in this function, so no code path can render styled output
	// before it takes effect.
	lipgloss.SetColorProfile(termenv.Ascii)

	// Recorded before the client exists, so the client's own connection
	// pool goroutines are counted in what "clean" means once it closes.
	baseline := runtime.NumGoroutine()

	client, err := newClient(cfg)
	if err != nil {
		fmt.Fprintf(out, "stockterm: %v\n", err)
		return 1
	}

	demoMode := client.DemoMode()
	if demoMode {
		cfg.symbols = []string{"AAPL"}
	}
	m := newModel(client, cfg, demoMode)

	// runErr is captured here in the loop, NOT read back from m.lastErr
	// afterward: Update deliberately clears lastErr on the next successful
	// data message of any kind (see isDataMsg), so an early failure
	// followed by later successes would leave m.lastErr nil by the end of
	// the run — exit 3 with no SUPPORT INFO block. The -once contract is
	// "SUPPORT INFO when an errMsg occurred", so the mode records the
	// error itself, last-error-wins.
	var runErr error
	for _, cmd := range onceFetchCmds(client, m) {
		msg := cmd()
		if em, ok := msg.(errMsg); ok {
			runErr = em.err
		}
		res, _ := m.Update(msg)
		m = res.(model)
	}

	res, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = res.(model)

	fmt.Fprintln(out, m.View())

	if runErr != nil {
		fmt.Fprintln(out, "SUPPORT INFO:")
		fmt.Fprintln(out, supportInfoOrError(runErr))
	}

	_ = client.Close()

	clean := waitForGoroutines(baseline, 1, 2*time.Second)
	n := runtime.NumGoroutine()
	if clean {
		fmt.Fprintf(out, "goroutines: clean (n=%d baseline=%d)\n", n, baseline)
	} else {
		fmt.Fprintf(out, "goroutines: LEAK (n=%d baseline=%d)\n", n, baseline)
	}

	switch {
	case !clean:
		return 1
	case runErr != nil:
		return 3
	default:
		return 0
	}
}

// run builds the client and model for cfg and, unless cfg.once is set,
// starts the interactive Bubble Tea program. It returns a process exit
// code and always closes the client before returning, even on error.
func run(cfg appConfig) int {
	client, err := newClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "stockterm: %v\n", err)
		return 1
	}
	defer client.Close()

	demoMode := client.DemoMode()
	if demoMode {
		cfg.symbols = []string{"AAPL"}
	}

	m := newModel(client, cfg, demoMode)

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "stockterm: %v\n", err)
		return 1
	}
	return 0
}

func main() {
	cfg := parseFlags()

	if cfg.once {
		os.Exit(runOnce(cfg, os.Stdout))
	}

	os.Exit(run(cfg))
}
