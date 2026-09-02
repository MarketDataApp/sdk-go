package tuitest

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// --- toy tea.Model for Drive/Frame tests ---

// incMsg tells counterModel to increment its counter.
type incMsg struct{}

// resetMsg tells counterModel to reset its counter to zero.
type resetMsg struct{}

// counterModel is a minimal tea.Model used only to exercise Drive and Frame.
type counterModel struct {
	n    int
	view string
}

func (m counterModel) Init() tea.Cmd { return nil }

func (m counterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case incMsg:
		m.n++
	case resetMsg:
		m.n = 0
	}
	return m, nil
}

func (m counterModel) View() string {
	if m.view != "" {
		return m.view
	}
	return fmt.Sprintf("count: %d", m.n)
}

func TestDriveAppliesMessagesInOrder(t *testing.T) {
	start := counterModel{}

	got := Drive(start, incMsg{}, incMsg{}, incMsg{}, resetMsg{}, incMsg{})

	cm, ok := got.(counterModel)
	if !ok {
		t.Fatalf("Drive returned %T, want counterModel", got)
	}
	if cm.n != 1 {
		t.Fatalf("Drive applied messages out of order or incorrectly: n = %d, want 1 (inc,inc,inc,reset,inc)", cm.n)
	}
}

// cmdModel returns a booby-trapped non-nil tea.Cmd from every Update call.
// If Drive ever invokes the command (instead of discarding it, per its
// contract), the command panics and the test fails loudly.
type cmdModel struct {
	n int
}

func (m cmdModel) Init() tea.Cmd { return nil }

func (m cmdModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(incMsg); ok {
		m.n++
	}
	return m, func() tea.Msg {
		panic("tuitest: Drive invoked a returned command; it must discard them")
	}
}

func (m cmdModel) View() string { return fmt.Sprintf("count: %d", m.n) }

func TestDriveDiscardsReturnedCommands(t *testing.T) {
	start := cmdModel{}

	// Every Update returns a non-nil command that panics if executed. If a
	// regression made Drive run cmd() (and possibly fold its msg back into
	// Update), this panics instead of silently passing.
	got := Drive(start, incMsg{}, incMsg{})

	cm, ok := got.(cmdModel)
	if !ok {
		t.Fatalf("Drive returned %T, want cmdModel", got)
	}
	if cm.n != 2 {
		t.Fatalf("Drive final state n = %d, want 2 (two incMsg, commands discarded)", cm.n)
	}
}

func TestDriveWithNoMessagesReturnsModelUnchanged(t *testing.T) {
	start := counterModel{n: 5}

	got := Drive(start)

	cm, ok := got.(counterModel)
	if !ok {
		t.Fatalf("Drive returned %T, want counterModel", got)
	}
	if cm.n != 5 {
		t.Fatalf("Drive with no messages changed model: n = %d, want 5", cm.n)
	}
}

// --- StripANSI tests ---

func TestStripANSIRemovesStyledEscapeSequences(t *testing.T) {
	// A lipgloss-rendered string: SGR color codes wrapping plain text.
	styled := "\x1b[38;2;255;0;0mHello\x1b[0m \x1b[1mWorld\x1b[0m"

	got := StripANSI(styled)

	want := "Hello World"
	if got != want {
		t.Fatalf("StripANSI(%q) = %q, want %q", styled, got, want)
	}
}

func TestStripANSIRemovesOSCSequences(t *testing.T) {
	// OSC sequences (e.g. terminal title, hyperlinks) end in BEL (\a).
	styled := "\x1b]0;window title\aHello"

	got := StripANSI(styled)

	want := "Hello"
	if got != want {
		t.Fatalf("StripANSI(%q) = %q, want %q", styled, got, want)
	}
}

func TestStripANSIPlainTextUnchanged(t *testing.T) {
	plain := "just plain text, no escapes here"

	got := StripANSI(plain)

	if got != plain {
		t.Fatalf("StripANSI(%q) = %q, want unchanged", plain, got)
	}
}

// --- Frame tests ---

func TestFrameStripsANSIAndTrimsTrailingWhitespace(t *testing.T) {
	m := counterModel{view: "\x1b[31mline one\x1b[0m   \nline two\t\n\x1b[1mline three\x1b[0m"}

	got := Frame(m)

	want := "line one\nline two\nline three"
	if got != want {
		t.Fatalf("Frame() = %q, want %q", got, want)
	}
}

func TestFramePreservesLeadingWhitespace(t *testing.T) {
	m := counterModel{view: "  indented line   \n    another   "}

	got := Frame(m)

	want := "  indented line\n    another"
	if got != want {
		t.Fatalf("Frame() = %q, want %q", got, want)
	}
}

// --- Golden tests ---
//
// Golden's public signature takes only *testing.T, path, and got — there is
// no way to pass a synthetic -update value from a test without either
// mutating the process-wide flag or shelling out to a subprocess. We drive
// the same code path Golden uses (golden) directly with an explicit update
// bool so both the "write" and "compare" branches are exercised without
// touching the real command-line flag.

func TestGoldenWritesOnUpdateThenMatches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "testdata", "sample.golden")

	golden(t, path, "first content\n", true)

	golden(t, path, "first content\n", false)
}

func TestGoldenMismatchFailsWithDiff(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "testdata", "mismatch.golden")

	golden(t, path, "want this\n", true)

	rt := &recordingT{}
	golden(rt, path, "got this instead\n", false)

	if !rt.failed {
		t.Fatal("golden did not fail on mismatch")
	}
	if !strings.Contains(rt.log, "want this") || !strings.Contains(rt.log, "got this instead") {
		t.Fatalf("golden failure message missing want/got diff content: %q", rt.log)
	}
}

func TestGoldenMissingFileFailsWithoutUpdate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "testdata", "missing.golden")

	rt := &recordingT{}
	golden(rt, path, "anything\n", false)

	if !rt.failed {
		t.Fatal("golden did not fail when golden file is missing and -update not set")
	}
}

func TestGoldenMkdirFailureFails(t *testing.T) {
	dir := t.TempDir()
	// blocker is a regular file; using it as a path component forces
	// os.MkdirAll to fail with ENOTDIR.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	path := filepath.Join(blocker, "testdata", "sample.golden")

	rt := &recordingT{}
	golden(rt, path, "content\n", true)

	if !rt.failed {
		t.Fatal("golden did not fail when the golden directory could not be created")
	}
}

func TestGoldenWriteFailureFails(t *testing.T) {
	dir := t.TempDir()
	// path itself is a directory, so os.WriteFile must fail.
	path := filepath.Join(dir, "testdata", "sample.golden")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	rt := &recordingT{}
	golden(rt, path, "content\n", true)

	if !rt.failed {
		t.Fatal("golden did not fail when the golden file could not be written")
	}
}

func TestGoldenRealTestingTMatches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "testdata", "real.golden")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(path, []byte("matches\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Exercise the exported Golden wrapper (not the golden helper) with a
	// real *testing.T, in the default (non -update) flag state.
	Golden(t, path, "matches\n")
}

func TestLineDiffUnevenLengths(t *testing.T) {
	// want has one line; got has three — exercises both the "only in got"
	// branch (extra trailing lines) and the equal-line branch.
	got := lineDiff("shared", "shared\nextra one\nextra two")

	for _, want := range []string{"shared", "extra one", "extra two"} {
		if !strings.Contains(got, want) {
			t.Fatalf("lineDiff output missing %q:\n%s", want, got)
		}
	}
}

func TestLineDiffWantLongerThanGot(t *testing.T) {
	// got has one line; want has three — exercises the "only in want"
	// branch (extra lines present only on the want side).
	got := lineDiff("shared\nextra one\nextra two", "shared")

	for _, want := range []string{"shared", "extra one", "extra two"} {
		if !strings.Contains(got, want) {
			t.Fatalf("lineDiff output missing %q:\n%s", want, got)
		}
	}
}

// recordingT is a minimal stand-in for *testing.T that records failures
// instead of stopping the goroutine, so golden's failure path can be
// asserted on directly.
type recordingT struct {
	failed bool
	log    string
}

func (r *recordingT) Helper() {}

func (r *recordingT) Fatalf(format string, args ...any) {
	r.failed = true
	r.log += fmt.Sprintf(format, args...)
}

// --- Settle tests ---

func TestSettleFalseWhileGoroutineParked(t *testing.T) {
	baseline := runtime.NumGoroutine()

	block := make(chan struct{})
	started := make(chan struct{})
	go func() {
		close(started)
		<-block
	}()
	defer close(block)
	<-started

	if Settle(baseline, 0, 150*time.Millisecond) {
		t.Fatal("Settle returned true while a goroutine was parked above baseline")
	}
}

func TestSettleTrueAfterGoroutineExits(t *testing.T) {
	baseline := runtime.NumGoroutine()

	block := make(chan struct{})
	started := make(chan struct{})
	go func() {
		close(started)
		<-block
	}()
	<-started

	close(block)

	if !Settle(baseline, 0, 2*time.Second) {
		t.Fatal("Settle returned false after the spawned goroutine exited")
	}
}
