// Package tuitest provides headless helpers for driving and grading
// Bubble Tea models in tests: message injection, ANSI-free frame capture,
// golden-file comparison, and goroutine-leak settlement.
package tuitest

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Drive applies msgs to m in order via Update, discarding returned
// commands, and returns the final model. It is the headless equivalent
// of the Bubble Tea event loop for deterministic tests.
func Drive(m tea.Model, msgs ...tea.Msg) tea.Model {
	for _, msg := range msgs {
		m, _ = m.Update(msg)
	}
	return m
}

// Frame renders m.View() with ANSI escape sequences stripped and
// trailing whitespace trimmed per line — a stable, diffable frame.
func Frame(m tea.Model) string {
	stripped := StripANSI(m.View())
	lines := strings.Split(stripped, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}

// ansiEscapeRE matches CSI sequences (ESC '[' params letter) and OSC
// sequences (ESC ']' ... BEL), the two escape forms lipgloss/termenv emit.
var ansiEscapeRE = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]|\x1b\\][^\x07]*\x07")

// StripANSI removes ANSI escape sequences from s.
func StripANSI(s string) string {
	return ansiEscapeRE.ReplaceAllString(s, "")
}

// updateGolden is the parsed value of the -update flag, registered once
// at package load so it is available before the testing package parses
// flags. registerUpdateFlagOnce guards the registration itself so that
// re-entrant or repeated calls never attempt to redefine the flag (which
// would panic).
var (
	updateGolden           *bool
	registerUpdateFlagOnce sync.Once
)

func init() {
	registerUpdateFlag()
}

// registerUpdateFlag registers the -update flag exactly once per process,
// no matter how many times it is invoked.
func registerUpdateFlag() {
	registerUpdateFlagOnce.Do(func() {
		updateGolden = flag.Bool("update", false, "update golden files (tuitest.Golden)")
	})
}

// tHelper is the subset of *testing.T that golden depends on. It exists so
// this package's own tests can exercise golden's write/compare/fail logic
// against a lightweight fake, without needing to fork a subprocess just to
// vary the -update flag.
type tHelper interface {
	Helper()
	Fatalf(format string, args ...any)
}

// Golden compares got against the golden file at path (relative to the
// test's testdata dir). Run tests with -update to rewrite golden files.
func Golden(t *testing.T, path, got string) {
	t.Helper()
	golden(t, path, got, updateGolden != nil && *updateGolden)
}

// golden implements Golden against an explicit update flag so it can be
// unit-tested directly.
func golden(t tHelper, path, got string, update bool) {
	t.Helper()

	if update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("tuitest: creating directory for golden file %s: %v", path, err)
			return
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("tuitest: writing golden file %s: %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("tuitest: reading golden file %s: %v (run tests with -update to create it)", path, err)
		return
	}

	if got != string(want) {
		t.Fatalf("tuitest: golden mismatch for %s:\n%s", path, lineDiff(string(want), got))
	}
}

// lineDiff renders a readable line-by-line diff of want vs got.
func lineDiff(want, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")

	max := len(wantLines)
	if len(gotLines) > max {
		max = len(gotLines)
	}

	var b strings.Builder
	b.WriteString("--- want\n+++ got\n")
	for i := 0; i < max; i++ {
		var w, g string
		hasWant, hasGot := i < len(wantLines), i < len(gotLines)
		if hasWant {
			w = wantLines[i]
		}
		if hasGot {
			g = gotLines[i]
		}
		if hasWant && hasGot && w == g {
			fmt.Fprintf(&b, "  %s\n", w) //nolint:errcheck // strings.Builder never errors
			continue
		}
		if hasWant {
			fmt.Fprintf(&b, "- %s\n", w) //nolint:errcheck // strings.Builder never errors
		}
		if hasGot {
			fmt.Fprintf(&b, "+ %s\n", g) //nolint:errcheck // strings.Builder never errors
		}
	}
	return b.String()
}

// Settle reports whether the process goroutine count returns to at most
// baseline+slack within timeout, polling every 25ms. Use it after
// client.Close() to prove a session leaks no goroutines.
func Settle(baseline, slack int, timeout time.Duration) bool {
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
