// runOnce mode: the fresh-context grader's instrument and the app's live
// canary. It builds a client, drives the app's full fetch surface
// synchronously against it — no tea.Program, no TTY — prints one frame, and
// reports whether the session leaked goroutines, all without starting the
// interactive event loop main() otherwise runs.
package main

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
)

// runOnce fetches everything the interactive UI would show for one frame —
// synchronously, in the same order the six operations the app can make
// occur in live use — prints that frame, and reports the client's
// goroutine footprint. It never starts a tea.Program; main's -once branch
// and the grader both call it directly.
//
// The fetch sequence is: the underlying quote, the expiration list, the
// chain for the expiration Update selects as nearest (first load, no
// strike filter), then — only if that chain produced at least one row —
// a contract-detail fetch, a one-symbol pin fan-out, and a lookup, all
// three for the at-the-money row. A no-data response at any step (chain
// included) is not an error: it simply leaves rows empty, which is what
// gates the last three steps, so they are skipped gracefully rather than
// failing.
//
// The return value is the process exit code: 0 when every fetch either
// succeeded or hit a documented no-data response, 3 when at least one
// fetch failed outright (the frame and a SUPPORT INFO block are still
// printed to out), and 1 when the client itself could not be constructed
// or the session's goroutines fail to settle after Close.
func runOnce(cfg appConfig, out io.Writer) int {
	baseline := runtime.NumGoroutine()

	client, err := newClient(cfg)
	if err != nil {
		fmt.Fprintln(out, "optionterm: failed to create client:", err)
		return 1
	}

	// -once output is meant for a pipe or a grader's diff, not a terminal:
	// force an ASCII color profile so View() emits no escape codes.
	lipgloss.SetColorProfile(termenv.Ascii)

	symbol := cfg.symbol
	demoMode := client.DemoMode()
	if demoMode {
		// Demo mode serves a fixed data set keyed to AAPL; honor that
		// regardless of what the caller asked for (mirrors main()).
		symbol = "AAPL"
	}

	m := newModel(client, symbol, cfg.refresh, demoMode)

	// runErr tracks the last errMsg seen across the whole sequence,
	// independent of m.lastErr: a later step's success clears m.lastErr
	// (so the status line reflects only the most recent outcome), but the
	// exit code and the SUPPORT INFO block must still reflect any failure
	// anywhere in the run.
	var runErr error
	drive := func(cmd tea.Cmd) {
		msg := cmd()
		if em, ok := msg.(errMsg); ok {
			runErr = em.err
		}
		next, _ := m.Update(msg)
		m = next.(model)
	}

	// a, b: underlying and expirations.
	drive(fetchUnderlying(client, symbol))
	drive(fetchExpirations(client, symbol))

	// c: the chain for the expiration Update just selected from step b,
	// first load (chainCmds omits the strike filter until a chain response
	// has set m.underlyingPx). No expirations — from an error or no-data —
	// leaves chainCmds() with nothing to return, so this loop is a no-op:
	// the graceful skip falls out of chainCmds' own decision logic.
	for _, cmd := range m.chainCmds() {
		drive(cmd)
	}

	// d, e, f: contract detail, pin, and lookup for the at-the-money row —
	// skipped gracefully whenever the chain came back with no rows
	// (no-data or an empty strike window).
	if idx := atmIndex(m.rows, m.underlyingPx); idx >= 0 {
		row := m.rows[idx]

		// Select the ATM row before rendering: a live first-load chain is
		// unfiltered (100+ rows), and the chain pane scrolls its viewport
		// to the selection — without this, the frame would show only the
		// lowest strikes and never the row steps d/e/f operate on.
		m.rowSelected = idx

		drive(fetchContract(client, row.OptionSymbol))

		m.pinned = append(m.pinned, row.OptionSymbol)
		drive(fetchPinned(client, m.pinned))

		drive(lookupContract(client, m.symbol, m.expirations[m.expSelected], row.Strike, row.Type))

		// fetchContract's contractMsg opens the contract-detail modal
		// (m.detail != nil), same as pressing Enter would. The canary's
		// point is one frame proving all six operations populated real
		// data — chain rows, the pinned entry, expirations — so close the
		// modal back out rather than leave the final frame stuck showing
		// only the detail view.
		m.detail = nil
	}

	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = next.(model)

	fmt.Fprintln(out, m.View())

	exitCode := 0
	if runErr != nil {
		exitCode = 3
		fmt.Fprintln(out, "SUPPORT INFO:")
		var apiErr marketdata.Error
		if errors.As(runErr, &apiErr) {
			fmt.Fprintln(out, apiErr.SupportInfo())
		} else {
			fmt.Fprintln(out, runErr.Error())
		}
	}

	_ = client.Close()

	if n, ok := settle(baseline, 1, 2*time.Second); ok {
		fmt.Fprintf(out, "goroutines: clean (n=%d baseline=%d)\n", n, baseline)
	} else {
		fmt.Fprintf(out, "goroutines: LEAK (n=%d baseline=%d)\n", n, baseline)
		exitCode = 1
	}

	return exitCode
}

// settle polls runtime.NumGoroutine every 25ms until it reports at most
// baseline+slack, or timeout elapses, returning the last observed count and
// whether it settled. This is runOnce's own copy of the same polling loop
// examples/tuitest.Settle uses in tests: production code does not import
// that test-only module, so the (small) logic is duplicated here instead of
// shared.
func settle(baseline, slack int, timeout time.Duration) (n int, ok bool) {
	deadline := time.Now().Add(timeout)
	for {
		n = runtime.NumGoroutine()
		if n <= baseline+slack {
			return n, true
		}
		if time.Now().After(deadline) {
			return n, false
		}
		time.Sleep(25 * time.Millisecond)
	}
}
