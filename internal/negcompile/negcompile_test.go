// Package negcompile runs the SDK's negative-compilation test set: a corpus of
// Go snippets that each combine mutually-exclusive API parameters (or otherwise
// misuse the sealed option unions) and MUST fail to build. The test compiles
// every snippet in an isolated throwaway module and asserts a non-zero exit
// whose diagnostics match the snippet's declared expectation. A companion
// corpus of "positive" snippets asserts the legal combinations DO build.
//
// This is the machine-checkable proof of the headline guarantee: incompatible
// parameters do not compile. Snippets live under testdata/negative and
// testdata/positive; the go tool ignores testdata, so they never affect the
// main build.
package negcompile

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// repoRoot returns the absolute path to the module root (two levels up).
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root, err := filepath.Abs(filepath.Join(wd, "..", ".."))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("go.mod not found at %s: %v", root, err)
	}
	return root
}

// wantDirective extracts the `// want: <regexp>` directive from a snippet.
//
// Patterns should name the identifier the snippet is about
// (`undefined: stocks\.WithFrom`) rather than the bare diagnostic kind
// (`undefined`). A bare kind is satisfied by any compile error at all, so a
// snippet could keep passing while the combination it guards had quietly
// become legal and something unrelated was failing instead.
// Negative snippets must declare one so the test asserts the RIGHT failure, not
// an incidental one (e.g. a typo).
func wantDirective(t *testing.T, path string) *regexp.Regexp {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "// want:") {
			pat := strings.TrimSpace(strings.TrimPrefix(line, "// want:"))
			re, err := regexp.Compile(pat)
			if err != nil {
				t.Fatalf("bad want regexp in %s: %v", path, err)
			}
			return re
		}
	}
	return nil
}

// buildSnippet writes the snippet into an isolated module that replaces the SDK
// with the local checkout and runs `go build`, returning combined output and
// whether the build succeeded.
func buildSnippet(t *testing.T, root, snippet string) (string, bool) {
	t.Helper()
	dir := t.TempDir()

	src, err := os.ReadFile(snippet)
	if err != nil {
		t.Fatalf("read snippet: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), src, 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	goMod := "module negcompiletest\n\ngo 1.22\n\n" +
		"require github.com/MarketDataApp/sdk-go/v2 v2.0.0\n\n" +
		"replace github.com/MarketDataApp/sdk-go/v2 => " + root + "\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GOFLAGS=-mod=mod",
		"GOPROXY=off",
		"GOSUMDB=off",
		"GOWORK=off",
	)
	out, err := cmd.CombinedOutput()
	return string(out), err == nil
}

func snippets(t *testing.T, subdir string) []string {
	t.Helper()
	glob := filepath.Join("testdata", subdir, "*.go")
	files, err := filepath.Glob(glob)
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	return files
}

// TestNegativeCompile asserts every negative snippet FAILS to build with the
// diagnostic it declares. A snippet that compiles is a hole in the guarantee.
func TestNegativeCompile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping negative-compile suite in -short mode (invokes go build)")
	}
	root := repoRoot(t)
	files := snippets(t, "negative")
	if len(files) == 0 {
		t.Fatal("no negative snippets found under testdata/negative")
	}
	for _, f := range files {
		f := f
		t.Run(filepath.Base(f), func(t *testing.T) {
			t.Parallel()
			want := wantDirective(t, f)
			if want == nil {
				t.Fatalf("%s: missing `// want:` directive", f)
			}
			out, ok := buildSnippet(t, root, f)
			if ok {
				t.Fatalf("snippet COMPILED but must not:\n%s", out)
			}
			if !want.MatchString(out) {
				t.Fatalf("compile error did not match /%s/:\n%s", want, out)
			}
		})
	}
}

// TestPositiveCompile asserts every legal snippet DOES build, proving the
// redesign did not over-constrain the surface.
func TestPositiveCompile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping positive-compile suite in -short mode (invokes go build)")
	}
	root := repoRoot(t)
	files := snippets(t, "positive")
	if len(files) == 0 {
		t.Fatal("no positive snippets found under testdata/positive")
	}
	for _, f := range files {
		f := f
		t.Run(filepath.Base(f), func(t *testing.T) {
			t.Parallel()
			out, ok := buildSnippet(t, root, f)
			if !ok {
				t.Fatalf("legal snippet FAILED to build:\n%s", out)
			}
		})
	}
}
